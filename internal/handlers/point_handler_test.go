package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────── HandlePointReport ────────────────────

func TestPointHandler_HandlePointReport_Success(t *testing.T) {
	svc, cleanup := newTestPointService(t)
	defer cleanup()

	h := NewPointHandler(svc, newTestHub(), newTestLogger())

	payload := map[string]interface{}{
		"header": map[string]interface{}{"source": "n1"},
		"body": map[string]interface{}{
			"node_id":   "n1",
			"device_id": "d1",
			"points": []interface{}{
				map[string]interface{}{"point_id": "p1", "device_id": "d1", "point_name": "Temperature", "data_type": "float"},
				map[string]interface{}{"point_id": "p2", "device_id": "d1", "point_name": "Humidity", "data_type": "float"},
			},
		},
	}
	h.HandlePointReport(nil, buildMsg("edgex/points/report", payload))

	pts, err := svc.ListByDevice("n1", "d1")
	require.NoError(t, err)
	assert.Len(t, pts, 2)
}

func TestPointHandler_HandlePointReport_NodeIDFromHeader(t *testing.T) {
	svc, cleanup := newTestPointService(t)
	defer cleanup()

	h := NewPointHandler(svc, nil, newTestLogger())

	// body.node_id 为空，从 header.source 补充
	payload := map[string]interface{}{
		"header": map[string]interface{}{"source": "n-hdr"},
		"body": map[string]interface{}{
			"node_id":   "",
			"device_id": "d-hdr",
			"points": []interface{}{
				map[string]interface{}{"point_id": "p-hdr", "device_id": "d-hdr", "point_name": "Voltage"},
			},
		},
	}
	h.HandlePointReport(nil, buildMsg("edgex/points/report", payload))

	pts, err := svc.ListByDevice("n-hdr", "d-hdr")
	require.NoError(t, err)
	assert.Len(t, pts, 1)
}

func TestPointHandler_HandlePointReport_MissingNodeID(t *testing.T) {
	svc, cleanup := newTestPointService(t)
	defer cleanup()

	h := NewPointHandler(svc, nil, newTestLogger())

	payload := map[string]interface{}{
		"header": map[string]interface{}{},
		"body": map[string]interface{}{
			"device_id": "d-no-node",
			"points": []interface{}{
				map[string]interface{}{"point_id": "p-no-node"},
			},
		},
	}
	h.HandlePointReport(nil, buildMsg("edgex/points/report", payload))
	// 静默返回，无 panic，无写入
}

func TestPointHandler_HandlePointReport_InvalidPayload(t *testing.T) {
	svc, cleanup := newTestPointService(t)
	defer cleanup()
	h := NewPointHandler(svc, nil, newTestLogger())
	msg := &mockMessage{topic: "edgex/points/report", payload: []byte(`{bad`)}
	h.HandlePointReport(nil, msg)
}

func TestPointHandler_HandlePointReport_RwFieldMapping(t *testing.T) {
	svc, cleanup := newTestPointService(t)
	defer cleanup()

	h := NewPointHandler(svc, nil, newTestLogger())

	// 测试 rw 字段映射：R -> ReadOnly, W -> WriteOnly, RW -> ReadWrite
	payload := map[string]interface{}{
		"header": map[string]interface{}{"source": "n-rw"},
		"body": map[string]interface{}{
			"node_id":   "n-rw",
			"device_id": "d-rw",
			"points": []interface{}{
				map[string]interface{}{"point_id": "p-read", "device_id": "d-rw", "point_name": "ReadOnly", "data_type": "float", "rw": "R", "unit": "C"},
				map[string]interface{}{"point_id": "p-write", "device_id": "d-rw", "point_name": "WriteOnly", "data_type": "int", "rw": "W"},
				map[string]interface{}{"point_id": "p-rw", "device_id": "d-rw", "point_name": "ReadWrite", "data_type": "bool", "rw": "RW"},
				map[string]interface{}{"point_id": "p-empty-rw", "device_id": "d-rw", "point_name": "EmptyRW", "data_type": "float"}, // 无 rw 字段
			},
		},
	}
	h.HandlePointReport(nil, buildMsg("edgex/points/report", payload))

	pts, err := svc.ListByDevice("n-rw", "d-rw")
	require.NoError(t, err)
	assert.Len(t, pts, 4)

	// 验证每个点的属性
	for _, p := range pts {
		switch p.PointID {
		case "p-read":
			assert.False(t, p.ReadWrite, "ReadOnly point should have ReadWrite=false")
			assert.Equal(t, "read", p.PointType)
			assert.Equal(t, "C", p.Units)
		case "p-write":
			assert.True(t, p.ReadWrite, "WriteOnly point should have ReadWrite=true")
			assert.Equal(t, "write", p.PointType)
		case "p-rw":
			assert.True(t, p.ReadWrite, "ReadWrite point should have ReadWrite=true")
			assert.Equal(t, "readwrite", p.PointType)
		case "p-empty-rw":
			assert.True(t, p.ReadWrite, "Empty rw defaults to ReadWrite=true")
			assert.Equal(t, "readwrite", p.PointType)
		}
		// 验证默认填充的字段
		assert.NotEmpty(t, p.ServiceName)
		assert.NotEmpty(t, p.ProfileName)
		assert.NotEmpty(t, p.Description)
	}
}

