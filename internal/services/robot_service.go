package services

import (
	"fmt"
	"sync"
	"time"

	"remote-ctrl-robot/internal/models"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type RobotService struct {
	websocketURL    string
	conn            *websocket.Conn
	mutex           sync.RWMutex
	connected       bool
	lastHeartbeat   time.Time
	latency         int64
	totalCommands   int64
	failedCommands  int64
	lastCommandTime time.Time
	clients         map[*websocket.Conn]bool
	clientMutex     sync.RWMutex
}

func NewRobotService(websocketURL string) *RobotService {
	return &RobotService{
		websocketURL: websocketURL,
		clients:      make(map[*websocket.Conn]bool),
	}
}

// 连接到机器人
func (s *RobotService) Connect() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.connected {
		return nil
	}

	// 连接到机器人WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(s.websocketURL, nil)
	if err != nil {
		log.Error().Err(err).Str("url", s.websocketURL).Msg("Failed to connect to robot")
		return fmt.Errorf("failed to connect to robot: %w", err)
	}

	s.conn = conn
	s.connected = true
	s.lastHeartbeat = time.Now()

	// 启动心跳检测
	go s.heartbeatLoop()

	log.Info().Str("url", s.websocketURL).Msg("Connected to robot")
	return nil
}

// 断开连接
func (s *RobotService) Disconnect() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.connected {
		return nil
	}

	if s.conn != nil {
		s.conn.Close()
	}

	s.connected = false
	log.Info().Msg("Disconnected from robot")
	return nil
}

// 发送命令到机器人
func (s *RobotService) SendCommand(command models.ControlCommand) error {
	s.mutex.Lock()
	s.totalCommands++
	s.lastCommandTime = time.Now()
	s.mutex.Unlock()

	// 检查连接状态
	s.mutex.RLock()
	if !s.connected || s.conn == nil {
		s.mutex.RUnlock()
		s.mutex.Lock()
		s.failedCommands++
		s.mutex.Unlock()

		log.Error().
			Str("command_id", command.CommandID).
			Msg("Cannot send command: robot not connected")

		return fmt.Errorf("robot not connected")
	}
	s.mutex.RUnlock()

	// 创建命令消息
	message := models.WebSocketMessage{
		Type:    "control_command",
		Message: "Control command from operator",
		Data:    command,
	}

	// 发送命令到真实机器人
	if err := s.conn.WriteJSON(message); err != nil {
		s.mutex.Lock()
		s.failedCommands++
		s.mutex.Unlock()

		log.Error().
			Err(err).
			Str("command_id", command.CommandID).
			Msg("Failed to send command to robot")

		return fmt.Errorf("failed to send command to robot: %w", err)
	}

	// 记录成功发送的命令
	log.Info().
		Str("command_id", command.CommandID).
		Int("priority", command.Priority).
		Int64("timestamp", command.Timestamp).
		Msg("Command sent to robot successfully")

	// 广播给所有客户端
	s.broadcastToClients(message)

	return nil
}

// 获取连接状态
func (s *RobotService) GetConnectionStatus() models.ConnectionStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return models.ConnectionStatus{
		Connected:       s.connected,
		LastHeartbeat:   s.lastHeartbeat,
		Latency:         s.latency,
		ActiveClients:   s.GetActiveClientsCount(),
		TotalCommands:   s.totalCommands,
		FailedCommands:  s.failedCommands,
		LastCommandTime: s.lastCommandTime,
	}
}

// 心跳检测循环
func (s *RobotService) heartbeatLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.sendHeartbeat(); err != nil {
				log.Error().Err(err).Msg("Heartbeat failed")
				s.mutex.Lock()
				s.connected = false
				s.mutex.Unlock()
				return
			}
		}
	}
}

// 发送心跳
func (s *RobotService) sendHeartbeat() error {
	s.mutex.RLock()
	if !s.connected {
		s.mutex.RUnlock()
		return fmt.Errorf("not connected")
	}
	s.mutex.RUnlock()

	start := time.Now()
	message := models.WebSocketMessage{
		Type:    "ping",
		Message: "Heartbeat",
		Data: map[string]interface{}{
			"timestamp": start.UnixMilli(),
		},
	}

	if err := s.conn.WriteJSON(message); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	// 等待pong响应
	s.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var response models.WebSocketMessage
	if err := s.conn.ReadJSON(&response); err != nil {
		return fmt.Errorf("failed to receive pong: %w", err)
	}

	if response.Type != "pong" {
		return fmt.Errorf("unexpected response type: %s", response.Type)
	}

	latency := time.Since(start).Milliseconds()
	s.mutex.Lock()
	s.latency = latency
	s.lastHeartbeat = time.Now()
	s.mutex.Unlock()

	log.Debug().Int64("latency_ms", latency).Msg("Heartbeat successful")
	return nil
}

// 添加客户端
func (s *RobotService) AddClient(conn *websocket.Conn) {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()
	s.clients[conn] = true
}

// 移除客户端
func (s *RobotService) RemoveClient(conn *websocket.Conn) {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()
	delete(s.clients, conn)
}

// 广播消息给所有客户端
func (s *RobotService) broadcastToClients(message models.WebSocketMessage) {
	s.clientMutex.RLock()
	defer s.clientMutex.RUnlock()

	for conn := range s.clients {
		if err := conn.WriteJSON(message); err != nil {
			log.Error().Err(err).Msg("Failed to broadcast message to client")
			// 标记连接为无效，稍后清理
			conn.Close()
		}
	}
}

// 获取活跃客户端数量
func (s *RobotService) GetActiveClientsCount() int {
	s.clientMutex.RLock()
	defer s.clientMutex.RUnlock()
	return len(s.clients)
}

// 清理断开的客户端
func (s *RobotService) CleanupDisconnectedClients() {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()

	for conn := range s.clients {
		// 发送ping来检查连接状态
		conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			log.Debug().Msg("Removing disconnected client")
			conn.Close()
			delete(s.clients, conn)
		}
	}
}
