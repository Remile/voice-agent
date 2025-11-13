package main

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/gen2brain/malgo"
)

// mulaw 编码表（符合 ITU-T G.711 标准）
var (
	mulawCompressTable = [256]byte{
		0, 0, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3,
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
		5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5,
		5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	}
	mulawBias = 0x84

	// mulaw 解码表
	mulawDecompressTable = [256]int16{
		-32124, -31100, -30076, -29052, -28028, -27004, -25980, -24956,
		-23932, -22908, -21884, -20860, -19836, -18812, -17788, -16764,
		-15996, -15484, -14972, -14460, -13948, -13436, -12924, -12412,
		-11900, -11388, -10876, -10364, -9852, -9340, -8828, -8316,
		-7932, -7676, -7420, -7164, -6908, -6652, -6396, -6140,
		-5884, -5628, -5372, -5116, -4860, -4604, -4348, -4092,
		-3900, -3772, -3644, -3516, -3388, -3260, -3132, -3004,
		-2876, -2748, -2620, -2492, -2364, -2236, -2108, -1980,
		-1884, -1820, -1756, -1692, -1628, -1564, -1500, -1436,
		-1372, -1308, -1244, -1180, -1116, -1052, -988, -924,
		-876, -844, -812, -780, -748, -716, -684, -652,
		-620, -588, -556, -524, -492, -460, -428, -396,
		-372, -356, -340, -324, -308, -292, -276, -260,
		-244, -228, -212, -196, -180, -164, -148, -132,
		-120, -112, -104, -96, -88, -80, -72, -64,
		-56, -48, -40, -32, -24, -16, -8, 0,
		32124, 31100, 30076, 29052, 28028, 27004, 25980, 24956,
		23932, 22908, 21884, 20860, 19836, 18812, 17788, 16764,
		15996, 15484, 14972, 14460, 13948, 13436, 12924, 12412,
		11900, 11388, 10876, 10364, 9852, 9340, 8828, 8316,
		7932, 7676, 7420, 7164, 6908, 6652, 6396, 6140,
		5884, 5628, 5372, 5116, 4860, 4604, 4348, 4092,
		3900, 3772, 3644, 3516, 3388, 3260, 3132, 3004,
		2876, 2748, 2620, 2492, 2364, 2236, 2108, 1980,
		1884, 1820, 1756, 1692, 1628, 1564, 1500, 1436,
		1372, 1308, 1244, 1180, 1116, 1052, 988, 924,
		876, 844, 812, 780, 748, 716, 684, 652,
		620, 588, 556, 524, 492, 460, 428, 396,
		372, 356, 340, 324, 308, 292, 276, 260,
		244, 228, 212, 196, 180, 164, 148, 132,
		120, 112, 104, 96, 88, 80, 72, 64,
		56, 48, 40, 32, 24, 16, 8, 0,
	}
)

// linearToMulaw 将 16-bit PCM 转换为 mulaw
func linearToMulaw(sample int16) byte {
	const clip = 32635

	// 获取符号位
	sign := byte(0x80)
	if sample < 0 {
		sample = -sample
		sign = 0x00
	}

	// 限幅
	if sample > clip {
		sample = clip
	}

	// 加偏置
	sample = sample + int16(mulawBias)
	exponent := mulawCompressTable[(sample>>7)&0xFF]
	mantissa := byte((sample >> (exponent + 3)) & 0x0F)
	mulaw := ^(sign | (exponent << 4) | mantissa)

	return mulaw
}

// mulawToLinear 将 mulaw 转换为 16-bit PCM
func mulawToLinear(mulaw byte) int16 {
	return mulawDecompressTable[mulaw]
}

