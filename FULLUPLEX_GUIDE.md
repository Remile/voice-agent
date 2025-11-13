# AWS Bedrock Nova 全双工语音对话系统

## 🎯 新功能概述

本项目已升级为**全双工实时语音对话系统**，支持以下高级特性：

### ✨ 核心特性

1. **VAD 语音活动检测**
   - 自动检测用户何时开始说话
   - 自动检测用户何时停止说话
   - 无需手动控制录音时长

2. **全双工实时对话**
   - 4 个并发线程同时运行
   - 边录音边处理边播放
   - 极低延迟的对话体验

3. **支持打断**
   - 可以在 AI 说话时打断它
   - VAD 检测到新的语音输入会立即停止播放
   - 自然流畅的交互体验

4. **多轮对话上下文**
   - 自动维护对话历史
   - 支持上下文相关的对话
   - 会话管理和统计

## 🏗️ 架构设计

### 线程架构

```
┌─────────────────────────────────────────────────────┐
│                   主控制线程                          │
│            (信号处理、错误管理、状态监控)               │
└─────────────────────────────────────────────────────┘
           │                │                │
           ▼                ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  录音线程      │  │  发送线程      │  │  播放线程      │
│  (VAD 检测)   │  │  (AWS API)    │  │  (流式播放)    │
└──────────────┘  └──────────────┘  └──────────────┘
      │                  │                  ▲
      │                  │                  │
      ▼                  ▼                  │
  audioInputChan    Bedrock API    audioOutputChan
      │                  │                  │
      └──────────────────┴──────────────────┘
                  interruptChan
                  (打断信号)
```

### 数据流

1. **录音 → 发送**
   - 麦克风持续监听
   - VAD 检测到语音开始 → 累积音频数据
   - VAD 检测到语音结束 → 通过 `audioInputChan` 发送

2. **发送 → AWS**
   - 从 `audioInputChan` 接收音频
   - 编码为 mulaw/base64
   - 调用 Bedrock API
   - 解析响应

3. **AWS → 播放**
   - 接收 API 响应
   - 解码音频数据
   - 通过 `audioOutputChan` 发送给播放线程
   - 实时播放

4. **打断机制**
   - 录音线程检测到新语音
   - 如果正在播放，发送中断信号到 `interruptChan`
   - 播放线程收到信号后立即清空缓冲并停止

## 🔧 配置和调优

### VAD 参数调整

在 `vad.go` 的 `DefaultVADConfig()` 中可以调整：

```go
return VADConfig{
    EnergyThreshold:   500.0,  // 能量阈值：越高越不敏感
    SpeechStartFrames: 3,      // 语音开始帧数：越大越不容易误触发
    SpeechEndFrames:   8,      // 语音结束帧数：越大越不容易过早切断
    SampleRate:        8000,
    FrameSize:         800,    // 帧大小（100ms @ 8000Hz）
}
```

#### 调优建议

**环境嘈杂（容易误触发）：**
```go
EnergyThreshold:   800.0    // 提高阈值
SpeechStartFrames: 5        // 增加确认帧数
```

**环境安静（反应太慢）：**
```go
EnergyThreshold:   300.0    // 降低阈值
SpeechStartFrames: 2        // 减少确认帧数
```

**说话容易被切断：**
```go
SpeechEndFrames:   12       // 增加结束延迟
```

### 动态校准阈值

程序启动时，可以自动校准噪音阈值：

```go
// 在 main 函数中添加
fmt.Println("正在校准环境噪音，请保持安静...")
noiseData, _ := agent.RecordAudio(2 * time.Second)
agent.vad.CalibrateThreshold(noiseData)
fmt.Println("✓ 校准完成")
```

## 📊 性能优化

### 音频缓冲

播放线程使用缓冲队列来避免卡顿：

```go
// 在 NewVoiceAgent 中调整缓冲大小
audioOutputChan: make(chan AudioChunk, 100),  // 增大缓冲
```

### 并发安全

- 使用 `sync.Mutex` 保护播放缓冲区
- 使用通道进行线程间通信
- 避免数据竞争

## 🧪 测试建议

### 1. 基础功能测试

```bash
# 编译
go build -o voice-agent main.go

# 运行
./voice-agent
```

**测试场景：**
- 说一句话，观察是否自动检测开始和结束
- 在 AI 播放时打断它，说新的内容
- 进行多轮对话，测试上下文记忆

### 2. VAD 参数测试

创建测试脚本 `test_vad.sh`：

