package ean

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/model"
	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/storage"
	"github.com/anviod/edgeOS/internal/ws"
)

// mockV1NATSTransport implements Subscribe(string, MessageHandler) for unit tests.
type mockV1NATSTransport struct {
	subs map[string]MessageHandler
}

func (m *mockV1NATSTransport) Subscribe(topic string, handler MessageHandler) error {
	if m.subs == nil {
		m.subs = make(map[string]MessageHandler)
	}
	m.subs[topic] = handler
	return nil
}

// newV1NATSPlaneTestEnv builds a V1NATSDataPlane test environment (in-memory bbolt).
func newV1NATSPlaneTestEnv(t *testing.T) (*V1NATSDataPlane, *mockV1NATSTransport, *services.DeviceService, *services.PointService, *services.RegistryService, *services.AlertService) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.NewStorage(dir)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	registrySvc := services.NewRegistryService(store.GetRuntimeDB())
	dataSvc := services.NewDataService(store.GetRuntimeDB())
	alertSvc := services.NewAlertService(store.GetRuntimeDB())
	hub := ws.NewHub(zap.NewNop())

	plane := NewV1NATSDataPlane(registrySvc, dataSvc.DeviceSvc, dataSvc.PointService, alertSvc, hub, zap.NewNop())
	trans := &mockV1NATSTransport{}
	require.NoError(t, plane.Subscribe(trans))
	return plane, trans, dataSvc.DeviceSvc, dataSvc.PointService, registrySvc, alertSvc
}

func envelope(subject string, msgType string, body any) []byte {
	payload, _ := json.Marshal(map[string]interface{}{
		"header": map[string]interface{}{
			"message_id":   "msg-1",
			"timestamp":    1744680000000,
			"source":       "edgeCore-node-001",
			"message_type": msgType,
			"version":      "1.0",
		},
		"body": body,
	})
	return payload
}

// TestV1NATSDataPlane_SubscriptionsRegistered verifies all required V1 NATS subjects are subscribed.
func TestV1NATSDataPlane_SubscriptionsRegistered(t *testing.T) {
	_, trans, _, _, _, _ := newV1NATSPlaneTestEnv(t)
	required := []string{
		"edgeCore.devices.report",
		"edgeCore.devices.*.*.online",
		"edgeCore.devices.*.*.offline",
		"edgeCore.points.report",
		"edgeCore.points.*.*",
		"edgeCore.data.*.*",
		"edgeCore.events.alert",
		"edgeCore.events.error",
		"edgeCore.events.info",
		"edgeCore.nodes.register",
		"edgeCore.nodes.*.heartbeat",
		"edgeCore.nodes.*.status",
	}
	for _, sub := range required {
		_, ok := trans.subs[sub]
		require.True(t, ok, "expected subscription for subject %s", sub)
	}
}

// TestV1NATSDataPlane_DeviceReport verifies NATS device_report reconciliation (OS-23 core).
func TestV1NATSDataPlane_DeviceReport(t *testing.T) {
	_, trans, deviceSvc, _, registrySvc, _ := newV1NATSPlaneTestEnv(t)

	require.NoError(t, deviceSvc.UpsertDevice("edgeCore-node-001", &model.EdgeCoreDeviceInfo{DeviceID: "stale-dev"}))
	require.NoError(t, registrySvc.EnsureNodeOnline("edgeCore-node-001", "edgeCore-node-001", "nats"))

	body := map[string]interface{}{
		"node_id": "edgeCore-node-001",
		"devices": []map[string]interface{}{
			{"device_id": "bacnet-2228316", "device_name": "RoomController 2228316"},
			{"device_id": "modbus-1", "device_name": "Modbus Slave 1"},
		},
	}
	payload := envelope("edgeCore.devices.report", "device_report", body)
	trans.subs["edgeCore.devices.report"]("edgeCore.devices.report", payload, "nats")

	devices, err := deviceSvc.ListDevices("edgeCore-node-001")
	require.NoError(t, err)
	require.Len(t, devices, 2)
	ids := map[string]bool{}
	for _, d := range devices {
		ids[d.DeviceID] = true
	}
	require.False(t, ids["stale-dev"], "stale device should be pruned")
	require.True(t, ids["bacnet-2228316"])
	require.True(t, ids["modbus-1"])

	node, err := registrySvc.GetNode("edgeCore-node-001")
	require.NoError(t, err)
	require.Equal(t, "online", node.Status)
}

