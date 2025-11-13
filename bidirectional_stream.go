package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

// NovaSonicStream Nova Sonic 双向流客户端
type NovaSonicStream struct {
	agent      *VoiceAgent
	promptName string
	contentName string
	audioContentName string
	httpReq    *http.Request
	httpResp   *http.Response
	reader     io.Reader
	writer     io.WriteCloser
}

// NewNovaSonicStream 创建双向流
func (va *VoiceAgent) NewNovaSonicStream(ctx context.Context) (*NovaSonicStream, error) {
	endpoint := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/invoke-with-bidirectional-stream",
		va.region, va.modelID)

	// 创建 pipe 用于双向通信
	pipeReader, pipeWriter := io.Pipe()

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, pipeReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// AWS SigV4 签名
	credentials, err := va.awsConfig.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取凭证失败: %w", err)
	}

	signer := v4.NewSigner()
	payloadHash := sha256.Sum256([]byte{})
	err = signer.SignHTTP(ctx, credentials, req, hex.EncodeToString(payloadHash[:]), "bedrock", va.region, time.Now())
	if err != nil {
		return nil, fmt.Errorf("签名失败: %w", err)
	}

	stream := &NovaSonicStream{
		agent:            va,
		promptName:       fmt.Sprintf("prompt_%d", time.Now().UnixNano()),
		contentName:      fmt.Sprintf("content_%d", time.Now().UnixNano()),
		audioContentName: fmt.Sprintf("audio_%d", time.Now().UnixNano()),
		httpReq:          req,
		writer:           pipeWriter,
	}

	return stream, nil
}

// Start 启动流
func (s *NovaSonicStream) Start(ctx context.Context) error {
	// 发送请求
	resp, err := s.agent.httpClient.Do(s.httpReq)
	if err != nil {
		return fmt.Errorf("建立连接失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("请求失败 %d: %s", resp.StatusCode, string(body))
	}

	s.httpResp = resp
	s.reader = resp.Body

	// 发送初始化事件序列
	if err := s.sendSessionStart(); err != nil {
		return err
	}

	if err := s.sendPromptStart(); err != nil {
		return err
	}

	if err := s.sendSystemPrompt(); err != nil {
		return err
	}

	return nil
}

// sendEvent 发送事件
func (s *NovaSonicStream) sendEvent(event map[string]interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// 添加换行符（流式传输需要）
	data = append(data, '\n')

	_, err = s.writer.Write(data)
	return err
}

// sendSessionStart 发送会话开始事件
func (s *NovaSonicStream) sendSessionStart() error {
	event := map[string]interface{}{
		"event": map[string]interface{}{
			"sessionStart": map[string]interface{}{
				"inferenceConfiguration": map[string]interface{}{
					"maxTokens":   1024,
					"topP":        0.9,
					"temperature": 0.7,
				},
			},
		},
	}
	fmt.Println("📤 发送 sessionStart")
	return s.sendEvent(event)
}

// sendPromptStart 发送提示开始事件
func (s *NovaSonicStream) sendPromptStart() error {
	event := map[string]interface{}{
		"event": map[string]interface{}{
			"promptStart": map[string]interface{}{
				"promptName": s.promptName,
				"textOutputConfiguration": map[string]interface{}{
					"mediaType": "text/plain",
				},
				"audioOutputConfiguration": map[string]interface{}{
					"mediaType":        "audio/lpcm",
					"sampleRateHertz":  24000,
					"sampleSizeBits":   16,
					"channelCount":     1,
					"voiceId":          "matthew",
					"encoding":         "base64",
					"audioType":        "SPEECH",
				},
			},
		},
	}
	fmt.Println("📤 发送 promptStart")
	return s.sendEvent(event)
}

// sendSystemPrompt 发送系统提示
func (s *NovaSonicStream) sendSystemPrompt() error {
	// contentStart
	event1 := map[string]interface{}{
		"event": map[string]interface{}{
			"contentStart": map[string]interface{}{
				"promptName":  s.promptName,
				"contentName": s.contentName,
				"type":        "TEXT",
				"interactive": true,
				"role":        "SYSTEM",
				"textInputConfiguration": map[string]interface{}{
					"mediaType": "text/plain",
				},
			},
		},
	}
	if err := s.sendEvent(event1); err != nil {
		return err
	}

	// textInput
	systemPrompt := "你是一个友好的中文助手。用简短的中文回复，一般2-3句话。"
	event2 := map[string]interface{}{
		"event": map[string]interface{}{
			"textInput": map[string]interface{}{
				"promptName":  s.promptName,
				"contentName": s.contentName,
				"content":     systemPrompt,
			},
		},
	}
	if err := s.sendEvent(event2); err != nil {
		return err
	}

	// contentEnd
	event3 := map[string]interface{}{
		"event": map[string]interface{}{
			"contentEnd": map[string]interface{}{
				"promptName":  s.promptName,
				"contentName": s.contentName,
			},
		},
	}
	fmt.Println("📤 发送 system prompt")
	return s.sendEvent(event3)
}

// StartAudioInput 开始音频输入
func (s *NovaSonicStream) StartAudioInput() error {
	event := map[string]interface{}{
		"event": map[string]interface{}{
			"contentStart": map[string]interface{}{
				"promptName":  s.promptName,
				"contentName": s.audioContentName,
				"type":        "AUDIO",
				"interactive": true,
				"role":        "USER",
				"audioInputConfiguration": map[string]interface{}{
					"mediaType":        "audio/lpcm",
					"sampleRateHertz":  16000,
					"sampleSizeBits":   16,
					"channelCount":     1,
					"audioType":        "SPEECH",
					"encoding":         "base64",
				},
			},
		},
	}
	fmt.Println("📤 开始音频输入")
	return s.sendEvent(event)
}

// SendAudioChunk 发送音频块
func (s *NovaSonicStream) SendAudioChunk(audioData []byte) error {
	audioBase64 := base64.StdEncoding.EncodeToString(audioData)
	event := map[string]interface{}{
		"event": map[string]interface{}{
			"audioInput": map[string]interface{}{
				"promptName":  s.promptName,
				"contentName": s.audioContentName,
				"content":     audioBase64,
			},
		},
	}
	return s.sendEvent(event)
}

// EndAudioInput 结束音频输入
func (s *NovaSonicStream) EndAudioInput() error {
	event := map[string]interface{}{
		"event": map[string]interface{}{
			"contentEnd": map[string]interface{}{
				"promptName":  s.promptName,
				"contentName": s.audioContentName,
			},
		},
	}
	fmt.Println("📤 结束音频输入")
	return s.sendEvent(event)
}

// ReadResponses 读取响应
func (s *NovaSonicStream) ReadResponses(ctx context.Context) error {
	decoder := json.NewDecoder(s.reader)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var response map[string]interface{}
			if err := decoder.Decode(&response); err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}

			// 处理响应
			if err := s.handleResponse(response); err != nil {
				fmt.Printf("❌ 处理响应错误: %v\n", err)
			}
		}
	}
}

