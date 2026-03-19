package dns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	baseURL = "https://api.cloudflare.com/client/v4"
)

// CFClient Cloudflare API 客户端
type CFClient struct {
	apiToken   string
	accountID  string
	httpClient *http.Client
}

// DNSRecord DNS 记录
type DNSRecord struct {
	ID       string `json:"id"`
	ZoneID   string `json:"zone_id"`
	ZoneName string `json:"zone_name"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	Proxied  bool   `json:"proxied"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"`
}

// APIResponse API 响应基础结构
type APIResponse struct {
	Success bool        `json:"success"`
	Errors  []APIError  `json:"errors"`
	Messages []string   `json:"messages"`
}

// APIError API 错误
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ZoneResponse Zone 列表响应
type ZoneResponse struct {
	APIResponse
	Result []Zone `json:"result"`
}

// Zone Zone 信息
type Zone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Paused bool   `json:"paused"`
}

// DNSRecordsResponse DNS 记录列表响应
type DNSRecordsResponse struct {
	APIResponse
	Result     []DNSRecord `json:"result"`
	ResultInfo ResultInfo  `json:"result_info"`
}

// ResultInfo 分页信息
type ResultInfo struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
	Total      int `json:"total"`
}

// NewCFClient 创建 Cloudflare API 客户端
func NewCFClient(apiToken, accountID string) *CFClient {
	return &CFClient{
		apiToken:  apiToken,
		accountID: accountID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest 执行 HTTP 请求
func (c *CFClient) doRequest(ctx context.Context, method, url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// handleResponse 处理 API 响应
func (c *CFClient) handleResponse(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if !apiResp.Success {
		if len(apiResp.Errors) > 0 {
			return fmt.Errorf("API 错误 (code %d): %s", apiResp.Errors[0].Code, apiResp.Errors[0].Message)
		}
		return fmt.Errorf("API 返回失败")
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("解析结果失败: %w", err)
		}
	}

	return nil
}

// ListZones 列出所有 Zones
func (c *CFClient) ListZones(ctx context.Context, name string) ([]Zone, error) {
	url := baseURL + "/zones"
	if name != "" {
		url += "?name=" + name
	}

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var result ZoneResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Result, nil
}

// GetZoneID 获取 Zone ID
func (c *CFClient) GetZoneID(ctx context.Context, name string) (string, error) {
	zones, err := c.ListZones(ctx, name)
	if err != nil {
		return "", err
	}

	if len(zones) == 0 {
		return "", fmt.Errorf("zone not found: %s", name)
	}

	return zones[0].ID, nil
}

// ListDNSRecords 列出 DNS 记录
func (c *CFClient) ListDNSRecords(ctx context.Context, zoneID, recordType, name string) ([]DNSRecord, error) {
	url := baseURL + "/zones/" + zoneID + "/dns_records"

	// 构建查询参数
	params := make([]string, 0)
	if recordType != "" {
		params = append(params, "type="+recordType)
	}
	if name != "" {
		params = append(params, "name="+name)
	}
	if len(params) > 0 {
		url += "?" + params[0]
		for i := 1; i < len(params); i++ {
			url += "&" + params[i]
		}
	}

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var result DNSRecordsResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Result, nil
}

// GetDNSRecord 获取 DNS 记录详情
func (c *CFClient) GetDNSRecord(ctx context.Context, zoneID, recordID string) (*DNSRecord, error) {
	url := baseURL + "/zones/" + zoneID + "/dns_records/" + recordID

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		APIResponse
		Result DNSRecord `json:"result"`
	}

	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result.Result, nil
}

// CreateDNSRecord 创建 DNS 记录
func (c *CFClient) CreateDNSRecord(ctx context.Context, zoneID string, record map[string]interface{}) error {
	url := baseURL + "/zones/" + zoneID + "/dns_records"

	body, err := json.Marshal(record)
	if err != nil {
		return err
	}

	resp, err := c.doRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return err
	}

	return c.handleResponse(resp, nil)
}

// UpdateDNSRecord 更新 DNS 记录
func (c *CFClient) UpdateDNSRecord(ctx context.Context, zoneID, recordID string, record map[string]interface{}) error {
	url := baseURL + "/zones/" + zoneID + "/dns_records/" + recordID

	body, err := json.Marshal(record)
	if err != nil {
		return err
	}

	resp, err := c.doRequest(ctx, http.MethodPatch, url, body)
	if err != nil {
		return err
	}

	return c.handleResponse(resp, nil)
}

// DeleteDNSRecord 删除 DNS 记录
func (c *CFClient) DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error {
	url := baseURL + "/zones/" + zoneID + "/dns_records/" + recordID

	resp, err := c.doRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	return c.handleResponse(resp, nil)
}

// FindRecordByName 根据名称查找记录
func (c *CFClient) FindRecordByName(ctx context.Context, zoneID, name string) (*DNSRecord, error) {
	records, err := c.ListDNSRecords(ctx, zoneID, "", name)
	if err != nil {
		return nil, err
	}

	for _, r := range records {
		if r.Name == name {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("record not found: %s", name)
}

// ParseTTL 解析 TTL 值
func ParseTTL(ttl string) (int, error) {
	if ttl == "auto" || ttl == "1" {
		return 1, nil
	}
	return strconv.Atoi(ttl)
}
