package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type MockRobotConfig struct {
	AmmoCount    int `yaml:"ammo_count"`
	AmmoCapacity int `yaml:"ammo_capacity"`
	Health       int `yaml:"health"`
}

// Config 配置结构
type Config struct {
	Robot     RobotConfig     `yaml:"robot"`
	WebSocket WebSocketConfig `yaml:"websocket"`
	Heartbeat HeartbeatConfig `yaml:"heartbeat"`
	Status    StatusConfig    `yaml:"status"`
	Logging   LoggingConfig   `yaml:"logging"`
	MockRobot MockRobotConfig `yaml:"mock_robot"`
	API       APIConfig       `yaml:"api"`
}

// RobotConfig 机器人配置
type RobotConfig struct {
	UCode      string `yaml:"ucode"`
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	ClientType string `yaml:"client_type"`
}

// APIConfig API配置
type APIConfig struct {
	Enable bool `yaml:"enable"`
	Port   int  `yaml:"port"`
}

// ServerConfig 服务器配置
type WebSocketConfig struct {
	Enable         bool   `yaml:"enable"`
	URL            string `yaml:"url"`
	ConnectTimeout int    `yaml:"connect_timeout"`
	ReadTimeout    int    `yaml:"read_timeout"`
	WriteTimeout   int    `yaml:"write_timeout"`
	ReconnectDelay int    `yaml:"reconnect_delay"`
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
	if c.WebSocket.URL == "" {
		c.WebSocket.URL = "ws://localhost:8000/ws/control"
	}
	if c.WebSocket.ConnectTimeout == 0 {
		c.WebSocket.ConnectTimeout = 30
	}
	if c.WebSocket.ReadTimeout == 0 {
		c.WebSocket.ReadTimeout = 30
	}
	if c.WebSocket.WriteTimeout == 0 {
		c.WebSocket.WriteTimeout = 10
	}
	if c.WebSocket.ReconnectDelay == 0 {
		c.WebSocket.ReconnectDelay = 10
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
	if env := os.Getenv("ROBOT_WEBSOCKET_URL"); env != "" {
		c.WebSocket.URL = env
	}
	if env := os.Getenv("ROBOT_WEBSOCKET_CONNECT_TIMEOUT"); env != "" {
		if val, err := parseInt(env); err == nil {
			c.WebSocket.ConnectTimeout = val
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

}

// validate 验证配置
func (c *Config) validate() error {
	// 验证机器人配置
	if c.Robot.UCode == "" {
		return fmt.Errorf("机器人UCode不能为空")
	}

	// 验证服务器配置
	if c.WebSocket.URL == "" {
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
	return time.Duration(c.WebSocket.ConnectTimeout) * time.Second
}

// GetReadTimeout 获取读取超时时间
func (c *Config) GetReadTimeout() time.Duration {
	return time.Duration(c.WebSocket.ReadTimeout) * time.Second
}

// GetWriteTimeout 获取写入超时时间
func (c *Config) GetWriteTimeout() time.Duration {
	return time.Duration(c.WebSocket.WriteTimeout) * time.Second
}

// GetHeartbeatInterval 获取心跳间隔
func (c *Config) GetHeartbeatInterval() time.Duration {
	return time.Duration(c.Heartbeat.Interval) * time.Second
}

// GetStatusInterval 获取状态上报间隔
func (c *Config) GetStatusInterval() time.Duration {
	return time.Duration(c.Status.Interval) * time.Second
}

// GetReconnectDelay 获取重连延迟
func (c *Config) GetReconnectDelay() time.Duration {
	return time.Duration(c.WebSocket.ReconnectDelay) * time.Second
}

// parseInt 解析整数
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
