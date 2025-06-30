package models

import (
	"time"
)

// WebRTC播放地址响应
type WebRTCPlayURLResponse struct {
	Success bool   `json:"success"`
	URL     string `json:"url,omitempty"`
	Message string `json:"message"`
}

// 控制命令
type ControlCommand struct {
	Type       string    `json:"type"`       // 命令类型: joint_position, velocity, emergency_stop, home
	CommandID  string    `json:"command_id"` // 命令ID
	Priority   int       `json:"priority"`   // 优先级 (1-10)
	Timestamp  int64     `json:"timestamp"`  // 时间戳
	JointPos   []float64 `json:"joint_pos"`  // 关节位置 (弧度)
	Velocities []float64 `json:"velocities"` // 关节速度 (弧度/秒)
}

// 机器人状态
type RobotState struct {
	JointPositions  []float64  `json:"joint_positions"`  // 当前关节位置
	JointVelocities []float64  `json:"joint_velocities"` // 当前关节速度
	BasePosition    [3]float64 `json:"base_position"`    // 基座位置 [x, y, z]
	BaseOrientation [4]float64 `json:"base_orientation"` // 基座方向 [x, y, z, w] (四元数)
	BatteryLevel    float64    `json:"battery_level"`    // 电池电量 (%)
	Temperature     float64    `json:"temperature"`      // 温度 (°C)
	Status          string     `json:"status"`           // 状态: idle, moving, error, emergency_stop
	ErrorCode       int        `json:"error_code"`       // 错误代码
	ErrorMessage    string     `json:"error_message"`    // 错误信息
	Timestamp       int64      `json:"timestamp"`        // 时间戳
}

// 控制命令响应
type ControlResponse struct {
	Success   bool   `json:"success"`
	CommandID string `json:"command_id,omitempty"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
}

// WebSocket消息
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 连接状态
type ConnectionStatus struct {
	Connected       bool      `json:"connected"`
	LastHeartbeat   time.Time `json:"last_heartbeat"`
	Latency         int64     `json:"latency_ms"`      // 延迟 (毫秒)
	ActiveClients   int       `json:"active_clients"`  // 活跃客户端数量
	TotalCommands   int64     `json:"total_commands"`  // 总命令数
	FailedCommands  int64     `json:"failed_commands"` // 失败命令数
	LastCommandTime time.Time `json:"last_command_time"`
}

// 系统状态
type SystemStatus struct {
	ServerTime    time.Time `json:"server_time"`
	Uptime        int64     `json:"uptime_seconds"`
	ActiveClients int       `json:"active_clients"`
	RobotStatus   string    `json:"robot_status"`
	JanusStatus   string    `json:"janus_status"`
}

// Janus会话信息
type JanusSession struct {
	SessionID int64  `json:"session_id"`
	HandleID  int64  `json:"handle_id"`
	StreamID  int    `json:"stream_id"`
	URL       string `json:"url"`
}

// 机器人配置
type RobotConfig struct {
	WebSocketURL string `json:"websocket_url"`
	Timeout      int    `json:"timeout_seconds"`
	MaxRetries   int    `json:"max_retries"`
}

// 安全配置
type SecurityConfig struct {
	EnableCORS      bool     `json:"enable_cors"`
	AllowedOrigins  []string `json:"allowed_origins"`
	APIKeyRequired  bool     `json:"api_key_required"`
	RateLimitPerMin int      `json:"rate_limit_per_min"`
}