// WAV 文件头结构
type WAVHeader struct {
	ChunkID       [4]byte // "RIFF"
	ChunkSize     uint32  // 文件大小 - 8
	Format        [4]byte // "WAVE"
	Subchunk1ID   [4]byte // "fmt "
	Subchunk1Size uint32  // 16 for PCM
	AudioFormat   uint16  // 7 for mulaw, 1 for PCM
	NumChannels   uint16  // 1 for mono
	SampleRate    uint32  // 8000
	ByteRate      uint32  // SampleRate * NumChannels * BitsPerSample/8
	BlockAlign    uint16  // NumChannels * BitsPerSample/8
	BitsPerSample uint16  // 8 for mulaw, 16 for PCM
	Subchunk2ID   [4]byte // "data"
	Subchunk2Size uint32  // NumSamples * NumChannels * BitsPerSample/8
}

// 创建 mulaw WAV 头
func createMulawWAVHeader(dataSize uint32) WAVHeader {
	header := WAVHeader{
		ChunkID:       [4]byte{'R', 'I', 'F', 'F'},
		ChunkSize:     dataSize + 36,
		Format:        [4]byte{'W', 'A', 'V', 'E'},
		Subchunk1ID:   [4]byte{'f', 'm', 't', ' '},
		Subchunk1Size: 18, // mulaw 需要 18 字节
		AudioFormat:   7,  // 7 = mulaw
		NumChannels:   1,
		SampleRate:    8000,
		ByteRate:      8000,
		BlockAlign:    1,
		BitsPerSample: 8,
		Subchunk2ID:   [4]byte{'d', 'a', 't', 'a'},
		Subchunk2Size: dataSize,
	}
	return header
}

// 写入 mulaw WAV 文件
func writeMulawWAVFile(filename string, mulawData []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	header := createMulawWAVHeader(uint32(len(mulawData)))

	// 写入 RIFF 头
	binary.Write(file, binary.LittleEndian, header.ChunkID)
	binary.Write(file, binary.LittleEndian, header.ChunkSize)
	binary.Write(file, binary.LittleEndian, header.Format)

	// 写入 fmt chunk
	binary.Write(file, binary.LittleEndian, header.Subchunk1ID)
	binary.Write(file, binary.LittleEndian, header.Subchunk1Size)
	binary.Write(file, binary.LittleEndian, header.AudioFormat)
	binary.Write(file, binary.LittleEndian, header.NumChannels)
	binary.Write(file, binary.LittleEndian, header.SampleRate)
	binary.Write(file, binary.LittleEndian, header.ByteRate)
	binary.Write(file, binary.LittleEndian, header.BlockAlign)
	binary.Write(file, binary.LittleEndian, header.BitsPerSample)

	// mulaw 格式需要额外的 2 字节（扩展大小）
	binary.Write(file, binary.LittleEndian, uint16(0))

	// 写入 data chunk
	binary.Write(file, binary.LittleEndian, header.Subchunk2ID)
	binary.Write(file, binary.LittleEndian, header.Subchunk2Size)
	binary.Write(file, binary.LittleEndian, mulawData)

	return nil
}

// AudioChunk 音频数据块
type AudioChunk struct {
	Data      []byte
	Timestamp time.Time
}

// ConversationMessage 对话消息
type ConversationMessage struct {
	Role    string // "user" 或 "assistant"
	Content []byte // 音频数据
	Text    string // 文本内容（可选）
}

// ConversationContext 对话上下文
type ConversationContext struct {
	SessionID string
	Messages  []ConversationMessage
	StartTime time.Time
}

// VoiceAgent 语音对话代理（全双工版本）
type VoiceAgent struct {
	bedrockClient *bedrockruntime.Client
	audioContext  *malgo.AllocatedContext
	modelID       string
	region        string
	awsConfig     aws.Config

	// VAD 检测器
	vad *VADDetector

	// 通道
	audioInputChan  chan AudioChunk // 录音 -> 发送
	audioOutputChan chan AudioChunk // 接收 -> 播放
	interruptChan   chan struct{}   // 打断信号

	// 对话上下文
	context *ConversationContext

	// 双向流
	httpClient *http.Client
	streamConn io.ReadWriteCloser

	// 播放控制
	playbackCtx    context.Context
	cancelPlayback context.CancelFunc

	// 状态标志
	isPlaying   bool
	isRecording bool
}

