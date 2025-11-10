package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
)

// linearToMulaw 将 16-bit PCM 转换为 mulaw
func linearToMulaw(sample int16) byte {
	const maxMulaw = 0x1FFF
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

// WAV 文件头结构
type WAVHeader struct {
	ChunkID       [4]byte // "RIFF"
	ChunkSize     uint32  // 文件大小 - 8
	Format        [4]byte // "WAVE"
	Subchunk1ID   [4]byte // "fmt "
	Subchunk1Size uint32  // 16 for PCM
	AudioFormat   uint16  // 7 for mulaw
	NumChannels   uint16  // 1 for mono
	SampleRate    uint32  // 8000
	ByteRate      uint32  // SampleRate * NumChannels * BitsPerSample/8
	BlockAlign    uint16  // NumChannels * BitsPerSample/8
	BitsPerSample uint16  // 8 for mulaw
	Subchunk2ID   [4]byte // "data"
	Subchunk2Size uint32  // NumSamples * NumChannels * BitsPerSample/8
}

// 创建 WAV 头
func createWAVHeader(dataSize uint32) WAVHeader {
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

// 写入 WAV 文件
func writeWAVFile(filename string, mulawData []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	header := createWAVHeader(uint32(len(mulawData)))

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

func main() {
	fmt.Println("=== macOS 麦克风录音程序 ===")
	fmt.Println("采样率: 8000 Hz")
	fmt.Println("编码: mulaw (G.711)")
	fmt.Println("按 Ctrl+C 停止录音")
	fmt.Println()

	// 初始化 malgo 上下文
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
		log.Printf("Malgo: %s", message)
	})
	if err != nil {
		log.Fatal("初始化音频上下文失败:", err)
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	// 存储录音数据
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
				// 读取 16-bit 小端序样本
				sample := int16(binary.LittleEndian.Uint16(pInputSamples[i : i+2]))
				// 转换为 mulaw
				mulawByte := linearToMulaw(sample)
				recordedData = append(recordedData, mulawByte)
			}
		}
	}

	fmt.Println("正在初始化麦克风...")
	device, err := malgo.InitDevice(ctx.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: onRecvFrames,
	})
	if err != nil {
		log.Fatal("初始化录音设备失败:", err)
	}

	// 开始录音
	err = device.Start()
	if err != nil {
		log.Fatal("启动录音失败:", err)
	}
	defer device.Uninit()

	fmt.Println("✓ 录音已开始...")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n正在停止录音...")
	device.Stop()

	// 生成文件名
	filename := fmt.Sprintf("recording_%s.wav", time.Now().Format("20060102_150405"))

	// 保存为 WAV 文件
	fmt.Printf("保存录音到文件: %s\n", filename)
	err = writeWAVFile(filename, recordedData)
	if err != nil {
		log.Fatal("保存 WAV 文件失败:", err)
	}

	fmt.Printf("✓ 录音完成！共录制 %.2f 秒\n", float64(len(recordedData))/8000.0)
	fmt.Printf("✓ 文件大小: %d 字节\n", len(recordedData))
	fmt.Println("✓ 格式: mulaw, 8000 Hz, 单声道")
}
