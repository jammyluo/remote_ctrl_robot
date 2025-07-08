package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"remote-ctrl-robot/internal/models"
	"remote-ctrl-robot/internal/services"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type WebSocketHandlers struct {
	upgrader websocket.Upgrader

	// 机器人连接管理
	Conn2Client map[*websocket.Conn]*models.Client
	Ucode2Conn  map[string]*websocket.Conn

	// 机器人状态
	RobotStatus map[string]models.RobotState

	// 绑定关系
	Operator2Robot map[string]string
	Robot2Operator map[string]string

	// 连接管理
	ctx    context.Context
	cancel context.CancelFunc
	mutex  sync.RWMutex

	robotService *services.RobotService
	gameService  *services.GameService
}

func NewWebSocketHandlers(robotService *services.RobotService, gameService *services.GameService) *WebSocketHandlers {
	ctx, cancel := context.WithCancel(context.Background())
	return &WebSocketHandlers{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		Conn2Client:    make(map[*websocket.Conn]*models.Client),
		Ucode2Conn:     make(map[string]*websocket.Conn),
		RobotStatus:    make(map[string]models.RobotState),
		Operator2Robot: make(map[string]string),
		Robot2Operator: make(map[string]string),
		ctx:            ctx,
		cancel:         cancel,
		robotService:   robotService,
		gameService:    gameService,
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
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if conn, exists := h.Ucode2Conn[ucode]; exists {
		return h.Conn2Client[conn]
	}
	return nil
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
	// 检查UCode是否已被使用
	h.mutex.Lock()
	if _, exists := h.Ucode2Conn[msg.UCode]; exists {
		h.mutex.Unlock()
		h.sendResponseError(conn, msg, "Robot UCODE already in use")
		return false
	}

	// 创建Client连接信息
	client := &models.Client{
		UCode:      msg.UCode,
		ClientType: msg.ClientType,
		Version:    msg.Version,
		Connected:  true,
		LastSeen:   time.Now(),
		RemoteAddr: conn.RemoteAddr().String(),
	}

	// 绑定连接
	h.Ucode2Conn[msg.UCode] = conn
	h.Conn2Client[conn] = client
	h.mutex.Unlock()

	log.Info().
		Str("ucode", msg.UCode).
		Str("client_type", string(msg.ClientType)).
		Str("sequence", strconv.FormatInt(msg.Sequence, 10)).
		Str("version", msg.Version).
		Str("remote_addr", conn.RemoteAddr().String()).
		Msg("Robot registered successfully")

	// 发送注册成功响应
	h.sendResponse(conn, msg, fmt.Sprintf("Successfully registered with UCODE %s", msg.UCode))
	return true
}

// 消息处理循环 - 带超时和心跳
func (h *WebSocketHandlers) handleMessagesWithTimeout(conn *websocket.Conn) {
	defer func() {
		h.cleanupConnection(conn)
		conn.Close()
	}()

	// 启动心跳 goroutine
	heartbeatTicker := time.NewTicker(25 * time.Second) // 25秒发送一次ping
	defer heartbeatTicker.Stop()

	// 启动心跳处理
	go func() {
		for {
			select {
			case <-heartbeatTicker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					log.Error().Err(err).Msg("Failed to send ping")
					return
				}
			case <-h.ctx.Done():
				return
			}
		}
	}()

	// 主消息处理循环
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
				} else {
					log.Info().Msg("WebSocket connection closed normally")
				}
				return
			}

			// 重置读取超时
			conn.SetReadDeadline(time.Now().Add(30 * time.Second))

			if err := h.checkWSMessage(msg); err != nil {
				h.sendResponseError(conn, &msg, err.Error())
				conn.Close()
				continue
			}
			// 处理消息
			h.handleMessage(conn, &msg)
		}
	}
}

// 连接清理
func (h *WebSocketHandlers) cleanupConnection(conn *websocket.Conn) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if client, ok := h.Conn2Client[conn]; ok {
		// 清理游戏连接
		if h.gameService != nil {
			h.gameService.RemoveRobotConnection(conn)
		}

		delete(h.Ucode2Conn, client.UCode)
		delete(h.Conn2Client, conn)
		delete(h.Operator2Robot, client.UCode)
		delete(h.Robot2Operator, client.UCode)
		delete(h.RobotStatus, client.UCode)

		log.Info().
			Str("ucode", client.UCode).
			Str("client_type", string(client.ClientType)).
			Str("remote_addr", conn.RemoteAddr().String()).
			Msg("Client disconnected")
	}
}