// NewVoiceAgent 创建新的语音对话代理
func NewVoiceAgent(ctx context.Context) (*VoiceAgent, error) {
	// 加载 AWS 配置，强制使用 us-east-1
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return nil, fmt.Errorf("加载AWS配置失败: %w", err)
	}

	// 创建 Bedrock Runtime 客户端
	bedrockClient := bedrockruntime.NewFromConfig(cfg)

	// 初始化音频上下文
	audioCtx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
		log.Printf("Malgo: %s", message)
	})
	if err != nil {
		return nil, fmt.Errorf("初始化音频上下文失败: %w", err)
	}

	// 创建 VAD 检测器
	vadConfig := DefaultVADConfig()
	vad := NewVADDetector(vadConfig)

	// 创建播放控制上下文
	playbackCtx, cancelPlayback := context.WithCancel(ctx)

	// 生成会话 ID
	sessionID := fmt.Sprintf("session_%d", time.Now().Unix())

	return &VoiceAgent{
		bedrockClient:   bedrockClient,
		audioContext:    audioCtx,
		modelID:         "amazon.nova-sonic-v1:0",
		region:          "us-east-1",
		awsConfig:       cfg,
		vad:             vad,
		audioInputChan:  make(chan AudioChunk, 10),
		audioOutputChan: make(chan AudioChunk, 100),
		interruptChan:   make(chan struct{}, 1),
		httpClient:      &http.Client{},
		context: &ConversationContext{
			SessionID: sessionID,
			Messages:  make([]ConversationMessage, 0),
			StartTime: time.Now(),
		},
		playbackCtx:    playbackCtx,
		cancelPlayback: cancelPlayback,
		isPlaying:      false,
		isRecording:    false,
	}, nil
}

// Close 清理资源
func (va *VoiceAgent) Close() {
	// 取消播放上下文
	if va.cancelPlayback != nil {
		va.cancelPlayback()
	}

	// 关闭通道
	close(va.interruptChan)
	close(va.audioInputChan)
	close(va.audioOutputChan)

	// 清理音频上下文
	if va.audioContext != nil {
		va.audioContext.Uninit()
		va.audioContext.Free()
	}
}

// AddUserMessage 添加用户消息到对话上下文
func (va *VoiceAgent) AddUserMessage(audioData []byte) {
	va.context.Messages = append(va.context.Messages, ConversationMessage{
		Role:    "user",
		Content: audioData,
	})
	fmt.Printf("📝 添加用户消息到上下文 (当前消息数: %d)\n", len(va.context.Messages))
}

// AddAssistantMessage 添加助手消息到对话上下文
func (va *VoiceAgent) AddAssistantMessage(audioData []byte, text string) {
	va.context.Messages = append(va.context.Messages, ConversationMessage{
		Role:    "assistant",
		Content: audioData,
		Text:    text,
	})
	fmt.Printf("📝 添加助手消息到上下文 (当前消息数: %d)\n", len(va.context.Messages))
}

// GetConversationHistory 获取对话历史
func (va *VoiceAgent) GetConversationHistory() []ConversationMessage {
	return va.context.Messages
}

// ClearConversationHistory 清除对话历史
func (va *VoiceAgent) ClearConversationHistory() {
	va.context.Messages = make([]ConversationMessage, 0)
	fmt.Println("🗑️  对话历史已清除")
}

// GetSessionInfo 获取会话信息
func (va *VoiceAgent) GetSessionInfo() (sessionID string, messageCount int, duration time.Duration) {
	return va.context.SessionID, len(va.context.Messages), time.Since(va.context.StartTime)
}

