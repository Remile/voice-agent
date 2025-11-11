package main

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
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

// VoiceAgent 语音对话代理
type VoiceAgent struct {
	bedrockClient *bedrockruntime.Client
	audioContext  *malgo.AllocatedContext
	modelID       string
}

// NewVoiceAgent 创建新的语音对话代理
func NewVoiceAgent(ctx context.Context) (*VoiceAgent, error) {
	// 加载 AWS 配置
	cfg, err := config.LoadDefaultConfig(ctx)
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

	return &VoiceAgent{
		bedrockClient: bedrockClient,
		audioContext:  audioCtx,
		modelID:       "us.amazon.nova-pro-v1:0", // Nova Pro 模型
	}, nil
}

// Close 清理资源
func (va *VoiceAgent) Close() {
	if va.audioContext != nil {
		va.audioContext.Uninit()
		va.audioContext.Free()
	}
}

// RecordAudio 录制音频
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

// PlayAudio 播放音频
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

// SendToNova 发送音频到 Nova 模型并获取响应
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
	fmt.Println("=== AWS Bedrock Nova 语音对话系统 ===")
	fmt.Println("采样率: 8000 Hz | 编码: mulaw | 声道: 单声道")
	fmt.Println()

	ctx := context.Background()

	// 创建语音代理
	agent, err := NewVoiceAgent(ctx)
	if err != nil {
		log.Fatalf("创建语音代理失败: %v", err)
	}
	defer agent.Close()

	fmt.Println("✓ 语音代理已初始化")
	fmt.Println("按 Ctrl+C 退出程序")
	fmt.Println()

	// 创建 output 目录
	outputDir := "output"
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Fatalf("创建 output 目录失败: %v", err)
		}
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 对话循环
	conversationCount := 0
	for {
		select {
		case <-sigChan:
			fmt.Println("\n\n程序已退出")
			return
		default:
			conversationCount++
			fmt.Printf("\n━━━━━━━━ 对话 #%d ━━━━━━━━\n\n", conversationCount)

			// 1. 录制用户语音
			fmt.Println("请说话...")
			audioData, err := agent.RecordAudio(5 * time.Second)
			if err != nil {
				log.Printf("录音失败: %v", err)
				continue
			}

			// 保存录音文件（可选）
			timestamp := time.Now().Format("20060102_150405")
			inputFile := fmt.Sprintf("%s/input_%s.wav", outputDir, timestamp)
			if err := writeMulawWAVFile(inputFile, audioData); err != nil {
				log.Printf("保存录音文件失败: %v", err)
			} else {
				fmt.Printf("💾 录音已保存: %s\n", inputFile)
			}

			// 2. 发送到 Nova 并获取响应
			responseAudio, responseText, err := agent.SendToNova(ctx, audioData)
			if err != nil {
				log.Printf("发送到 Nova 失败: %v", err)
				continue
			}

			// 如果有音频响应
			if len(responseAudio) > 0 {
				// 保存响应音频文件（可选）
				outputFile := fmt.Sprintf("%s/response_%s.wav", outputDir, timestamp)
				if err := writeMulawWAVFile(outputFile, responseAudio); err != nil {
					log.Printf("保存响应文件失败: %v", err)
				} else {
					fmt.Printf("💾 响应已保存: %s\n", outputFile)
				}

				// 3. 播放 Nova 的响应
				if err := agent.PlayAudio(responseAudio); err != nil {
					log.Printf("播放音频失败: %v", err)
					continue
				}
			} else if responseText != "" {
				// 如果只有文本响应，显示文本
				fmt.Printf("💬 Nova 回复（仅文本）: %s\n", responseText)
				fmt.Println("⚠️  注意：此模型可能不支持音频输出，请检查模型配置")
			}

			fmt.Println("\n准备下一轮对话...")
			time.Sleep(1 * time.Second)
		}
	}
}
