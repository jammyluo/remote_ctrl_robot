package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"remote-ctrl-robot/internal/models"

	"github.com/rs/zerolog/log"
)

// RobotManager 机器人管理器 - 合并了注册表功能
type RobotManager struct {
	robots        map[string]*models.Robot // 机器人映射表
	services      map[string]*RobotService // 机器人服务映射
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
		robots:        make(map[string]*models.Robot),
		services:      make(map[string]*RobotService),
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

	// 停止所有机器人服务
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for ucode, service := range s.services {
		if err := service.Stop(); err != nil {
			log.Error().
				Err(err).
				Str("ucode", ucode).
				Msg("Failed to stop robot service")
		}
	}

	return nil
}

// RegisterRobot 注册机器人
func (s *RobotManager) RegisterRobot(registration *models.RobotRegistration) (*models.Robot, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 检查机器人是否已存在
	if existing, exists := s.robots[registration.UCode]; exists {
		// 更新现有机器人信息
		existing.Name = registration.Name
		existing.Type = registration.Type
		existing.LastSeen = time.Now()
		existing.UpdatedAt = time.Now()

		log.Info().
			Str("ucode", registration.UCode).
			Str("name", registration.Name).
			Msg("Robot updated in registry")

		// 如果服务不存在，创建新服务
		if _, serviceExists := s.services[registration.UCode]; !serviceExists {
			service := NewRobotService(existing, s)
			existing.SetService(service)
			s.setupRobotService(service, existing)
			s.services[registration.UCode] = service
		}

		return existing, nil
	}

	// 创建新机器人
	robot := &models.Robot{
		UCode:     registration.UCode,
		Name:      registration.Name,
		Type:      registration.Type,
		Config:    s.getDefaultConfig(registration.Type),
		Status:    &models.RobotStatus{},
		LastSeen:  time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.robots[registration.UCode] = robot

	// 创建机器人服务
	service := NewRobotService(robot, s)
	robot.SetService(service)

	// 设置服务事件处理器
	s.setupRobotService(service, robot)

	// 启动服务
	if err := service.Start(); err != nil {
		log.Error().
			Err(err).
			Str("ucode", robot.UCode).
			Msg("Failed to start robot service")
		return nil, err
	}

	// 保存服务实例
	s.services[robot.UCode] = service

	log.Info().
		Str("ucode", robot.UCode).
		Str("name", robot.Name).
		Str("type", string(registration.Type)).
		Msg("Robot registered and service started")

	return robot, nil
}

// setupRobotService 设置机器人服务的事件处理器
func (s *RobotManager) setupRobotService(service *RobotService, robot *models.Robot) {
	service.AddEventHandler(models.RobotEventConnected, s.handleRobotEvent)
	service.AddEventHandler(models.RobotEventDisconnected, s.handleRobotEvent)
	service.AddEventHandler(models.RobotEventError, s.handleRobotEvent)
	service.AddEventHandler(models.RobotEventStatusUpdate, s.handleRobotEvent)
	service.AddEventHandler(models.RobotEventCommand, s.handleRobotEvent)
	service.AddEventHandler(models.RobotEventHeartbeat, s.handleRobotEvent)
}

// UnregisterRobot 注销机器人
func (s *RobotManager) UnregisterRobot(ucode string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 停止服务
	if service, exists := s.services[ucode]; exists {
		if err := service.Stop(); err != nil {
			log.Error().
				Err(err).
				Str("ucode", ucode).
				Msg("Failed to stop robot service")
		}
		delete(s.services, ucode)
	}

	// 从注册表删除
	if _, exists := s.robots[ucode]; !exists {
		return fmt.Errorf("robot %s not found in registry", ucode)
	}

	delete(s.robots, ucode)

	log.Info().
		Str("ucode", ucode).
		Msg("Robot unregistered and service stopped")

	return nil
}

// GetRobot 获取机器人
func (s *RobotManager) GetRobot(ucode string) (*models.Robot, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	robot, exists := s.robots[ucode]
	if !exists {
		return nil, fmt.Errorf("robot %s not found in registry", ucode)
	}

	return robot, nil
}

// GetAllRobots 获取所有机器人
func (s *RobotManager) GetAllRobots() []*models.Robot {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	robots := make([]*models.Robot, 0, len(s.robots))
	for _, robot := range s.robots {
		robots = append(robots, robot)
	}

	return robots
}

// GetOnlineRobots 获取在线机器人
func (s *RobotManager) GetOnlineRobots() []*models.Robot {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var onlineRobots []*models.Robot
	for _, robot := range s.robots {
		if robot.IsOnline() {
			onlineRobots = append(onlineRobots, robot)
		}
	}

	return onlineRobots
}

// GetHealthyRobots 获取健康机器人
func (s *RobotManager) GetHealthyRobots() []*models.Robot {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var healthyRobots []*models.Robot
	for _, robot := range s.robots {
		if robot.IsHealthy() {
			healthyRobots = append(healthyRobots, robot)
		}
	}

	return healthyRobots
}

// SendCommand 发送命令到机器人
func (s *RobotManager) SendCommand(ucode string, command *models.RobotCommand) (*models.RobotCommandResponse, error) {
	s.mutex.RLock()
	service, exists := s.services[ucode]
	s.mutex.RUnlock()

	if !exists {
		return &models.RobotCommandResponse{
			Success:   false,
			Message:   "机器人服务不存在",
			Timestamp: time.Now().Unix(),
		}, fmt.Errorf("robot service not found: %s", ucode)
	}

	return service.SendCommand(command)
}

// GetRobotService 获取机器人服务
func (s *RobotManager) GetRobotService(ucode string) (*RobotService, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	service, exists := s.services[ucode]
	if !exists {
		return nil, fmt.Errorf("robot service not found: %s", ucode)
	}

	return service, nil
}

// GetRobotStatus 获取机器人状态
func (s *RobotManager) GetRobotStatus(ucode string) (*models.RobotStatus, error) {
	robot, err := s.GetRobot(ucode)
	if err != nil {
		return nil, err
	}

	return robot.GetStatus(), nil
}

// GetRobotStatistics 获取机器人统计信息
func (s *RobotManager) GetRobotStatistics() map[string]*models.RobotStatistics {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	statistics := make(map[string]*models.RobotStatistics)

	for ucode, robot := range s.robots {
		status := robot.GetStatus()
		if status == nil {
			continue
		}

		successRate := 0.0
		if status.TotalCommands > 0 {
			successRate = float64(status.TotalCommands-status.FailedCommands) / float64(status.TotalCommands) * 100
		}

		uptime := int64(0)
		if !robot.CreatedAt.IsZero() {
			uptime = int64(time.Since(robot.CreatedAt).Seconds())
		}

		statistics[ucode] = &models.RobotStatistics{
			UCode:          ucode,
			TotalCommands:  status.TotalCommands,
			FailedCommands: status.FailedCommands,
			SuccessRate:    successRate,
			AverageLatency: float64(status.Latency),
			Uptime:         uptime,
			LastSeen:       robot.LastSeen,
			ErrorCount:     0, // TODO: 实现错误计数
			ReconnectCount: 0, // TODO: 实现重连计数
		}
	}

	return statistics
}

// GetSystemStatus 获取系统状态
func (s *RobotManager) GetSystemStatus() *models.SystemStatus {
	onlineCount := s.GetOnlineRobotCount()
	healthyCount := s.GetHealthyRobotCount()
	totalCount := s.GetRobotCount()

	status := "unknown"
	if totalCount == 0 {
		status = "no_robots"
	} else if healthyCount == totalCount {
		status = "healthy"
	} else if onlineCount > 0 {
		status = "partial"
	} else {
		status = "offline"
	}

	return &models.SystemStatus{
		ServerTime:    time.Now(),
		Uptime:        int64(time.Since(time.Now()).Seconds()), // TODO: 实现真正的启动时间
		ActiveClients: onlineCount,
		RobotStatus:   status,
		JanusStatus:   "unknown", // TODO: 集成Janus状态
	}
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
		status := robot.GetStatus()
		if status == nil {
			continue
		}

		// 检查连接状态
		if status.Connected && time.Since(status.LastHeartbeat) > 30*time.Second {
			log.Warn().
				Str("ucode", robot.UCode).
				Msg("Robot heartbeat timeout detected")

			// 标记为离线，等待机器人重新连接
			status.Connected = false
			robot.UpdateStatus(status)
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
		if now.Sub(robot.LastSeen) > timeout {
			toDelete = append(toDelete, ucode)
		}
	}

	for _, ucode := range toDelete {
		// 停止服务
		if service, exists := s.services[ucode]; exists {
			service.Stop()
			delete(s.services, ucode)
		}
		delete(s.robots, ucode)
		log.Info().
			Str("ucode", ucode).
			Msg("Offline robot cleaned up from registry")
	}
}

// GetRobotConfig 获取机器人配置
func (s *RobotManager) GetRobotConfig(ucode string) (*models.RobotConfig, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	robot, exists := s.robots[ucode]
	if !exists {
		return nil, fmt.Errorf("robot %s not found in registry", ucode)
	}

	return &robot.Config, nil
}

// SetRobotConfig 设置机器人配置
func (s *RobotManager) SetRobotConfig(ucode string, config models.RobotConfig) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	robot, exists := s.robots[ucode]
	if !exists {
		return fmt.Errorf("robot %s not found in registry", ucode)
	}

	robot.Config = config
	robot.UpdatedAt = time.Now()

	log.Info().
		Str("ucode", ucode).
		Str("websocket_url", config.WebSocketURL).
		Msg("Robot config updated")

	// 重启机器人服务以应用新配置
	if service, exists := s.services[ucode]; exists {
		// 停止当前服务
		if err := service.Stop(); err != nil {
			log.Error().
				Err(err).
				Str("ucode", ucode).
				Msg("Failed to stop robot service for config update")
		}

		// 创建新服务
		newService := NewRobotService(robot, s)
		s.setupRobotService(newService, robot)

		// 启动新服务
		if err := newService.Start(); err != nil {
			log.Error().
				Err(err).
				Str("ucode", ucode).
				Msg("Failed to start robot service with new config")
			return err
		}

		// 更新服务映射
		s.services[ucode] = newService

		log.Info().
			Str("ucode", ucode).
			Msg("Robot service restarted with new config")
	}

	return nil
}