// handleResponse 处理响应事件
func (s *NovaSonicStream) handleResponse(response map[string]interface{}) error {
	event, ok := response["event"].(map[string]interface{})
	if !ok {
		return nil
	}

	// 处理文本输出
	if textOutput, ok := event["textOutput"].(map[string]interface{}); ok {
		if content, ok := textOutput["content"].(string); ok {
			if role, ok := textOutput["role"].(string); ok {
				if role == "ASSISTANT" {
					fmt.Printf("💬 Nova: %s\n", content)
				} else if role == "USER" {
					fmt.Printf("👤 识别: %s\n", content)
				}
			}
		}
	}

	// 处理音频输出
	if audioOutput, ok := event["audioOutput"].(map[string]interface{}); ok {
		if content, ok := audioOutput["content"].(string); ok {
			audioBytes, err := base64.StdEncoding.DecodeString(content)
			if err == nil && len(audioBytes) > 0 {
				// 注意：输出是 24kHz PCM，需要转换为 8kHz mulaw
				// 暂时跳过播放
				fmt.Printf("🔊 收到音频 %d 字节\n", len(audioBytes))
			}
		}
	}

	return nil
}

// Close 关闭流
func (s *NovaSonicStream) Close() error {
	// 发送结束事件
	event1 := map[string]interface{}{
		"event": map[string]interface{}{
			"promptEnd": map[string]interface{}{
				"promptName": s.promptName,
			},
		},
	}
	s.sendEvent(event1)

	event2 := map[string]interface{}{
		"event": map[string]interface{}{
			"sessionEnd": map[string]interface{}{},
		},
	}
	s.sendEvent(event2)

	if s.writer != nil {
		s.writer.Close()
	}
	if s.httpResp != nil {
		s.httpResp.Body.Close()
	}

	return nil
}