// ResetSession 重置会话（保留配置，清除历史）
func (va *VoiceAgent) ResetSession() {
	oldSessionID := va.context.SessionID
	va.context = &ConversationContext{
		SessionID: fmt.Sprintf("session_%d", time.Now().Unix()),
		Messages:  make([]ConversationMessage, 0),
		StartTime: time.Now(),
	}
	fmt.Printf("🔄 会话已重置: %s -> %s\n", oldSessionID, va.context.SessionID)
}

// StartContinuousRecording 启动连续录音线程（带 VAD 检测）
func (va *VoiceAgent) StartContinuousRecording(ctx context.Context) error {
	va.isRecording = true

	// 配置录音设备
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16 // 16-bit PCM
	deviceConfig.Capture.Channels = 1             // 单声道
	deviceConfig.SampleRate = 8000                // 8000 Hz
	deviceConfig.Alsa.NoMMap = 1

	// 语音缓冲区
	var currentSpeechBuffer []byte
	var isSpeaking bool = false

	// 数据回调函数
	onRecvFrames := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		if len(pInputSamples) == 0 {
			return
		}

		// 检测语音活动
		vadState := va.vad.Detect(pInputSamples)

		switch vadState {
		case StateSpeech:
			if !isSpeaking {
				// 语音开始
				fmt.Println("🎤 检测到语音，开始录音...")
				isSpeaking = true
				currentSpeechBuffer = make([]byte, 0)

				// 如果正在播放，触发打断
				if va.isPlaying {
					select {
					case va.interruptChan <- struct{}{}:
						fmt.Println("⚠️  打断 AI 播放")
					default:
					}
				}
			}

			// 将 PCM 数据转换为 mulaw 并添加到缓冲区
			for i := 0; i < len(pInputSamples); i += 2 {
				if i+1 < len(pInputSamples) {
					sample := int16(binary.LittleEndian.Uint16(pInputSamples[i : i+2]))
					mulawByte := linearToMulaw(sample)
					currentSpeechBuffer = append(currentSpeechBuffer, mulawByte)
				}
			}

		case StateSpeechEnd:
			if isSpeaking && len(currentSpeechBuffer) > 0 {
				// 语音结束，发送音频数据
				fmt.Printf("✓ 语音结束，录制了 %.2f 秒\n", float64(len(currentSpeechBuffer))/8000.0)

				// 发送到输入通道
				select {
				case va.audioInputChan <- AudioChunk{
					Data:      currentSpeechBuffer,
					Timestamp: time.Now(),
				}:
				case <-ctx.Done():
					return
				}

				// 重置状态
				isSpeaking = false
				currentSpeechBuffer = nil
			}

		case StateSilence:
			// 静音状态，什么都不做
		}
	}

	// 初始化设备
	device, err := malgo.InitDevice(va.audioContext.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: onRecvFrames,
	})
	if err != nil {
		return fmt.Errorf("初始化录音设备失败: %w", err)
	}

	// 启动录音
	err = device.Start()
	if err != nil {
		device.Uninit()
		return fmt.Errorf("启动录音失败: %w", err)
	}

	fmt.Println("✓ 连续录音已启动（使用 VAD 自动检测）")

	// 等待上下文取消
	go func() {
		<-ctx.Done()
		device.Stop()
		device.Uninit()
		va.isRecording = false
		fmt.Println("✓ 录音线程已停止")
	}()

	return nil
}

