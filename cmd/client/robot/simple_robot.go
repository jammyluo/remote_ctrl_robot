package robot

import (
	"fmt"

	"remote-ctrl-robot/cmd/client/config"
	"github.com/rs/zerolog/log"
)

// SimpleRobot 简单机器人实现
type SimpleRobot struct {
	*BaseRobotClient
	state RobotState
}

// NewSimpleRobot 创建新的简单机器人
func NewSimpleRobot(config *config.Config) *SimpleRobot {
	robot := &SimpleRobot{
		BaseRobotClient: NewBaseRobotClient(config),
		state: RobotState{
			BasePosition:    [3]float64{0, 0, 0},
			BaseOrientation: [4]float64{0, 0, 0, 1},
			BatteryLevel:    100.0,
			Temperature:     25.0,
			Status:          "idle",
			ErrorCode:       0,
			ErrorMessage:    "",
		},
	}

	return robot
}

// Start 启动机器人
func (r *SimpleRobot) Start() error {
	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Str("server", r.config.WebSocket.URL).
		Msg("Starting simple robot")

	// 启动基础客户端
	if err := r.BaseRobotClient.Start(); err != nil {
		return err
	}

	return nil
}

// Stop 停止机器人
func (r *SimpleRobot) Stop() error {
	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Msg("Stopping simple robot")

	return r.BaseRobotClient.Stop()
}

// HandleMessage 处理接收到的消息
func (r *SimpleRobot) HandleMessage(message []byte) error {
	// var msg WebSocketMessage
	// if err := json.Unmarshal(message, &msg); err != nil {
	// 	return fmt.Errorf("parse message failed: %v", err)
	// }

	// log.Debug().
	// 	Str("ucode", r.config.Robot.UCode).
	// 	Str("command", string(msg.Command)).
	// 	Int64("sequence", msg.Sequence).
	// 	Msg("Received message")

	// // 根据命令类型处理消息
	// switch msg.Command {
	// case CMD_TYPE_PING:
	// 	// 心跳消息，无需特殊处理
	// 	return nil
	// default:
	// 	log.Debug().
	// 		Str("ucode", r.config.Robot.UCode).
	// 		Str("command", string(msg.Command)).
	// 		Msg("Unknown command, ignoring")
	// 	return nil
	// }
	return nil
}

// GetRobotState 获取机器人状态
func (r *SimpleRobot) GetRobotState() RobotState {
	return r.state
}

// GetBatteryLevel 获取电池电量
func (r *SimpleRobot) GetBatteryLevel() float64 {
	return r.state.BatteryLevel
}

// GetTemperature 获取温度
func (r *SimpleRobot) GetTemperature() float64 {
	return r.state.Temperature
}

// GetPosition 获取位置
func (r *SimpleRobot) GetPosition() [3]float64 {
	return r.state.BasePosition
}

// GetOrientation 获取方向
func (r *SimpleRobot) GetOrientation() [4]float64 {
	return r.state.BaseOrientation
}

// GetStatus 获取状态
func (r *SimpleRobot) GetStatus() string {
	return r.state.Status
}

// GetErrorInfo 获取错误信息
func (r *SimpleRobot) GetErrorInfo() (int, string) {
	return r.state.ErrorCode, r.state.ErrorMessage
}

// ExecuteCommand 执行命令
func (r *SimpleRobot) ExecuteCommand(command string, params map[string]interface{}) error {
	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Str("command", command).
		Interface("params", params).
		Msg("Executing command")

	// 这里可以添加具体的命令执行逻辑
	switch command {
	case "move":
		// 移动命令
		r.state.Status = "moving"
		return nil
	case "stop":
		// 停止命令
		r.state.Status = "idle"
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// EmergencyStop 紧急停止
func (r *SimpleRobot) EmergencyStop() error {
	log.Warn().
		Str("ucode", r.config.Robot.UCode).
		Msg("Emergency stop triggered")

	r.state.Status = "emergency_stop"
	r.state.ErrorCode = 1
	r.state.ErrorMessage = "Emergency stop activated"

	return nil
}