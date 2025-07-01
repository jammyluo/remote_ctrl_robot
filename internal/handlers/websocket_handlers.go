package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"remote-ctrl-robot/internal/models"
	"remote-ctrl-robot/internal/services"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type WebSocketHandlers struct {
	robotService *services.RobotService
	upgrader     websocket.Upgrader

	// 机器人连接管理
	robotClients   map[*websocket.Conn]*models.RobotConnection
	robotUcodeConn map[string]*websocket.Conn
	robotConnUcode map[*websocket.Conn]string

	// 操作者连接管理
	operatorClients map[*websocket.Conn]*models.OperatorConnection
	operatorIdConn  map[string]*websocket.Conn
	operatorConnId  map[*websocket.Conn]string

	mutex sync.RWMutex
}

func NewWebSocketHandlers(robotService *services.RobotService) *WebSocketHandlers {
	return &WebSocketHandlers{
		robotService: robotService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源，生产环境应该限制
			},
		},
		robotClients:    make(map[*websocket.Conn]*models.RobotConnection),
		robotUcodeConn:  make(map[string]*websocket.Conn),
		robotConnUcode:  make(map[*websocket.Conn]string),
		operatorClients: make(map[*websocket.Conn]*models.OperatorConnection),
		operatorIdConn:  make(map[string]*websocket.Conn),
		operatorConnId:  make(map[*websocket.Conn]string),
	}
}

// 处理WebSocket连接
func (h *WebSocketHandlers) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("WebSocket connection received")
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade connection to WebSocket")
		return
	}
	defer conn.Close()

	// 注册：第一条消息必须为register，带ucode和client_type
	var rawMsg map[string]interface{}
	if err := conn.ReadJSON(&rawMsg); err != nil {
		h.sendRegisterError(conn, "Failed to parse registration message")
		return
	}

	// 检查消息类型
	msgType, ok := rawMsg["type"].(string)
	if !ok || msgType != "register" {
		h.sendRegisterError(conn, "UCODE and CLIENT_TYPE required, please send register message first")
		return
	}

	// 获取客户端类型
	clientTypeStr, ok := rawMsg["client_type"].(string)
	if !ok || clientTypeStr == "" {
		h.sendRegisterError(conn, "Invalid CLIENT_TYPE")
		return
	}

	// 验证客户端类型
	clientType := models.ClientType(clientTypeStr)
	if clientType != models.ClientTypeRobot && clientType != models.ClientTypeOperator {
		h.sendRegisterError(conn, "Invalid CLIENT_TYPE, must be 'robot' or 'operator'")
		return
	}

	// 根据客户端类型分别处理注册
	switch clientType {
	case models.ClientTypeRobot:
		h.handleRobotRegistration(conn, rawMsg)
	case models.ClientTypeOperator:
		h.handleOperatorRegistration(conn, rawMsg)
	}
}

// 处理机器人注册
func (h *WebSocketHandlers) handleRobotRegistration(conn *websocket.Conn, rawMsg map[string]interface{}) {
	// 获取机器人UCode
	ucode, ok := rawMsg["ucode"].(string)
	if !ok || ucode == "" {
		h.sendRegisterError(conn, "Invalid UCODE for robot")
		return
	}

	// 获取可选字段
	version, _ := rawMsg["version"].(string)

	// 检查机器人UCode是否已被使用
	h.mutex.Lock()
	if _, exists := h.robotUcodeConn[ucode]; exists {
		h.mutex.Unlock()
		h.sendRegisterError(conn, "Robot UCODE already in use")
		return
	}

	// 创建机器人连接信息
	robotConn := &models.RobotConnection{
		UCode:      ucode,
		Version:    version,
		Connected:  true,
		LastSeen:   time.Now(),
		RemoteAddr: conn.RemoteAddr().String(),
	}

	// 绑定连接
	h.robotUcodeConn[ucode] = conn
	h.robotConnUcode[conn] = ucode
	h.robotClients[conn] = robotConn
	h.mutex.Unlock()

	log.Info().
		Str("ucode", ucode).
		Str("remote_addr", conn.RemoteAddr().String()).
		Msg("Robot registered successfully")

	// 发送注册成功响应
	registerResponse := models.RegisterResponse{
		Success:    true,
		UCode:      ucode,
		ClientType: models.ClientTypeRobot,
		Message:    fmt.Sprintf("Successfully registered as robot with UCODE %s", ucode),
		Timestamp:  time.Now().UnixMilli(),
	}

	response := models.WebSocketMessage{
		Type:    "register_response",
		Message: registerResponse.Message,
		Data:    registerResponse,
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Error().Err(err).Msg("Failed to send robot register response")
		return
	}

	// 处理后续消息
	h.handleMessages(conn, models.ClientTypeRobot)
}

