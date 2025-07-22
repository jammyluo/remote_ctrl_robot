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

	"github.com/gorilla/websocket"
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
	conn      *websocket.Conn
	sequence  int64
	connected bool
	done      chan struct{}

	// 重连相关字段
	reconnectAttempts int
	lastReconnectTime time.Time
	reconnectTimer    *time.Timer

	// 并发安全
	connMutex sync.Mutex
	seqMutex  sync.Mutex
}

// NewRobotClient 创建新的机器人客户端
func NewRobotClient(config *Config) *RobotClient {
	return &RobotClient{
		config:   config,
		sequence: 1,
		done:     make(chan struct{}),
	}
}

// Start 启动客户端
func (r *RobotClient) Start() error {
	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Str("server", r.config.Server.URL).
		Msg("Starting to connect to server")

	// 尝试连接
	return r.connect()
}

// connect 建立连接
func (r *RobotClient) connect() error {
	// 设置WebSocket连接选项
	dialer := websocket.Dialer{
		HandshakeTimeout: r.config.GetConnectTimeout(),
	}

	// 连接WebSocket
	conn, _, err := dialer.Dial(r.config.Server.URL, nil)
	if err != nil {
		return fmt.Errorf("connect failed: %v", err)
	}

	r.connMutex.Lock()
	r.conn = conn
	r.connected = true
	r.connMutex.Unlock()

	// 重置重连计数
	r.reconnectAttempts = 0

	// 设置连接超时
	r.connMutex.Lock()
	r.conn.SetReadDeadline(time.Now().Add(r.config.GetReadTimeout()))
	r.conn.SetWriteDeadline(time.Now().Add(r.config.GetWriteTimeout()))
	r.connMutex.Unlock()

	// 发送注册消息
	if err := r.sendRegister(); err != nil {
		r.connMutex.Lock()
		r.conn.Close()
		r.connected = false
		r.connMutex.Unlock()
		return fmt.Errorf("register failed: %v", err)
	}

	// 启动消息处理
	go r.handleMessages()

	// 启动心跳
	go r.keepAlive()

	// 启动状态上报
	go r.reportStatus()

	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Msg("Connected successfully")
	return nil
}

// Stop 停止客户端
func (r *RobotClient) Stop() {
	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Msg("Stopping client")

	r.connMutex.Lock()
	r.connected = false
	r.connMutex.Unlock()

	close(r.done)

	// 停止重连定时器
	if r.reconnectTimer != nil {
		r.reconnectTimer.Stop()
	}

	r.connMutex.Lock()
	if r.conn != nil {
		r.conn.Close()
	}
	r.connMutex.Unlock()

	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Msg("Client stopped")
}

// scheduleReconnect 安排重连
func (r *RobotClient) scheduleReconnect() {
	if !r.config.Reconnect.Enabled {
		log.Warn().
			Str("ucode", r.config.Robot.UCode).
			Msg("Reconnect disabled, not attempting to reconnect")
		return
	}

	if r.reconnectAttempts >= r.config.Reconnect.MaxAttempts {
		log.Error().
			Str("ucode", r.config.Robot.UCode).
			Int("max_attempts", r.config.Reconnect.MaxAttempts).
			Msg("Max reconnect attempts reached, giving up")
		return
	}

	// 计算重连延迟
	delay := r.calculateReconnectDelay()

	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Int("attempt", r.reconnectAttempts+1).
		Int("max_attempts", r.config.Reconnect.MaxAttempts).
		Dur("delay", delay).
		Msg("Scheduling reconnect")

	// 设置重连定时器
	r.reconnectTimer = time.AfterFunc(delay, func() {
		r.performReconnect()
	})
}

// calculateReconnectDelay 计算重连延迟
func (r *RobotClient) calculateReconnectDelay() time.Duration {
	// 指数退避算法
	baseDelay := time.Duration(r.config.Reconnect.InitialDelay) * time.Second
	maxDelay := time.Duration(r.config.Reconnect.MaxDelay) * time.Second

	// 计算延迟：baseDelay * (backoff_multiplier ^ attempts)
	delay := float64(baseDelay) * pow(r.config.Reconnect.BackoffMultiplier, r.reconnectAttempts)

	// 添加随机抖动 (±20%)
	jitter := delay * 0.2 * (rand.Float64()*2 - 1)
	delay += jitter

	// 确保延迟在合理范围内
	if delay < float64(baseDelay) {
		delay = float64(baseDelay)
	}
	if delay > float64(maxDelay) {
		delay = float64(maxDelay)
	}

	return time.Duration(delay)
}

