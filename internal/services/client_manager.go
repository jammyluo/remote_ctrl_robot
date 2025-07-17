package services

import (
	"fmt"
	"sync"
	"time"

	"remote-ctrl-robot/internal/models"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// ClientManager 客户端管理器
type ClientManager struct {
	clients      map[string]*models.Client  // 客户端映射表
	connections  map[string]*websocket.Conn // 连接映射表
	services     map[string]*ClientService  // 客户端服务映射表
	mutex        sync.RWMutex
	robotManager *RobotManager
	gameService  *GameService
}

// NewClientManager 创建新的客户端管理器
func NewClientManager(robotManager *RobotManager, gameService *GameService) *ClientManager {
	return &ClientManager{
		clients:      make(map[string]*models.Client),
		connections:  make(map[string]*websocket.Conn),
		services:     make(map[string]*ClientService),
		robotManager: robotManager,
		gameService:  gameService,
	}
}

// AddClient 添加客户端
func (cm *ClientManager) AddClient(client *models.Client, conn *websocket.Conn) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 检查客户端是否已存在
	if existing, exists := cm.clients[client.UCode]; exists {
		// 关闭旧连接
		if oldConn, exists := cm.connections[client.UCode]; exists {
			oldConn.Close()
		}

		// 停止旧服务
		if oldService, exists := cm.services[client.UCode]; exists {
			oldService.Stop()
		}

		// 更新客户端信息
		existing.Name = client.Name
		existing.Version = client.Version
		existing.Connected = true
		existing.LastSeen = time.Now()
		existing.RemoteAddr = client.RemoteAddr

		log.Info().
			Str("ucode", client.UCode).
			Str("name", client.Name).
			Msg("Client reconnected")
	} else {
		// 创建新客户端
		client.Connected = true
		client.LastSeen = time.Now()
		cm.clients[client.UCode] = client

		log.Info().
			Str("ucode", client.UCode).
			Str("name", client.Name).
			Str("type", string(client.ClientType)).
			Msg("New client connected")
	}

	// 创建客户端服务
	service := NewClientService(client, cm, cm.robotManager, cm.gameService)
	if err := service.Start(); err != nil {
		log.Error().
			Err(err).
			Str("ucode", client.UCode).
			Msg("Failed to start client service")
		return err
	}

	// 设置连接
	service.SetConnection(conn)

	// 保存连接和服务
	cm.connections[client.UCode] = conn
	cm.services[client.UCode] = service

	// 发送欢迎消息
	welcomeMsg := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    models.CMD_TYPE_REGISTER,
		Sequence:   0,
		UCode:      client.UCode,
		ClientType: client.ClientType,
		Version:    client.Version,
		Data: models.CMD_RESPONSE{
			Success:   true,
			Message:   "连接成功",
			Timestamp: time.Now().Unix(),
		},
	}

	if err := conn.WriteJSON(welcomeMsg); err != nil {
		log.Error().
			Err(err).
			Str("ucode", client.UCode).
			Msg("Failed to send welcome message")
	}

	return nil
}

// RemoveClient 移除客户端
func (cm *ClientManager) RemoveClient(ucode string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 停止服务
	if service, exists := cm.services[ucode]; exists {
		if err := service.Stop(); err != nil {
			log.Error().
				Err(err).
				Str("ucode", ucode).
				Msg("Failed to stop client service")
		}
		delete(cm.services, ucode)
	}

	// 关闭连接
	if conn, exists := cm.connections[ucode]; exists {
		conn.Close()
		delete(cm.connections, ucode)
	}

	// 更新客户端状态
	if client, exists := cm.clients[ucode]; exists {
		client.Connected = false
		client.LastSeen = time.Now()

		log.Info().
			Str("ucode", ucode).
			Str("name", client.Name).
			Msg("Client disconnected")
	}

	return nil
}

// GetClient 获取客户端
func (cm *ClientManager) GetClient(ucode string) (*models.Client, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	client, exists := cm.clients[ucode]
	if !exists {
		return nil, fmt.Errorf("client %s not found", ucode)
	}

	return client, nil
}

