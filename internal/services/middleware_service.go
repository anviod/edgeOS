package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/config"
	"github.com/anviod/edgeOS/internal/model"
	"github.com/google/uuid"
)

const bucketMiddlewares = "middlewares"

// MiddlewareService 中间件配置服务
// 管理中间件的 CRUD（存储在 BoltDB），数据库为唯一数据源，不再写回 YAML
// MiddlewareService manages middleware CRUD (stored in BoltDB); DB is the single source of truth, no YAML sync
type MiddlewareService struct {
	db     *bbolt.DB
	logger *zap.Logger
}

// NewMiddlewareService 创建中间件配置服务
// NewMiddlewareService creates a middleware service
func NewMiddlewareService(db *bbolt.DB, logger *zap.Logger) *MiddlewareService {
	svc := &MiddlewareService{db: db, logger: logger}
	// 确保 bucket 存在 | Ensure bucket exists
	db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketMiddlewares))
		return err
	})
	return svc
}

// InitFromConfig 从数据库配置初始化中间件（仅初始化 DB 中不存在的条目）
// InitFromConfig initializes middlewares from config (only entries not yet in DB)
func (s *MiddlewareService) InitFromConfig(middlewares []config.MiddlewareMiddlewareConfig) error {
	for _, cfg := range middlewares {
		// 检查是否已存在（Get 返回非 nil 指针 + error；仅在确实存在时跳过）
		// Check if already exists (Get returns non-nil pointer + error; only skip when it truly exists)
		existing, getErr := s.Get(cfg.ID)
		if getErr == nil && existing != nil && existing.ID != "" {
			s.logger.Info("Middleware already exists in DB, skip init",
				zap.String("id", cfg.ID))
			continue
		}

		// 从配置创建 | Create from config
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
		// 高级设置 | Advanced settings
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

// Create 创建中间件配置 | Create middleware configuration
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
	// 设置默认值（如果未指定）| Set defaults if not specified
	if cfg.KeepAlive == 0 {
		cfg.KeepAlive = 30
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 10
	}
	// 确保 Broker URL 正确：从 host/port 构建，或解析 broker | Ensure Broker URL
	cfg.EnsureBrokerURL()
	// 合并 topics 和 subscriptions | Merge topics and subscriptions
	cfg.Subscriptions = mergeStringSlices(cfg.Subscriptions, cfg.Topics)
	cfg.Topics = cfg.Subscriptions

	// 保存到 BoltDB | Save to BoltDB
	return s.saveToDB(cfg)
}

// Update 更新中间件配置 | Update middleware configuration
func (s *MiddlewareService) Update(cfg *model.MiddlewareConfig) error {
	// 先获取现有配置 | Get existing config first
	existing, err := s.Get(cfg.ID)
	if err != nil {
		return fmt.Errorf("middleware not found: %s", cfg.ID)
	}
	cfg.CreatedAt = existing.CreatedAt
	cfg.UpdatedAt = time.Now().Unix()
	// 确保 Broker URL 正确构建 | Ensure Broker URL
	cfg.EnsureBrokerURL()
	// 合并 topics 和 subscriptions | Merge topics and subscriptions
	cfg.Subscriptions = mergeStringSlices(cfg.Subscriptions, cfg.Topics)
	cfg.Topics = cfg.Subscriptions

	// 保存到 BoltDB | Save to BoltDB
	return s.saveToDB(cfg)
}

// Delete 删除中间件配置 | Delete middleware configuration
func (s *MiddlewareService) Delete(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketMiddlewares))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(id))
	})
}

// Get 获取中间件配置 | Get middleware configuration
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
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// List 列出所有中间件配置 | List all middleware configurations
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

// ListEnabled 列出所有启用的中间件配置 | List all enabled middleware configurations
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

// UpdateStatus 更新消息总线状态 | Update middleware status
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

// saveToDB 保存到 BoltDB | Save to BoltDB
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

// mergeStringSlices 合并去重字符串切片 | Merge and deduplicate string slices
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
