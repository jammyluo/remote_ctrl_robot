package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// WebSocketService WebSocket连接管理服务
type WebSocketService struct {
	config    *Config
	conn      *websocket.Conn
	connected bool
	done      chan struct{}

	// 重连相关字段
	reconnectAttempts int
	lastReconnectTime time.Time
	reconnectTimer    *time.Timer

	// 并发安全
	connMutex sync.Mutex

	// 回调函数
	onConnect    func() error
	onDisconnect func()
	onMessage    func([]byte) error
	onError      func(error)
}

// NewWebSocketService 创建新的WebSocket服务
func NewWebSocketService(config *Config) *WebSocketService {
	return &WebSocketService{
		config: config,
		done:   make(chan struct{}),
	}
}

// SetCallbacks 设置回调函数
func (ws *WebSocketService) SetCallbacks(
	onConnect func() error,
	onDisconnect func(),
	onMessage func([]byte) error,
	onError func(error),
) {
	ws.onConnect = onConnect
	ws.onDisconnect = onDisconnect
	ws.onMessage = onMessage
	ws.onError = onError
}

// Start 启动WebSocket服务
func (ws *WebSocketService) Start() error {
	log.Info().
		Str("server", ws.config.Server.URL).
		Msg("Starting WebSocket service")

	return ws.connect()
}

// Stop 停止WebSocket服务
func (ws *WebSocketService) Stop() {
	log.Info().Msg("Stopping WebSocket service")

	ws.connMutex.Lock()
	ws.connected = false
	ws.connMutex.Unlock()

	close(ws.done)

	// 停止重连定时器
	if ws.reconnectTimer != nil {
		ws.reconnectTimer.Stop()
	}

	ws.connMutex.Lock()
	if ws.conn != nil {
		ws.conn.Close()
	}
	ws.connMutex.Unlock()

	log.Info().Msg("WebSocket service stopped")
}

// IsConnected 检查是否已连接
func (ws *WebSocketService) IsConnected() bool {
	ws.connMutex.Lock()
	defer ws.connMutex.Unlock()
	return ws.connected
}

// SendMessage 发送消息
func (ws *WebSocketService) SendMessage(message interface{}) error {
	ws.connMutex.Lock()
	defer ws.connMutex.Unlock()

	if ws.conn == nil {
		return fmt.Errorf("connection is nil")
	}

	// 设置写入超时
	ws.conn.SetWriteDeadline(time.Now().Add(ws.config.GetWriteTimeout()))
	return ws.conn.WriteJSON(message)
}

// connect 建立连接
func (ws *WebSocketService) connect() error {
	// 设置WebSocket连接选项
	dialer := websocket.Dialer{
		HandshakeTimeout: ws.config.GetConnectTimeout(),
	}

	// 连接WebSocket
	conn, _, err := dialer.Dial(ws.config.Server.URL, nil)
	if err != nil {
		return fmt.Errorf("connect failed: %v", err)
	}

	ws.connMutex.Lock()
	ws.conn = conn
	ws.connected = true
	ws.connMutex.Unlock()

	// 重置重连计数
	ws.reconnectAttempts = 0

	// 设置连接超时
	ws.connMutex.Lock()
	ws.conn.SetReadDeadline(time.Now().Add(ws.config.GetReadTimeout()))
	ws.conn.SetWriteDeadline(time.Now().Add(ws.config.GetWriteTimeout()))
	ws.connMutex.Unlock()

	// 启动消息处理
	go ws.handleMessages()

	// 调用连接回调
	if ws.onConnect != nil {
		if err := ws.onConnect(); err != nil {
			ws.connMutex.Lock()
			ws.conn.Close()
			ws.connected = false
			ws.connMutex.Unlock()
			return fmt.Errorf("onConnect callback failed: %v", err)
		}
	}

	log.Info().
		Str("server", ws.config.Server.URL).
		Msg("WebSocket connected successfully")
	return nil
}

