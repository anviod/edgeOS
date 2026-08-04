package config

import (
	"fmt"
	"sync"

	"github.com/anviod/edgeOS/internal/storage"
	"go.etcd.io/bbolt"
)

// MiddlewareMiddlewareConfig 单个中间件配置 | Single middleware configuration
type MiddlewareMiddlewareConfig struct {
	ID                string   `yaml:"id" json:"id"`
	Name              string   `yaml:"name" json:"name"`
	Type              string   `yaml:"type" json:"type"` // "mqtt" or "nats"
	Enabled           bool     `yaml:"enabled" json:"enabled"`
	Broker            string   `yaml:"broker" json:"broker"`
	ClientID          string   `yaml:"client_id" json:"client_id"`
	Username          string   `yaml:"username" json:"username"`
	Password          string   `yaml:"password" json:"password"`
	QoS               byte     `yaml:"qos" json:"qos"`
	CleanSession      bool     `yaml:"clean_session" json:"clean_session"`
	KeepAlive         int      `yaml:"keep_alive" json:"keep_alive"`
	ConnectTimeout    int      `yaml:"connect_timeout" json:"connect_timeout"`
	AutoReconnect     bool     `yaml:"auto_reconnect" json:"auto_reconnect"`
	Subscriptions     []string `yaml:"subscriptions" json:"subscriptions"`
	MQTTVersion       int      `yaml:"mqtt_version" json:"mqtt_version"`           // 4 = 3.1.1, 5 = 5.0
	SSL               bool     `yaml:"ssl" json:"ssl"`                           // 启用 SSL/TLS | Enable SSL/TLS
	CAFile            string   `yaml:"ca_file" json:"ca_file"`                   // CA 证书文件路径 | CA certificate file path
	ClientCertFile    string   `yaml:"client_cert_file" json:"client_cert_file"` // 客户端证书文件路径 | Client cert file path
	ClientKeyFile     string   `yaml:"client_key_file" json:"client_key_file"`   // 客户端私钥文件路径 | Client key file path
	ReconnectInterval int      `yaml:"reconnect_interval" json:"reconnect_interval"` // 重连间隔（秒）| Reconnect interval (seconds)
}

// Config 配置结构 | Configuration structure
type Config struct {
	// 用户配置 | User configuration
	User struct {
		Username string `yaml:"username" json:"username"`
		Password string `yaml:"password" json:"password"`
		Role     string `yaml:"role" json:"role"`
	} `yaml:"user" json:"user"`

	// 节点配置 | Node configuration
	Node struct {
		NodeID        string `yaml:"node_id" json:"node_id"`               // 节点ID | Node ID
		NodeType      string `yaml:"node_type" json:"node_type"`           // primary, secondary, collector
		PrimaryNodeID string `yaml:"primary_node_id" json:"primary_node_id"` // 备用节点需要配置主节点ID | Backup node needs primary node ID
		Listen        string `yaml:"listen" json:"listen"`                 // 监听地址 | Listen address
	} `yaml:"node" json:"node"`

	// 数据库配置 | Database configuration
	Database struct {
		Type     string `yaml:"type" json:"type"`         // bolt, etcd
		Path     string `yaml:"path" json:"path"`         // 数据库路径 | Database path
		Endpoint string `yaml:"endpoint" json:"endpoint"` // etcd 端点 | etcd endpoint
	} `yaml:"database" json:"database"`

	// 安全配置 | Security configuration
	Security struct {
		JWTSecret  string `yaml:"jwt_secret" json:"jwt_secret"`   // JWT 密钥 | JWT secret
		TLSEnabled bool   `yaml:"tls_enabled" json:"tls_enabled"` // 是否启用 TLS | Enable TLS
		CertFile   string `yaml:"cert_file" json:"cert_file"`     // 证书文件 | Certificate file
		KeyFile    string `yaml:"key_file" json:"key_file"`       // 密钥文件 | Key file
	} `yaml:"security" json:"security"`

	// 监控配置 | Monitoring configuration
	Monitoring struct {
		Enabled    bool   `yaml:"enabled" json:"enabled"`           // 是否启用监控 | Enable monitoring
		Prometheus string `yaml:"prometheus" json:"prometheus"`     // Prometheus 端口 | Prometheus port
	} `yaml:"monitoring" json:"monitoring"`

	// 中间件配置 | Middleware configuration
	Middlewares []MiddlewareMiddlewareConfig `yaml:"middlewares" json:"middlewares"`

	// EAN 2.0 配置 | EAN 2.0 configuration
	EAN EANConfig `yaml:"ean" json:"ean"`
}

