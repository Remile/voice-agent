#!/bin/bash

# Voice Agent 启动脚本

echo "=== AWS Bedrock Nova 语音对话系统 ==="
echo ""

# 检查是否存在 .env 文件
if [ -f .env ]; then
    echo "✓ 加载 .env 配置文件"
    export $(cat .env | grep -v '^#' | xargs)
fi

# 检查 AWS 凭证
if [ -z "$AWS_ACCESS_KEY_ID" ] && [ -z "$AWS_PROFILE" ]; then
    echo "⚠️  警告：未检测到 AWS 凭证"
    echo ""
    echo "请通过以下方式之一配置 AWS 凭证："
    echo "  1. 运行 'aws configure'"
    echo "  2. 设置环境变量 AWS_ACCESS_KEY_ID 和 AWS_SECRET_ACCESS_KEY"
    echo "  3. 复制 .env.example 为 .env 并填入凭证"
    echo ""
    read -p "是否继续？(y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# 检查是否已编译
if [ ! -f voice-agent ]; then
    echo "📦 正在编译程序..."
    go build -o voice-agent main.go
    if [ $? -ne 0 ]; then
        echo "❌ 编译失败"
        exit 1
    fi
    echo "✓ 编译完成"
fi

echo ""
echo "🚀 启动语音对话系统..."
echo ""

# 运行程序
./voice-agent