// RecordAudio 录制音频（保留旧方法用于兼容）
func (va *VoiceAgent) RecordAudio(duration time.Duration) ([]byte, error) {
	var recordedData []byte

	// 配置录音设备
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16 // 16-bit PCM
	deviceConfig.Capture.Channels = 1             // 单声道
	deviceConfig.SampleRate = 8000                // 8000 Hz
	deviceConfig.Alsa.NoMMap = 1

	// 数据回调函数
	onRecvFrames := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		// 将输入的 PCM 数据转换为 mulaw
		for i := 0; i < len(pInputSamples); i += 2 {
			if i+1 < len(pInputSamples) {
				sample := int16(binary.LittleEndian.Uint16(pInputSamples[i : i+2]))
				mulawByte := linearToMulaw(sample)
				recordedData = append(recordedData, mulawByte)
			}
		}
	}

	device, err := malgo.InitDevice(va.audioContext.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: onRecvFrames,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化录音设备失败: %w", err)
	}
	defer device.Uninit()

	// 开始录音
	err = device.Start()
	if err != nil {
		return nil, fmt.Errorf("启动录音失败: %w", err)
	}

	fmt.Printf("🎤 正在录音 (%v)...\n", duration)
	time.Sleep(duration)

	device.Stop()
	fmt.Printf("✓ 录音完成，共 %.2f 秒\n", float64(len(recordedData))/8000.0)

	return recordedData, nil
}

// StartContinuousPlayback 启动连续播放线程（支持流式播放和打断）
func (va *VoiceAgent) StartContinuousPlayback(ctx context.Context) error {
	// 配置播放设备
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Playback)
	deviceConfig.Playback.Format = malgo.FormatS16
	deviceConfig.Playback.Channels = 1
	deviceConfig.SampleRate = 8000
	deviceConfig.Alsa.NoMMap = 1

	// 播放缓冲队列
	var playbackBuffer []byte
	var bufferMutex sync.Mutex

	// 播放回调函数
	onSendFrames := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		bufferMutex.Lock()
		defer bufferMutex.Unlock()

		bytesNeeded := int(framecount) * 2 // 16-bit = 2 bytes per sample

		if len(playbackBuffer) == 0 {
			// 没有数据，输出静音
			for i := range pOutputSample {
				pOutputSample[i] = 0
			}
			return
		}

		bytesToCopy := bytesNeeded
		if bytesToCopy > len(playbackBuffer) {
			bytesToCopy = len(playbackBuffer)
		}

		copy(pOutputSample, playbackBuffer[:bytesToCopy])
		playbackBuffer = playbackBuffer[bytesToCopy:]

		// 填充剩余部分为静音
		for i := bytesToCopy; i < len(pOutputSample); i++ {
			pOutputSample[i] = 0
		}
	}

	// 初始化播放设备
	device, err := malgo.InitDevice(va.audioContext.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: onSendFrames,
	})
	if err != nil {
		return fmt.Errorf("初始化播放设备失败: %w", err)
	}

	// 启动播放
	err = device.Start()
	if err != nil {
		device.Uninit()
		return fmt.Errorf("启动播放失败: %w", err)
	}

	fmt.Println("✓ 连续播放已启动")

	// 播放控制协程
	go func() {
		defer device.Stop()
		defer device.Uninit()
		defer fmt.Println("✓ 播放线程已停止")

		for {
			select {
			case <-ctx.Done():
				return

			case <-va.interruptChan:
				// 收到打断信号，清空播放缓冲
				bufferMutex.Lock()
				playbackBuffer = nil
				bufferMutex.Unlock()
				va.isPlaying = false
				fmt.Println("⚠️  播放已中断")

			case chunk := <-va.audioOutputChan:
				// 收到音频数据
				if !va.isPlaying {
					va.isPlaying = true
					fmt.Println("🔊 开始播放 AI 回复...")
				}

				// 将 mulaw 转换为 PCM
				pcmData := make([]byte, len(chunk.Data)*2)
				for i, mulaw := range chunk.Data {
					sample := mulawToLinear(mulaw)
					binary.LittleEndian.PutUint16(pcmData[i*2:i*2+2], uint16(sample))
				}

				// 添加到播放缓冲
				bufferMutex.Lock()
				playbackBuffer = append(playbackBuffer, pcmData...)
				bufferMutex.Unlock()
			}
		}
	}()

	return nil
}

