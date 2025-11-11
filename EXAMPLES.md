# 使用示例和场景

本文档提供详细的使用示例和常见场景。

## 📝 基本使用示例

### 示例 1：简单对话

```bash
$ ./voice-agent

=== AWS Bedrock Nova 语音对话系统 ===
采样率: 8000 Hz | 编码: mulaw | 声道: 单声道

✓ 语音代理已初始化
按 Ctrl+C 退出程序

━━━━━━━━ 对话 #1 ━━━━━━━━

请说话...
🎤 正在录音 (5s)...
```

**你说**："你好，请介绍一下自己"

```
✓ 录音完成，共 5.00 秒
💾 录音已保存: output/input_20251111_143022.wav
📤 正在发送音频到 Nova 模型...
✓ 收到 Nova 响应
💬 Nova 回复（文本）: 你好！我是 Amazon Nova，一个由 AWS 开发的大型语言模型。我可以帮助你...
💾 响应已保存: output/response_20251111_143022.wav
🔊 正在播放回复...
✓ 播放完成

准备下一轮对话...
```

### 示例 2：连续多轮对话

```bash
━━━━━━━━ 对话 #1 ━━━━━━━━
你: "今天天气怎么样？"
Nova: "抱歉，我无法获取实时天气信息..."

━━━━━━━━ 对话 #2 ━━━━━━━━
你: "那你能做什么？"
Nova: "我可以回答各种问题，帮助你理解复杂概念..."

━━━━━━━━ 对话 #3 ━━━━━━━━
你: "解释一下机器学习"
Nova: "机器学习是人工智能的一个分支..."
```

## 🎯 实际应用场景

### 场景 1：语言学习助手

**用途**：练习英语口语

```bash
━━━━━━━━ 对话 #1 ━━━━━━━━
你: "Hello, can you help me practice English?"
Nova: "Of course! I'd be happy to help you practice English..."

━━━━━━━━ 对话 #2 ━━━━━━━━
你: "How do I introduce myself in a job interview?"
Nova: "When introducing yourself in a job interview..."
```

**优势**：
- 实时语音反馈
- 自然对话练习
- 保存录音回顾

### 场景 2：技术问答助手

**用途**：快速查询技术问题

```bash
你: "什么是 Docker？"
Nova: "Docker 是一个开源的容器化平台..."

你: "如何使用 Docker 部署应用？"
Nova: "部署应用到 Docker 主要有以下步骤..."
```

### 场景 3：创意头脑风暴

**用途**：产品创意讨论

```bash
你: "我想做一个健身 App，有什么建议？"
Nova: "一个成功的健身 App 应该包含以下功能..."

你: "目标用户应该是哪些人？"
Nova: "根据健身 App 的特点，主要目标用户包括..."
```

### 场景 4：学习辅导

**用途**：讲解复杂概念

```bash
你: "能解释一下量子计算吗？"
Nova: "量子计算利用量子力学原理进行信息处理..."

你: "它和传统计算机有什么区别？"
Nova: "传统计算机使用比特（0或1），而量子计算机使用量子比特..."
```

## 🔧 高级配置示例

### 配置 1：使用不同的 Nova 模型

编辑 `main.go`，修改模型 ID：

```go
// 使用 Nova Lite（更快，成本更低）
modelID: "us.amazon.nova-lite-v1:0",

// 使用 Nova Pro（默认，平衡性能和成本）
modelID: "us.amazon.nova-pro-v1:0",

// 使用 Nova Premier（最强大，成本最高）
modelID: "us.amazon.nova-premier-v1:0",
```

### 配置 2：调整录音时长

```go
// 短对话（3 秒）
audioData, err := agent.RecordAudio(3 * time.Second)

// 默认（5 秒）
audioData, err := agent.RecordAudio(5 * time.Second)

// 长对话（10 秒）
audioData, err := agent.RecordAudio(10 * time.Second)
```

### 配置 3：修改 AI 参数

```go
"inferenceConfig": map[string]interface{}{
    "maxTokens":   2048,      // 最大输出长度
    "temperature": 0.7,       // 创造性（0.0-1.0）
    "topP":        0.9,       // 多样性控制
},
```

**temperature 效果**：
- `0.0-0.3`：更加确定和一致，适合技术问答
- `0.4-0.7`：平衡，适合日常对话
- `0.8-1.0`：更有创造性，适合头脑风暴

### 配置 4：使用环境变量

创建 `.env` 文件：

```bash
# AWS 配置
AWS_ACCESS_KEY_ID=AKIA...
AWS_SECRET_ACCESS_KEY=wJalrXUtn...
AWS_REGION=us-east-1

# 可选：自定义配置
RECORDING_DURATION=5
MODEL_ID=us.amazon.nova-pro-v1:0
SAVE_RECORDINGS=true
```

## 📊 使用统计和分析

### 查看录音文件

```bash
# 列出所有录音
ls -lh output/

# 查看最近的录音
ls -lt output/ | head -10

# 统计录音数量
echo "总对话次数: $(ls output/input_*.wav 2>/dev/null | wc -l)"
```

### 播放保存的录音

```bash
# macOS
afplay output/input_20251111_143022.wav

# Linux
aplay output/input_20251111_143022.wav

# 使用 ffplay（跨平台）
ffplay output/input_20251111_143022.wav
```

### 分析音频文件

```bash
# 查看 WAV 文件信息
file output/input_20251111_143022.wav

# 使用 ffprobe 查看详细信息
ffprobe output/input_20251111_143022.wav
```

## 🎨 自定义开发示例

### 示例 1：添加对话历史记录

