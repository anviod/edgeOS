package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type EdgeXMQTTNATSConfig struct {
	Communication   CommunicationConfig   `yaml:"communication"`
	MQTT            MQTTConfig            `yaml:"mqtt"`
	NATS            NATSConfig            `yaml:"nats"`
	Subscriptions   SubscriptionsConfig   `yaml:"subscriptions"`
	MessageHandler  MessageHandlerConfig  `yaml:"message_handler"`
	NodeManagement  NodeManagementConfig  `yaml:"node_management"`
	ShadowDevice    ShadowDeviceConfig    `yaml:"shadow_device"`
	CommandPublisher CommandPublisherConfig `yaml:"command_publisher"`
}

type CommunicationConfig struct {
	Protocol string `yaml:"protocol"` // "mqtt" or "nats"
}

type MQTTConfig struct {
	Enabled              bool   `yaml:"enabled"`
	Broker               string `yaml:"broker"`
	ClientID             string `yaml:"client_id"`
	Username             string `yaml:"username"`
	Password             string `yaml:"password"`
	QoS                  byte   `yaml:"qos"`
	Retain               bool   `yaml:"retain"`
	CleanSession         bool   `yaml:"clean_session"`
	KeepAlive            int    `yaml:"keep_alive"`
	ConnectTimeout       int    `yaml:"connect_timeout"`
	AutoReconnect        bool   `yaml:"auto_reconnect"`
	MaxReconnectInterval int    `yaml:"max_reconnect_interval"`
}

type NATSConfig struct {
	Enabled             bool   `yaml:"enabled"`
	URL                 string `yaml:"url"`
	ClientName          string `yaml:"client_name"`
	Username            string `yaml:"username"`
	Password            string `yaml:"password"`
	Token               string `yaml:"token"`
	ConnectTimeout      int    `yaml:"connect_timeout"`
	ReconnectWait       int    `yaml:"reconnect_wait"`
	MaxReconnects       int    `yaml:"max_reconnects"`
	PingInterval        int    `yaml:"ping_interval"`
	MaxPingsOutstanding int    `yaml:"max_pings_outstanding"`
	JetStreamEnabled    bool   `yaml:"jetstream_enabled"`
}

type SubscriptionsConfig struct {
	Nodes   NodesSubscriptions   `yaml:"nodes"`
	Devices DevicesSubscriptions `yaml:"devices"`
	Points  PointsSubscriptions  `yaml:"points"`
	Data    DataSubscriptions    `yaml:"data"`
	Events  EventsSubscriptions  `yaml:"events"`
	Responses string             `yaml:"responses"`
}

type NodesSubscriptions struct {
	Register  string `yaml:"register"`
	Heartbeat string `yaml:"heartbeat"`
	Status    string `yaml:"status"`
	Unregister string `yaml:"unregister"`
}

type DevicesSubscriptions struct {
	Report string `yaml:"report"`
}

type PointsSubscriptions struct {
	Report string `yaml:"report"`
}

type DataSubscriptions struct {
	Stream string `yaml:"stream"`
	Batch  string `yaml:"batch"`
}

type EventsSubscriptions struct {
	Alert string `yaml:"alert"`
	Error string `yaml:"error"`
	Info  string `yaml:"info"`
}

type MessageHandlerConfig struct {
	MaxWorkers int `yaml:"max_workers"`
	QueueSize  int `yaml:"queue_size"`
	Timeout    int `yaml:"timeout"`
}

type NodeManagementConfig struct {
	HeartbeatTimeout int  `yaml:"heartbeat_timeout"`
	AutoUnregister   bool `yaml:"auto_unregister"`
	CleanupInterval  int  `yaml:"cleanup_interval"`
}

type ShadowDeviceConfig struct {
	Enabled      bool `yaml:"enabled"`
	SyncInterval int  `yaml:"sync_interval"`
	AutoCreate   bool `yaml:"auto_create"`
	AutoUpdate   bool `yaml:"auto_update"`
}

type CommandPublisherConfig struct {
	Timeout int `yaml:"timeout"`
	Retry   int `yaml:"retry"`
	QoS     int `yaml:"qos"`
}

func LoadEdgeXMQTTNATSConfig(path string) (*EdgeXMQTTNATSConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg EdgeXMQTTNATSConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if cfg.Communication.Protocol == "" {
		cfg.Communication.Protocol = "mqtt"
	}

	return &cfg, nil
}

func GetConnectTimeout(cfg *EdgeXMQTTNATSConfig) time.Duration {
	if cfg.Communication.Protocol == "mqtt" {
		return time.Duration(cfg.MQTT.ConnectTimeout) * time.Second
	}
	return time.Duration(cfg.NATS.ConnectTimeout) * time.Second
}