// PlayAudio 播放音频（保留旧方法用于兼容）
func (va *VoiceAgent) PlayAudio(mulawData []byte) error {
	// 将 mulaw 转换为 PCM
	pcmData := make([]byte, len(mulawData)*2)
	for i, mulaw := range mulawData {
		sample := mulawToLinear(mulaw)
		binary.LittleEndian.PutUint16(pcmData[i*2:i*2+2], uint16(sample))
	}

	playbackFinished := make(chan bool)
	currentPos := 0

	// 配置播放设备
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Playback)
	deviceConfig.Playback.Format = malgo.FormatS16
	deviceConfig.Playback.Channels = 1
	deviceConfig.SampleRate = 8000
	deviceConfig.Alsa.NoMMap = 1

	// 播放回调函数
	onSendFrames := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		bytesNeeded := int(framecount) * 2 // 16-bit = 2 bytes per sample
		if currentPos >= len(pcmData) {
			playbackFinished <- true
			return
		}

		bytesToCopy := bytesNeeded
		if currentPos+bytesToCopy > len(pcmData) {
			bytesToCopy = len(pcmData) - currentPos
		}

		copy(pOutputSample, pcmData[currentPos:currentPos+bytesToCopy])
		currentPos += bytesToCopy
	}

	device, err := malgo.InitDevice(va.audioContext.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: onSendFrames,
	})
	if err != nil {
		return fmt.Errorf("初始化播放设备失败: %w", err)
	}
	defer device.Uninit()

	err = device.Start()
	if err != nil {
		return fmt.Errorf("启动播放失败: %w", err)
	}

	fmt.Println("🔊 正在播放回复...")
	<-playbackFinished
	device.Stop()
	fmt.Println("✓ 播放完成")

	return nil
}

// ReceiveFromNova 流式接收 Nova 响应（占位符，当前集成在发送线程中）
// 注意：当 AWS SDK 真正支持 ConverseStream 时，这个方法将处理事件流
func (va *VoiceAgent) ReceiveFromNova(ctx context.Context, eventStream chan *bedrockruntime.ConverseStreamOutput) error {
	fmt.Println("📥 ConverseStream 接收线程已启动（当前集成在发送线程中）")

	for {
		select {
		case <-ctx.Done():
			fmt.Println("✓ 接收线程已停止")
			return ctx.Err()

		case event := <-eventStream:
			if event == nil {
				continue
			}

			// 处理不同类型的流式事件
			// 这里是 ConverseStream API 的事件处理逻辑
			// 当 AWS SDK 支持时，需要处理以下事件：
			// - ContentBlockStart
			// - ContentBlockDelta (音频数据块)
			// - ContentBlockStop
			// - MessageStart
			// - MessageStop
			// - Metadata

			fmt.Println("📥 收到流式事件（占位符）")
		}
	}
}

