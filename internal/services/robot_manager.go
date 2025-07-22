package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"remote-ctrl-robot/internal/models"

	"github.com/rs/zerolog/log"
)

// RobotManager 机器人管理器 - 合并了注册表功能和连接管理
type RobotManager struct {
	robots        map[string]*WebSocketService // 机器人映射表
	mutex         sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	eventHandlers map[models.RobotEventType][]func(*models.RobotEvent)
	handlerMutex  sync.RWMutex
}

// NewRobotManager 创建新的机器人管理器
func NewRobotManager() *RobotManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &RobotManager{
		robots:        make(map[string]*WebSocketService),
		ctx:           ctx,
		cancel:        cancel,
		eventHandlers: make(map[models.RobotEventType][]func(*models.RobotEvent)),
	}
}

// Start 启动机器人管理器
func (s *RobotManager) Start() error {
	log.Info().Msg("Starting robot manager")

	// 启动清理任务
	go s.cleanupTask()

	// 启动健康检查任务
	go s.healthCheckTask()

	return nil
}

// Stop 停止机器人管理器
func (s *RobotManager) Stop() error {
	log.Info().Msg("Stopping robot manager")

	s.cancel()

	// 关闭所有机器人连接
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, robot := range s.robots {
		s.UnregisterRobot(robot)
	}

	return nil
}

func (s *RobotManager) RegisterRobot(robot *WebSocketService) (*WebSocketService, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 检查Robot是否已存在
	if _, exists := s.robots[robot.UCode]; exists {
		log.Error().
			Str("ucode", robot.UCode).
			Msg("Robot already registered")
		return nil, fmt.Errorf("robot already registered")
	}
	// 创建Robot
	s.robots[robot.UCode] = robot
	log.Info().
		Str("ucode", robot.UCode).
		Msg("Robot registered")

	return s.robots[robot.UCode], nil
}

func (s *RobotManager) UnregisterRobot(robot *WebSocketService) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, exists := s.robots[robot.UCode]; exists {
		delete(s.robots, robot.UCode)
		s.emitEvent(models.RobotEventDisconnected, robot.UCode, "连接断开")
		log.Info().
			Str("ucode", robot.UCode).
			Msg("UnregisterRobot Success")
	} else {
		log.Error().
			Str("ucode", robot.UCode).
			Msg("UnregisterRobot: Robot not found")
	}
}

// GetRobot 获取机器人
func (s *RobotManager) GetRobot(ucode string) (*WebSocketService, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	robot, exists := s.robots[ucode]
	if !exists {
		return nil, fmt.Errorf("robot %s not found in registry", ucode)
	}

	return robot, nil
}

// GetAllRobots 获取所有机器人
func (s *RobotManager) GetAllRobots() []*WebSocketService {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	robots := make([]*WebSocketService, 0, len(s.robots))
	for _, robot := range s.robots {
		robots = append(robots, robot)
	}

	return robots
}

// GetOnlineRobots 获取在线机器人
func (s *RobotManager) GetOnlineRobots() []*WebSocketService {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var onlineRobots []*WebSocketService
	for _, robot := range s.robots {
		if robot.IsOnline() {
			onlineRobots = append(onlineRobots, robot)
		}
	}

	return onlineRobots
}

// SendCommand 发送命令到机器人
func (s *RobotManager) SendCommand(ucode string, command *models.RobotCommand) error {
	// 获取机器人连接
	robot, err := s.GetRobot(ucode)
	if err != nil {
		return fmt.Errorf("robot not found: %w", err)
	}

	if !robot.IsOnline() {
		return fmt.Errorf("robot not online: %w", err)
	}

	// 创建WebSocket消息
	message := models.WebSocketMessage{
		Type:       models.WSMessageTypeRequest,
		Command:    models.CMD_TYPE_CONTROL_ROBOT,
		Sequence:   time.Now().UnixNano(),
		UCode:      ucode,
		ClientType: models.ClientTypeOperator,
		Version:    "1.0",
		Data: models.CMD_CONTROL_ROBOT{
			Action:    command.Action,
			ParamMaps: command.Params,
			Timestamp: command.Timestamp,
		},
	}

	// 发送命令
	if err := robot.SendMessage(message); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	s.emitEvent(models.RobotEventCommand, ucode, fmt.Sprintf("命令执行: %s", command.Action))

	log.Info().
		Str("ucode", ucode).
		Str("action", command.Action).
		Int("priority", command.Priority).
		Msg("Command sent to robot successfully")

	return nil
}

