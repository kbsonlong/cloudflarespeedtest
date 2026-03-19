package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	ipv4URL = "https://www.cloudflare.com/ips-v4"
	ipv6URL = "https://www.cloudflare.com/ips-v6"
	ipv4File = "ip.txt"
	ipv6File = "ipv6.txt"
)

func main() {
	fmt.Println("==========================================")
	fmt.Println("Cloudflare IP Ranges 更新工具")
	fmt.Println("==========================================")
	fmt.Println()

	// 获取项目根目录
	projectRoot := getProjectRoot()
	fmt.Printf("项目根目录: %s\n\n", projectRoot)

	client := &http.Client{Timeout: 10 * time.Second}

	// 获取 IPv4
	ipv4, err := fetchIPRanges(client, ipv4URL)
	if err != nil {
		fmt.Printf("获取 IPv4 失败: %v\n", err)
		os.Exit(1)
	}

	// 获取 IPv6
	ipv6, err := fetchIPRanges(client, ipv6URL)
	if err != nil {
		fmt.Printf("获取 IPv6 失败: %v\n", err)
		os.Exit(1)
	}

	// 写入文件 (使用项目根目录的绝对路径)
	ipv4Path := filepath.Join(projectRoot, ipv4File)
	ipv6Path := filepath.Join(projectRoot, ipv6File)

	if err := writeIPFile(ipv4Path, ipv4); err != nil {
		fmt.Printf("写入 IPv4 文件失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ IPv4: %d 个地址段 -> %s\n", len(ipv4), ipv4Path)

	if err := writeIPFile(ipv6Path, ipv6); err != nil {
		fmt.Printf("写入 IPv6 文件失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ IPv6: %d 个地址段 -> %s\n", len(ipv6), ipv6Path)

	fmt.Println()
	fmt.Println("更新完成!")
}

// getProjectRoot 获取项目根目录
func getProjectRoot() string {
	// 首先尝试使用环境变量 GITHUB_WORKSPACE (GitHub Actions)
	if workspace := os.Getenv("GITHUB_WORKSPACE"); workspace != "" {
		return workspace
	}

	// 获取当前工作目录
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	return findProjectRoot(cwd)
}

// findProjectRoot 从当前目录向上查找项目根目录
func findProjectRoot(startDir string) string {
	dir := startDir
	maxDepth := 10 // 防止无限循环

	for i := 0; i < maxDepth; i++ {
		// 检查是否存在 go.mod 文件
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		// 检查是否存在 .git 目录
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}

		// 如果当前目录名是 scripts，返回上级目录
		if filepath.Base(dir) == "scripts" {
			return filepath.Dir(dir)
		}

		// 到达根目录
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// 未找到，返回当前目录
	return startDir
}

func fetchIPRanges(client *http.Client, url string) ([]string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var ranges []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			ranges = append(ranges, line)
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return nil, err
	}

	sort.Strings(ranges)
	return ranges, nil
}

func writeIPFile(filename string, ranges []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入头部
	fmt.Fprintf(file, "# Cloudflare IP Ranges\n")
	fmt.Fprintf(file, "# 生成时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "# 来源: https://www.cloudflare.com/ips/\n\n")

	for _, r := range ranges {
		fmt.Fprintln(file, r)
	}

	return nil
}
