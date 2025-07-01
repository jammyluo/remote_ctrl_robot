package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"remote-ctrl-robot/internal/models"

	"github.com/rs/zerolog/log"
)

type JanusService struct {
	HTTPURL      string
	WebSocketURL string
	StreamID     int
	HTTPClient   *http.Client
	streams      map[string]*models.WebRTCStream // UCode -> WebRTCStream
	mutex        sync.RWMutex
}

type JanusRequest struct {
	Janus     string      `json:"janus"`
	SessionID int         `json:"session_id,omitempty"`
	HandleID  int         `json:"handle_id,omitempty"`
	Body      interface{} `json:"body,omitempty"`
	JSEP      interface{} `json:"jsep,omitempty"`
}

type JanusResponse struct {
	Janus     string      `json:"janus"`
	SessionID int         `json:"session_id,omitempty"`
	HandleID  int         `json:"handle_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	JSEP      interface{} `json:"jsep,omitempty"`
	Error     *JanusError `json:"error,omitempty"`
}

type JanusError struct {
	Code   int    `json:"code"`
	Reason string `json:"reason"`
}

func NewJanusService(httpURL, wsURL string, streamID int) *JanusService {
	return &JanusService{
		HTTPURL:      httpURL,
		WebSocketURL: wsURL,
		StreamID:     streamID,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		streams: make(map[string]*models.WebRTCStream),
	}
}

// 创建会话
func (js *JanusService) CreateSession() (int, error) {
	req := JanusRequest{
		Janus: "create",
	}

	response, err := js.sendRequest(req)
	if err != nil {
		return 0, fmt.Errorf("failed to create session: %w", err)
	}

	if response.Error != nil {
		return 0, fmt.Errorf("janus error: %s", response.Error.Reason)
	}

	if data, ok := response.Data.(map[string]interface{}); ok {
		if id, ok := data["id"].(float64); ok {
			return int(id), nil
		}
	}

	return 0, fmt.Errorf("invalid session response")
}

// 附加流插件
func (js *JanusService) AttachStreamingPlugin(sessionID int) (int, error) {
	req := JanusRequest{
		Janus:     "attach",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"plugin": "janus.plugin.streaming",
		},
	}

	response, err := js.sendRequest(req)
	if err != nil {
		return 0, fmt.Errorf("failed to attach streaming plugin: %w", err)
	}

	if response.Error != nil {
		return 0, fmt.Errorf("janus error: %s", response.Error.Reason)
	}

	if data, ok := response.Data.(map[string]interface{}); ok {
		if id, ok := data["id"].(float64); ok {
			return int(id), nil
		}
	}

	return 0, fmt.Errorf("invalid handle response")
}

// 为UCode注册WebRTC流
func (js *JanusService) RegisterWebRTCStream(ucode string) (*models.WebRTCStream, error) {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	// 检查是否已存在
	if stream, exists := js.streams[ucode]; exists {
		if stream.Status == "active" {
			return stream, nil
		}
	}

	// 创建新的Janus会话
	sessionID, err := js.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// 附加流插件
	handleID, err := js.AttachStreamingPlugin(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to attach streaming plugin: %w", err)
	}

	// 生成流ID（使用递增的方式）
	streamID := js.StreamID + len(js.streams)

	// 构建URL
	playURL := fmt.Sprintf("%s?session=%d&handle=%d&stream=%d",
		js.WebSocketURL, sessionID, handleID, streamID)

	pushURL := fmt.Sprintf("%s/push?session=%d&handle=%d&stream=%d",
		js.HTTPURL, sessionID, handleID, streamID)

	// 创建流信息
	stream := &models.WebRTCStream{
		UCode:     ucode,
		StreamID:  streamID,
		SessionID: int64(sessionID),
		HandleID:  int64(handleID),
		PlayURL:   playURL,
		PushURL:   pushURL,
		Status:    "active",
		CreatedAt: time.Now().UnixMilli(),
	}

	// 保存流信息
	js.streams[ucode] = stream

	log.Info().
		Str("ucode", ucode).
		Int("session_id", sessionID).
		Int("handle_id", handleID).
		Int("stream_id", streamID).
		Msg("Registered WebRTC stream for UCode")

	return stream, nil
}

// 获取UCode的WebRTC播放地址列表
func (js *JanusService) GetWebRTCPlayURLs(ucode string) ([]string, error) {
	js.mutex.RLock()
	defer js.mutex.RUnlock()

	stream, exists := js.streams[ucode]
	if !exists {
		return nil, fmt.Errorf("no stream found for UCode: %s", ucode)
	}

	if stream.Status != "active" {
		return nil, fmt.Errorf("stream is not active for UCode: %s", ucode)
	}

	return []string{stream.PlayURL}, nil
}

// 获取所有活跃流的播放地址
func (js *JanusService) GetAllWebRTCPlayURLs() map[string][]string {
	js.mutex.RLock()
	defer js.mutex.RUnlock()

	result := make(map[string][]string)
	for ucode, stream := range js.streams {
		if stream.Status == "active" {
			result[ucode] = []string{stream.PlayURL}
		}
	}
	return result
}

// 获取UCode的推流地址
func (js *JanusService) GetWebRTCPushURL(ucode string) (string, error) {
	js.mutex.RLock()
	defer js.mutex.RUnlock()

	stream, exists := js.streams[ucode]
	if !exists {
		return "", fmt.Errorf("no stream found for UCode: %s", ucode)
	}

	if stream.Status != "active" {
		return "", fmt.Errorf("stream is not active for UCode: %s", ucode)
	}

	return stream.PushURL, nil
}

// 停用UCode的流
func (js *JanusService) DeactivateStream(ucode string) error {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	stream, exists := js.streams[ucode]
	if !exists {
		return fmt.Errorf("no stream found for UCode: %s", ucode)
	}

	stream.Status = "inactive"
	log.Info().Str("ucode", ucode).Msg("Deactivated WebRTC stream")
	return nil
}

// 删除UCode的流
func (js *JanusService) DeleteStream(ucode string) error {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	if _, exists := js.streams[ucode]; !exists {
		return fmt.Errorf("no stream found for UCode: %s", ucode)
	}

	delete(js.streams, ucode)
	log.Info().Str("ucode", ucode).Msg("Deleted WebRTC stream")
	return nil
}

// 获取所有流信息
func (js *JanusService) GetAllStreams() map[string]*models.WebRTCStream {
	js.mutex.RLock()
	defer js.mutex.RUnlock()

	result := make(map[string]*models.WebRTCStream)
	for ucode, stream := range js.streams {
		result[ucode] = stream
	}
	return result
}

// 检查Janus状态
func (js *JanusService) CheckStatus() error {
	req := JanusRequest{
		Janus: "info",
	}

	response, err := js.sendRequest(req)
	if err != nil {
		return fmt.Errorf("janus not responding: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("janus error: %s", response.Error.Reason)
	}

	log.Info().Msg("Janus server is healthy")
	return nil
}

// 发送请求到Janus
func (js *JanusService) sendRequest(req JanusRequest) (*JanusResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", js.HTTPURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := js.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var response JanusResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// 批量注册WebRTC流
func (js *JanusService) BatchRegisterWebRTCStreams(ucodes []string) map[string]*models.WebRTCStream {
	results := make(map[string]*models.WebRTCStream)

	for _, ucode := range ucodes {
		stream, err := js.RegisterWebRTCStream(ucode)
		if err != nil {
			log.Error().Err(err).Str("ucode", ucode).Msg("Failed to register WebRTC stream")
			continue
		}
		results[ucode] = stream
	}

	return results
}

// 获取流统计信息
func (js *JanusService) GetStreamStats() map[string]interface{} {
	js.mutex.RLock()
	defer js.mutex.RUnlock()

	total := len(js.streams)
	active := 0
	inactive := 0

	for _, stream := range js.streams {
		switch stream.Status {
		case "active":
			active++
		case "inactive":
			inactive++
		}
	}

	return map[string]interface{}{
		"total_streams":    total,
		"active_streams":   active,
		"inactive_streams": inactive,
	}
}

// 清理无效流
func (js *JanusService) CleanupInactiveStreams() int {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	cleaned := 0
	for ucode, stream := range js.streams {
		if stream.Status == "inactive" {
			delete(js.streams, ucode)
			cleaned++
			log.Info().Str("ucode", ucode).Msg("Cleaned up inactive stream")
		}
	}

	return cleaned
}