// AddEventHandler 添加事件处理器
func (s *RobotManager) AddEventHandler(eventType models.RobotEventType, handler func(*models.RobotEvent)) {
	s.handlerMutex.Lock()
	defer s.handlerMutex.Unlock()

	s.eventHandlers[eventType] = append(s.eventHandlers[eventType], handler)
}

// RemoveEventHandler 移除事件处理器
func (s *RobotManager) RemoveEventHandler(eventType models.RobotEventType, handler func(*models.RobotEvent)) {
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
func (s *RobotManager) emitEvent(eventType models.RobotEventType, ucode string, message string) {
	event := &models.RobotEvent{
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
func (s *RobotManager) cleanupTask() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// 清理离线机器人
			s.CleanupOfflineRobots(10 * time.Minute)
		}
	}
}

// healthCheckTask 健康检查任务
func (s *RobotManager) healthCheckTask() {
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
func (s *RobotManager) performHealthCheck() {
	robots := s.GetAllRobots()

	for _, robot := range robots {
		if !robot.IsOnline() {
			log.Warn().
				Str("ucode", robot.UCode).
				Msg("Robot disconnected detected")
			s.UnregisterRobot(robot)
		}
	}
}

// CleanupOfflineRobots 清理离线机器人
func (s *RobotManager) CleanupOfflineRobots(timeout time.Duration) {
	robots := s.GetAllRobots()

	for _, robot := range robots {
		if robot.IsTimeout(timeout) {
			log.Warn().
				Str("ucode", robot.UCode).
				Msg("Robot heartbeat timeout detected")
			s.UnregisterRobot(robot)
		}
	}
}

// GetRobotCount 获取机器人数量
func (s *RobotManager) GetRobotCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.robots)
}

// HandleMessage 处理机器人消息
func (s *RobotManager) HandleMessage(message *models.WebSocketMessage) error {
	robot, exists := s.robots[message.UCode]
	// 更新机器人状态
	if !exists {
		return fmt.Errorf("robot %s not found in registry", message.UCode)
	}

	robot.UpdatedAt = time.Now()

	log.Debug().
		Str("ucode", message.UCode).
		Int64("sequence", message.Sequence).
		Str("remote_addr", robot.RemoteAddr).
		Str("command", string(message.Command)).
		Msg("Received robot message")

	// 机器人端消息处理（状态上报、心跳等）
	switch message.Command {
	case models.CMD_TYPE_UPDATE_ROBOT_STATUS:
		return s.HandleStatusUpdate(robot, message)
	case models.CMD_TYPE_PING:
		return s.HandlePing(robot, message)
	default:
		log.Debug().
			Str("ucode", message.UCode).
			Str("command", string(message.Command)).
			Msg("Unknown robot message command")
	}
	return nil
}

// HandleStatusUpdate 处理机器人状态更新
func (s *RobotManager) HandleStatusUpdate(robot *WebSocketService, message *models.WebSocketMessage) error {
	s.emitEvent(models.RobotEventStatusUpdate, robot.UCode, "Robot status updated")

	return robot.SendSuccess(message, "status updated")

}

// HandlePing 处理机器人心跳响应
func (s *RobotManager) HandlePing(robot *WebSocketService, message *models.WebSocketMessage) error {
	s.emitEvent(models.RobotEventHeartbeat, robot.UCode, "机器人心跳")
	// 发送pong响应
	return robot.SendSuccess(message, "pong")
}
