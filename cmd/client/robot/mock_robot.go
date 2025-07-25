package robot

import (
	"remote-ctrl-robot/cmd/client/config"

	"github.com/rs/zerolog/log"
)

// SimpleRobot 简单机器人实现
type MockRobot struct {
	*BaseRobotClient
}

// NewMockRobot 创建新的简单机器人
func NewMockRobot(config *config.Config) *MockRobot {
	robot := &MockRobot{
		BaseRobotClient: NewBaseRobotClient(config),
	}

	robot.state.Ammo.Count = config.MockRobot.AmmoCount
	robot.state.Ammo.CurrentCapacity = config.MockRobot.AmmoCapacity
	robot.state.Ammo.MaxCapacity = config.MockRobot.AmmoCapacity
	robot.state.Health = config.MockRobot.Health

	return robot
}

// Start 启动机器人
func (r *MockRobot) Start() error {
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
func (r *MockRobot) Stop() error {
	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Msg("Stopping simple robot")

	return r.BaseRobotClient.Stop()
}

func (r *MockRobot) Shoot() error {
	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Msg("Shooting")

	return r.BaseRobotClient.Shoot()
}
