package storage

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.etcd.io/bbolt"
)

// ── Bucket 常量 | Bucket constants ──────────────────────────────────────────
const (
	BucketConfigVersion = "ConfigVersion"
	BucketNode          = "Node"
	BucketSecurity      = "Security"
	BucketMonitoring    = "Monitoring"
	BucketMiddlewares   = "Middlewares"
	BucketEAN           = "EAN"
	BucketUsers         = "Users"

	ConfigVersionKey   = "version"
	ConfigVersionValue = "1.0"
)

// configBucketNames 配置库 bucket 列表 | Config DB bucket list
var configBucketNames = []string{
	BucketConfigVersion,
	BucketNode,
	BucketSecurity,
	BucketMonitoring,
	BucketMiddlewares,
	BucketEAN,
	BucketUsers,
}

// runtimeBucketNames 运行时库 bucket 列表 | Runtime DB bucket list
var runtimeBucketNames = []string{
	"devices", "tasks", "state", "stats",
	"edgex_nodes", "edgex_devices", "edgex_points",
	"edgex_data", "edgex_alerts", "middlewares", "edgex_commands",
}

// configBucketMap 用于判断 bucket 是否属于配置库 | Map for config bucket classification
var configBucketMap = map[string]bool{
	"ConfigVersion": true,
	"Node":          true,
	"Security":      true,
	"Monitoring":    true,
	"Middlewares":   true,
	"EAN":           true,
	"Users":         true,
}

// IsConfigBucket 判断 bucket 是否属于配置库 | Check if bucket belongs to config DB
func IsConfigBucket(name string) bool {
	return configBucketMap[name]
}

// ConfigStore 配置存储器，封装所有配置的读写操作
// ConfigStore wraps all configuration read/write operations on config.db
type ConfigStore struct {
	db *bbolt.DB
}

