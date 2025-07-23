package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWebSocketMessage(t *testing.T) {
	// 测试WebSocket消息结构
	msg := WebSocketMessage{
		Type:       WSMessageTypeRequest,
		Command:    CMD_TYPE_REGISTER,
		Sequence:   1,
		UCode:      "test_robot",
		ClientType: ClientTypeRobot,
		Version:    "1.0.0",
		Data:       map[string]interface{}{"test": "data"},
	}

	// 测试序列化
	jsonData, err := json.Marshal(msg)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// 测试反序列化
	var parsedMsg WebSocketMessage
	err = json.Unmarshal(jsonData, &parsedMsg)
	assert.NoError(t, err)
	assert.Equal(t, msg.Type, parsedMsg.Type)
	assert.Equal(t, msg.Command, parsedMsg.Command)
	assert.Equal(t, msg.UCode, parsedMsg.UCode)
}

func TestRobotState(t *testing.T) {
	// 测试机器人状态结构
	state := RobotState{
		BasePosition:    [3]float64{1.0, 2.0, 0.0},
		BaseOrientation: [4]float64{0, 0, 0, 1},
		BatteryLevel:    85.5,
		Temperature:     25.0,
		Status:          "idle",
		ErrorCode:       0,
		ErrorMessage:    "",
	}

	// 测试序列化
	jsonData, err := json.Marshal(state)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// 测试反序列化
	var parsedState RobotState
	err = json.Unmarshal(jsonData, &parsedState)
	assert.NoError(t, err)
	assert.Equal(t, state.BatteryLevel, parsedState.BatteryLevel)
	assert.Equal(t, state.Status, parsedState.Status)
}

func TestConstants(t *testing.T) {
	// 测试常量定义
	assert.Equal(t, "robot", string(ClientTypeRobot))
	assert.Equal(t, "operator", string(ClientTypeOperator))
	assert.Equal(t, "Request", string(WSMessageTypeRequest))
	assert.Equal(t, "Response", string(WSMessageTypeResponse))
	assert.Equal(t, "CMD_REGISTER", string(CMD_TYPE_REGISTER))
	assert.Equal(t, "CMD_PING", string(CMD_TYPE_PING))
}

func TestCMD_CONTROL_ROBOT(t *testing.T) {
	// 测试控制命令结构
	control := CMD_CONTROL_ROBOT{
		Action: "move",
		ParamMaps: map[string]string{
			"direction": "forward",
			"speed":     "50",
		},
		Timestamp: time.Now().Unix(),
	}

	// 测试序列化
	jsonData, err := json.Marshal(control)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// 测试反序列化
	var parsedControl CMD_CONTROL_ROBOT
	err = json.Unmarshal(jsonData, &parsedControl)
	assert.NoError(t, err)
	assert.Equal(t, control.Action, parsedControl.Action)
	assert.Equal(t, control.ParamMaps["direction"], parsedControl.ParamMaps["direction"])
}

func TestConnectionStatus(t *testing.T) {
	// 测试连接状态结构
	status := ConnectionStatus{
		Connected:       true,
		LastHeartbeat:   time.Now(),
		Latency:         50,
		ActiveClients:   5,
		TotalCommands:   100,
		FailedCommands:  2,
		LastCommandTime: time.Now(),
	}

	// 测试序列化
	jsonData, err := json.Marshal(status)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// 测试反序列化
	var parsedStatus ConnectionStatus
	err = json.Unmarshal(jsonData, &parsedStatus)
	assert.NoError(t, err)
	assert.Equal(t, status.Connected, parsedStatus.Connected)
	assert.Equal(t, status.Latency, parsedStatus.Latency)
	assert.Equal(t, status.ActiveClients, parsedStatus.ActiveClients)
}
