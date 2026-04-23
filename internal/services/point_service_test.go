package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anviod/edgeOS/internal/model"
)

// ======================== PointService ========================

func TestPointService_HasCache_SetCache(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewPointService(db)

	assert.False(t, svc.HasCache("n1", "d1"))
	svc.SetCache("n1", "d1")
	assert.True(t, svc.HasCache("n1", "d1"))
	// 不同设备不受影响
	assert.False(t, svc.HasCache("n1", "d2"))
}

func TestPointService_SaveMetaWithNode_GetMeta(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewPointService(db)

	p := &model.EdgeXPointInfo{
		PointID:   "p-1",
		PointName: "Temperature",
		DeviceID:  "dev-1",
		DataType:  "float",
	}
	require.NoError(t, svc.SaveMetaWithNode("node-1", p))

	got, err := svc.GetMeta("node-1", "dev-1", "p-1")
	require.NoError(t, err)
	assert.Equal(t, "p-1", got.PointID)
	assert.Equal(t, "Temperature", got.PointName)
	assert.Greater(t, got.LastSync, int64(0))
}

func TestPointService_GetMeta_NotFound(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewPointService(db)
	_, err := svc.GetMeta("n", "d", "ghost")
	assert.Error(t, err)
}

func TestPointService_ListByDevice(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewPointService(db)

	// 空时返回空切片
	pts, err := svc.ListByDevice("n1", "d1")
	require.NoError(t, err)
	assert.Empty(t, pts)

	// 写入 n1/d1 的 3 个点位 + n1/d2 的 1 个点位
	for _, pid := range []string{"p1", "p2", "p3"} {
		require.NoError(t, svc.SaveMetaWithNode("n1", &model.EdgeXPointInfo{
			PointID:  pid,
			DeviceID: "d1",
		}))
	}
	require.NoError(t, svc.SaveMetaWithNode("n1", &model.EdgeXPointInfo{
		PointID:  "px",
		DeviceID: "d2",
	}))

	pts, err = svc.ListByDevice("n1", "d1")
	require.NoError(t, err)
	assert.Len(t, pts, 3)
}

func TestPointService_SaveSnapshot_Full(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewPointService(db)

	points := map[string]interface{}{
		"temp":     25.5,
		"humidity": 60,
	}
	require.NoError(t, svc.SaveSnapshot("n1", "d1", points, "good", 1000, true))

	// SetCache 应被调用
	assert.True(t, svc.HasCache("n1", "d1"))

	snap, err := svc.GetSnapshot("n1", "d1")
	require.NoError(t, err)
	assert.Equal(t, "n1", snap.NodeID)
	assert.Equal(t, "d1", snap.DeviceID)
	assert.Equal(t, 25.5, snap.Points["temp"])
}

func TestPointService_SaveSnapshot_DeltaMerge(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewPointService(db)

	// 先全量
	require.NoError(t, svc.SaveSnapshot("n1", "d1", map[string]interface{}{
		"a": 1, "b": 2,
	}, "good", 1000, true))

	// 差量更新 a，新增 c
	require.NoError(t, svc.SaveSnapshot("n1", "d1", map[string]interface{}{
		"a": 99, "c": 3,
	}, "good", 2000, false))

	snap, err := svc.GetSnapshot("n1", "d1")
	require.NoError(t, err)
	// a 被更新，b 保留，c 新增
	assert.Equal(t, float64(99), snap.Points["a"])
	assert.Equal(t, float64(2), snap.Points["b"])
	assert.Equal(t, float64(3), snap.Points["c"])
	assert.Equal(t, int64(2000), snap.Timestamp)
}

func TestPointService_GetSnapshot_NotFound(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewPointService(db)
	_, err := svc.GetSnapshot("n1", "d1")
	assert.Error(t, err)
}
