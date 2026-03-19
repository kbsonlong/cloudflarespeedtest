package scanner

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/zengshenglong/cf-speed-test/internal/config"
)

// Result 测速结果
type Result struct {
	IP            net.IP
	Sent          int
	Received      int
	LossRate      float32
	AvgDelay      time.Duration
	MaxDelay      time.Duration
	MinDelay      time.Duration
	DownloadSpeed float64 // bytes/s
	UploadSpeed   float64 // bytes/s
	Colo          string  // CDN 节点代码
	Score         float64 // 综合评分 (0-100)
	TestedAt      time.Time
}

// Filter 结果过滤器
type Filter struct {
	MaxDelay    time.Duration
	MinDelay    time.Duration
	MaxLossRate float32
}

// Scanner 扫描器接口
type Scanner interface {
	Scan(ctx context.Context) []Result
}

// scanner 扫描器实现
type scanner struct {
	cfg    config.ScanConfig
	filter Filter
	ipGen  *ipGenerator
}

// New 创建扫描器
func New(cfg config.ScanConfig) (Scanner, error) {
	// 验证配置
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return &scanner{
		cfg: cfg,
		filter: Filter{
			MaxDelay:    cfg.Filter.MaxDelay,
			MinDelay:    cfg.Filter.MinDelay,
			MaxLossRate: cfg.Filter.MaxLossRate,
		},
		ipGen: newIPGenerator(cfg.Sources),
	}, nil
}

func validateConfig(cfg config.ScanConfig) error {
	if cfg.Ping.Routines <= 0 || cfg.Ping.Routines > 1000 {
		return fmt.Errorf("无效的并发数: %d (范围: 1-1000)", cfg.Ping.Routines)
	}
	if cfg.Ping.Times <= 0 {
		return fmt.Errorf("无效的 ping 次数: %d", cfg.Ping.Times)
	}
	if cfg.Ping.Mode != "tcp" && cfg.Ping.Mode != "http" {
		return fmt.Errorf("无效的 ping 模式: %s", cfg.Ping.Mode)
	}
	return nil
}

// Scan 执行扫描
func (s *scanner) Scan(ctx context.Context) []Result {
	// 生成 IP 列表
	ips, err := s.ipGen.Generate()
	if err != nil {
		color.Red("生成 IP 列表失败: %v", err)
		return nil
	}

	if len(ips) == 0 {
		color.Yellow("没有可用的 IP")
		return nil
	}

	color.Cyan("开始延迟测速 (模式: %s, IP数量: %d, 并发: %d)",
		s.cfg.Ping.Mode, len(ips), s.cfg.Ping.Routines)

	// 延迟测速
	pingResults := s.runPing(ctx, ips)

	if len(pingResults) == 0 {
		color.Yellow("延迟测速没有可用的结果")
		return nil
	}

	color.Cyan("延迟测速完成，可用 IP: %d", len(pingResults))

	// 过滤结果
	filtered := s.filterResults(pingResults)
	if len(filtered) == 0 {
		color.Yellow("过滤后没有可用的结果")
		return nil
	}

	color.Cyan("过滤后剩余 IP: %d", len(filtered))

	// 下载测速
	if s.cfg.Speed.Enabled {
		color.Cyan("开始下载测速 (数量: %d)", s.cfg.Speed.Count)
		speedResults := s.runSpeedTest(ctx, filtered)
		if len(speedResults) > 0 {
			// 计算评分
			for i := range speedResults {
				speedResults[i].Score = calculateScore(speedResults[i])
			}
			return speedResults
		}
	}

	// 仅延迟测速时，计算评分后返回
	for i := range filtered {
		filtered[i].Score = calculateScore(filtered[i])
	}
	return filtered
}

