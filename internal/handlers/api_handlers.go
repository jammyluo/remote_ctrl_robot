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
		response := models.WebRTCPlayURLResponse{
			Success: false,
			Message: "UCODE parameter is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 检查机器人是否在线
	if !h.isRobotOnline(ucode) {
		response := models.WebRTCPlayURLResponse{
			Success: false,
			Message: fmt.Sprintf("Robot with UCODE %s is not online", ucode),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Info().Str("ucode", ucode).Msg("Request for WebRTC play URL")

	url, err := h.janusService.GetWebRTCPlayURL()
	if err != nil {
		log.Error().Err(err).Str("ucode", ucode).Msg("Failed to get WebRTC play URL")
		response := models.WebRTCPlayURLResponse{
			Success: false,
			Message: "Failed to get WebRTC play URL: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := models.WebRTCPlayURLResponse{
		Success: true,
		URL:     url,
		Message: fmt.Sprintf("WebRTC play URL generated successfully for robot %s", ucode),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
			Success:   false,
			CommandID: command.CommandID,
			Error:     "Failed to send command: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Info().
		Str("command_id", command.CommandID).
		Str("command_type", command.Type).
		Str("ucode", ucode).
		Msg("Control command sent successfully")

	response := models.ControlResponse{
		Success:   true,
		CommandID: command.CommandID,
		Message:   fmt.Sprintf("Command sent successfully to robot %s", ucode),
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
		JointPositions:  []float64{0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
		JointVelocities: []float64{0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
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

	if !validTypes[command.Type] {
		return fmt.Errorf("invalid command type: %s", command.Type)
	}

	// 检查优先级
	if command.Priority < 1 || command.Priority > 10 {
		return fmt.Errorf("priority must be between 1 and 10")
	}

	// 检查关节位置数量（假设6自由度机器人）
	if command.Type == "joint_position" && len(command.JointPos) != 6 {
		return fmt.Errorf("joint positions must have exactly 6 values")
	}

	// 检查关节速度数量
	if command.Type == "velocity" && len(command.Velocities) != 6 {
		return fmt.Errorf("velocities must have exactly 6 values")
	}

	// 检查关节角度范围（-π 到 π）
	for i, pos := range command.JointPos {
		if pos < -3.14159 || pos > 3.14159 {
			return fmt.Errorf("joint position %d out of range (-π to π): %f", i, pos)
		}
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

	_, exists := h.wsHandlers.ucodeConnMap[ucode]
	return exists
}

// 发送命令到指定机器人
func (h *APIHandlers) sendCommandToRobot(ucode string, command models.ControlCommand) error {
	h.wsHandlers.mutex.RLock()
	conn, exists := h.wsHandlers.ucodeConnMap[ucode]
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

	robots := make([]string, 0, len(h.wsHandlers.ucodeConnMap))
	for ucode := range h.wsHandlers.ucodeConnMap {
		robots = append(robots, ucode)
	}
	return robots
}
