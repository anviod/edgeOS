package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/model"
	"github.com/google/uuid"
)

const bucketCommands = "edgex_commands"

// ControlService 设备控制服务（命令下发 + 响应追踪）
type ControlService struct {
	db      *bbolt.DB
	logger  *zap.Logger
	pending sync.Map // requestID -> chan *model.CommandRecord
}

// NewControlService 创建控制服务
func NewControlService(db *bbolt.DB, logger *zap.Logger) *ControlService {
	svc := &ControlService{db: db, logger: logger}
	db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketCommands))
		return err
	})
	return svc
}

// CreateCommand 创建命令记录并返回 requestID
func (s *ControlService) CreateCommand(nodeID, deviceID, pointID string, value interface{}) (*model.CommandRecord, error) {
	cmd := &model.CommandRecord{
		ID:        uuid.New().String(),
		NodeID:    nodeID,
		DeviceID:  deviceID,
		PointID:   pointID,
		Value:     value,
		Status:    "pending",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	if err := s.saveCommand(cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}

// WaitResponse 等待命令响应（带超时）
func (s *ControlService) WaitResponse(cmdID string, timeout time.Duration) (*model.CommandRecord, error) {
	ch := make(chan *model.CommandRecord, 1)
	s.pending.Store(cmdID, ch)
	defer s.pending.Delete(cmdID)

	select {
	case cmd := <-ch:
		return cmd, nil
	case <-time.After(timeout):
		s.UpdateCommandStatus(cmdID, "timeout", "command timed out")
		cmd, _ := s.GetCommand(cmdID)
		return cmd, fmt.Errorf("command timeout: %s", cmdID)
	}
}

// HandleResponse 处理命令响应（由 MQTT handler 调用）
func (s *ControlService) HandleResponse(cmdID, status, errMsg string) {
	s.UpdateCommandStatus(cmdID, status, errMsg)
	if ch, ok := s.pending.Load(cmdID); ok {
		cmd, err := s.GetCommand(cmdID)
		if err == nil {
			select {
			case ch.(chan *model.CommandRecord) <- cmd:
			default:
			}
		}
	}
}

// UpdateCommandStatus 更新命令状态
func (s *ControlService) UpdateCommandStatus(cmdID, status, errMsg string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketCommands))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(cmdID))
		if v == nil {
			return fmt.Errorf("command not found: %s", cmdID)
		}
		var cmd model.CommandRecord
		if err := json.Unmarshal(v, &cmd); err != nil {
			return err
		}
		cmd.Status = status
		cmd.Error = errMsg
		cmd.UpdatedAt = time.Now().Unix()
		data, err := json.Marshal(cmd)
		if err != nil {
			return err
		}
		return b.Put([]byte(cmdID), data)
	})
}

// GetCommand 获取命令
func (s *ControlService) GetCommand(cmdID string) (*model.CommandRecord, error) {
	var cmd model.CommandRecord
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketCommands))
		if b == nil {
			return fmt.Errorf("command not found: %s", cmdID)
		}
		v := b.Get([]byte(cmdID))
		if v == nil {
			return fmt.Errorf("command not found: %s", cmdID)
		}
		return json.Unmarshal(v, &cmd)
	})
	return &cmd, err
}

// ListCommands 列出命令（最近N条）
func (s *ControlService) ListCommands(nodeID, deviceID string, limit int) ([]*model.CommandRecord, error) {
	var cmds []*model.CommandRecord
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketCommands))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var cmd model.CommandRecord
			if err := json.Unmarshal(v, &cmd); err != nil {
				return nil
			}
			if (nodeID == "" || cmd.NodeID == nodeID) &&
				(deviceID == "" || cmd.DeviceID == deviceID) {
				cmds = append(cmds, &cmd)
			}
			return nil
		})
	})
	if cmds == nil {
		cmds = []*model.CommandRecord{}
	}
	if limit > 0 && len(cmds) > limit {
		cmds = cmds[len(cmds)-limit:]
	}
	return cmds, err
}

func (s *ControlService) saveCommand(cmd *model.CommandRecord) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketCommands))
		if err != nil {
			return err
		}
		data, err := json.Marshal(cmd)
		if err != nil {
			return err
		}
		return b.Put([]byte(cmd.ID), data)
	})
}

func (s *ControlService) ClearCommands() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		// 删除命令记录 bucket
		err := tx.DeleteBucket([]byte(bucketCommands))
		if err != nil && err != bbolt.ErrBucketNotFound {
			return err
		}
		// 重新创建 bucket
		_, err = tx.CreateBucketIfNotExists([]byte(bucketCommands))
		return err
	})
}
