package models

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 前向声明，避免循环依赖
type RobotService interface {
	SetConnection(conn *websocket.Conn)
	Disconnect() error
	SendCommand(command *RobotCommand) (*RobotCommandResponse, error)
	GetStatus() *RobotStatus
	IsConnected() bool
	Start() error
	Stop() error
	AddEventHandler(eventType RobotEventType, handler func(*RobotEvent))
	RemoveEventHandler(eventType RobotEventType, handler func(*RobotEvent))
}

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

// 机器人实体
type Robot struct {
	UCode     string       `json:"ucode"`      // 机器人唯一标识
	Name      string       `json:"name"`       // 机器人名称
	Type      RobotType    `json:"type"`       // 机器人类型
	Config    RobotConfig  `json:"config"`     // 机器人配置
	Status    *RobotStatus `json:"status"`     // 当前状态
	LastSeen  time.Time    `json:"last_seen"`  // 最后活跃时间
	CreatedAt time.Time    `json:"created_at"` // 创建时间
	UpdatedAt time.Time    `json:"updated_at"` // 更新时间
	service   RobotService `json:"-"`          // 机器人服务（不序列化）
	mutex     sync.RWMutex `json:"-"`          // 状态锁
}

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

// 机器人注册信息
type RobotRegistration struct {
	UCode        string    `json:"ucode"`        // 机器人唯一标识
	Name         string    `json:"name"`         // 机器人名称
	Type         RobotType `json:"type"`         // 机器人类型
	Version      string    `json:"version"`      // 版本
	IPAddress    string    `json:"ip_address"`   // IP地址
	Port         int       `json:"port"`         // 端口
	Capabilities []string  `json:"capabilities"` // 能力列表
}

// 机器人绑定请求
type RobotBindRequest struct {
	RobotUCode    string `json:"robot_ucode"`    // 机器人UCode
	OperatorUCode string `json:"operator_ucode"` // 操作员UCode
}

// 机器人绑定响应
type RobotBindResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Robot   *Robot `json:"robot,omitempty"`
}

// 机器人列表响应
type RobotListResponse struct {
	Success bool     `json:"success"`
	Robots  []*Robot `json:"robots"`
	Total   int      `json:"total"`
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

// 方法：更新机器人状态
func (r *Robot) UpdateStatus(status *RobotStatus) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.Status = status
	r.LastSeen = time.Now()
	r.UpdatedAt = time.Now()
}

// 方法：获取机器人状态
func (r *Robot) GetStatus() *RobotStatus {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.Status
}

// GetService 获取机器人服务
func (r *Robot) GetService() RobotService {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.service
}

// SetService 设置机器人服务
func (r *Robot) SetService(service RobotService) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.service = service
}

// 方法：检查机器人是否在线
func (r *Robot) IsOnline() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.Status == nil {
		return false
	}

	// 检查最后心跳时间，超过30秒认为离线
	return r.Status.Connected && time.Since(r.Status.LastHeartbeat) < 30*time.Second
}

// 方法：检查机器人是否健康
func (r *Robot) IsHealthy() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.Status == nil {
		return false
	}

	return r.Status.Connected &&
		r.Status.ErrorCode == 0 &&
		time.Since(r.Status.LastHeartbeat) < 30*time.Second
}
