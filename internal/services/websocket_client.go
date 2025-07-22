package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// WebSocketClient WebSocket连接管理服务
type WebSocketClient struct {
	conn      *websocket.Conn
	connected bool
	done      chan struct{}
	url        string
	writeTimeout time.Duration
	readTimeout  time.Duration
	connectTimeout time.Duration
	reconnectDelay time.Duration

	// 重连相关字段
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

// NewWebSocketClient 创建新的WebSocket服务
func NewWebSocketClient(url string, writeTimeout time.Duration, readTimeout time.Duration, connectTimeout time.Duration, reconnectDelay time.Duration) *WebSocketClient {
	return &WebSocketClient{
		done: make(chan struct{}),
		url:  url,
		writeTimeout: writeTimeout,
		readTimeout:  readTimeout,
		connectTimeout: connectTimeout,
		reconnectDelay: reconnectDelay,
	}
}

// SetCallbacks 设置回调函数
func (ws *WebSocketClient) SetCallbacks(
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
func (ws *WebSocketClient) Start() error {
	log.Info().
		Str("server", ws.url).
		Msg("Starting WebSocket service")

	return ws.connect()
}

// Stop 停止WebSocket服务
func (ws *WebSocketClient) Stop() {
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
func (ws *WebSocketClient) IsConnected() bool {
	ws.connMutex.Lock()
	defer ws.connMutex.Unlock()
	return ws.connected
}

// SendMessage 发送消息
func (ws *WebSocketClient) SendMessage(message interface{}) error {
	ws.connMutex.Lock()
	defer ws.connMutex.Unlock()

	if ws.conn == nil {
		return fmt.Errorf("connection is nil")
	}

	// 设置写入超时
	ws.conn.SetWriteDeadline(time.Now().Add(ws.writeTimeout))
	return ws.conn.WriteJSON(message)
}

// connect 建立连接
func (ws *WebSocketClient) connect() error {
	// 设置WebSocket连接选项
	dialer := websocket.Dialer{
		HandshakeTimeout: ws.connectTimeout,
	}

	// 连接WebSocket
	conn, _, err := dialer.Dial(ws.url, nil)
	if err != nil {
		return fmt.Errorf("connect failed: %v", err)
	}

	ws.connMutex.Lock()
	ws.conn = conn
	ws.connected = true
	ws.connMutex.Unlock()

	// 设置连接超时
	ws.connMutex.Lock()
	ws.conn.SetReadDeadline(time.Now().Add(ws.readTimeout))
	ws.conn.SetWriteDeadline(time.Now().Add(ws.writeTimeout))
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

	// log.Info().
	// 	Str("server", ws.url).
	// 	Msg("WebSocket connected successfully")
	return nil
}

// scheduleReconnect 安排重连
func (ws *WebSocketClient) scheduleReconnect() {
	log.Info().
		Dur("delay", ws.reconnectDelay).
		Msg("Scheduling reconnect")

	// 设置重连定时器
	ws.reconnectTimer = time.AfterFunc(ws.reconnectDelay, func() {
		ws.performReconnect()
	})
}

// performReconnect 执行重连
func (ws *WebSocketClient) performReconnect() {
	select {
	case <-ws.done:
		return
	default:
	}

	ws.lastReconnectTime = time.Now()

	log.Info().
		Msg("Attempting to reconnect")

	if err := ws.connect(); err != nil {
		log.Error().
			Err(err).
			Msg("Reconnect failed")

		// 安排下次重连
		ws.scheduleReconnect()
	} else {
		log.Info().
			Msg("Reconnect successful")
	}
}

// handleMessages 处理接收到的消息
func (ws *WebSocketClient) handleMessages() {
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
			ws.conn.SetReadDeadline(time.Now().Add(ws.readTimeout))
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
				log.Info().
					Str("message", string(message)).
					Msg("Received message")
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
func (ws *WebSocketClient) GetStats() map[string]interface{} {
	ws.connMutex.Lock()
	defer ws.connMutex.Unlock()

	return map[string]interface{}{
		"connected":           ws.connected,
		"last_reconnect_time": ws.lastReconnectTime,
		"server_url":          ws.url,
	}
}
