package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.etcd.io/bbolt"
)

// Storage 双库存储管理器：config.db（配置）+ runtime.db（运行时数据）
// Storage manages dual databases: config.db (configuration) + runtime.db (runtime data)
type Storage struct {
	configDB  *bbolt.DB
	runtimeDB *bbolt.DB
	dataDir   string
}

// NewStorage 创建存储管理器，打开配置库和运行时库并初始化 bucket
// NewStorage creates a Storage, opens both databases and initializes buckets
func NewStorage(dataDir string) (*Storage, error) {
	if dataDir == "" {
		dataDir = "data"
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
	}

	configPath := filepath.Join(dataDir, "config.db")
	runtimePath := filepath.Join(dataDir, "edgeos.db")

	// 打开配置数据库（强一致写入）| Open config DB (strong consistency)
	configDB, err := openBoltDB(configPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to open config database %s: %w", configPath, err)
	}

	// 初始化配置 bucket | Initialize config buckets
	err = configDB.Update(func(tx *bbolt.Tx) error {
		for _, bucket := range configBucketNames {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return err
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
		configDB.Close()
		return nil, fmt.Errorf("failed to init config buckets: %w", err)
	}

	// 打开运行时数据库（允许 NoGrowSync 提升写入性能）| Open runtime DB
	runtimeDB, err := openBoltDB(runtimePath, true)
	if err != nil {
		configDB.Close()
		return nil, fmt.Errorf("failed to open runtime database %s: %w", runtimePath, err)
	}

	// 初始化运行时 bucket | Initialize runtime buckets
	err = runtimeDB.Update(func(tx *bbolt.Tx) error {
		for _, bucket := range runtimeBucketNames {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		runtimeDB.Close()
		configDB.Close()
		return nil, fmt.Errorf("failed to init runtime buckets: %w", err)
	}

	return &Storage{
		configDB:  configDB,
		runtimeDB: runtimeDB,
		dataDir:   dataDir,
	}, nil
}

// Close 关闭所有数据库 | Close all databases
func (s *Storage) Close() error {
	err1 := s.configDB.Close()
	err2 := s.runtimeDB.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

// GetConfigDB 获取配置数据库句柄 | Get config DB handle
func (s *Storage) GetConfigDB() *bbolt.DB {
	return s.configDB
}

// GetRuntimeDB 获取运行时数据库句柄 | Get runtime DB handle
func (s *Storage) GetRuntimeDB() *bbolt.DB {
	return s.runtimeDB
}

// GetDB 获取运行时数据库句柄（兼容旧接口）| Get runtime DB handle (compat alias)
func (s *Storage) GetDB() *bbolt.DB {
	return s.runtimeDB
}

// GetConfigPath 返回配置库文件路径 | Return config DB file path
func (s *Storage) GetConfigPath() string {
	return s.configDB.Path()
}

// GetRuntimePath 返回运行时库文件路径 | Return runtime DB file path
func (s *Storage) GetRuntimePath() string {
	return s.runtimeDB.Path()
}

// GetPath 返回运行时库文件路径（兼容旧接口）| Return runtime DB path (compat alias)
func (s *Storage) GetPath() string {
	return s.GetRuntimePath()
}

// GetDataDir 返回数据目录 | Return data directory
func (s *Storage) GetDataDir() string {
	return s.dataDir
}

// SyncConfigDB 强制同步配置库到磁盘 | Force sync config DB to disk
func (s *Storage) SyncConfigDB() error {
	return s.configDB.Sync()
}

// ── 通用数据读写（运行时库）| Generic data read/write (runtime DB) ──────────

// SaveData 保存数据到运行时库 | Save data to runtime DB
func (s *Storage) SaveData(bucketName string, key string, data interface{}) error {
	return s.runtimeDB.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}
		bytes, err := json.Marshal(data)
		if err != nil {
			return err
		}
		return b.Put([]byte(key), bytes)
	})
}

// GetData 从运行时库读取数据 | Load data from runtime DB
func (s *Storage) GetData(bucketName string, key string, result interface{}) error {
	return s.runtimeDB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}
		data := b.Get([]byte(key))
		if data == nil {
			return fmt.Errorf("key %s not found in bucket %s", key, bucketName)
		}
		return json.Unmarshal(data, result)
	})
}

// DeleteData 从运行时库删除数据 | Delete data from runtime DB
func (s *Storage) DeleteData(bucketName string, key string) error {
	return s.runtimeDB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(key))
	})
}

// LoadAll 遍历运行时库中的所有键值 | Iterate all key-values in runtime DB bucket
func (s *Storage) LoadAll(bucketName string, callback func(k, v []byte) error) error {
	return s.runtimeDB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return nil
		}
		return b.ForEach(callback)
	})
}