// 关闭所有连接
func (h *WebSocketHandlers) Shutdown() {
	h.cancel()

	h.mutex.Lock()
	defer h.mutex.Unlock()

	// 关闭所有机器人连接
	for conn := range h.Conn2Client {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "Server shutdown"))
		conn.Close()
	}

	// 关闭所有操作者连接
	for conn := range h.Conn2Client {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "Server shutdown"))
		conn.Close()
	}
	log.Info().Msg("All connections closed")
	// 清空映射
	h.Conn2Client = make(map[*websocket.Conn]*models.Client)
	h.Ucode2Conn = make(map[string]*websocket.Conn)
	h.RobotStatus = make(map[string]models.RobotState)
	h.Operator2Robot = make(map[string]string)
	h.Robot2Operator = make(map[string]string)
}

// 处理WebSocket消息
func (h *WebSocketHandlers) handleMessage(conn *websocket.Conn, msg *models.WebSocketMessage) {
	// 解析命令数据
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		h.sendResponseError(conn, msg, "Command data must be an object")
		return
	}

	// 转换为JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		h.sendResponseError(conn, msg, "Failed to serialize command")
		return
	}

	log.Debug().
		Str("command", string(msg.Command)).
		Str("ucode", msg.UCode).
		Str("client_type", string(msg.ClientType)).
		Str("version", msg.Version).
		Str("sequence", strconv.FormatInt(msg.Sequence, 10)).
		Str("data", string(dataJSON)).
		Str("client", conn.RemoteAddr().String()).
		Msg("Received WebSocket message")

	err = errors.New("Unknown message type: " + string(msg.Command))
	switch msg.Command {
	case models.CMD_TYPE_BIND_ROBOT:
		err = h.handleBindRobot(conn, dataJSON)
	case models.CMD_TYPE_CONTROL_ROBOT:
		err = h.handleControlRobot(conn, dataJSON)
	case models.CMD_TYPE_UPDATE_ROBOT_STATUS:
		err = h.handleUpdateRobotStatus(conn, dataJSON)
	case models.CMD_TYPE_PING:
		err = h.handlePing(conn, dataJSON)
	// 游戏相关命令
	case models.CMD_TYPE_JOIN_GAME:
		err = h.handleJoinGame(conn, dataJSON)
	case models.CMD_TYPE_LEAVE_GAME:
		err = h.handleLeaveGame(conn, dataJSON)
	case models.CMD_TYPE_GAME_SHOOT:
		err = h.handleGameShoot(conn, dataJSON)
	case models.CMD_TYPE_GAME_MOVE:
		err = h.handleGameMove(conn, dataJSON)
	case models.CMD_TYPE_GAME_STATUS:
		err = h.handleGameStatus(conn, dataJSON)
	case models.CMD_TYPE_GAME_START:
		err = h.handleGameStart(conn, dataJSON)
	case models.CMD_TYPE_GAME_STOP:
		err = h.handleGameStop(conn, dataJSON)
	}

	if err != nil {
		h.sendResponseError(conn, msg, err.Error())
		log.Error().Err(err).Msg("Failed to handle message " + string(msg.Command) + " " + err.Error())
	} else {
		h.sendResponse(conn, msg, "Success")
	}
}

// 处理绑定机器人
func (h *WebSocketHandlers) handleBindRobot(conn *websocket.Conn, dataJSON []byte) error {
	// 获取操作者信息
	h.mutex.RLock()
	client, exists := h.Conn2Client[conn]
	h.mutex.RUnlock()

	if !exists {
		return errors.New("client not found")
	}

	if client.ClientType != models.ClientTypeOperator {
		return errors.New("client is not an operator")
	}

	var data models.CMD_BIND_ROBOT
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return errors.New("failed to parse command: " + err.Error())
	}

	// 绑定机器人
	clientRobot := h.GetClientByUcode(data.UCode)
	if clientRobot == nil || clientRobot.ClientType != models.ClientTypeRobot {
		return errors.New("target robot not connected")
	}

	if _, exists := h.Robot2Operator[client.UCode]; exists {
		return errors.New("robot already bound to another operator")
	}
	if _, exists := h.Operator2Robot[client.UCode]; exists {
		return errors.New("operator already bound to another robot")
	}

	// 绑定成功
	h.Operator2Robot[client.UCode] = clientRobot.UCode
	h.Robot2Operator[clientRobot.UCode] = client.UCode

	return nil
}

