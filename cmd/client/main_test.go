package client_test

import (
	"sync"
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

	if client.sequence != 0 {
		t.Errorf("期望 sequence 为 0, 实际为 %d", client.sequence)
	}

	if client.wsService == nil {
		t.Errorf("期望 wsService 不为 nil")
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

	// 验证WebSocketClient已停止
	if client.wsService == nil {
		t.Errorf("期望 wsService 不为 nil")
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
		Temperature:     25.0,
		Status:          "idle",
		ErrorCode:       0,
		ErrorMessage:    "",
	}

	if state.BasePosition[0] != 1.0 {
		t.Errorf("期望 BasePosition[0] 为 1.0, 实际为 %f", state.BasePosition[0])
	}

	if state.BatteryLevel != 85.5 {
		t.Errorf("期望 BatteryLevel 为 85.5, 实际为 %f", state.BatteryLevel)
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
			Timeout:  10,
		},
		Status: StatusConfig{
			Interval: 10,
		},
	}

	// 测试超时方法
	if config.GetConnectTimeout() != 30*time.Second {
		t.Errorf("期望 GetConnectTimeout 为 30秒, 实际为 %v", config.GetConnectTimeout())
	}

	if config.GetReadTimeout() != 30*time.Second {
		t.Errorf("期望 GetReadTimeout 为 30秒, 实际为 %v", config.GetReadTimeout())
	}

	if config.GetWriteTimeout() != 10*time.Second {
		t.Errorf("期望 GetWriteTimeout 为 10秒, 实际为 %v", config.GetWriteTimeout())
	}

	if config.GetHeartbeatInterval() != 30*time.Second {
		t.Errorf("期望 GetHeartbeatInterval 为 30秒, 实际为 %v", config.GetHeartbeatInterval())
	}

	if config.GetStatusInterval() != 10*time.Second {
		t.Errorf("期望 GetStatusInterval 为 10秒, 实际为 %v", config.GetStatusInterval())
	}
}

func TestPowFunction(t *testing.T) {
	// 测试幂次计算函数
	if pow(2, 0) != 1.0 {
		t.Errorf("期望 pow(2, 0) 为 1.0, 实际为 %f", pow(2, 0))
	}

	if pow(2, 1) != 2.0 {
		t.Errorf("期望 pow(2, 1) 为 2.0, 实际为 %f", pow(2, 1))
	}

	if pow(2, 3) != 8.0 {
		t.Errorf("期望 pow(2, 3) 为 8.0, 实际为 %f", pow(2, 3))
	}

	if pow(1.5, 2) != 2.25 {
		t.Errorf("期望 pow(1.5, 2) 为 2.25, 实际为 %f", pow(1.5, 2))
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

	if !config.Reconnect.Enabled {
		t.Error("期望 Reconnect.Enabled 为 true")
	}

	if config.Reconnect.MaxAttempts != 5 {
		t.Errorf("期望 MaxAttempts 为 5, 实际为 %d", config.Reconnect.MaxAttempts)
	}

	if config.Reconnect.BackoffMultiplier != 2.0 {
		t.Errorf("期望 BackoffMultiplier 为 2.0, 实际为 %f", config.Reconnect.BackoffMultiplier)
	}
}

func TestConcurrentSequenceGeneration(t *testing.T) {
	config := &Config{
		Robot: RobotConfig{
			UCode: "test_robot",
		},
	}

	client := NewRobotClient(config)

	// 并发测试序列号生成
	var wg sync.WaitGroup
	sequenceChan := make(chan int64, 100)

	// 启动多个goroutine同时生成序列号
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				seq := client.getNextSequence()
				sequenceChan <- seq
			}
		}()
	}

	wg.Wait()
	close(sequenceChan)

	// 收集所有序列号
	sequences := make(map[int64]bool)
	for seq := range sequenceChan {
		if sequences[seq] {
			t.Errorf("发现重复的序列号: %d", seq)
		}
		sequences[seq] = true
	}

	// 验证序列号范围
	expectedCount := 100 // 10个goroutine * 10次调用
	if len(sequences) != expectedCount {
		t.Errorf("期望生成 %d 个唯一序列号, 实际生成 %d 个", expectedCount, len(sequences))
	}

	// 验证序列号连续性（从1开始）
	for i := int64(1); i <= int64(expectedCount); i++ {
		if !sequences[i] {
			t.Errorf("缺少序列号: %d", i)
		}
	}
}

// TestWebSocketClientCreation 测试WebSocketClient创建
func TestWebSocketClientCreation(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			URL: "ws://localhost:8000/ws/control",
		},
	}

	wsService := NewWebSocketClient(config)
	if wsService == nil {
		t.Fatal("WebSocketClient should not be nil")
	}

	if wsService.config != config {
		t.Error("WebSocketClient config should match input config")
	}
}

// TestWebSocketClientCallbacks 测试WebSocketClient回调设置
func TestWebSocketClientCallbacks(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			URL: "ws://localhost:8000/ws/control",
		},
	}

	wsService := NewWebSocketClient(config)

	// 设置回调函数
	wsService.SetCallbacks(
		func() error { return nil },
		func() {},
		func([]byte) error { return nil },
		func(error) {},
	)

	// 验证回调函数已设置
	if wsService.onConnect == nil {
		t.Error("onConnect callback should be set")
	}
	if wsService.onDisconnect == nil {
		t.Error("onDisconnect callback should be set")
	}
	if wsService.onMessage == nil {
		t.Error("onMessage callback should be set")
	}
	if wsService.onError == nil {
		t.Error("onError callback should be set")
	}
}

// TestWebSocketClientStats 测试WebSocketClient统计信息
func TestWebSocketClientStats(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			URL: "ws://localhost:8000/ws/control",
		},
	}

	wsService := NewWebSocketClient(config)
	stats := wsService.GetStats()

	// 验证统计信息包含必要字段
	if stats["connected"] == nil {
		t.Error("Stats should contain 'connected' field")
	}
	if stats["reconnect_attempts"] == nil {
		t.Error("Stats should contain 'reconnect_attempts' field")
	}
	if stats["server_url"] != config.Server.URL {
		t.Error("Stats should contain correct server URL")
	}
}

// TestRobotClientGetStats 测试RobotClient统计信息
func TestRobotClientGetStats(t *testing.T) {
	config := &Config{
		Robot: RobotConfig{
			UCode: "test_robot",
		},
		Server: ServerConfig{
			URL: "ws://localhost:8000/ws/control",
		},
	}

	client := NewRobotClient(config)
	stats := client.GetStats()

	// 验证统计信息包含必要字段
	if stats["ucode"] != "test_robot" {
		t.Error("Stats should contain correct ucode")
	}
	if stats["sequence"] == nil {
		t.Error("Stats should contain sequence field")
	}
	if stats["connected"] == nil {
		t.Error("Stats should contain connected field")
	}
}
