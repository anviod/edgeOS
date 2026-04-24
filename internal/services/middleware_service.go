package services

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/anviod/edgeOS/internal/config"
	"github.com/anviod/edgeOS/internal/model"
	"github.com/google/uuid"
)

const bucketMiddlewares = "middlewares"

// MiddlewareService 中间件配置服务
// 管理中间件的 CRUD（存储在 BoltDB）+ 配置初始化（从 config.yaml）+ 持久化到 config.yaml
type MiddlewareService struct {
	db         *bbolt.DB
	logger     *zap.Logger
	configPath string
}

// NewMiddlewareService 创建中间件配置服务
func NewMiddlewareService(db *bbolt.DB, logger *zap.Logger) *MiddlewareService {
	svc := &MiddlewareService{db: db, logger: logger, configPath: "./config/config.yaml"}
	// 确保 bucket 存在
	db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketMiddlewares))
		return err
	})
	return svc
}

// SetConfigPath 设置配置文件路径
func (s *MiddlewareService) SetConfigPath(path string) {
	s.configPath = path
}

// InitFromConfig 从 config.yaml 加载中间件配置
// 仅初始化配置文件中定义但 BoltDB 中不存在的中间件
func (s *MiddlewareService) InitFromConfig(middlewares []config.MiddlewareMiddlewareConfig) error {
	for _, cfg := range middlewares {
		// 检查是否已存在
		existing, _ := s.Get(cfg.ID)
		if existing != nil {
			s.logger.Info("Middleware already exists in DB, skip init",
				zap.String("id", cfg.ID))
			continue
		}

		// 从配置创建
		m := model.NewMiddlewareConfig()
		m.ID = cfg.ID
		m.Name = cfg.Name
		m.Type = cfg.Type
		if m.Type == "" {
			m.Type = "mqtt"
		}
		m.SetBrokerURL(cfg.Broker)
		m.Username = cfg.Username
		m.Password = cfg.Password
		m.ClientID = cfg.ClientID
		m.QoS = cfg.QoS
		if cfg.QoS == 0 {
			m.QoS = 1
		}
		m.CleanSession = cfg.CleanSession
		m.KeepAlive = cfg.KeepAlive
		if cfg.KeepAlive == 0 {
			m.KeepAlive = 30
		}
		m.ConnectTimeout = cfg.ConnectTimeout
		if cfg.ConnectTimeout == 0 {
			m.ConnectTimeout = 10
		}
		m.AutoReconnect = cfg.AutoReconnect
		m.Enabled = cfg.Enabled
		m.Subscriptions = cfg.Subscriptions
		m.Topics = cfg.Subscriptions
		// 高级设置
		m.MQTTVersion = cfg.MQTTVersion
		m.SSL = cfg.SSL
		m.CAFile = cfg.CAFile
		m.ClientCertFile = cfg.ClientCertFile
		m.ClientKeyFile = cfg.ClientKeyFile
		m.ReconnectInterval = cfg.ReconnectInterval
		if m.ReconnectInterval == 0 {
			m.ReconnectInterval = 5
		}
		if m.Name == "" {
			m.Name = cfg.ID
		}
		m.Status = "disconnected"

		if err := s.saveToDB(m); err != nil {
			s.logger.Error("Failed to init middleware from config",
				zap.String("id", cfg.ID), zap.Error(err))
			continue
		}
		s.logger.Info("Middleware initialized from config",
			zap.String("id", cfg.ID),
			zap.String("name", m.Name),
			zap.String("broker", m.Broker),
			zap.Strings("subscriptions", m.Subscriptions))
	}
	return nil
}

// Create 创建中间件配置
func (s *MiddlewareService) Create(cfg *model.MiddlewareConfig) error {
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}
	now := time.Now().Unix()
	cfg.CreatedAt = now
	cfg.UpdatedAt = now
	if cfg.Status == "" {
		cfg.Status = "disconnected"
	}
	if cfg.QoS == 0 {
		cfg.QoS = 1
	}
	// 设置默认值（如果未指定）
	if cfg.KeepAlive == 0 {
		cfg.KeepAlive = 30
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 10
	}
	// 确保 Broker URL 正确：从 host/port 构建，或解析 broker
	cfg.EnsureBrokerURL()
	// 合并 topics 和 subscriptions
	cfg.Subscriptions = mergeStringSlices(cfg.Subscriptions, cfg.Topics)
	cfg.Topics = cfg.Subscriptions

	// 保存到 BoltDB
	if err := s.saveToDB(cfg); err != nil {
		return err
	}
	// 同步到 config.yaml
	s.syncToConfigFile()
	return nil
}

// Update 更新中间件配置
func (s *MiddlewareService) Update(cfg *model.MiddlewareConfig) error {
	// 先获取现有配置
	existing, err := s.Get(cfg.ID)
	if err != nil {
		return fmt.Errorf("middleware not found: %s", cfg.ID)
	}
	cfg.CreatedAt = existing.CreatedAt
	cfg.UpdatedAt = time.Now().Unix()
	// 确保 Broker URL 正确构建
	cfg.EnsureBrokerURL()
	// 合并 topics 和 subscriptions
	cfg.Subscriptions = mergeStringSlices(cfg.Subscriptions, cfg.Topics)
	cfg.Topics = cfg.Subscriptions

	// 保存到 BoltDB
	if err := s.saveToDB(cfg); err != nil {
		return err
	}
	// 同步到 config.yaml
	s.syncToConfigFile()
	return nil
}

