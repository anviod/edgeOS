package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// MiddlewareMiddlewareConfig 单个中间件配置
type MiddlewareMiddlewareConfig struct {
	ID             string   `yaml:"id"`
	Name           string   `yaml:"name"`
	Type           string   `yaml:"type"` // "mqtt" or "nats"
	Enabled        bool     `yaml:"enabled"`
	Broker         string   `yaml:"broker"`
	ClientID       string   `yaml:"client_id"`
	Username       string   `yaml:"username"`
	Password       string   `yaml:"password"`
	QoS            byte     `yaml:"qos"`
	CleanSession   bool     `yaml:"clean_session"`
	KeepAlive      int      `yaml:"keep_alive"`
	ConnectTimeout int      `yaml:"connect_timeout"`
	AutoReconnect  bool     `yaml:"auto_reconnect"`
	Subscriptions  []string `yaml:"subscriptions"`
	// 高级设置
	MQTTVersion       int    `yaml:"mqtt_version"`       // 4 = 3.1.1, 5 = 5.0
	SSL               bool   `yaml:"ssl"`                // 启用 SSL/TLS
	CAFile            string `yaml:"ca_file"`            // CA 证书文件路径
	ClientCertFile    string `yaml:"client_cert_file"`   // 客户端证书文件路径
	ClientKeyFile     string `yaml:"client_key_file"`    // 客户端私钥文件路径
	ReconnectInterval int    `yaml:"reconnect_interval"` // 重连间隔（秒）
}

// MiddlewareConfigs []MiddlewareMiddlewareConfig

// Config 配置结构
type Config struct {
	//用户配置
	User struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Role     string `yaml:"role"`
	} `yaml:"user"`

	// 节点配置
	Node struct {
		NodeID        string `yaml:"node_id"`         // 节点ID
		NodeType      string `yaml:"node_type"`       // primary, secondary, collector
		PrimaryNodeID string `yaml:"primary_node_id"` // 备用节点需要配置主节点ID
		Listen        string `yaml:"listen"`          // 监听地址
	} `yaml:"node"`

	// 数据库配置
	Database struct {
		Type     string `yaml:"type"`     // bolt, etcd
		Path     string `yaml:"path"`     // 数据库路径
		Endpoint string `yaml:"endpoint"` // etcd 端点
	} `yaml:"database"`

	// 安全配置
	Security struct {
		JWTSecret  string `yaml:"jwt_secret"`  // JWT 密钥
		TLSEnabled bool   `yaml:"tls_enabled"` // 是否启用 TLS
		CertFile   string `yaml:"cert_file"`   // 证书文件
		KeyFile    string `yaml:"key_file"`    // 密钥文件
	}

	// 监控配置
	Monitoring struct {
		Enabled    bool   `yaml:"enabled"`    // 是否启用监控
		Prometheus string `yaml:"prometheus"` // Prometheus 端口
	}

	// 中间件配置
	Middlewares []MiddlewareMiddlewareConfig `yaml:"middlewares"`
}

// LoadConfig 加载配置
func LoadConfig(configPath string) (*Config, error) {
	// 确保配置文件存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// 解析配置
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	// 规范化路径
	config.Database.Path = normalizePath(config.Database.Path)
	config.Security.CertFile = normalizePath(config.Security.CertFile)
	config.Security.KeyFile = normalizePath(config.Security.KeyFile)

	// 规范化中间件配置
	for i := range config.Middlewares {
		m := &config.Middlewares[i]
		m.Subscriptions = mergeTopics(m.Subscriptions)

		// 设置默认值
		if m.Type == "" {
			m.Type = "mqtt"
		}
		if m.QoS == 0 {
			m.QoS = 1
		}
		if m.ConnectTimeout == 0 {
			m.ConnectTimeout = 10
		}
		if m.KeepAlive == 0 {
			m.KeepAlive = 30
		}
	}

	return &config, nil
}

func mergeTopics(topics []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, t := range topics {
		t = strings.TrimSpace(t)
		if t != "" && !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	return result
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	// 验证节点配置
	if config.Node.NodeID == "" {
		return fmt.Errorf("node_id is required")
	}

	if config.Node.NodeType == "" {
		return fmt.Errorf("node_type is required")
	}

	// 验证数据库配置
	if config.Database.Type == "" {
		config.Database.Type = "bolt"
	}

	if config.Database.Type == "bolt" && config.Database.Path == "" {
		config.Database.Path = "./data"
	}

	return nil
}

// normalizePath 规范化路径
func normalizePath(path string) string {
	if path == "" {
		return ""
	}

	// 如果是相对路径，转换为绝对路径
	if !filepath.IsAbs(path) {
		execDir, err := os.Getwd()
		if err == nil {
			path = filepath.Join(execDir, path)
		}
	}

	return path
}
