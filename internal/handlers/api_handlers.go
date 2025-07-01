package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"remote-ctrl-robot/internal/models"
	"remote-ctrl-robot/internal/services"

	"github.com/rs/zerolog/log"
)

type APIHandlers struct {
	janusService *services.JanusService
	robotService *services.RobotService
	wsHandlers   *WebSocketHandlers
	startTime    time.Time
}

func NewAPIHandlers(janusService *services.JanusService, robotService *services.RobotService, wsHandlers *WebSocketHandlers) *APIHandlers {
	return &APIHandlers{
		janusService: janusService,
		robotService: robotService,
		wsHandlers:   wsHandlers,
		startTime:    time.Now(),
	}
}

// 获取WebRTC播放地址
func (h *APIHandlers) GetWebRTCPlayURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从查询参数获取UCODE
	ucode := r.URL.Query().Get("ucode")
	if ucode == "" {
		h.sendJSONResponse(w, http.StatusBadRequest, models.WebRTCPlayURLResponse{
			Success: false,
			Message: "UCODE parameter is required",
		})
		return
	}

	log.Info().Str("ucode", ucode).Msg("Request for WebRTC play URLs")

	urls, err := h.janusService.GetWebRTCPlayURLs(ucode)
	if err != nil {
		log.Error().Err(err).Str("ucode", ucode).Msg("Failed to get WebRTC play URLs")
		h.sendJSONResponse(w, http.StatusInternalServerError, models.WebRTCPlayURLResponse{
			Success: false,
			Message: "Failed to get WebRTC play URLs: " + err.Error(),
		})
		return
	}

	h.sendJSONResponse(w, http.StatusOK, models.WebRTCPlayURLResponse{
		Success: true,
		URLs:    urls,
		Message: fmt.Sprintf("WebRTC play URLs retrieved successfully for robot %s", ucode),
	})
}

// 注册WebRTC流
func (h *APIHandlers) RegisterWebRTC(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request models.WebRTCRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Error().Err(err).Msg("Failed to parse WebRTC register request")
		h.sendJSONResponse(w, http.StatusBadRequest, models.WebRTCRegisterResponse{
			Success: false,
			Message: "Invalid request format: " + err.Error(),
		})
		return
	}

	// 验证UCODE
	if request.UCode == "" {
		h.sendJSONResponse(w, http.StatusBadRequest, models.WebRTCRegisterResponse{
			Success: false,
			Message: "UCODE is required",
		})
		return
	}

	log.Info().Str("ucode", request.UCode).Msg("Registering WebRTC stream")

	// 注册WebRTC流
	stream, err := h.janusService.RegisterWebRTCStream(request.UCode)
	if err != nil {
		log.Error().Err(err).Str("ucode", request.UCode).Msg("Failed to register WebRTC stream")
		h.sendJSONResponse(w, http.StatusInternalServerError, models.WebRTCRegisterResponse{
			Success: false,
			Message: "Failed to register WebRTC stream: " + err.Error(),
		})
		return
	}

	h.sendJSONResponse(w, http.StatusOK, models.WebRTCRegisterResponse{
		Success: true,
		Stream:  stream,
		Message: fmt.Sprintf("WebRTC stream registered successfully for robot %s", request.UCode),
	})
}