// Delete 删除中间件配置
func (s *MiddlewareService) Delete(id string) error {
	if err := s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketMiddlewares))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(id))
	}); err != nil {
		return err
	}
	// 同步到 config.yaml
	s.syncToConfigFile()
	return nil
}

// Get 获取中间件配置
func (s *MiddlewareService) Get(id string) (*model.MiddlewareConfig, error) {
	var cfg model.MiddlewareConfig
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketMiddlewares))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}
		v := b.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("middleware not found: %s", id)
		}
		return json.Unmarshal(v, &cfg)
	})
	return &cfg, err
}

// List 列出所有中间件配置
func (s *MiddlewareService) List() ([]*model.MiddlewareConfig, error) {
	var cfgs []*model.MiddlewareConfig
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketMiddlewares))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var cfg model.MiddlewareConfig
			if err := json.Unmarshal(v, &cfg); err != nil {
				return nil
			}
			cfgs = append(cfgs, &cfg)
			return nil
		})
	})
	if cfgs == nil {
		cfgs = []*model.MiddlewareConfig{}
	}
	return cfgs, err
}

// ListEnabled 列出所有启用的中间件配置
func (s *MiddlewareService) ListEnabled() ([]*model.MiddlewareConfig, error) {
	all, err := s.List()
	if err != nil {
		return nil, err
	}
	var enabled []*model.MiddlewareConfig
	for _, cfg := range all {
		if cfg.Enabled {
			enabled = append(enabled, cfg)
		}
	}
	return enabled, nil
}

// UpdateStatus 更新消息总线状态
func (s *MiddlewareService) UpdateStatus(id, status, lastError string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketMiddlewares))
		if b == nil {
			return fmt.Errorf("middleware not found: %s", id)
		}
		v := b.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("middleware not found: %s", id)
		}
		var cfg model.MiddlewareConfig
		if err := json.Unmarshal(v, &cfg); err != nil {
			return err
		}
		cfg.Status = status
		cfg.LastError = lastError
		cfg.UpdatedAt = time.Now().Unix()
		data, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), data)
	})
}

// saveToDB 保存到 BoltDB
func (s *MiddlewareService) saveToDB(cfg *model.MiddlewareConfig) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketMiddlewares))
		if err != nil {
			return err
		}
		data, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return b.Put([]byte(cfg.ID), data)
	})
}

// mergeStringSlices 合并去重字符串切片
func mergeStringSlices(a, b []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range append(a, b...) {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// syncToConfigFile 同步中间件配置到 config.yaml
func (s *MiddlewareService) syncToConfigFile() {
	if s.configPath == "" {
		s.logger.Warn("Config path not set, skipping sync to config.yaml")
		return
	}

	// 读取当前配置文件
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		s.logger.Error("Failed to read config file for sync", zap.String("path", s.configPath), zap.Error(err))
		return
	}

	// 解析 YAML
	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		s.logger.Error("Failed to parse config file", zap.Error(err))
		return
	}

	// 获取所有中间件配置
	middlewares, err := s.List()
	if err != nil {
		s.logger.Error("Failed to list middlewares for sync", zap.Error(err))
		return
	}

	// 转换为配置格式
	var mwConfigs []map[string]interface{}
	for _, mw := range middlewares {
		mwCfg := map[string]interface{}{
			"id":                  mw.ID,
			"name":                mw.Name,
			"type":                mw.Type,
			"enabled":             mw.Enabled,
			"broker":              mw.Broker,
			"client_id":           mw.ClientID,
			"username":            mw.Username,
			"password":            mw.Password,
			"qos":                 mw.QoS,
			"clean_session":       mw.CleanSession,
			"keep_alive":          mw.KeepAlive,
			"connect_timeout":     mw.ConnectTimeout,
			"auto_reconnect":      mw.AutoReconnect,
			"subscriptions":       mw.Subscriptions,
			"mqtt_version":        mw.MQTTVersion,
			"ssl":                 mw.SSL,
			"ca_file":             mw.CAFile,
			"client_cert_file":    mw.ClientCertFile,
			"client_key_file":     mw.ClientKeyFile,
			"reconnect_interval":  mw.ReconnectInterval,
		}
		mwConfigs = append(mwConfigs, mwCfg)
	}

	// 更新中间件配置
	cfg["middlewares"] = mwConfigs

	// 写回配置文件
	output, err := yaml.Marshal(cfg)
	if err != nil {
		s.logger.Error("Failed to marshal config for sync", zap.Error(err))
		return
	}

	if err := os.WriteFile(s.configPath, output, 0644); err != nil {
		s.logger.Error("Failed to write config file", zap.String("path", s.configPath), zap.Error(err))
		return
	}

	s.logger.Info("Middleware config synced to config.yaml", zap.String("path", s.configPath))
}