// 处理操作者注册
func (h *WebSocketHandlers) handleOperatorRegistration(conn *websocket.Conn, rawMsg map[string]interface{}) {
	// 获取操作者ID
	operatorId, ok := rawMsg["operator_id"].(string)
	if !ok || operatorId == "" {
		h.sendRegisterError(conn, "Invalid operator_id for operator")
		return
	}

	// 获取机器人UCode（操作者要控制的机器人）
	robotUCode, ok := rawMsg["robot_ucode"].(string)
	if !ok || robotUCode == "" {
		h.sendRegisterError(conn, "Invalid robot_ucode for operator")
		return
	}

	// 检查机器人是否在线
	h.mutex.RLock()
	if _, exists := h.robotUcodeConn[robotUCode]; !exists {
		h.mutex.RUnlock()
		h.sendRegisterError(conn, "Target robot not online")
		return
	}
	h.mutex.RUnlock()

	// 获取可选字段
	name, _ := rawMsg["name"].(string)
	version, _ := rawMsg["version"].(string)

	// 检查操作者ID是否已被使用
	h.mutex.Lock()
	if _, exists := h.operatorIdConn[operatorId]; exists {
		h.mutex.Unlock()
		h.sendRegisterError(conn, "Operator ID already in use")
		return
	}

	// 创建操作者连接信息
	operatorConn := &models.OperatorConnection{
		OperatorID: operatorId,
		RobotUCode: robotUCode,
		Name:       name,
		Version:    version,
		Connected:  true,
		LastSeen:   time.Now(),
		RemoteAddr: conn.RemoteAddr().String(),
	}

	// 绑定连接
	h.operatorIdConn[operatorId] = conn
	h.operatorConnId[conn] = operatorId
	h.operatorClients[conn] = operatorConn
	h.mutex.Unlock()

	log.Info().
		Str("operator_id", operatorId).
		Str("robot_ucode", robotUCode).
		Str("remote_addr", conn.RemoteAddr().String()).
		Msg("Operator registered")

	// 发送注册成功响应
	registerResponse := models.RegisterResponse{
		Success:    true,
		UCode:      robotUCode, // 返回要控制的机器人UCode
		ClientType: models.ClientTypeOperator,
		Message:    fmt.Sprintf("Successfully registered as operator %s for robot %s", operatorId, robotUCode),
		Timestamp:  time.Now().UnixMilli(),
	}

	response := models.WebSocketMessage{
		Type:    "register_response",
		Message: registerResponse.Message,
		Data:    registerResponse,
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Error().Err(err).Msg("Failed to send operator register response")
		return
	}

	// 处理后续消息
	h.handleMessages(conn, models.ClientTypeOperator)
}

// 处理后续消息
func (h *WebSocketHandlers) handleMessages(conn *websocket.Conn, clientType models.ClientType) {
	for {
		var msg models.WebSocketMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Msg("WebSocket read error")
			}
			break
		}
		h.handleMessage(conn, msg, clientType)
	}

	// 清理连接
	h.cleanupConnection(conn, clientType)
}

