# Cloudflare DNS API 集成文档

## 一、认证

### API Token

推荐使用 API Token 进行认证，需要以下权限：

| 权限 | 说明 |
|------|------|
| `Zone:Read` | 读取 Zone 信息 |
| `DNS:Edit` | 编辑 DNS 记录 |

### 创建 API Token

1. 登录 Cloudflare Dashboard
2. 进入 My Profile → API Tokens
3. 点击 "Create Token"
4. 使用模板 "Edit zone DNS" 或自定义权限

### 请求头

```http
Authorization: Bearer <API_TOKEN>
Content-Type: application/json
```

---

## 二、API 端点

### 2.1 列出 Zones

获取账户下的所有 Zone。

```http
GET https://api.cloudflare.com/client/v4/zones
```

**查询参数：**

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| name | string | 否 | 按 Zone 名称过滤 |
| status | string | 否 | Zone 状态 (active, pending, initializing) |
| page | number | 否 | 页码 |
| per_page | number | 否 | 每页数量 (1-1000) |

**响应示例：**

```json
{
  "result": [
    {
      "id": "0da42c8d2132a9ddaf714f9e7c920711",
      "name": "example.com",
      "status": "active",
      "paused": false,
      "type": "full",
      "name_servers": [
        "kate.ns.cloudflare.com",
        "rob.ns.cloudflare.com"
      ]
    }
  ],
  "success": true,
  "errors": [],
  "messages": [],
  "result_info": {
    "page": 1,
    "per_page": 20,
    "total_pages": 1,
    "total": 1
  }
}
```

### 2.2 获取 Zone 详情

```http
GET https://api.cloudflare.com/client/v4/zones/{zone_id}
```

### 2.3 列出 DNS 记录

```http
GET https://api.cloudflare.com/client/v4/zones/{zone_id}/dns_records
```

**查询参数：**

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| type | string | 否 | 记录类型 (A, AAAA, CNAME, TXT 等) |
| name | string | 否 | 记录名称 |
| content | string | 否 | 记录内容 |
| page | number | 否 | 页码 |
| per_page | number | 否 | 每页数量 |

**响应示例：**

```json
{
  "result": [
    {
      "id": "372e67954025e0ba6aaa6d586b9e0b59",
      "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
      "zone_name": "example.com",
      "name": "example.com",
      "type": "A",
      "content": "198.51.100.4",
      "proxiable": true,
      "proxied": true,
      "ttl": 1,
      "locked": false,
      "meta": {
        "auto_added": true,
        "source": "primary"
      },
      "created_on": "2014-01-01T05:20:00Z",
      "modified_on": "2014-01-01T05:20:00Z"
    }
  ],
  "success": true,
  "errors": [],
  "messages": []
}
```

### 2.4 获取 DNS 记录详情

```http
GET https://api.cloudflare.com/client/v4/zones/{zone_id}/dns_records/{dns_record_id}
```

### 2.5 更新 DNS 记录

```http
PATCH https://api.cloudflare.com/client/v4/zones/{zone_id}/dns_records/{dns_record_id}
```

**请求体 (A 记录)：**

```json
{
  "type": "A",
  "name": "example.com",
  "content": "192.0.2.1",
  "ttl": 1,
  "proxied": true
}
```

**请求体 (CNAME 记录)：**

```json
{
  "type": "CNAME",
  "name": "www.example.com",
  "content": "example.com",
  "ttl": 1,
  "proxied": false
}
```

**字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| type | string | 记录类型 (A, AAAA, CNAME, TXT, MX 等) |
| name | string | 记录名称 |
| content | string | 记录内容 (IP 地址或域名) |
| ttl | number | TTL 值，1 = 自动 |
| proxied | boolean | 是否通过 Cloudflare 代理 |

**响应示例：**

```json
{
  "result": {
    "id": "372e67954025e0ba6aaa6d586b9e0b59",
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "zone_name": "example.com",
    "name": "example.com",
    "type": "A",
    "content": "192.0.2.1",
    "proxiable": true,
    "proxied": true,
    "ttl": 1,
    "modified_on": "2014-01-01T05:20:00Z",
    "created_on": "2014-01-01T05:20:00Z"
  },
  "success": true,
  "errors": [],
  "messages": []
}
```

### 2.6 创建 DNS 记录

```http
POST https://api.cloudflare.com/client/v4/zones/{zone_id}/dns_records
```

**请求体：** 同更新 DNS 记录

### 2.7 删除 DNS 记录

```http
DELETE https://api.cloudflare.com/client/v4/zones/{zone_id}/dns_records/{dns_record_id}
```

---

## 三、错误处理

### 错误响应格式

```json
{
  "success": false,
  "errors": [
    {
      "code": 1003,
      "message": "Invalid or missing zone id."
    }
  ],
  "messages": []
}
```

### 常见错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 请求已成功接受（异步处理） |
| 1000 | 无效 API Token |
| 1001 | API Token 验证失败 |
| 1003 | 无效或缺失 zone_id |
| 1004 | 无效 DNS 记录 ID |
| 1010 | 需要订阅 |
| 1102 | 请求限流 |
| 1200 | 请求超时 |
| 81053 | DNS 记录已存在 |
| 81054 | DNS 记录不存在 |