// runPing 执行延迟测速
func (s *scanner) runPing(ctx context.Context, ips []net.IP) []Result {
	results := make([]Result, 0, len(ips))
	resultsMu := sync.Mutex{}

	control := make(chan struct{}, s.cfg.Ping.Routines)
	var wg sync.WaitGroup

	bar := progressbar.NewOptions(len(ips),
		progressbar.OptionSetDescription("延迟测速"),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWriter(color.Error),
	)

	for _, ip := range ips {
		select {
		case <-ctx.Done():
			wg.Wait()
			return results
		default:
		}

		wg.Add(1)
		control <- struct{}{}

		go func(ip net.IP) {
			defer wg.Done()
			defer func() { <-control }()
			defer bar.Add(1)

			var result Result

			if s.cfg.Ping.Mode == "http" {
				result = s.httping(ip)
			} else {
				result = s.tcping(ip)
			}

			result.IP = ip
			result.TestedAt = time.Now()

			if result.Received > 0 {
				resultsMu.Lock()
				results = append(results, result)
				resultsMu.Unlock()
			}
		}(ip)
	}

	wg.Wait()
	_ = bar.Finish()

	return results
}

// tcping TCP 延迟测试
func (s *scanner) tcping(ip net.IP) Result {
	result := Result{
		Sent:     s.cfg.Ping.Times,
		TestedAt: time.Now(),
	}

	var totalDelay time.Duration

	for i := 0; i < s.cfg.Ping.Times; i++ {
		start := time.Now()

		address := fmt.Sprintf("%s:%d", ip.String(), s.cfg.Ping.Port)
		if ip.To4() == nil {
			address = fmt.Sprintf("[%s]:%d", ip.String(), s.cfg.Ping.Port)
		}

		conn, err := net.DialTimeout("tcp", address, s.cfg.Ping.Timeout)
		if err == nil {
			result.Received++
			delay := time.Since(start)
			totalDelay += delay

			if result.MinDelay == 0 || delay < result.MinDelay {
				result.MinDelay = delay
			}
			if delay > result.MaxDelay {
				result.MaxDelay = delay
			}
			conn.Close()
		}
	}

	if result.Received > 0 {
		result.AvgDelay = totalDelay / time.Duration(result.Received)
		result.LossRate = float32(result.Sent-result.Received) / float32(result.Sent)
	}

	return result
}

// httping HTTP 延迟测试
func (s *scanner) httping(ip net.IP) Result {
	result := Result{
		Sent:     s.cfg.Ping.Times,
		TestedAt: time.Now(),
	}

	// TODO: 实现 HTTPing
	// 需要发送 HTTP 请求并测量延迟
	// 同时获取 CF-Ray 头部获取地区码

	return result
}

// runSpeedTest 执行下载测速
func (s *scanner) runSpeedTest(ctx context.Context, results []Result) []Result {
	count := s.cfg.Speed.Count
	if count > len(results) {
		count = len(results)
	}

	bar := progressbar.NewOptions(count,
		progressbar.OptionSetDescription("速度测速"),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWriter(color.Error),
	)

	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			return results[:i]
		default:
		}

		// TODO: 实现下载测速
		// 使用 EWMA 计算速度

		bar.Add(1)
	}

	_ = bar.Finish()
	return results
}

// filterResults 过滤结果
func (s *scanner) filterResults(results []Result) []Result {
	filtered := make([]Result, 0, len(results))

	for _, r := range results {
		// 延迟过滤
		if s.filter.MaxDelay > 0 && r.AvgDelay > s.filter.MaxDelay {
			continue
		}
		if s.filter.MinDelay > 0 && r.AvgDelay < s.filter.MinDelay {
			continue
		}

		// 丢包率过滤
		if s.filter.MaxLossRate < 1.0 && r.LossRate > s.filter.MaxLossRate {
			continue
		}

		filtered = append(filtered, r)
	}

	return filtered
}

// calculateScore 计算综合评分
func calculateScore(r Result) float64 {
	// 延迟评分 (40%): 延迟越低分数越高
	delayScore := 100.0
	if r.AvgDelay > 0 {
		// 假设 500ms 为最差 (0分)，0ms 为最好 (100分)
		delayScore = 100.0 * (1.0 - min(r.AvgDelay.Seconds()/0.5, 1.0))
	}

	// 丢包率评分 (30%): 丢包率越低分数越高
	lossScore := 100.0 * (1.0 - float64(r.LossRate))

	// 速度评分 (30%): 速度越高分数越高
	speedScore := 0.0
	if r.DownloadSpeed > 0 {
		// 假设 100MB/s 为满分
		speedScore = min(r.DownloadSpeed/(100*1024*1024), 1.0) * 100.0
	}

	return delayScore*0.4 + lossScore*0.3 + speedScore*0.3
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