// 清理连接
func (h *WebSocketHandlers) cleanupConnection(conn *websocket.Conn, clientType models.ClientType) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	switch clientType {
	case models.ClientTypeRobot:
		if ucode, ok := h.robotConnUcode[conn]; ok {
			delete(h.robotUcodeConn, ucode)
			delete(h.robotConnUcode, conn)
			delete(h.robotClients, conn)

			log.Info().
				Str("ucode", ucode).
				Str("remote_addr", conn.RemoteAddr().String()).
				Msg("Robot disconnected")
		}
	case models.ClientTypeOperator:
		if operatorId, ok := h.operatorConnId[conn]; ok {
			delete(h.operatorIdConn, operatorId)
			delete(h.operatorConnId, conn)
			delete(h.operatorClients, conn)
			log.Info().
				Str("operator_id", operatorId).
				Str("remote_addr", conn.RemoteAddr().String()).
				Msg("Operator disconnected")
		}
	}
}

// 发送注册错误消息
func (h *WebSocketHandlers) sendRegisterError(conn *websocket.Conn, message string) {
	registerResponse := models.RegisterResponse{
		Success:   false,
		Message:   message,
		Error:     message,
		Timestamp: time.Now().UnixMilli(),
	}

	response := models.WebSocketMessage{
		Type:    "register_response",
		Message: message,
		Data:    registerResponse,
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Error().Err(err).Msg("Failed to send register error")
	}
}

// 处理WebSocket消息
func (h *WebSocketHandlers) handleMessage(conn *websocket.Conn, msg models.WebSocketMessage, clientType models.ClientType) {
	log.Debug().
		Str("type", msg.Type).
		Str("client_type", string(clientType)).
		Str("client", conn.RemoteAddr().String()).
		Msg("Received WebSocket message")

	switch msg.Type {
	case "control_command":
		h.handleControlCommand(conn, msg, clientType)
	case "status_request":
		h.handleStatusRequest(conn, msg)
	case "ping":
		h.handlePing(conn, msg)
	default:
		h.sendError(conn, "unknown_message_type", "Unknown message type: "+msg.Type)
	}
}

// 处理控制命令
func (h *WebSocketHandlers) handleControlCommand(conn *websocket.Conn, msg models.WebSocketMessage, clientType models.ClientType) {
	// 只允许操作者发送控制命令
	if clientType != models.ClientTypeOperator {
		h.sendError(conn, "permission_denied", "Only operators can send control commands")
		return
	}

	// 获取操作者信息
	h.mutex.RLock()
	operatorConn, exists := h.operatorClients[conn]
	h.mutex.RUnlock()

	if !exists {
		h.sendError(conn, "client_not_found", "Operator not found")
		return
	}

	// 解析控制命令
	commandData, ok := msg.Data.(map[string]interface{})
	if !ok {
		h.sendError(conn, "invalid_command_format", "Command data must be an object")
		return
	}

	// 转换为ControlCommand
	commandJSON, err := json.Marshal(commandData)
	if err != nil {
		h.sendError(conn, "command_serialization_error", "Failed to serialize command")
		return
	}

	var command models.ControlCommand
	if err := json.Unmarshal(commandJSON, &command); err != nil {
		h.sendError(conn, "command_parsing_error", "Failed to parse command: "+err.Error())
		return
	}

	// 获取机器人连接
	h.mutex.RLock()
	robotConn, exists := h.robotUcodeConn[operatorConn.RobotUCode]
	h.mutex.RUnlock()

	if !exists {
		h.sendError(conn, "robot_not_found", "Target robot not connected")
		return
	}

	// 创建命令消息
	commandMessage := models.WebSocketMessage{
		Type:    "control_command",
		Message: "Control command from operator",
		Data:    command,
	}

	// 发送命令到机器人
	if err := robotConn.WriteJSON(commandMessage); err != nil {
		h.sendError(conn, "command_send_error", "Failed to send command to robot: "+err.Error())
		return
	}

	// 发送成功响应
	response := models.WebSocketMessage{
		Type:    "control_response",
		Message: "Command sent successfully",
		Data: map[string]interface{}{
			"command_id":  command.CommandID,
			"timestamp":   time.Now().UnixMilli(),
			"operator_id": operatorConn.OperatorID,
			"robot_ucode": operatorConn.RobotUCode,
		},
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Error().Err(err).Msg("Failed to send command response")
	}
}

