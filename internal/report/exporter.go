package report

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/zengshenglong/cf-speed-test/internal/config"
	"github.com/zengshenglong/cf-speed-test/internal/scanner"
)

// Export 导出结果
func Export(results []scanner.Result, cfg config.OutputConfig) error {
	if len(results) == 0 {
		color.Yellow("没有可用的结果")
		return nil
	}

	// 按评分排序
	results = sortByScore(results)

	// 终端输出
	if cfg.Console.Enabled {
		printConsole(results, cfg.Console.Top)
	}

	// CSV 导出
	if cfg.CSV.Enabled {
		if err := exportCSV(results, cfg.CSV.Path); err != nil {
			return fmt.Errorf("CSV 导出失败: %w", err)
		}
		color.Green("CSV 结果已写入: %s", cfg.CSV.Path)
	}

	// JSON 导出
	if cfg.JSON.Enabled {
		if err := exportJSON(results, cfg.JSON.Path, cfg.JSON.Pretty); err != nil {
			return fmt.Errorf("JSON 导出失败: %w", err)
		}
		color.Green("JSON 结果已写入: %s", cfg.JSON.Path)
	}

	return nil
}

// sortByScore 按评分排序
func sortByScore(results []scanner.Result) []scanner.Result {
	sorted := make([]scanner.Result, len(results))
	copy(sorted, results)

	// 简单冒泡排序 (实际项目可用 sort.Slice)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Score < sorted[j].Score {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// printConsole 终端输出
func printConsole(results []scanner.Result, top int) {
	if top <= 0 || top > len(results) {
		top = len(results)
	}

	color.Cyan("\n测速结果 (前 %d):\n", top)

	// 检查是否有 IPv6
	hasIPv6 := false
	for i := 0; i < top; i++ {
		if len(results[i].IP) > 15 || results[i].IP.To4() == nil {
			hasIPv6 = true
			break
		}
	}

	// 表头格式
	var headerFormat, dataFormat string
	if hasIPv6 {
		headerFormat = "%-40s %-8s %-8s %-8s %-10s %-12s %-8s %-6s\n"
		dataFormat = "%-42s %-8s %-8s %-8s %-10s %-12s %-8s %-6s\n"
	} else {
		headerFormat = "%-16s %-8s %-8s %-8s %-10s %-12s %-8s %-6s\n"
		dataFormat = "%-18s %-8s %-8s %-8s %-10s %-12s %-8s %-6s\n"
	}

	// 打印表头
	color.Cyan(headerFormat,
		"IP 地址", "已发送", "已接收", "丢包率",
		"平均延迟", "下载速度", "地区码", "评分")

	// 打印数据
	for i := 0; i < top; i++ {
		r := results[i]

		ipStr := r.IP.String()
		sent := strconv.Itoa(r.Sent)
		recv := strconv.Itoa(r.Received)
		loss := strconv.FormatFloat(float64(r.LossRate), 'f', 2, 32)
		delay := strconv.FormatFloat(r.AvgDelay.Seconds()*1000, 'f', 2, 32)
		speed := formatSpeed(r.DownloadSpeed)
		colo := r.Colo
		if colo == "" {
			colo = "N/A"
		}
		score := strconv.FormatFloat(r.Score, 'f', 1, 32)

		// 根据评分着色
		var line string
		if r.Score >= 80 {
			line = color.GreenString(dataFormat, ipStr, sent, recv, loss, delay+"ms", speed, colo, score)
		} else if r.Score >= 60 {
			line = color.YellowString(dataFormat, ipStr, sent, recv, loss, delay+"ms", speed, colo, score)
		} else {
			line = dataFormat
			fmt.Printf(line, ipStr, sent, recv, loss, delay+"ms", speed, colo, score)
		}

		if r.Score >= 60 {
			fmt.Print(line)
		}
	}
	fmt.Println()
}

// formatSpeed 格式化速度
func formatSpeed(bps float64) string {
	if bps == 0 {
		return "N/A"
	}

	mbps := bps / 1024 / 1024
	return fmt.Sprintf("%.2f MB/s", mbps)
}

// exportCSV 导出 CSV
func exportCSV(results []scanner.Result, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	headers := []string{"IP 地址", "已发送", "已接收", "丢包率", "平均延迟(ms)",
		"最小延迟(ms)", "最大延迟(ms)", "下载速度(MB/s)", "地区码", "评分"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// 写入数据
	for _, r := range results {
		record := []string{
			r.IP.String(),
			strconv.Itoa(r.Sent),
			strconv.Itoa(r.Received),
			strconv.FormatFloat(float64(r.LossRate), 'f', 2, 32),
			strconv.FormatFloat(r.AvgDelay.Seconds()*1000, 'f', 2, 32),
			strconv.FormatFloat(r.MinDelay.Seconds()*1000, 'f', 2, 32),
			strconv.FormatFloat(r.MaxDelay.Seconds()*1000, 'f', 2, 32),
			strconv.FormatFloat(r.DownloadSpeed/1024/1024, 'f', 2, 32),
			r.Colo,
			strconv.FormatFloat(r.Score, 'f', 1, 32),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// exportJSON 导出 JSON
func exportJSON(results []scanner.Result, path string, pretty bool) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if pretty {
		encoder.SetIndent("", "  ")
	}

	type jsonResult struct {
		IP            string  `json:"ip"`
		Sent          int     `json:"sent"`
		Received      int     `json:"received"`
		LossRate      float32 `json:"loss_rate"`
		AvgDelay      float64 `json:"avg_delay_ms"`
		MinDelay      float64 `json:"min_delay_ms"`
		MaxDelay      float64 `json:"max_delay_ms"`
		DownloadSpeed float64 `json:"download_speed_mbps"`
		Colo          string  `json:"colo"`
		Score         float64 `json:"score"`
		TestedAt      string  `json:"tested_at"`
	}

	jsonResults := make([]jsonResult, len(results))
	for i, r := range results {
		jsonResults[i] = jsonResult{
			IP:            r.IP.String(),
			Sent:          r.Sent,
			Received:      r.Received,
			LossRate:      r.LossRate,
			AvgDelay:      r.AvgDelay.Seconds() * 1000,
			MinDelay:      r.MinDelay.Seconds() * 1000,
			MaxDelay:      r.MaxDelay.Seconds() * 1000,
			DownloadSpeed: r.DownloadSpeed / 1024 / 1024,
			Colo:          r.Colo,
			Score:         r.Score,
			TestedAt:      r.TestedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return encoder.Encode(jsonResults)
}