// pow 计算幂次
func pow(base float64, exponent int) float64 {
	result := 1.0
	for i := 0; i < exponent; i++ {
		result *= base
	}
	return result
}

// performReconnect 执行重连
func (r *RobotClient) performReconnect() {
	select {
	case <-r.done:
		return
	default:
	}

	r.reconnectAttempts++
	r.lastReconnectTime = time.Now()

	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Int("attempt", r.reconnectAttempts).
		Int("max_attempts", r.config.Reconnect.MaxAttempts).
		Msg("Attempting to reconnect")

	if err := r.connect(); err != nil {
		log.Error().
			Err(err).
			Str("ucode", r.config.Robot.UCode).
			Int("attempt", r.reconnectAttempts).
			Msg("Reconnect failed")

		// 安排下次重连
		r.scheduleReconnect()
	} else {
		log.Info().
			Str("ucode", r.config.Robot.UCode).
			Int("attempt", r.reconnectAttempts).
			Msg("Reconnect successful")
	}
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

	r.connMutex.Lock()
	defer r.connMutex.Unlock()

	if r.conn == nil {
		return fmt.Errorf("connection is nil")
	}

	// 设置写入超时
	r.conn.SetWriteDeadline(time.Now().Add(r.config.GetWriteTimeout()))
	return r.conn.WriteJSON(msg)
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

	r.connMutex.Lock()
	defer r.connMutex.Unlock()

	if r.conn == nil {
		return fmt.Errorf("connection is nil")
	}

	// 设置写入超时
	r.conn.SetWriteDeadline(time.Now().Add(r.config.GetWriteTimeout()))
	return r.conn.WriteJSON(msg)
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

	r.connMutex.Lock()
	defer r.connMutex.Unlock()

	if r.conn == nil {
		return fmt.Errorf("connection is nil")
	}

	// 设置写入超时
	r.conn.SetWriteDeadline(time.Now().Add(r.config.GetWriteTimeout()))
	return r.conn.WriteJSON(msg)
}

// handleMessages 处理接收到的消息
func (r *RobotClient) handleMessages() {
	for {
		select {
		case <-r.done:
			return
		default:
			r.connMutex.Lock()
			if r.conn == nil {
				r.connMutex.Unlock()
				return
			}
			// 设置读取超时
			r.conn.SetReadDeadline(time.Now().Add(r.config.GetReadTimeout()))
			r.connMutex.Unlock()

			_, message, err := r.conn.ReadMessage()
			if err != nil {
				r.connMutex.Lock()
				if r.connected {
					log.Error().
						Err(err).
						Str("ucode", r.config.Robot.UCode).
						Msg("Read message error")

					// 标记连接断开
					r.connected = false
					r.conn.Close()
					r.conn = nil
					r.connMutex.Unlock()

					// 安排重连
					r.scheduleReconnect()
				} else {
					r.connMutex.Unlock()
				}
				return
			}

			var msg WebSocketMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Error().
					Err(err).
					Str("ucode", r.config.Robot.UCode).
					Msg("Parse message failed")
				continue
			}

			log.Debug().
				Str("ucode", r.config.Robot.UCode).
				Str("command", string(msg.Command)).
				Int64("sequence", msg.Sequence).
				Msg("Received message")
		}
	}
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
			r.connMutex.Lock()
			connected := r.connected
			r.connMutex.Unlock()

			if connected {
				if err := r.sendPing(); err != nil {
					log.Error().
						Err(err).
						Str("ucode", r.config.Robot.UCode).
						Msg("Send heartbeat failed")

					r.connMutex.Lock()
					// 标记连接断开
					r.connected = false
					if r.conn != nil {
						r.conn.Close()
						r.conn = nil
					}
					r.connMutex.Unlock()

					// 安排重连
					r.scheduleReconnect()
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
			r.connMutex.Lock()
			connected := r.connected
			r.connMutex.Unlock()

			if connected {
				if err := r.sendStatusUpdate(); err != nil {
					log.Error().
						Err(err).
						Str("ucode", r.config.Robot.UCode).
						Msg("Send status update failed")

					r.connMutex.Lock()
					// 标记连接断开
					r.connected = false
					if r.conn != nil {
						r.conn.Close()
						r.conn = nil
					}
					r.connMutex.Unlock()

					// 安排重连
					r.scheduleReconnect()
				} else {
					log.Debug().
						Str("ucode", r.config.Robot.UCode).
						Msg("Send status update")
				}
			}
		}
	}
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
