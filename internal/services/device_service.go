package services

import (
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/anviod/edgeOS/internal/model"
)

const bucketDevices = "edgeCore_devices"

// DeviceService 设备管理服务
type DeviceService struct {
	db *bbolt.DB
}

// NewDeviceService 创建设备服务
func NewDeviceService(db *bbolt.DB) *DeviceService {
	return &DeviceService{db: db}
}

// deviceKey 构造设备存储key
func deviceKey(nodeID, deviceID string) string {
	return fmt.Sprintf("%s:%s", nodeID, deviceID)
}

// UpsertDevice 幂等更新设备
func (s *DeviceService) UpsertDevice(nodeID string, device *model.EdgeCoreDeviceInfo) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketDevices))
		if err != nil {
			return err
		}
		device.LastSync = time.Now().Unix()
		data, err := json.Marshal(device)
		if err != nil {
			return err
		}
		return b.Put([]byte(deviceKey(nodeID, device.DeviceID)), data)
	})
}

// GetDevice 获取设备
func (s *DeviceService) GetDevice(nodeID, deviceID string) (*model.EdgeCoreDeviceInfo, error) {
	var device model.EdgeCoreDeviceInfo
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketDevices))
		if b == nil {
			return fmt.Errorf("device not found: %s/%s", nodeID, deviceID)
		}
		v := b.Get([]byte(deviceKey(nodeID, deviceID)))
		if v == nil {
			return fmt.Errorf("device not found: %s/%s", nodeID, deviceID)
		}
		return json.Unmarshal(v, &device)
	})
	return &device, err
}

// ListDevices 列出节点下所有设备
func (s *DeviceService) ListDevices(nodeID string) ([]*model.EdgeCoreDeviceInfo, error) {
	prefix := nodeID + ":"
	var devices []*model.EdgeCoreDeviceInfo
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketDevices))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.Seek([]byte(prefix)); k != nil && len(k) > len(prefix) && string(k[:len(prefix)]) == prefix; k, v = c.Next() {
			var device model.EdgeCoreDeviceInfo
			if err := json.Unmarshal(v, &device); err != nil {
				continue
			}
			devices = append(devices, &device)
		}
		return nil
	})
	if devices == nil {
		devices = []*model.EdgeCoreDeviceInfo{}
	}
	return devices, err
}

// DeleteDevice 删除设备
func (s *DeviceService) DeleteDevice(nodeID, deviceID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketDevices))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(deviceKey(nodeID, deviceID)))
	})
}

// CountDevices 统计设备数量
func (s *DeviceService) CountDevices() int {
	count := 0
	s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketDevices))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			count++
			return nil
		})
	})
	return count
}

// ReconcileDevices 按节点对账全量设备上报：upsert 上报列表，并删除该节点下未再上报的设备。
// edgeCore `edgeCore/devices/report` 为全量快照；若只 upsert 不剪枝，历史已删除设备会残留，导致
// EdgeOS 设备数与 edgeCore 实际设备数不一致。
func (s *DeviceService) ReconcileDevices(nodeID string, reported []model.EdgeCoreDeviceInfo) (upserted, removed int, err error) {
	if nodeID == "" {
		return 0, 0, fmt.Errorf("nodeID is required")
	}

	keep := make(map[string]struct{}, len(reported))
	for i := range reported {
		dev := reported[i]
		if dev.DeviceID == "" {
			continue
		}
		keep[dev.DeviceID] = struct{}{}
		d := dev
		if err := s.UpsertDevice(nodeID, &d); err != nil {
			return upserted, removed, err
		}
		upserted++
	}

	existing, err := s.ListDevices(nodeID)
	if err != nil {
		return upserted, removed, err
	}
	for _, old := range existing {
		if old == nil || old.DeviceID == "" {
			continue
		}
		if _, ok := keep[old.DeviceID]; ok {
			continue
		}
		if err := s.DeleteDevice(nodeID, old.DeviceID); err != nil {
			return upserted, removed, err
		}
		removed++
	}
	return upserted, removed, nil
}

// UpdateDeviceStatus 更新设备状态
func (s *DeviceService) UpdateDeviceStatus(nodeID, deviceID, status string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketDevices))
		if b == nil {
			return fmt.Errorf("device bucket not found")
		}
		key := []byte(deviceKey(nodeID, deviceID))
		v := b.Get(key)
		if v == nil {
			return fmt.Errorf("device not found: %s/%s", nodeID, deviceID)
		}
		var device model.EdgeCoreDeviceInfo
		if err := json.Unmarshal(v, &device); err != nil {
			return err
		}
		device.OperatingState = status
		device.LastSync = time.Now().Unix()
		data, err := json.Marshal(&device)
		if err != nil {
			return err
		}
		return b.Put(key, data)
	})
}
