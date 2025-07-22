package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"remote-ctrl-robot/internal/models"

	"github.com/rs/zerolog/log"
)

// OperatorManager 客户端管理器
type OperatorManager struct {
	operators      map[string]*WebSocketService // 客户端映射表
	operator2Robot map[string]string            // 客户端到机器人的映射表
	mutex          sync.RWMutex
	robotManager   *RobotManager
	ctx            context.Context
	cancel         context.CancelFunc
	eventHandlers  map[models.OperatorEventType][]func(*models.OperatorEvent)
	handlerMutex   sync.RWMutex
}

// NewOperatorManager 创建新的客户端管理器
func NewOperatorManager(robotManager *RobotManager) *OperatorManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &OperatorManager{
		operators:      make(map[string]*WebSocketService),
		operator2Robot: make(map[string]string),
		robotManager:   robotManager,
		ctx:            ctx,
		cancel:         cancel,
		eventHandlers:  make(map[models.OperatorEventType][]func(*models.OperatorEvent)),
	}
}

// Start 启动操作员管理器
func (s *OperatorManager) Start() error {
	log.Info().Msg("Starting operator manager")

	// 启动清理任务
	go s.cleanupTask()

	// 启动健康检查任务
	go s.healthCheckTask()

	return nil
}

// Stop 停止操作员管理器
func (s *OperatorManager) Stop() error {
	log.Info().Msg("Stopping operator manager")

	s.cancel()

	// 关闭所有操作员连接
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, operator := range s.operators {
		s.UnregisterOperator(operator)
	}

	return nil
}

// RegisterOperator 注册操作员
func (cm *OperatorManager) RegisterOperator(operator *WebSocketService) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 检查客户端是否已存在
	if _, exists := cm.operators[operator.UCode]; exists {
		log.Error().
			Str("ucode", operator.UCode).
			Msg("Operator already registered")
		return fmt.Errorf("operator already registered")
	}

	cm.operators[operator.UCode] = operator
	cm.emitEvent(models.OperatorEventConnected, operator.UCode, "操作员连接成功")
	log.Info().
		Str("ucode", operator.UCode).
		Msg("Register Operator Success")

	return nil
}

// UnregisterOperator 移除操作员（立即清理）
func (cm *OperatorManager) UnregisterOperator(operator *WebSocketService) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if _, exists := cm.operators[operator.UCode]; exists {
		// 更新操作员状态
		delete(cm.operators, operator.UCode)
		cm.emitEvent(models.OperatorEventDisconnected, operator.UCode, "操作员连接断开")
		log.Info().
			Str("ucode", operator.UCode).
			Msg("UnregisterOperator Success")
	} else {
		log.Error().
			Str("ucode", operator.UCode).
			Msg("UnregisterOperator: Operator not found")
	}
}

// GetOperator 获取操作员
func (cm *OperatorManager) GetOperator(ucode string) (*WebSocketService, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	operator, exists := cm.operators[ucode]
	if !exists {
		return nil, fmt.Errorf("operator %s not found", ucode)
	}

	return operator, nil
}

// GetAllOperators 获取所有操作员
func (cm *OperatorManager) GetAllOperators() []*WebSocketService {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	operators := make([]*WebSocketService, 0, len(cm.operators))
	for _, operator := range cm.operators {
		operators = append(operators, operator)
	}

	return operators
}

// GetClientCount 获取客户端数量
func (cm *OperatorManager) GetClientCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return len(cm.operators)
}

// AddEventHandler 添加事件处理器
func (s *OperatorManager) AddEventHandler(eventType models.OperatorEventType, handler func(*models.OperatorEvent)) {
	s.handlerMutex.Lock()
	defer s.handlerMutex.Unlock()

	s.eventHandlers[eventType] = append(s.eventHandlers[eventType], handler)
}

// RemoveEventHandler 移除事件处理器
func (s *OperatorManager) RemoveEventHandler(eventType models.OperatorEventType, handler func(*models.OperatorEvent)) {
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
func (s *OperatorManager) emitEvent(eventType models.OperatorEventType, ucode string, message string) {
	event := &models.OperatorEvent{
		Type:      eventType,
		UCode:     ucode,
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

// cleanupTask 清理任务
func (s *OperatorManager) cleanupTask() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// 清理离线操作员
			s.CleanupOfflineOperators(10 * time.Minute)
		}
	}
}

// healthCheckTask 健康检查任务
func (s *OperatorManager) healthCheckTask() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (s *OperatorManager) performHealthCheck() {
	operators := s.GetAllOperators()

	for _, operator := range operators {
		if !operator.IsOnline() {
			log.Warn().
				Str("ucode", operator.UCode).
				Msg("Operator disconnected detected")
			s.UnregisterOperator(operator)
		}
	}
}

// CleanupOfflineOperators 清理离线操作员
func (s *OperatorManager) CleanupOfflineOperators(timeout time.Duration) {
	operators := s.GetAllOperators()

	for _, operator := range operators {
		if operator.IsTimeout(timeout) {
			log.Warn().
				Str("ucode", operator.UCode).
				Msg("Operator heartbeat timeout detected")
			s.UnregisterOperator(operator)
		}
	}
}

// GetOperatorCount 获取操作员数量
func (s *OperatorManager) GetOperatorCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.operators)
}

