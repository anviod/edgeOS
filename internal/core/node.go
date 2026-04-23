package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.etcd.io/bbolt"
	"go.uber.org/zap"
)

// Node 节点接口
type Node interface {
	GetNodeID() string
	GetNodeType() string
	GetStatus() string
	Start(ctx context.Context) error
	Stop() error
}

// baseNode 基础节点实现
type baseNode struct {
	mu       sync.RWMutex
	nodeID   string
	nodeType string
	status   string
	db       *bbolt.DB
	logger   *zap.Logger
	cancel   context.CancelFunc
}

func (n *baseNode) GetNodeID() string {
	return n.nodeID
}

func (n *baseNode) GetNodeType() string {
	return n.nodeType
}

func (n *baseNode) GetStatus() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.status
}

func (n *baseNode) setStatus(s string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.status = s
}

// PrimaryQueen 主控节点
type PrimaryQueen struct {
	baseNode
}

// NewPrimaryQueen 创建主控节点
func NewPrimaryQueen(nodeID string, db *bbolt.DB) *PrimaryQueen {
	logger, _ := zap.NewProduction()
	return &PrimaryQueen{
		baseNode: baseNode{
			nodeID:   nodeID,
			nodeType: "primary",
			status:   "initializing",
			db:       db,
			logger:   logger,
		},
	}
}

func (n *PrimaryQueen) Start(ctx context.Context) error {
	childCtx, cancel := context.WithCancel(ctx)
	n.cancel = cancel
	n.setStatus("running")
	n.logger.Info("PrimaryQueen started", zap.String("node_id", n.nodeID))

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-childCtx.Done():
				return
			case <-ticker.C:
				n.logger.Debug("PrimaryQueen heartbeat", zap.String("node_id", n.nodeID))
			}
		}
	}()
	return nil
}

func (n *PrimaryQueen) Stop() error {
	if n.cancel != nil {
		n.cancel()
	}
	n.setStatus("stopped")
	n.logger.Info("PrimaryQueen stopped", zap.String("node_id", n.nodeID))
	return nil
}

// SecondaryQueen 备用控制节点
type SecondaryQueen struct {
	baseNode
	primaryNodeID string
}

// NewSecondaryQueen 创建备用控制节点
func NewSecondaryQueen(nodeID string, db *bbolt.DB) *SecondaryQueen {
	logger, _ := zap.NewProduction()
	return &SecondaryQueen{
		baseNode: baseNode{
			nodeID:   nodeID,
			nodeType: "secondary",
			status:   "initializing",
			db:       db,
			logger:   logger,
		},
	}
}

func (n *SecondaryQueen) Start(ctx context.Context) error {
	childCtx, cancel := context.WithCancel(ctx)
	n.cancel = cancel
	n.setStatus("running")
	n.logger.Info("SecondaryQueen started", zap.String("node_id", n.nodeID))

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-childCtx.Done():
				return
			case <-ticker.C:
				n.logger.Debug("SecondaryQueen heartbeat", zap.String("node_id", n.nodeID))
			}
		}
	}()
	return nil
}

func (n *SecondaryQueen) Stop() error {
	if n.cancel != nil {
		n.cancel()
	}
	n.setStatus("stopped")
	n.logger.Info("SecondaryQueen stopped", zap.String("node_id", n.nodeID))
	return nil
}

// EdgeCollector 边缘采集节点
type EdgeCollector struct {
	baseNode
}

// NewEdgeCollector 创建边缘采集节点
func NewEdgeCollector(nodeID string, db *bbolt.DB) *EdgeCollector {
	logger, _ := zap.NewProduction()
	return &EdgeCollector{
		baseNode: baseNode{
			nodeID:   nodeID,
			nodeType: "collector",
			status:   "initializing",
			db:       db,
			logger:   logger,
		},
	}
}

func (n *EdgeCollector) Start(ctx context.Context) error {
	childCtx, cancel := context.WithCancel(ctx)
	n.cancel = cancel
	n.setStatus("running")
	n.logger.Info("EdgeCollector started", zap.String("node_id", n.nodeID))

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-childCtx.Done():
				return
			case <-ticker.C:
				n.logger.Debug("EdgeCollector heartbeat", zap.String("node_id", n.nodeID))
			}
		}
	}()
	return nil
}

func (n *EdgeCollector) Stop() error {
	if n.cancel != nil {
		n.cancel()
	}
	n.setStatus("stopped")
	n.logger.Info("EdgeCollector stopped", zap.String("node_id", n.nodeID))
	return nil
}

// ValidateNodeType 验证节点类型
func ValidateNodeType(nodeType string) error {
	switch nodeType {
	case "primary", "secondary", "collector":
		return nil
	default:
		return fmt.Errorf("invalid node type: %s, must be one of: primary, secondary, collector", nodeType)
	}
}
