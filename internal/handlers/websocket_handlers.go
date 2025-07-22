package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"remote-ctrl-robot/internal/models"
	"remote-ctrl-robot/internal/services"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type WebSocketHandlers struct {
	upgrader websocket.Upgrader

	// 连接管理
	ctx             context.Context
	cancel          context.CancelFunc
	operatorManager *services.OperatorManager
	robotManager    *services.RobotManager
}

func NewWebSocketHandlers(robotManager *services.RobotManager, operatorManager *services.OperatorManager, gameService *services.GameService) *WebSocketHandlers {
	ctx, cancel := context.WithCancel(context.Background())
	return &WebSocketHandlers{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		ctx:             ctx,
		cancel:          cancel,
		operatorManager: operatorManager,
		robotManager:    robotManager,
	}
}

func (h *WebSocketHandlers) GetOperatorByUcode(ucode string) *services.WebSocketService {
	operator, err := h.operatorManager.GetOperator(ucode)
	if err != nil {
		return nil
	}
	return operator
}

func (h *WebSocketHandlers) checkWSMessage(msg models.WebSocketMessage) error {
	// 检查消息类型
	if msg.Type != models.WSMessageTypeRequest {
		return errors.New("invalid message type")
	}

	// 获取UCode
	if msg.UCode == "" {
		return errors.New("invalid ucode")
	}

	// 获取客户端类型
	if msg.ClientType == "" {
		return errors.New("invalid client type")
	}

	// 验证客户端类型
	if msg.ClientType != models.ClientTypeRobot && msg.ClientType != models.ClientTypeOperator {
		return errors.New("invalid client type")
	}

	if msg.Sequence == 0 {
		return errors.New("invalid sequence")
	}

	if msg.Version == "" {
		return errors.New("invalid version")
	}
	return nil
}

// 处理WebSocket连接
func (h *WebSocketHandlers) HandleWebSocket(w http.ResponseWriter, r *http.Request) {

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade connection to WebSocket")
		return
	}

	log.Info().Msg("WebSocket connection received conn: " + conn.RemoteAddr().String())

	// 设置连接参数
	conn.SetReadLimit(512 * 1024)                          // 512KB 读取限制
	conn.SetReadDeadline(time.Now().Add(30 * time.Second)) // 30秒读取超时
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		return nil
	})

	// 创建WebSocket服务
	wsService := services.NewWebSocketService(conn)

	// 注册：第一条消息必须为register
	var msg models.WebSocketMessage
	if err := conn.ReadJSON(&msg); err != nil {
		wsService.SendError(&msg, "Failed to parse registration message")
		conn.Close()
		return
	}
	if msg.Command != models.CMD_TYPE_REGISTER {
		wsService.SendError(&msg, "Invalid message type")
		conn.Close()
		return
	}

	err = h.checkWSMessage(msg)
	if err != nil {
		wsService.SendError(&msg, err.Error())
		conn.Close()
		return
	}

	h.handleRegistration(wsService, &msg)
}

// 注册 - 返回是否成功
func (h *WebSocketHandlers) handleRegistration(wsService *services.WebSocketService, msg *models.WebSocketMessage) bool {

	// 设置身份信息
	wsService.SetIdentity(msg.UCode, msg.Version)

	// 如果是机器人客户端，注册到机器人管理器
	if msg.ClientType == models.ClientTypeRobot {
		// 创建新机器人
		_, err := h.robotManager.RegisterRobot(wsService)
		if err != nil {
			wsService.SendError(msg, err.Error())
			log.Error().Err(err).Str("ucode", msg.UCode).Msg("Failed to register robot")
			return false
		}

		wsService.SendSuccess(msg, "RegisterRobot successful")
		// 设置机器人消息处理器回调
		wsService.SetMessageHandler(h.createRobotMessageHandler())
		// 设置机器人断开连接处理器回调
		wsService.SetDisconnectHandler(h.createRobotDisconnectHandler())
	}

	// 如果是操作员客户端，添加到客户端管理器
	if msg.ClientType == models.ClientTypeOperator {
		if err := h.operatorManager.RegisterOperator(wsService); err != nil {
			wsService.SendError(msg, err.Error())
			log.Error().Err(err).Str("ucode", msg.UCode).Msg("Failed to add client")
			return false
		}

		wsService.SendSuccess(msg, "RegisterOperator successful")
		// 设置客户端消息处理器回调
		wsService.SetMessageHandler(h.createClientMessageHandler())

		// 设置操作员断开连接处理器回调
		wsService.SetDisconnectHandler(h.createOperatorDisconnectHandler())
	}

	// 启动WebSocket服务
	if err := wsService.Start(); err != nil {
		log.Error().Err(err).Str("ucode", msg.UCode).Msg("Failed to start WebSocket service")
		wsService.SendError(msg, "Failed to start WebSocket service")
		return false
	}

	log.Info().
		Str("ucode", msg.UCode).
		Str("type", string(msg.ClientType)).
		Str("version", msg.Version).
		Msg("Client registered successfully")

	return true
}

// createRobotMessageHandler 创建机器人消息处理器回调
func (h *WebSocketHandlers) createRobotMessageHandler() services.MessageHandler {
	return func(service *services.WebSocketService, message *models.WebSocketMessage) error {
		return h.robotManager.HandleMessage(message)
	}
}

// createClientMessageHandler 创建客户端消息处理器回调
func (h *WebSocketHandlers) createClientMessageHandler() services.MessageHandler {
	return func(service *services.WebSocketService, message *models.WebSocketMessage) error {
		return h.operatorManager.HandleMessage(message)
	}
}

// createRobotDisconnectHandler 创建机器人断开连接处理器回调
func (h *WebSocketHandlers) createRobotDisconnectHandler() services.DisconnectHandler {
	return func(service *services.WebSocketService) {
		log.Info().
			Str("remote_addr", service.RemoteAddr).
			Msg("Robot disconnected, cleaning up...")

		// 通知机器人管理器进行清理
		h.robotManager.UnregisterRobot(service)
	}
}

// createOperatorDisconnectHandler 创建操作员断开连接处理器回调
func (h *WebSocketHandlers) createOperatorDisconnectHandler() services.DisconnectHandler {
	return func(service *services.WebSocketService) {
		log.Info().
			Str("remote_addr", service.RemoteAddr).
			Msg("Operator disconnected, cleaning up...")

		// 通知操作员管理器进行清理
		h.operatorManager.UnregisterOperator(service)
	}
}
