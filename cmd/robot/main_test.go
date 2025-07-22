package main

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func init() {
	// 在测试中禁用日志输出
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestNewRobotClient(t *testing.T) {
	config := &Config{
		Robot: RobotConfig{
			UCode:      "test_robot",
			Name:       "测试机器人",
			Version:    "1.0.0",
			ClientType: "robot",
		},
		Server: ServerConfig{
			URL: "ws://localhost:8000/ws/control",
		},
	}

	client := NewRobotClient(config)

	if client.config.Robot.UCode != "test_robot" {
		t.Errorf("期望 ucode 为 'test_robot', 实际为 '%s'", client.config.Robot.UCode)
	}

	if client.config.Server.URL != "ws://localhost:8000/ws/control" {
		t.Errorf("期望 serverURL 为 'ws://localhost:8000/ws/control', 实际为 '%s'", client.config.Server.URL)
	}

	if client.sequence != 1 {
		t.Errorf("期望 sequence 为 1, 实际为 %d", client.sequence)
	}

	if client.connected != false {
		t.Errorf("期望 connected 为 false, 实际为 %v", client.connected)
	}

	// 测试重连相关字段
	if client.reconnectAttempts != 0 {
		t.Errorf("期望 reconnectAttempts 为 0, 实际为 %d", client.reconnectAttempts)
	}
}

func TestRobotClientStop(t *testing.T) {
	config := &Config{
		Robot: RobotConfig{
			UCode: "test_robot",
		},
	}

	client := NewRobotClient(config)

	// 测试停止功能
	client.Stop()

	if client.connected != false {
		t.Errorf("期望 connected 为 false, 实际为 %v", client.connected)
	}
}

func TestMessageStructures(t *testing.T) {
	// 测试消息结构
	msg := WebSocketMessage{
		Type:       WSMessageTypeRequest,
		Command:    CMD_TYPE_REGISTER,
		Sequence:   1,
		UCode:      "test_robot",
		ClientType: ClientTypeRobot,
		Version:    "1.0.0",
		Data:       map[string]interface{}{},
	}

	if msg.Type != WSMessageTypeRequest {
		t.Errorf("期望 Type 为 WSMessageTypeRequest, 实际为 %s", msg.Type)
	}

	if msg.Command != CMD_TYPE_REGISTER {
		t.Errorf("期望 Command 为 CMD_TYPE_REGISTER, 实际为 %s", msg.Command)
	}

	if msg.UCode != "test_robot" {
		t.Errorf("期望 UCode 为 'test_robot', 实际为 '%s'", msg.UCode)
	}
}

func TestRobotState(t *testing.T) {
	// 测试机器人状态结构
	state := RobotState{
		BasePosition:    [3]float64{1.0, 2.0, 0.0},
		BaseOrientation: [4]float64{0, 0, 0, 1},
		BatteryLevel:    85.5,
		Temperature:     25.3,
		Status:          "idle",
		ErrorCode:       0,
		ErrorMessage:    "",
	}

	if state.BatteryLevel != 85.5 {
		t.Errorf("期望 BatteryLevel 为 85.5, 实际为 %f", state.BatteryLevel)
	}

	if state.Temperature != 25.3 {
		t.Errorf("期望 Temperature 为 25.3, 实际为 %f", state.Temperature)
	}

	if state.Status != "idle" {
		t.Errorf("期望 Status 为 'idle', 实际为 '%s'", state.Status)
	}
}

func TestConstants(t *testing.T) {
	// 测试常量定义
	if ClientTypeRobot != "robot" {
		t.Errorf("期望 ClientTypeRobot 为 'robot', 实际为 '%s'", ClientTypeRobot)
	}

	if WSMessageTypeRequest != "Request" {
		t.Errorf("期望 WSMessageTypeRequest 为 'Request', 实际为 '%s'", WSMessageTypeRequest)
	}

	if CMD_TYPE_REGISTER != "CMD_REGISTER" {
		t.Errorf("期望 CMD_TYPE_REGISTER 为 'CMD_REGISTER', 实际为 '%s'", CMD_TYPE_REGISTER)
	}

	if CMD_TYPE_PING != "CMD_PING" {
		t.Errorf("期望 CMD_TYPE_PING 为 'CMD_PING', 实际为 '%s'", CMD_TYPE_PING)
	}

	if CMD_TYPE_UPDATE_ROBOT_STATUS != "CMD_UPDATE_ROBOT_STATUS" {
		t.Errorf("期望 CMD_TYPE_UPDATE_ROBOT_STATUS 为 'CMD_UPDATE_ROBOT_STATUS', 实际为 '%s'", CMD_TYPE_UPDATE_ROBOT_STATUS)
	}
}

func TestConfigMethods(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			ConnectTimeout: 30,
			ReadTimeout:    30,
			WriteTimeout:   10,
		},
		Heartbeat: HeartbeatConfig{
			Interval: 30,
		},
		Status: StatusConfig{
			Interval: 10,
		},
	}

	// 测试超时方法
	if config.GetConnectTimeout() != 30*time.Second {
		t.Errorf("期望连接超时为 30秒, 实际为 %v", config.GetConnectTimeout())
	}

	if config.GetReadTimeout() != 30*time.Second {
		t.Errorf("期望读取超时为 30秒, 实际为 %v", config.GetReadTimeout())
	}

	if config.GetWriteTimeout() != 10*time.Second {
		t.Errorf("期望写入超时为 10秒, 实际为 %v", config.GetWriteTimeout())
	}

	if config.GetHeartbeatInterval() != 30*time.Second {
		t.Errorf("期望心跳间隔为 30秒, 实际为 %v", config.GetHeartbeatInterval())
	}

	if config.GetStatusInterval() != 10*time.Second {
		t.Errorf("期望状态间隔为 10秒, 实际为 %v", config.GetStatusInterval())
	}
}

