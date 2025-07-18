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

// 客户端类型
type ClientType string
type WSMessageType string
type CommandType string

const (
	ClientTypeRobot       ClientType    = "robot"    // 机器人
	ClientTypeOperator    ClientType    = "operator" // 操作者
	WSMessageTypeResponse WSMessageType = "Response"
	WSMessageTypeRequest  WSMessageType = "Request"
)

const (
	CMD_TYPE_REGISTER            CommandType = "CMD_REGISTER"            // 注册
	CMD_TYPE_BIND_ROBOT          CommandType = "CMD_BIND_ROBOT"          // 机器人绑定
	CMD_TYPE_UPDATE_ROBOT_STATUS CommandType = "CMD_UPDATE_ROBOT_STATUS" // 机器人状态
	CMD_TYPE_PING                CommandType = "CMD_PING"                // 心跳
	CMD_TYPE_CONTROL_ROBOT       CommandType = "CMD_CONTROL_ROBOT"       // 控制机器人
	CMD_TYPE_REPORT_HIT_DATA     CommandType = "CMD_REPORT_HIT_DATA"     // 上报伤害数据
	CMD_TYPE_UPDATE_LIFE_DATA    CommandType = "CMD_UPDATE_LIFE_DATA"    // 更新生命数据
)

// WebSocket消息
type WebSocketMessage struct {
	Type       WSMessageType `json:"type"`        // 消息类型: Response, Request
	ClientType ClientType    `json:"client_type"` // 客户端类型: robot, operator
	Command    CommandType   `json:"command"`     // 命令类型
	Sequence   int64         `json:"sequence"`    // 序列号
	UCode      string        `json:"ucode"`       // UCode
	Version    string        `json:"version"`     // 版本
	Data       interface{}   `json:"data"`        // 数据
}

type CMD_BIND_ROBOT struct {
	UCode string `json:"ucode"` // 机器人UCode
}

type CMD_CONTROL_ROBOT struct {
	Action    string            `json:"action"`    // 动作: move, stop, reset, etc.
	ParamMaps map[string]string `json:"params"`    // 参数: 动作参数
	Timestamp int64             `json:"timestamp"` // 时间戳
}

type CMD_REPORT_HIT_DATA struct {
	HitPress  int64 `json:"hit_press"` // 伤害值
	Timestamp int64 `json:"timestamp"` // 时间戳
}

type CMD_UPDATE_LIFE_DATA struct {
	LifePress int64 `json:"life_press"` // 生命值
	Timestamp int64 `json:"timestamp"`  // 时间戳
}

// 命令响应
type CMD_RESPONSE struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Timestamp int64  `json:"timestamp"`
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
	WebSocketURL      string `json:"websocket_url"`              // WebSocket连接地址
	Timeout           int    `json:"timeout_seconds"`            // 连接超时时间
	MaxRetries        int    `json:"max_retries"`                // 最大重试次数
	HeartbeatInterval int    `json:"heartbeat_interval_seconds"` // 心跳间隔
	ReconnectInterval int    `json:"reconnect_interval_seconds"` // 重连间隔
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

// 客户端事件类型
type ClientEventType string

const (
	ClientEventConnected    ClientEventType = "connected"
	ClientEventDisconnected ClientEventType = "disconnected"
	ClientEventCommand      ClientEventType = "command"
	ClientEventError        ClientEventType = "error"
	ClientEventGameJoin     ClientEventType = "game_join"
	ClientEventGameLeave    ClientEventType = "game_leave"
)

// 客户端事件
type ClientEvent struct {
	Type      ClientEventType `json:"type"`
	UCode     string          `json:"ucode"`
	Timestamp time.Time       `json:"timestamp"`
	Message   string          `json:"message"`
	Data      interface{}     `json:"data,omitempty"`
}
