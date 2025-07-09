package models

import (
	"time"
)

// 游戏相关常量
const (
	// 游戏状态
	GameStatusWaiting  = "waiting"  // 等待开始
	GameStatusPlaying  = "playing"  // 游戏中
	GameStatusFinished = "finished" // 游戏结束
	GameStatusPaused   = "paused"   // 游戏暂停

	// 游戏配置
	DefaultMaxHealth    = 100 // 默认最大血量
	DefaultBulletDamage = 20  // 默认子弹伤害
	DefaultBulletRange  = 50  // 默认射程
	DefaultBulletSpeed  = 10  // 默认子弹速度
	DefaultRespawnTime  = 10  // 默认复活时间(秒)
	DefaultGameDuration = 300 // 默认游戏时长(秒)
)

// 游戏状态
type GameState struct {
	GameID     string                `json:"game_id"`    // 游戏ID
	Status     string                `json:"status"`     // 游戏状态
	StartTime  time.Time             `json:"start_time"` // 开始时间
	EndTime    time.Time             `json:"end_time"`   // 结束时间
	Duration   int                   `json:"duration"`   // 游戏时长(秒)
	Robots     map[string]*GameRobot `json:"robots"`     // 机器人状态
	Bullets    []*GameBullet         `json:"bullets"`    // 子弹列表
	Config     *GameConfig           `json:"config"`     // 游戏配置
	Winner     string                `json:"winner"`     // 获胜者
	Statistics *GameStatistics       `json:"statistics"` // 游戏统计
}

// 子弹
type Bullet struct {
	ID          string  `json:"id"`           // 子弹ID
	Damage      int     `json:"damage"`       // 伤害
	Range       float64 `json:"range"`        // 射程
	Caliber     float64 `json:"caliber"`      // 口径
	BulletType  string  `json:"bullet_type"`  // 子弹类型
	BulletColor string  `json:"bullet_color"` // 子弹颜色
}

// 弹夹
type Magazine struct {
	ID            string    `json:"id"`             // 弹夹ID
	Bullet        Bullet    `json:"bullet"`         // 子弹
	MagazineSize  int       `json:"magazine_size"`  // 弹夹容量
	ReloadingTime time.Time `json:"reloading_time"` // 装弹时间
}

// 游戏机器人
type GameRobot struct {
	UCode         string    `json:"ucode"`          // 机器人唯一标识
	Name          string    `json:"name"`           // 机器人名称
	Health        int       `json:"health"`         // 当前血量
	MaxHealth     int       `json:"max_health"`     // 最大血量
	Position      Position  `json:"position"`       // 位置
	Direction     float64   `json:"direction"`      // 朝向(弧度)
	IsAlive       bool      `json:"is_alive"`       // 是否存活
	LastShot      time.Time `json:"last_shot"`      // 上次射击时间
	Magazine      Magazine  `json:"magazine"`       // 弹夹
	RemainingAmmo int       `json:"remaining_ammo"` // 剩余弹药
	Reloading     bool      `json:"reloading"`      // 是否正在装弹
	RespawnTime   time.Time `json:"respawn_time"`   // 复活时间
	Score         int       `json:"score"`          // 得分
	Kills         int       `json:"kills"`          // 击杀数
	Deaths        int       `json:"deaths"`         // 死亡数
	ShotsFired    int       `json:"shots_fired"`    // 射击次数
	ShotsHit      int       `json:"shots_hit"`      // 命中次数
}

// 位置信息
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// 游戏子弹
type GameBullet struct {
	ID           string    `json:"id"`            // 子弹ID
	ShooterUCode string    `json:"shooter_ucode"` // 射击者
	StartPos     Position  `json:"start_pos"`     // 起始位置
	CurrentPos   Position  `json:"current_pos"`   // 当前位置
	Direction    Position  `json:"direction"`     // 方向向量
	Damage       int       `json:"damage"`        // 实际伤害值
	Range        float64   `json:"range"`         // 实际射程值
	Created      time.Time `json:"created"`       // 创建时间
	IsActive     bool      `json:"is_active"`     // 是否活跃
}