// StreamAudioToNova 使用双向流发送音频到 Nova Sonic
func (va *VoiceAgent) StreamAudioToNova(ctx context.Context, receiveChan chan<- *bedrockruntime.ConverseStreamOutput) error {
	fmt.Println("📤 Nova Sonic 双向流已启动")

	// 创建双向流
	stream, err := va.NewNovaSonicStream(ctx)
	if err != nil {
		return fmt.Errorf("创建流失败: %w", err)
	}
	defer stream.Close()

	// 启动流
	if err := stream.Start(ctx); err != nil {
		return fmt.Errorf("启动流失败: %w", err)
	}

	// 启动响应读取线程
	go func() {
		if err := stream.ReadResponses(ctx); err != nil && err != context.Canceled {
			log.Printf("❌ 读取响应错误: %v", err)
		}
	}()

	// 开始音频输入
	if err := stream.StartAudioInput(); err != nil {
		return fmt.Errorf("开始音频输入失败: %w", err)
	}

	// 持续发送音频
	for {
		select {
		case <-ctx.Done():
			stream.EndAudioInput()
			fmt.Println("✓ 发送线程已停止")
			return ctx.Err()

		case audioChunk := <-va.audioInputChan:
			// 收到音频数据
			fmt.Printf("📤 发送音频 (%.2f 秒)...\n", float64(len(audioChunk.Data))/8000.0)

			// mulaw 转 PCM (Nova Sonic 需要 16kHz PCM)
			pcmData := make([]byte, len(audioChunk.Data)*2)
			for i, mulaw := range audioChunk.Data {
				sample := mulawToLinear(mulaw)
				binary.LittleEndian.PutUint16(pcmData[i*2:], uint16(sample))
			}

			// 发送音频块
			if err := stream.SendAudioChunk(pcmData); err != nil {
				log.Printf("❌ 发送音频失败: %v", err)
				continue
			}

			// 音频发送完毕，结束并重新开始
			if err := stream.EndAudioInput(); err != nil {
				log.Printf("❌ 结束音频输入失败: %v", err)
			}

			// 等待短暂时间后重新开始新的音频输入
			time.Sleep(100 * time.Millisecond)
			if err := stream.StartAudioInput(); err != nil {
				log.Printf("❌ 重新开始音频输入失败: %v", err)
			}
		}
	}
}