// TestV1NATSDataPlane_RealtimeData verifies NATS realtime data into snapshot.
func TestV1NATSDataPlane_RealtimeData(t *testing.T) {
	_, trans, _, pointSvc, _, _ := newV1NATSPlaneTestEnv(t)

	body := map[string]interface{}{
		"node_id":   "edgeCore-node-001",
		"device_id": "bacnet-2228316",
		"timestamp": 1744680000000,
		"points": map[string]interface{}{
			"temperature": 25.5,
			"humidity":    60.2,
		},
		"quality": "good",
	}
	payload := envelope("edgeCore.data.edgeCore-node-001.bacnet-2228316", "data", body)
	trans.subs["edgeCore.data.*.*"]("edgeCore.data.edgeCore-node-001.bacnet-2228316", payload, "nats")

	snap, err := pointSvc.GetSnapshot("edgeCore-node-001", "bacnet-2228316")
	require.NoError(t, err)
	require.NotNil(t, snap)
	require.Equal(t, 25.5, snap.Points["temperature"])
}

// TestV1NATSDataPlane_PointSync verifies NATS full point sync.
func TestV1NATSDataPlane_PointSync(t *testing.T) {
	_, trans, _, pointSvc, _, _ := newV1NATSPlaneTestEnv(t)

	body := map[string]interface{}{
		"node_id":   "edgeCore-node-001",
		"device_id": "bacnet-2228316",
		"points": []map[string]interface{}{
			{"point_id": "ai_0", "point_name": "AnalogInput 0", "data_type": "float32"},
			{"point_id": "av_1", "point_name": "AnalogValue 1", "data_type": "float32"},
		},
	}
	payload := envelope("edgeCore.points.edgeCore-node-001.bacnet-2228316", "point_sync", body)
	trans.subs["edgeCore.points.*.*"]("edgeCore.points.edgeCore-node-001.bacnet-2228316", payload, "nats")

	points, err := pointSvc.ListByDevice("edgeCore-node-001", "bacnet-2228316")
	require.NoError(t, err)
	require.Len(t, points, 2)
}

// TestV1NATSDataPlane_Alert verifies NATS alert ingestion.
func TestV1NATSDataPlane_Alert(t *testing.T) {
	_, trans, _, _, _, alertSvc := newV1NATSPlaneTestEnv(t)

	body := map[string]interface{}{
		"id":       "alert-1",
		"node_id":  "edgeCore-node-001",
		"level":    "critical",
		"message":  "device offline",
	}
	payload := envelope("edgeCore.events.alert", "alert", body)
	trans.subs["edgeCore.events.alert"]("edgeCore.events.alert", payload, "nats")

	alerts, err := alertSvc.ListAlerts("", 10)
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	require.Equal(t, "alert-1", alerts[0].ID)
}

// TestV1NATSDataPlane_NodeRegister verifies NATS node registration.
func TestV1NATSDataPlane_NodeRegister(t *testing.T) {
	_, trans, _, _, registrySvc, _ := newV1NATSPlaneTestEnv(t)

	body := map[string]interface{}{
		"node_id":   "edgeCore-node-001",
		"node_name": "edgeCore Gateway",
		"model":     "edgeCore",
		"version":   "1.0.0",
	}
	payload := envelope("edgeCore.nodes.register", "node_register", body)
	trans.subs["edgeCore.nodes.register"]("edgeCore.nodes.register", payload, "nats")

	node, err := registrySvc.GetNode("edgeCore-node-001")
	require.NoError(t, err)
	require.Equal(t, "online", node.Status)
	require.Equal(t, "edgeCore Gateway", node.NodeName)
}
