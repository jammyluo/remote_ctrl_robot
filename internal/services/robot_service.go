package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"remote-ctrl-robot/internal/models"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// RobotService 单个机器人服务
type RobotService struct {
	robot         *models.Robot
	manager       *RobotManager
	conn          *websocket.Conn
	mutex         sync.RWMutex
	connected     bool
	ctx           context.Context
	cancel        context.CancelFunc
	eventHandlers map[models.RobotEventType][]func(*models.RobotEvent)
	handlerMutex  sync.RWMutex
}

// NewRobotService 创建新的单个机器人服务
func NewRobotService(robot *models.Robot, manager *RobotManager) *RobotService {
	ctx, cancel := context.WithCancel(context.Background())

	return &RobotService{
		robot:         robot,
		manager:       manager,
		ctx:           ctx,
		cancel:        cancel,
		eventHandlers: make(map[models.RobotEventType][]func(*models.RobotEvent)),
	}
}

// Start 启动机器人服务
func (s *RobotService) Start() error {
	log.Info().
		Str("ucode", s.robot.UCode).
		Str("name", s.robot.Name).
		Msg("Starting robot service")

	// 启动状态监控
	go s.statusMonitor()

	return nil
}

// Stop 停止机器人服务
func (s *RobotService) Stop() error {
	log.Info().
		Str("ucode", s.robot.UCode).
		Msg("Stopping robot service")

	s.cancel()
	s.disconnect()

	return nil
}

// SetConnection 设置机器人连接（由WebSocket处理器调用）
func (s *RobotService) SetConnection(conn *websocket.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.conn = conn
	s.connected = true

	// 更新状态
	status := s.robot.GetStatus()
	if status == nil {
		status = &models.RobotStatus{}
	}
	status.Connected = true
	status.LastHeartbeat = time.Now()
	status.ErrorCode = 0
	status.ErrorMessage = ""
	s.robot.UpdateStatus(status)

	log.Info().
		Str("ucode", s.robot.UCode).
		Msg("Robot connected")

	s.emitEvent(models.RobotEventConnected, "机器人连接成功")

	// 启动消息处理
	go s.handleMessages()
}

// handleMessages 处理来自机器人的消息
func (s *RobotService) handleMessages() {
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
					log.Error().Err(err).Str("ucode", s.robot.UCode).Msg("WebSocket read error")
				}
				return
			}

			// 处理消息
			if err := s.handleMessage(websocket.TextMessage, data); err != nil {
				log.Error().Err(err).Str("ucode", s.robot.UCode).Msg("Failed to handle message")
			}
		}
	}
}

// Disconnect 断开连接
func (s *RobotService) Disconnect() error {
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

	// 更新状态
	status := s.robot.GetStatus()
	if status != nil {
		status.Connected = false
		s.robot.UpdateStatus(status)
	}

	log.Info().
		Str("ucode", s.robot.UCode).
		Msg("Disconnected from robot")

	s.emitEvent(models.RobotEventDisconnected, "连接断开")

	return nil
}

// SendCommand 发送命令到机器人
func (s *RobotService) SendCommand(command *models.RobotCommand) (*models.RobotCommandResponse, error) {
	s.mutex.RLock()
	if !s.connected || s.conn == nil {
		s.mutex.RUnlock()
		return &models.RobotCommandResponse{
			Success:   false,
			Message:   "机器人未连接",
			Timestamp: time.Now().Unix(),
		}, fmt.Errorf("robot not connected")
	}
	s.mutex.RUnlock()

	// 更新命令统计
	status := s.robot.GetStatus()
	if status != nil {
		status.TotalCommands++
		status.LastCommandTime = time.Now()
		s.robot.UpdateStatus(status)
	}

	// 创建WebSocket消息
	message := models.WebSocketMessage{
		Type:       models.WSMessageTypeRequest,
		Command:    models.CMD_TYPE_CONTROL_ROBOT,
		Sequence:   time.Now().UnixNano(),
		UCode:      s.robot.UCode,
		ClientType: models.ClientTypeOperator,
		Version:    "1.0",
		Data: models.CMD_CONTROL_ROBOT{
			Action:    command.Action,
			ParamMaps: command.Params,
			Timestamp: command.Timestamp,
		},
	}

	// 发送命令
	if err := s.conn.WriteJSON(message); err != nil {
		// 更新失败统计
		if status != nil {
			status.FailedCommands++
			s.robot.UpdateStatus(status)
		}

		log.Error().
			Err(err).
			Str("ucode", s.robot.UCode).
			Str("action", command.Action).
			Msg("Failed to send command to robot")

		s.emitEvent(models.RobotEventError, fmt.Sprintf("命令发送失败: %v", err))

		return &models.RobotCommandResponse{
			Success:   false,
			Message:   fmt.Sprintf("命令发送失败: %v", err),
			Timestamp: time.Now().Unix(),
		}, fmt.Errorf("failed to send command: %w", err)
	}

	log.Info().
		Str("ucode", s.robot.UCode).
		Str("action", command.Action).
		Int("priority", command.Priority).
		Msg("Command sent to robot successfully")

	s.emitEvent(models.RobotEventCommand, fmt.Sprintf("命令执行: %s", command.Action))

	return &models.RobotCommandResponse{
		Success:   true,
		Message:   "命令发送成功",
		CommandID: fmt.Sprintf("%d", message.Sequence),
		Timestamp: time.Now().Unix(),
	}, nil
}

