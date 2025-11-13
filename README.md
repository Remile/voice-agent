# AWS Bedrock Nova 全双工语音对话系统

一个基于 AWS Bedrock Nova 模型的**全双工实时语音对话系统**，支持 VAD 自动检测、实时流式对话和打断功能。

## 🎯 功能特性

### 核心功能
- ✅ **VAD 自动检测**：智能检测语音活动，无需手动控制录音
- ✅ **全双工对话**：4 个并发线程实现真正的实时对话
- ✅ **支持打断**：可以在 AI 说话时打断它，自然流畅的交互
- ✅ **多轮上下文**：自动维护对话历史和会话管理
- ✅ **AWS Bedrock 集成**：调用 Nova Sonic 模型（专为语音对话优化）
- ✅ **实时流式**：边录音边处理边播放，极低延迟
- ✅ **多模态支持**：支持音频输入和音频输出

## 🔧 系统要求

- macOS 操作系统
- Go 1.18 或更高版本
- AWS 账号并配置好凭证
- 麦克风和扬声器设备

## 📦 安装依赖

### 1. 克隆项目

```bash
git clone <your-repo-url>
cd voice-agent
```

### 2. 安装 Go 依赖

```bash
go mod download
```

或者

```bash
GOPROXY=https://goproxy.cn,direct go mod download
```

## ⚙️ AWS 配置

### 1. 配置 AWS 凭证

确保你已经配置了 AWS 凭证。可以通过以下任一方式：

**方式 1: 使用 AWS CLI 配置**

```bash
aws configure
```

输入你的：
- AWS Access Key ID
- AWS Secret Access Key
- Default region name（建议：us-east-1 或 us-west-2）
- Default output format（可选：json）

**方式 2: 设置环境变量**

```bash
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_REGION=us-east-1
```

**方式 3: 使用 AWS SSO**

```bash
aws sso login --profile your-profile
export AWS_PROFILE=your-profile
```

### 2. 确保 Bedrock 访问权限

确保你的 AWS 账号有权限访问 Bedrock 服务，并且已在对应区域启用了 Nova Pro 模型。

#### 启用 Nova Pro 模型：

1. 登录 AWS Console
2. 进入 Amazon Bedrock 服务
3. 选择 "Model access" 
4. 找到 "Amazon Nova Pro" 并请求访问权限
5. 等待审批通过（通常几分钟内完成）

#### 支持 Nova 的 AWS 区域：

- us-east-1 (N. Virginia)
- us-west-2 (Oregon)

## 🚀 快速开始

### 编译程序

```bash
go build -o voice-agent main.go
```

### 运行程序

```bash
./voice-agent
```

或者直接运行：

```bash
go run main.go
```

### 使用方式

**全双工模式（推荐）：**

1. 程序启动后会自动初始化所有组件
2. 系统持续监听，**直接开始说话即可**
3. VAD 会自动检测你的语音并发送
4. AI 回复会实时播放
5. 可以在 AI 说话时打断它
6. 按 `Ctrl+C` 退出

**传统模式（兼容旧版本）：**

如果需要使用固定时长录音的旧模式，可以调用 `RecordAudio()` 和 `SendToNova()` 方法。

### 输出文件

程序会自动在 `output` 目录下保存：
- `input_YYYYMMDD_HHMMSS.wav`：用户输入的录音（可选）
- `response_YYYYMMDD_HHMMSS.wav`：Nova 的语音回复（可选）

## 📝 技术细节

### 音频格式

**录音参数：**
- 采样率：8000 Hz
- 编码：mulaw (G.711 μ-law)
- 声道：单声道 (Mono)
- 比特率：8 bits per sample

**播放参数：**
- 格式：16-bit PCM
- 采样率：8000 Hz
- 声道：单声道

### 使用的技术栈

- **音频处理**：`github.com/gen2brain/malgo` - 跨平台音频库
- **AWS SDK**：`github.com/aws/aws-sdk-go-v2` - AWS Go SDK v2
- **AI 模型**：Amazon Nova Sonic (us.amazon.nova-sonic-v1:0) - 专为语音对话优化

