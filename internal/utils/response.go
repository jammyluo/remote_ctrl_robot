package utils

import (
	"time"

	"remote-ctrl-robot/internal/models"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// ResponseBuilder WebSocket响应构建器
type ResponseBuilder struct {
}

// SendSuccess 发送成功响应
func SendSuccess(conn *websocket.Conn, originalMsg *models.WebSocketMessage, message string) error {
	return sendResponse(conn, originalMsg, message, true)
}

// SendError 发送错误响应
func SendError(conn *websocket.Conn, originalMsg *models.WebSocketMessage, message string) error {
	return sendResponse(conn, originalMsg, message, false)
}

// SendCustom 发送自定义响应
func SendCustom(conn *websocket.Conn, originalMsg *models.WebSocketMessage, data interface{}) error {
	response := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    originalMsg.Command,
		Sequence:   originalMsg.Sequence,
		UCode:      originalMsg.UCode,
		ClientType: originalMsg.ClientType,
		Version:    originalMsg.Version,
		Data:       data,
	}

	return conn.WriteJSON(response)
}

// sendResponse 发送统一响应
func sendResponse(conn *websocket.Conn, originalMsg *models.WebSocketMessage, message string, success bool) error {
	response := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    originalMsg.Command,
		Sequence:   originalMsg.Sequence,
		UCode:      originalMsg.UCode,
		ClientType: originalMsg.ClientType,
		Version:    originalMsg.Version,
		Data: models.CMD_RESPONSE{
			Success:   success,
			Message:   message,
			Timestamp: time.Now().Unix(),
		},
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Error().
			Err(err).
			Str("ucode", originalMsg.UCode).
			Str("command", string(originalMsg.Command)).
			Msg("Failed to send response")
		return err
	}

	log.Debug().
		Str("ucode", originalMsg.UCode).
		Str("command", string(originalMsg.Command)).
		Bool("success", success).
		Msg("Response sent successfully")

	return nil
}

// SendControlResponse 发送控制命令响应
func SendControlResponse(conn *websocket.Conn, originalMsg *models.WebSocketMessage, response *models.RobotCommandResponse) error {
	wsResponse := models.WebSocketMessage{
		Type:       models.WSMessageTypeResponse,
		Command:    models.CMD_TYPE_CONTROL_ROBOT,
		Sequence:   originalMsg.Sequence,
		UCode:      originalMsg.UCode,
		ClientType: originalMsg.ClientType,
		Version:    originalMsg.Version,
		Data:       response,
	}

	if err := conn.WriteJSON(wsResponse); err != nil {
		log.Error().
			Err(err).
			Str("ucode", originalMsg.UCode).
			Msg("Failed to send control response")
		return err
	}

	log.Debug().
		Str("ucode", originalMsg.UCode).
		Bool("success", response.Success).
		Msg("Control response sent successfully")

	return nil
}