// ── ConfigManager 配置管理器 | Config Manager ───────────────────────────────

var saveMu sync.Mutex

// ConfigManager 管理配置的加载与持久化（运行时以数据库为唯一数据源）
// ConfigManager manages config loading and persistence (DB is the single source of truth at runtime)
type ConfigManager struct {
	Config  *Config
	dataDir string
	db      *bbolt.DB
	useDB   bool
}

// NewConfigManagerWithEmptyConfig 创建使用默认空配置的配置管理器（用于安装前）
// NewConfigManagerWithEmptyConfig creates a ConfigManager with default empty config (pre-install)
func NewConfigManagerWithEmptyConfig(dataDir string) *ConfigManager {
	cfg := DefaultConfig()
	return &ConfigManager{
		Config:  cfg,
		dataDir: dataDir,
		db:      nil,
		useDB:   false,
	}
}

// NewConfigManagerWithDB 创建使用数据库存储配置的配置管理器
// NewConfigManagerWithDB creates a ConfigManager backed by the config database
func NewConfigManagerWithDB(dataDir string, db *bbolt.DB) (*ConfigManager, error) {
	configStore, err := storage.NewConfigStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create config store: %w", err)
	}

	hasData, err := configStore.HasConfigData()
	if err != nil {
		return nil, fmt.Errorf("failed to check config data: %w", err)
	}

	var cfg *Config
	if hasData {
		cfg, err = LoadConfigFromDB(db)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from DB: %w", err)
		}
	} else {
		cfg = DefaultConfig()
	}

	return &ConfigManager{
		Config:  cfg,
		dataDir: dataDir,
		db:      db,
		useDB:   true,
	}, nil
}

// GetConfig 获取当前配置 | Get current configuration
func (cm *ConfigManager) GetConfig() *Config {
	return cm.Config
}

// AttachDB 将运行时数据库绑定到配置管理器（安装完成后使用）
// AttachDB binds the runtime database to the config manager (after install)
func (cm *ConfigManager) AttachDB(db *bbolt.DB) {
	cm.db = db
	cm.useDB = true
}

// Reload 从数据库重新加载配置 | Reload configuration from database
func (cm *ConfigManager) Reload() error {
	if !cm.useDB || cm.db == nil {
		return fmt.Errorf("configuration reload requires database attachment")
	}
	newCfg, err := LoadConfigFromDB(cm.db)
	if err != nil {
		return err
	}
	cm.Config = newCfg
	return nil
}

// SaveConfig 保存配置到数据库 | Save configuration to database
func (cm *ConfigManager) SaveConfig(cfg *Config) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	if cm.useDB && cm.db != nil {
		return SaveConfigToDB(cm.db, cfg)
	}

	return fmt.Errorf("configuration persistence requires database attachment")
}

// ── DefaultConfig 默认配置 | Default configuration ──────────────────────────

// DefaultConfig 返回默认配置（安装模式使用）
// DefaultConfig returns default configuration (used in install mode)
func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.User.Username = "admin"
	cfg.User.Password = "passwd@123"
	cfg.User.Role = "admin"

	cfg.Node.NodeID = "node-001"
	cfg.Node.NodeType = "primary"
	cfg.Node.PrimaryNodeID = "node-primary"
	cfg.Node.Listen = ":8000"

	cfg.Database.Type = "bolt"
	cfg.Database.Path = "./data"

	cfg.Security.JWTSecret = "edgeos-secret-key"
	cfg.Security.TLSEnabled = false

	cfg.Monitoring.Enabled = true
	cfg.Monitoring.Prometheus = ":9090"

	cfg.Middlewares = []MiddlewareMiddlewareConfig{}
	cfg.EAN = DefaultEANConfig()

	return cfg
}

