package scanner

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

// ipGenerator IP 生成器
type ipGenerator struct {
	sources []string
	rand    *rand.Rand
}

// newIPGenerator 创建 IP 生成器
func newIPGenerator(sources []string) *ipGenerator {
	return &ipGenerator{
		sources: sources,
		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Generate 生成 IP 列表
func (g *ipGenerator) Generate() ([]net.IP, error) {
	ips := make([]net.IP, 0)

	for _, source := range g.sources {
		source = strings.TrimSpace(source)

		// 跳过空行
		if source == "" {
			continue
		}

		// 检查是否为 URL
		if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
			urlIPs, err := g.loadFromURL(source)
			if err != nil {
				return nil, fmt.Errorf("从 URL 加载 %s 失败: %w", source, err)
			}
			ips = append(ips, urlIPs...)
			continue
		}

		// 检查是否为文件路径（需要先检查，避免和 CIDR 冲突）
		if fileExists(source) {
			fileIPs, err := g.loadFromFile(source)
			if err != nil {
				return nil, fmt.Errorf("从文件加载 %s 失败: %w", source, err)
			}
			ips = append(ips, fileIPs...)
			continue
		}

		// 检查是否为 CIDR 格式
		if strings.Contains(source, "/") {
			generated, err := g.generateFromCIDR(source)
			if err != nil {
				return nil, fmt.Errorf("解析 CIDR %s 失败: %w", source, err)
			}
			ips = append(ips, generated...)
			continue
		}

		// 尝试解析为单个 IP
		if ip := net.ParseIP(source); ip != nil {
			ips = append(ips, ip)
		} else {
			return nil, fmt.Errorf("无法解析 source: %s", source)
		}
	}

	return ips, nil
}

// generateFromCIDR 从 CIDR 生成 IP
func (g *ipGenerator) generateFromCIDR(cidr string) ([]net.IP, error) {
	// 添加子网掩码
	if !strings.Contains(cidr, "/") {
		ip := net.ParseIP(cidr)
		if ip == nil {
			return nil, fmt.Errorf("无效的 IP: %s", cidr)
		}
		if ip.To4() != nil {
			cidr += "/32"
		} else {
			cidr += "/128"
		}
	}

	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	ips := make([]net.IP, 0)

	// 检查是 IPv4 还是 IPv6
	if ip.To4() != nil {
		ips = g.generateIPv4(ip, ipNet)
	} else {
		ips = g.generateIPv6(ip, ipNet)
	}

	return ips, nil
}

// generateIPv4 生成 IPv4 地址
func (g *ipGenerator) generateIPv4(ip net.IP, ipNet *net.IPNet) []net.IP {
	ips := make([]net.IP, 0)

	// 获取网络部分和掩码
	network := ip.Mask(ipNet.Mask).To4()

	// 计算主机数量
	ones, bits := ipNet.Mask.Size()
	hosts := 1 << (bits - ones)

	// /32 单个 IP
	if ones == 32 {
		return append(ips, network)
	}

	// 从每个 /24 网段随机选择一个 IP
	if hosts > 256 {
		// 遍历第三段
		for i := 0; i < 256; i++ {
			network[2] = byte(i)
			// 随机第四段
			network[3] = byte(g.rand.Intn(256))
			ip := make(net.IP, len(network))
			copy(ip, network)
			ips = append(ips, ip)
		}
	} else {
		// 小网段，生成所有 IP
		for i := 0; i < hosts; i++ {
			network[3] = byte(i)
			ip := make(net.IP, len(network))
			copy(ip, network)
			ips = append(ips, ip)
		}
	}

	return ips
}

// generateIPv6 生成 IPv6 地址
func (g *ipGenerator) generateIPv6(ip net.IP, ipNet *net.IPNet) []net.IP {
	ips := make([]net.IP, 0)

	// /128 单个 IP
	ones, _ := ipNet.Mask.Size()
	if ones == 128 {
		return append(ips, ip)
	}

	// 随机生成一些 IPv6 地址
	for i := 0; i < 100; i++ {
		newIP := make(net.IP, len(ip))
		copy(newIP, ip)

		// 随机最后 4 个字节
		for j := 12; j < 16; j++ {
			newIP[j] = byte(g.rand.Intn(256))
		}

		// 确保在网段内
		if ipNet.Contains(newIP) {
			ips = append(ips, newIP)
		}
	}

	return ips
}

// loadFromFile 从文件加载 IP
func (g *ipGenerator) loadFromFile(path string) ([]net.IP, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ips := make([]net.IP, 0)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析 CIDR
		generated, err := g.generateFromCIDR(line)
		if err != nil {
			continue // 跳过无效行
		}

		ips = append(ips, generated...)
	}

	return ips, scanner.Err()
}

// loadFromURL 从 URL 加载 IP
func (g *ipGenerator) loadFromURL(url string) ([]net.IP, error) {
	// TODO: 实现 HTTP 请求
	return nil, fmt.Errorf("URL 加载尚未实现")
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
