package model

import (
	"fmt"
	"strings"
	"time"
)

// MiddlewareConfig 消息总线配置
type MiddlewareConfig struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"` // mqtt, nats
	Host            string   `json:"host"` // 解析后的主机名
	Port            int      `json:"port"` // 解析后的端口
	Broker          string   `json:"broker"` // 完整连接 URL (tcp://127.0.0.1:1883)
	Username        string   `json:"username"`
	Password        string   `json:"password"`
	ClientID        string   `json:"client_id"`
	Topics          []string `json:"topics"`
	Subscriptions   []string `json:"subscriptions"` // 订阅主题列表
	Enabled         bool     `json:"enabled"`
	Status          string   `json:"status"` // connected, disconnected, error, connecting
	LastError       string   `json:"last_error,omitempty"`
	QoS             byte     `json:"qos"`
	CleanSession    bool     `json:"clean_session"`
	KeepAlive       int      `json:"keep_alive"`
	ConnectTimeout  int      `json:"connect_timeout"`
	AutoReconnect   bool     `json:"auto_reconnect"`
	CreatedAt       int64    `json:"created_at"`
	UpdatedAt       int64    `json:"updated_at"`
	// 高级设置
	MQTTVersion       int    `json:"mqtt_version"`       // 4 = 3.1.1, 5 = 5.0
	SSL                bool   `json:"ssl"`               // 启用 SSL/TLS
	CAFile             string `json:"ca_file"`           // CA 证书文件路径
	ClientCertFile     string `json:"client_cert_file"`  // 客户端证书文件路径
	ClientKeyFile      string `json:"client_key_file"`  // 客户端私钥文件路径
	ReconnectInterval  int    `json:"reconnect_interval"` // 重连间隔（秒）
}

// SetBrokerURL 设置 Broker URL 并解析 host/port
func (m *MiddlewareConfig) SetBrokerURL(broker string) {
	m.Broker = broker
	// 解析 tcp://127.0.0.1:1883 或 mqtt://127.0.0.1:1883
	broker = strings.TrimPrefix(broker, "tcp://")
	broker = strings.TrimPrefix(broker, "mqtt://")
	broker = strings.TrimPrefix(broker, "mqtts://")
	broker = strings.TrimPrefix(broker, "ssl://")

	if idx := strings.LastIndex(broker, ":"); idx != -1 {
		m.Host = broker[:idx]
		fmt.Sscanf(broker[idx+1:], "%d", &m.Port)
	} else {
		m.Host = broker
	}
}

// EnsureBrokerURL 确保 Broker URL 正确构建
// 如果 Broker 为空但 Host/Port 有值，则构建 Broker URL
func (m *MiddlewareConfig) EnsureBrokerURL() {
	if m.Broker == "" && m.Host != "" && m.Port > 0 {
		m.Broker = fmt.Sprintf("tcp://%s:%d", m.Host, m.Port)
	} else if m.Broker != "" {
		// 解析已存在的 broker URL 以确保 Host/Port 正确
		m.SetBrokerURL(m.Broker)
	}
}

// AlertInfo 告警信息
type AlertInfo struct {
	ID             string      `json:"id"`
	NodeID         string      `json:"node_id"`
	DeviceID       string      `json:"device_id"`
	PointID        string      `json:"point_id"`
	Level          string      `json:"level"`    // critical, major, minor, info
	Category       string      `json:"category"`
	Message        string      `json:"message"`
	Value          interface{} `json:"value,omitempty"`
	Threshold      interface{} `json:"threshold,omitempty"`
	Status         string      `json:"status"` // active, acknowledged, resolved
	CreatedAt      int64       `json:"created_at"`
	AcknowledgedAt int64       `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string      `json:"acknowledged_by,omitempty"`
}

// CommandRecord 命令执行记录
type CommandRecord struct {
	ID        string      `json:"id"`
	NodeID    string      `json:"node_id"`
	DeviceID  string      `json:"device_id"`
	PointID   string      `json:"point_id"`
	Value     interface{} `json:"value"`
	Status    string      `json:"status"` // pending, success, error, timeout
	Error     string      `json:"error,omitempty"`
	CreatedAt int64       `json:"created_at"`
	UpdatedAt int64       `json:"updated_at"`
}

// NewMiddlewareConfig 创建新的中间件配置（带默认值）
func NewMiddlewareConfig() *MiddlewareConfig {
	now := time.Now().Unix()
	return &MiddlewareConfig{
		Status:            "disconnected",
		QoS:               1,
		CleanSession:      true,
		KeepAlive:         30,
		ConnectTimeout:    10,
		AutoReconnect:    true,
		Subscriptions:     []string{},
		Topics:            []string{},
		CreatedAt:         now,
		UpdatedAt:         now,
		MQTTVersion:       4, // MQTT 3.1.1
		SSL:               false,
		ReconnectInterval: 5,
	}
}
