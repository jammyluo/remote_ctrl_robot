package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"remote-ctrl-robot/internal/models"
	"remote-ctrl-robot/internal/services"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type WebSocketHandlers struct {
	robotService *services.RobotService
	upgrader     websocket.Upgrader
	clients      map[*websocket.Conn]bool
	ucodeConnMap map[string]*websocket.Conn
	connUcodeMap map[*websocket.Conn]string
	mutex        sync.RWMutex
}

func NewWebSocketHandlers(robotService *services.RobotService) *WebSocketHandlers {
	return &WebSocketHandlers{
		robotService: robotService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源，生产环境应该限制
			},
		},
		clients:      make(map[*websocket.Conn]bool),
		ucodeConnMap: make(map[string]*websocket.Conn),
		connUcodeMap: make(map[*websocket.Conn]string),
	}
}

// 处理WebSocket连接
func (h *WebSocketHandlers) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade connection to WebSocket")
		return
	}
	defer conn.Close()

	// 注册：第一条消息必须为register，带ucode
	var rawMsg map[string]interface{}
	if err := conn.ReadJSON(&rawMsg); err != nil {
		conn.WriteJSON(models.WebSocketMessage{Type: "error", Message: "Failed to parse registration message"})
		return
	}

	// 检查消息类型
	msgType, ok := rawMsg["type"].(string)
	if !ok || msgType != "register" {
		conn.WriteJSON(models.WebSocketMessage{Type: "error", Message: "UCODE required, please send register message first"})
		return
	}

	// 获取UCODE
	ucode, ok := rawMsg["ucode"].(string)
	if !ok || ucode == "" {
		conn.WriteJSON(models.WebSocketMessage{Type: "error", Message: "Invalid UCODE"})
		return
	}

	// 绑定 UCODE
	h.mutex.Lock()
	h.ucodeConnMap[ucode] = conn
	h.connUcodeMap[conn] = ucode
	h.clients[conn] = true
	h.mutex.Unlock()

	log.Info().Str("ucode", ucode).Str("remote_addr", conn.RemoteAddr().String()).Msg("WebSocket client registered")

	welcomeMsg := models.WebSocketMessage{
		Type:    "welcome",
		Message: fmt.Sprintf("Connected as UCODE %s", ucode),
		Data: map[string]interface{}{
			"timestamp": time.Now().UnixMilli(),
			"ucode":     ucode,
		},
	}
	if err := conn.WriteJSON(welcomeMsg); err != nil {
		log.Error().Err(err).Msg("Failed to send welcome message")
		return
	}

	// 处理后续消息
	for {
		var msg models.WebSocketMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Msg("WebSocket read error")
			}
			break
		}
		h.handleMessage(conn, msg)
	}

	// 清理连接
	h.mutex.Lock()
	delete(h.clients, conn)
	if u, ok := h.connUcodeMap[conn]; ok {
		delete(h.ucodeConnMap, u)
		delete(h.connUcodeMap, conn)
	}
	h.mutex.Unlock()

	log.Info().Str("ucode", ucode).Str("remote_addr", conn.RemoteAddr().String()).Msg("WebSocket client disconnected")
}

// 处理WebSocket消息
func (h *WebSocketHandlers) handleMessage(conn *websocket.Conn, msg models.WebSocketMessage) {
	log.Debug().
		Str("type", msg.Type).
		Str("client", conn.RemoteAddr().String()).
		Msg("Received WebSocket message")

	switch msg.Type {
	case "control_command":
		h.handleControlCommand(conn, msg)
	case "status_request":
		h.handleStatusRequest(conn, msg)
	case "ping":
		h.handlePing(conn, msg)
	default:
		h.sendError(conn, "unknown_message_type", "Unknown message type: "+msg.Type)
	}
}

// 处理控制命令
func (h *WebSocketHandlers) handleControlCommand(conn *websocket.Conn, msg models.WebSocketMessage) {
	// 解析控制命令
	commandData, ok := msg.Data.(map[string]interface{})
	if !ok {
		h.sendError(conn, "invalid_command_format", "Command data must be an object")
		return
	}

	// 转换为ControlCommand
	commandJSON, err := json.Marshal(commandData)
	if err != nil {
		h.sendError(conn, "command_serialization_error", "Failed to serialize command")
		return
	}

	var command models.ControlCommand
	if err := json.Unmarshal(commandJSON, &command); err != nil {
		h.sendError(conn, "command_parsing_error", "Failed to parse command: "+err.Error())
		return
	}

	// 发送命令到机器人
	err = h.robotService.SendCommand(command)
	if err != nil {
		h.sendError(conn, "command_send_error", "Failed to send command: "+err.Error())
		return
	}

	// 发送成功响应
	response := models.WebSocketMessage{
		Type:    "command_response",
		Message: "Command sent successfully",
		Data: map[string]interface{}{
			"command_id": command.CommandID,
			"timestamp":  time.Now().UnixMilli(),
		},
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Error().Err(err).Msg("Failed to send command response")
	}
}

// 处理状态请求
func (h *WebSocketHandlers) handleStatusRequest(conn *websocket.Conn, msg models.WebSocketMessage) {
	// 获取机器人状态
	status := h.robotService.GetConnectionStatus()

	response := models.WebSocketMessage{
		Type:    "status_response",
		Message: "Robot status retrieved",
		Data:    status,
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Error().Err(err).Msg("Failed to send status response")
	}
}

// 处理ping消息
func (h *WebSocketHandlers) handlePing(conn *websocket.Conn, msg models.WebSocketMessage) {
	response := models.WebSocketMessage{
		Type:    "pong",
		Message: "Pong",
		Data: map[string]interface{}{
			"timestamp": time.Now().UnixMilli(),
		},
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Error().Err(err).Msg("Failed to send pong response")
	}
}

// 发送错误消息
func (h *WebSocketHandlers) sendError(conn *websocket.Conn, errorType, message string) {
	errorMsg := models.WebSocketMessage{
		Type:    "error",
		Message: message,
		Data: map[string]interface{}{
			"error_type": errorType,
			"timestamp":  time.Now().UnixMilli(),
		},
	}

	if err := conn.WriteJSON(errorMsg); err != nil {
		log.Error().Err(err).Msg("Failed to send error message")
	}
}

// 广播消息给所有客户端
func (h *WebSocketHandlers) Broadcast(message models.WebSocketMessage) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for conn := range h.clients {
		if err := conn.WriteJSON(message); err != nil {
			log.Error().Err(err).Msg("Failed to broadcast message")
			// 标记连接为无效，稍后清理
			conn.Close()
		}
	}
}

// 获取活跃客户端数量
func (h *WebSocketHandlers) GetActiveClientsCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}
