# CloudflareSpeedTest Go 重构执行计划

## 一、项目概述

### 1.1 原项目分析

**项目名称**: XIU2/CloudflareSpeedTest
**功能**: 测试 Cloudflare CDN IP 延迟和速度，获取最快 IP (IPv4+IPv6)
**语言**: Go 1.18
**架构特点**: 单体应用，命令行工具

#### 原项目模块结构

```
CloudflareSpeedTest/
├── main.go           # 入口，参数解析
├── task/
│   ├── ip.go         # IP段解析、生成
│   ├── tcping.go     # TCP延迟测试
│   ├── httping.go    # HTTP延迟测试
│   └── download.go   # 下载速度测试
├── utils/
│   ├── color.go      # 终端颜色
│   ├── csv.go        # 结果导出
│   └── progress.go   # 进度条
└── script/           # shell脚本
```

#### 原项目核心流程

```
1. 解析命令行参数
2. 加载 IP 段（文件/参数）
3. 延迟测速（TCPing/HTTPing）
4. 过滤结果（延迟、丢包率）
5. 下载测速（可选）
6. 输出结果（终端/CSV）
```

### 1.2 新增功能需求

1. **自动更新 Cloudflare DNS**
   - 支持 A 记录（IPv4）
   - 支持 CNAME 记录
   - 通过 Cloudflare API 更新
   - 支持多个域名/记录同时更新

2. **代理模式（SNI 代理）**
   - 使用优选 IP 作为代理服务器
   - 支持本地透明代理
   - 无需修改 hosts 文件

### 1.3 重构目标

1. **模块化设计** - 清晰的模块边界，易于维护和扩展
2. **可配置化** - 支持 YAML/TOML 配置文件
3. **可测试性** - 每个模块可独立测试
4. **可观测性** - 结构化日志、指标收集
5. **生产就绪** - 错误处理、优雅关闭、健康检查

---

## 二、新架构设计

### 2.1 项目结构

```
cf-speed-test/
├── cmd/
│   └── cfst/                    # 主程序入口
│       └── main.go
│
├── internal/                    # 内部包，不对外暴露
│   ├── config/                  # 配置管理
│   │   ├── config.go           # 配置结构定义
│   │   ├── loader.go           # 配置加载器
│   │   └── validator.go        # 配置验证
│   │
│   ├── scanner/                 # IP 扫描测速核心
│   │   ├── scanner.go          # 扫描器接口
│   │   ├── ip_generator.go     # IP 生成器
│   │   ├── ping.go             # 延迟测试（TCP/HTTP）
│   │   ├── speed.go            # 速度测试
│   │   ├── filter.go           # 结果过滤
│   │   └── result.go           # 结果数据结构
│   │
│   ├── dns/                     # DNS 更新模块
│   │   ├── client.go           # Cloudflare API 客户端
│   │   ├── updater.go          # DNS 更新器
│   │   └── types.go            # DNS 记录类型
│   │
│   ├── proxy/                   # 代理模式（可选）
│   │   ├── server.go           # 代理服务器
│   │   ├── handler.go          # 请求处理
│   │   └── router.go           # 路由规则
│   │
│   └── report/                  # 结果输出
│       ├── exporter.go         # 多格式导出
│       ├── console.go          # 终端输出
│       ├── csv.go              # CSV 导出
│       └── json.go             # JSON 导出
│
├── pkg/                         # 外部可用包
│   ├── cfapi/                   # Cloudflare API 封装
│   │   ├── client.go
│   │   ├── dns.go
│   │   └── zones.go
│   │
│   └── http/                    # HTTP 工具包
│       ├── tcping/
│       │   └── tcping.go       # TCP 连接测试
│       └── download/
│           └── download.go     # 下载速度测试
│
├── docs/
│   ├── PLAN.md                  # 本执行计划
│   ├── API.md                   # API 文档
│   └── CONFIG.md                # 配置说明
│
├── configs/
│   └── config.yaml              # 默认配置
│
├── test/
│   ├── ip.txt                   # IP 段文件
│   └── config.example.yaml      # 配置示例
│
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── LICENSE
```

### 2.2 核心模块设计

#### 2.2.1 配置管理 (internal/config)

```go
type Config struct {
    // 扫描配置
    Scan ScanConfig `yaml:"scan"`

    // DNS 配置
    DNS DNSConfig `yaml:"dns"`

    // 代理配置（可选）
    Proxy ProxyConfig `yaml:"proxy,omitempty"`

    // 输出配置
    Output OutputConfig `yaml:"output"`

    // 日志配置
    Log LogConfig `yaml:"log"`
}

type ScanConfig struct {
    // IP 源
    Sources []string `yaml:"sources"`

    // 延迟测试
    Ping PingConfig `yaml:"ping"`

    // 速度测试
    Speed SpeedConfig `yaml:"speed"`

    // 过滤条件
    Filter FilterConfig `yaml:"filter"`
}

type DNSConfig struct {
    // Cloudflare API
    APIToken  string `yaml:"api_token"`
    AccountID string `yaml:"account_id"`

    // 记录配置
    Records []DNSRecord `yaml:"records"`
}
```

