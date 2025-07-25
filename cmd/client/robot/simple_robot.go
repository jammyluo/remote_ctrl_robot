package robot

import (
	"fmt"
	"time"

	"remote-ctrl-robot/cmd/client/config"
	"remote-ctrl-robot/internal/gpio"
	"remote-ctrl-robot/internal/models"

	"github.com/rs/zerolog/log"
)

// SimpleRobot 简单机器人实现
type SimpleRobot struct {
	*BaseRobotClient
}

// NewSimpleRobot 创建新的简单机器人
func NewSimpleRobot(config *config.Config) *SimpleRobot {
	robot := &SimpleRobot{
		BaseRobotClient: NewBaseRobotClient(config),
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
func (r *SimpleRobot) HandleMessage(msg *models.WebSocketMessage) error {
	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Str("command", string(msg.Command)).
		Msg("HandleMessage")

	if msg.Command == models.CMD_TYPE_CONTROL_ROBOT {
		// 处理控制机器人命令
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if action, exists := data["action"].(string); exists {
				switch action {
				case "shoot":
					log.Info().
						Str("ucode", r.config.Robot.UCode).
						Msg("Shoot command received")
					// 创建GPIO控制器并演示高低电平控制
					gpioCtrl := gpio.NewGPIOController(27)
					gpioCtrl.Pulse(time.Millisecond * time.Duration(70))
				case "move":
					log.Info().
						Str("ucode", r.config.Robot.UCode).
						Msg("Move command received")
					// 处理移动命令
					if direction, exists := data["direction"].(string); exists {
						log.Info().
							Str("ucode", r.config.Robot.UCode).
							Str("direction", direction).
							Msg("Moving robot")
					}
				case "stop":
					log.Info().
						Str("ucode", r.config.Robot.UCode).
						Msg("Stop command received")
					// 处理停止命令
				default:
					log.Warn().
						Str("ucode", r.config.Robot.UCode).
						Str("action", action).
						Msg("Unknown action")
				}
			}
		} else {
			log.Error().
				Str("ucode", r.config.Robot.UCode).
				Msg("Invalid data format for control command")
		}
	}

	return nil
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
		return nil
	case "stop":
		// 停止命令
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

	return nil
}
