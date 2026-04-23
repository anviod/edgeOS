package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/model"
)

// ======================== MiddlewareService ========================

func newTestMiddlewareSvc(t *testing.T) (*MiddlewareService, func()) {
	t.Helper()
	db, cleanup := openTestDB(t)
	return NewMiddlewareService(db, zap.NewNop()), cleanup
}

func TestMiddlewareService_CreateAndGet(t *testing.T) {
	svc, cleanup := newTestMiddlewareSvc(t)
	defer cleanup()

	cfg := &model.MiddlewareConfig{
		Name: "Local MQTT",
		Type: "mqtt",
		Host: "localhost",
		Port: 1883,
	}
	require.NoError(t, svc.Create(cfg))

	// ID 应被自动生成
	assert.NotEmpty(t, cfg.ID)
	assert.Equal(t, "disconnected", cfg.Status)
	assert.Greater(t, cfg.CreatedAt, int64(0))

	got, err := svc.Get(cfg.ID)
	require.NoError(t, err)
	assert.Equal(t, "Local MQTT", got.Name)
	assert.Equal(t, "mqtt", got.Type)
}

func TestMiddlewareService_Create_PresetID(t *testing.T) {
	svc, cleanup := newTestMiddlewareSvc(t)
	defer cleanup()

	cfg := &model.MiddlewareConfig{ID: "my-id", Name: "test", Type: "nats"}
	require.NoError(t, svc.Create(cfg))

	got, err := svc.Get("my-id")
	require.NoError(t, err)
	assert.Equal(t, "my-id", got.ID)
}

func TestMiddlewareService_Get_NotFound(t *testing.T) {
	svc, cleanup := newTestMiddlewareSvc(t)
	defer cleanup()

	_, err := svc.Get("ghost")
	assert.Error(t, err)
}

func TestMiddlewareService_Update(t *testing.T) {
	svc, cleanup := newTestMiddlewareSvc(t)
	defer cleanup()

	cfg := &model.MiddlewareConfig{Name: "before", Type: "mqtt"}
	require.NoError(t, svc.Create(cfg))

	cfg.Name = "after"
	cfg.Port = 1884
	require.NoError(t, svc.Update(cfg))

	got, err := svc.Get(cfg.ID)
	require.NoError(t, err)
	assert.Equal(t, "after", got.Name)
	assert.Equal(t, 1884, got.Port)
}

func TestMiddlewareService_Delete(t *testing.T) {
	svc, cleanup := newTestMiddlewareSvc(t)
	defer cleanup()

	cfg := &model.MiddlewareConfig{Name: "to-delete", Type: "mqtt"}
	require.NoError(t, svc.Create(cfg))

	require.NoError(t, svc.Delete(cfg.ID))

	_, err := svc.Get(cfg.ID)
	assert.Error(t, err)
}

func TestMiddlewareService_Delete_EmptyBucket(t *testing.T) {
	svc, cleanup := newTestMiddlewareSvc(t)
	defer cleanup()

	// 删除不存在的 ID 不应报错
	require.NoError(t, svc.Delete("nonexistent"))
}

func TestMiddlewareService_List(t *testing.T) {
	svc, cleanup := newTestMiddlewareSvc(t)
	defer cleanup()

	// 空时返回空切片
	list, err := svc.List()
	require.NoError(t, err)
	assert.Empty(t, list)

	require.NoError(t, svc.Create(&model.MiddlewareConfig{Name: "m1", Type: "mqtt"}))
	require.NoError(t, svc.Create(&model.MiddlewareConfig{Name: "m2", Type: "nats"}))

	list, err = svc.List()
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestMiddlewareService_UpdateStatus(t *testing.T) {
	svc, cleanup := newTestMiddlewareSvc(t)
	defer cleanup()

	cfg := &model.MiddlewareConfig{Name: "m", Type: "mqtt"}
	require.NoError(t, svc.Create(cfg))

	require.NoError(t, svc.UpdateStatus(cfg.ID, "connected", ""))

	got, err := svc.Get(cfg.ID)
	require.NoError(t, err)
	assert.Equal(t, "connected", got.Status)
	assert.Empty(t, got.LastError)
}

func TestMiddlewareService_UpdateStatus_WithError(t *testing.T) {
	svc, cleanup := newTestMiddlewareSvc(t)
	defer cleanup()

	cfg := &model.MiddlewareConfig{Name: "m", Type: "mqtt"}
	require.NoError(t, svc.Create(cfg))

	require.NoError(t, svc.UpdateStatus(cfg.ID, "error", "connection refused"))

	got, err := svc.Get(cfg.ID)
	require.NoError(t, err)
	assert.Equal(t, "error", got.Status)
	assert.Equal(t, "connection refused", got.LastError)
}

func TestMiddlewareService_UpdateStatus_NotFound(t *testing.T) {
	svc, cleanup := newTestMiddlewareSvc(t)
	defer cleanup()

	err := svc.UpdateStatus("ghost", "connected", "")
	assert.Error(t, err)
}

func TestMiddlewareService_ListEnabled(t *testing.T) {
	svc, cleanup := newTestMiddlewareSvc(t)
	defer cleanup()

	require.NoError(t, svc.Create(&model.MiddlewareConfig{Name: "enabled-1", Type: "mqtt", Enabled: true}))
	require.NoError(t, svc.Create(&model.MiddlewareConfig{Name: "disabled", Type: "nats", Enabled: false}))
	require.NoError(t, svc.Create(&model.MiddlewareConfig{Name: "enabled-2", Type: "mqtt", Enabled: true}))

	enabled, err := svc.ListEnabled()
	require.NoError(t, err)
	assert.Len(t, enabled, 2)
	for _, c := range enabled {
		assert.True(t, c.Enabled)
	}
}