// SendToNova 发送音频到 Nova 模型并获取响应（保留旧方法用于兼容）
func (va *VoiceAgent) SendToNova(ctx context.Context, audioData []byte) ([]byte, string, error) {
	// 将音频数据编码为 base64
	audioBase64 := base64.StdEncoding.EncodeToString(audioData)

	// 构建请求 - 使用 Nova 模型的正确格式
	request := map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"audio": map[string]interface{}{
							"format": "mulaw",
							"source": map[string]interface{}{
								"bytes": audioBase64,
							},
						},
					},
				},
			},
		},
		"inferenceConfig": map[string]interface{}{
			"maxTokens":   2048,
			"temperature": 0.7,
		},
		"audioOutput": map[string]interface{}{
			"format": "mulaw",
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, "", fmt.Errorf("序列化请求失败: %w", err)
	}

	fmt.Println("📤 正在发送音频到 Nova 模型...")

	// 调用 Bedrock InvokeModel API
	output, err := va.bedrockClient.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(va.modelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        requestBody,
	})
	if err != nil {
		return nil, "", fmt.Errorf("调用 Bedrock API 失败: %w", err)
	}

	// 解析响应
	var response map[string]interface{}
	if err := json.Unmarshal(output.Body, &response); err != nil {
		return nil, "", fmt.Errorf("解析响应失败: %w", err)
	}

	fmt.Println("✓ 收到 Nova 响应")

	// 提取文本和音频响应
	textResponse := ""
	var audioBytes []byte

	// 尝试从不同的响应结构中提取数据
	if outputData, ok := response["output"].(map[string]interface{}); ok {
		if message, ok := outputData["message"].(map[string]interface{}); ok {
			if content, ok := message["content"].([]interface{}); ok && len(content) > 0 {
				for _, item := range content {
					if contentItem, ok := item.(map[string]interface{}); ok {
						// 提取文本响应
						if text, ok := contentItem["text"].(string); ok {
							textResponse = text
							fmt.Printf("💬 Nova 回复（文本）: %s\n", text)
						}
						// 提取音频响应
						if audio, ok := contentItem["audio"].(map[string]interface{}); ok {
							if source, ok := audio["source"].(map[string]interface{}); ok {
								if bytesStr, ok := source["bytes"].(string); ok {
									audioBytes, err = base64.StdEncoding.DecodeString(bytesStr)
									if err != nil {
										return nil, "", fmt.Errorf("解码音频数据失败: %w", err)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// 如果有音频响应，返回音频
	if len(audioBytes) > 0 {
		return audioBytes, textResponse, nil
	}

	// 如果没有音频但有文本，使用文本转语音（TTS）
	// 注意：这里简化处理，实际可能需要调用其他TTS服务
	if textResponse != "" {
		// 返回空音频和文本，让调用者处理
		return nil, textResponse, nil
	}

	return nil, "", fmt.Errorf("响应中未找到音频或文本数据")
}

func main() {
	fmt.Println("=== AWS Bedrock Nova 全双工语音对话系统 ===")
	fmt.Println("模型: Nova Sonic | 采样率: 8000 Hz | 编码: mulaw")
	fmt.Println("特性: VAD 自动检测 | 实时流式对话 | 支持打断")
	fmt.Println()

	// 创建主上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建语音代理
	agent, err := NewVoiceAgent(ctx)
	if err != nil {
		log.Fatalf("❌ 创建语音代理失败: %v", err)
	}
	defer agent.Close()

	fmt.Println("✓ 语音代理已初始化")
	sessionID, _, _ := agent.GetSessionInfo()
	fmt.Printf("📋 会话 ID: %s\n", sessionID)
	fmt.Println()

	// 创建 output 目录（用于保存录音，可选）
	outputDir := "output"
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Printf("⚠️  创建 output 目录失败: %v", err)
		}
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 创建错误通道
	errChan := make(chan error, 4)

	// 启动所有线程
	fmt.Println("🚀 启动全双工语音对话系统...")
	fmt.Println()

	// 1. 启动连续录音线程（带 VAD 检测）
	go func() {
		if err := agent.StartContinuousRecording(ctx); err != nil {
			if err != context.Canceled {
				errChan <- fmt.Errorf("录音线程错误: %w", err)
			}
		}
	}()

	// 2. 启动连续播放线程（支持流式播放和打断）
	go func() {
		if err := agent.StartContinuousPlayback(ctx); err != nil {
			if err != context.Canceled {
				errChan <- fmt.Errorf("播放线程错误: %w", err)
			}
		}
	}()

	// 3. 启动流式发送线程（ConverseStream）
	go func() {
		if err := agent.StreamAudioToNova(ctx, nil); err != nil {
			if err != context.Canceled {
				errChan <- fmt.Errorf("发送线程错误: %w", err)
			}
		}
	}()

	// 4. 启动流式接收线程（占位符，当前集成在发送线程中）
	// 当真正的 ConverseStream API 可用时，启用此线程
	// go func() {
	// 	eventStream := make(chan *bedrockruntime.ConverseStreamOutput, 10)
	// 	if err := agent.ReceiveFromNova(ctx, eventStream); err != nil {
	// 		if err != context.Canceled {
	// 			errChan <- fmt.Errorf("接收线程错误: %w", err)
	// 		}
	// 	}
	// }()

	fmt.Println("✓ 所有线程已启动")
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("系统就绪！开始说话，系统会自动检测并处理。")
	fmt.Println("按 Ctrl+C 退出程序")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// 定期显示会话信息
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 主事件循环
	for {
		select {
		case <-sigChan:
			// 收到退出信号
			fmt.Println("\n\n🛑 收到退出信号，正在关闭...")
			cancel()

			// 显示最终统计
			sessionID, msgCount, duration := agent.GetSessionInfo()
			fmt.Printf("\n📊 会话统计:\n")
			fmt.Printf("   会话 ID: %s\n", sessionID)
			fmt.Printf("   消息数量: %d\n", msgCount)
			fmt.Printf("   会话时长: %s\n", duration.Round(time.Second))
			fmt.Println("\n✓ 程序已退出")
			return

		case err := <-errChan:
			// 收到线程错误
			log.Printf("❌ 线程错误: %v", err)
			log.Println("⚠️  尝试继续运行...")

		case <-ticker.C:
			// 定期显示会话信息
			sessionID, msgCount, duration := agent.GetSessionInfo()
			fmt.Printf("\n📊 [会话信息] ID: %s | 消息: %d | 时长: %s\n\n",
				sessionID, msgCount, duration.Round(time.Second))
		}
	}
}