// getDefaultConfig 获取默认配置
func (s *RobotManager) getDefaultConfig(robotType models.RobotType) models.RobotConfig {
	config := models.RobotConfig{
		Timeout:           30,
		MaxRetries:        3,
		HeartbeatInterval: 10,
		ReconnectInterval: 5,
	}

	// 根据机器人类型设置不同的默认配置
	switch robotType {
	case models.RobotTypeB2, models.RobotTypeB2W:
		config.WebSocketURL = "ws://192.168.1.100:8080"
	case models.RobotTypeGo2, models.RobotTypeGo2W:
		config.WebSocketURL = "ws://192.168.1.101:8080"
	case models.RobotTypeG1:
		config.WebSocketURL = "ws://192.168.1.102:8080"
	case models.RobotTypeH1, models.RobotTypeH1_2:
		config.WebSocketURL = "ws://192.168.1.103:8080"
	default:
		config.WebSocketURL = "ws://192.168.1.100:8080"
	}

	return config
}

// GetRobotCount 获取机器人数量
func (s *RobotManager) GetRobotCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.robots)
}

// GetOnlineRobotCount 获取在线机器人数量
func (s *RobotManager) GetOnlineRobotCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	count := 0
	for _, robot := range s.robots {
		if robot.IsOnline() {
			count++
		}
	}
	return count
}

// GetHealthyRobotCount 获取健康机器人数量
func (s *RobotManager) GetHealthyRobotCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	count := 0
	for _, robot := range s.robots {
		if robot.IsHealthy() {
			count++
		}
	}
	return count
}
