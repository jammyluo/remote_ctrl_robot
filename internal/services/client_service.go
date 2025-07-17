package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"remote-ctrl-robot/internal/models"
	"remote-ctrl-robot/internal/utils"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// ClientService 单个客户端服务
type ClientService struct {
	client        *models.Client
	manager       *ClientManager
	robotManager  *RobotManager
	gameService   *GameService
	conn          *websocket.Conn
	mutex         sync.RWMutex
	connected     bool
	ctx           context.Context
	cancel        context.CancelFunc
	eventHandlers map[models.ClientEventType][]func(*models.ClientEvent)
	handlerMutex  sync.RWMutex
}

// NewClientService 创建新的客户端服务
func NewClientService(client *models.Client, manager *ClientManager, robotManager *RobotManager, gameService *GameService) *ClientService {
	ctx, cancel := context.WithCancel(context.Background())

	return &ClientService{
		client:        client,
		manager:       manager,
		robotManager:  robotManager,
		gameService:   gameService,
		ctx:           ctx,
		cancel:        cancel,
		eventHandlers: make(map[models.ClientEventType][]func(*models.ClientEvent)),
	}
}

// Start 启动客户端服务
func (s *ClientService) Start() error {
	log.Info().
		Str("ucode", s.client.UCode).
		Str("name", s.client.Name).
		Str("type", string(s.client.ClientType)).
		Msg("Starting client service")

	// 启动状态监控
	go s.statusMonitor()

	return nil
}

// Stop 停止客户端服务
func (s *ClientService) Stop() error {
	log.Info().
		Str("ucode", s.client.UCode).
		Msg("Stopping client service")

	s.cancel()
	s.disconnect()

	return nil
}

// SetConnection 设置客户端连接（由WebSocket处理器调用）
func (s *ClientService) SetConnection(conn *websocket.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.conn = conn
	s.connected = true

	// 更新客户端状态
	s.client.Connected = true
	s.client.LastSeen = time.Now()

	log.Info().
		Str("ucode", s.client.UCode).
		Msg("Client connected")

	s.emitEvent(models.ClientEventConnected, "客户端连接成功")

	// 启动消息处理
	go s.handleMessages()
}

// handleMessages 处理来自客户端的消息
func (s *ClientService) handleMessages() {
	defer func() {
		s.Disconnect()
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// 设置读取超时
			s.conn.SetReadDeadline(time.Now().Add(30 * time.Second))

			_, data, err := s.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Error().Err(err).Str("ucode", s.client.UCode).Msg("WebSocket read error")
				}
				return
			}

			// 处理消息
			if err := s.handleMessage(websocket.TextMessage, data); err != nil {
				log.Error().Err(err).Str("ucode", s.client.UCode).Msg("Failed to handle message")
			}
		}
	}
}

// Disconnect 断开连接
func (s *ClientService) Disconnect() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.connected {
		return nil
	}

	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}

	s.connected = false

	// 更新客户端状态
	s.client.Connected = false
	s.client.LastSeen = time.Now()

	log.Info().
		Str("ucode", s.client.UCode).
		Msg("Disconnected from client")

	s.emitEvent(models.ClientEventDisconnected, "连接断开")

	return nil
}

// IsConnected 检查是否连接
func (s *ClientService) IsConnected() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.connected && s.conn != nil
}

// GetClient 获取客户端信息
func (s *ClientService) GetClient() *models.Client {
	return s.client
}

// AddEventHandler 添加事件处理器
func (s *ClientService) AddEventHandler(eventType models.ClientEventType, handler func(*models.ClientEvent)) {
	s.handlerMutex.Lock()
	defer s.handlerMutex.Unlock()

	s.eventHandlers[eventType] = append(s.eventHandlers[eventType], handler)
}

// RemoveEventHandler 移除事件处理器
func (s *ClientService) RemoveEventHandler(eventType models.ClientEventType, handler func(*models.ClientEvent)) {
	s.handlerMutex.Lock()
	defer s.handlerMutex.Unlock()

	handlers := s.eventHandlers[eventType]
	for i, h := range handlers {
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
			s.eventHandlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// emitEvent 发送事件
func (s *ClientService) emitEvent(eventType models.ClientEventType, message string) {
	event := &models.ClientEvent{
		Type:      eventType,
		UCode:     s.client.UCode,
		Timestamp: time.Now(),
		Message:   message,
	}

	s.handlerMutex.RLock()
	handlers := s.eventHandlers[eventType]
	s.handlerMutex.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}

// statusMonitor 状态监控器
func (s *ClientService) statusMonitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkHealth()
		}
	}
}

// checkHealth 健康检查
func (s *ClientService) checkHealth() {
	// 检查连接超时
	if s.client.Connected && time.Since(s.client.LastSeen) > 30*time.Second {
		log.Warn().
			Str("ucode", s.client.UCode).
			Msg("Client heartbeat timeout, marking as disconnected")

		s.disconnect()
	}
}

// disconnect 断开连接（内部方法）
func (s *ClientService) disconnect() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}

	s.connected = false
	s.client.Connected = false
	s.client.LastSeen = time.Now()
}

