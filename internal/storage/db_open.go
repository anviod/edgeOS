package storage

import (
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

// boltOpenTimeout 数据库打开超时时间 | Database open timeout
const boltOpenTimeout = 30 * time.Second

// openBoltDB 打开 bboltDB 文件（带超时与错误提示）| Open bboltDB file with timeout and error hint
func openBoltDB(path string, noGrowSync bool) (*bbolt.DB, error) {
	db, err := bbolt.Open(path, 0600, &bbolt.Options{
		Timeout:    boltOpenTimeout,
		NoGrowSync: noGrowSync,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w", path, err)
	}
	return db, nil
}
