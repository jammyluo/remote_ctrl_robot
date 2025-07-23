package robot

import (
	"remote-ctrl-robot/cmd/client/config"
	"remote-ctrl-robot/internal/models"
	"sync"
)

// 机器人状态结构
type RobotState struct {
	BasePosition    [3]float64 `json:"base_position"`
	BaseOrientation [4]float64 `json:"base_orientation"`
	BatteryLevel    float64    `json:"battery_level"`
	Temperature     float64    `json:"temperature"`
	Status          string     `json:"status"`
	ErrorCode       int        `json:"error_code"`
	ErrorMessage    string     `json:"error_message"`
}

// RobotInterface 机器人接口定义
type RobotInterface interface {
	// 基础生命周期方法
	Start() error
	Stop() error
	IsRunning() bool

	// 消息处理方法
	HandleMessage(msg *models.WebSocketMessage) error

	// 状态获取方法
	GetRobotState() RobotState
	GetBatteryLevel() float64
	GetTemperature() float64
	GetPosition() [3]float64
	GetOrientation() [4]float64
	GetStatus() string
	GetErrorInfo() (int, string)

	// 控制方法
	ExecuteCommand(command string, params map[string]interface{}) error
	EmergencyStop() error

	// 统计信息
	GetStats() map[string]interface{}
}

// BaseRobotClient 基础机器人客户端实现
type BaseRobotClient struct {
	config *config.Config
	done   chan struct{}

	// 并发安全
	seqMutex sync.Mutex
}

// NewBaseRobotClient 创建基础机器人客户端
func NewBaseRobotClient(config *config.Config) *BaseRobotClient {
	client := &BaseRobotClient{
		config: config,
		done:   make(chan struct{}),
	}

	return client
}

// Start 启动基础客户端
func (r *BaseRobotClient) Start() error {
	return nil
}

// Stop 停止基础客户端
func (r *BaseRobotClient) Stop() error {
	close(r.done)
	return nil
}

// IsRunning 检查是否正在运行
func (r *BaseRobotClient) IsRunning() bool {
	return true
}

// GetStats 获取基础统计信息
func (r *BaseRobotClient) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"ucode": r.config.Robot.UCode,
	}
}
