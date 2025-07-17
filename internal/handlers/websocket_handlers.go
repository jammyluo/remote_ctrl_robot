package handlers

import (
	"context"
	"errors"
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
	upgrader websocket.Upgrader

	// 连接管理
	ctx    context.Context
	cancel context.CancelFunc
	mutex  sync.RWMutex

	robotManager  *services.RobotManager
	clientManager *services.ClientManager
	gameService   *services.GameService
}

func NewWebSocketHandlers(robotManager *services.RobotManager, clientManager *services.ClientManager, gameService *services.GameService) *WebSocketHandlers {
	ctx, cancel := context.WithCancel(context.Background())
	return &WebSocketHandlers{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		ctx:           ctx,
		cancel:        cancel,
		robotManager:  robotManager,
		clientManager: clientManager,
		gameService:   gameService,
	}
}

func (h *WebSocketHandlers) sendResponseError(conn *websocket.Conn, msg *models.WebSocketMessage, message string) {
	cmdResponse := models.CMD_RESPONSE{
		Success:   false,
		Message:   message,
		Timestamp: time.Now().UnixMilli(),
	}

	response := models.WebSocketMessage{
		Type:     models.WSMessageTypeResponse,
		Command:  msg.Command,
		Sequence: msg.Sequence,
		Data:     cmdResponse,
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Error().Err(err).Msg("Failed to send register error")
	}
}

func (h *WebSocketHandlers) sendResponse(conn *websocket.Conn, msg *models.WebSocketMessage, message string) {
	cmdResponse := models.CMD_RESPONSE{
		Success:   true,
		Message:   message,
		Timestamp: time.Now().UnixMilli(),
	}

	response := models.WebSocketMessage{
		Type:     models.WSMessageTypeResponse,
		Command:  msg.Command,
		Sequence: msg.Sequence,
		Data:     cmdResponse,
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Error().Err(err).Msg("Failed to send success message")
	}
}

func (h *WebSocketHandlers) GetClientByUcode(ucode string) *models.Client {
	client, err := h.clientManager.GetClient(ucode)
	if err != nil {
		return nil
	}
	return client
}

func (h *WebSocketHandlers) checkWSMessage(msg models.WebSocketMessage) error {
	// 检查消息类型
	if msg.Type != models.WSMessageTypeRequest {
		return errors.New("invalid message type")
	}

	// 获取UCode
	if msg.UCode == "" {
		return errors.New("invalid ucode")
	}

	// 获取客户端类型
	if msg.ClientType == "" {
		return errors.New("invalid client type")
	}

	// 验证客户端类型
	if msg.ClientType != models.ClientTypeRobot && msg.ClientType != models.ClientTypeOperator {
		return errors.New("invalid client type")
	}

	if msg.Sequence == 0 {
		return errors.New("invalid sequence")
	}

	if msg.Version == "" {
		return errors.New("invalid version")
	}
	return nil
}

// 处理WebSocket连接
func (h *WebSocketHandlers) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("WebSocket connection received")

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade connection to WebSocket")
		return
	}

	// 设置连接参数
	conn.SetReadLimit(512 * 1024)                          // 512KB 读取限制
	conn.SetReadDeadline(time.Now().Add(30 * time.Second)) // 30秒读取超时
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		return nil
	})

	// 注册：第一条消息必须为register
	var msg models.WebSocketMessage
	if err := conn.ReadJSON(&msg); err != nil {
		h.sendResponseError(conn, &msg, "Failed to parse registration message")
		conn.Close()
		return
	}
	if msg.Command != models.CMD_TYPE_REGISTER {
		h.sendResponseError(conn, &msg, "Invalid message type")
		conn.Close()
		return
	}

	err = h.checkWSMessage(msg)
	if err != nil {
		h.sendResponseError(conn, &msg, err.Error())
		conn.Close()
		return
	}
	if h.handleRegistration(conn, &msg) {
		// 使用 goroutine 异步处理消息
		go h.handleMessagesWithTimeout(conn)
	}
}

