package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// 消息类型定义
type ClientType string
type WSMessageType string
type CommandType string

const (
	ClientTypeRobot              ClientType    = "robot"
	WSMessageTypeRequest         WSMessageType = "Request"
	CMD_TYPE_REGISTER            CommandType   = "CMD_REGISTER"
	CMD_TYPE_PING                CommandType   = "CMD_PING"
	CMD_TYPE_UPDATE_ROBOT_STATUS CommandType   = "CMD_UPDATE_ROBOT_STATUS"
)

// WebSocket消息结构
type WebSocketMessage struct {
	Type       WSMessageType `json:"type"`
	Command    CommandType   `json:"command"`
	Sequence   int64         `json:"sequence"`
	UCode      string        `json:"ucode"`
	ClientType ClientType    `json:"client_type"`
	Version    string        `json:"version"`
	Data       interface{}   `json:"data"`
}

// 机器人状态结构
type RobotState struct {
	BasePosition    [3]float64 `json:"base_position"`
	BaseOrientation [4]float64 `json:"base_orientation"`
	BatteryLevel    float64    `json:"battery_level"`
	Temperature     float64    `json:"temperature"`
	Status          string     `json:"status"`
	ErrorCode       int        `json:"error_code"`
	ErrorMessage    string     `json:"error_message"`
}

// RobotClient 极简机器人客户端
type RobotClient struct {
	config    *Config
	wsService *WebSocketService
	sequence  int64
	done      chan struct{}

	// 并发安全
	seqMutex sync.Mutex
}

// NewRobotClient 创建新的机器人客户端
func NewRobotClient(config *Config) *RobotClient {
	client := &RobotClient{
		config: config,
		done:   make(chan struct{}),
	}

	// 创建WebSocket服务
	client.wsService = NewWebSocketService(config)

	// 设置WebSocket服务回调
	client.wsService.SetCallbacks(
		client.onConnect,    // 连接成功回调
		client.onDisconnect, // 连接断开回调
		client.onMessage,    // 消息接收回调
		client.onError,      // 错误处理回调
	)

	return client
}

// Start 启动客户端
func (r *RobotClient) Start() error {
	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Str("server", r.config.Server.URL).
		Msg("Starting robot client")

	// 启动WebSocket服务
	return r.wsService.Start()
}

// Stop 停止客户端
func (r *RobotClient) Stop() {
	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Msg("Stopping robot client")

	close(r.done)

	// 停止WebSocket服务
	r.wsService.Stop()

	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Msg("Robot client stopped")
}

// onConnect 连接成功回调
func (r *RobotClient) onConnect() error {
	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Msg("Robot connected, sending register message")

	// 发送注册消息
	if err := r.sendRegister(); err != nil {
		return fmt.Errorf("send register failed: %v", err)
	}

	// 启动心跳
	go r.keepAlive()

	// 启动状态上报
	go r.reportStatus()

	return nil
}

// onDisconnect 连接断开回调
func (r *RobotClient) onDisconnect() {
	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Msg("Robot disconnected")
}

// onMessage 消息接收回调
func (r *RobotClient) onMessage(message []byte) error {
	var msg WebSocketMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		return fmt.Errorf("parse message failed: %v", err)
	}

	log.Debug().
		Str("ucode", r.config.Robot.UCode).
		Str("command", string(msg.Command)).
		Int64("sequence", msg.Sequence).
		Msg("Received message")

	return nil
}

// onError 错误处理回调
func (r *RobotClient) onError(err error) {
	log.Error().
		Err(err).
		Str("ucode", r.config.Robot.UCode).
		Msg("Robot error occurred")
}

// getNextSequence 获取下一个序列号
func (r *RobotClient) getNextSequence() int64 {
	r.seqMutex.Lock()
	defer r.seqMutex.Unlock()
	r.sequence++
	return r.sequence
}

// sendRegister 发送注册消息
func (r *RobotClient) sendRegister() error {
	msg := WebSocketMessage{
		Type:       WSMessageTypeRequest,
		Command:    CMD_TYPE_REGISTER,
		Sequence:   r.getNextSequence(),
		UCode:      r.config.Robot.UCode,
		ClientType: ClientTypeRobot,
		Version:    r.config.Robot.Version,
		Data:       map[string]interface{}{},
	}

	return r.wsService.SendMessage(msg)
}

// sendPing 发送心跳消息
func (r *RobotClient) sendPing() error {
	msg := WebSocketMessage{
		Type:       WSMessageTypeRequest,
		Command:    CMD_TYPE_PING,
		Sequence:   r.getNextSequence(),
		UCode:      r.config.Robot.UCode,
		ClientType: ClientTypeRobot,
		Version:    r.config.Robot.Version,
		Data:       map[string]interface{}{},
	}

	return r.wsService.SendMessage(msg)
}

