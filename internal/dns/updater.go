package dns

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/zengshenglong/cf-speed-test/internal/config"
	"github.com/zengshenglong/cf-speed-test/internal/scanner"
)

// Updater DNS 更新器接口
type Updater interface {
	UpdateARecord(ctx context.Context, zoneID, recordID, ip string) error
	UpdateCNAMERecord(ctx context.Context, zoneID, recordID, target string) error
	BatchUpdate(ctx context.Context, updates []Update) error
	GetBestIP(records []config.DNSRecord, results []scanner.Result) []Update
}

// Update DNS 更新请求
type Update struct {
	ZoneID   string
	RecordID string
	Name     string
	Type     string
	Content  string
	Proxied  bool
}

// updater DNS 更新器实现
type updater struct {
	client   *CFClient
	policy   config.UpdatePolicy
}

// NewUpdater 创建 DNS 更新器
func NewUpdater(cfg config.DNSConfig) (Updater, error) {
	if cfg.APIToken == "" {
		return nil, fmt.Errorf("缺少 Cloudflare API Token")
	}

	client := &CFClient{
		apiToken: cfg.APIToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	return &updater{
		client: client,
		policy: cfg.UpdatePolicy,
	}, nil
}

// UpdateARecord 更新 A 记录
func (u *updater) UpdateARecord(ctx context.Context, zoneID, recordID, ip string) error {
	// 验证 IP 地址
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("无效的 IP 地址: %s", ip)
	}

	payload := map[string]interface{}{
		"type":    "A",
		"content": ip,
	}

	// 获取当前记录配置以保留 proxied 设置
	record, err := u.client.GetDNSRecord(ctx, zoneID, recordID)
	if err == nil && record != nil {
		payload["proxied"] = record.Proxied
		payload["name"] = record.Name
		payload["ttl"] = record.TTL
	}

	return u.client.UpdateDNSRecord(ctx, zoneID, recordID, payload)
}

// UpdateCNAMERecord 更新 CNAME 记录
func (u *updater) UpdateCNAMERecord(ctx context.Context, zoneID, recordID, target string) error {
	payload := map[string]interface{}{
		"type":    "CNAME",
		"content": target,
	}

	// 获取当前记录配置
	record, err := u.client.GetDNSRecord(ctx, zoneID, recordID)
	if err == nil && record != nil {
		payload["proxied"] = record.Proxied
		payload["name"] = record.Name
		payload["ttl"] = record.TTL
	}

	return u.client.UpdateDNSRecord(ctx, zoneID, recordID, payload)
}

// BatchUpdate 批量更新 DNS 记录
func (u *updater) BatchUpdate(ctx context.Context, updates []Update) error {
	// Cloudflare API 限流: 1200 req/5min
	// 安全起见，每秒最多 4 个请求
	rateLimiter := time.NewTicker(250 * time.Millisecond)
	defer rateLimiter.Stop()

	for i, update := range updates {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-rateLimiter.C:
		}

		var err error
		switch update.Type {
		case "A", "AAAA":
			err = u.UpdateARecord(ctx, update.ZoneID, update.RecordID, update.Content)
		case "CNAME":
			err = u.UpdateCNAMERecord(ctx, update.ZoneID, update.RecordID, update.Content)
		default:
			err = fmt.Errorf("不支持的记录类型: %s", update.Type)
		}

		if err != nil {
			return fmt.Errorf("更新 %s (记录 %d/%d) 失败: %w", update.Name, i+1, len(updates), err)
		}

		fmt.Fprintf(os.Stderr, "[%d/%d] %s -> %s\n", i+1, len(updates), update.Name, update.Content)
	}

	return nil
}

// GetBestIP 根据测速结果获取最优 IP
func (u *updater) GetBestIP(records []config.DNSRecord, results []scanner.Result) []Update {
	if len(results) == 0 || len(records) == 0 {
		return nil
	}

	// 根据策略过滤结果
	filtered := u.filterResults(results)

	updates := make([]Update, 0, len(records))

	for _, record := range records {
		// 为每条记录选择最优结果
		best := u.selectBestForRecord(record, filtered)
		if best == nil {
			continue
		}

		var content string
		switch record.Type {
		case "A", "AAAA":
			content = best.IP.String()
		case "CNAME":
			// CNAME 使用域名，这里可以配置映射规则
			content = record.Name // 保持不变
		default:
			continue
		}

		updates = append(updates, Update{
			ZoneID:   record.ZoneID,
			RecordID: record.RecordID,
			Name:     record.Name,
			Type:     record.Type,
			Content:  content,
			Proxied:  record.Proxied,
		})
	}

	return updates
}

// filterResults 根据策略过滤结果
func (u *updater) filterResults(results []scanner.Result) []scanner.Result {
	filtered := make([]scanner.Result, 0, len(results))

	for _, r := range results {
		// 根据策略过滤
		if u.policy.Mode == "best" || u.policy.Mode == "top" {
			// 使用评分排序的结果
			filtered = append(filtered, r)
		} else if u.policy.Mode == "filtered" {
			// 仅使用符合过滤条件的结果
			if r.Score >= u.policy.MinScore {
				filtered = append(filtered, r)
			}
		}
	}

	return filtered
}

// selectBestForRecord 为特定记录选择最优结果
func (u *updater) selectBestForRecord(record config.DNSRecord, results []scanner.Result) *scanner.Result {
	if len(results) == 0 {
		return nil
	}

	// A/AAAA 记录需要匹配 IP 版本
	if record.Type == "A" {
		for _, r := range results {
			if r.IP.To4() != nil {
				return &r
			}
		}
	} else if record.Type == "AAAA" {
		for _, r := range results {
			if r.IP.To4() == nil && len(r.IP) == net.IPv6len {
				return &r
			}
		}
	}

	// 默认返回第一个结果
	return &results[0]
}