// 处理状态请求
func (h *WebSocketHandlers) handleStatusRequest(conn *websocket.Conn, msg models.WebSocketMessage) {
	// 获取客户端统计信息
	robotCount := len(h.robotClients)
	operatorCount := len(h.operatorClients)

	// 构建机器人状态（基于WebSocket连接状态）
	robotStatus := models.ConnectionStatus{
		Connected:       robotCount > 0,
		LastHeartbeat:   time.Now(),
		Latency:         0, // WebSocket连接不计算延迟
		ActiveClients:   robotCount + operatorCount,
		TotalCommands:   0, // 这些统计可以在后续实现
		FailedCommands:  0,
		LastCommandTime: time.Now(),
	}

	// 构建详细状态信息
	statusData := map[string]interface{}{
		"robot_status": robotStatus,
		"clients": map[string]interface{}{
			"total":             robotCount + operatorCount,
			"robots":            robotCount,
			"operators":         operatorCount,
			"robots_details":    h.GetAllRobotConnections(),
			"operators_details": h.GetAllOperatorConnections(),
		},
		"timestamp": time.Now().UnixMilli(),
	}

	response := models.WebSocketMessage{
		Type:    "status_response",
		Message: "System status retrieved",
		Data:    statusData,
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Error().Err(err).Msg("Failed to send status response")
	}
}

// 处理ping消息
func (h *WebSocketHandlers) handlePing(conn *websocket.Conn, msg models.WebSocketMessage) {
	response := models.WebSocketMessage{
		Type:    "pong",
		Message: "Pong",
		Data: map[string]interface{}{
			"timestamp": time.Now().UnixMilli(),
		},
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Error().Err(err).Msg("Failed to send pong response")
	}
}

// 发送错误消息
func (h *WebSocketHandlers) sendError(conn *websocket.Conn, errorType, message string) {
	errorMsg := models.WebSocketMessage{
		Type:    "error",
		Message: message,
		Data: map[string]interface{}{
			"error_type": errorType,
			"timestamp":  time.Now().UnixMilli(),
		},
	}

	if err := conn.WriteJSON(errorMsg); err != nil {
		log.Error().Err(err).Msg("Failed to send error message")
	}
}

// 广播消息给所有客户端
func (h *WebSocketHandlers) Broadcast(message models.WebSocketMessage) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// 广播给所有机器人
	for conn := range h.robotClients {
		if err := conn.WriteJSON(message); err != nil {
			log.Error().Err(err).Msg("Failed to broadcast message to robot")
			conn.Close()
		}
	}

	// 广播给所有操作者
	for conn := range h.operatorClients {
		if err := conn.WriteJSON(message); err != nil {
			log.Error().Err(err).Msg("Failed to broadcast message to operator")
			conn.Close()
		}
	}
}

// 广播消息给指定类型的客户端
func (h *WebSocketHandlers) BroadcastToType(message models.WebSocketMessage, clientType models.ClientType) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	switch clientType {
	case models.ClientTypeRobot:
		for conn := range h.robotClients {
			if err := conn.WriteJSON(message); err != nil {
				log.Error().Err(err).Msg("Failed to broadcast message to robot")
				conn.Close()
			}
		}
	case models.ClientTypeOperator:
		for conn := range h.operatorClients {
			if err := conn.WriteJSON(message); err != nil {
				log.Error().Err(err).Msg("Failed to broadcast message to operator")
				conn.Close()
			}
		}
	}
}

