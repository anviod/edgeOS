package discovery

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/model"
)

const bucketName = "edgex_nodes"

// DiscoveryService EdgeX节点发现服务
type DiscoveryService struct {
	db     *bbolt.DB
	nodeID string
	host   string
	secret string
	logger *zap.Logger
	client *http.Client
}

// NewDiscoveryService 创建发现服务
func NewDiscoveryService(db *bbolt.DB, nodeID, host, secret string) *DiscoveryService {
	logger, _ := zap.NewProduction()
	return &DiscoveryService{
		db:     db,
		nodeID: nodeID,
		host:   host,
		secret: secret,
		logger: logger,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Start 启动发现服务
func (s *DiscoveryService) Start() error {
	if err := s.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	}); err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}
	s.logger.Info("DiscoveryService started", zap.String("node_id", s.nodeID))
	return nil
}

// ListNodes 列出所有节点
func (s *DiscoveryService) ListNodes() ([]model.EdgeXNodeInfo, error) {
	var nodes []model.EdgeXNodeInfo
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			node, err := model.DecodeNodeInfo(v)
			if err != nil {
				return nil
			}
			nodes = append(nodes, *node)
			return nil
		})
	})
	if nodes == nil {
		nodes = []model.EdgeXNodeInfo{}
	}
	return nodes, err
}

// GetNode 根据ID获取节点
func (s *DiscoveryService) GetNode(id string) (*model.EdgeXNodeInfo, error) {
	var node *model.EdgeXNodeInfo
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}
		v := b.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("node not found: %s", id)
		}
		n, err := model.DecodeNodeInfo(v)
		if err != nil {
			return err
		}
		node = n
		return nil
	})
	return node, err
}

// SaveNode 保存节点信息
func (s *DiscoveryService) SaveNode(node *model.EdgeXNodeInfo) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}
		data, err := model.EncodeNodeInfo(node)
		if err != nil {
			return err
		}
		return b.Put([]byte(node.NodeID), data)
	})
}

// DeleteNode 删除节点
func (s *DiscoveryService) DeleteNode(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}
		return b.Delete([]byte(id))
	})
}

// AddNode 手动添加节点
func (s *DiscoveryService) AddNode(ip, port, username, password string) (*model.EdgeXNodeInfo, error) {
	if port == "" {
		port = "59880"
	}
	target := fmt.Sprintf("%s:%s", ip, port)
	node, err := s.ScanIP(target)
	if err != nil {
		// 扫描失败时创建离线节点记录
		nodeID := fmt.Sprintf("edgex-%s-%s", strings.ReplaceAll(ip, ".", "-"), port)
		node = &model.EdgeXNodeInfo{
			NodeID:   nodeID,
			NodeName: fmt.Sprintf("EdgeX@%s:%s", ip, port),
			Endpoint: &model.EndpointInfo{Host: ip, Port: port},
			Status:   "offline",
			LastSeen: time.Now().Unix(),
		}
	}
	if err := s.SaveNode(node); err != nil {
		return nil, fmt.Errorf("failed to save node: %w", err)
	}
	return node, nil
}

// ScanIP 扫描IP探测EdgeX节点
func (s *DiscoveryService) ScanIP(target string) (*model.EdgeXNodeInfo, error) {
	url := fmt.Sprintf("http://%s/api/v2/ping", target)
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("scan failed for %s: %w", target, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, target)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	parts := strings.Split(target, ":")
	host := parts[0]
	port := 59880
	if len(parts) > 1 {
		port = parsePort(parts[1])
	}

	nodeID := fmt.Sprintf("edgex-%s-%d", strings.ReplaceAll(host, ".", "-"), port)
	nodeName := fmt.Sprintf("EdgeX@%s", target)
	if v, ok := result["serviceName"].(string); ok && v != "" {
		nodeName = v
	}

	node := &model.EdgeXNodeInfo{
		NodeID:   nodeID,
		NodeName: nodeName,
		Endpoint: &model.EndpointInfo{Host: host, Port: fmt.Sprintf("%d", port)},
		Status:   "online",
		LastSeen: time.Now().Unix(),
	}

	_ = s.SaveNode(node)
	return node, nil
}

func parsePort(s string) int {
	p := 59880
	fmt.Sscanf(s, "%d", &p)
	return p
}