// 游戏配置
type GameConfig struct {
	MaxHealth     int     `json:"max_health"`     // 最大血量
	BulletDamage  int     `json:"bullet_damage"`  // 子弹伤害
	BulletRange   float64 `json:"bullet_range"`   // 射程
	BulletSpeed   float64 `json:"bullet_speed"`   // 子弹速度
	RespawnTime   int     `json:"respawn_time"`   // 复活时间(秒)
	GameDuration  int     `json:"game_duration"`  // 游戏时长(秒)
	MapWidth      float64 `json:"map_width"`      // 地图宽度
	MapHeight     float64 `json:"map_height"`     // 地图高度
	ShootCooldown float64 `json:"shoot_cooldown"` // 射击冷却时间(秒)
}

// 游戏统计
type GameStatistics struct {
	TotalShots  int                    `json:"total_shots"`  // 总射击数
	TotalHits   int                    `json:"total_hits"`   // 总命中数
	TotalKills  int                    `json:"total_kills"`  // 总击杀数
	TotalDeaths int                    `json:"total_deaths"` // 总死亡数
	RobotStats  map[string]*RobotStats `json:"robot_stats"`  // 各机器人统计
	GameEvents  []*GameEvent           `json:"game_events"`  // 游戏事件
}

// 机器人统计
type RobotStats struct {
	UCode      string  `json:"ucode"`
	Score      int     `json:"score"`
	Kills      int     `json:"kills"`
	Deaths     int     `json:"deaths"`
	ShotsFired int     `json:"shots_fired"`
	ShotsHit   int     `json:"shots_hit"`
	Accuracy   float64 `json:"accuracy"` // 命中率
	KDRatio    float64 `json:"kd_ratio"` // 击杀死亡比
}

// 游戏事件
type GameEvent struct {
	Type         string    `json:"type"`          // 事件类型: shot, hit, kill, respawn, game_start, game_end
	Timestamp    time.Time `json:"timestamp"`     // 时间戳
	ShooterUCode string    `json:"shooter_ucode"` // 射击者
	TargetUCode  string    `json:"target_ucode"`  // 目标
	Damage       int       `json:"damage"`        // 伤害值
	Position     Position  `json:"position"`      // 位置
	Message      string    `json:"message"`       // 事件描述
}

// 游戏命令类型
const (
	CMD_TYPE_JOIN_GAME    CommandType = "CMD_JOIN_GAME"    // 加入游戏
	CMD_TYPE_LEAVE_GAME   CommandType = "CMD_LEAVE_GAME"   // 离开游戏
	CMD_TYPE_GAME_MOVE    CommandType = "CMD_GAME_MOVE"    // 游戏移动
	CMD_TYPE_GAME_STATUS  CommandType = "CMD_GAME_STATUS"  // 游戏状态
	CMD_TYPE_GAME_CONFIG  CommandType = "CMD_GAME_CONFIG"  // 游戏配置
	CMD_TYPE_GAME_RELOAD  CommandType = "CMD_GAME_RELOAD"  // 游戏装弹
	CMD_TYPE_GAME_SHOOT   CommandType = "CMD_GAME_SHOOT"   // 游戏射击
	CMD_TYPE_GAME_HIT     CommandType = "CMD_GAME_HIT"     // 被击中
	CMD_TYPE_GAME_START   CommandType = "CMD_GAME_START"   // 开始游戏
	CMD_TYPE_GAME_STOP    CommandType = "CMD_GAME_STOP"    // 暂停游戏
	CMD_TYPE_GAME_END     CommandType = "CMD_GAME_END"     // 结束游戏
	CMD_TYPE_GAME_RESPAWN CommandType = "CMD_GAME_RESPAWN" // 复活
)

// 加入游戏请求
type CMD_JOIN_GAME struct {
	GameID string `json:"game_id"` // 游戏ID
	Name   string `json:"name"`    // 机器人名称
}

// 游戏射击请求
type CMD_GAME_SHOOT struct {
	TargetX float64 `json:"target_x"` // 目标X坐标
	TargetY float64 `json:"target_y"` // 目标Y坐标
	TargetZ float64 `json:"target_z"` // 目标Z坐标
}

// 游戏移动请求
type CMD_GAME_MOVE struct {
	Position  Position `json:"position"`  // 目标位置
	Direction float64  `json:"direction"` // 朝向
}

// 游戏状态响应
type CMD_GAME_STATUS_RESPONSE struct {
	GameState *GameState `json:"game_state"`
	MyRobot   *GameRobot `json:"my_robot"`
}

// 游戏事件通知
type CMD_GAME_EVENT struct {
	Event *GameEvent `json:"event"`
}

// 游戏统计响应
type CMD_GAME_STATISTICS_RESPONSE struct {
	Statistics *GameStatistics `json:"statistics"`
}