// ──────────────────── HandleRealtimeData ────────────────────

func TestPointHandler_HandleRealtimeData_FullSnapshot(t *testing.T) {
	svc, cleanup := newTestPointService(t)
	defer cleanup()

	h := NewPointHandler(svc, newTestHub(), newTestLogger())

	payload := map[string]interface{}{
		"header": map[string]interface{}{"message_type": "data_full"},
		"body": map[string]interface{}{
			"node_id":          "n1",
			"device_id":        "d1",
			"is_full_snapshot": true,
			"timestamp":        1700000000000,
			"quality":          "good",
			"points": map[string]interface{}{
				"p1": 3.14,
				"p2": 42.0,
			},
		},
	}
	h.HandleRealtimeData(nil, buildMsg("edgex/data/stream", payload))

	assert.True(t, svc.HasCache("n1", "d1"), "snapshot should be cached after full upload")
}

func TestPointHandler_HandleRealtimeData_DeltaMerge(t *testing.T) {
	svc, cleanup := newTestPointService(t)
	defer cleanup()

	h := NewPointHandler(svc, newTestHub(), newTestLogger())

	// 先发全量
	full := map[string]interface{}{
		"header": map[string]interface{}{"message_type": "data_full"},
		"body": map[string]interface{}{
			"node_id": "n1", "device_id": "d1",
			"is_full_snapshot": true,
			"timestamp":        1700000000000,
			"quality":          "good",
			"points":           map[string]interface{}{"p1": 1.0, "p2": 2.0},
		},
	}
	h.HandleRealtimeData(nil, buildMsg("edgex/data/stream", full))

	// 再发差量（只更新 p1）
	delta := map[string]interface{}{
		"header": map[string]interface{}{"message_type": "data_delta"},
		"body": map[string]interface{}{
			"node_id": "n1", "device_id": "d1",
			"is_full_snapshot": false,
			"timestamp":        1700000001000,
			"quality":          "good",
			"points":           map[string]interface{}{"p1": 9.9},
		},
	}
	h.HandleRealtimeData(nil, buildMsg("edgex/data/stream", delta))

	// 缓存应仍存在
	assert.True(t, svc.HasCache("n1", "d1"))
}

func TestPointHandler_HandleRealtimeData_AutoFullWhenNoCache(t *testing.T) {
	svc, cleanup := newTestPointService(t)
	defer cleanup()

	h := NewPointHandler(svc, nil, newTestLogger())

	// 没有缓存时，即使 is_full_snapshot=false，也应建立快照
	payload := map[string]interface{}{
		"header": map[string]interface{}{"message_type": "data_delta"},
		"body": map[string]interface{}{
			"node_id": "n2", "device_id": "d2",
			"is_full_snapshot": false,
			"timestamp":        0, // handler 应自动填充
			"points":           map[string]interface{}{"px": 7.7},
		},
	}
	h.HandleRealtimeData(nil, buildMsg("edgex/data/stream", payload))

	assert.True(t, svc.HasCache("n2", "d2"), "should create cache even for delta when no prior snapshot")
}

func TestPointHandler_HandleRealtimeData_MissingNodeOrDevice(t *testing.T) {
	svc, cleanup := newTestPointService(t)
	defer cleanup()

	h := NewPointHandler(svc, nil, newTestLogger())

	// node_id 为空，静默返回
	payload := map[string]interface{}{
		"header": map[string]interface{}{},
		"body": map[string]interface{}{
			"node_id": "", "device_id": "",
			"points": map[string]interface{}{"p1": 1.0},
		},
	}
	h.HandleRealtimeData(nil, buildMsg("edgex/data/stream", payload))
}

func TestPointHandler_HandleRealtimeData_InvalidPayload(t *testing.T) {
	svc, cleanup := newTestPointService(t)
	defer cleanup()
	h := NewPointHandler(svc, nil, newTestLogger())
	msg := &mockMessage{topic: "edgex/data/stream", payload: []byte(`not-json`)}
	h.HandleRealtimeData(nil, msg)
}
