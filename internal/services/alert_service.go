package services

import (
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/anviod/edgeOS/internal/model"
	"github.com/google/uuid"
)

const bucketAlerts = "edgex_alerts"

// AlertService 告警服务
type AlertService struct {
	db *bbolt.DB
}

// NewAlertService 创建告警服务
func NewAlertService(db *bbolt.DB) *AlertService {
	return &AlertService{db: db}
}

// AddAlert 添加告警
func (s *AlertService) AddAlert(alert *model.AlertInfo) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketAlerts))
		if err != nil {
			return err
		}
		if alert.ID == "" {
			alert.ID = uuid.New().String()
		}
		if alert.CreatedAt == 0 {
			alert.CreatedAt = time.Now().Unix()
		}
		if alert.Status == "" {
			alert.Status = "active"
		}
		data, err := json.Marshal(alert)
		if err != nil {
			return err
		}
		return b.Put([]byte(alert.ID), data)
	})
}

// GetAlert 获取告警
func (s *AlertService) GetAlert(id string) (*model.AlertInfo, error) {
	var alert model.AlertInfo
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketAlerts))
		if b == nil {
			return fmt.Errorf("alert not found: %s", id)
		}
		v := b.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("alert not found: %s", id)
		}
		return json.Unmarshal(v, &alert)
	})
	return &alert, err
}

// ListAlerts 列出告警（按创建时间倒序）
func (s *AlertService) ListAlerts(status string, limit int) ([]*model.AlertInfo, error) {
	var alerts []*model.AlertInfo
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketAlerts))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var a model.AlertInfo
			if err := json.Unmarshal(v, &a); err != nil {
				return nil
			}
			if status == "" || a.Status == status {
				alerts = append(alerts, &a)
			}
			return nil
		})
	})
	if alerts == nil {
		alerts = []*model.AlertInfo{}
	}
	// 限制数量
	if limit > 0 && len(alerts) > limit {
		alerts = alerts[len(alerts)-limit:]
	}
	return alerts, err
}

// AcknowledgeAlert 确认告警
func (s *AlertService) AcknowledgeAlert(id, username string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketAlerts))
		if b == nil {
			return fmt.Errorf("alert not found: %s", id)
		}
		v := b.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("alert not found: %s", id)
		}
		var alert model.AlertInfo
		if err := json.Unmarshal(v, &alert); err != nil {
			return err
		}
		alert.Status = "acknowledged"
		alert.AcknowledgedAt = time.Now().Unix()
		alert.AcknowledgedBy = username
		data, err := json.Marshal(alert)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), data)
	})
}

// CountAlerts 统计活跃告警数量
func (s *AlertService) CountAlerts() int {
	count := 0
	s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketAlerts))
		if b == nil {
			return nil
		}
		today := time.Now().Truncate(24 * time.Hour).Unix()
		return b.ForEach(func(k, v []byte) error {
			var a model.AlertInfo
			if json.Unmarshal(v, &a) == nil && a.CreatedAt >= today {
				count++
			}
			return nil
		})
	})
	return count
}
