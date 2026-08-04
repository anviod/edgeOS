package config

// EANConfig EAN 2.0 总配置 | EAN 2.0 configuration
type EANConfig struct {
	// Enabled 是否启用 EAN Bus | Enable EAN Bus
	Enabled bool `yaml:"enabled" json:"enabled"`
	// PlannerID EdgeOS Planner 标识，用于 Invoke reply topic 路由 | EdgeOS Planner ID for Invoke reply routing
	PlannerID string `yaml:"planner_id" json:"planner_id"`
	// MQTT MQTT 传输层配置 | MQTT transport config
	MQTT EANMQTTConfig `yaml:"mqtt" json:"mqtt"`
	// NATS NATS 传输层配置 | NATS transport config
	NATS EANNATSConfig `yaml:"nats" json:"nats"`
	// Heartbeat 心跳监控配置 | Heartbeat monitoring config
	Heartbeat EANHeartbeatConfig `yaml:"heartbeat" json:"heartbeat"`
	// V1CommandEnabled V1 命令面开关（Phase 4 已全面下线，默认 false；命令统一 EAN Invoke，
	// V1 数据面/告警不受影响）
	// V1 command plane switch (Phase 4: fully retired, default false; commands use EAN Invoke;
	// V1 data plane & alerts unaffected)
	V1CommandEnabled bool `yaml:"v1_command_enabled" json:"v1_command_enabled"`
}

// EANMQTTConfig MQTT 传输层配置 | MQTT transport configuration
type EANMQTTConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	Broker         string `yaml:"broker" json:"broker"`               // 例: tcp://127.0.0.1:1883
	ClientID       string `yaml:"client_id" json:"client_id"`
	Username       string `yaml:"username" json:"username"`
	Password       string `yaml:"password" json:"password"`
	QoS            int    `yaml:"qos" json:"qos"`                     // 默认 1 | default 1
	KeepAlive      int    `yaml:"keep_alive" json:"keep_alive"`       // 默认 30 | default 30
	ConnectTimeout int    `yaml:"connect_timeout" json:"connect_timeout"` // 默认 10（秒）| default 10 (seconds)
	MQTTVersion    int    `yaml:"mqtt_version" json:"mqtt_version"`   // 4 = 3.1.1, 5 = 5.0
}

// EANNATSConfig NATS 传输层配置 | NATS transport configuration
type EANNATSConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	URL            string `yaml:"url" json:"url"`                     // 例: nats://127.0.0.1:4222
	ClientName     string `yaml:"client_name" json:"client_name"`
	Username       string `yaml:"username" json:"username"`
	Password       string `yaml:"password" json:"password"`
	Token          string `yaml:"token" json:"token"`
	ConnectTimeout int    `yaml:"connect_timeout" json:"connect_timeout"` // 默认 10（秒）| default 10 (seconds)
	ReconnectWait  int    `yaml:"reconnect_wait" json:"reconnect_wait"`   // 默认 2（秒）| default 2 (seconds)
	MaxReconnects  int    `yaml:"max_reconnects" json:"max_reconnects"`   // 默认 5 | default 5
	PingInterval   int    `yaml:"ping_interval" json:"ping_interval"`     // 默认 0（使用 NATS 默认）| default 0 (use NATS default)
}

// EANHeartbeatConfig 心跳监控配置 | Heartbeat monitoring configuration
type EANHeartbeatConfig struct {
	// TimeoutMultiplier 超时倍数，默认 3（3 个心跳周期未收到则超时）| Timeout multiplier, default 3
	TimeoutMultiplier int `yaml:"timeout_multiplier" json:"timeout_multiplier"`
	// CheckIntervalSec 检查循环间隔（秒），默认 5 | Check interval (seconds), default 5
	CheckIntervalSec int `yaml:"check_interval_sec" json:"check_interval_sec"`
	// MaxOfflineRetentionSec 离线保留期（秒），默认 600（10 分钟）。
	// Agent 心跳超时标记 offline 后，超过该时长仍未重新上线（如 EdgeX 关闭 EAN）
	// 则彻底删除（DeleteAgent），避免 Agent 管理页残留离线 Agent。0 表示不自动清除。
	// | Offline retention (seconds); expired offline agents are purged automatically.
	MaxOfflineRetentionSec int `yaml:"max_offline_retention_sec" json:"max_offline_retention_sec"`
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
			Enabled:        false, // 联调对称时在系统配置中设 ean.nats.enabled=true
			URL:            "nats://127.0.0.1:4222",
			ClientName:     "edgeos-ean",
			ConnectTimeout:  10,
			ReconnectWait:   2,
			MaxReconnects:   5,
		},
		Heartbeat: EANHeartbeatConfig{
			TimeoutMultiplier:      3,
			CheckIntervalSec:       5,
			MaxOfflineRetentionSec: 600, // 10 分钟离线保留期，超时自动清除残留离线 Agent
		},
		// Phase 4: V1 命令面全面下线——默认 false；命令统一走 EAN Invoke。
		V1CommandEnabled: false,
	}
}
