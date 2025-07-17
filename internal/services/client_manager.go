package services

import (
	"encoding/json"
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
	mutex        sync.RWMutex
	robotManager *RobotManager
}

// NewClientManager 创建新的客户端管理器
func NewClientManager(robotManager *RobotManager) *ClientManager {
	return &ClientManager{
		clients:      make(map[string]*models.Client),
		connections:  make(map[string]*websocket.Conn),
		robotManager: robotManager,
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

	// 保存连接
	cm.connections[client.UCode] = conn

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

// HandleMessage 处理客户端消息
func (cm *ClientManager) HandleMessage(ucode string, messageType int, data []byte) error {
	var message models.WebSocketMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// 更新客户端最后活跃时间
	cm.mutex.Lock()
	if client, exists := cm.clients[ucode]; exists {
		client.LastSeen = time.Now()
	}
	cm.mutex.Unlock()

	// 根据命令类型处理消息
	switch message.Command {
	case models.CMD_TYPE_BIND_ROBOT:
		return cm.handleBindRobot(ucode, message)
	case models.CMD_TYPE_CONTROL_ROBOT:
		return cm.handleControlRobot(ucode, message)
	case models.CMD_TYPE_PING:
		return cm.handlePing(ucode, message)
	default:
		log.Debug().
			Str("ucode", ucode).
			Str("command", string(message.Command)).
			Msg("Received message from client")
	}

	return nil
}

// handleBindRobot 处理机器人绑定请求
func (cm *ClientManager) handleBindRobot(ucode string, message models.WebSocketMessage) error {
	var bindData models.CMD_BIND_ROBOT
	if data, ok := message.Data.(map[string]interface{}); ok {
		if robotUCode, exists := data["ucode"].(string); exists {
			bindData.UCode = robotUCode
		}
	}

	// 检查机器人是否存在
	robot, err := cm.robotManager.GetRobot(bindData.UCode)
	if err != nil {
		response := models.WebSocketMessage{
			Type:       models.WSMessageTypeResponse,
			Command:    models.CMD_TYPE_BIND_ROBOT,
			Sequence:   message.Sequence,
			UCode:      ucode,
			ClientType: models.ClientTypeOperator,
			Version:    message.Version,
			Data: models.CMD_RESPONSE{
				Success:   false,
				Message:   fmt.Sprintf("机器人 %s 不存在", bindData.UCode),
				Timestamp: time.Now().Unix(),
			},
		}
		return cm.SendMessageToClient(ucode, response)
	}

	// 发送绑定成功响应
	response := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    models.CMD_TYPE_BIND_ROBOT,
		Sequence:   message.Sequence,
		UCode:      ucode,
		ClientType: models.ClientTypeOperator,
		Version:    message.Version,
		Data: models.RobotBindResponse{
			Success: true,
			Message: "机器人绑定成功",
			Robot:   robot,
		},
	}

	return cm.SendMessageToClient(ucode, response)
}

// handleControlRobot 处理机器人控制命令
func (cm *ClientManager) handleControlRobot(ucode string, message models.WebSocketMessage) error {
	// 解析控制命令
	var controlData models.CMD_CONTROL_ROBOT
	if data, ok := message.Data.(map[string]interface{}); ok {
		if action, exists := data["action"].(string); exists {
			controlData.Action = action
		}
		if params, exists := data["params"].(map[string]interface{}); exists {
			controlData.ParamMaps = make(map[string]string)
			for k, v := range params {
				if str, ok := v.(string); ok {
					controlData.ParamMaps[k] = str
				}
			}
		}
		if timestamp, exists := data["timestamp"].(float64); exists {
			controlData.Timestamp = int64(timestamp)
		}
	}

	// 创建机器人命令
	command := &models.RobotCommand{
		Action:        controlData.Action,
		Params:        controlData.ParamMaps,
		Priority:      5, // 默认优先级
		Timestamp:     controlData.Timestamp,
		OperatorUCode: ucode,
	}

	// 这里需要确定目标机器人，暂时使用第一个在线机器人
	robots := cm.robotManager.GetOnlineRobots()
	if len(robots) == 0 {
		response := models.WebSocketMessage{
			Type:       models.WSMessageTypeResponse,
			Command:    models.CMD_TYPE_CONTROL_ROBOT,
			Sequence:   message.Sequence,
			UCode:      ucode,
			ClientType: models.ClientTypeOperator,
			Version:    message.Version,
			Data: models.CMD_RESPONSE{
				Success:   false,
				Message:   "没有可用的在线机器人",
				Timestamp: time.Now().Unix(),
			},
		}
		return cm.SendMessageToClient(ucode, response)
	}

	// 发送命令到第一个在线机器人
	targetRobot := robots[0]
	commandResponse, err := cm.robotManager.SendCommand(targetRobot.UCode, command)
	if err != nil {
		response := models.WebSocketMessage{
			Type:       models.WSMessageTypeResponse,
			Command:    models.CMD_TYPE_CONTROL_ROBOT,
			Sequence:   message.Sequence,
			UCode:      ucode,
			ClientType: models.ClientTypeOperator,
			Version:    message.Version,
			Data: models.CMD_RESPONSE{
				Success:   false,
				Message:   fmt.Sprintf("命令发送失败: %v", err),
				Timestamp: time.Now().Unix(),
			},
		}
		return cm.SendMessageToClient(ucode, response)
	}

	// 发送成功响应
	response := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    models.CMD_TYPE_CONTROL_ROBOT,
		Sequence:   message.Sequence,
		UCode:      ucode,
		ClientType: models.ClientTypeOperator,
		Version:    message.Version,
		Data:       commandResponse,
	}

	return cm.SendMessageToClient(ucode, response)
}

// handlePing 处理心跳消息
func (cm *ClientManager) handlePing(ucode string, message models.WebSocketMessage) error {
	response := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    models.CMD_TYPE_PING,
		Sequence:   message.Sequence,
		UCode:      ucode,
		ClientType: models.ClientTypeOperator,
		Version:    message.Version,
		Data: models.CMD_RESPONSE{
			Success:   true,
			Message:   "pong",
			Timestamp: time.Now().Unix(),
		},
	}

	return cm.SendMessageToClient(ucode, response)
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
