package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 配置结构
type Config struct {
	Robot       RobotConfig       `yaml:"robot"`
	Server      ServerConfig      `yaml:"server"`
	Heartbeat   HeartbeatConfig   `yaml:"heartbeat"`
	Status      StatusConfig      `yaml:"status"`
	Logging     LoggingConfig     `yaml:"logging"`
	Reconnect   ReconnectConfig   `yaml:"reconnect"`
	Security    SecurityConfig    `yaml:"security"`
	Performance PerformanceConfig `yaml:"performance"`
}

// RobotConfig 机器人配置
type RobotConfig struct {
	UCode      string `yaml:"ucode"`
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	ClientType string `yaml:"client_type"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	URL            string `yaml:"url"`
	ConnectTimeout int    `yaml:"connect_timeout"`
	ReadTimeout    int    `yaml:"read_timeout"`
	WriteTimeout   int    `yaml:"write_timeout"`
}

// HeartbeatConfig 心跳配置
type HeartbeatConfig struct {
	Interval int `yaml:"interval"`
	Timeout  int `yaml:"timeout"`
}

// StatusConfig 状态配置
type StatusConfig struct {
	Interval         int  `yaml:"interval"`
	EnableSimulation bool `yaml:"enable_simulation"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	FilePath   string `yaml:"file_path"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
}

// ReconnectConfig 重连配置
type ReconnectConfig struct {
	Enabled           bool    `yaml:"enabled"`
	MaxAttempts       int     `yaml:"max_attempts"`
	InitialDelay      int     `yaml:"initial_delay"`
	MaxDelay          int     `yaml:"max_delay"`
	BackoffMultiplier float64 `yaml:"backoff_multiplier"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	EnableTLS  bool   `yaml:"enable_tls"`
	SkipVerify bool   `yaml:"skip_verify"`
	AuthToken  string `yaml:"auth_token"`
}

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	MessageBufferSize int  `yaml:"message_buffer_size"`
	WorkerPoolSize    int  `yaml:"worker_pool_size"`
	EnableMetrics     bool `yaml:"enable_metrics"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 设置默认值
	config.setDefaults()

	// 应用环境变量覆盖
	config.applyEnvOverrides()

	// 验证配置
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}

	return &config, nil
}

// setDefaults 设置默认值
func (c *Config) setDefaults() {
	// 机器人配置默认值
	if c.Robot.UCode == "" {
		c.Robot.UCode = "robot_001"
	}
	if c.Robot.Name == "" {
		c.Robot.Name = "测试机器人"
	}
	if c.Robot.Version == "" {
		c.Robot.Version = "1.0.0"
	}
	if c.Robot.ClientType == "" {
		c.Robot.ClientType = "robot"
	}

	// 服务器配置默认值
	if c.Server.URL == "" {
		c.Server.URL = "ws://localhost:8000/ws/control"
	}
	if c.Server.ConnectTimeout == 0 {
		c.Server.ConnectTimeout = 30
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 10
	}

	// 心跳配置默认值
	if c.Heartbeat.Interval == 0 {
		c.Heartbeat.Interval = 30
	}
	if c.Heartbeat.Timeout == 0 {
		c.Heartbeat.Timeout = 10
	}

	// 状态配置默认值
	if c.Status.Interval == 0 {
		c.Status.Interval = 10
	}

	// 日志配置默认值
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
	if c.Logging.Output == "" {
		c.Logging.Output = "stdout"
	}
	if c.Logging.MaxSize == 0 {
		c.Logging.MaxSize = 100
	}
	if c.Logging.MaxBackups == 0 {
		c.Logging.MaxBackups = 3
	}
	if c.Logging.MaxAge == 0 {
		c.Logging.MaxAge = 28
	}

	// 重连配置默认值
	if c.Reconnect.MaxAttempts == 0 {
		c.Reconnect.MaxAttempts = 5
	}
	if c.Reconnect.InitialDelay == 0 {
		c.Reconnect.InitialDelay = 1
	}
	if c.Reconnect.MaxDelay == 0 {
		c.Reconnect.MaxDelay = 60
	}
	if c.Reconnect.BackoffMultiplier == 0 {
		c.Reconnect.BackoffMultiplier = 2.0
	}

	// 性能配置默认值
	if c.Performance.MessageBufferSize == 0 {
		c.Performance.MessageBufferSize = 1000
	}
	if c.Performance.WorkerPoolSize == 0 {
		c.Performance.WorkerPoolSize = 4
	}
}

// applyEnvOverrides 应用环境变量覆盖
func (c *Config) applyEnvOverrides() {
	// 机器人配置
	if env := os.Getenv("ROBOT_UCODE"); env != "" {
		c.Robot.UCode = env
	}
	if env := os.Getenv("ROBOT_NAME"); env != "" {
		c.Robot.Name = env
	}
	if env := os.Getenv("ROBOT_VERSION"); env != "" {
		c.Robot.Version = env
	}

	// 服务器配置
	if env := os.Getenv("ROBOT_SERVER_URL"); env != "" {
		c.Server.URL = env
	}
	if env := os.Getenv("ROBOT_CONNECT_TIMEOUT"); env != "" {
		if val, err := parseInt(env); err == nil {
			c.Server.ConnectTimeout = val
		}
	}

	// 日志配置
	if env := os.Getenv("ROBOT_LOG_LEVEL"); env != "" {
		c.Logging.Level = env
	}
	if env := os.Getenv("ROBOT_LOG_FORMAT"); env != "" {
		c.Logging.Format = env
	}

	// 心跳配置
	if env := os.Getenv("ROBOT_HEARTBEAT_INTERVAL"); env != "" {
		if val, err := parseInt(env); err == nil {
			c.Heartbeat.Interval = val
		}
	}

	// 状态配置
	if env := os.Getenv("ROBOT_STATUS_INTERVAL"); env != "" {
		if val, err := parseInt(env); err == nil {
			c.Status.Interval = val
		}
	}

	// 重连配置
	if env := os.Getenv("ROBOT_RECONNECT_ENABLED"); env != "" {
		c.Reconnect.Enabled = env == "true"
	}
	if env := os.Getenv("ROBOT_MAX_RECONNECT_ATTEMPTS"); env != "" {
		if val, err := parseInt(env); err == nil {
			c.Reconnect.MaxAttempts = val
		}
	}
}

// validate 验证配置
func (c *Config) validate() error {
	// 验证机器人配置
	if c.Robot.UCode == "" {
		return fmt.Errorf("机器人UCode不能为空")
	}

	// 验证服务器配置
	if c.Server.URL == "" {
		return fmt.Errorf("服务器URL不能为空")
	}

	// 验证心跳配置
	if c.Heartbeat.Interval <= 0 {
		return fmt.Errorf("心跳间隔必须大于0")
	}

	// 验证状态配置
	if c.Status.Interval <= 0 {
		return fmt.Errorf("状态上报间隔必须大于0")
	}

	// 验证日志配置
	if c.Logging.Level != "debug" && c.Logging.Level != "info" &&
		c.Logging.Level != "warn" && c.Logging.Level != "error" {
		return fmt.Errorf("无效的日志级别: %s", c.Logging.Level)
	}

	if c.Logging.Format != "json" && c.Logging.Format != "console" {
		return fmt.Errorf("无效的日志格式: %s", c.Logging.Format)
	}

	return nil
}

// GetConnectTimeout 获取连接超时时间
func (c *Config) GetConnectTimeout() time.Duration {
	return time.Duration(c.Server.ConnectTimeout) * time.Second
}

// GetReadTimeout 获取读取超时时间
func (c *Config) GetReadTimeout() time.Duration {
	return time.Duration(c.Server.ReadTimeout) * time.Second
}

// GetWriteTimeout 获取写入超时时间
func (c *Config) GetWriteTimeout() time.Duration {
	return time.Duration(c.Server.WriteTimeout) * time.Second
}

// GetHeartbeatInterval 获取心跳间隔
func (c *Config) GetHeartbeatInterval() time.Duration {
	return time.Duration(c.Heartbeat.Interval) * time.Second
}

// GetStatusInterval 获取状态上报间隔
func (c *Config) GetStatusInterval() time.Duration {
	return time.Duration(c.Status.Interval) * time.Second
}

// parseInt 解析整数
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