// ── DB 加载/保存 | DB load/save ─────────────────────────────────────────────

// LoadConfigFromDB 从数据库加载完整配置
// LoadConfigFromDB loads complete configuration from the database
func LoadConfigFromDB(db *bbolt.DB) (*Config, error) {
	configStore, err := storage.NewConfigStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create config store: %w", err)
	}

	cfg := DefaultConfig()

	// 加载节点配置 | Load node config
	nodeData, err := configStore.LoadNodeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load node config: %w", err)
	}
	if nodeData != nil {
		cfg.Node.NodeID = nodeData.NodeID
		cfg.Node.NodeType = nodeData.NodeType
		cfg.Node.PrimaryNodeID = nodeData.PrimaryNodeID
		cfg.Node.Listen = nodeData.Listen
	}

	// 加载安全配置 | Load security config
	secData, err := configStore.LoadSecurity()
	if err != nil {
		return nil, fmt.Errorf("failed to load security config: %w", err)
	}
	if secData != nil {
		cfg.Security.JWTSecret = secData.JWTSecret
		cfg.Security.TLSEnabled = secData.TLSEnabled
		cfg.Security.CertFile = secData.CertFile
		cfg.Security.KeyFile = secData.KeyFile
	}

	// 加载监控配置 | Load monitoring config
	monData, err := configStore.LoadMonitoring()
	if err != nil {
		return nil, fmt.Errorf("failed to load monitoring config: %w", err)
	}
	if monData != nil {
		cfg.Monitoring.Enabled = monData.Enabled
		cfg.Monitoring.Prometheus = monData.Prometheus
	}

	// 加载中间件配置 | Load middlewares
	mwData, err := configStore.LoadMiddlewares()
	if err != nil {
		return nil, fmt.Errorf("failed to load middlewares: %w", err)
	}
	cfg.Middlewares = convertMiddlewaresFromDB(mwData)

	// 加载 EAN 配置 | Load EAN config
	eanData, err := configStore.LoadEAN()
	if err != nil {
		return nil, fmt.Errorf("failed to load EAN config: %w", err)
	}
	if eanData != nil {
		cfg.EAN = convertEANFromDB(*eanData)
	}

	// 加载用户配置 | Load users
	userData, err := configStore.LoadUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to load users: %w", err)
	}
	if len(userData) > 0 {
		u := userData[0]
		cfg.User.Username = u.Username
		cfg.User.Password = u.Password
		cfg.User.Role = u.Role
	}

	// 数据库路径始终使用默认值（bootstrap 配置）| DB path always uses default (bootstrap config)
	cfg.Database.Type = "bolt"
	cfg.Database.Path = "./data"

	return cfg, nil
}

// SaveConfigToDB 保存完整配置到数据库
// SaveConfigToDB saves complete configuration to the database
func SaveConfigToDB(db *bbolt.DB, cfg *Config) error {
	configStore, err := storage.NewConfigStore(db)
	if err != nil {
		return fmt.Errorf("failed to create config store: %w", err)
	}

	nodeData := storage.NodeConfigData{
		NodeID:        cfg.Node.NodeID,
		NodeType:      cfg.Node.NodeType,
		PrimaryNodeID: cfg.Node.PrimaryNodeID,
		Listen:        cfg.Node.Listen,
	}

	secData := storage.SecurityConfigData{
		JWTSecret:  cfg.Security.JWTSecret,
		TLSEnabled: cfg.Security.TLSEnabled,
		CertFile:   cfg.Security.CertFile,
		KeyFile:    cfg.Security.KeyFile,
	}

	monData := storage.MonitoringConfigData{
		Enabled:    cfg.Monitoring.Enabled,
		Prometheus: cfg.Monitoring.Prometheus,
	}

	mwData := convertMiddlewaresToDB(cfg.Middlewares)

	eanData := convertEANToDB(cfg.EAN)

	users := []storage.UserConfigData{
		{
			Username: cfg.User.Username,
			Password: cfg.User.Password,
			Role:     cfg.User.Role,
		},
	}

	return configStore.SaveAllConfig(nodeData, secData, monData, mwData, eanData, users)
}

