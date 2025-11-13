# 从旧版本迁移到全双工版本

## 概述

本指南帮助你从旧的同步语音对话系统迁移到新的全双工实时对话系统。

## 主要变更

### 1. API 变更

#### 旧版本（同步模式）

```go
// 创建代理
agent, _ := NewVoiceAgent(ctx)

// 录音（固定时长）
audioData, _ := agent.RecordAudio(5 * time.Second)

// 发送并等待完整响应
responseAudio, text, _ := agent.SendToNova(ctx, audioData)

// 播放响应
agent.PlayAudio(responseAudio)
```

#### 新版本（全双工模式）

```go
// 创建代理
agent, _ := NewVoiceAgent(ctx)

// 启动所有并发线程
go agent.StartContinuousRecording(ctx)
go agent.StartContinuousPlayback(ctx)
go agent.StreamAudioToNova(ctx, nil)

// 系统自动处理一切，无需手动调用
// 用户说话 → VAD 自动检测 → 发送 → 接收 → 播放
```

### 2. 模型变更

```go
// 旧版本
modelID: "us.amazon.nova-pro-v1:0"

// 新版本（推荐）
modelID: "us.amazon.nova-sonic-v1:0"  // 专为语音对话优化
```

### 3. 新增结构

#### AudioChunk

```go
type AudioChunk struct {
    Data      []byte
    Timestamp time.Time
}
```

#### ConversationContext

```go
type ConversationContext struct {
    SessionID string
    Messages  []ConversationMessage
    StartTime time.Time
}
```

## 兼容性说明

### 保留的旧方法

为了向后兼容，以下方法仍然可用：

- `RecordAudio(duration)` - 固定时长录音
- `SendToNova(ctx, audioData)` - 同步发送
- `PlayAudio(mulawData)` - 阻塞式播放

### 新增方法

- `StartContinuousRecording(ctx)` - 启动连续录音线程
- `StartContinuousPlayback(ctx)` - 启动连续播放线程
- `StreamAudioToNova(ctx, chan)` - 流式发送线程
- `AddUserMessage(audioData)` - 添加用户消息
- `AddAssistantMessage(audioData, text)` - 添加助手消息
- `GetConversationHistory()` - 获取对话历史
- `ResetSession()` - 重置会话

## 迁移步骤

### 步骤 1: 更新依赖

```bash
go get github.com/aws/aws-sdk-go-v2@latest
go get github.com/gen2brain/malgo@latest
go mod tidy
```

### 步骤 2: 选择运行模式

#### 选项 A: 完全迁移到全双工模式（推荐）

使用新的 `main.go`，享受所有新特性。

#### 选项 B: 保持旧模式（兼容）

如果你的代码依赖旧的 API，可以继续使用：

```go
func oldStyleConversation() {
    ctx := context.Background()
    agent, _ := NewVoiceAgent(ctx)
    defer agent.Close()

    for {
        // 旧的录音方式仍然有效
        audioData, _ := agent.RecordAudio(5 * time.Second)
        
        // 旧的发送方式仍然有效
        responseAudio, _, _ := agent.SendToNova(ctx, audioData)
        
        // 旧的播放方式仍然有效
        if len(responseAudio) > 0 {
            agent.PlayAudio(responseAudio)
        }
    }
}
```

#### 选项 C: 混合模式

你可以在全双工系统中使用部分旧 API：

```go
func hybridMode() {
    ctx := context.Background()
    agent, _ := NewVoiceAgent(ctx)
    
    // 使用新的 VAD 自动录音
    go agent.StartContinuousRecording(ctx)
    
    // 但手动处理音频
    for audioChunk := range agent.audioInputChan {
        responseAudio, _, _ := agent.SendToNova(ctx, audioChunk.Data)
        agent.PlayAudio(responseAudio)
    }
}
```

### 步骤 3: 测试

```bash
# 编译新版本
go build -o voice-agent main.go

# 运行测试
./voice-agent

# 或使用启动脚本
./start.sh
```

## 常见问题

### Q1: 我必须升级吗？

**A:** 不必须。旧的 API 完全兼容，你可以继续使用固定时长录音的方式。

### Q2: 全双工模式有什么好处？

**A:** 
- 自动 VAD 检测，无需手动控制录音时长
- 支持打断，更自然的对话体验
- 更低的延迟
- 多轮上下文自动管理

### Q3: 如何回退到旧版本？

**A:** 如果遇到问题，可以：
1. 使用 git 回退：`git checkout <old-commit>`
2. 或者在新版本中使用旧的 API（完全兼容）

### Q4: VAD 检测不准怎么办？

**A:** 调整 `vad.go` 中的参数：
- 降低 `EnergyThreshold` - 更敏感
- 提高 `EnergyThreshold` - 更不容易误触发
- 使用 `CalibrateThreshold()` 自动校准

### Q5: 性能会变差吗？

**A:** 不会。全双工模式使用 goroutine 并发处理，CPU 使用率可能略高，但响应延迟更低。

## 回滚方案

如果需要完全回退到旧版本：

```bash
# 1. 备份当前版本
cp main.go main_new.go
cp vad.go vad_new.go

# 2. 从 git 恢复旧版本
git checkout HEAD~1 main.go

# 3. 重新编译
go build -o voice-agent main.go
```

## 获取帮助

- 查看 [README.md](README.md) - 基础使用
- 查看 [FULLUPLEX_GUIDE.md](FULLUPLEX_GUIDE.md) - 全双工详细说明
- 提交 GitHub Issue - 报告问题

