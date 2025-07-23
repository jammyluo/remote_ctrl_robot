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

	// 注册路由
	mux.HandleFunc("/api/v1/name", api.handleName)
	mux.HandleFunc("/api/v1/status", api.handleStatus)

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

// handleName 处理名称相关请求
func (api *APIServer) handleName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		api.handleGetName(w, r)
	case "POST":
		api.handleSetName(w, r)
	default:
		api.sendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGetName 处理获取名称请求
func (api *APIServer) handleGetName(w http.ResponseWriter, r *http.Request) {
	api.mutex.RLock()
	defer api.mutex.RUnlock()

	response := APIResponse{
		Success: true,
		Message: "获取名称成功",
		Data: map[string]string{
			"name": "test",
		},
	}

	api.sendResponse(w, http.StatusOK, response)
}

// handleSetName 处理设置名称请求
func (api *APIServer) handleSetName(w http.ResponseWriter, r *http.Request) {
	var req NameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendErrorResponse(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	if req.Name == "" {
		api.sendErrorResponse(w, http.StatusBadRequest, "名称不能为空")
		return
	}

	api.mutex.Lock()
	defer api.mutex.Unlock()

	response := APIResponse{
		Success: true,
		Message: "设置名称成功",
		Data: map[string]string{
			"name": req.Name,
		},
	}

	api.sendResponse(w, http.StatusOK, response)
}

// handleStatus 处理状态查询请求
func (api *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		api.sendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	api.mutex.RLock()
	defer api.mutex.RUnlock()

	response := APIResponse{
		Success: true,
		Message: "获取状态成功",
		Data:    nil,
	}

	api.sendResponse(w, http.StatusOK, response)
}

// sendResponse 发送成功响应
func (api *APIServer) sendResponse(w http.ResponseWriter, statusCode int, response APIResponse) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
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
