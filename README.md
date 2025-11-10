# macOS 语音录音工具

这是一个使用 Go 语言实现的 macOS 麦克风录音工具，支持 mulaw 编码格式。

## 功能特性

- ✅ 在 macOS 上唤起并使用麦克风进行录音
- ✅ 音频编码：audio/x-mulaw (G.711 μ-law)
- ✅ 采样率：8000 Hz
- ✅ 声道：单声道 (Mono)
- ✅ 自动保存为 WAV 文件格式

## 系统要求

- macOS 操作系统
- Go 1.18 或更高版本

## 安装依赖

```bash
go mod download
```

## 编译

```bash
go build -o voice-recorder main.go
```

## 运行

```bash
go run main.go
```

或者运行编译后的可执行文件：

```bash
./voice-recorder
```

## 使用说明

1. 运行程序后，会自动开始录音
2. 程序会显示录音状态信息
3. 按 `Ctrl+C` 停止录音
4. 录音文件会自动保存为 `recording_YYYYMMDD_HHMMSS.wav` 格式

## 输出文件格式

- 文件格式：WAV
- 音频编码：mulaw (G.711 μ-law)
- 采样率：8000 Hz
- 比特率：8 bits per sample
- 声道数：1 (单声道)

## 注意事项

1. 首次运行时，macOS 可能会要求授权麦克风访问权限，请允许该请求
2. 确保麦克风设备已正确连接并设置为系统默认录音设备
3. 录音文件会保存在程序运行的当前目录下

## 技术实现

- 使用 `github.com/gen2brain/malgo` 库进行跨平台音频采集
- 实现了 ITU-T G.711 μ-law 编码算法
- 自定义 WAV 文件格式写入，支持 mulaw 编码

## 许可证

MIT License