### 架构设计

**4 个并发线程：**
1. **录音线程**：持续监听麦克风 + VAD 检测
2. **发送线程**：流式发送音频到 Nova API
3. **接收线程**：接收 API 响应（当前集成在发送线程中）
4. **播放线程**：实时流式播放 + 支持打断

**通道通信：**
- `audioInputChan`：录音 → 发送
- `audioOutputChan`：接收 → 播放
- `interruptChan`：打断信号

### Nova Sonic 模型

Amazon Nova Sonic 是 AWS Bedrock 专为语音对话优化的模型，支持：
- 低延迟语音处理
- 实时流式对话
- 多轮上下文记忆
- 自然语言理解

## 📚 高级功能

详细的全双工功能说明、VAD 参数调优、性能优化等，请参阅：
- **[FULLUPLEX_GUIDE.md](FULLUPLEX_GUIDE.md)** - 完整的全双工特性文档

## 🔍 故障排查

### 1. 麦克风权限问题

如果遇到麦克风访问被拒绝：
- macOS：系统偏好设置 → 安全性与隐私 → 隐私 → 麦克风 → 允许终端访问

### 2. AWS 认证失败

```
Error: 加载AWS配置失败
```

解决方案：
- 检查 AWS 凭证是否正确配置
- 运行 `aws sts get-caller-identity` 验证凭证
- 确保使用正确的 AWS Profile

### 3. Bedrock 访问被拒

```
Error: AccessDeniedException
```

解决方案：
- 确保在 AWS Console 中启用了 Nova Pro 模型访问权限
- 检查 IAM 角色/用户是否有 `bedrock:InvokeModel` 权限
- 确认使用的 region 支持 Nova 模型

### 4. 模型未找到

```
Error: Model not found
```

解决方案：
- 确保使用正确的模型 ID：`us.amazon.nova-pro-v1:0`
- 切换到支持 Nova 的 region（us-east-1 或 us-west-2）

### 5. 音频播放无声音

- 检查系统音量设置
- 确保扬声器设备正常工作
- 查看程序输出的错误信息

## 🛠️ 自定义配置

### 调整 VAD 参数

在 `vad.go` 的 `DefaultVADConfig()` 中修改：

```go
return VADConfig{
    EnergyThreshold:   500.0,  // 能量阈值（越高越不敏感）
    SpeechStartFrames: 3,      // 语音开始确认帧数
    SpeechEndFrames:   8,      // 语音结束确认帧数
}
```

**快速调优建议：**
- 环境嘈杂 → 提高 `EnergyThreshold` 到 800
- 反应太慢 → 降低 `EnergyThreshold` 到 300
- 容易被切断 → 增加 `SpeechEndFrames` 到 12

### 更换模型

在 `main.go` 的 `NewVoiceAgent` 函数中修改：

```go
modelID: "us.amazon.nova-sonic-v1:0",  // Nova Sonic (推荐)
```

可选的模型：
- `us.amazon.nova-sonic-v1:0` - Nova Sonic（语音对话专用，推荐）
- `us.amazon.nova-pro-v1:0` - Nova Pro（多模态，通用）
- `us.amazon.nova-lite-v1:0` - Nova Lite（更快，能力较弱）

### 调整 AI 参数

在 `StreamAudioToNova` 函数中修改 `inferenceConfig`：

```go
"inferenceConfig": map[string]interface{}{
    "maxTokens":   2048,
    "temperature": 0.7,  // 0.0 - 1.0，越高越有创造性
},
```

### 音频缓冲大小

在 `NewVoiceAgent` 中调整通道缓冲：

```go
audioInputChan:  make(chan AudioChunk, 10),   // 输入缓冲
audioOutputChan: make(chan AudioChunk, 100),  // 输出缓冲（可增大）
```

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📧 联系方式

如有问题，请提交 GitHub Issue。

---

**注意**：使用此程序会产生 AWS Bedrock 使用费用，请查看 AWS 定价页面了解详情。