// 注册 - 返回是否成功
func (h *WebSocketHandlers) handleRegistration(conn *websocket.Conn, msg *models.WebSocketMessage) bool {
	// 创建Client连接信息
	client := &models.Client{
		UCode:      msg.UCode,
		ClientType: msg.ClientType,
		Version:    msg.Version,
		Connected:  true,
		LastSeen:   time.Now(),
		RemoteAddr: conn.RemoteAddr().String(),
	}

	// 如果是机器人客户端，注册到机器人管理器
	if msg.ClientType == models.ClientTypeRobot {
		registration := &models.RobotRegistration{
			UCode:        msg.UCode,
			Name:         fmt.Sprintf("Robot_%s", msg.UCode),
			Type:         models.RobotTypeB2, // 默认类型，可以从消息中获取
			Version:      msg.Version,
			IPAddress:    conn.RemoteAddr().String(),
			Port:         8080,
			Capabilities: []string{"move", "stop", "reset"},
		}

		robot, err := h.robotManager.RegisterRobot(registration)
		if err != nil {
			log.Error().Err(err).Str("ucode", msg.UCode).Msg("Failed to register robot")
			h.sendResponseError(conn, msg, "Failed to register robot")
			return false
		}

		// 设置机器人连接
		if robot != nil && robot.GetService() != nil {
			robot.GetService().SetConnection(conn)
		}
	}

	// 添加到客户端管理器
	if err := h.clientManager.AddClient(client, conn); err != nil {
		log.Error().Err(err).Str("ucode", msg.UCode).Msg("Failed to add client")
		h.sendResponseError(conn, msg, "Failed to add client")
		return false
	}

	log.Info().
		Str("ucode", msg.UCode).
		Str("type", string(msg.ClientType)).
		Str("version", msg.Version).
		Msg("Client registered successfully")

	h.sendResponse(conn, msg, "Registration successful")
	return true
}

// 处理消息（带超时）
func (h *WebSocketHandlers) handleMessagesWithTimeout(conn *websocket.Conn) {
	defer func() {
		h.cleanupConnection(conn)
		conn.Close()
	}()

	for {
		select {
		case <-h.ctx.Done():
			return
		default:
			// 设置读取超时
			conn.SetReadDeadline(time.Now().Add(30 * time.Second))

			var msg models.WebSocketMessage
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Error().Err(err).Msg("WebSocket read error")
				}
				return
			}

			// 处理消息
			if err := h.handleMessage(conn, &msg); err != nil {
				log.Error().Err(err).Str("ucode", msg.UCode).Msg("Failed to handle message")
				h.sendResponseError(conn, &msg, err.Error())
			}
		}
	}
}

// 清理连接
func (h *WebSocketHandlers) cleanupConnection(conn *websocket.Conn) {
	// 查找对应的客户端
	clients := h.clientManager.GetAllClients()
	for _, client := range clients {
		if client.RemoteAddr == conn.RemoteAddr().String() {
			h.clientManager.RemoveClient(client.UCode)
			break
		}
	}
}

// 关闭处理器
func (h *WebSocketHandlers) Shutdown() {
	h.cancel()
	log.Info().Msg("WebSocket handlers shutdown")
}

// 处理消息
func (h *WebSocketHandlers) handleMessage(conn *websocket.Conn, msg *models.WebSocketMessage) error {
	// 更新客户端最后活跃时间
	clients := h.clientManager.GetAllClients()
	for _, client := range clients {
		if client.RemoteAddr == conn.RemoteAddr().String() {
			client.LastSeen = time.Now()
			break
		}
	}

	// 根据命令类型处理消息
	switch msg.Command {
	case models.CMD_TYPE_BIND_ROBOT:
		return h.handleBindRobot(conn, msg)
	case models.CMD_TYPE_CONTROL_ROBOT:
		return h.handleControlRobot(conn, msg)
	case models.CMD_TYPE_UPDATE_ROBOT_STATUS:
		return h.handleUpdateRobotStatus(conn, msg)
	case models.CMD_TYPE_PING:
		return h.handlePing(conn, msg)
	case models.CMD_TYPE_JOIN_GAME:
		return h.handleJoinGame(conn, msg)
	case models.CMD_TYPE_LEAVE_GAME:
		return h.handleLeaveGame(conn, msg)
	case models.CMD_TYPE_GAME_SHOOT:
		return h.handleGameShoot(conn, msg)
	case models.CMD_TYPE_GAME_MOVE:
		return h.handleGameMove(conn, msg)
	case models.CMD_TYPE_GAME_STATUS:
		return h.handleGameStatus(conn, msg)
	case models.CMD_TYPE_GAME_START:
		return h.handleGameStart(conn, msg)
	case models.CMD_TYPE_GAME_STOP:
		return h.handleGameStop(conn, msg)
	default:
		log.Debug().
			Str("ucode", msg.UCode).
			Str("command", string(msg.Command)).
			Msg("Received message from client")
	}

	return nil
}

// 处理机器人绑定
func (h *WebSocketHandlers) handleBindRobot(conn *websocket.Conn, msg *models.WebSocketMessage) error {
	var bindData models.CMD_BIND_ROBOT
	if data, ok := msg.Data.(map[string]interface{}); ok {
		if robotUCode, exists := data["ucode"].(string); exists {
			bindData.UCode = robotUCode
		}
	}

	// 检查机器人是否存在
	robot, err := h.robotManager.GetRobot(bindData.UCode)
	if err != nil {
		h.sendResponseError(conn, msg, fmt.Sprintf("机器人 %s 不存在", bindData.UCode))
		return err
	}

	// 发送绑定成功响应
	response := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    models.CMD_TYPE_BIND_ROBOT,
		Sequence:   msg.Sequence,
		UCode:      msg.UCode,
		ClientType: msg.ClientType,
		Version:    msg.Version,
		Data: models.RobotBindResponse{
			Success: true,
			Message: "机器人绑定成功",
			Robot:   robot,
		},
	}

	return conn.WriteJSON(response)
}

