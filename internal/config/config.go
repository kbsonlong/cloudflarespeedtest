package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// Config 是应用程序的主配置结构
type Config struct {
	Scan   ScanConfig   `yaml:"scan"`
	DNS    DNSConfig    `yaml:"dns"`
	Proxy  ProxyConfig  `yaml:"proxy,omitempty"`
	Output OutputConfig `yaml:"output"`
	Log    LogConfig    `yaml:"log"`
}

// ScanConfig 扫描配置
type ScanConfig struct {
	Sources []string  `yaml:"sources"`
	Ping    PingConfig `yaml:"ping"`
	Speed   SpeedConfig `yaml:"speed"`
	Filter  FilterConfig `yaml:"filter"`
}

// PingConfig 延迟测试配置
type PingConfig struct {
	Mode     string        `yaml:"mode"`      // tcp, http
	Port     int           `yaml:"port"`      // 端口号
	Times    int           `yaml:"times"`     // 测试次数
	Routines int           `yaml:"routines"`  // 并发数
	Timeout  time.Duration `yaml:"timeout"`   // 超时时间

	// HTTPing 特定配置
	HTTP struct {
		URL          string   `yaml:"url"`
		StatusCodes  []int    `yaml:"status_codes"`
		ExpectedColo []string `yaml:"expected_colo"`
	} `yaml:"http,omitempty"`
}

// SpeedConfig 速度测试配置
type SpeedConfig struct {
	Enabled bool          `yaml:"enabled"`
	Count   int           `yaml:"count"`
	Timeout time.Duration `yaml:"timeout"`
	MinSpeed float64      `yaml:"min_speed"` // MB/s
}

// FilterConfig 过滤条件配置
type FilterConfig struct {
	MaxDelay    time.Duration `yaml:"max_delay"`
	MinDelay    time.Duration `yaml:"min_delay"`
	MaxLossRate float32       `yaml:"max_loss_rate"`
}

// DNSConfig DNS 更新配置
type DNSConfig struct {
	APIToken  string     `yaml:"api_token"`
	AccountID string     `yaml:"account_id"`
	ZoneID    string     `yaml:"zone_id"`
	Records   []DNSRecord `yaml:"records"`
	UpdatePolicy UpdatePolicy `yaml:"update_policy"`
}

// DNSRecord DNS 记录配置
type DNSRecord struct {
	ZoneID   string `yaml:"zone_id"`
	RecordID string `yaml:"record_id"`
	Name     string `yaml:"name"`
	Type     string `yaml:"type"` // A, CNAME, AAAA
	Proxied  bool   `yaml:"proxied"`
}

// UpdatePolicy 更新策略
type UpdatePolicy struct {
	Mode      string  `yaml:"mode"`       // best, filtered, top
	MinScore  float64 `yaml:"min_score"`  // 0-100
	MaxRecords int    `yaml:"max_records"`
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Enabled bool   `yaml:"enabled"`
	Listen  string `yaml:"listen"`
	Mode    string `yaml:"mode"` // sni, forward
}

// OutputConfig 输出配置
type OutputConfig struct {
	Console ConsoleOutput `yaml:"console"`
	CSV     CSVOutput     `yaml:"csv"`
	JSON    JSONOutput    `yaml:"json"`
}

// ConsoleOutput 终端输出配置
type ConsoleOutput struct {
	Enabled bool `yaml:"enabled"`
	Top     int  `yaml:"top"`
}

// CSVOutput CSV 输出配置
type CSVOutput struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

