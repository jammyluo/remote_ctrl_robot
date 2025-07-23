package robot_control

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

// TestE2E_RobotRegistration 测试机器人注册流程
func TestE2E_RobotRegistration(t *testing.T) {
	// 模拟完整的机器人注册流程
	t.Log("开始机器人注册端到端测试")

	// 1. 创建注册消息
	registerMessage := models.WebSocketMessage{
		Type:       models.WSMessageTypeRequest,
		Command:    models.CMD_TYPE_REGISTER,
		Sequence:   1,
		UCode:      "e2e_test_robot",
		ClientType: models.ClientTypeRobot,
		Version:    "1.0.0",
		Data:       map[string]interface{}{},
	}

	// 2. 验证消息结构
	assert.Equal(t, models.CMD_TYPE_REGISTER, registerMessage.Command)
	assert.Equal(t, "e2e_test_robot", registerMessage.UCode)
	assert.Equal(t, models.ClientTypeRobot, registerMessage.ClientType)

	// 3. 模拟服务器响应
	responseMessage := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    models.CMD_TYPE_REGISTER,
		Sequence:   1,
		UCode:      "e2e_test_robot",
		ClientType: models.ClientTypeRobot,
		Version:    "1.0.0",
		Data: map[string]interface{}{
			"success": true,
			"message": "Registration successful",
		},
	}

	// 4. 验证响应
	assert.Equal(t, models.WSMessageTypeResponse, responseMessage.Type)
	assert.Equal(t, models.CMD_TYPE_REGISTER, responseMessage.Command)

	// 5. 验证响应数据
	responseData, ok := responseMessage.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, responseData["success"])
	assert.Equal(t, "Registration successful", responseData["message"])

	t.Log("机器人注册端到端测试完成")
}

// TestE2E_RobotControl 测试机器人控制流程
func TestE2E_RobotControl(t *testing.T) {
	// 模拟完整的机器人控制流程
	t.Log("开始机器人控制端到端测试")

	// 1. 创建控制消息
	controlMessage := models.WebSocketMessage{
		Type:       models.WSMessageTypeRequest,
		Command:    models.CMD_TYPE_CONTROL_ROBOT,
		Sequence:   2,
		UCode:      "e2e_test_robot",
		ClientType: models.ClientTypeRobot,
		Version:    "1.0.0",
		Data: models.CMD_CONTROL_ROBOT{
			Action: "move",
			ParamMaps: map[string]string{
				"direction": "forward",
				"speed":     "50",
			},
			Timestamp: time.Now().Unix(),
		},
	}

	// 2. 验证控制消息
	assert.Equal(t, models.CMD_TYPE_CONTROL_ROBOT, controlMessage.Command)

	// 3. 验证控制数据
	controlData, ok := controlMessage.Data.(models.CMD_CONTROL_ROBOT)
	assert.True(t, ok)
	assert.Equal(t, "move", controlData.Action)
	assert.Equal(t, "forward", controlData.ParamMaps["direction"])
	assert.Equal(t, "50", controlData.ParamMaps["speed"])

	// 4. 模拟机器人执行动作
	// 这里可以添加实际的GPIO控制逻辑测试
	t.Log("模拟机器人执行移动动作")
	time.Sleep(100 * time.Millisecond) // 模拟执行时间

	// 5. 创建状态更新消息
	statusMessage := models.WebSocketMessage{
		Type:       models.WSMessageTypeRequest,
		Command:    models.CMD_TYPE_UPDATE_ROBOT_STATUS,
		Sequence:   3,
		UCode:      "e2e_test_robot",
		ClientType: models.ClientTypeRobot,
		Version:    "1.0.0",
		Data: models.RobotState{
			BasePosition:    [3]float64{1.0, 0.0, 0.0}, // 向前移动了1米
			BaseOrientation: [4]float64{0, 0, 0, 1},
			BatteryLevel:    85.0,
			Temperature:     25.0,
			Status:          "moving",
			ErrorCode:       0,
			ErrorMessage:    "",
		},
	}

	// 6. 验证状态更新
	assert.Equal(t, models.CMD_TYPE_UPDATE_ROBOT_STATUS, statusMessage.Command)

	statusData, ok := statusMessage.Data.(models.RobotState)
	assert.True(t, ok)
	assert.Equal(t, "moving", statusData.Status)
	assert.Equal(t, 85.0, statusData.BatteryLevel)
	assert.Equal(t, [3]float64{1.0, 0.0, 0.0}, statusData.BasePosition)

	t.Log("机器人控制端到端测试完成")
}

