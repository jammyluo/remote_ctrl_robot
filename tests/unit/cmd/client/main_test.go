package main

import (
	"testing"
	"time"

	"remote-ctrl-robot/internal/models"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func init() {
	// 在测试中禁用日志输出
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestRobotClientSequence(t *testing.T) {
	// 测试序列号生成逻辑
	sequence := int64(0)

	// 模拟序列号生成
	sequence++
	assert.Equal(t, int64(1), sequence)

	sequence++
	assert.Equal(t, int64(2), sequence)

	sequence++
	assert.Equal(t, int64(3), sequence)
}

func TestWebSocketMessageStructure(t *testing.T) {
	// 测试WebSocket消息结构
	msg := models.WebSocketMessage{
		Type:       models.WSMessageTypeRequest,
		Command:    models.CMD_TYPE_REGISTER,
		Sequence:   1,
		UCode:      "test_robot",
		ClientType: models.ClientTypeRobot,
		Version:    "1.0.0",
		Data:       map[string]interface{}{"test": "data"},
	}

	assert.Equal(t, models.WSMessageTypeRequest, msg.Type)
	assert.Equal(t, models.CMD_TYPE_REGISTER, msg.Command)
	assert.Equal(t, "test_robot", msg.UCode)
	assert.Equal(t, models.ClientTypeRobot, msg.ClientType)
}

func TestRobotStateStructure(t *testing.T) {
	// 测试机器人状态结构
	state := models.RobotState{
		BasePosition:    [3]float64{1.0, 2.0, 0.0},
		BaseOrientation: [4]float64{0, 0, 0, 1},
		BatteryLevel:    85.5,
		Temperature:     25.0,
		Status:          "idle",
		ErrorCode:       0,
		ErrorMessage:    "",
	}

	assert.Equal(t, 85.5, state.BatteryLevel)
	assert.Equal(t, "idle", state.Status)
	assert.Equal(t, 0, state.ErrorCode)
}

func TestControlCommandStructure(t *testing.T) {
	// 测试控制命令结构
	control := models.CMD_CONTROL_ROBOT{
		Action: "move",
		ParamMaps: map[string]string{
			"direction": "forward",
			"speed":     "50",
		},
		Timestamp: time.Now().Unix(),
	}

	assert.Equal(t, "move", control.Action)
	assert.Equal(t, "forward", control.ParamMaps["direction"])
	assert.Equal(t, "50", control.ParamMaps["speed"])
	assert.Greater(t, control.Timestamp, int64(0))
}

func TestConstants(t *testing.T) {
	// 测试常量定义
	assert.Equal(t, "robot", string(models.ClientTypeRobot))
	assert.Equal(t, "operator", string(models.ClientTypeOperator))
	assert.Equal(t, "Request", string(models.WSMessageTypeRequest))
	assert.Equal(t, "Response", string(models.WSMessageTypeResponse))
	assert.Equal(t, "CMD_REGISTER", string(models.CMD_TYPE_REGISTER))
	assert.Equal(t, "CMD_PING", string(models.CMD_TYPE_PING))
}

func TestConnectionStatusStructure(t *testing.T) {
	// 测试连接状态结构
	status := models.ConnectionStatus{
		Connected:       true,
		LastHeartbeat:   time.Now(),
		Latency:         50,
		ActiveClients:   5,
		TotalCommands:   100,
		FailedCommands:  2,
		LastCommandTime: time.Now(),
	}

	assert.True(t, status.Connected)
	assert.Equal(t, int64(50), status.Latency)
	assert.Equal(t, int64(5), status.ActiveClients)
	assert.Equal(t, int64(100), status.TotalCommands)
	assert.Equal(t, int64(2), status.FailedCommands)
}