// JSONOutput JSON 输出配置
type JSONOutput struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
	Pretty  bool   `yaml:"pretty"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, text
	Output string `yaml:"output"` // stdout, stderr, file path
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Scan: ScanConfig{
			Sources: []string{"./ip.txt"},
			Ping: PingConfig{
				Mode:     "tcp",
				Port:     443,
				Times:    4,
				Routines: 200,
				Timeout:  time.Second,
				HTTP: struct {
					URL          string   `yaml:"url"`
					StatusCodes  []int    `yaml:"status_codes"`
					ExpectedColo []string `yaml:"expected_colo"`
				}{
					URL:         "https://cf.xiu2.xyz/url",
					StatusCodes: []int{200, 301, 302},
				},
			},
			Speed: SpeedConfig{
				Enabled:  true,
				Count:    10,
				Timeout:  10 * time.Second,
				MinSpeed: 0,
			},
			Filter: FilterConfig{
				MaxDelay:    200 * time.Millisecond,
				MinDelay:    40 * time.Millisecond,
				MaxLossRate: 0.2,
			},
		},
		Output: OutputConfig{
			Console: ConsoleOutput{
				Enabled: true,
				Top:     10,
			},
			CSV: CSVOutput{
				Enabled: true,
				Path:    "result.csv",
			},
			JSON: JSONOutput{
				Enabled: false,
				Path:    "result.json",
				Pretty:  true,
			},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
	}
}

// Load 从文件加载配置
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		// 尝试默认路径
		for _, p := range []string{"./config.yaml", "./configs/config.yaml", "/etc/cfst/config.yaml"} {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件失败: %w", err)
		}
	}

	// 环境变量覆盖
	cfg.loadFromEnv()

	return cfg, nil
}

func (c *Config) loadFromEnv() {
	if token := os.Getenv("CF_API_TOKEN"); token != "" {
		c.DNS.APIToken = token
	}
	if id := os.Getenv("CF_ACCOUNT_ID"); id != "" {
		c.DNS.AccountID = id
	}
	if id := os.Getenv("CF_ZONE_ID"); id != "" {
		c.DNS.ZoneID = id
	}
}

// MergeFlags 合并命令行参数到配置
func MergeFlags(cfg *Config, flags *pflag.FlagSet) error {
	// IP 源
	if ipFile, _ := flags.GetString("ip-file"); ipFile != "" {
		cfg.Scan.Sources = []string{ipFile}
	}
	if ipText, _ := flags.GetString("ip-text"); ipText != "" {
		cfg.Scan.Sources = append(cfg.Scan.Sources, ipText)
	}

	// Ping 配置
	if routines, _ := flags.GetInt("routines"); flags.Changed("routines") {
		cfg.Scan.Ping.Routines = routines
	}
	if times, _ := flags.GetInt("ping-times"); flags.Changed("ping-times") {
		cfg.Scan.Ping.Times = times
	}
	if mode, _ := flags.GetString("ping-mode"); flags.Changed("ping-mode") {
		cfg.Scan.Ping.Mode = mode
	}
	if timeout, _ := flags.GetDuration("ping-timeout"); flags.Changed("ping-timeout") {
		cfg.Scan.Ping.Timeout = timeout
	}

	// 速度测试配置
	if count, _ := flags.GetInt("speed-count"); flags.Changed("speed-count") {
		cfg.Scan.Speed.Count = count
	}
	if timeout, _ := flags.GetDuration("speed-timeout"); flags.Changed("speed-timeout") {
		cfg.Scan.Speed.Timeout = timeout
	}
	if speed, _ := flags.GetFloat64("min-speed"); flags.Changed("min-speed") {
		cfg.Scan.Speed.MinSpeed = speed
	}
	if disable, _ := flags.GetBool("disable-download"); flags.Changed("disable-download") && disable {
		cfg.Scan.Speed.Enabled = false
	}

	// 过滤配置
	if maxDelay, _ := flags.GetDuration("max-delay"); flags.Changed("max-delay") {
		cfg.Scan.Filter.MaxDelay = maxDelay
	}
	if minDelay, _ := flags.GetDuration("min-delay"); flags.Changed("min-delay") {
		cfg.Scan.Filter.MinDelay = minDelay
	}
	if lossRate, _ := flags.GetFloat32("max-loss-rate"); flags.Changed("max-loss-rate") {
		cfg.Scan.Filter.MaxLossRate = lossRate
	}

	// 输出配置
	if top, _ := flags.GetInt("top"); flags.Changed("top") {
		cfg.Output.Console.Top = top
	}
	if output, _ := flags.GetString("output"); flags.Changed("output") {
		cfg.Output.CSV.Path = output
	}
	if json, _ := flags.GetBool("json"); flags.Changed("json") {
		cfg.Output.JSON.Enabled = json
	}

	// 代理配置
	if proxy, _ := flags.GetBool("proxy"); flags.Changed("proxy") {
		cfg.Proxy.Enabled = proxy
	}
	if listen, _ := flags.GetString("proxy-listen"); flags.Changed("proxy-listen") {
		cfg.Proxy.Listen = listen
	}

	return nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Scan.Ping.Mode != "tcp" && c.Scan.Ping.Mode != "http" {
		return fmt.Errorf("无效的 ping 模式: %s", c.Scan.Ping.Mode)
	}

	if c.Scan.Ping.Routines <= 0 || c.Scan.Ping.Routines > 1000 {
		return fmt.Errorf("无效的并发数: %d (1-1000)", c.Scan.Ping.Routines)
	}

	if c.Scan.Ping.Port <= 0 || c.Scan.Ping.Port > 65535 {
		return fmt.Errorf("无效的端口号: %d", c.Scan.Ping.Port)
	}

	return nil
}
