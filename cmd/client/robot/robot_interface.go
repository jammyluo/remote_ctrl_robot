package robot

import (
	"fmt"
	"remote-ctrl-robot/cmd/client/config"
	"sync"
)

type Ammo struct {
	CurrentCapacity int    `json:"current_capacity"`
	MaxCapacity     int    `json:"max_capacity"`
	Count           int    `json:"count"`
	Type            string `json:"type"`
	State           string `json:"state"`
}

// 机器人状态结构
type state struct {
	BasePosition    [3]float64 `json:"base_position"`
	BaseOrientation [4]float64 `json:"base_orientation"`
	BatteryLevel    float64    `json:"battery_level"`
	Temperature     float64    `json:"temperature"`
	Health          int        `json:"health"`
	Ammo            Ammo       `json:"ammo"`
}

// RobotInterface 机器人接口定义
type RobotInterface interface {
	// 基础生命周期方法
	Start() error
	Stop() error
	IsRunning() bool

	// 状态获取方法
	Shoot() error
	GetAmmo() Ammo
	AmmoChange()
	GetHealth() int
	GetBatteryLevel() float64
	GetTemperature() float64
	GetPosition() [3]float64
	GetOrientation() [4]float64
	GetState() state

	// 控制方法
	// ExecuteCommand(command string, params map[string]interface{}) error
	// EmergencyStop() error
	// GetErrorInfo() (int, string)
	// 消息处理方法
	// HandleMessage(msg *models.WebSocketMessage) error
}

// BaseRobotClient 基础机器人客户端实现
type BaseRobotClient struct {
	config *config.Config
	state  state

	// 并发安全
	seqMutex sync.Mutex
}

func (r *BaseRobotClient) GetState() state {
	return r.state
}

func (r *BaseRobotClient) GetAmmo() Ammo {
	return r.state.Ammo
}

func (r *BaseRobotClient) Shoot() error {
	r.seqMutex.Lock()
	defer r.seqMutex.Unlock()

	if r.state.Ammo.Count > 0 && r.state.Ammo.CurrentCapacity > 0 {
		r.state.Ammo.Count = r.state.Ammo.Count - 1
		r.state.Ammo.CurrentCapacity = r.state.Ammo.CurrentCapacity - 1
	} else {
		return fmt.Errorf("no ammo")
	}

	return nil
}

func (r *BaseRobotClient) AmmoChange() {
	r.seqMutex.Lock()
	defer r.seqMutex.Unlock()

	r.state.Ammo.CurrentCapacity = r.state.Ammo.MaxCapacity
	r.state.Ammo.State = "ready"
}

func (r *BaseRobotClient) GetHealth() int {
	return r.state.Health
}

// NewBaseRobotClient 创建基础机器人客户端
func NewBaseRobotClient(config *config.Config) *BaseRobotClient {
	client := &BaseRobotClient{
		config: config,
		state: state{
			BasePosition:    [3]float64{0, 0, 0},
			BaseOrientation: [4]float64{0, 0, 0, 1},
			BatteryLevel:    100.0,
			Temperature:     25.0,
			Health:          100,
			Ammo: Ammo{
				CurrentCapacity: 30,
				MaxCapacity:     30,
				Count:           100,
				Type:            "bullet",
				State:           "ready",
			},
		},
	}

	return client
}

// Start 启动基础客户端
func (r *BaseRobotClient) Start() error {
	return nil
}

// Stop 停止基础客户端
func (r *BaseRobotClient) Stop() error {
	return nil
}

// IsRunning 检查是否正在运行
func (r *BaseRobotClient) IsRunning() bool {
	return true
}

// GetBatteryLevel 获取电池电量
func (r *BaseRobotClient) GetBatteryLevel() float64 {
	return r.state.BatteryLevel
}

// GetTemperature 获取温度
func (r *BaseRobotClient) GetTemperature() float64 {
	return r.state.Temperature
}

// GetPosition 获取位置
func (r *BaseRobotClient) GetPosition() [3]float64 {
	return r.state.BasePosition
}

// GetOrientation 获取方向
func (r *BaseRobotClient) GetOrientation() [4]float64 {
	return r.state.BaseOrientation
}
