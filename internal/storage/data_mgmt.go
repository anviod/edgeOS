package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

// ── 数据库管理 | Database management ─────────────────────────────────────────

// BucketStats 单个 bucket 统计 | Bucket statistics
type BucketStats struct {
	Name        string `json:"name"`
	RecordCount int    `json:"record_count"`
	TotalSize   int64  `json:"total_size"`
	Category    string `json:"category"`
	Clearable   bool   `json:"clearable"`
	Database    string `json:"database"`
}

// BackupInfo 备份结果 | Backup result info
type BackupInfo struct {
	BackupPath    string `json:"backup_path"`
	BackupTime    string `json:"backup_time"`
	OriginalPath  string `json:"original"`
	FileSizeBytes int64  `json:"size_bytes"`
}

// runtimeClearableBucketMap 运行时库中可清理的 bucket | Clearable runtime buckets
var runtimeClearableBucketMap = map[string]bool{
	"devices":        true,
	"tasks":          true,
	"state":          true,
	"stats":          true,
	"edgeCore_nodes":    true,
	"edgeCore_devices":  true,
	"edgeCore_points":   true,
	"edgeCore_data":     true,
	"edgeCore_alerts":   true,
	"edgeCore_commands": true,
	"middlewares":    true,
}

// ClassifyBucket 按 bucket 名称分类：返回 (类别, 是否可清理)
// ClassifyBucket returns the category and clearability of a bucket by name
func ClassifyBucket(name string) (category string, clearable bool) {
	if IsConfigBucket(name) {
		return "config", false
	}
	if runtimeClearableBucketMap[name] {
		return "runtime", true
	}
	if strings.HasPrefix(name, "device_history_") {
		return "history", true
	}
	return "unknown", false
}

// GetBucketStats 遍历 config.db 与 runtime.db 所有 bucket 统计，返回列表与两库文件总大小。
// GetBucketStats scans all buckets in config.db and runtime.db and returns stats plus total file size.
func (s *Storage) GetBucketStats() ([]BucketStats, int64, error) {
	var stats []BucketStats
	var totalSize int64

	configFileInfo, err := os.Stat(s.configDB.Path())
	if err != nil {
		return nil, 0, err
	}
	totalSize += configFileInfo.Size()

	runtimeFileInfo, err := os.Stat(s.runtimeDB.Path())
	if err != nil {
		return nil, 0, err
	}
	totalSize += runtimeFileInfo.Size()

	scan := func(db *bbolt.DB, database string) error {
		return db.View(func(tx *bbolt.Tx) error {
			return tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
				count := 0
				size := int64(0)
				_ = b.ForEach(func(k, v []byte) error {
					count++
					size += int64(len(k) + len(v))
					return nil
				})
				category, clearable := ClassifyBucket(string(name))
				stats = append(stats, BucketStats{
					Name:        string(name),
					RecordCount: count,
					TotalSize:   size,
					Category:    category,
					Clearable:   clearable,
					Database:    database,
				})
				return nil
			})
		})
	}

	if err := scan(s.configDB, "config"); err != nil {
		return nil, 0, err
	}
	if err := scan(s.runtimeDB, "runtime"); err != nil {
		return nil, 0, err
	}

	sort.SliceStable(stats, func(i, j int) bool {
		if stats[i].Database != stats[j].Database {
			return stats[i].Database == "config"
		}
		return stats[i].Name < stats[j].Name
	})

	return stats, totalSize, nil
}

// ClearBucket 清空指定 bucket 的全部记录（配置库 bucket 受保护，禁止清理）。
// ClearBucket clears all records in a bucket (config buckets are protected).
func (s *Storage) ClearBucket(bucketName string) error {
	if IsConfigBucket(bucketName) {
		return fmt.Errorf("config bucket %s cannot be cleared", bucketName)
	}
	return s.runtimeDB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if err := c.Delete(); err != nil {
				return err
			}
		}
		return nil
	})
}

// ClearAllRuntimeBuckets 清空 runtime.db 中全部 bucket 的记录（配置库不受影响）。
// ClearAllRuntimeBuckets clears all records in every runtime.db bucket (config unaffected).
func (s *Storage) ClearAllRuntimeBuckets() ([]string, error) {
	var cleared []string

	err := s.runtimeDB.Update(func(tx *bbolt.Tx) error {
		var bucketNames []string
		if err := tx.ForEach(func(name []byte, _ *bbolt.Bucket) error {
			bucketNames = append(bucketNames, string(name))
			return nil
		}); err != nil {
			return err
		}

		for _, bucketName := range bucketNames {
			if IsConfigBucket(bucketName) {
				return fmt.Errorf("config bucket %s found in runtime db", bucketName)
			}
			b := tx.Bucket([]byte(bucketName))
			if b == nil {
				continue
			}
			c := b.Cursor()
			for k, _ := c.First(); k != nil; k, _ = c.Next() {
				if err := c.Delete(); err != nil {
					return err
				}
			}
			cleared = append(cleared, bucketName)
		}
		return nil
	})

	return cleared, err
}

const compactTxMaxSize = 65536 // matches bbolt CLI default; avoids OOM during large compacts

