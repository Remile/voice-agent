# 快速开始指南

## 第一步：安装依赖

```bash
go mod download
```

如果网络较慢，可以使用国内镜像：

```bash
GOPROXY=https://goproxy.cn,direct go mod download
```

## 第二步：配置 AWS 凭证

### 方法 1：使用 AWS CLI（推荐）

```bash
aws configure
```

输入信息：
```
AWS Access Key ID [None]: your_access_key
AWS Secret Access Key [None]: your_secret_key
Default region name [None]: us-east-1
Default output format [None]: json
```

### 方法 2：设置环境变量

```bash
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_REGION=us-east-1
```

### 方法 3：使用 AWS SSO

```bash
aws sso login --profile your-profile
export AWS_PROFILE=your-profile
```

## 第三步：启用 Nova 模型

1. 登录 [AWS Console](https://console.aws.amazon.com/)
2. 切换到 us-east-1 或 us-west-2 区域
3. 进入 **Amazon Bedrock** 服务
4. 点击左侧菜单 **Model access**
5. 找到 **Amazon Nova Pro** 并点击 **Request access**
6. 勾选同意条款，点击 **Submit**
7. 等待几分钟直到状态变为 **Access granted**

## 第四步：编译和运行

### 编译

```bash
go build -o voice-agent main.go
```

### 运行

```bash
./voice-agent
```

或使用启动脚本：

```bash
./run.sh
```

或直接运行：

```bash
go run main.go
```

## 使用示例

运行后你会看到：

```
=== AWS Bedrock Nova 语音对话系统 ===
采样率: 8000 Hz | 编码: mulaw | 声道: 单声道

✓ 语音代理已初始化
按 Ctrl+C 退出程序

━━━━━━━━ 对话 #1 ━━━━━━━━

请说话...
🎤 正在录音 (5s)...
✓ 录音完成，共 5.00 秒
💾 录音已保存: output/input_20251111_123456.wav
📤 正在发送音频到 Nova 模型...
✓ 收到 Nova 响应
💬 Nova 回复（文本）: [AI的文本回复]
💾 响应已保存: output/response_20251111_123456.wav
🔊 正在播放回复...
✓ 播放完成

准备下一轮对话...
```

## 测试对话示例

### 对话 1：简单问候

**你说**："你好，请介绍一下自己"

**Nova 回复**："你好！我是 Amazon Nova，一个由 AWS 开发的 AI 助手..."

### 对话 2：询问信息

**你说**："今天天气怎么样？"

**Nova 回复**："抱歉，我无法获取实时天气信息..."

### 对话 3：技术问题

**你说**："什么是机器学习？"

**Nova 回复**："机器学习是人工智能的一个分支..."

## 常见问题

### Q: 录音时听不到声音？

A: 这是正常的，程序会在后台录音。你只需要对着麦克风说话即可。

### Q: 播放时听不到 AI 的回复？

A: 检查系统音量，确保扬声器正常工作。

### Q: 提示 AWS 认证失败？

A: 运行 `aws sts get-caller-identity` 检查凭证是否正确。

### Q: 提示 Model not found？

A: 确保你在 AWS Console 中启用了 Nova Pro 模型，且使用 us-east-1 或 us-west-2 区域。

### Q: 想修改录音时长？

A: 编辑 `main.go`，找到 `agent.RecordAudio(5 * time.Second)`，修改数字。

## 查看保存的录音

所有录音文件保存在 `output` 目录：

```bash
ls -lh output/
```

播放录音：

```bash
# macOS
afplay output/input_20251111_123456.wav

# Linux
aplay output/input_20251111_123456.wav
```

## 下一步

- 阅读完整的 [README.md](README.md) 了解更多功能
- 修改代码以自定义对话流程
- 尝试不同的 Nova 模型参数

祝你使用愉快！🎉