// 处理控制命令
func (h *WebSocketHandlers) handleControlRobot(conn *websocket.Conn, dataJSON []byte) error {
	// 获取操作者信息
	h.mutex.RLock()
	client, exists := h.Conn2Client[conn]
	h.mutex.RUnlock()

	if !exists {
		return errors.New("client not found")
	}

	robotUcode := h.Operator2Robot[client.UCode]
	if robotUcode == "" {
		return errors.New("robot not bound to operator")
	}

	// 获取机器人连接
	h.mutex.RLock()
	robotConn, exists := h.Ucode2Conn[robotUcode]
	h.mutex.RUnlock()

	if !exists {
		return errors.New("target robot not connected")
	}

	var data models.CMD_CONTROL_ROBOT
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return errors.New("failed to parse command: " + err.Error())
	}

	// 创建命令消息
	commandMessage := models.WebSocketMessage{
		Type:       models.WSMessageTypeRequest,
		Command:    models.CMD_TYPE_CONTROL_ROBOT,
		Sequence:   1,
		UCode:      client.UCode,
		ClientType: client.ClientType,
		Version:    client.Version,
		Data:       data,
	}

	// 发送命令到机器人
	if err := robotConn.WriteJSON(commandMessage); err != nil {
		return errors.New("failed to send command to robot: " + err.Error())
	}
	return nil
}

// 处理状态请求
func (h *WebSocketHandlers) handleUpdateRobotStatus(conn *websocket.Conn, data []byte) error {

	h.mutex.RLock()
	client, exists := h.Conn2Client[conn]
	h.mutex.RUnlock()

	if !exists {
		return errors.New("client not found")
	}

	var status models.RobotState
	if err := json.Unmarshal(data, &status); err != nil {
		return errors.New("failed to parse command: " + err.Error())
	}

	h.RobotStatus[client.UCode] = status

	return nil
}

// 处理ping消息
func (h *WebSocketHandlers) handlePing(conn *websocket.Conn, commandJSON []byte) error {
	h.mutex.RLock()
	client, exists := h.Conn2Client[conn]
	h.mutex.RUnlock()

	if !exists {
		return errors.New("client not found")
	}

	client.LastSeen = time.Now()
	return nil
}

// 获取所有机器人连接信息
func (h *WebSocketHandlers) GetAllRobotConnections() []*models.Client {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	robots := make([]*models.Client, 0, len(h.Conn2Client))
	for _, client := range h.Conn2Client {
		if client.ClientType == models.ClientTypeRobot {
			robots = append(robots, client)
		}
	}
	return robots
}

// 获取所有操作者连接信息
func (h *WebSocketHandlers) GetAllOperatorConnections() []*models.Client {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	operators := make([]*models.Client, 0, len(h.Conn2Client))
	for _, client := range h.Conn2Client {
		if client.ClientType == models.ClientTypeOperator {
			operators = append(operators, client)
		}
	}
	return operators
}

// 游戏相关处理方法

// 处理加入游戏
func (h *WebSocketHandlers) handleJoinGame(conn *websocket.Conn, dataJSON []byte) error {
	h.mutex.RLock()
	client, exists := h.Conn2Client[conn]
	h.mutex.RUnlock()

	if !exists {
		return errors.New("client not found")
	}

	if client.ClientType != models.ClientTypeRobot {
		return errors.New("only robots can join games")
	}

	var data models.CMD_JOIN_GAME
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return errors.New("failed to parse command: " + err.Error())
	}

	// 如果游戏不存在，创建新游戏
	if _, err := h.gameService.GetGameState(data.GameID); err != nil {
		h.gameService.CreateGame(data.GameID)
	}

	// 加入游戏
	return h.gameService.JoinGame(data.GameID, client.UCode, data.Name, conn)
}

