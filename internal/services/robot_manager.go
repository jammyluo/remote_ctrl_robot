package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"remote-ctrl-robot/internal/models"

	"github.com/rs/zerolog/log"
)

// 机器人实体
type Robot struct {
	UCode     string            // 唯一标识
	Version   string            // 版本
	wsService *WebSocketService // 机器人连接
	CreatedAt time.Time         // 创建时间
	UpdatedAt time.Time         // 更新时间
	mutex     sync.RWMutex      // 状态锁
}

func (r *Robot) IsOnline() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.wsService.IsConnected()
}

// RobotManager 机器人管理器 - 合并了注册表功能和连接管理
type RobotManager struct {
	robots        map[string]*Robot // 机器人映射表
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
		robots:        make(map[string]*Robot),
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

	for ucode := range s.robots {
		s.RemoveRobotConnection(ucode)
	}

	return nil
}

// HandleMessage 处理机器人消息
func (s *RobotManager) HandleMessage(message *models.WebSocketMessage) error {
	robot, exists := s.robots[message.UCode]
	// 更新机器人状态
	if !exists {
		return fmt.Errorf("robot %s not found in registry", message.UCode)
	}

	robot.wsService.LastSeen = time.Now()

	log.Debug().
		Str("ucode", message.UCode).
		Int64("sequence", message.Sequence).
		Str("remote_addr", robot.wsService.RemoteAddr).
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

// SetRobotConnection 设置机器人连接
func (s *RobotManager) SetRobotConnection(ucode string, wsService *WebSocketService) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.robots[ucode].wsService = wsService
}

// RemoveRobotConnection 移除机器人连接
func (s *RobotManager) RemoveRobotConnection(ucode string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if robot, exists := s.robots[ucode]; exists {
		robot.wsService.Stop()
		delete(s.robots, ucode)
	}

	// 更新机器人状态
	if robot, exists := s.robots[ucode]; exists {
		robot.wsService.Stop()

		log.Info().
			Str("ucode", ucode).
			Msg("Robot disconnected")

		s.emitEvent(models.RobotEventDisconnected, ucode, "连接断开")
	}
}

// HandleStatusUpdate 处理机器人状态更新
func (s *RobotManager) HandleStatusUpdate(robot *Robot, message *models.WebSocketMessage) error {
	s.emitEvent(models.RobotEventStatusUpdate, robot.UCode, "机器人状态更新")

	return robot.wsService.SendSuccess(message, "status updated")

}

// HandlePing 处理机器人心跳响应
func (s *RobotManager) HandlePing(robot *Robot, message *models.WebSocketMessage) error {
	s.emitEvent(models.RobotEventHeartbeat, robot.UCode, "机器人心跳")
	// 发送pong响应
	return robot.wsService.SendSuccess(message, "pong")
}

// RegisterRobot 注册机器人
func (s *RobotManager) RegisterRobot(ucode string, wsService *WebSocketService) (*Robot, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 检查机器人是否已存在
	if robot, exists := s.robots[ucode]; exists {
		// 更新现有机器人信息
		robot.wsService = wsService

		log.Info().
			Str("ucode", robot.UCode).
			Msg("Robot reconnected")

		return robot, nil
	}

	// 创建新机器人
	s.robots[ucode] = &Robot{
		UCode:     ucode,
		wsService: wsService,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	log.Info().
		Str("ucode", ucode).
		Msg("Robot registered")

	return s.robots[ucode], nil
}

// GetRobot 获取机器人
func (s *RobotManager) GetRobot(ucode string) (*Robot, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	robot, exists := s.robots[ucode]
	if !exists {
		return nil, fmt.Errorf("robot %s not found in registry", ucode)
	}

	return robot, nil
}

// GetAllRobots 获取所有机器人
func (s *RobotManager) GetAllRobots() []*Robot {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	robots := make([]*Robot, 0, len(s.robots))
	for _, robot := range s.robots {
		robots = append(robots, robot)
	}

	return robots
}

// GetOnlineRobots 获取在线机器人
func (s *RobotManager) GetOnlineRobots() []*Robot {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var onlineRobots []*Robot
	for _, robot := range s.robots {
		if robot.wsService.IsConnected() {
			onlineRobots = append(onlineRobots, robot)
		}
	}

	return onlineRobots
}

// GetHealthyRobots 获取健康机器人
func (s *RobotManager) GetHealthyRobots() []*Robot {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var healthyRobots []*Robot
	for _, robot := range s.robots {
		if robot.wsService.IsConnected() {
			healthyRobots = append(healthyRobots, robot)
		}
	}

	return healthyRobots
}

// SendCommand 发送命令到机器人
func (s *RobotManager) SendCommand(ucode string, command *models.RobotCommand) error {
	// 获取机器人连接
	robot, err := s.GetRobot(ucode)
	if err != nil {
		return fmt.Errorf("robot not connected: %w", err)
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
	if err := robot.wsService.SendMessage(message); err != nil {
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

// handleRobotEvent 处理机器人事件
func (s *RobotManager) handleRobotEvent(event *models.RobotEvent) {
	// 转发事件给管理器的事件处理器
	s.handlerMutex.RLock()
	handlers := s.eventHandlers[event.Type]
	s.handlerMutex.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}

	log.Debug().
		Str("ucode", event.UCode).
		Str("type", string(event.Type)).
		Msg("Robot event handled")
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
		if !robot.wsService.IsConnected() {
			log.Warn().
				Str("ucode", robot.UCode).
				Msg("Robot heartbeat timeout detected")

			// 标记为离线，等待机器人重新连接
			robot.wsService.Stop()
		}
	}
}

// CleanupOfflineRobots 清理离线机器人
func (s *RobotManager) CleanupOfflineRobots(timeout time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	var toDelete []string

	for ucode, robot := range s.robots {
		if now.Sub(robot.wsService.LastSeen) > timeout {
			toDelete = append(toDelete, ucode)
		}
	}

	for _, ucode := range toDelete {
		// 停止服务
		if robot, exists := s.robots[ucode]; exists {
			robot.wsService.Stop()
			delete(s.robots, ucode)
		}
		delete(s.robots, ucode)
		log.Info().
			Str("ucode", ucode).
			Msg("Offline robot cleaned up from registry")
	}
}

// GetRobotCount 获取机器人数量
func (s *RobotManager) GetRobotCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.robots)
}