// ── 默认安装初始化 | Default install initialization ─────────────────────────

// InitializeDefaultConfig 首次安装时将默认配置写入 config.db（默认安装初始化）。
// 运行时配置全部保存在 bboltDB，不依赖任何 YAML 配置文件；重启从数据库恢复。
// InitializeDefaultConfig persists default configuration to config.db on first install.
// All runtime configuration lives in bboltDB with no YAML file dependency; restarts load from DB.
func InitializeDefaultConfig(db *bbolt.DB) error {
	if err := SaveConfigToDB(db, DefaultConfig()); err != nil {
		return fmt.Errorf("failed to initialize default config: %w", err)
	}
	return nil
}

// ── 转换函数 | Conversion functions ──────────────────────────────────────────

func convertMiddlewaresToDB(mws []MiddlewareMiddlewareConfig) []storage.MiddlewareConfigData {
	result := make([]storage.MiddlewareConfigData, len(mws))
	for i, m := range mws {
		result[i] = storage.MiddlewareConfigData{
			ID:                m.ID,
			Name:              m.Name,
			Type:              m.Type,
			Enabled:           m.Enabled,
			Broker:            m.Broker,
			ClientID:          m.ClientID,
			Username:          m.Username,
			Password:          m.Password,
			QoS:               m.QoS,
			CleanSession:      m.CleanSession,
			KeepAlive:         m.KeepAlive,
			ConnectTimeout:    m.ConnectTimeout,
			AutoReconnect:     m.AutoReconnect,
			Subscriptions:     m.Subscriptions,
			MQTTVersion:       m.MQTTVersion,
			SSL:               m.SSL,
			CAFile:            m.CAFile,
			ClientCertFile:    m.ClientCertFile,
			ClientKeyFile:     m.ClientKeyFile,
			ReconnectInterval: m.ReconnectInterval,
		}
	}
	return result
}

func convertMiddlewaresFromDB(mws []storage.MiddlewareConfigData) []MiddlewareMiddlewareConfig {
	result := make([]MiddlewareMiddlewareConfig, len(mws))
	for i, m := range mws {
		result[i] = MiddlewareMiddlewareConfig{
			ID:                m.ID,
			Name:              m.Name,
			Type:              m.Type,
			Enabled:           m.Enabled,
			Broker:            m.Broker,
			ClientID:          m.ClientID,
			Username:          m.Username,
			Password:          m.Password,
			QoS:               m.QoS,
			CleanSession:      m.CleanSession,
			KeepAlive:         m.KeepAlive,
			ConnectTimeout:    m.ConnectTimeout,
			AutoReconnect:     m.AutoReconnect,
			Subscriptions:     m.Subscriptions,
			MQTTVersion:       m.MQTTVersion,
			SSL:               m.SSL,
			CAFile:            m.CAFile,
			ClientCertFile:    m.ClientCertFile,
			ClientKeyFile:     m.ClientKeyFile,
			ReconnectInterval: m.ReconnectInterval,
		}
	}
	return result
}

