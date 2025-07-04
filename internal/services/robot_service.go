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
func (s *RobotService) SendCommand(command models.CMD_CONTROL_ROBOT) error {
	// s.mutex.Lock()
	// s.totalCommands++
	// s.lastCommandTime = time.Now()
	// s.mutex.Unlock()

	// // 检查连接状态
	// s.mutex.RLock()
	// if !s.connected || s.conn == nil {
	// 	s.mutex.RUnlock()
	// 	s.mutex.Lock()
	// 	s.failedCommands++
	// 	s.mutex.Unlock()

	// 	log.Error().
	// 		Str("command_id", command.Action).
	// 		Msg("Cannot send command: robot not connected")

	// 	return fmt.Errorf("robot not connected")
	// }
	// s.mutex.RUnlock()

	// // 创建命令消息
	// message := models.WebSocketMessage{
	// 	Type:       models.WSMessageTypeRequest,
	// 	Command:    models.CMD_TYPE_CONTROL_ROBOT,
	// 	Sequence:   0,
	// 	UCode:      client.UCode,
	// 	ClientType: client.ClientType,
	// 	Version:    client.Version,
	// 	Data:       command,
	// }

	// // 发送命令到真实机器人
	// if err := s.conn.WriteJSON(message); err != nil {
	// 	s.mutex.Lock()
	// 	s.failedCommands++
	// 	s.mutex.Unlock()

	// 	log.Error().
	// 		Err(err).
	// 		Str("command_id", command.CommandID).
	// 		Msg("Failed to send command to robot")

	// 	return fmt.Errorf("failed to send command to robot: %w", err)
	// }

	// // 记录成功发送的命令
	// log.Info().
	// 	Str("command_id", command.CommandID).
	// 	Int("priority", command.Priority).
	// 	Int64("timestamp", command.Timestamp).
	// 	Msg("Command sent to robot successfully")

	// // 广播给所有客户端
	// s.broadcastToClients(message)

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
		// 根据最后心跳时间来检查连接状态
		if time.Since(s.lastHeartbeat) > 10*time.Second {
			log.Debug().Msg("Removing disconnected client")
			conn.Close()
			delete(s.clients, conn)
		}
	}
}
