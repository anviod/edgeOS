package services

import (
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/anviod/edgeOS/internal/model"
)

const bucketNodes = "edgeCore_nodes"

// RegistryService 节点注册服务
type RegistryService struct {
	db *bbolt.DB
}

// NewRegistryService 创建节点注册服务
func NewRegistryService(db *bbolt.DB) *RegistryService {
	return &RegistryService{db: db}
}

// UpsertNode 幂等注册/更新节点
func (s *RegistryService) UpsertNode(node *model.EdgeCoreNodeInfo) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketNodes))
		if err != nil {
			return err
		}
		// 检查已有记录
		existing := b.Get([]byte(node.NodeID))
		if existing != nil {
			var n model.EdgeCoreNodeInfo
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
		var node model.EdgeCoreNodeInfo
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
func (s *RegistryService) GetNode(nodeID string) (*model.EdgeCoreNodeInfo, error) {
	var node model.EdgeCoreNodeInfo
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
func (s *RegistryService) ListNodes() ([]*model.EdgeCoreNodeInfo, error) {
	var nodes []*model.EdgeCoreNodeInfo
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketNodes))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var node model.EdgeCoreNodeInfo
			if err := json.Unmarshal(v, &node); err != nil {
				return nil
			}
			nodes = append(nodes, &node)
			return nil
		})
	})
	if nodes == nil {
		nodes = []*model.EdgeCoreNodeInfo{}
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
			var node model.EdgeCoreNodeInfo
			if json.Unmarshal(v, &node) == nil && node.Status == "online" {
				online++
			}
			return nil
		})
	})
	return total, online
}

// EnsureNodeOnline 若节点不存在则创建最小在线记录；若已存在则刷新为 online。
// 用于 EAN Discovery 镜像与 V1 设备上报兜底，避免「有设备无节点」导致 Dashboard/API 不一致。
func (s *RegistryService) EnsureNodeOnline(nodeID, nodeName, protocol string) error {
	if nodeID == "" {
		return fmt.Errorf("nodeID is required")
	}
	if nodeName == "" {
		nodeName = nodeID
	}
	if protocol == "" {
		protocol = "unknown"
	}

	existing, err := s.GetNode(nodeID)
	if err == nil && existing != nil {
		existing.Status = "online"
		if existing.NodeName == "" {
			existing.NodeName = nodeName
		}
		if existing.Protocol == "" {
			existing.Protocol = protocol
		}
		return s.UpsertNode(existing)
	}

	return s.UpsertNode(&model.EdgeCoreNodeInfo{
		NodeID:   nodeID,
		NodeName: nodeName,
		Protocol: protocol,
		Status:   "online",
	})
}