// 发送消息给指定UCode的机器人
func (h *WebSocketHandlers) SendToRobotUCode(message models.WebSocketMessage, ucode string) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if conn, exists := h.robotUcodeConn[ucode]; exists {
		if err := conn.WriteJSON(message); err != nil {
			log.Error().Err(err).Str("ucode", ucode).Msg("Failed to send message to robot")
			return false
		}
		return true
	}
	return false
}

// 发送消息给指定ID的操作者
func (h *WebSocketHandlers) SendToOperatorID(message models.WebSocketMessage, operatorId string) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if conn, exists := h.operatorIdConn[operatorId]; exists {
		if err := conn.WriteJSON(message); err != nil {
			log.Error().Err(err).Str("operator_id", operatorId).Msg("Failed to send message to operator")
			return false
		}
		return true
	}
	return false
}

// 获取活跃客户端数量
func (h *WebSocketHandlers) GetActiveClientsCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.robotClients) + len(h.operatorClients)
}

// 获取所有机器人连接信息
func (h *WebSocketHandlers) GetAllRobotConnections() []*models.RobotConnection {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	robots := make([]*models.RobotConnection, 0, len(h.robotClients))
	for _, robotConn := range h.robotClients {
		robots = append(robots, robotConn)
	}
	return robots
}

// 获取所有操作者连接信息
func (h *WebSocketHandlers) GetAllOperatorConnections() []*models.OperatorConnection {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	operators := make([]*models.OperatorConnection, 0, len(h.operatorClients))
	for _, operatorConn := range h.operatorClients {
		operators = append(operators, operatorConn)
	}
	return operators
}

// 获取机器人数量
func (h *WebSocketHandlers) GetRobotCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.robotClients)
}

// 获取操作者数量
func (h *WebSocketHandlers) GetOperatorCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.operatorClients)
}

// 获取指定UCode的机器人连接信息
func (h *WebSocketHandlers) GetRobotByUCode(ucode string) *models.RobotConnection {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if conn, exists := h.robotUcodeConn[ucode]; exists {
		if robotConn, exists := h.robotClients[conn]; exists {
			return robotConn
		}
	}
	return nil
}

// 获取指定ID的操作者连接信息
func (h *WebSocketHandlers) GetOperatorByID(operatorId string) *models.OperatorConnection {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if conn, exists := h.operatorIdConn[operatorId]; exists {
		if operatorConn, exists := h.operatorClients[conn]; exists {
			return operatorConn
		}
	}
	return nil
}

// 检查机器人UCode是否在线
func (h *WebSocketHandlers) IsRobotUCodeOnline(ucode string) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	_, exists := h.robotUcodeConn[ucode]
	return exists
}

// 检查操作者ID是否在线
func (h *WebSocketHandlers) IsOperatorIDOnline(operatorId string) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	_, exists := h.operatorIdConn[operatorId]
	return exists
}

// 兼容性方法 - 为了保持向后兼容
func (h *WebSocketHandlers) GetAllClients() []interface{} {
	// 这个方法现在返回空，因为我们已经分离了机器人和操作者
	return []interface{}{}
}

func (h *WebSocketHandlers) GetClientsCountByType(clientType models.ClientType) int {
	switch clientType {
	case models.ClientTypeRobot:
		return h.GetRobotCount()
	case models.ClientTypeOperator:
		return h.GetOperatorCount()
	default:
		return 0
	}
}

func (h *WebSocketHandlers) GetClientByUCode(ucode string) interface{} {
	// 这个方法现在返回nil，因为我们已经分离了机器人和操作者
	return nil
}

func (h *WebSocketHandlers) IsUCodeOnline(ucode string) bool {
	return h.IsRobotUCodeOnline(ucode)
}

func (h *WebSocketHandlers) SendToUCode(message models.WebSocketMessage, ucode string) bool {
	return h.SendToRobotUCode(message, ucode)
}