// 处理离开游戏
func (h *WebSocketHandlers) handleLeaveGame(conn *websocket.Conn, dataJSON []byte) error {
	h.mutex.RLock()
	client, exists := h.Conn2Client[conn]
	h.mutex.RUnlock()

	if !exists {
		return errors.New("client not found")
	}

	if client.ClientType != models.ClientTypeRobot {
		return errors.New("only robots can leave games")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return errors.New("failed to parse command: " + err.Error())
	}

	gameID, ok := data["game_id"].(string)
	if !ok {
		return errors.New("game_id is required")
	}

	return h.gameService.LeaveGame(gameID, client.UCode)
}

// 处理游戏射击
func (h *WebSocketHandlers) handleGameShoot(conn *websocket.Conn, dataJSON []byte) error {
	h.mutex.RLock()
	client, exists := h.Conn2Client[conn]
	h.mutex.RUnlock()

	if !exists {
		return errors.New("client not found")
	}

	if client.ClientType != models.ClientTypeRobot {
		return errors.New("only robots can shoot")
	}

	var data models.CMD_GAME_SHOOT
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return errors.New("failed to parse command: " + err.Error())
	}

	// 需要从消息中获取游戏ID，这里简化处理，假设只有一个活跃游戏
	// 在实际应用中，应该从消息中获取游戏ID
	gameID := "default_game" // 简化处理

	return h.gameService.ProcessShot(gameID, client.UCode, data.TargetX, data.TargetY, data.TargetZ)
}

// 处理游戏移动
func (h *WebSocketHandlers) handleGameMove(conn *websocket.Conn, dataJSON []byte) error {
	h.mutex.RLock()
	client, exists := h.Conn2Client[conn]
	h.mutex.RUnlock()

	if !exists {
		return errors.New("client not found")
	}

	if client.ClientType != models.ClientTypeRobot {
		return errors.New("only robots can move")
	}

	var data models.CMD_GAME_MOVE
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return errors.New("failed to parse command: " + err.Error())
	}

	// 简化处理，假设只有一个活跃游戏
	gameID := "default_game"

	return h.gameService.ProcessMove(gameID, client.UCode, data.Position, data.Direction)
}

// 处理游戏状态请求
func (h *WebSocketHandlers) handleGameStatus(conn *websocket.Conn, dataJSON []byte) error {
	h.mutex.RLock()
	client, exists := h.Conn2Client[conn]
	h.mutex.RUnlock()

	if !exists {
		return errors.New("client not found")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return errors.New("failed to parse command: " + err.Error())
	}

	gameID, ok := data["game_id"].(string)
	if !ok {
		return errors.New("game_id is required")
	}

	gameState, err := h.gameService.GetGameState(gameID)
	if err != nil {
		return err
	}

	myRobot, err := h.gameService.GetRobotInGame(gameID, client.UCode)
	if err != nil {
		return err
	}

	// 发送游戏状态响应
	response := models.WebSocketMessage{
		Type:     models.WSMessageTypeResponse,
		Command:  models.CMD_TYPE_GAME_STATUS,
		Sequence: time.Now().UnixNano(),
		UCode:    client.UCode,
		Data: models.CMD_GAME_STATUS_RESPONSE{
			GameState: gameState,
			MyRobot:   myRobot,
		},
	}

	return conn.WriteJSON(response)
}

// 处理开始游戏
func (h *WebSocketHandlers) handleGameStart(conn *websocket.Conn, dataJSON []byte) error {
	h.mutex.RLock()
	client, exists := h.Conn2Client[conn]
	h.mutex.RUnlock()

	if !exists {
		return errors.New("client not found")
	}

	if client.ClientType != models.ClientTypeOperator {
		return errors.New("only operators can start games")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return errors.New("failed to parse command: " + err.Error())
	}

	gameID, ok := data["game_id"].(string)
	if !ok {
		return errors.New("game_id is required")
	}

	return h.gameService.StartGame(gameID)
}

// 处理停止游戏
func (h *WebSocketHandlers) handleGameStop(conn *websocket.Conn, dataJSON []byte) error {
	h.mutex.RLock()
	client, exists := h.Conn2Client[conn]
	h.mutex.RUnlock()

	if !exists {
		return errors.New("client not found")
	}

	if client.ClientType != models.ClientTypeOperator {
		return errors.New("only operators can stop games")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return errors.New("failed to parse command: " + err.Error())
	}

	gameID, ok := data["game_id"].(string)
	if !ok {
		return errors.New("game_id is required")
	}

	return h.gameService.StopGame(gameID)
}
