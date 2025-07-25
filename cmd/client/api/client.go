package api

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"remote-ctrl-robot/cmd/client/config"
	"remote-ctrl-robot/cmd/client/robot"
	"remote-ctrl-robot/internal/models"
	"remote-ctrl-robot/internal/services"

	"github.com/rs/zerolog/log"
)

// Client 极简机器人客户端
type Client struct {
	config    *config.Config
	wsService *services.WebSocketClient
	robot     robot.RobotInterface
	apiServer *APIServer
	sequence  int64
	done      chan struct{}

	// 并发安全
	seqMutex sync.Mutex
}

// NewClient 创建新的机器人客户端
func NewClient(config *config.Config) *Client {
	client := &Client{
		config: config,
		done:   make(chan struct{}),
	}

	// 创建WebSocket服务
	client.apiServer = NewAPIServer(client, config.API.Port) // 默认端口8080
	client.wsService = services.NewWebSocketClient(config.WebSocket.URL, config.GetWriteTimeout(), config.GetReadTimeout(), config.GetConnectTimeout(), config.GetReconnectDelay())
	if config.Robot.ClientType == "mock" {
		client.robot = robot.NewMockRobot(config)
	} else {
		client.robot = robot.NewSimpleRobot(config)
	}

	// 创建API服务器

	// 设置WebSocket服务回调
	client.wsService.SetCallbacks(
		client.OnConnect,    // 连接成功回调
		client.OnDisconnect, // 连接断开回调
		client.OnMessage,    // 消息接收回调
		client.OnError,      // 错误处理回调
	)

	return client
}

// Start 启动客户端
func (r *Client) Start() error {
	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Str("server", r.config.WebSocket.URL).
		Msg("Starting robot client")

	if r.config.WebSocket.Enable {
		// 启动WebSocket服务
		if err := r.wsService.Start(); err != nil {
			return err
		}
	}

	// 启动API服务器（在goroutine中运行）
	go func() {
		if r.config.API.Enable {
			if err := r.apiServer.Start(); err != nil {
				log.Error().Err(err).Msg("API server failed")
			}
		} else {
			log.Error().Msg("API server is not enabled")
		}
	}()

	return nil
}

// Stop 停止客户端
func (r *Client) Stop() {
	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Msg("Stopping robot client")

	close(r.done)

	// 停止API服务器
	if r.apiServer != nil {
		r.apiServer.Stop()
	}

	// 停止WebSocket服务
	r.wsService.Stop()

	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Msg("Robot client stopped")
}

// onConnect 连接成功回调
func (r *Client) OnConnect() error {
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
func (r *Client) OnDisconnect() {
	log.Info().
		Str("ucode", r.config.Robot.UCode).
		Msg("Robot disconnected")
}

// onMessage 消息接收回调
func (r *Client) OnMessage(message []byte) error {
	var msg models.WebSocketMessage
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
func (r *Client) OnError(err error) {
	log.Error().
		Err(err).
		Str("ucode", r.config.Robot.UCode).
		Msg("Robot error occurred")
}

// getNextSequence 获取下一个序列号
func (r *Client) getNextSequence() int64 {
	r.seqMutex.Lock()
	defer r.seqMutex.Unlock()
	r.sequence++
	return r.sequence
}

// sendRegister 发送注册消息
func (r *Client) sendRegister() error {
	msg := models.WebSocketMessage{
		Type:       models.WSMessageTypeRequest,
		Command:    models.CMD_TYPE_REGISTER,
		ClientType: models.ClientTypeRobot,
		Sequence:   r.getNextSequence(),
		UCode:      r.config.Robot.UCode,
		Version:    r.config.Robot.Version,
		Data:       map[string]interface{}{},
	}

	return r.wsService.SendMessage(msg)
}

// sendPing 发送心跳消息
func (r *Client) sendPing() error {
	msg := models.WebSocketMessage{
		Type:       models.WSMessageTypeRequest,
		Command:    models.CMD_TYPE_PING,
		Sequence:   r.getNextSequence(),
		UCode:      r.config.Robot.UCode,
		ClientType: models.ClientTypeRobot,
		Version:    r.config.Robot.Version,
		Data:       map[string]interface{}{},
	}

	return r.wsService.SendMessage(msg)
}

// sendStatusUpdate 发送状态更新
func (r *Client) sendStatusUpdate() error {
	var status = r.robot.GetState()

	msg := models.WebSocketMessage{
		Type:       models.WSMessageTypeRequest,
		Command:    models.CMD_TYPE_UPDATE_ROBOT_STATUS,
		Sequence:   r.getNextSequence(),
		UCode:      r.config.Robot.UCode,
		ClientType: models.ClientTypeRobot,
		Version:    r.config.Robot.Version,
		Data:       status,
	}

	return r.wsService.SendMessage(msg)
}

// keepAlive 心跳保持
func (r *Client) keepAlive() {
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
func (r *Client) reportStatus() {
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
func (r *Client) GetStats() map[string]interface{} {
	stats := r.wsService.GetStats()
	stats["ucode"] = r.config.Robot.UCode
	stats["sequence"] = r.sequence
	return stats
}
