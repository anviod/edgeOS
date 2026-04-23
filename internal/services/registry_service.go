package services

import (
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/anviod/edgeOS/internal/model"
)

const bucketNodes = "edgex_nodes"

// RegistryService 节点注册服务
type RegistryService struct {
	db *bbolt.DB
}

// NewRegistryService 创建节点注册服务
func NewRegistryService(db *bbolt.DB) *RegistryService {
	return &RegistryService{db: db}
}

// UpsertNode 幂等注册/更新节点
func (s *RegistryService) UpsertNode(node *model.EdgeXNodeInfo) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketNodes))
		if err != nil {
			return err
		}
		// 检查已有记录
		existing := b.Get([]byte(node.NodeID))
		if existing != nil {
			var n model.EdgeXNodeInfo
			if json.Unmarshal(existing, &n) == nil {
				// 保留 access_token 和 expires_at
				if node.AccessToken == "" {
					node.AccessToken = n.AccessToken
					node.ExpiresAt = n.ExpiresAt
				}
			}
		}
		node.LastSeen = time.Now().Unix()
		data, err := json.Marshal(node)
		if err != nil {
			return err
		}
		return b.Put([]byte(node.NodeID), data)
	})
}

// UpdateNodeStatus 更新节点状态
func (s *RegistryService) UpdateNodeStatus(nodeID, status string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketNodes))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}
		v := b.Get([]byte(nodeID))
		if v == nil {
			return fmt.Errorf("node not found: %s", nodeID)
		}
		var node model.EdgeXNodeInfo
		if err := json.Unmarshal(v, &node); err != nil {
			return err
		}
		node.Status = status
		node.LastSeen = time.Now().Unix()
		data, err := json.Marshal(node)
		if err != nil {
			return err
		}
		return b.Put([]byte(nodeID), data)
	})
}

// GetNode 获取节点
func (s *RegistryService) GetNode(nodeID string) (*model.EdgeXNodeInfo, error) {
	var node model.EdgeXNodeInfo
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketNodes))
		if b == nil {
			return fmt.Errorf("node not found: %s", nodeID)
		}
		v := b.Get([]byte(nodeID))
		if v == nil {
			return fmt.Errorf("node not found: %s", nodeID)
		}
		return json.Unmarshal(v, &node)
	})
	return &node, err
}

// ListNodes 列出所有节点
func (s *RegistryService) ListNodes() ([]*model.EdgeXNodeInfo, error) {
	var nodes []*model.EdgeXNodeInfo
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketNodes))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var node model.EdgeXNodeInfo
			if err := json.Unmarshal(v, &node); err != nil {
				return nil
			}
			nodes = append(nodes, &node)
			return nil
		})
	})
	if nodes == nil {
		nodes = []*model.EdgeXNodeInfo{}
	}
	return nodes, err
}

// DeleteNode 删除节点
func (s *RegistryService) DeleteNode(nodeID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketNodes))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(nodeID))
	})
}

// CountNodes 统计节点数量
func (s *RegistryService) CountNodes() (int, int) {
	total, online := 0, 0
	s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketNodes))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			total++
			var node model.EdgeXNodeInfo
			if json.Unmarshal(v, &node) == nil && node.Status == "online" {
				online++
			}
			return nil
		})
	})
	return total, online
}