// CompactRuntimeDB 压缩运行时数据库文件，回收已删除数据的空间（配置库不受影响）。
// CompactRuntimeDB compacts runtime.db to reclaim space from deleted data.
func (s *Storage) CompactRuntimeDB() error {
	runtimePath := s.runtimeDB.Path()

	if err := s.runtimeDB.Close(); err != nil {
		return fmt.Errorf("failed to close runtime db: %w", err)
	}

	tmpPath := runtimePath + ".compact.tmp"
	_ = os.Remove(tmpPath)

	src, err := bbolt.Open(runtimePath, 0600, &bbolt.Options{
		Timeout:  boltOpenTimeout,
		ReadOnly: true,
	})
	if err != nil {
		if reopenErr := s.reopenRuntimeDB(runtimePath); reopenErr != nil {
			return fmt.Errorf("failed to open runtime db for compact: %w (reopen: %v)", err, reopenErr)
		}
		return fmt.Errorf("failed to open runtime db for compact: %w", err)
	}

	dst, err := openBoltDB(tmpPath, true)
	if err != nil {
		_ = src.Close()
		if reopenErr := s.reopenRuntimeDB(runtimePath); reopenErr != nil {
			return fmt.Errorf("failed to create compact temp db: %w (reopen: %v)", err, reopenErr)
		}
		return fmt.Errorf("failed to create compact temp db: %w", err)
	}

	if err := bbolt.Compact(dst, src, compactTxMaxSize); err != nil {
		_ = dst.Close()
		_ = src.Close()
		_ = os.Remove(tmpPath)
		if reopenErr := s.reopenRuntimeDB(runtimePath); reopenErr != nil {
			return fmt.Errorf("failed to compact runtime db: %w (reopen: %v)", err, reopenErr)
		}
		return fmt.Errorf("failed to compact runtime db: %w", err)
	}

	if err := dst.Close(); err != nil {
		_ = src.Close()
		_ = os.Remove(tmpPath)
		if reopenErr := s.reopenRuntimeDB(runtimePath); reopenErr != nil {
			return fmt.Errorf("failed to close compact temp db: %w (reopen: %v)", err, reopenErr)
		}
		return fmt.Errorf("failed to close compact temp db: %w", err)
	}
	if err := src.Close(); err != nil {
		_ = os.Remove(tmpPath)
		if reopenErr := s.reopenRuntimeDB(runtimePath); reopenErr != nil {
			return fmt.Errorf("failed to close runtime db source: %w (reopen: %v)", err, reopenErr)
		}
		return fmt.Errorf("failed to close runtime db source: %w", err)
	}

	tmpInfo, err := os.Stat(tmpPath)
	if err != nil || tmpInfo.Size() == 0 {
		_ = os.Remove(tmpPath)
		if reopenErr := s.reopenRuntimeDB(runtimePath); reopenErr != nil {
			return fmt.Errorf("compacted file is invalid (reopen: %v)", reopenErr)
		}
		return fmt.Errorf("compacted file is empty, aborting")
	}

	backupPath := runtimePath + ".pre-compact.bak"
	if err := os.Rename(runtimePath, backupPath); err != nil {
		_ = os.Remove(tmpPath)
		if reopenErr := s.reopenRuntimeDB(runtimePath); reopenErr != nil {
			return fmt.Errorf("failed to backup original runtime db: %w (reopen: %v)", err, reopenErr)
		}
		return fmt.Errorf("failed to backup original runtime db: %w", err)
	}

	if err := os.Rename(tmpPath, runtimePath); err != nil {
		_ = os.Rename(backupPath, runtimePath)
		if reopenErr := s.reopenRuntimeDB(runtimePath); reopenErr != nil {
			return fmt.Errorf("failed to replace runtime db with compacted file: %w (reopen: %v)", err, reopenErr)
		}
		return fmt.Errorf("failed to replace runtime db with compacted file: %w", err)
	}

	if err := s.reopenRuntimeDB(runtimePath); err != nil {
		return fmt.Errorf("failed to reopen compacted runtime db: %w", err)
	}

	if err := s.runtimeDB.Update(func(tx *bbolt.Tx) error {
		for _, bucket := range runtimeBucketNames {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to reinit runtime buckets after compact: %w", err)
	}

	_ = os.Remove(backupPath)
	return nil
}

// reopenRuntimeDB 重新打开运行时数据库并替换句柄 | Reopen the runtime DB and replace the handle
func (s *Storage) reopenRuntimeDB(path string) error {
	db, err := openBoltDB(path, true)
	if err != nil {
		return err
	}
	s.runtimeDB = db
	return nil
}

// BackupConfigDB 备份配置数据库到指定目录（优先备份，不包含运行时数据）。
// BackupConfigDB backs up config.db to the given directory (priority backup, runtime data excluded).
func (s *Storage) BackupConfigDB(backupDir string) (*BackupInfo, error) {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup dir: %w", err)
	}

	now := time.Now()
	name := "config." + now.Format("20060102-150405") + ".db"
	backupPath := filepath.Join(backupDir, name)

	f, err := os.Create(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup file: %w", err)
	}
	defer f.Close()

	if err := s.configDB.View(func(tx *bbolt.Tx) error {
		_, err := tx.WriteTo(f)
		return err
	}); err != nil {
		_ = os.Remove(backupPath)
		return nil, fmt.Errorf("failed to write backup: %w", err)
	}

	if err := f.Sync(); err != nil {
		return nil, fmt.Errorf("failed to sync backup file: %w", err)
	}

	info, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	return &BackupInfo{
		BackupPath:    backupPath,
		BackupTime:    now.Format(time.RFC3339),
		OriginalPath:  s.configDB.Path(),
		FileSizeBytes: info.Size(),
	}, nil
}
