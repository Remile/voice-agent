package main

import (
	"encoding/binary"
	"math"
)

// VADState 语音活动检测状态
type VADState int

const (
	// StateSilence 静音状态
	StateSilence VADState = iota
	// StateSpeech 语音活动中
	StateSpeech
	// StateSpeechEnd 语音结束（过渡状态）
	StateSpeechEnd
)

// VADConfig VAD 配置参数
type VADConfig struct {
	// EnergyThreshold 能量阈值（RMS），用于判断是否为语音
	EnergyThreshold float64
	// SpeechStartFrames 连续多少帧超过阈值才判定为语音开始
	SpeechStartFrames int
	// SpeechEndFrames 连续多少帧低于阈值才判定为语音结束
	SpeechEndFrames int
	// SampleRate 采样率
	SampleRate int
	// FrameSize 每帧的样本数
	FrameSize int
}

// DefaultVADConfig 返回默认的 VAD 配置
func DefaultVADConfig() VADConfig {
	return VADConfig{
		EnergyThreshold:   500.0, // 根据实际环境调整
		SpeechStartFrames: 3,     // 约 300ms @ 100ms per frame
		SpeechEndFrames:   8,     // 约 800ms @ 100ms per frame
		SampleRate:        8000,
		FrameSize:         800, // 100ms @ 8000Hz
	}
}

// VADDetector 语音活动检测器
type VADDetector struct {
	config        VADConfig
	currentState  VADState
	speechFrames  int // 连续语音帧计数
	silenceFrames int // 连续静音帧计数
}

// NewVADDetector 创建新的 VAD 检测器
func NewVADDetector(config VADConfig) *VADDetector {
	return &VADDetector{
		config:       config,
		currentState: StateSilence,
	}
}

// CalculateRMS 计算音频帧的 RMS（均方根）能量
func (vad *VADDetector) CalculateRMS(audioData []byte) float64 {
	if len(audioData) < 2 {
		return 0
	}

	var sum float64
	sampleCount := len(audioData) / 2 // 16-bit samples

	for i := 0; i < len(audioData)-1; i += 2 {
		sample := int16(binary.LittleEndian.Uint16(audioData[i : i+2]))
		sum += float64(sample) * float64(sample)
	}

	if sampleCount == 0 {
		return 0
	}

	rms := math.Sqrt(sum / float64(sampleCount))
	return rms
}

// CalculateRMSMulaw 计算 mulaw 音频帧的 RMS 能量
func (vad *VADDetector) CalculateRMSMulaw(mulawData []byte) float64 {
	if len(mulawData) == 0 {
		return 0
	}

	var sum float64
	for _, mulaw := range mulawData {
		sample := mulawToLinear(mulaw)
		sum += float64(sample) * float64(sample)
	}

	rms := math.Sqrt(sum / float64(len(mulawData)))
	return rms
}

// Detect 检测音频帧的语音活动状态
// audioData: 16-bit PCM 音频数据
// 返回当前状态
func (vad *VADDetector) Detect(audioData []byte) VADState {
	energy := vad.CalculateRMS(audioData)
	return vad.processEnergy(energy)
}

// DetectMulaw 检测 mulaw 编码音频帧的语音活动状态
func (vad *VADDetector) DetectMulaw(mulawData []byte) VADState {
	energy := vad.CalculateRMSMulaw(mulawData)
	return vad.processEnergy(energy)
}

// processEnergy 根据能量值处理状态转换
func (vad *VADDetector) processEnergy(energy float64) VADState {
	isSpeech := energy > vad.config.EnergyThreshold

	switch vad.currentState {
	case StateSilence:
		if isSpeech {
			vad.speechFrames++
			vad.silenceFrames = 0
			if vad.speechFrames >= vad.config.SpeechStartFrames {
				vad.currentState = StateSpeech
				return StateSpeech
			}
		} else {
			vad.speechFrames = 0
		}
		return StateSilence

	case StateSpeech:
		if isSpeech {
			vad.silenceFrames = 0
			vad.speechFrames++
			return StateSpeech
		} else {
			vad.silenceFrames++
			vad.speechFrames = 0
			if vad.silenceFrames >= vad.config.SpeechEndFrames {
				vad.currentState = StateSpeechEnd
				return StateSpeechEnd
			}
			// 还在语音状态，只是暂时的静音（可能是停顿）
			return StateSpeech
		}

	case StateSpeechEnd:
		// 语音结束后自动转到静音状态
		vad.currentState = StateSilence
		vad.speechFrames = 0
		vad.silenceFrames = 0
		return StateSilence
	}

	return vad.currentState
}

// Reset 重置 VAD 状态
func (vad *VADDetector) Reset() {
	vad.currentState = StateSilence
	vad.speechFrames = 0
	vad.silenceFrames = 0
}

// GetState 获取当前状态
func (vad *VADDetector) GetState() VADState {
	return vad.currentState
}

// SetEnergyThreshold 动态调整能量阈值
func (vad *VADDetector) SetEnergyThreshold(threshold float64) {
	vad.config.EnergyThreshold = threshold
}

// GetEnergyThreshold 获取当前能量阈值
func (vad *VADDetector) GetEnergyThreshold() float64 {
	return vad.config.EnergyThreshold
}

// CalibrateThreshold 根据环境噪音自动校准阈值
// noiseData: 环境噪音样本
func (vad *VADDetector) CalibrateThreshold(noiseData []byte) {
	noiseRMS := vad.CalculateRMS(noiseData)
	// 设置阈值为噪音的 3 倍
	vad.config.EnergyThreshold = noiseRMS * 3.0
}