#### 2.2.2 扫描器 (internal/scanner)

```go
type Scanner interface {
    // 执行扫描
    Scan(ctx context.Context) <-chan Result

    // 添加 IP 段
    AddCIDR(cidr string) error

    // 设置过滤器
    SetFilter(f Filter)

    // 获取结果
    GetResults() []Result
}

type Result struct {
    IP         net.IP
    Sent       int
    Received   int
    LossRate   float32
    AvgDelay   time.Duration
    MaxDelay   time.Duration
    MinDelay   time.Duration
    DownloadSpeed float64  // bytes/s
    UploadSpeed   float64  // bytes/s
    Colo        string    // CDN 节点代码
    TestedAt    time.Time
}
```

#### 2.2.3 DNS 更新器 (internal/dns)

```go
type Updater interface {
    // 更新 A 记录
    UpdateARecord(ctx context.Context, zoneID, recordID, ip string) error

    // 更新 CNAME 记录
    UpdateCNAMERecord(ctx context.Context, zoneID, recordID, target string) error

    // 批量更新
    BatchUpdate(ctx context.Context, updates []Update) error

    // 获取最优 IP 的记录
    GetBestIP(records []DNSRecord, results []scanner.Result) []Update
}

type DNSRecord struct {
    ZoneID   string `yaml:"zone_id"`
    RecordID string `yaml:"record_id"`
    Name     string `yaml:"name"`
    Type     string `yaml:"type"`  // A, CNAME
    Proxied  bool   `yaml:"proxied"`
}

type Update struct {
    ZoneID   string
    RecordID string
    Name     string
    Type     string
    Content  string
    Proxied  bool
}
```

### 2.3 Cloudflare API 集成

#### API 端点

| 操作 | 方法 | 端点 |
|------|------|------|
| 列出 Zones | GET | /zones |
| 列出 DNS 记录 | GET | /zones/{zone_id}/dns_records |
| 获取 DNS 记录 | GET | /zones/{zone_id}/dns_records/{record_id} |
| 更新 DNS 记录 | PATCH | /zones/{zone_id}/dns_records/{record_id} |
| 创建 DNS 记录 | POST | /zones/{zone_id}/dns_records |

#### 请求头

```
Authorization: Bearer <API_Token>
Content-Type: application/json
```

---

## 三、实施步骤

### Phase 1: 基础设施 (预计 2-3 天)

#### 1.1 项目初始化
- [x] 创建目录结构
- [ ] 初始化 go.mod
- [ ] 创建 Makefile
- [ ] 配置 GitHub Actions

#### 1.2 配置管理模块
- [ ] 定义配置结构
- [ ] 实现 YAML 加载器
- [ ] 实现配置验证
- [ ] 环境变量支持
- [ ] 命令行参数覆盖

#### 1.3 日志系统
- [ ] 集成 zap/logrus
- [ ] 结构化日志
- [ ] 日志级别���制
- [ ] 日志轮转

### Phase 2: 核心扫描模块 (预计 3-4 天)

#### 2.1 IP 生成器
- [ ] 从文件加载 IP 段
- [ ] 从 URL 加载 IP 段
- [ ] CIDR 解析
- [ ] IP 随机生成
- [ ] IPv6 支持

#### 2.2 延迟测试
- [ ] TCPing 实现
- [ ] HTTPing 实现
- [ ] 并发控制
- [ ] 超时处理
- [ ] 地区码识别

#### 2.3 速度测试
- [ ] 下载速度测试
- [ ] EWMA 速度计算
- [ ] 进度跟踪
- [ ] 超时处理

#### 2.4 结果处理
- [ ] 结果排序
- [ ] 延迟过滤
- [ ] 丢包率过滤
- [ ] 速度过滤

### Phase 3: Cloudflare API 集成 (预计 2 天)

#### 3.1 API 客户端
- [ ] HTTP 客户端封装
- [ ] 认证处理
- [ ] 请求重试
- [ ] 限流处理

#### 3.2 DNS 操作
- [ ] Zone 查询
- [ ] DNS 记录查询
- [ ] DNS 记录更新
- [ ] 批量更新

#### 3.3 自动更新逻辑
- [ ] 最优 IP 选择
- [ ] 更新策略配置
- [ ] 变更通知
- [ ] 回滚机制

### Phase 4: 代理模式 (可选，预计 3 天)

#### 4.1 代理服务器
- [ ] HTTP 代理实现
- [ ] HTTPS 代理实现
- [ ] SNI 处理
- [ ] 连接池管理

#### 4.2 路由规则
- [ ] 域名匹配
- [ ] IP 分配
- [ ] 健康检查
- [ ] 故障转移

### Phase 5: 输出与报告 (预计 1-2 天)

#### 5.1 导出功能
- [ ] CSV 导出
- [ ] JSON 导出
- [ ] 终端表格输出
- [ ] 模板支持

