package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type JanusService struct {
	HTTPURL      string
	WebSocketURL string
	StreamID     int
	HTTPClient   *http.Client
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

// 获取WebRTC播放URL
func (js *JanusService) GetWebRTCPlayURL() (string, error) {
	// 创建会话
	sessionID, err := js.CreateSession()
	if err != nil {
		return "", err
	}

	// 附加流插件
	handleID, err := js.AttachStreamingPlugin(sessionID)
	if err != nil {
		return "", err
	}

	// 构建WebRTC URL
	// 这里返回一个简化的URL，实际使用时需要根据Janus的WebRTC API构建完整的SDP
	webrtcURL := fmt.Sprintf("%s?session=%d&handle=%d&stream=%d",
		js.WebSocketURL, sessionID, handleID, js.StreamID)

	log.Info().
		Int("session_id", sessionID).
		Int("handle_id", handleID).
		Int("stream_id", js.StreamID).
		Str("webrtc_url", webrtcURL).
		Msg("Generated WebRTC play URL")

	return webrtcURL, nil
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