// scheduleReconnect 安排重连
func (ws *WebSocketService) scheduleReconnect() {
	if !ws.config.Reconnect.Enabled {
		log.Warn().Msg("Reconnect disabled, not attempting to reconnect")
		return
	}

	if ws.reconnectAttempts >= ws.config.Reconnect.MaxAttempts {
		log.Error().
			Int("max_attempts", ws.config.Reconnect.MaxAttempts).
			Msg("Max reconnect attempts reached, giving up")
		return
	}

	// 计算重连延迟
	delay := ws.calculateReconnectDelay()

	log.Info().
		Int("attempt", ws.reconnectAttempts+1).
		Int("max_attempts", ws.config.Reconnect.MaxAttempts).
		Dur("delay", delay).
		Msg("Scheduling reconnect")

	// 设置重连定时器
	ws.reconnectTimer = time.AfterFunc(delay, func() {
		ws.performReconnect()
	})
}

// calculateReconnectDelay 计算重连延迟
func (ws *WebSocketService) calculateReconnectDelay() time.Duration {
	// 指数退避算法
	baseDelay := time.Duration(ws.config.Reconnect.InitialDelay) * time.Second
	maxDelay := time.Duration(ws.config.Reconnect.MaxDelay) * time.Second

	// 计算延迟：baseDelay * (backoff_multiplier ^ attempts)
	delay := float64(baseDelay) * pow(ws.config.Reconnect.BackoffMultiplier, ws.reconnectAttempts)

	// 添加随机抖动 (±20%)
	jitter := delay * 0.2 * (rand.Float64()*2 - 1)
	delay += jitter

	// 确保延迟在合理范围内
	if delay < float64(baseDelay) {
		delay = float64(baseDelay)
	}
	if delay > float64(maxDelay) {
		delay = float64(maxDelay)
	}

	return time.Duration(delay)
}

// performReconnect 执行重连
func (ws *WebSocketService) performReconnect() {
	select {
	case <-ws.done:
		return
	default:
	}

	ws.reconnectAttempts++
	ws.lastReconnectTime = time.Now()

	log.Info().
		Int("attempt", ws.reconnectAttempts).
		Int("max_attempts", ws.config.Reconnect.MaxAttempts).
		Msg("Attempting to reconnect")

	if err := ws.connect(); err != nil {
		log.Error().
			Err(err).
			Int("attempt", ws.reconnectAttempts).
			Msg("Reconnect failed")

		// 安排下次重连
		ws.scheduleReconnect()
	} else {
		log.Info().
			Int("attempt", ws.reconnectAttempts).
			Msg("Reconnect successful")
	}
}

// handleMessages 处理接收到的消息
func (ws *WebSocketService) handleMessages() {
	for {
		select {
		case <-ws.done:
			return
		default:
			ws.connMutex.Lock()
			if ws.conn == nil {
				ws.connMutex.Unlock()
				return
			}
			// 设置读取超时
			ws.conn.SetReadDeadline(time.Now().Add(ws.config.GetReadTimeout()))
			ws.connMutex.Unlock()

			_, message, err := ws.conn.ReadMessage()
			if err != nil {
				ws.connMutex.Lock()
				if ws.connected {
					log.Error().
						Err(err).
						Msg("Read message error")

					// 标记连接断开
					ws.connected = false
					ws.conn.Close()
					ws.conn = nil
					ws.connMutex.Unlock()

					// 调用断开回调
					if ws.onDisconnect != nil {
						ws.onDisconnect()
					}

					// 安排重连
					ws.scheduleReconnect()
				} else {
					ws.connMutex.Unlock()
				}
				return
			}

			// 调用消息回调
			if ws.onMessage != nil {
				if err := ws.onMessage(message); err != nil {
					log.Error().
						Err(err).
						Msg("Message callback failed")

					// 调用错误回调
					if ws.onError != nil {
						ws.onError(err)
					}
				}
			}
		}
	}
}

// GetStats 获取连接统计信息
func (ws *WebSocketService) GetStats() map[string]interface{} {
	ws.connMutex.Lock()
	defer ws.connMutex.Unlock()

	return map[string]interface{}{
		"connected":           ws.connected,
		"reconnect_attempts":  ws.reconnectAttempts,
		"last_reconnect_time": ws.lastReconnectTime,
		"server_url":          ws.config.Server.URL,
	}
}