```go
// 在 VoiceAgent 结构中添加
type VoiceAgent struct {
    // ... 现有字段
    conversationHistory []Message
}

type Message struct {
    Role      string    // "user" or "assistant"
    Content   string
    Timestamp time.Time
}

// 修改 SendToNova 函数以包含历史
func (va *VoiceAgent) SendToNova(ctx context.Context, audioData []byte) ([]byte, string, error) {
    // 构建包含历史的请求
    messages := va.buildMessagesWithHistory(audioData)
    // ... 其余代码
}
```

### 示例 2：添加语音激活检测（VAD）

```go
// 添加静音检测函数
func detectSilence(audioData []byte, threshold float64) bool {
    var sum float64
    for _, sample := range audioData {
        decoded := float64(mulawToLinear(sample))
        sum += math.Abs(decoded)
    }
    average := sum / float64(len(audioData))
    return average < threshold
}

// 在录音回调中使用
if detectSilence(currentBuffer, 500.0) {
    // 检测到静音，停止录音
    stopRecording <- true
}
```

### 示例 3：添加文本转语音备用方案

```go
import "github.com/aws/aws-sdk-go-v2/service/polly"

func (va *VoiceAgent) textToSpeech(text string) ([]byte, error) {
    // 使用 AWS Polly 将文本转为语音
    pollyClient := polly.NewFromConfig(va.awsConfig)
    
    output, err := pollyClient.SynthesizeSpeech(ctx, &polly.SynthesizeSpeechInput{
        Text:         aws.String(text),
        OutputFormat: types.OutputFormatPcm,
        VoiceId:      types.VoiceIdJoanna,
    })
    
    // ... 处理输出
}
```

### 示例 4：添加进度条显示

```go
import "github.com/schollz/progressbar/v3"

func (va *VoiceAgent) RecordAudioWithProgress(duration time.Duration) ([]byte, error) {
    bar := progressbar.Default(int64(duration.Seconds()))
    
    // 在录音循环中
    for i := 0; i < int(duration.Seconds()); i++ {
        time.Sleep(1 * time.Second)
        bar.Add(1)
    }
    
    return recordedData, nil
}
```

## 🐛 调试和问题排查

### 调试模式

添加详细日志：

```go
import "log"

// 在 main 函数开始处
log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
log.Println("程序启动")

// 在关键位置添加日志
log.Printf("录音数据大小: %d 字节", len(audioData))
log.Printf("API 响应: %+v", response)
```

### 测试音频设备

```bash
# 测试麦克风
go run main.go

# 测试扬声器
afplay output/response_*.wav
```

### 验证 AWS 连接

```bash
# 测试 AWS 凭证
aws sts get-caller-identity

# 测试 Bedrock 访问
aws bedrock list-foundation-models --region us-east-1

# 测试 API 调用
aws bedrock-runtime invoke-model \
    --region us-east-1 \
    --model-id us.amazon.nova-pro-v1:0 \
    --body '{"messages":[{"role":"user","content":[{"text":"Hi"}]}]}' \
    --cli-binary-format raw-in-base64-out \
    /tmp/test.json
```

## 📈 性能优化建议

### 1. 减少延迟

```go
// 使用流式 API（需要 AWS SDK 支持）
// 边录音边发送，减少等待时间

// 预加载模型（如果支持）
// 缓存常见问答
```

### 2. 降低成本

```go
// 使用 Nova Lite 模型
modelID: "us.amazon.nova-lite-v1:0"

// 减少 maxTokens
"maxTokens": 1024  // 从 2048 减少到 1024

// 压缩音频（如果支持）
```

### 3. 提升用户体验

```go
// 添加加载动画
fmt.Print("处理中")
for i := 0; i < 3; i++ {
    time.Sleep(500 * time.Millisecond)
    fmt.Print(".")
}
fmt.Println()

// 添加进度提示
fmt.Println("🎤 录音中...")
fmt.Println("📤 上传中...")
fmt.Println("🤔 思考中...")
fmt.Println("🔊 回复中...")
```

## 🔒 安全使用建议

### 1. 保护敏感信息

```bash
# 不要录制包含密码、密钥等敏感信息的对话
# 定期清理录音文件
rm output/*.wav

# 加密存储敏感录音
openssl enc -aes-256-cbc -in recording.wav -out recording.wav.enc
```

### 2. 访问控制

```bash
# 限制录音文件权限
chmod 600 output/*.wav

# 使用专用 IAM 用户
aws iam create-user --user-name voice-agent-prod
```

### 3. 审计日志

```go
// 记录所有 API 调用
log.Printf("API 调用: modelID=%s, timestamp=%s", modelID, time.Now())

// 启用 CloudTrail
// 在 AWS Console 中配置
```

## 📚 更多资源

### 官方文档
- [AWS Bedrock Developer Guide](https://docs.aws.amazon.com/bedrock/)
- [Nova Model Documentation](https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-nova.html)
- [Go SDK Documentation](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2)

### 社区资源
- [AWS re:Post - Bedrock](https://repost.aws/tags/TA4ckVRBiHQ2yjspray9exRDg/amazon-bedrock)
- [GitHub Issues](https://github.com/aws/aws-sdk-go-v2/issues)
- [Stack Overflow - AWS Bedrock](https://stackoverflow.com/questions/tagged/amazon-bedrock)

### 相关项目
- [Bedrock Examples](https://github.com/aws-samples/amazon-bedrock-samples)
- [Voice AI Projects](https://github.com/topics/voice-ai)

---

**提示**：如果你有其他使用场景或示例，欢迎贡献到此文档！

