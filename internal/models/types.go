package models

import (
	"time"
)

// WebRTC播放地址响应
type WebRTCPlayURLResponse struct {
	Success bool     `json:"success"`
	URLs    []string `json:"urls"` // 播放地址列表
	Message string   `json:"message"`
}

// 控制命令
type ControlCommand struct {
	CommandID string `json:"command_id"` // 命令ID
	Priority  int    `json:"priority"`   // 优先级 (1-10)
	Timestamp int64  `json:"timestamp"`  // 时间戳
}

// 机器人状态
type RobotState struct {
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

// WebRTC流信息
type WebRTCStream struct {
	UCode     string `json:"ucode"`      // 机器人唯一标识
	StreamID  int    `json:"stream_id"`  // 流ID
	SessionID int64  `json:"session_id"` // Janus会话ID
	HandleID  int64  `json:"handle_id"`  // Janus句柄ID
	PlayURL   string `json:"play_url"`   // 播放地址
	PushURL   string `json:"push_url"`   // 推流地址
	Status    string `json:"status"`     // 状态: active, inactive, error
	CreatedAt int64  `json:"created_at"` // 创建时间戳
}

// WebRTC注册请求
type WebRTCRegisterRequest struct {
	UCode string `json:"ucode"` // 机器人唯一标识
}

// WebRTC注册响应
type WebRTCRegisterResponse struct {
	Success bool          `json:"success"`
	Stream  *WebRTCStream `json:"stream,omitempty"`
	Message string        `json:"message"`
}

// 客户端类型
type ClientType string

const (
	ClientTypeRobot    ClientType = "robot"    // 机器人
	ClientTypeOperator ClientType = "operator" // 操作者
)

// 注册请求
type RegisterRequest struct {
	UCode      string     `json:"ucode"`       // 唯一标识
	ClientType ClientType `json:"client_type"` // 客户端类型: robot, operator
	Name       string     `json:"name"`        // 名称（可选）
	Version    string     `json:"version"`     // 版本（可选）
}

// 注册响应
type RegisterResponse struct {
	Success    bool       `json:"success"`
	UCode      string     `json:"ucode"`
	ClientType ClientType `json:"client_type"`
	Message    string     `json:"message"`
	Error      string     `json:"error,omitempty"`
	Timestamp  int64      `json:"timestamp"`
}

type RobotConnection struct {
	UCode      string    `json:"ucode"` // 机器人UCode（对于操作者，这是要控制的机器人UCode）
	Version    string    `json:"version"`
	Connected  bool      `json:"connected"`
	LastSeen   time.Time `json:"last_seen"`
	RemoteAddr string    `json:"remote_addr"`
}

// 客户端信息
type OperatorConnection struct {
	OperatorID string    `json:"operator_id"` // 操作者标识（仅对操作者有效）
	RobotUCode string    `json:"robot_ucode"` // 机器人UCode（对于操作者，这是要控制的机器人UCode）
	Name       string    `json:"name"`
	Version    string    `json:"version"`
	Connected  bool      `json:"connected"`
	LastSeen   time.Time `json:"last_seen"`
	RemoteAddr string    `json:"remote_addr"`
}
