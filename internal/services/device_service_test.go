package services

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anviod/edgeOS/internal/model"
)

// ======================== DeviceService ========================

func TestDeviceService_UpsertAndGet(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)

	dev := &model.EdgeXDeviceInfo{
		DeviceID:   "dev-1",
		DeviceName: "Sensor A",
		AdminState: "UNLOCKED",
	}
	require.NoError(t, svc.UpsertDevice("node-1", dev))

	got, err := svc.GetDevice("node-1", "dev-1")
	require.NoError(t, err)
	assert.Equal(t, "dev-1", got.DeviceID)
	assert.Equal(t, "Sensor A", got.DeviceName)
	assert.Greater(t, got.LastSync, int64(0))
}

func TestDeviceService_GetDevice_NotFound(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)
	_, err := svc.GetDevice("node-x", "dev-x")
	assert.Error(t, err)
}

func TestDeviceService_ListDevices(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)

	// 空节点
	devs, err := svc.ListDevices("node-1")
	require.NoError(t, err)
	assert.Empty(t, devs)

	// 写入 node-1 的 2 个设备和 node-2 的 1 个设备
	require.NoError(t, svc.UpsertDevice("node-1", &model.EdgeXDeviceInfo{DeviceID: "d1"}))
	require.NoError(t, svc.UpsertDevice("node-1", &model.EdgeXDeviceInfo{DeviceID: "d2"}))
	require.NoError(t, svc.UpsertDevice("node-2", &model.EdgeXDeviceInfo{DeviceID: "d3"}))

	devs, err = svc.ListDevices("node-1")
	require.NoError(t, err)
	assert.Len(t, devs, 2)

	devs2, err := svc.ListDevices("node-2")
	require.NoError(t, err)
	assert.Len(t, devs2, 1)
}

func TestDeviceService_DeleteDevice(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)
	require.NoError(t, svc.UpsertDevice("node-1", &model.EdgeXDeviceInfo{DeviceID: "del-dev"}))

	require.NoError(t, svc.DeleteDevice("node-1", "del-dev"))

	_, err := svc.GetDevice("node-1", "del-dev")
	assert.Error(t, err)
}

func TestDeviceService_CountDevices(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)
	assert.Equal(t, 0, svc.CountDevices())

	for i := 0; i < 5; i++ {
		require.NoError(t, svc.UpsertDevice("node-1", &model.EdgeXDeviceInfo{
			DeviceID: fmt.Sprintf("d%d", i),
		}))
	}
	assert.Equal(t, 5, svc.CountDevices())
}
