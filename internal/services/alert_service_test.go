package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anviod/edgeOS/internal/model"
)

// ======================== AlertService ========================

func TestAlertService_AddAndGet(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewAlertService(db)

	alert := &model.AlertInfo{
		NodeID:  "n1",
		Level:   "critical",
		Message: "Temperature high",
	}
	require.NoError(t, svc.AddAlert(alert))

	// ID 应被自动填充
	assert.NotEmpty(t, alert.ID)
	assert.Equal(t, "active", alert.Status)

	got, err := svc.GetAlert(alert.ID)
	require.NoError(t, err)
	assert.Equal(t, "critical", got.Level)
	assert.Equal(t, "Temperature high", got.Message)
}

func TestAlertService_AddAlert_PresetID(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewAlertService(db)
	a := &model.AlertInfo{ID: "preset-id", Level: "info"}
	require.NoError(t, svc.AddAlert(a))

	got, err := svc.GetAlert("preset-id")
	require.NoError(t, err)
	assert.Equal(t, "preset-id", got.ID)
}

func TestAlertService_GetAlert_NotFound(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewAlertService(db)
	_, err := svc.GetAlert("ghost")
	assert.Error(t, err)
}

func TestAlertService_ListAlerts_All(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewAlertService(db)

	// 空时返回空切片
	list, err := svc.ListAlerts("", 0)
	require.NoError(t, err)
	assert.Empty(t, list)

	require.NoError(t, svc.AddAlert(&model.AlertInfo{Level: "info"}))
	require.NoError(t, svc.AddAlert(&model.AlertInfo{Level: "critical"}))

	list, err = svc.ListAlerts("", 0)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestAlertService_ListAlerts_FilterStatus(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewAlertService(db)

	a1 := &model.AlertInfo{Level: "info"}
	a2 := &model.AlertInfo{Level: "critical"}
	require.NoError(t, svc.AddAlert(a1))
	require.NoError(t, svc.AddAlert(a2))

	// 手动确认其中一个
	require.NoError(t, svc.AcknowledgeAlert(a1.ID, "admin"))

	active, err := svc.ListAlerts("active", 0)
	require.NoError(t, err)
	assert.Len(t, active, 1)
	assert.Equal(t, a2.ID, active[0].ID)

	ack, err := svc.ListAlerts("acknowledged", 0)
	require.NoError(t, err)
	assert.Len(t, ack, 1)
}

func TestAlertService_ListAlerts_Limit(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewAlertService(db)

	for i := 0; i < 10; i++ {
		require.NoError(t, svc.AddAlert(&model.AlertInfo{Level: "info"}))
	}

	list, err := svc.ListAlerts("", 3)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestAlertService_AcknowledgeAlert(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewAlertService(db)

	a := &model.AlertInfo{Level: "major"}
	require.NoError(t, svc.AddAlert(a))

	require.NoError(t, svc.AcknowledgeAlert(a.ID, "operator"))

	got, err := svc.GetAlert(a.ID)
	require.NoError(t, err)
	assert.Equal(t, "acknowledged", got.Status)
	assert.Equal(t, "operator", got.AcknowledgedBy)
	assert.Greater(t, got.AcknowledgedAt, int64(0))
}

func TestAlertService_AcknowledgeAlert_NotFound(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewAlertService(db)
	err := svc.AcknowledgeAlert("ghost", "admin")
	assert.Error(t, err)
}

func TestAlertService_CountAlerts_Today(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewAlertService(db)

	// 无告警时为 0
	assert.Equal(t, 0, svc.CountAlerts())

	// 添加今天的告警
	now := time.Now().Unix()
	require.NoError(t, svc.AddAlert(&model.AlertInfo{CreatedAt: now, Level: "info"}))
	require.NoError(t, svc.AddAlert(&model.AlertInfo{CreatedAt: now, Level: "critical"}))

	assert.Equal(t, 2, svc.CountAlerts())
}