#### 5.2 通知功能
- [ ] Webhook 支持
- [ ] 邮件通知
- [ ] Telegram Bot
- [ ] 企业微信

### Phase 6: 测试与文档 (预计 2-3 天)

#### 6.1 测试
- [ ] 单元测试
- [ ] 集成测试
- [ ] 压力测试
- [ ] 基准测试

#### 6.2 文档
- [ ] API 文档
- [ ] 配置说明
- [ ] 部署指南
- [ ] FAQ

---

## 四、配置文件示例

### config.yaml

```yaml
# 扫描配置
scan:
  # IP 源（文件路径或 URL）
  sources:
    - ./ip.txt
    - https://example.com/cloudflare-ips.txt

  # 延迟测试
  ping:
    mode: tcp  # tcp 或 http
    port: 443
    times: 4
    routines: 200
    timeout: 1s

    # HTTPing 特定配置
    http:
      url: https://cf.xiu2.xyz/url
      status_codes: [200, 301, 302]
      expected_colo: []  # 空 = 所有地区

  # 速度测试
  speed:
    enabled: true
    count: 10
    timeout: 10s
    min_speed: 5.0  # MB/s

  # 过滤条件
  filter:
    max_delay: 200ms
    min_delay: 40ms
    max_loss_rate: 0.2

# Cloudflare DNS 配置
dns:
  # API 凭证
  api_token: ${CF_API_TOKEN}  # 从环境变量读取
  account_id: ${CF_ACCOUNT_ID}

  # 记录配置
  records:
    - zone_id: ${ZONE_ID}
      record_id: ${RECORD_ID}
      name: example.com
      type: A
      proxied: true

    - zone_id: ${ZONE_ID}
      record_id: ${RECORD_ID_2}
      name: www.example.com
      type: CNAME
      proxied: false

  # 更新策略
  update_policy:
    mode: best  # best: 最优, filtered: 过滤后最优, top: 前 N 个
    min_score: 60  # 最小评分 (0-100)
    max_records: 3  # 最多更新记录数

# 代理配置（可选）
proxy:
  enabled: false
  listen: ":8080"
  mode: sni  # sni, forward

# 输出配置
output:
  console:
    enabled: true
    top: 10

  csv:
    enabled: true
    path: "./result.csv"

  json:
    enabled: true
    path: "./result.json"
    pretty: true

# 日志配置
log:
  level: info  # debug, info, warn, error
  format: json  # json, text
  output: stdout  # stdout, stderr, file path
```

---

## 五、使用示例

### 5.1 基本测速

```bash
# 使用默认配置
./cfst

# 指定配置文件
./cfst -c /path/to/config.yaml

# 命令行参数覆盖
./cfst -n 500 -tl 150 -o result.csv
```

### 5.2 测速并更新 DNS

```bash
# 测速后自动更新 Cloudflare DNS
./cfst --update-dns

# 仅更新 DNS（使用上次结果）
./cfst --dns-only
```

### 5.3 代理模式

```bash
# 启动 SNI 代理
./cfst --proxy --proxy-listen :8080

# 使用代理
export https_proxy=http://localhost:8080
curl https://example.com
```

### 5.4 定时任务 (cron)

```cron
# 每小时测速并更新 DNS
0 * * * * /usr/local/bin/cfst -c /etc/cfst/config.yaml --update-dns
```

---

## 六、技术栈

| 类别 | 技术选择 |
|------|---------|
| 语言 | Go 1.21+ |
| 配置 | yaml.v3 |
| 日志 | zap |
| HTTP | net/http, fasthttp |
| 并发 | goroutines, channels |
| 测试 | testing, testify |
| CI/CD | GitHub Actions |

### 主要依赖

```go
require (
    github.com/spf13/cobra v1.8.0        // CLI
    github.com/spf13/viper v1.18.0       // 配置
    go.uber.org/zap v1.26.0              // 日志
    gopkg.in/yaml.v3 v3.0.1              // YAML
    github.com/cloudflare/cloudflare-go v0.85.0  // CF API
    github.com/fatih/color v1.16.0       // 终端颜色
    github.com/schollz/progressbar/v3 v3.14.1  // 进度条
)
```

---

## 七、风险与注意事项

### 7.1 API 限流

- Cloudflare API 有 1200 req/min 限流
- 大量 DNS 记录更新时需要考虑分批处理

### 7.2 DNS 传播

- DNS 更新后需要等待传播时间
- TTL 设置影响生效速度
- 建议设置合理的 TTL 值

### 7.3 IP 测速准确性

- 网络波动影响测速结果
- 建议多次测速取平均值
- 不同时段结果可能不同

### 7.4 代理模式合规性

- 仅用于合法用途
- 需遵守 Cloudflare ToS
- 注意隐私和法律责任

---

## 八、后续扩展

1. **Web 界面** - 可视化配置和监控
2. **数据库存储** - 历史数据分析和趋势
3. **多 CDN 支持** - 支持 Cloudflare 外的其他 CDN
4. **智能调度** - 基于地理位置和时间的智能 IP 选择
5. **分布式测速** - 多节点协同测速
