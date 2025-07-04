package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"remote-ctrl-robot/internal/handlers"
	"remote-ctrl-robot/internal/services"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func main() {
	// 加载配置
	if err := loadConfig(); err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// 设置日志
	setupLogging()

	// 创建服务
	janusService := services.NewJanusService(
		viper.GetString("janus.http_url"),
		viper.GetString("janus.websocket_url"),
		viper.GetInt("janus.stream_id"),
	)

	robotService := services.NewRobotService(
		viper.GetString("robot.websocket_url"),
	)

	// 创建处理器
	wsHandlers := handlers.NewWebSocketHandlers(robotService)
	apiHandlers := handlers.NewAPIHandlers(janusService, robotService, wsHandlers)

	// 创建HTTP服务器
	mux := http.NewServeMux()

	// API路由
	mux.HandleFunc("/api/v1/webrtc/play-url", apiHandlers.GetWebRTCPlayURL)
	mux.HandleFunc("/api/v1/webrtc/register", apiHandlers.RegisterWebRTC)
	mux.HandleFunc("/api/v1/webrtc/batch-register", apiHandlers.BatchRegisterWebRTC)
	mux.HandleFunc("/api/v1/webrtc/stats", apiHandlers.GetWebRTCStats)
	mux.HandleFunc("/api/v1/webrtc/cleanup", apiHandlers.CleanupWebRTCStreams)
	mux.HandleFunc("/api/v1/webrtc/all-play-urls", apiHandlers.GetAllWebRTCPlayURLs)
	mux.HandleFunc("/api/v1/control/command", apiHandlers.SendControlCommand)
	mux.HandleFunc("/api/v1/control/status", apiHandlers.GetRobotStatus)
	mux.HandleFunc("/api/v1/control/connection", apiHandlers.GetConnectionStatus)
	mux.HandleFunc("/api/v1/system/status", apiHandlers.GetSystemStatus)
	mux.HandleFunc("/api/v1/clients", apiHandlers.GetClients)
	mux.HandleFunc("/api/v1/clients/info", apiHandlers.GetClientByUCode)
	mux.HandleFunc("/api/v1/clients/online", apiHandlers.CheckUCodeOnline)
	mux.HandleFunc("/health", apiHandlers.HealthCheck)

	// WebSocket路由
	mux.HandleFunc("/ws/control", wsHandlers.HandleWebSocket)

	// 静态文件服务
	mux.HandleFunc("/test_client.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "test_client.html")
	})
	mux.HandleFunc("/mobile_operator.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "mobile_operator.html")
	})
	// 创建HTTP服务器
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", viper.GetString("server.host"), viper.GetInt("server.port")),
		Handler:      mux,
		ReadTimeout:  viper.GetDuration("server.read_timeout"),
		WriteTimeout: viper.GetDuration("server.write_timeout"),
	}

	// 启动服务器
	go func() {
		log.Info().Str("address", server.Addr).Msg("Starting server")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// 启动清理协程
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				robotService.CleanupDisconnectedClients()
				// 清理无效的WebRTC流
				cleaned := janusService.CleanupInactiveStreams()
				if cleaned > 0 {
					log.Info().Int("cleaned_streams", cleaned).Msg("Cleaned up inactive WebRTC streams")
				}
			}
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Error during server shutdown")
	}

	log.Info().Msg("Server stopped")
}

// 加载配置
func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config")
	viper.AddConfigPath("../config")
	viper.AddConfigPath("../../config")

	// 设置默认值
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("janus.websocket_url", "ws://localhost:8188")
	viper.SetDefault("janus.http_url", "http://localhost:8088")
	viper.SetDefault("janus.stream_id", 1)
	viper.SetDefault("robot.websocket_url", "ws://localhost:9090")
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("security.enable_cors", true)
	viper.SetDefault("security.allowed_origins", "*")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		log.Warn().Msg("No config file found, using defaults")
	}

	return nil
}

// 设置日志
func setupLogging() {
	level := viper.GetString("logging.level")
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(logLevel)

	if viper.GetString("logging.format") == "console" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	} else {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	}

	log.Info().
		Str("level", level).
		Str("format", viper.GetString("logging.format")).
		Msg("Logging configured")
}
