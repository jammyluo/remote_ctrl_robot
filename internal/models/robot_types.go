package models

import (
	"time"
)

// 机器人类型
type RobotType string

const (
	RobotTypeB2   RobotType = "b2"
	RobotTypeGo2  RobotType = "go2"
	RobotTypeG1   RobotType = "g1"
	RobotTypeH1   RobotType = "h1"
	RobotTypeB2W  RobotType = "b2w"
	RobotTypeGo2W RobotType = "go2w"
	RobotTypeH1_2 RobotType = "h1_2"
)

// 机器人状态
type RobotStatus struct {
	Connected       bool        `json:"connected"`         // 连接状态
	RobotState      *RobotState `json:"robot_state"`       // 机器人物理状态
	GameState       *GameRobot  `json:"game_state"`        // 游戏状态
	LastHeartbeat   time.Time   `json:"last_heartbeat"`    // 最后心跳
	Latency         int64       `json:"latency_ms"`        // 延迟(毫秒)
	ErrorCode       int         `json:"error_code"`        // 错误代码
	ErrorMessage    string      `json:"error_message"`     // 错误信息
	TotalCommands   int64       `json:"total_commands"`    // 总命令数
	FailedCommands  int64       `json:"failed_commands"`   // 失败命令数
	LastCommandTime time.Time   `json:"last_command_time"` // 最后命令时间
}

// 机器人绑定请求
type RobotBindRequest struct {
	RobotUCode    string `json:"robot_ucode"`    // 机器人UCode
	OperatorUCode string `json:"operator_ucode"` // 操作员UCode
}

// 机器人状态更新
type RobotStatusUpdate struct {
	UCode  string       `json:"ucode"`
	Status *RobotStatus `json:"status"`
}

// 机器人命令
type RobotCommand struct {
	Action        string            `json:"action"`         // 动作: move, stop, reset, etc.
	Params        map[string]string `json:"params"`         // 参数
	Priority      int               `json:"priority"`       // 优先级 (1-10, 10最高)
	Timestamp     int64             `json:"timestamp"`      // 时间戳
	OperatorUCode string            `json:"operator_ucode"` // 操作员UCode
}

// 机器人命令响应
type RobotCommandResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	CommandID string `json:"command_id,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// 机器人事件类型
type RobotEventType string

const (
	RobotEventConnected    RobotEventType = "connected"     // 连接成功
	RobotEventDisconnected RobotEventType = "disconnected"  // 连接断开
	RobotEventError        RobotEventType = "error"         // 错误
	RobotEventStatusUpdate RobotEventType = "status_update" // 状态更新
	RobotEventCommand      RobotEventType = "command"       // 命令执行
	RobotEventHeartbeat    RobotEventType = "heartbeat"     // 心跳
)

// 机器人事件
type RobotEvent struct {
	Type      RobotEventType `json:"type"`
	UCode     string         `json:"ucode"`
	Timestamp time.Time      `json:"timestamp"`
	Data      interface{}    `json:"data"`
	Message   string         `json:"message,omitempty"`
}

// 机器人统计信息
type RobotStatistics struct {
	UCode          string    `json:"ucode"`
	TotalCommands  int64     `json:"total_commands"`
	FailedCommands int64     `json:"failed_commands"`
	SuccessRate    float64   `json:"success_rate"`
	AverageLatency float64   `json:"average_latency_ms"`
	Uptime         int64     `json:"uptime_seconds"`
	LastSeen       time.Time `json:"last_seen"`
	ErrorCount     int64     `json:"error_count"`
	ReconnectCount int64     `json:"reconnect_count"`
}

// 机器人健康检查
type RobotHealthCheck struct {
	UCode        string    `json:"ucode"`
	Healthy      bool      `json:"healthy"`
	LastCheck    time.Time `json:"last_check"`
	Issues       []string  `json:"issues"`
	Latency      int64     `json:"latency_ms"`
	BatteryLevel float64   `json:"battery_level"`
	Temperature  float64   `json:"temperature"`
}

// 操作员事件类型
type OperatorEventType string

const (
	OperatorEventConnected    OperatorEventType = "connected"     // 连接成功
	OperatorEventDisconnected OperatorEventType = "disconnected"  // 连接断开
	OperatorEventError        OperatorEventType = "error"         // 错误
	OperatorEventBindRobot    OperatorEventType = "bind_robot"    // 绑定机器人
	OperatorEventControlRobot OperatorEventType = "control_robot" // 控制机器人
	OperatorEventHeartbeat    OperatorEventType = "heartbeat"     // 心跳
	OperatorEventJoinGame     OperatorEventType = "join_game"     // 加入游戏
	OperatorEventLeaveGame    OperatorEventType = "leave_game"    // 离开游戏
)

// 操作员事件
type OperatorEvent struct {
	Type      OperatorEventType `json:"type"`
	UCode     string            `json:"ucode"`
	Timestamp time.Time         `json:"timestamp"`
	Data      interface{}       `json:"data"`
	Message   string            `json:"message,omitempty"`
}
