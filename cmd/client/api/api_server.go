package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/rs/zerolog/log"
)

// APIServer HTTP API服务器
type APIServer struct {
	client *Client
	port   int
	server *http.Server
	mutex  sync.RWMutex
}

// APIResponse API响应结构
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NameRequest 名称请求结构
type NameRequest struct {
	Name string `json:"name"`
}

// NewAPIServer 创建新的API服务器
func NewAPIServer(client *Client, port int) *APIServer {
	return &APIServer{
		client: client,
		port:   port,
	}
}

// Start 启动API服务器
func (api *APIServer) Start() error {
	mux := http.NewServeMux()

	// 射击
	mux.HandleFunc("/api/v1/shoot", api.handleShoot)
	// 获取弹药数量
	mux.HandleFunc("/api/v1/ammo", api.handleAmmo)
	// 更换弹药
	mux.HandleFunc("/api/v1/ammo/change", api.handleAmmoChange)
	// 获取生命值
	mux.HandleFunc("/api/v1/health", api.handleHealth)

	api.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", api.port),
		Handler: mux,
	}

	log.Info().
		Int("port", api.port).
		Msg("Starting HTTP API server")

	return api.server.ListenAndServe()
}

// Stop 停止API服务器
func (api *APIServer) Stop() error {
	if api.server != nil {
		log.Info().Msg("Stopping HTTP API server")
		return api.server.Close()
	}
	return nil
}

// handleShoot 处理射击请求
func (api *APIServer) handleShoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		api.sendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	err := api.client.robot.Shoot()
	if err != nil {
		api.sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.sendSuccessResponse(w, http.StatusOK, "Success")
}

// handleAmmo 处理弹药查询请求
func (api *APIServer) handleAmmo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		api.sendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	api.mutex.RLock()
	defer api.mutex.RUnlock()

	response := APIResponse{
		Success: true,
		Message: "Success",
		Data:    api.client.robot.GetAmmo(),
	}

	api.sendResponse(w, http.StatusOK, response)
}

// handleAmmoChange 处理更换弹药请求
func (api *APIServer) handleAmmoChange(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		api.sendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	api.client.robot.AmmoChange()

	api.sendSuccessResponse(w, http.StatusOK, "Success")
}

// handleHealth 处理生命值查询请求
func (api *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := APIResponse{
		Success: true,
		Message: "Success",
		Data: map[string]int{
			"health": api.client.robot.GetHealth(),
		},
	}

	api.sendResponse(w, http.StatusOK, response)
}

// sendResponse 发送成功响应
func (api *APIServer) sendResponse(w http.ResponseWriter, statusCode int, response APIResponse) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// sendSuccessResponse 发送成功响应
func (api *APIServer) sendSuccessResponse(w http.ResponseWriter, statusCode int, message string) {
	response := APIResponse{
		Success: true,
		Message: message,
	}

	api.sendResponse(w, statusCode, response)
}

// sendErrorResponse 发送错误响应
func (api *APIServer) sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := APIResponse{
		Success: false,
		Message: message,
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
