package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"remote-ctrl-robot/cmd/client/api"
	"remote-ctrl-robot/cmd/client/config"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// initLogger 初始化日志系统
func initLogger(config *config.Config) {
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
	config, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Load config failed: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	initLogger(config)

	log.Info().
		Str("ucode", config.Robot.UCode).
		Str("server", config.WebSocket.URL).
		Str("log_level", config.Logging.Level).
		Str("config_file", *configPath).
		Msg("Start robot client")

	// 创建机器人客户端
	client := api.NewClient(config)

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