// NewConfigStore 创建配置存储器并初始化所有 bucket
// NewConfigStore creates a ConfigStore and initializes all buckets
func NewConfigStore(db *bbolt.DB) (*ConfigStore, error) {
	err := db.Update(func(tx *bbolt.Tx) error {
		for _, bucket := range configBucketNames {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}

		b := tx.Bucket([]byte(BucketConfigVersion))
		if b != nil {
			currentVersion := b.Get([]byte(ConfigVersionKey))
			if currentVersion == nil {
				return b.Put([]byte(ConfigVersionKey), []byte(ConfigVersionValue))
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &ConfigStore{db: db}, nil
}

// ── Node 配置 | Node config ─────────────────────────────────────────────────

// SaveNodeConfig 保存节点配置 | Save node configuration
func (cs *ConfigStore) SaveNodeConfig(node NodeConfigData) error {
	return cs.saveJSON(BucketNode, "node", node)
}

// LoadNodeConfig 加载节点配置 | Load node configuration
func (cs *ConfigStore) LoadNodeConfig() (*NodeConfigData, error) {
	var config NodeConfigData
	err := cs.loadJSON(BucketNode, "node", &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ── Security 配置 | Security config ─────────────────────────────────────────

// SaveSecurity 保存安全配置 | Save security configuration
func (cs *ConfigStore) SaveSecurity(config SecurityConfigData) error {
	return cs.saveJSON(BucketSecurity, "security", config)
}

// LoadSecurity 加载安全配置 | Load security configuration
func (cs *ConfigStore) LoadSecurity() (*SecurityConfigData, error) {
	var config SecurityConfigData
	err := cs.loadJSON(BucketSecurity, "security", &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ── Monitoring 配置 | Monitoring config ─────────────────────────────────────

// SaveMonitoring 保存监控配置 | Save monitoring configuration
func (cs *ConfigStore) SaveMonitoring(config MonitoringConfigData) error {
	return cs.saveJSON(BucketMonitoring, "monitoring", config)
}

// LoadMonitoring 加载监控配置 | Load monitoring configuration
func (cs *ConfigStore) LoadMonitoring() (*MonitoringConfigData, error) {
	var config MonitoringConfigData
	err := cs.loadJSON(BucketMonitoring, "monitoring", &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ── Middlewares 配置 | Middlewares config ───────────────────────────────────

// SaveMiddlewares 保存中间件列表 | Save middleware list
func (cs *ConfigStore) SaveMiddlewares(middlewares []MiddlewareConfigData) error {
	return cs.saveJSON(BucketMiddlewares, "middlewares", middlewares)
}

// LoadMiddlewares 加载中间件列表 | Load middleware list
func (cs *ConfigStore) LoadMiddlewares() ([]MiddlewareConfigData, error) {
	var middlewares []MiddlewareConfigData
	err := cs.loadJSON(BucketMiddlewares, "middlewares", &middlewares)
	if err != nil {
		return nil, err
	}
	if middlewares == nil {
		return []MiddlewareConfigData{}, nil
	}
	return middlewares, nil
}

// ── EAN 配置 | EAN config ───────────────────────────────────────────────────

// SaveEAN 保存 EAN 配置 | Save EAN configuration
func (cs *ConfigStore) SaveEAN(config EANConfigData) error {
	return cs.saveJSON(BucketEAN, "ean", config)
}

// LoadEAN 加载 EAN 配置 | Load EAN configuration
func (cs *ConfigStore) LoadEAN() (*EANConfigData, error) {
	var config EANConfigData
	err := cs.loadJSON(BucketEAN, "ean", &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ── Users 配置 | Users config ───────────────────────────────────────────────

// SaveUsers 保存用户列表 | Save user list
func (cs *ConfigStore) SaveUsers(users []UserConfigData) error {
	return cs.saveJSON(BucketUsers, "users", users)
}

// LoadUsers 加载用户列表 | Load user list
func (cs *ConfigStore) LoadUsers() ([]UserConfigData, error) {
	var users []UserConfigData
	err := cs.loadJSON(BucketUsers, "users", &users)
	if err != nil {
		return nil, err
	}
	if users == nil {
		return []UserConfigData{}, nil
	}
	return users, nil
}

// ── 状态检测 | Status checks ────────────────────────────────────────────────

// HasConfigData 检查配置库是否已有业务数据
// HasConfigData checks if config DB already has business data
func (cs *ConfigStore) HasConfigData() (bool, error) {
	var hasData bool
	err := cs.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketNode))
		if b != nil {
			data := b.Get([]byte("node"))
			if data != nil && len(data) > 0 {
				hasData = true
				return nil
			}
		}

		b = tx.Bucket([]byte(BucketMiddlewares))
		if b != nil {
			data := b.Get([]byte("middlewares"))
			if data != nil && len(data) > 2 { // "[]" means empty
				hasData = true
				return nil
			}
		}

		b = tx.Bucket([]byte(BucketEAN))
		if b != nil {
			data := b.Get([]byte("ean"))
			if data != nil && len(data) > 0 {
				hasData = true
				return nil
			}
		}

		return nil
	})
	return hasData, err
}

// ── 导入/导出 | Import/Export ────────────────────────────────────────────────

// ConfigExport 配置导出结构体 | Config export structure
type ConfigExport struct {
	Node        NodeConfigData        `json:"node"`
	Security    SecurityConfigData    `json:"security"`
	Monitoring  MonitoringConfigData  `json:"monitoring"`
	Middlewares []MiddlewareConfigData `json:"middlewares"`
	EAN         EANConfigData         `json:"ean"`
	Users       []UserConfigData      `json:"users"`
}

// ExportAllConfig 导出所有配置 | Export all configuration
func (cs *ConfigStore) ExportAllConfig() (*ConfigExport, error) {
	export := &ConfigExport{}
	err := cs.db.View(func(tx *bbolt.Tx) error {
		if err := loadFromBucket(tx, BucketNode, "node", &export.Node); err != nil {
			return err
		}
		if err := loadFromBucket(tx, BucketSecurity, "security", &export.Security); err != nil {
			return err
		}
		if err := loadFromBucket(tx, BucketMonitoring, "monitoring", &export.Monitoring); err != nil {
			return err
		}
		if err := loadFromBucket(tx, BucketMiddlewares, "middlewares", &export.Middlewares); err != nil {
			return err
		}
		if err := loadFromBucket(tx, BucketEAN, "ean", &export.EAN); err != nil {
			return err
		}
		if err := loadFromBucket(tx, BucketUsers, "users", &export.Users); err != nil {
			return err
		}
		return nil
	})
	return export, err
}

// ImportConfigReplace 替换式导入配置 | Replace-import configuration
func (cs *ConfigStore) ImportConfigReplace(export *ConfigExport) error {
	if export == nil {
		return fmt.Errorf("import data is required")
	}
	return cs.db.Update(func(tx *bbolt.Tx) error {
		if err := saveToBucket(tx, BucketNode, "node", export.Node); err != nil {
			return err
		}
		if err := saveToBucket(tx, BucketSecurity, "security", export.Security); err != nil {
			return err
		}
		if err := saveToBucket(tx, BucketMonitoring, "monitoring", export.Monitoring); err != nil {
			return err
		}
		if err := saveToBucket(tx, BucketMiddlewares, "middlewares", export.Middlewares); err != nil {
			return err
		}
		if err := saveToBucket(tx, BucketEAN, "ean", export.EAN); err != nil {
			return err
		}
		if err := saveToBucket(tx, BucketUsers, "users", export.Users); err != nil {
			return err
		}

		b := tx.Bucket([]byte(BucketConfigVersion))
		if b != nil {
			return b.Put([]byte(ConfigVersionKey), []byte(ConfigVersionValue))
		}
		return nil
	})
}

// SaveAllConfig 一次性保存全部配置 | Save all configuration in one transaction
func (cs *ConfigStore) SaveAllConfig(
	node NodeConfigData,
	security SecurityConfigData,
	monitoring MonitoringConfigData,
	middlewares []MiddlewareConfigData,
	ean EANConfigData,
	users []UserConfigData,
) error {
	return cs.db.Update(func(tx *bbolt.Tx) error {
		if err := saveToBucket(tx, BucketNode, "node", node); err != nil {
			return err
		}
		if err := saveToBucket(tx, BucketSecurity, "security", security); err != nil {
			return err
		}
		if err := saveToBucket(tx, BucketMonitoring, "monitoring", monitoring); err != nil {
			return err
		}
		if err := saveToBucket(tx, BucketMiddlewares, "middlewares", middlewares); err != nil {
			return err
		}
		if err := saveToBucket(tx, BucketEAN, "ean", ean); err != nil {
			return err
		}
		if err := saveToBucket(tx, BucketUsers, "users", users); err != nil {
			return err
		}
		return nil
	})
}

// ── 配置数据结构体 | Config data structures ──────────────────────────────────

// NodeConfigData 节点配置数据 | Node configuration data
type NodeConfigData struct {
	NodeID        string `json:"node_id"`
	NodeType      string `json:"node_type"`
	PrimaryNodeID string `json:"primary_node_id,omitempty"`
	Listen        string `json:"listen"`
}

// SecurityConfigData 安全配置数据 | Security configuration data
type SecurityConfigData struct {
	JWTSecret  string `json:"jwt_secret"`
	TLSEnabled bool   `json:"tls_enabled"`
	CertFile   string `json:"cert_file,omitempty"`
	KeyFile    string `json:"key_file,omitempty"`
}

// MonitoringConfigData 监控配置数据 | Monitoring configuration data
type MonitoringConfigData struct {
	Enabled    bool   `json:"enabled"`
	Prometheus string `json:"prometheus"`
}

// MiddlewareConfigData 中间件配置数据 | Middleware configuration data
type MiddlewareConfigData struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Type              string   `json:"type"`
	Enabled           bool     `json:"enabled"`
	Broker            string   `json:"broker"`
	ClientID          string   `json:"client_id"`
	Username          string   `json:"username"`
	Password          string   `json:"password"`
	QoS               byte     `json:"qos"`
	CleanSession      bool     `json:"clean_session"`
	KeepAlive         int      `json:"keep_alive"`
	ConnectTimeout    int      `json:"connect_timeout"`
	AutoReconnect     bool     `json:"auto_reconnect"`
	Subscriptions     []string `json:"subscriptions"`
	MQTTVersion       int      `json:"mqtt_version"`
	SSL               bool     `json:"ssl"`
	CAFile            string   `json:"ca_file"`
	ClientCertFile    string   `json:"client_cert_file"`
	ClientKeyFile     string   `json:"client_key_file"`
	ReconnectInterval int      `json:"reconnect_interval"`
}

// EANConfigData EAN 配置数据 | EAN configuration data
type EANConfigData struct {
	Enabled   bool               `json:"enabled"`
	PlannerID string             `json:"planner_id"`
	MQTT      EANMQTTConfigData  `json:"mqtt"`
	NATS      EANNATSConfigData  `json:"nats"`
	Heartbeat EANHeartbeatData   `json:"heartbeat"`
	// V1CommandEnabled V1 命令面开关（指针：nil=未持久化，回退默认 true）
	// V1 command plane switch (pointer: nil=not persisted, fall back to default true)
	V1CommandEnabled *bool `json:"v1_command_enabled,omitempty"`
}

// EANMQTTConfigData EAN MQTT 配置 | EAN MQTT configuration
type EANMQTTConfigData struct {
	Enabled        bool   `json:"enabled"`
	Broker         string `json:"broker"`
	ClientID       string `json:"client_id"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	QoS            int    `json:"qos"`
	KeepAlive      int    `json:"keep_alive"`
	ConnectTimeout int    `json:"connect_timeout"`
	MQTTVersion    int    `json:"mqtt_version"`
}

// EANNATSConfigData EAN NATS 配置 | EAN NATS configuration
type EANNATSConfigData struct {
	Enabled        bool   `json:"enabled"`
	URL            string `json:"url"`
	ClientName     string `json:"client_name"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	Token          string `json:"token"`
	ConnectTimeout int    `json:"connect_timeout"`
	ReconnectWait  int    `json:"reconnect_wait"`
	MaxReconnects  int    `json:"max_reconnects"`
	PingInterval   int    `json:"ping_interval"`
}

// EANHeartbeatData EAN 心跳配置 | EAN heartbeat configuration
type EANHeartbeatData struct {
	TimeoutMultiplier      int `json:"timeout_multiplier"`
	CheckIntervalSec       int `json:"check_interval_sec"`
	MaxOfflineRetentionSec int `json:"max_offline_retention_sec"`
}

// UserConfigData 用户配置数据 | User configuration data
type UserConfigData struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// ── 内部工具函数 | Internal helpers ──────────────────────────────────────────

func (cs *ConfigStore) saveJSON(bucketName, key string, data interface{}) error {
	return cs.db.Update(func(tx *bbolt.Tx) error {
		return saveToBucket(tx, bucketName, key, data)
	})
}

func (cs *ConfigStore) loadJSON(bucketName, key string, result interface{}) error {
	return cs.db.View(func(tx *bbolt.Tx) error {
		return loadFromBucket(tx, bucketName, key, result)
	})
}

func saveToBucket(tx *bbolt.Tx, bucketName, key string, data interface{}) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("config key is required for bucket %s", bucketName)
	}
	b := tx.Bucket([]byte(bucketName))
	if b == nil {
		return fmt.Errorf("bucket %s not found", bucketName)
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return b.Put([]byte(key), bytes)
}

func loadFromBucket(tx *bbolt.Tx, bucketName, key string, result interface{}) error {
	b := tx.Bucket([]byte(bucketName))
	if b == nil {
		return nil
	}
	data := b.Get([]byte(key))
	if data == nil {
		return nil
	}
	return json.Unmarshal(data, result)
}
