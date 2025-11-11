# AWS Bedrock Nova 语音对话系统

一个基于 AWS Bedrock Nova 模型的实时语音对话系统，支持语音输入并获取 AI 语音回复。

## 🎯 功能特性

- ✅ **实时语音录制**：使用麦克风录制用户语音（8000 Hz, mulaw 编码）
- ✅ **AWS Bedrock 集成**：直接调用 Nova Pro 模型进行对话
- ✅ **多模态支持**：支持音频输入和音频输出
- ✅ **自动播放**：将 AI 的语音回复自动播放到扬声器
- ✅ **循环对话**：支持连续多轮对话
- ✅ **录音保存**：可选择保存对话录音文件

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

## 🚀 使用方法

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

### 使用流程

1. 程序启动后会自动初始化音频设备和 AWS Bedrock 客户端
2. 每轮对话流程：
   - 📢 提示 "请说话..."
   - 🎤 录音 5 秒（可在代码中修改时长）
   - 📤 自动发送音频到 Nova 模型
   - 📥 接收 Nova 的语音回复
   - 🔊 自动播放 AI 的回复
3. 按 `Ctrl+C` 退出程序

### 输出文件

程序会自动在 `output` 目录下保存：
- `input_YYYYMMDD_HHMMSS.wav`：用户输入的录音
- `response_YYYYMMDD_HHMMSS.wav`：Nova 的语音回复

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
- **AI 模型**：Amazon Nova Pro (us.amazon.nova-pro-v1:0)

### Nova 模型

Amazon Nova Pro 是 AWS Bedrock 提供的多模态 AI 模型，支持：
- 文本理解和生成
- 音频输入处理
- 音频输出生成
- 多轮对话

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

### 修改录音时长

在 `main.go` 中找到：

```go
audioData, err := agent.RecordAudio(5 * time.Second)
```

将 `5 * time.Second` 改为你想要的时长，例如 `10 * time.Second`。

### 更换模型

在 `main.go` 的 `NewVoiceAgent` 函数中修改：

```go
modelID: "us.amazon.nova-pro-v1:0",
```

可选的模型：
- `us.amazon.nova-pro-v1:0` - Nova Pro（推荐）
- `us.amazon.nova-lite-v1:0` - Nova Lite（更快，但能力较弱）

### 调整 AI 参数

在 `SendToNova` 函数中修改 `inferenceConfig`：

```go
"inferenceConfig": map[string]interface{}{
    "max_new_tokens": 2048,
    "temperature":    0.7,  // 0.0 - 1.0，越高越有创造性
},
```

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📧 联系方式

如有问题，请提交 GitHub Issue。

---

**注意**：使用此程序会产生 AWS Bedrock 使用费用，请查看 AWS 定价页面了解详情。
