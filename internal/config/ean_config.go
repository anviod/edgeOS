package config

// EANConfig EAN 2.0 总配置
type EANConfig struct {
	// Enabled 是否启用 EAN Bus
	Enabled bool `yaml:"enabled"`
	// PlannerID EdgeOS Planner 标识，用于 Invoke reply topic 路由
	PlannerID string `yaml:"planner_id"`
	// MQTT MQTT 传输层配置
	MQTT EANMQTTConfig `yaml:"mqtt"`
	// NATS NATS 传输层配置
	NATS EANNATSConfig `yaml:"nats"`
	// Heartbeat 心跳监控配置
	Heartbeat EANHeartbeatConfig `yaml:"heartbeat"`
}

// EANMQTTConfig MQTT 传输层配置
type EANMQTTConfig struct {
	Enabled         bool   `yaml:"enabled"`
	Broker          string `yaml:"broker"`            // 例: tcp://127.0.0.1:1883
	ClientID        string `yaml:"client_id"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	QoS             int    `yaml:"qos"`                // 默认 1
	KeepAlive       int    `yaml:"keep_alive"`        // 默认 30
	ConnectTimeout  int    `yaml:"connect_timeout"`   // 默认 10（秒）
	MQTTVersion     int    `yaml:"mqtt_version"`       // 4 = 3.1.1, 5 = 5.0
}

// EANNATSConfig NATS 传输层配置
type EANNATSConfig struct {
	Enabled         bool   `yaml:"enabled"`
	URL             string `yaml:"url"`                // 例: nats://127.0.0.1:4222
	ClientName      string `yaml:"client_name"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	Token           string `yaml:"token"`
	ConnectTimeout  int    `yaml:"connect_timeout"`   // 默认 10（秒）
	ReconnectWait   int    `yaml:"reconnect_wait"`    // 默认 2（秒）
	MaxReconnects   int    `yaml:"max_reconnects"`    // 默认 5
	PingInterval    int    `yaml:"ping_interval"`     // 默认 0（使用 NATS 默认）
}

// EANHeartbeatConfig 心跳监控配置
type EANHeartbeatConfig struct {
	// TimeoutMultiplier 超时倍数，默认 3（3 个心跳周期未收到则超时）
	TimeoutMultiplier int `yaml:"timeout_multiplier"`
	// CheckIntervalSec 检查循环间隔（秒），默认 5
	CheckIntervalSec int `yaml:"check_interval_sec"`
}

// DefaultEANConfig 返回默认 EAN 配置（零值填充）
func DefaultEANConfig() EANConfig {
	return EANConfig{
		Enabled:   false,
		PlannerID: "edgeos-planner",
		MQTT: EANMQTTConfig{
			Enabled:        true,
			Broker:         "tcp://127.0.0.1:18083", // 与 EdgeX 北向联调默认端口一致
			ClientID:       "edgeos-ean",
			QoS:            1,
			KeepAlive:      30,
			ConnectTimeout: 10,
			MQTTVersion:    4,
		},
		NATS: EANNATSConfig{
			Enabled:        false, // 联调对称时在 config.yaml 设 ean.nats.enabled=true
			URL:            "nats://127.0.0.1:4222",
			ClientName:     "edgeos-ean",
			ConnectTimeout:  10,
			ReconnectWait:   2,
			MaxReconnects:   5,
		},
		Heartbeat: EANHeartbeatConfig{
			TimeoutMultiplier: 3,
			CheckIntervalSec:  5,
		},
	}
}
