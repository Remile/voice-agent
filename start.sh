#!/bin/bash

# AWS Bedrock Nova 全双工语音对话系统启动脚本

echo "=== AWS Bedrock Nova 全双工语音对话系统 ==="
echo ""

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo "❌ 错误: 未安装 Go"
    echo "请访问 https://golang.org/dl/ 下载安装 Go"
    exit 1
fi

echo "✓ Go 版本: $(go version)"

# 检查 AWS 凭证
if [ -z "$AWS_ACCESS_KEY_ID" ] && [ ! -f ~/.aws/credentials ]; then
    echo "⚠️  警告: 未找到 AWS 凭证"
    echo "请运行 'aws configure' 或设置环境变量"
    read -p "是否继续? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# 检查依赖
echo ""
echo "📦 检查依赖..."
if [ ! -d "vendor" ] && [ ! -f "go.sum" ]; then
    echo "下载依赖..."
    go mod download
fi

# 编译
echo ""
echo "🔨 编译程序..."
if go build -o voice-agent main.go; then
    echo "✓ 编译成功"
else
    echo "❌ 编译失败"
    exit 1
fi

# 创建输出目录
mkdir -p output

# 运行
echo ""
echo "🚀 启动全双工语音对话系统..."
echo ""
./voice-agent

