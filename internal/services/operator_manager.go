package services

import (
	"fmt"
	"sync"
	"time"

	"remote-ctrl-robot/internal/models"

	"github.com/rs/zerolog/log"
)

type Operator struct {
	UCode      string            // 唯一标识
	RobotUCode string            // 机器人唯一标识
	wsService  *WebSocketService // 机器人连接
	Version    string            // 客户端版本
	CreatedAt  time.Time         `json:"created_at"` // 创建时间
	UpdatedAt  time.Time         `json:"updated_at"` // 更新时间
	mutex      sync.RWMutex      // 状态锁
}

// OperatorManager 客户端管理器
type OperatorManager struct {
	operators    map[string]*Operator // 客户端映射表
	mutex        sync.RWMutex
	robotManager *RobotManager
	gameService  *GameService
}

// NewOperatorManager 创建新的客户端管理器
func NewOperatorManager(robotManager *RobotManager, gameService *GameService) *OperatorManager {
	return &OperatorManager{
		operators:    make(map[string]*Operator),
		robotManager: robotManager,
		gameService:  gameService,
	}
}

// AddClient 添加客户端
func (cm *OperatorManager) RegisterOperator(ucode string, wsService *WebSocketService) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 检查客户端是否已存在
	if operator, exists := cm.operators[ucode]; exists {
		// 更新客户端信息
		operator.wsService = wsService
		log.Info().
			Str("ucode", operator.UCode).
			Msg("Operator reconnected")
	} else {
		// 创建新客户端
		operator := &Operator{
			UCode:     ucode,
			wsService: wsService,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		cm.operators[operator.UCode] = operator
		log.Info().
			Str("ucode", operator.UCode).
			Msg("Operator registered")
	}

	return nil
}

// GetOperator 获取操作员
func (cm *OperatorManager) GetOperator(ucode string) (*Operator, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	operator, exists := cm.operators[ucode]
	if !exists {
		return nil, fmt.Errorf("operator %s not found", ucode)
	}

	return operator, nil
}

// GetAllOperators 获取所有操作员
func (cm *OperatorManager) GetAllOperators() []*Operator {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	operators := make([]*Operator, 0, len(cm.operators))
	for _, operator := range cm.operators {
		operators = append(operators, operator)
	}

	return operators
}

// GetConnectedOperators 获取已连接的操作员
func (cm *OperatorManager) GetConnectedOperators() []*Operator {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var connectedOperators []*Operator
	for _, operator := range cm.operators {
		if operator.wsService.IsConnected() {
			connectedOperators = append(connectedOperators, operator)
		}
	}

	return connectedOperators
}

// GetClientCount 获取客户端数量
func (cm *OperatorManager) GetClientCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return len(cm.operators)
}

// GetConnectedClientCount 获取已连接客户端数量
func (cm *OperatorManager) GetConnectedClientCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	count := 0
	for _, operator := range cm.operators {
		if operator.wsService.IsConnected() {
			count++
		}
	}
	return count
}

// CleanupDisconnectedClients 清理断开的客户端
func (cm *OperatorManager) CleanupDisconnectedClients() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	var toRemove []*Operator

	for _, operator := range cm.operators {
		// 检查最后活跃时间，超过5分钟认为断开
		if now.Sub(operator.wsService.LastSeen) > 5*time.Minute {
			toRemove = append(toRemove, operator)
		}
	}

	for _, operator := range toRemove {
		// 关闭连接
		delete(cm.operators, operator.UCode)
		log.Info().
			Str("ucode", operator.UCode).
			Str("version", operator.Version).
			Msg("Disconnected client cleaned up")
	}
}

// handleMessage 处理来自客户端的消息（客户端消息，与机器人消息不同）
func (s *OperatorManager) HandleMessage(message *models.WebSocketMessage) error {
	operator, err := s.GetOperator(message.UCode)
	if err != nil {
		return err
	}
	operator.wsService.LastSeen = time.Now()

	log.Debug().
		Str("ucode", message.UCode).
		Int64("sequence", message.Sequence).
		Str("remote_addr", operator.wsService.RemoteAddr).
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
func (s *OperatorManager) handleBindRobot(operator *Operator, message *models.WebSocketMessage) error {
	var bindData models.CMD_BIND_ROBOT
	if data, ok := message.Data.(map[string]interface{}); ok {
		if ucode, exists := data["ucode"].(string); exists {
			bindData.UCode = ucode
		}
	}

	// 检查机器人是否存在
	robot, err := s.robotManager.GetRobot(bindData.UCode)
	if err != nil {
		log.Error().Err(err).Str("ucode", bindData.UCode).Msg("机器人不存在")
		return operator.wsService.SendError(message, "机器人不存在")
	}

	// 检查机器人是否在线
	if !robot.wsService.IsConnected() {
		log.Error().Str("ucode", bindData.UCode).Msg("机器人未连接")
		return operator.wsService.SendError(message, "机器人未连接")
	}

	// 绑定机器人
	operator.RobotUCode = bindData.UCode
	log.Info().
		Str("client_ucode", operator.UCode).
		Str("robot_ucode", bindData.UCode).
		Msg("Client bound to robot")

	return operator.wsService.SendSuccess(message, "机器人绑定成功")
}

// handleControlRobot 处理机器人控制
func (s *OperatorManager) handleControlRobot(operator *Operator, message *models.WebSocketMessage) error {
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

	s.robotManager.SendCommand(operator.RobotUCode, command)

	return operator.wsService.SendSuccess(message, "控制命令发送成功")
}

// handlePing 处理心跳
func (s *OperatorManager) handlePing(operator *Operator, message *models.WebSocketMessage) error {
	operator.wsService.LastSeen = time.Now()

	return operator.wsService.SendSuccess(message, "pong")
}

// 游戏相关处理方法
func (s *OperatorManager) handleJoinGame(operator *Operator, message *models.WebSocketMessage) error {
	// 这里可以添加游戏加入逻辑
	return operator.wsService.SendSuccess(message, "加入游戏成功")
}

func (s *OperatorManager) handleLeaveGame(operator *Operator, message *models.WebSocketMessage) error {
	// 这里可以添加游戏离开逻辑
	return operator.wsService.SendSuccess(message, "离开游戏成功")
}

func (s *OperatorManager) handleGameShoot(operator *Operator, message *models.WebSocketMessage) error {
	// 这里可以添加游戏射击逻辑
	return operator.wsService.SendSuccess(message, "射击命令发送成功")
}

func (s *OperatorManager) handleGameMove(operator *Operator, message *models.WebSocketMessage) error {
	// 这里可以添加游戏移动逻辑
	return operator.wsService.SendSuccess(message, "移动命令发送成功")
}

func (s *OperatorManager) handleGameStatus(operator *Operator, message *models.WebSocketMessage) error {
	// 这里可以添加游戏状态获取逻辑
	return operator.wsService.SendSuccess(message, "游戏状态获取成功")
}

func (s *OperatorManager) handleGameStart(operator *Operator, message *models.WebSocketMessage) error {
	// 这里可以添加游戏开始逻辑
	return operator.wsService.SendSuccess(message, "游戏开始成功")
}

func (s *OperatorManager) handleGameStop(operator *Operator, message *models.WebSocketMessage) error {
	// 这里可以添加游戏停止逻辑
	return operator.wsService.SendSuccess(message, "游戏停止成功")
}