// 发送控制命令
func (h *APIHandlers) SendControlCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var command models.ControlCommand
	if err := json.NewDecoder(r.Body).Decode(&command); err != nil {
		log.Error().Err(err).Msg("Failed to parse control command")
		response := models.ControlResponse{
			Success: false,
			Error:   "Invalid command format: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 从查询参数获取UCODE
	ucode := r.URL.Query().Get("ucode")
	if ucode == "" {
		response := models.ControlResponse{
			Success: false,
			Error:   "UCODE parameter is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 检查机器人是否在线
	if !h.isRobotOnline(ucode) {
		response := models.ControlResponse{
			Success: false,
			Error:   fmt.Sprintf("Robot with UCODE %s is not online", ucode),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 验证命令
	if err := h.validateCommand(&command); err != nil {
		log.Error().Err(err).Str("ucode", ucode).Msg("Command validation failed")
		response := models.ControlResponse{
			Success: false,
			Error:   "Command validation failed: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 发送命令到指定机器人
	err := h.sendCommandToRobot(ucode, command)
	if err != nil {
		log.Error().Err(err).Str("ucode", ucode).Msg("Failed to send command to robot")
		response := models.ControlResponse{
			Success: false,
			Error:   "Failed to send command: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Info().
		Str("command_id", command.CommandID).
		Str("ucode", ucode).
		Msg("Control command sent successfully")

	response := models.ControlResponse{
		Success: true,
		Message: fmt.Sprintf("Command sent successfully to robot %s", ucode),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 获取机器人状态
func (h *APIHandlers) GetRobotStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从查询参数获取UCODE
	ucode := r.URL.Query().Get("ucode")
	if ucode == "" {
		response := models.RobotState{
			Status:       "error",
			ErrorCode:    400,
			ErrorMessage: "UCODE parameter is required",
			Timestamp:    time.Now().UnixMilli(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 检查机器人是否在线
	if !h.isRobotOnline(ucode) {
		response := models.RobotState{
			Status:       "offline",
			ErrorCode:    404,
			ErrorMessage: fmt.Sprintf("Robot with UCODE %s is not online", ucode),
			Timestamp:    time.Now().UnixMilli(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 这里应该从机器人获取实时状态
	// 目前返回模拟数据
	status := models.RobotState{
		BasePosition:    [3]float64{0.0, 0.0, 0.0},
		BaseOrientation: [4]float64{0.0, 0.0, 0.0, 1.0},
		BatteryLevel:    85.5,
		Temperature:     45.2,
		Status:          "idle",
		ErrorCode:       0,
		ErrorMessage:    "",
		Timestamp:       time.Now().UnixMilli(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// 获取系统状态
func (h *APIHandlers) GetSystemStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 检查Janus状态
	janusStatus := "healthy"
	if err := h.janusService.CheckStatus(); err != nil {
		janusStatus = "error"
		log.Error().Err(err).Msg("Janus status check failed")
	}

	// 获取机器人连接状态
	robotStatus := "connected"
	connStatus := h.robotService.GetConnectionStatus()
	if !connStatus.Connected {
		robotStatus = "disconnected"
	}

	// 获取在线机器人列表
	onlineRobots := h.getOnlineRobots()

	status := models.SystemStatus{
		ServerTime:    time.Now(),
		Uptime:        int64(time.Since(h.startTime).Seconds()),
		ActiveClients: h.robotService.GetActiveClientsCount(),
		RobotStatus:   robotStatus,
		JanusStatus:   janusStatus,
	}

	// 添加在线机器人信息
	statusData := map[string]interface{}{
		"server_time":    status.ServerTime,
		"uptime_seconds": status.Uptime,
		"active_clients": status.ActiveClients,
		"robot_status":   status.RobotStatus,
		"janus_status":   status.JanusStatus,
		"online_robots":  onlineRobots,
		"total_robots":   len(onlineRobots),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statusData)
}

// 获取连接状态
func (h *APIHandlers) GetConnectionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := h.robotService.GetConnectionStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// 验证控制命令
func (h *APIHandlers) validateCommand(command *models.ControlCommand) error {
	// 检查命令类型
	validTypes := map[string]bool{
		"joint_position": true,
		"velocity":       true,
		"emergency_stop": true,
		"home":           true,
	}

	if !validTypes[command.CommandID] {
		return fmt.Errorf("invalid command type: %s", command.CommandID)
	}

	// 检查优先级
	if command.Priority < 1 || command.Priority > 10 {
		return fmt.Errorf("priority must be between 1 and 10")
	}

	return nil
}

// 健康检查
func (h *APIHandlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(h.startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 检查机器人是否在线
func (h *APIHandlers) isRobotOnline(ucode string) bool {
	h.wsHandlers.mutex.RLock()
	defer h.wsHandlers.mutex.RUnlock()

	_, exists := h.wsHandlers.robotUcodeConn[ucode]
	return exists
}

// 发送命令到指定机器人
func (h *APIHandlers) sendCommandToRobot(ucode string, command models.ControlCommand) error {
	h.wsHandlers.mutex.RLock()
	conn, exists := h.wsHandlers.robotUcodeConn[ucode]
	h.wsHandlers.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("robot with UCODE %s is not online", ucode)
	}

	message := models.WebSocketMessage{
		Type:    "control_command",
		Message: "Control command from API",
		Data:    command,
	}

	return conn.WriteJSON(message)
}

// 获取在线机器人列表
func (h *APIHandlers) getOnlineRobots() []string {
	h.wsHandlers.mutex.RLock()
	defer h.wsHandlers.mutex.RUnlock()

	robots := make([]string, 0, len(h.wsHandlers.robotUcodeConn))
	for ucode := range h.wsHandlers.robotUcodeConn {
		robots = append(robots, ucode)
	}
	return robots
}

// 发送JSON响应的辅助方法
func (h *APIHandlers) sendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// 批量注册WebRTC流
func (h *APIHandlers) BatchRegisterWebRTC(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		UCodes []string `json:"ucodes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Error().Err(err).Msg("Failed to parse batch register request")
		h.sendJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Invalid request format: " + err.Error(),
		})
		return
	}

	if len(request.UCodes) == 0 {
		h.sendJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "UCodes list cannot be empty",
		})
		return
	}

	log.Info().Strs("ucodes", request.UCodes).Msg("Batch registering WebRTC streams")

	results := h.janusService.BatchRegisterWebRTCStreams(request.UCodes)

	successCount := len(results)
	failedCount := len(request.UCodes) - successCount

	h.sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"streams":       results,
		"success_count": successCount,
		"failed_count":  failedCount,
		"message":       fmt.Sprintf("Batch registration completed: %d success, %d failed", successCount, failedCount),
	})
}

// 获取WebRTC流统计
func (h *APIHandlers) GetWebRTCStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.janusService.GetStreamStats()
	allStreams := h.janusService.GetAllStreams()

	response := map[string]interface{}{
		"success": true,
		"stats":   stats,
		"streams": allStreams,
		"message": "WebRTC statistics retrieved successfully",
	}

	h.sendJSONResponse(w, http.StatusOK, response)
}

// 清理无效WebRTC流
func (h *APIHandlers) CleanupWebRTCStreams(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cleaned := h.janusService.CleanupInactiveStreams()

	h.sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"cleaned_count": cleaned,
		"message":       fmt.Sprintf("Cleaned up %d inactive streams", cleaned),
	})
}

// 获取所有播放地址
func (h *APIHandlers) GetAllWebRTCPlayURLs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	allURLs := h.janusService.GetAllWebRTCPlayURLs()

	h.sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"urls":    allURLs,
		"message": "All WebRTC play URLs retrieved successfully",
	})
}

// 获取客户端列表
func (h *APIHandlers) GetClients(w http.ResponseWriter, r *http.Request) {
	clients := h.wsHandlers.GetAllClients()

	response := map[string]interface{}{
		"success": true,
		"clients": clients,
		"stats": map[string]interface{}{
			"total":     len(clients),
			"robots":    h.wsHandlers.GetClientsCountByType(models.ClientTypeRobot),
			"operators": h.wsHandlers.GetClientsCountByType(models.ClientTypeOperator),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 获取指定UCode的客户端信息
func (h *APIHandlers) GetClientByUCode(w http.ResponseWriter, r *http.Request) {
	ucode := r.URL.Query().Get("ucode")
	if ucode == "" {
		http.Error(w, "UCode parameter is required", http.StatusBadRequest)
		return
	}

	client := h.wsHandlers.GetClientByUCode(ucode)
	if client == nil {
		response := map[string]interface{}{
			"success": false,
			"message": "Client not found",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"client":  client,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 检查UCode是否在线
func (h *APIHandlers) CheckUCodeOnline(w http.ResponseWriter, r *http.Request) {
	ucode := r.URL.Query().Get("ucode")
	if ucode == "" {
		http.Error(w, "UCode parameter is required", http.StatusBadRequest)
		return
	}

	isOnline := h.wsHandlers.IsUCodeOnline(ucode)

	response := map[string]interface{}{
		"success": true,
		"ucode":   ucode,
		"online":  isOnline,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
