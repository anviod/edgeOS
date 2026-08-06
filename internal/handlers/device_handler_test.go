package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anviod/edgeOS/internal/model"
)

// ──────────────────── HandleDeviceReport ────────────────────

func TestDeviceHandler_HandleDeviceReport_Success(t *testing.T) {
	svc, cleanup := newTestDeviceService(t)
	defer cleanup()

	h := NewDeviceHandler(svc, newTestHub(), newTestLogger())

	payload := map[string]interface{}{
		"header": map[string]interface{}{"source": "n1"},
		"body": map[string]interface{}{
			"node_id": "n1",
			"devices": []interface{}{
				map[string]interface{}{"device_id": "d1", "device_name": "Device One", "protocol": "modbus"},
				map[string]interface{}{"device_id": "d2", "device_name": "Device Two", "protocol": "opcua"},
			},
		},
	}
	h.HandleDeviceReport(nil, buildMsg("edgeCore/devices/report", payload))

	// 验证两个设备均已持久化
	d1, err := svc.GetDevice("n1", "d1")
	require.NoError(t, err)
	assert.Equal(t, "Device One", d1.DeviceName)

	d2, err := svc.GetDevice("n1", "d2")
	require.NoError(t, err)
	assert.Equal(t, "Device Two", d2.DeviceName)
}

func TestDeviceHandler_HandleDeviceReport_NodeIDFromHeader(t *testing.T) {
	svc, cleanup := newTestDeviceService(t)
	defer cleanup()

	h := NewDeviceHandler(svc, nil, newTestLogger())

	// body.node_id 为空，从 header.source 补充
	payload := map[string]interface{}{
		"header": map[string]interface{}{"source": "n-from-header"},
		"body": map[string]interface{}{
			"node_id": "",
			"devices": []interface{}{
				map[string]interface{}{"device_id": "d-hdr", "device_name": "From Header"},
			},
		},
	}
	h.HandleDeviceReport(nil, buildMsg("edgeCore/devices/report", payload))

	d, err := svc.GetDevice("n-from-header", "d-hdr")
	require.NoError(t, err)
	assert.Equal(t, "From Header", d.DeviceName)
}

func TestDeviceHandler_HandleDeviceReport_MissingNodeID(t *testing.T) {
	svc, cleanup := newTestDeviceService(t)
	defer cleanup()

	h := NewDeviceHandler(svc, nil, newTestLogger())

	// 既无 body.node_id 也无 header.source，handler 应静默返回
	payload := map[string]interface{}{
		"header": map[string]interface{}{},
		"body": map[string]interface{}{
			"devices": []interface{}{
				map[string]interface{}{"device_id": "d-missing", "device_name": "No Node"},
			},
		},
	}
	h.HandleDeviceReport(nil, buildMsg("edgeCore/devices/report", payload))

	// 没有 node_id，设备不应写入任何 prefix
	count := svc.CountDevices()
	assert.Equal(t, 0, count)
}

func TestDeviceHandler_HandleDeviceReport_EmptyDevices(t *testing.T) {
	svc, cleanup := newTestDeviceService(t)
	defer cleanup()

	h := NewDeviceHandler(svc, newTestHub(), newTestLogger())

	// 先写入历史设备，空全量上报应对账清空
	require.NoError(t, svc.UpsertDevice("n-empty", &model.EdgeCoreDeviceInfo{DeviceID: "stale"}))

	payload := map[string]interface{}{
		"header": map[string]interface{}{},
		"body": map[string]interface{}{
			"node_id": "n-empty",
			"devices": []interface{}{},
		},
	}
	h.HandleDeviceReport(nil, buildMsg("edgeCore/devices/report", payload))

	count := svc.CountDevices()
	assert.Equal(t, 0, count)
}

func TestDeviceHandler_HandleDeviceReport_ReconcilePrunes(t *testing.T) {
	svc, cleanup := newTestDeviceService(t)
	defer cleanup()

	h := NewDeviceHandler(svc, newTestHub(), newTestLogger())
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{DeviceID: "stale", DeviceName: "Stale"}))
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{DeviceID: "keep", DeviceName: "Old"}))

	payload := map[string]interface{}{
		"header": map[string]interface{}{"source": "n1"},
		"body": map[string]interface{}{
			"node_id": "n1",
			"devices": []interface{}{
				map[string]interface{}{"device_id": "keep", "device_name": "Keep"},
				map[string]interface{}{"device_id": "fresh", "device_name": "Fresh"},
			},
		},
	}
	h.HandleDeviceReport(nil, buildMsg("edgeCore/devices/report", payload))

	devs, err := svc.ListDevices("n1")
	require.NoError(t, err)
	require.Len(t, devs, 2)
	_, err = svc.GetDevice("n1", "stale")
	assert.Error(t, err)
}

func TestDeviceHandler_HandleDeviceReport_InvalidPayload(t *testing.T) {
	svc, cleanup := newTestDeviceService(t)
	defer cleanup()
	h := NewDeviceHandler(svc, nil, newTestLogger())

	// 非法 JSON，不应 panic
	msg := &mockMessage{topic: "edgeCore/devices/report", payload: []byte(`{bad json`)}
	h.HandleDeviceReport(nil, msg)
}

func TestDeviceHandler_HandleDeviceReport_BroadcastNilHub(t *testing.T) {
	svc, cleanup := newTestDeviceService(t)
	defer cleanup()

	// hub 为 nil 时不应 panic
	h := NewDeviceHandler(svc, nil, newTestLogger())

	payload := map[string]interface{}{
		"header": map[string]interface{}{},
		"body": map[string]interface{}{
			"node_id": "n-nil-hub",
			"devices": []interface{}{
				map[string]interface{}{"device_id": "d-nil", "device_name": "Nil Hub"},
			},
		},
	}
	h.HandleDeviceReport(nil, buildMsg("edgeCore/devices/report", payload))
}