// 处理机器人控制
func (h *WebSocketHandlers) handleControlRobot(conn *websocket.Conn, msg *models.WebSocketMessage) error {
	// 解析控制命令
	var controlData models.CMD_CONTROL_ROBOT
	if data, ok := msg.Data.(map[string]interface{}); ok {
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
		Priority:      5,
		Timestamp:     controlData.Timestamp,
		OperatorUCode: msg.UCode,
	}

	// 获取在线机器人
	robots := h.robotManager.GetOnlineRobots()
	if len(robots) == 0 {
		h.sendResponseError(conn, msg, "没有可用的在线机器人")
		return fmt.Errorf("no online robots available")
	}

	// 发送命令到第一个在线机器人
	response, err := h.robotManager.SendCommand(robots[0].UCode, command)
	if err != nil {
		h.sendResponseError(conn, msg, fmt.Sprintf("命令发送失败: %v", err))
		return err
	}

	// 发送成功响应
	wsResponse := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    models.CMD_TYPE_CONTROL_ROBOT,
		Sequence:   msg.Sequence,
		UCode:      msg.UCode,
		ClientType: msg.ClientType,
		Version:    msg.Version,
		Data:       response,
	}

	return conn.WriteJSON(wsResponse)
}

// 处理机器人状态更新
func (h *WebSocketHandlers) handleUpdateRobotStatus(conn *websocket.Conn, msg *models.WebSocketMessage) error {
	// 这里可以处理机器人状态更新
	// 暂时简单处理
	h.sendResponse(conn, msg, "状态更新成功")
	return nil
}

// 处理心跳
func (h *WebSocketHandlers) handlePing(conn *websocket.Conn, msg *models.WebSocketMessage) error {
	response := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    models.CMD_TYPE_PING,
		Sequence:   msg.Sequence,
		UCode:      msg.UCode,
		ClientType: msg.ClientType,
		Version:    msg.Version,
		Data: models.CMD_RESPONSE{
			Success:   true,
			Message:   "pong",
			Timestamp: time.Now().Unix(),
		},
	}

	return conn.WriteJSON(response)
}

// 获取所有机器人连接
func (h *WebSocketHandlers) GetAllRobotConnections() []*models.Client {
	var robotClients []*models.Client
	clients := h.clientManager.GetAllClients()

	for _, client := range clients {
		if client.ClientType == models.ClientTypeRobot {
			robotClients = append(robotClients, client)
		}
	}

	return robotClients
}

// 获取所有操作员连接
func (h *WebSocketHandlers) GetAllOperatorConnections() []*models.Client {
	var operatorClients []*models.Client
	clients := h.clientManager.GetAllClients()

	for _, client := range clients {
		if client.ClientType == models.ClientTypeOperator {
			operatorClients = append(operatorClients, client)
		}
	}

	return operatorClients
}

// 游戏相关处理方法（简化版本）
func (h *WebSocketHandlers) handleJoinGame(conn *websocket.Conn, msg *models.WebSocketMessage) error {
	h.sendResponse(conn, msg, "加入游戏成功")
	return nil
}

func (h *WebSocketHandlers) handleLeaveGame(conn *websocket.Conn, msg *models.WebSocketMessage) error {
	h.sendResponse(conn, msg, "离开游戏成功")
	return nil
}

func (h *WebSocketHandlers) handleGameShoot(conn *websocket.Conn, msg *models.WebSocketMessage) error {
	h.sendResponse(conn, msg, "射击命令发送成功")
	return nil
}

func (h *WebSocketHandlers) handleGameMove(conn *websocket.Conn, msg *models.WebSocketMessage) error {
	h.sendResponse(conn, msg, "移动命令发送成功")
	return nil
}

func (h *WebSocketHandlers) handleGameStatus(conn *websocket.Conn, msg *models.WebSocketMessage) error {
	h.sendResponse(conn, msg, "游戏状态获取成功")
	return nil
}

func (h *WebSocketHandlers) handleGameStart(conn *websocket.Conn, msg *models.WebSocketMessage) error {
	h.sendResponse(conn, msg, "游戏开始成功")
	return nil
}

func (h *WebSocketHandlers) handleGameStop(conn *websocket.Conn, msg *models.WebSocketMessage) error {
	h.sendResponse(conn, msg, "游戏停止成功")
	return nil
}