// GetStatus 获取机器人状态
func (s *RobotService) GetStatus() *models.RobotStatus {
	return s.robot.GetStatus()
}

// IsConnected 检查是否连接
func (s *RobotService) IsConnected() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.connected && s.conn != nil
}

// AddEventHandler 添加事件处理器
func (s *RobotService) AddEventHandler(eventType models.RobotEventType, handler func(*models.RobotEvent)) {
	s.handlerMutex.Lock()
	defer s.handlerMutex.Unlock()

	s.eventHandlers[eventType] = append(s.eventHandlers[eventType], handler)
}

// RemoveEventHandler 移除事件处理器
func (s *RobotService) RemoveEventHandler(eventType models.RobotEventType, handler func(*models.RobotEvent)) {
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
func (s *RobotService) emitEvent(eventType models.RobotEventType, message string) {
	event := &models.RobotEvent{
		Type:      eventType,
		UCode:     s.robot.UCode,
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
func (s *RobotService) statusMonitor() {
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
func (s *RobotService) checkHealth() {
	status := s.robot.GetStatus()
	if status == nil {
		return
	}

	// 检查连接超时
	if status.Connected && time.Since(status.LastHeartbeat) > 30*time.Second {
		log.Warn().
			Str("ucode", s.robot.UCode).
			Msg("Robot heartbeat timeout, marking as disconnected")

		s.disconnect()
	}
}

// disconnect 断开连接（内部方法）
func (s *RobotService) disconnect() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}

	s.connected = false

	// 更新状态
	status := s.robot.GetStatus()
	if status != nil {
		status.Connected = false
		s.robot.UpdateStatus(status)
	}
}

// handleMessage 处理来自机器人的消息（机器人端消息，与客户端消息不同）
func (s *RobotService) handleMessage(messageType int, data []byte) error {
	var message models.WebSocketMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return fmt.Errorf("failed to unmarshal robot message: %w", err)
	}

	log.Debug().
		Str("ucode", s.robot.UCode).
		Str("command", string(message.Command)).
		Int64("sequence", message.Sequence).
		Msg("Received message from robot")

	// 机器人端消息处理（状态上报、心跳等）
	switch message.Command {
	case models.CMD_TYPE_UPDATE_ROBOT_STATUS:
		return s.handleStatusUpdate(message.Data)
	case models.CMD_TYPE_PING:
		return s.handlePing(message)
	default:
		log.Debug().
			Str("ucode", s.robot.UCode).
			Str("command", string(message.Command)).
			Msg("Unknown robot message command")
	}

	return nil
}

// handleStatusUpdate 处理机器人状态更新
func (s *RobotService) handleStatusUpdate(data interface{}) error {
	// 更新机器人状态
	status := s.robot.GetStatus()
	if status != nil {
		status.LastHeartbeat = time.Now()
		s.robot.UpdateStatus(status)
	}

	s.emitEvent(models.RobotEventStatusUpdate, "机器人状态更新")
	return nil
}

// handlePing 处理机器人心跳响应
func (s *RobotService) handlePing(message models.WebSocketMessage) error {
	status := s.robot.GetStatus()
	if status != nil {
		status.LastHeartbeat = time.Now()
		// 计算延迟
		if message.Sequence > 0 {
			status.Latency = time.Now().UnixNano() - message.Sequence
		}
		s.robot.UpdateStatus(status)
	}

	s.emitEvent(models.RobotEventHeartbeat, "机器人心跳")
	return nil
}