func TestPowFunction(t *testing.T) {
	// 测试幂次计算函数
	testCases := []struct {
		base     float64
		exponent int
		expected float64
	}{
		{2.0, 0, 1.0},
		{2.0, 1, 2.0},
		{2.0, 2, 4.0},
		{2.0, 3, 8.0},
		{1.5, 2, 2.25},
		{1.0, 5, 1.0},
	}

	for _, tc := range testCases {
		result := pow(tc.base, tc.exponent)
		if result != tc.expected {
			t.Errorf("pow(%f, %d) = %f, 期望 %f", tc.base, tc.exponent, result, tc.expected)
		}
	}
}

func TestReconnectConfig(t *testing.T) {
	config := &Config{
		Reconnect: ReconnectConfig{
			Enabled:           true,
			MaxAttempts:       5,
			InitialDelay:      1,
			MaxDelay:          60,
			BackoffMultiplier: 2.0,
		},
	}

	client := NewRobotClient(config)

	// 测试重连配置
	if !client.config.Reconnect.Enabled {
		t.Errorf("期望重连启用, 实际为禁用")
	}

	if client.config.Reconnect.MaxAttempts != 5 {
		t.Errorf("期望最大重连次数为 5, 实际为 %d", client.config.Reconnect.MaxAttempts)
	}

	if client.config.Reconnect.InitialDelay != 1 {
		t.Errorf("期望初始重连延迟为 1秒, 实际为 %d", client.config.Reconnect.InitialDelay)
	}

	if client.config.Reconnect.MaxDelay != 60 {
		t.Errorf("期望最大重连延迟为 60秒, 实际为 %d", client.config.Reconnect.MaxDelay)
	}

	if client.config.Reconnect.BackoffMultiplier != 2.0 {
		t.Errorf("期望退避倍数为 2.0, 实际为 %f", client.config.Reconnect.BackoffMultiplier)
	}
}

func TestReconnectDelayCalculation(t *testing.T) {
	config := &Config{
		Reconnect: ReconnectConfig{
			Enabled:           true,
			MaxAttempts:       5,
			InitialDelay:      1,
			MaxDelay:          60,
			BackoffMultiplier: 2.0,
		},
	}

	client := NewRobotClient(config)

	// 测试重连延迟计算
	delay1 := client.calculateReconnectDelay()
	if delay1 < time.Second || delay1 > 2*time.Second {
		t.Errorf("第一次重连延迟应该在1-2秒之间, 实际为 %v", delay1)
	}

	// 模拟重连尝试
	client.reconnectAttempts = 1
	delay2 := client.calculateReconnectDelay()
	if delay2 < 1500*time.Millisecond || delay2 > 3*time.Second {
		t.Errorf("第二次重连延迟应该在1.5-3秒之间, 实际为 %v", delay2)
	}

	// 测试最大延迟限制
	client.reconnectAttempts = 10
	delay3 := client.calculateReconnectDelay()
	if delay3 > 60*time.Second {
		t.Errorf("重连延迟不应该超过60秒, 实际为 %v", delay3)
	}
}