```bash
#!/bin/bash
echo "测试 VAD 参数..."

# 测试 1: 默认参数
echo "测试 1: 默认参数"
./voice-agent &
PID=$!
sleep 30
kill $PID

# 测试 2: 高灵敏度
echo "测试 2: 高灵敏度（修改代码中的阈值为 300）"
# 修改代码，重新编译测试
```

### 3. 压力测试

```go
// 添加到 main 函数进行压力测试
func stressTest(agent *VoiceAgent) {
    for i := 0; i < 100; i++ {
        audioData := generateTestAudio(3 * time.Second)
        agent.audioInputChan <- AudioChunk{Data: audioData}
        time.Sleep(100 * time.Millisecond)
    }
}
```

## 🐛 常见问题

### 1. VAD 不敏感（说话没反应）

**症状：** 说话时没有"检测到语音"的提示

**解决方案：**
- 降低 `EnergyThreshold` (例如从 500 降到 300)
- 减少 `SpeechStartFrames` (例如从 3 降到 2)
- 检查麦克风权限和音量

### 2. VAD 太敏感（频繁误触发）

**症状：** 没说话也提示"检测到语音"

**解决方案：**
- 提高 `EnergyThreshold` (例如从 500 升到 800)
- 增加 `SpeechStartFrames` (例如从 3 升到 5)
- 使用 `CalibrateThreshold()` 自动校准

### 3. 说话容易被切断

**症状：** 说话时经常话还没说完就被发送了

**解决方案：**
- 增加 `SpeechEndFrames` (例如从 8 升到 12)
- 考虑说话时的停顿习惯

### 4. 播放有延迟或卡顿

**解决方案：**
- 增大 `audioOutputChan` 缓冲大小
- 检查网络连接速度
- 优化音频帧大小

### 5. 打断不及时

**解决方案：**
- 检查 `interruptChan` 是否被正确监听
- 减少播放缓冲区大小
- 优化 VAD 响应速度

## 🔍 调试技巧

### 启用详细日志

在关键位置添加日志：

```go
// 在 VAD 检测中
fmt.Printf("[VAD] Energy: %.2f, Threshold: %.2f, State: %v\n", 
    energy, vad.config.EnergyThreshold, vadState)

// 在发送线程中
fmt.Printf("[Send] Queue size: %d, Buffer: %d bytes\n",
    len(va.audioInputChan), len(audioChunk.Data))

// 在播放线程中
fmt.Printf("[Play] Buffer size: %d bytes, Playing: %v\n",
    len(playbackBuffer), va.isPlaying)
```

### 性能监控

添加性能统计：

```go
type Stats struct {
    VADDetections   int
    SentMessages    int
    ReceivedChunks  int
    Interruptions   int
    TotalDuration   time.Duration
}

// 定期输出统计
ticker := time.NewTicker(10 * time.Second)
go func() {
    for range ticker.C {
        fmt.Printf("Stats: VAD=%d, Sent=%d, Recv=%d, Int=%d\n",
            stats.VADDetections, stats.SentMessages, 
            stats.ReceivedChunks, stats.Interruptions)
    }
}()
```

## 📝 代码结构

```
voice-agent/
├── main.go              # 主程序（包含所有核心逻辑）
├── vad.go               # VAD 语音活动检测模块
├── go.mod               # Go 模块定义
├── go.sum               # 依赖锁定
├── output/              # 录音输出目录
├── README.md            # 基础使用说明
└── FULLUPLEX_GUIDE.md   # 本文档（全双工特性说明）
```

## 🚀 未来改进

### 短期改进

1. **真正的 ConverseStream API**
   - 当前使用 `InvokeModel` (同步)
   - 等待 AWS SDK 支持真正的流式 API
   - 将获得更低的延迟

2. **更智能的 VAD**
   - 集成机器学习模型
   - 支持更复杂的语音场景
   - 自适应阈值调整

3. **音频质量优化**
   - 支持更高采样率 (16kHz)
   - 降噪处理
   - 回声消除

### 长期改进

1. **多模态支持**
   - 同时支持文本输入
   - 视频输入支持
   - 屏幕共享

2. **分布式架构**
   - 支持多用户
   - 负载均衡
   - 会话持久化

3. **高级功能**
   - 情感识别
   - 说话人识别
   - 实时翻译

## 📞 技术支持

如遇到问题：

1. 检查本文档的"常见问题"部分
2. 启用详细日志进行调试
3. 查看 AWS Bedrock 服务状态
4. 提交 GitHub Issue

## 📄 许可证

MIT License - 详见 README.md

