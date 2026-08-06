package server

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anviod/edgeOS/internal/model"
	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/storage"
)

// ─── handleExportConfig 测试 ────────────────────────────────

func TestHandleExportConfig_Empty(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	// 初始化 config buckets（模拟 config.db）
	configStore, err := storage.NewConfigStore(db)
	require.NoError(t, err)
	_ = configStore

	registrySvc := services.NewRegistryService(db)
	dataSvc := &services.DataService{
		DeviceSvc:    services.NewDeviceService(db),
		PointService: services.NewPointService(db),
	}

	app.Get("/api/system/export-config", handleExportConfig(db, registrySvc, dataSvc))

	req := httptest.NewRequest("GET", "/api/system/export-config", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var export FullConfigExport
	err = json.NewDecoder(resp.Body).Decode(&export)
	require.NoError(t, err)

	// 空数据也应返回有效结构 | Empty data should return valid structure
	assert.Equal(t, "1.0", export.Version)
	assert.NotEmpty(t, export.ExportedAt)
	assert.NotNil(t, export.Config)
	assert.Empty(t, export.Runtime.Nodes)
	assert.Empty(t, export.Runtime.Devices)
	assert.Empty(t, export.Runtime.Points)
}

func TestHandleExportConfig_WithNodesAndMappings(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	// 初始化 config store 并写入配置数据
	configStore, err := storage.NewConfigStore(db)
	require.NoError(t, err)
	require.NoError(t, configStore.SaveNodeConfig(storage.NodeConfigData{
		NodeID:   "edgeos-001",
		NodeType: "primary",
		Listen:   ":8000",
	}))
	require.NoError(t, configStore.SaveMiddlewares([]storage.MiddlewareConfigData{
		{ID: "mw-1", Name: "MQTT", Type: "mqtt", Enabled: true, Broker: "127.0.0.1:1883"},
	}))

	// 写入运行时节点和映射关系 | Write runtime nodes and mappings
	registrySvc := services.NewRegistryService(db)
	dataSvc := &services.DataService{
		DeviceSvc:    services.NewDeviceService(db),
		PointService: services.NewPointService(db),
	}

	require.NoError(t, registrySvc.UpsertNode(&model.EdgeCoreNodeInfo{
		NodeID:   "edgeCore-node-001",
		NodeName: "edgeCore Node 1",
		Status:   "online",
	}))

	require.NoError(t, dataSvc.DeviceSvc.UpsertDevice("edgeCore-node-001", &model.EdgeCoreDeviceInfo{
		DeviceID:       "bacnet-2228316",
		DeviceName:     "BACnet Device 2228316",
		OperatingState: "enabled",
	}))

	require.NoError(t, dataSvc.PointService.UpsertPoint("edgeCore-node-001", "bacnet-2228316", &model.EdgeCorePointInfo{
		PointID:   "AnalogInput:0",
		PointName: "Temperature",
		DeviceID:  "bacnet-2228316",
		PointType: "analog_input",
		DataType:  "float",
	}))

	app.Get("/api/system/export-config", handleExportConfig(db, registrySvc, dataSvc))

	req := httptest.NewRequest("GET", "/api/system/export-config", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var export FullConfigExport
	err = json.NewDecoder(resp.Body).Decode(&export)
	require.NoError(t, err)

	// 验证配置数据 | Verify config data
	assert.Equal(t, "edgeos-001", export.Config.Node.NodeID)
	assert.Equal(t, "primary", export.Config.Node.NodeType)
	require.Len(t, export.Config.Middlewares, 1)
	assert.Equal(t, "mw-1", export.Config.Middlewares[0].ID)

	// 验证运行时映射关系 | Verify runtime mappings
	require.Len(t, export.Runtime.Nodes, 1)
	assert.Equal(t, "edgeCore-node-001", export.Runtime.Nodes[0].NodeID)

	require.Len(t, export.Runtime.Devices, 1)
	assert.Equal(t, "bacnet-2228316", export.Runtime.Devices[0].DeviceID)

	require.Len(t, export.Runtime.Points, 1)
	assert.Equal(t, "AnalogInput:0", export.Runtime.Points[0].PointID)
	assert.Equal(t, "bacnet-2228316", export.Runtime.Points[0].DeviceID)
}

func TestHandleExportConfig_MultipleNodesDevices(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	configStore, err := storage.NewConfigStore(db)
	require.NoError(t, err)
	_ = configStore

	registrySvc := services.NewRegistryService(db)
	dataSvc := &services.DataService{
		DeviceSvc:    services.NewDeviceService(db),
		PointService: services.NewPointService(db),
	}

	// 创建多个节点和设备映射 | Create multiple nodes and device mappings
	for i := 0; i < 3; i++ {
		nodeID := "node-" + string(rune('A'+i))
		require.NoError(t, registrySvc.UpsertNode(&model.EdgeCoreNodeInfo{
			NodeID: nodeID,
			Status: "online",
		}))
		for j := 0; j < 2; j++ {
			devID := "dev-" + string(rune('A'+i)) + string(rune('0'+j))
			require.NoError(t, dataSvc.DeviceSvc.UpsertDevice(nodeID, &model.EdgeCoreDeviceInfo{
				DeviceID: devID,
			}))
			require.NoError(t, dataSvc.PointService.UpsertPoint(nodeID, devID, &model.EdgeCorePointInfo{
				PointID:  "point-" + devID,
				DeviceID: devID,
			}))
		}
	}

	app.Get("/api/system/export-config", handleExportConfig(db, registrySvc, dataSvc))

	req := httptest.NewRequest("GET", "/api/system/export-config", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var export FullConfigExport
	err = json.NewDecoder(resp.Body).Decode(&export)
	require.NoError(t, err)

	assert.Len(t, export.Runtime.Nodes, 3)
	assert.Len(t, export.Runtime.Devices, 6)
	assert.Len(t, export.Runtime.Points, 6)
}

// ─── handleServiceRestart 测试 ────────────────────────────────

func TestHandleServiceRestart_ReturnsSuccess(t *testing.T) {
	app := newTestApp()
	app.Post("/api/system/restart", handleServiceRestart())

	req := httptest.NewRequest("POST", "/api/system/restart", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "0", body["code"])
	data := body["data"].(map[string]interface{})
	assert.Contains(t, data["message"], "重启")
	assert.Contains(t, data["delay"], "5s")
}

// ─── Content-Disposition 头测试 ────────────────────────────────

func TestHandleExportConfig_ContentDisposition(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	configStore, _ := storage.NewConfigStore(db)
	_ = configStore

	registrySvc := services.NewRegistryService(db)
	dataSvc := &services.DataService{
		DeviceSvc:    services.NewDeviceService(db),
		PointService: services.NewPointService(db),
	}

	app.Get("/api/system/export-config", handleExportConfig(db, registrySvc, dataSvc))

	req := httptest.NewRequest("GET", "/api/system/export-config", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Disposition"), "attachment; filename=edgeos-config-")
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
}

// ─── 确保 configDB 为 nil 时不 panic（容错测试） ────────────────

func TestHandleExportConfig_NilConfigDB(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	registrySvc := services.NewRegistryService(db)
	dataSvc := &services.DataService{
		DeviceSvc:    services.NewDeviceService(db),
		PointService: services.NewPointService(db),
	}

	// configDB 为 nil 时应返回 500 而非 panic
	app.Get("/api/system/export-config", handleExportConfig(nil, registrySvc, dataSvc))

	req := httptest.NewRequest("GET", "/api/system/export-config", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "1", body["code"])
}