// GetAllClients 获取所有客户端
func (cm *ClientManager) GetAllClients() []*models.Client {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	clients := make([]*models.Client, 0, len(cm.clients))
	for _, client := range cm.clients {
		clients = append(clients, client)
	}

	return clients
}

// GetConnectedClients 获取已连接的客户端
func (cm *ClientManager) GetConnectedClients() []*models.Client {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var connectedClients []*models.Client
	for _, client := range cm.clients {
		if client.Connected {
			connectedClients = append(connectedClients, client)
		}
	}

	return connectedClients
}

// GetClientCount 获取客户端数量
func (cm *ClientManager) GetClientCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return len(cm.clients)
}

// GetConnectedClientCount 获取已连接客户端数量
func (cm *ClientManager) GetConnectedClientCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	count := 0
	for _, client := range cm.clients {
		if client.Connected {
			count++
		}
	}
	return count
}

// SendMessageToClient 发送消息给指定客户端
func (cm *ClientManager) SendMessageToClient(ucode string, message models.WebSocketMessage) error {
	cm.mutex.RLock()
	conn, exists := cm.connections[ucode]
	cm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not connected", ucode)
	}

	if err := conn.WriteJSON(message); err != nil {
		log.Error().
			Err(err).
			Str("ucode", ucode).
			Msg("Failed to send message to client")
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// BroadcastMessage 广播消息给所有客户端
func (cm *ClientManager) BroadcastMessage(message models.WebSocketMessage) {
	cm.mutex.RLock()
	connections := make(map[string]*websocket.Conn)
	for ucode, conn := range cm.connections {
		connections[ucode] = conn
	}
	cm.mutex.RUnlock()

	for ucode, conn := range connections {
		if err := conn.WriteJSON(message); err != nil {
			log.Error().
				Err(err).
				Str("ucode", ucode).
				Msg("Failed to broadcast message to client")

			// 标记客户端断开
			go cm.RemoveClient(ucode)
		}
	}
}

// BroadcastToOperators 广播消息给所有操作员
func (cm *ClientManager) BroadcastToOperators(message models.WebSocketMessage) {
	cm.mutex.RLock()
	connections := make(map[string]*websocket.Conn)
	for ucode, conn := range cm.connections {
		if client, exists := cm.clients[ucode]; exists && client.ClientType == models.ClientTypeOperator {
			connections[ucode] = conn
		}
	}
	cm.mutex.RUnlock()

	for ucode, conn := range connections {
		if err := conn.WriteJSON(message); err != nil {
			log.Error().
				Err(err).
				Str("ucode", ucode).
				Msg("Failed to broadcast message to operator")

			// 标记客户端断开
			go cm.RemoveClient(ucode)
		}
	}
}

// CleanupDisconnectedClients 清理断开的客户端
func (cm *ClientManager) CleanupDisconnectedClients() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	var toRemove []string

	for ucode, client := range cm.clients {
		// 检查最后活跃时间，超过5分钟认为断开
		if now.Sub(client.LastSeen) > 5*time.Minute {
			toRemove = append(toRemove, ucode)
		}
	}

	for _, ucode := range toRemove {
		// 关闭连接
		if conn, exists := cm.connections[ucode]; exists {
			conn.Close()
			delete(cm.connections, ucode)
		}

		// 更新客户端状态
		if client, exists := cm.clients[ucode]; exists {
			client.Connected = false
			log.Info().
				Str("ucode", ucode).
				Str("name", client.Name).
				Msg("Disconnected client cleaned up")
		}
	}
}

// GetClientStatistics 获取客户端统计信息
func (cm *ClientManager) GetClientStatistics() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_clients"] = len(cm.clients)

	connectedCount := 0
	operatorCount := 0
	robotCount := 0

	for _, client := range cm.clients {
		if client.Connected {
			connectedCount++
		}
		if client.ClientType == models.ClientTypeOperator {
			operatorCount++
		} else if client.ClientType == models.ClientTypeRobot {
			robotCount++
		}
	}

	stats["connected_clients"] = connectedCount
	stats["operator_clients"] = operatorCount
	stats["robot_clients"] = robotCount

	return stats
}