// handleMessage 处理来自客户端的消息（客户端消息，与机器人消息不同）
func (s *ClientService) handleMessage(messageType int, data []byte) error {
	var message models.WebSocketMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return fmt.Errorf("failed to unmarshal client message: %w", err)
	}

	// 更新客户端最后活跃时间
	s.client.LastSeen = time.Now()

	log.Debug().
		Str("ucode", s.client.UCode).
		Str("command", string(message.Command)).
		Int64("sequence", message.Sequence).
		Msg("Received message from client")

	// 客户端消息处理（控制命令、游戏命令等）
	switch message.Command {
	case models.CMD_TYPE_BIND_ROBOT:
		return s.handleBindRobot(&message)
	case models.CMD_TYPE_CONTROL_ROBOT:
		return s.handleControlRobot(&message)
	case models.CMD_TYPE_PING:
		return s.handlePing(&message)
	case models.CMD_TYPE_JOIN_GAME:
		return s.handleJoinGame(&message)
	case models.CMD_TYPE_LEAVE_GAME:
		return s.handleLeaveGame(&message)
	case models.CMD_TYPE_GAME_SHOOT:
		return s.handleGameShoot(&message)
	case models.CMD_TYPE_GAME_MOVE:
		return s.handleGameMove(&message)
	case models.CMD_TYPE_GAME_STATUS:
		return s.handleGameStatus(&message)
	case models.CMD_TYPE_GAME_START:
		return s.handleGameStart(&message)
	case models.CMD_TYPE_GAME_STOP:
		return s.handleGameStop(&message)
	default:
		log.Debug().
			Str("ucode", s.client.UCode).
			Str("command", string(message.Command)).
			Msg("Unknown client message command")
		return utils.SendError(s.conn, &message, "未知命令")
	}
}

// handleBindRobot 处理机器人绑定
func (s *ClientService) handleBindRobot(message *models.WebSocketMessage) error {
	var bindData models.CMD_BIND_ROBOT
	if data, ok := message.Data.(map[string]interface{}); ok {
		if ucode, exists := data["ucode"].(string); exists {
			bindData.UCode = ucode
		}
	}

	// 检查机器人是否存在
	robot, err := s.robotManager.GetRobot(bindData.UCode)
	if err != nil {
		return utils.SendError(s.conn, message, "机器人不存在")
	}

	// 检查机器人是否在线
	if !robot.GetService().IsConnected() {
		return utils.SendError(s.conn, message, "机器人未连接")
	}

	// 绑定机器人（这里可以添加绑定逻辑）
	log.Info().
		Str("client_ucode", s.client.UCode).
		Str("robot_ucode", bindData.UCode).
		Msg("Client bound to robot")

	s.emitEvent(models.ClientEventCommand, fmt.Sprintf("绑定机器人: %s", bindData.UCode))

	return utils.SendSuccess(s.conn, message, "机器人绑定成功")
}

// handleControlRobot 处理机器人控制
func (s *ClientService) handleControlRobot(message *models.WebSocketMessage) error {
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

	command := &models.RobotCommand{
		Action:        controlData.Action,
		Params:        controlData.ParamMaps,
		Priority:      5,
		Timestamp:     controlData.Timestamp,
		OperatorUCode: s.client.UCode,
	}

	// 获取在线机器人
	robots := s.robotManager.GetOnlineRobots()
	if len(robots) == 0 {
		return utils.SendError(s.conn, message, "没有可用的在线机器人")
	}

	// 发送命令到第一个在线机器人
	response, err := s.robotManager.SendCommand(robots[0].UCode, command)
	if err != nil {
		return utils.SendError(s.conn, message, fmt.Sprintf("命令发送失败: %v", err))
	}

	s.emitEvent(models.ClientEventCommand, fmt.Sprintf("控制命令: %s", controlData.Action))

	return utils.SendCustom(s.conn, message, response)
}

// handlePing 处理心跳
func (s *ClientService) handlePing(message *models.WebSocketMessage) error {
	return utils.SendSuccess(s.conn, message, "pong")
}

// 游戏相关处理方法
func (s *ClientService) handleJoinGame(message *models.WebSocketMessage) error {
	// 这里可以添加游戏加入逻辑
	s.emitEvent(models.ClientEventGameJoin, "加入游戏")
	return utils.SendSuccess(s.conn, message, "加入游戏成功")
}

func (s *ClientService) handleLeaveGame(message *models.WebSocketMessage) error {
	// 这里可以添加游戏离开逻辑
	s.emitEvent(models.ClientEventGameLeave, "离开游戏")
	return utils.SendSuccess(s.conn, message, "离开游戏成功")
}

func (s *ClientService) handleGameShoot(message *models.WebSocketMessage) error {
	// 这里可以添加游戏射击逻辑
	return utils.SendSuccess(s.conn, message, "射击命令发送成功")
}

func (s *ClientService) handleGameMove(message *models.WebSocketMessage) error {
	// 这里可以添加游戏移动逻辑
	return utils.SendSuccess(s.conn, message, "移动命令发送成功")
}

func (s *ClientService) handleGameStatus(message *models.WebSocketMessage) error {
	// 这里可以添加游戏状态获取逻辑
	return utils.SendSuccess(s.conn, message, "游戏状态获取成功")
}

func (s *ClientService) handleGameStart(message *models.WebSocketMessage) error {
	// 这里可以添加游戏开始逻辑
	return utils.SendSuccess(s.conn, message, "游戏开始成功")
}

func (s *ClientService) handleGameStop(message *models.WebSocketMessage) error {
	// 这里可以添加游戏停止逻辑
	return utils.SendSuccess(s.conn, message, "游戏停止成功")
}
