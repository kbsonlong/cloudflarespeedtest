# CloudflareSpeedTest Go 重构版

> Cloudflare CDN IP 测速工具，支持自动更新 DNS 记录 (A/CNAME)

## 功能特性

- **延迟测试**: TCPing / HTTPing 两种模式
- **速度测试**: 下载速度测速，自动筛选最优 IP
- **DNS 更新**: 自动更新 Cloudflare DNS (A 记录、CNAME 记录)
- **多格式输出**: 终端表格、CSV、JSON
- **配置灵活**: 支持 YAML 配置文件、命令行参数、环境变量
- **IPv6 支持**: 完整支持 IPv6 测速

## 快速开始

### 安装

```bash
# 克隆仓库
git clone https://github.com/zengshenglong/cf-speed-test.git
cd cf-speed-test

# 下载依赖
go mod download

# 编译
make build

# 安装到 $GOPATH/bin
make install
```

### 基本使用

```bash
# 使用默认配置
./cfst

# 指定配置文件
./cfst -c /path/to/config.yaml

# 命令行参数
./cfst -n 500 -tl 150 -sl 5 -o result.csv

# 测速并更新 DNS
./cfst --update-dns
```

## 配置

### 环境变量

```bash
export CF_API_TOKEN=your_api_token
export CF_ACCOUNT_ID=your_account_id
export CF_ZONE_ID=your_zone_id
```

### 配置文件

```yaml
scan:
  sources:
    - ./ip.txt
  ping:
    mode: tcp
    port: 443
    times: 4
    routines: 200
  speed:
    enabled: true
    count: 10
  filter:
    max_delay: 200ms
    max_loss_rate: 0.2

dns:
  api_token: ${CF_API_TOKEN}
  records:
    - zone_id: ${CF_ZONE_ID}
      name: example.com
      type: A
      proxied: true

output:
  csv:
    path: "./result.csv"
```

## 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-c, --config` | 配置文件路径 | config.yaml |
| `-f, --ip-file` | IP 段文件 | ip.txt |
| `-n, --routines` | 并发数 | 200 |
| `-t, --ping-times` | Ping 次数 | 4 |
| `-dn, --speed-count` | 下载测速数量 | 10 |
| `-tl, --max-delay` | 最大延迟 | 200ms |
| `-sl, --min-speed` | 最小速度 (MB/s) | 0 |
| `-p, --top` | 显示结果数量 | 10 |
| `-o, --output` | 输出文件路径 | result.csv |
| `--update-dns` | 更新 DNS | false |
| `--proxy` | 启用代理模式 | false |

## DNS 更新

### A 记录更新

```yaml
dns:
  api_token: your_token
  records:
    - zone_id: abc123
      name: example.com
      type: A
      proxied: true
```

### CNAME 记录更新

```yaml
dns:
  api_token: your_token
  records:
    - zone_id: abc123
      name: www.example.com
      type: CNAME
      proxied: false
```

## 项目结构

```
cf-speed-test/
├── cmd/cfst/          # 主程序
├── internal/
│   ├── config/        # 配置管理
│   ├── scanner/       # IP 扫描测速
│   ├── dns/           # DNS 更新
│   └── report/        # 结果输出
├── pkg/               # 外部可用包
├── docs/              # 文档
└── configs/           # 配置示例
```

## 开发

```bash
# 运行测试
make test

# 代码检查
make lint

# 格式化
make fmt

# 交叉编译
make build-all
```

## 文档

- [执行计划](docs/PLAN.md)
- [API 文档](docs/API.md)
- [配置说明](docs/CONFIG.md)

## 许可证

MIT License

## 致谢

本项目基于 [XIU2/CloudflareSpeedTest](https://github.com/XIU2/CloudflareSpeedTest) 重构。