// handleMessage 处理来自客户端的消息（客户端消息，与机器人消息不同）
func (s *OperatorManager) HandleMessage(message *models.WebSocketMessage) error {
	operator, err := s.GetOperator(message.UCode)
	if err != nil {
		return err
	}
	operator.UpdatedAt = time.Now()

	log.Debug().
		Str("ucode", message.UCode).
		Int64("sequence", message.Sequence).
		Str("remote_addr", operator.RemoteAddr).
		Str("command", string(message.Command)).
		Msg("Received operator message")

	// 客户端消息处理（控制命令、游戏命令等）
	switch message.Command {
	case models.CMD_TYPE_BIND_ROBOT:
		return s.handleBindRobot(operator, message)
	case models.CMD_TYPE_CONTROL_ROBOT:
		return s.handleControlRobot(operator, message)
	case models.CMD_TYPE_PING:
		return s.handlePing(operator, message)
	case models.CMD_TYPE_JOIN_GAME:
		return s.handleJoinGame(operator, message)
	case models.CMD_TYPE_LEAVE_GAME:
		return s.handleLeaveGame(operator, message)
	case models.CMD_TYPE_GAME_SHOOT:
		return s.handleGameShoot(operator, message)
	case models.CMD_TYPE_GAME_MOVE:
		return s.handleGameMove(operator, message)
	case models.CMD_TYPE_GAME_STATUS:
		return s.handleGameStatus(operator, message)
	case models.CMD_TYPE_GAME_START:
		return s.handleGameStart(operator, message)
	case models.CMD_TYPE_GAME_STOP:
		return s.handleGameStop(operator, message)
	default:
		log.Debug().
			Str("ucode", message.UCode).
			Str("command", string(message.Command)).
			Msg("Unknown client message command")
	}

	return nil
}

// handleBindRobot 处理机器人绑定
func (s *OperatorManager) handleBindRobot(operator *WebSocketService, message *models.WebSocketMessage) error {
	var bindData models.CMD_BIND_ROBOT
	if data, ok := message.Data.(map[string]interface{}); ok {
		if ucode, exists := data["ucode"].(string); exists {
			bindData.UCode = ucode
		}
	}

	// 检查机器人是否存在
	robot, err := s.robotManager.GetRobot(bindData.UCode)
	if err != nil {
		log.Error().Err(err).Str("ucode", bindData.UCode).Msg("Robot not found")
		return operator.SendError(message, "Robot not found")
	}

	// 检查机器人是否在线
	if !robot.IsOnline() {
		log.Error().Str("ucode", bindData.UCode).Msg("Robot not connected")
		return operator.SendError(message, "Robot not connected")
	}

	// 绑定机器人
	s.operator2Robot[operator.UCode] = bindData.UCode
	s.emitEvent(models.OperatorEventBindRobot, operator.UCode, fmt.Sprintf("绑定机器人: %s", bindData.UCode))
	log.Info().
		Str("client_ucode", operator.UCode).
		Str("robot_ucode", bindData.UCode).
		Msg("Client bound to robot")

	return operator.SendSuccess(message, "Robot bound success")
}

// handleControlRobot 处理机器人控制
func (s *OperatorManager) handleControlRobot(operator *WebSocketService, message *models.WebSocketMessage) error {

	robotUcode, exists := s.operator2Robot[operator.UCode]
	if !exists {
		log.Error().Str("ucode", operator.UCode).Msg("Robot not bound")
		return operator.SendError(message, "Robot not bound")
	}

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
		OperatorUCode: operator.UCode,
	}

	err := s.robotManager.SendCommand(robotUcode, command)
	if err != nil {
		log.Error().Err(err).Str("ucode", robotUcode).Msg("Send command to robot failed")
		return operator.SendError(message, "Send command to robot failed")
	}

	s.emitEvent(models.OperatorEventControlRobot, operator.UCode, fmt.Sprintf("控制机器人: %s", controlData.Action))

	return operator.SendSuccess(message, "Send command to robot success")
}

// handlePing 处理心跳
func (s *OperatorManager) handlePing(operator *WebSocketService, message *models.WebSocketMessage) error {
	operator.UpdatedAt = time.Now()
	s.emitEvent(models.OperatorEventHeartbeat, operator.UCode, "操作员心跳")

	return operator.SendSuccess(message, "pong")
}

// 游戏相关处理方法
func (s *OperatorManager) handleJoinGame(operator *WebSocketService, message *models.WebSocketMessage) error {
	// 这里可以添加游戏加入逻辑
	s.emitEvent(models.OperatorEventJoinGame, operator.UCode, "加入游戏")
	return operator.SendSuccess(message, "加入游戏成功")
}

func (s *OperatorManager) handleLeaveGame(operator *WebSocketService, message *models.WebSocketMessage) error {
	// 这里可以添加游戏离开逻辑
	s.emitEvent(models.OperatorEventLeaveGame, operator.UCode, "离开游戏")
	return operator.SendSuccess(message, "离开游戏成功")
}

func (s *OperatorManager) handleGameShoot(operator *WebSocketService, message *models.WebSocketMessage) error {
	// 这里可以添加游戏射击逻辑
	return operator.SendSuccess(message, "射击命令发送成功")
}

func (s *OperatorManager) handleGameMove(operator *WebSocketService, message *models.WebSocketMessage) error {
	// 这里可以添加游戏移动逻辑
	return operator.SendSuccess(message, "移动命令发送成功")
}

func (s *OperatorManager) handleGameStatus(operator *WebSocketService, message *models.WebSocketMessage) error {
	// 这里可以添加游戏状态获取逻辑
	return operator.SendSuccess(message, "游戏状态获取成功")
}

func (s *OperatorManager) handleGameStart(operator *WebSocketService, message *models.WebSocketMessage) error {
	// 这里可以添加游戏开始逻辑
	return operator.SendSuccess(message, "游戏开始成功")
}

func (s *OperatorManager) handleGameStop(operator *WebSocketService, message *models.WebSocketMessage) error {
	// 这里可以添加游戏停止逻辑
	return operator.SendSuccess(message, "游戏停止成功")
}