// sendStatusUpdate 发送状态更新
func (r *RobotClient) sendStatusUpdate() error {
	var status RobotState

	if r.config.Status.EnableSimulation {
		// 模拟机器人状态
		status = RobotState{
			BasePosition:    [3]float64{rand.Float64() * 10, rand.Float64() * 10, 0},
			BaseOrientation: [4]float64{0, 0, 0, 1},
			BatteryLevel:    rand.Float64() * 100,
			Temperature:     20 + rand.Float64()*30,
			Status:          "idle",
			ErrorCode:       0,
			ErrorMessage:    "",
		}
	} else {
		// 使用固定状态
		status = RobotState{
			BasePosition:    [3]float64{0, 0, 0},
			BaseOrientation: [4]float64{0, 0, 0, 1},
			BatteryLevel:    100.0,
			Temperature:     25.0,
			Status:          "idle",
			ErrorCode:       0,
			ErrorMessage:    "",
		}
	}

	msg := WebSocketMessage{
		Type:       WSMessageTypeRequest,
		Command:    CMD_TYPE_UPDATE_ROBOT_STATUS,
		Sequence:   r.getNextSequence(),
		UCode:      r.config.Robot.UCode,
		ClientType: ClientTypeRobot,
		Version:    r.config.Robot.Version,
		Data:       status,
	}

	return r.wsService.SendMessage(msg)
}

// keepAlive 心跳保持
func (r *RobotClient) keepAlive() {
	ticker := time.NewTicker(r.config.GetHeartbeatInterval())
	defer ticker.Stop()

	for {
		select {
		case <-r.done:
			return
		case <-ticker.C:
			if r.wsService.IsConnected() {
				if err := r.sendPing(); err != nil {
					log.Error().
						Err(err).
						Str("ucode", r.config.Robot.UCode).
						Msg("Send heartbeat failed")
				} else {
					log.Debug().
						Str("ucode", r.config.Robot.UCode).
						Msg("Send heartbeat")
				}
			}
		}
	}
}

// reportStatus 状态上报
func (r *RobotClient) reportStatus() {
	ticker := time.NewTicker(r.config.GetStatusInterval())
	defer ticker.Stop()

	for {
		select {
		case <-r.done:
			return
		case <-ticker.C:
			if r.wsService.IsConnected() {
				if err := r.sendStatusUpdate(); err != nil {
					log.Error().
						Err(err).
						Str("ucode", r.config.Robot.UCode).
						Msg("Send status update failed")
				} else {
					log.Debug().
						Str("ucode", r.config.Robot.UCode).
						Msg("Send status update")
				}
			}
		}
	}
}

// GetStats 获取客户端统计信息
func (r *RobotClient) GetStats() map[string]interface{} {
	stats := r.wsService.GetStats()
	stats["ucode"] = r.config.Robot.UCode
	stats["sequence"] = r.sequence
	return stats
}

// initLogger 初始化日志系统
func initLogger(config *Config) {
	// 设置日志级别
	logLevel, err := zerolog.ParseLevel(config.Logging.Level)
	if err != nil {
		fmt.Printf("Invalid log level %s, using default level info\n", config.Logging.Level)
		logLevel = zerolog.InfoLevel
	}

	// 设置时区
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.SetGlobalLevel(logLevel)

	// 设置输出格式
	if config.Logging.Format == "console" || logLevel == zerolog.DebugLevel {
		// 控制台格式
		writer := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		log.Logger = zerolog.New(writer).With().Timestamp().Logger()
	} else {
		// JSON格式
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}
}

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config.yaml", "config file path")
	flag.Parse()

	// 加载配置
	config, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Load config failed: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	initLogger(config)

	log.Info().
		Str("ucode", config.Robot.UCode).
		Str("server", config.Server.URL).
		Str("log_level", config.Logging.Level).
		Str("config_file", *configPath).
		Bool("reconnect_enabled", config.Reconnect.Enabled).
		Int("max_reconnect_attempts", config.Reconnect.MaxAttempts).
		Msg("Start robot client")

	// 创建机器人客户端
	client := NewRobotClient(config)

	// 启动客户端
	if err := client.Start(); err != nil {
		log.Fatal().
			Err(err).
			Msg("Start failed")
	}

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info().Msg("Received exit signal, shutting down...")

	// 优雅停止
	client.Stop()
}
