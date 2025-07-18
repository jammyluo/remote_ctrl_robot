package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"remote-ctrl-robot/internal/models"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// MessageHandler 消息处理器回调函数类型
type MessageHandler func(service *WebSocketService, message *models.WebSocketMessage) error

// WebSocketService 统一的WebSocket连接管理服务
type WebSocketService struct {
	Conn       *websocket.Conn // WebSocket连接
	LastSeen   time.Time       // 最后活跃时间
	RemoteAddr string          // 远程地址
	Connected  bool            // 是否连接
	WriteMutex sync.Mutex      // 写锁
	Mutex      sync.RWMutex    // 读写锁
	ctx        context.Context
	cancel     context.CancelFunc

	// 消息处理器回调
	messageHandler MessageHandler
}

// NewWebSocketService 创建新的WebSocket服务
func NewWebSocketService(conn *websocket.Conn) *WebSocketService {
	ctx, cancel := context.WithCancel(context.Background())

	return &WebSocketService{
		Conn:       conn,
		RemoteAddr: conn.RemoteAddr().String(),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start 启动WebSocket服务
func (s *WebSocketService) Start() error {
	s.Mutex.Lock()
	s.Connected = true
	s.Mutex.Unlock()

	log.Info().
		Str("remote_addr", s.RemoteAddr).
		Msg("WebSocket service started")

	// 启动消息处理
	go s.handleMessages()

	return nil
}

// Stop 停止WebSocket服务
func (s *WebSocketService) Stop() error {
	log.Info().
		Str("remote_addr", s.RemoteAddr).
		Msg("Stopping WebSocket service")

	s.cancel()
	s.disconnect()

	return nil
}

// IsConnected 检查是否连接
func (s *WebSocketService) IsConnected() bool {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	// 检查连接状态和连接对象
	if !s.Connected || s.Conn == nil {
		return false
	}

	// 检查连接是否已经关闭
	select {
	case <-s.ctx.Done():
		return false
	default:
		return true
	}
}

// GetConnection 获取WebSocket连接
func (s *WebSocketService) GetConnection() *websocket.Conn {
	return s.Conn
}

// SendMessage 发送消息（带锁保护）
func (s *WebSocketService) SendMessage(message models.WebSocketMessage) error {
	if !s.IsConnected() {
		return fmt.Errorf("websocket not connected")
	}

	// 使用安全的写入方法
	return s.safeWriteJSON(message)
}

// SendSuccess 发送成功响应（带锁保护）
func (s *WebSocketService) SendSuccess(originalMsg *models.WebSocketMessage, message string) error {
	if !s.IsConnected() {
		return fmt.Errorf("websocket not connected")
	}

	if originalMsg == nil {
		return fmt.Errorf("original message is nil")
	}

	response := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    originalMsg.Command,
		Sequence:   originalMsg.Sequence,
		UCode:      originalMsg.UCode,
		ClientType: originalMsg.ClientType,
		Version:    originalMsg.Version,
		Data: models.CMD_RESPONSE{
			Success:   true,
			Message:   message,
			Timestamp: time.Now().Unix(),
		},
	}
	return s.safeWriteJSON(response)
}

// SendError 发送错误响应（带锁保护）
func (s *WebSocketService) SendError(originalMsg *models.WebSocketMessage, message string) error {
	if !s.IsConnected() {
		return fmt.Errorf("websocket not connected")
	}

	if originalMsg == nil {
		return fmt.Errorf("original message is nil")
	}

	response := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    originalMsg.Command,
		Sequence:   originalMsg.Sequence,
		UCode:      originalMsg.UCode,
		ClientType: originalMsg.ClientType,
		Version:    originalMsg.Version,
		Data: models.CMD_RESPONSE{
			Success:   false,
			Message:   message,
			Timestamp: time.Now().Unix(),
		},
	}

	return s.safeWriteJSON(response)
}

// handleMessages 处理来自WebSocket的消息
func (s *WebSocketService) handleMessages() {
	defer func() {
		s.disconnect()
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// 检查连接状态
			if !s.IsConnected() {
				log.Debug().Str("remote_addr", s.RemoteAddr).Msg("Connection lost, stopping message handler")
				return
			}

			// 设置读取超时
			s.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))

			_, data, err := s.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Error().Err(err).Str("remote_addr", s.RemoteAddr).Msg("WebSocket read error")
				} else {
					log.Debug().Err(err).Str("remote_addr", s.RemoteAddr).Msg("WebSocket connection closed")
				}
				return
			}

			// 处理消息
			if err := s.handleMessage(data); err != nil {
				log.Error().Err(err).Str("remote_addr", s.RemoteAddr).Msg("Failed to handle message")
				// 不在这里发送错误响应，避免并发写入问题
			}
		}
	}
}

// SetMessageHandler 设置消息处理器回调
func (s *WebSocketService) SetMessageHandler(handler MessageHandler) {
	s.messageHandler = handler
}

// handleMessage 处理单条消息
func (s *WebSocketService) handleMessage(data []byte) error {
	var message models.WebSocketMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}
	// 使用回调函数处理消息
	if s.messageHandler != nil {
		return s.messageHandler(s, &message)
	}

	return fmt.Errorf("no message handler set")
}

// disconnect 断开连接（内部方法）
func (s *WebSocketService) disconnect() {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	// 先标记为断开，防止新的写入操作
	s.Connected = false

	if s.Conn != nil {
		// 设置关闭状态，防止新的读写操作
		s.Conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
		s.Conn.Close()
		s.Conn = nil
	}

	log.Info().
		Str("remote_addr", s.RemoteAddr).
		Msg("WebSocket service disconnected")
}

// safeWriteJSON 安全的JSON写入方法
func (s *WebSocketService) safeWriteJSON(v interface{}) error {
	s.WriteMutex.Lock()
	defer s.WriteMutex.Unlock()

	// 再次检查连接状态，防止在获取锁期间连接断开
	if s.Conn == nil || !s.Connected {
		return fmt.Errorf("websocket connection is nil")
	}
	// 设置写入超时
	s.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	// 尝试写入
	err := s.Conn.WriteJSON(v)
	if err != nil {
		// 如果写入失败，标记连接为断开
		log.Warn().Err(err).Str("remote_addr", s.RemoteAddr).Msg("Failed to write to WebSocket, marking as disconnected")
		s.Connected = false
	}
	return err
}
