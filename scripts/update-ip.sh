#!/bin/bash

# Cloudflare IP Ranges 更新脚本
# 从 https://www.cloudflare.com/ips-v4 和 https://www.cloudflare.com/ips-v6 获取最新 IP 段

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

IPV4_URL="https://www.cloudflare.com/ips-v4"
IPV6_URL="https://www.cloudflare.com/ips-v6"
IPV4_FILE="$PROJECT_ROOT/ip.txt"
IPV6_FILE="$PROJECT_ROOT/ipv6.txt"

echo "=========================================="
echo "Cloudflare IP Ranges 更新脚本"
echo "=========================================="
echo ""

# 创建临时文件
TEMP_IPV4=$(mktemp)
TEMP_IPV6=$(mktemp)

trap "rm -f $TEMP_IPV4 $TEMP_IPV6" EXIT

# 获取 IPv4 段
echo "正在获取 IPv4 地址段..."
if curl -fsS "$IPV4_URL" -o "$TEMP_IPV4"; then
    if [ -s "$TEMP_IPV4" ]; then
        cp "$TEMP_IPV4" "$IPV4_FILE"
        IPV4_COUNT=$(wc -l < "$IPV4_FILE")
        echo "✓ IPv4 地址段已更新 (共 $IPV4_COUNT 个)"
    else
        echo "✗ IPv4 文件为空，跳过更新"
        exit 1
    fi
else
    echo "✗ 获取 IPv4 地址段失败"
    exit 1
fi

echo ""

# 获取 IPv6 段
echo "正在获取 IPv6 地址段..."
if curl -fsS "$IPV6_URL" -o "$TEMP_IPV6"; then
    if [ -s "$TEMP_IPV6" ]; then
        cp "$TEMP_IPV6" "$IPV6_FILE"
        IPV6_COUNT=$(wc -l < "$IPV6_FILE")
        echo "✓ IPv6 地址段已更新 (共 $IPV6_COUNT 个)"
    else
        echo "✗ IPv6 文件为空，跳过更新"
        exit 1
    fi
else
    echo "✗ 获取 IPv6 地址段失败"
    exit 1
fi

echo ""
echo "=========================================="
echo "更新完成！"
echo "=========================================="
echo ""

# 显示部分内容
echo "IPv4 地址段 (前 5 行):"
head -n 5 "$IPV4_FILE"
echo ""
echo "IPv6 地址段 (前 5 行):"
head -n 5 "$IPV6_FILE"
