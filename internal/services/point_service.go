package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.etcd.io/bbolt"

	"github.com/anviod/edgeOS/internal/model"
)

const (
	bucketPoints    = "edgex_points"
	bucketPointData = "edgex_data"
)

// PointService 点位管理服务
type PointService struct {
	db    *bbolt.DB
	cache sync.Map // key: "nodeID:deviceID" -> bool (has snapshot)
}

// NewPointService 创建点位服务
func NewPointService(db *bbolt.DB) *PointService {
	return &PointService{db: db}
}

// pointMetaKey 点位元数据 key
func pointMetaKey(nodeID, deviceID, pointID string) string {
	return fmt.Sprintf("%s:%s:%s", nodeID, deviceID, pointID)
}

// pointDevicePrefix 设备点位前缀
func pointDevicePrefix(nodeID, deviceID string) string {
	return fmt.Sprintf("%s:%s:", nodeID, deviceID)
}

// snapshotKey 快照 key
func snapshotKey(nodeID, deviceID string) string {
	return fmt.Sprintf("%s:%s", nodeID, deviceID)
}

// HasCache 检查设备是否有物模型快照
func (s *PointService) HasCache(nodeID, deviceID string) bool {
	_, ok := s.cache.Load(snapshotKey(nodeID, deviceID))
	return ok
}

// SetCache 标记已有快照
func (s *PointService) SetCache(nodeID, deviceID string) {
	s.cache.Store(snapshotKey(nodeID, deviceID), true)
}

// SaveMeta 保存点位元数据
func (s *PointService) SaveMeta(nodeID string, p *model.EdgeXPointInfo) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketPoints))
		if err != nil {
			return err
		}
		p.LastSync = time.Now().Unix()
		data, err := json.Marshal(p)
		if err != nil {
			return err
		}
		key := pointMetaKey(nodeID, p.DeviceID, p.PointID) // use device-level key
		return b.Put([]byte(key), data)
	})
}

// SaveMetaWithNode 保存带节点ID的点位元数据
func (s *PointService) SaveMetaWithNode(nodeID string, p *model.EdgeXPointInfo) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketPoints))
		if err != nil {
			return err
		}
		p.LastSync = time.Now().Unix()
		data, err := json.Marshal(p)
		if err != nil {
			return err
		}
		key := pointMetaKey(nodeID, p.DeviceID, p.PointID)
		return b.Put([]byte(key), data)
	})
}

// GetMeta 获取点位元数据
func (s *PointService) GetMeta(nodeID, deviceID, pointID string) (*model.EdgeXPointInfo, error) {
	var point model.EdgeXPointInfo
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketPoints))
		if b == nil {
			return fmt.Errorf("point not found: %s/%s/%s", nodeID, deviceID, pointID)
		}
		v := b.Get([]byte(pointMetaKey(nodeID, deviceID, pointID)))
		if v == nil {
			return fmt.Errorf("point not found: %s/%s/%s", nodeID, deviceID, pointID)
		}
		return json.Unmarshal(v, &point)
	})
	return &point, err
}

// ListByDevice 列出设备所有点位元数据
func (s *PointService) ListByDevice(nodeID, deviceID string) ([]*model.EdgeXPointInfo, error) {
	prefix := pointDevicePrefix(nodeID, deviceID)
	var points []*model.EdgeXPointInfo
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketPoints))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.Seek([]byte(prefix)); k != nil && len(string(k)) > len(prefix) && string(k[:len(prefix)]) == prefix; k, v = c.Next() {
			var p model.EdgeXPointInfo
			if err := json.Unmarshal(v, &p); err != nil {
				continue
			}
			points = append(points, &p)
		}
		return nil
	})
	if points == nil {
		points = []*model.EdgeXPointInfo{}
	}
	return points, err
}

// UpsertPoint 保存或更新点位元数据
func (s *PointService) UpsertPoint(nodeID, deviceID string, pt *model.EdgeXPointInfo) error {
	pt.LastSync = time.Now().Unix()
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketPoints))
		if err != nil {
			return err
		}
		data, err := json.Marshal(pt)
		if err != nil {
			return err
		}
		key := pointMetaKey(nodeID, deviceID, pt.PointID)
		return b.Put([]byte(key), data)
	})
}

// UpdatePointValues 更新点位实时值（快照）
func (s *PointService) UpdatePointValues(nodeID, deviceID string, points map[string]interface{}, quality string) error {
	return s.SaveSnapshot(nodeID, deviceID, points, quality, time.Now().Unix(), false)
}

// SaveSnapshot 保存设备数据快照（全量/差量）
func (s *PointService) SaveSnapshot(nodeID, deviceID string, points map[string]interface{}, quality string, ts int64, full bool) error {
	snapshot := model.DeviceSnapshot{
		NodeID:    nodeID,
		DeviceID:  deviceID,
		Points:    points,
		Quality:   quality,
		Timestamp: ts,
	}

	if full {
		// 全量保存
		if err := s.db.Update(func(tx *bbolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte(bucketPointData))
			if err != nil {
				return err
			}
			data, err := json.Marshal(snapshot)
			if err != nil {
				return err
			}
			return b.Put([]byte(snapshotKey(nodeID, deviceID)), data)
		}); err != nil {
			return err
		}
		s.SetCache(nodeID, deviceID)
		return nil
	}

	// 差量 Merge
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketPointData))
		if err != nil {
			return err
		}
		key := []byte(snapshotKey(nodeID, deviceID))
		existing := b.Get(key)
		var merged model.DeviceSnapshot
		if existing != nil {
			if err := json.Unmarshal(existing, &merged); err != nil {
				merged = snapshot
			} else {
				if merged.Points == nil {
					merged.Points = make(map[string]interface{})
				}
				for k, v := range points {
					merged.Points[k] = v
				}
				merged.Timestamp = ts
				merged.Quality = quality
			}
		} else {
			merged = snapshot
		}
		data, err := json.Marshal(merged)
		if err != nil {
			return err
		}
		return b.Put(key, data)
	})
}

// GetSnapshot 获取设备快照
func (s *PointService) GetSnapshot(nodeID, deviceID string) (*model.DeviceSnapshot, error) {
	var snapshot model.DeviceSnapshot
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketPointData))
		if b == nil {
			return fmt.Errorf("no snapshot for %s/%s", nodeID, deviceID)
		}
		v := b.Get([]byte(snapshotKey(nodeID, deviceID)))
		if v == nil {
			return fmt.Errorf("no snapshot for %s/%s", nodeID, deviceID)
		}
		return json.Unmarshal(v, &snapshot)
	})
	return &snapshot, err
}

// DataService 数据持久化服务（保留兼容性）
type DataService struct {
	db           *bbolt.DB
	PointService *PointService
	DeviceSvc    *DeviceService
}

// NewDataService 创建数据服务
func NewDataService(db *bbolt.DB) *DataService {
	return &DataService{
		db:           db,
		PointService: NewPointService(db),
		DeviceSvc:    NewDeviceService(db),
	}
}