func convertEANToDB(ean EANConfig) storage.EANConfigData {
	return storage.EANConfigData{
		Enabled:   ean.Enabled,
		PlannerID: ean.PlannerID,
		MQTT: storage.EANMQTTConfigData{
			Enabled:        ean.MQTT.Enabled,
			Broker:         ean.MQTT.Broker,
			ClientID:       ean.MQTT.ClientID,
			Username:       ean.MQTT.Username,
			Password:       ean.MQTT.Password,
			QoS:            ean.MQTT.QoS,
			KeepAlive:      ean.MQTT.KeepAlive,
			ConnectTimeout: ean.MQTT.ConnectTimeout,
			MQTTVersion:    ean.MQTT.MQTTVersion,
		},
		NATS: storage.EANNATSConfigData{
			Enabled:        ean.NATS.Enabled,
			URL:            ean.NATS.URL,
			ClientName:     ean.NATS.ClientName,
			Username:       ean.NATS.Username,
			Password:       ean.NATS.Password,
			Token:          ean.NATS.Token,
			ConnectTimeout: ean.NATS.ConnectTimeout,
			ReconnectWait:  ean.NATS.ReconnectWait,
			MaxReconnects:  ean.NATS.MaxReconnects,
			PingInterval:   ean.NATS.PingInterval,
		},
		Heartbeat: storage.EANHeartbeatData{
			TimeoutMultiplier:      ean.Heartbeat.TimeoutMultiplier,
			CheckIntervalSec:       ean.Heartbeat.CheckIntervalSec,
			MaxOfflineRetentionSec: ean.Heartbeat.MaxOfflineRetentionSec,
		},
		V1CommandEnabled: boolPtr(ean.V1CommandEnabled),
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func convertEANFromDB(data storage.EANConfigData) EANConfig {
	cfg := EANConfig{
		Enabled:   data.Enabled,
		PlannerID: data.PlannerID,
		MQTT: EANMQTTConfig{
			Enabled:        data.MQTT.Enabled,
			Broker:         data.MQTT.Broker,
			ClientID:       data.MQTT.ClientID,
			Username:       data.MQTT.Username,
			Password:       data.MQTT.Password,
			QoS:            data.MQTT.QoS,
			KeepAlive:      data.MQTT.KeepAlive,
			ConnectTimeout: data.MQTT.ConnectTimeout,
			MQTTVersion:    data.MQTT.MQTTVersion,
		},
		NATS: EANNATSConfig{
			Enabled:        data.NATS.Enabled,
			URL:            data.NATS.URL,
			ClientName:     data.NATS.ClientName,
			Username:       data.NATS.Username,
			Password:       data.NATS.Password,
			Token:          data.NATS.Token,
			ConnectTimeout: data.NATS.ConnectTimeout,
			ReconnectWait:  data.NATS.ReconnectWait,
			MaxReconnects:  data.NATS.MaxReconnects,
			PingInterval:   data.NATS.PingInterval,
		},
		Heartbeat: EANHeartbeatConfig{
			TimeoutMultiplier:      data.Heartbeat.TimeoutMultiplier,
			CheckIntervalSec:       data.Heartbeat.CheckIntervalSec,
			MaxOfflineRetentionSec: data.Heartbeat.MaxOfflineRetentionSec,
		},
		// Phase 4: V1 命令面全面下线——nil=未持久化（老版本 config.db）→ 回退默认 false（已下线）
		V1CommandEnabled: data.V1CommandEnabled != nil && *data.V1CommandEnabled,
	}
	// 兼容老版本 config.db：新字段未持久化时回退默认值
	cfg.Heartbeat = applyEANHeartbeatDefaults(cfg.Heartbeat)
	return cfg
}

// applyEANHeartbeatDefaults 对 EAN 心跳配置做默认值回退（兼容老版本 config.db 缺少新字段）
func applyEANHeartbeatDefaults(hb EANHeartbeatConfig) EANHeartbeatConfig {
	if hb.TimeoutMultiplier <= 0 {
		hb.TimeoutMultiplier = 3
	}
	if hb.CheckIntervalSec <= 0 {
		hb.CheckIntervalSec = 5
	}
	// MaxOfflineRetentionSec: 0 且老库未持久化 → 回退默认 600s（10 分钟）。
	// 注意：显式配置 0 表示禁用自动清除，此处在 DB 加载路径不区分「未设置」与「显式 0」，
	// 统一回退默认 600s 以保证老库升级后自动清除生效。显式禁用的配置从 YAML/设置接口写入。
	if hb.MaxOfflineRetentionSec <= 0 {
		hb.MaxOfflineRetentionSec = DefaultEANConfig().Heartbeat.MaxOfflineRetentionSec
	}
	return hb
}