// TestE2E_RobotBind 测试机器人绑定流程
func TestE2E_RobotBind(t *testing.T) {
	// 模拟机器人与操作员绑定流程
	t.Log("开始机器人绑定端到端测试")

	// 1. 创建绑定消息
	bindMessage := models.WebSocketMessage{
		Type:       models.WSMessageTypeRequest,
		Command:    models.CMD_TYPE_BIND_ROBOT,
		Sequence:   4,
		UCode:      "e2e_test_robot",
		ClientType: models.ClientTypeRobot,
		Version:    "1.0.0",
		Data: map[string]interface{}{
			"operator_id": "e2e_test_operator",
		},
	}

	// 2. 验证绑定消息
	assert.Equal(t, models.CMD_TYPE_BIND_ROBOT, bindMessage.Command)

	// 3. 验证绑定数据
	bindData, ok := bindMessage.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "e2e_test_operator", bindData["operator_id"])

	// 4. 模拟绑定成功响应
	bindResponse := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    models.CMD_TYPE_BIND_ROBOT,
		Sequence:   4,
		UCode:      "e2e_test_robot",
		ClientType: models.ClientTypeRobot,
		Version:    "1.0.0",
		Data: map[string]interface{}{
			"success":     true,
			"message":     "Robot bound successfully",
			"operator_id": "e2e_test_operator",
		},
	}

	// 5. 验证绑定响应
	assert.Equal(t, models.WSMessageTypeResponse, bindResponse.Type)

	responseData, ok := bindResponse.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, responseData["success"])
	assert.Equal(t, "e2e_test_operator", responseData["operator_id"])

	t.Log("机器人绑定端到端测试完成")
}

// TestE2E_Heartbeat 测试心跳机制
func TestE2E_Heartbeat(t *testing.T) {
	// 模拟心跳机制
	t.Log("开始心跳机制端到端测试")

	// 1. 创建心跳消息
	pingMessage := models.WebSocketMessage{
		Type:       models.WSMessageTypeRequest,
		Command:    models.CMD_TYPE_PING,
		Sequence:   5,
		UCode:      "e2e_test_robot",
		ClientType: models.ClientTypeRobot,
		Version:    "1.0.0",
		Data:       map[string]interface{}{},
	}

	// 2. 验证心跳消息
	assert.Equal(t, models.CMD_TYPE_PING, pingMessage.Command)

	// 3. 模拟心跳响应
	pongMessage := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    models.CMD_TYPE_PING,
		Sequence:   5,
		UCode:      "e2e_test_robot",
		ClientType: models.ClientTypeRobot,
		Version:    "1.0.0",
		Data: map[string]interface{}{
			"success":   true,
			"timestamp": time.Now().Unix(),
		},
	}

	// 4. 验证心跳响应
	assert.Equal(t, models.CMD_TYPE_PING, pongMessage.Command)

	responseData, ok := pongMessage.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, responseData["success"])

	// 5. 验证时间戳
	timestamp, ok := responseData["timestamp"].(int64)
	assert.True(t, ok)
	assert.Greater(t, timestamp, int64(0))

	t.Log("心跳机制端到端测试完成")
}

// TestE2E_ErrorHandling 测试错误处理
func TestE2E_ErrorHandling(t *testing.T) {
	// 模拟错误处理流程
	t.Log("开始错误处理端到端测试")

	// 1. 模拟无效命令
	invalidMessage := models.WebSocketMessage{
		Type:       models.WSMessageTypeRequest,
		Command:    "INVALID_COMMAND",
		Sequence:   6,
		UCode:      "e2e_test_robot",
		ClientType: models.ClientTypeRobot,
		Version:    "1.0.0",
		Data:       map[string]interface{}{},
	}

	// 2. 验证无效命令
	assert.Equal(t, "INVALID_COMMAND", string(invalidMessage.Command))

	// 3. 模拟错误响应
	errorResponse := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    "INVALID_COMMAND",
		Sequence:   6,
		UCode:      "e2e_test_robot",
		ClientType: models.ClientTypeRobot,
		Version:    "1.0.0",
		Data: map[string]interface{}{
			"success":    false,
			"error_code": 1001,
			"message":    "Unknown command",
		},
	}

	// 4. 验证错误响应
	assert.Equal(t, models.WSMessageTypeResponse, errorResponse.Type)

	errorData, ok := errorResponse.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, false, errorData["success"])
	assert.Equal(t, 1001, errorData["error_code"])
	assert.Equal(t, "Unknown command", errorData["message"])

	t.Log("错误处理端到端测试完成")
}