---

## 四、使用示例 (Go)

### 4.1 获取 Zone ID

```go
package cfapi

import (
    "encoding/json"
    "net/http"
)

func GetZoneID(apiToken, domain string) (string, error) {
    req, _ := http.NewRequest("GET", "https://api.cloudflare.com/client/v4/zones?name="+domain, nil)
    req.Header.Set("Authorization", "Bearer "+apiToken)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    var result struct {
        Success bool `json:"success"`
        Result  []struct {
            ID   string `json:"id"`
            Name string `json:"name"`
        } `json:"result"`
    }

    json.NewDecoder(resp.Body).Decode(&result)

    if !result.Success || len(result.Result) == 0 {
        return "", fmt.Errorf("zone not found")
    }

    return result.Result[0].ID, nil
}
```

### 4.2 更新 A 记录

```go
func UpdateARecord(apiToken, zoneID, recordID, ip string) error {
    payload := map[string]interface{}{
        "type":    "A",
        "content": ip,
        "ttl":     1,  // 自动
        "proxied": true,
    }

    body, _ := json.Marshal(payload)
    url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, recordID)

    req, _ := http.NewRequest("PATCH", url, bytes.NewReader(body))
    req.Header.Set("Authorization", "Bearer "+apiToken)
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("update failed: %s", resp.Status)
    }

    return nil
}
```

### 4.3 批量更新

```go
func BatchUpdate(apiToken string, updates []DNSUpdate) error {
    for _, u := range updates {
        err := UpdateARecord(apiToken, u.ZoneID, u.RecordID, u.IP)
        if err != nil {
            return err
        }
        // 避免触发限流
        time.Sleep(250 * time.Millisecond)
    }
    return nil
}
```

---

## 五、SDK 使用

项目推荐使用官方 Go SDK: [cloudflare-go](https://github.com/cloudflare/cloudflare-go)

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/cloudflare/cloudflare-go"
)

func main() {
    apiToken := os.Getenv("CF_API_TOKEN")

    // 创建客户端
    api, err := cloudflare.NewWithAPIToken(apiToken)
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // 获取 Zone ID
    zoneID, err := api.ZoneIDByName("example.com")
    if err != nil {
        panic(err)
    }

    // 更新 DNS 记录
    record := cloudflare.DNSRecord{
        Type:    "A",
        Name:    "example.com",
        Content: "192.0.2.1",
        TTL:     1,
        Proxied: cloudflare.BoolPtr(true),
    }

    err = api.UpdateDNSRecord(ctx, zoneID, "record-id", record)
    if err != nil {
        panic(err)
    }

    fmt.Println("DNS record updated successfully")
}
```

---

## 六、限流策略

### 限流规则

- API Token: 1200 requests/5 minutes
- 全局限流: 10000 requests/5 minutes

### 建议策略

1. **批量操作**: 一次请求处理多个操作
2. **指数退避**: 遇到 429 错误时指数退避重试
3. **缓存**: 缓存 Zone 和 DNS 记录信息
4. **并发控制**: 限制并发请求数

### 退避重试示例

```go
func updateWithRetry(api *cloudflare.API, ctx context.Context, zoneID, recordID string, record cloudflare.DNSRecord) error {
    var err error
    for i := 0; i < 3; i++ {
        err = api.UpdateDNSRecord(ctx, zoneID, recordID, record)
        if err == nil {
            return nil
        }

        // 检查是否为限流错误
        if strings.Contains(err.Error(), "rate limit") {
            waitTime := time.Duration(1<<uint(i)) * time.Second
            time.Sleep(waitTime)
            continue
        }
        return err
    }
    return err
}
```

---

## 七、Webhook 集成

Cloudflare 支持 Webhook 通知 DNS 变更：

1. 进入 Zone → DNS → Records
2. 配置 Webhook URL
3. 记录变更时会发送 POST 请求

**Webhook Payload 示例：**

```json
{
  "event": "dns_record_updated",
  "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
  "zone_name": "example.com",
  "record": {
    "id": "372e67954025e0ba6aaa6d586b9e0b59",
    "name": "example.com",
    "type": "A",
    "content": "192.0.2.1"
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

## 八、安全建议

1. **API Token 存储**
   - 使用环境变量
   - 不要硬编码在代码中
   - 使用密钥管理服务 (如 HashiCorp Vault)

2. **权限最小化**
   - 仅授予必要的权限
   - 使用特定 Zone 限制
   - 设置 Token 过期时间

3. **审计日志**
   - Cloudflare 会记录所有 API 调用
   - 定期审查审计日志
   - 监控异常活动

---

## 九、测试

### 测试环境

Cloudflare 没有专门的测试环境，建议：

1. 使用测试域名
2. 创建单独的测试 Zone
3. 使用临时的 DNS 记录

### 测试清单

- [ ] 获取 Zone ID
- [ ] 列出 DNS 记录
- [ ] 更新 A 记录
- [ ] 更新 CNAME 记录
- [ ] 批量更新
- [ ] 错误处理
- [ ] 限流处理
- [ ] 并发请求
