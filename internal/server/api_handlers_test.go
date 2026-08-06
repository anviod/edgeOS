package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/config"
	"github.com/anviod/edgeOS/internal/ean"
	"github.com/anviod/edgeOS/internal/messaging"
	"github.com/anviod/edgeOS/internal/model"
	"github.com/anviod/edgeOS/internal/services"
)

// ─── 测试辅助 ────────────────────────────────

func openTestDB(t *testing.T) (*bbolt.DB, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "edgeos_server_test_*.db")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	f.Close()
	db, err := bbolt.Open(f.Name(), 0600, nil)
	if err != nil {
		os.Remove(f.Name())
		t.Fatalf("open bbolt: %v", err)
	}
	return db, func() {
		db.Close()
		os.Remove(f.Name())
	}
}

func newTestApp() *fiber.App {
	return fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
}

// getTestToken 返回一个有效的 JWT token
func getTestToken() string {
	j := NewJWT()
	claims := CustomClaims{
		Name:  "admin",
		Email: "admin@edgeos.local",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "admin",
		},
	}
	token, _ := j.CreateToken(claims)
	return token
}

func setAuthHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+getTestToken())
}

// ─── handleLogin 测试 ────────────────────────────────

func TestHandleLogin_Success(t *testing.T) {
	app := newTestApp()
	cfg := &config.Config{}
	app.Post("/api/auth/login", handleLogin(cfg))

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader([]byte(`{"username":"admin","password":"passwd@123"}`)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "0", body["code"])
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "admin", data["username"])
	assert.NotEmpty(t, data["token"])
}

func TestHandleLogin_InvalidCredentials(t *testing.T) {
	app := newTestApp()
	cfg := &config.Config{}
	app.Post("/api/auth/login", handleLogin(cfg))

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader([]byte(`{"username":"admin","password":"wrong"}`)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "1", body["code"])
	assert.Contains(t, body["msg"], "错误")
}

func TestHandleLogin_InvalidBody(t *testing.T) {
	app := newTestApp()
	cfg := &config.Config{}
	app.Post("/api/auth/login", handleLogin(cfg))

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader([]byte(`{invalid json}`)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// ─── handleListMiddlewares 测试 ────────────────────────────────

func TestHandleListMiddlewares_Empty(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewMiddlewareService(db, zap.NewNop())
	app.Get("/api/middlewares", handleListMiddlewares(svc))

	req := httptest.NewRequest("GET", "/api/middlewares", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "0", body["code"])
	data := body["data"].(map[string]interface{})
	assert.Empty(t, data["middlewares"])
}

func TestHandleListMiddlewares_WithData(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewMiddlewareService(db, zap.NewNop())
	app.Get("/api/middlewares", handleListMiddlewares(svc))

	// 先创建数据
	svc.Create(&model.MiddlewareConfig{ID: "mw-1", Name: "MQTT Broker", Type: "mqtt"})
	svc.Create(&model.MiddlewareConfig{ID: "mw-2", Name: "NATS Server", Type: "nats"})

	req := httptest.NewRequest("GET", "/api/middlewares", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data := body["data"].(map[string]interface{})
	mws := data["middlewares"].([]interface{})
	assert.Len(t, mws, 2)
}

// ─── handleCreateMiddleware 测试 ────────────────────────────────

func TestHandleCreateMiddleware_Success(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewMiddlewareService(db, zap.NewNop())
	app.Post("/api/middlewares", handleCreateMiddleware(svc))

	payload := `{"name":"Test MQTT","type":"mqtt","host":"localhost","port":1883}`
	req := httptest.NewRequest("POST", "/api/middlewares", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data := body["data"].(map[string]interface{})
	mw := data["middleware"].(map[string]interface{})
	assert.Equal(t, "Test MQTT", mw["name"])
	assert.NotEmpty(t, mw["id"])
}

func TestHandleCreateMiddleware_InvalidBody(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewMiddlewareService(db, zap.NewNop())
	app.Post("/api/middlewares", handleCreateMiddleware(svc))

	req := httptest.NewRequest("POST", "/api/middlewares", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// ─── handleConnectMiddleware 测试 ────────────────────────────────

func TestHandleConnectMiddleware_NotFound(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewMiddlewareService(db, zap.NewNop())
	mgr := &messaging.Manager{}
	app.Post("/api/middlewares/:id/connect", handleConnectMiddleware(svc, mgr))

	req := httptest.NewRequest("POST", "/api/middlewares/nonexistent/connect", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestHandleConnectMiddleware_Success(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewMiddlewareService(db, zap.NewNop())
	svc.Create(&model.MiddlewareConfig{ID: "mw-1", Name: "Test", Type: "mqtt"})

	// nil manager: will return 503 since manager is nil
	app.Post("/api/middlewares/:id/connect", handleConnectMiddleware(svc, nil))

	req := httptest.NewRequest("POST", "/api/middlewares/mw-1/connect", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}

// ─── handleNodeDiscovery 测试 (Stage 2) ────────────────────────────────

func TestHandleNodeDiscovery_NilManager(t *testing.T) {
	app := newTestApp()
	app.Post("/api/edgeCore/discover", handleNodeDiscovery(nil, nil))

	req := httptest.NewRequest("POST", "/api/edgeCore/discover", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}

func TestHandleNodeDiscoveryTo_NilManager(t *testing.T) {
	app := newTestApp()
	app.Post("/api/edgeCore/discover/:middlewareId", handleNodeDiscoveryTo(nil))

	req := httptest.NewRequest("POST", "/api/edgeCore/discover/mw-1", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}

// Note: "not connected" behavior (manager exists but middleware ID not in clients)
// is covered by TestManager_PublishNodeDiscoveryTo_NotConnected in manager_test.go
// which has access to unexported fields within the messaging package.

// ─── handleSendCommand 测试 ────────────────────────────────

func TestHandleSendCommand_NilManager(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	controlSvc := services.NewControlService(db, zap.NewNop())
	app.Post("/api/nodes/:nodeId/devices/:deviceId/commands", handleSendCommand(controlSvc, nil))

	payload := `{"point_id":"temp","value":25.5}`
	req := httptest.NewRequest("POST", "/api/nodes/node1/devices/dev1/commands", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	// Should still return 200 - command is created even without manager
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "0", body["code"])
	data := body["data"].(map[string]interface{})
	cmd := data["command"].(map[string]interface{})
	assert.Equal(t, "temp", cmd["point_id"])
	assert.NotEmpty(t, cmd["id"])
}

func TestHandleSendCommand_InvalidBody(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	controlSvc := services.NewControlService(db, zap.NewNop())
	app.Post("/api/nodes/:nodeId/devices/:deviceId/commands", handleSendCommand(controlSvc, nil))

	req := httptest.NewRequest("POST", "/api/nodes/n1/devices/d1/commands", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// ─── handleListNodes 测试 ────────────────────────────────

func TestHandleListNodes_Empty(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewRegistryService(db)
	app.Get("/api/nodes", handleListNodes(svc))

	req := httptest.NewRequest("GET", "/api/nodes", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "0", body["code"])
}

func TestHandleGetNode_NotFound(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewRegistryService(db)
	app.Get("/api/nodes/:nodeId", handleGetNode(svc))

	req := httptest.NewRequest("GET", "/api/nodes/ghost", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

// ─── handleDashboardStats 测试 ────────────────────────────────

func TestHandleDashboardStats_Success(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	registrySvc := services.NewRegistryService(db)
	dataSvc := &services.DataService{DeviceSvc: services.NewDeviceService(db), PointService: services.NewPointService(db)}
	alertSvc := services.NewAlertService(db)
	app.Get("/api/dashboard/stats", handleDashboardStats(registrySvc, dataSvc, alertSvc))

	req := httptest.NewRequest("GET", "/api/dashboard/stats", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "0", body["code"])
	data := body["data"].(map[string]interface{})
	assert.Contains(t, data, "total_nodes")
	assert.Contains(t, data, "online_nodes")
	assert.Contains(t, data, "total_devices")
	assert.Contains(t, data, "today_alerts")
}

// ─── handleListAlerts 测试 ────────────────────────────────

func TestHandleListAlerts_Empty(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewAlertService(db)
	app.Get("/api/alerts", handleListAlerts(svc))

	req := httptest.NewRequest("GET", "/api/alerts", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "0", body["code"])
}

func TestHandleListAlerts_WithStatusFilter(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewAlertService(db)
	app.Get("/api/alerts", handleListAlerts(svc))

	req := httptest.NewRequest("GET", "/api/alerts?status=active", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// ─── handleAcknowledgeAlert 测试 ────────────────────────────────

func TestHandleAcknowledgeAlert_NotFound(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewAlertService(db)
	app.Post("/api/alerts/:id/acknowledge", handleAcknowledgeAlert(svc))

	req := httptest.NewRequest("POST", "/api/alerts/ghost/acknowledge", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// ─── handleListDevices 测试 ────────────────────────────────

func TestHandleListDevices_Empty(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	dataSvc := &services.DataService{DeviceSvc: services.NewDeviceService(db), PointService: services.NewPointService(db)}
	app.Get("/api/nodes/:nodeId/devices", handleListDevices(dataSvc))

	req := httptest.NewRequest("GET", "/api/nodes/node1/devices", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "0", body["code"])
	data := body["data"].(map[string]interface{})
	assert.Empty(t, data["devices"])
}

func TestHandleReconcileDevices_PrunesStale(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	deviceSvc := services.NewDeviceService(db)
	registrySvc := services.NewRegistryService(db)
	require.NoError(t, deviceSvc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{DeviceID: "stale"}))
	require.NoError(t, deviceSvc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{DeviceID: "keep"}))

	dataSvc := &services.DataService{DeviceSvc: deviceSvc, PointService: services.NewPointService(db)}
	app.Post("/api/nodes/:nodeId/devices/reconcile", handleReconcileDevices(dataSvc, registrySvc))

	payload := `{"devices":[{"device_id":"keep","device_name":"Keep"},{"device_id":"fresh","device_name":"Fresh"}]}`
	req := httptest.NewRequest("POST", "/api/nodes/n1/devices/reconcile", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["upserted"])
	assert.Equal(t, float64(1), data["removed"])
	assert.Equal(t, float64(2), data["total"])

	_, err = deviceSvc.GetDevice("n1", "stale")
	assert.Error(t, err)
	node, err := registrySvc.GetNode("n1")
	require.NoError(t, err)
	assert.Equal(t, "online", node.Status)
}

// ─── handleListPoints 测试 ────────────────────────────────

func TestHandleListPoints_Empty(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	dataSvc := &services.DataService{DeviceSvc: services.NewDeviceService(db), PointService: services.NewPointService(db)}
	app.Get("/api/nodes/:nodeId/devices/:deviceId/points", handleListPoints(dataSvc))

	req := httptest.NewRequest("GET", "/api/nodes/node1/devices/dev1/points", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// ─── handleGetSnapshot 测试 ────────────────────────────────

func TestHandleGetSnapshot_Empty(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	dataSvc := &services.DataService{DeviceSvc: services.NewDeviceService(db), PointService: services.NewPointService(db)}
	app.Get("/api/nodes/:nodeId/devices/:deviceId/snapshot", handleGetSnapshot(dataSvc))

	req := httptest.NewRequest("GET", "/api/nodes/node1/devices/dev1/snapshot", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "0", body["code"])
	data := body["data"].(map[string]interface{})
	assert.Nil(t, data["snapshot"])
}

// ─── handleListCommands 测试 ────────────────────────────────

func TestHandleListCommands_Empty(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	controlSvc := services.NewControlService(db, zap.NewNop())
	app.Get("/api/nodes/:nodeId/devices/:deviceId/commands", handleListCommands(controlSvc))

	req := httptest.NewRequest("GET", "/api/nodes/n1/devices/d1/commands", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// ─── handleGetCommand 测试 ────────────────────────────────

func TestHandleGetCommand_NotFound(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	controlSvc := services.NewControlService(db, zap.NewNop())
	app.Get("/api/nodes/:nodeId/devices/:deviceId/commands/:cmdId", handleGetCommand(controlSvc))

	req := httptest.NewRequest("GET", "/api/nodes/n1/devices/d1/commands/ghost", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

// ─── JWT 鉴权中间件测试 ────────────────────────────────

func TestJWTAuth_NoToken(t *testing.T) {
	app := newTestApp()
	app.Use(JWTAuth())
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	app := newTestApp()
	app.Use(JWTAuth())
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestJWTAuth_ValidToken(t *testing.T) {
	app := newTestApp()
	app.Use(JWTAuth())
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken())
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// ─── handleUpdateMiddleware / handleDeleteMiddleware ────────────────────────────────

func TestHandleUpdateMiddleware_NotFound(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewMiddlewareService(db, zap.NewNop())
	app.Put("/api/middlewares/:id", handleUpdateMiddleware(svc))

	payload := `{"name":"updated"}`
	req := httptest.NewRequest("PUT", "/api/middlewares/ghost", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestHandleDeleteMiddleware_Success(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewMiddlewareService(db, zap.NewNop())
	svc.Create(&model.MiddlewareConfig{ID: "to-delete", Name: "delete me", Type: "mqtt"})
	app.Delete("/api/middlewares/:id", handleDeleteMiddleware(svc))

	req := httptest.NewRequest("DELETE", "/api/middlewares/to-delete", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// ─── handleGetMiddlewareStatus 测试 ────────────────────────────────

func TestHandleGetMiddlewareStatus_NotFound(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewMiddlewareService(db, zap.NewNop())
	app.Get("/api/middlewares/:id/status", handleGetMiddlewareStatus(svc))

	req := httptest.NewRequest("GET", "/api/middlewares/ghost/status", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestHandleGetMiddlewareStatus_Success(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewMiddlewareService(db, zap.NewNop())
	svc.Create(&model.MiddlewareConfig{
		ID:     "mw-status",
		Name:   "MQTT Broker",
		Type:   "mqtt",
		Status: "connected",
		Host:   "localhost",
		Port:   1883,
	})
	app.Get("/api/middlewares/:id/status", handleGetMiddlewareStatus(svc))

	req := httptest.NewRequest("GET", "/api/middlewares/mw-status/status", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "connected", data["status"])
	assert.Equal(t, "MQTT Broker", data["name"])
}

// ─── handleDisconnectMiddleware 测试 ────────────────────────────────

func TestHandleDisconnectMiddleware_Success(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := services.NewMiddlewareService(db, zap.NewNop())
	svc.Create(&model.MiddlewareConfig{ID: "mw-disc", Name: "Test", Type: "mqtt"})

	// nil manager - just update status in DB
	app.Post("/api/middlewares/:id/disconnect", handleDisconnectMiddleware(svc, nil))

	req := httptest.NewRequest("POST", "/api/middlewares/mw-disc/disconnect", nil)
	setAuthHeader(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Verify status updated in DB
	got, _ := svc.Get("mw-disc")
	assert.Equal(t, "disconnected", got.Status)
}
// ─── handleDeleteNode 联动删除 Agent 测试 ────────────────

func TestHandleDeleteNode_RemovesLinkedAgent(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	regSvc := services.NewRegistryService(db)
	require.NoError(t, regSvc.EnsureNodeOnline("linked-node", "linked", "ean"))

	// 用最小 Bus（仅 Discovery 字段）验证节点删除联动清除 Agent
	dc := ean.NewDiscoveryCenter(ean.DiscoveryConfig{}, zap.NewNop())
	bus := &ean.Bus{Discovery: dc}
	agentPayload := []byte(`{
		"header":{"message_id":"m1","timestamp":1,"source":"linked-node","message_type":"agent_descriptor","version":"2.0"},
		"body":{"agent":{"id":"linked-node","kind":"device","version":"2.0.0","status":"online","transport":"mqtt","heartbeat_interval_sec":60}}
	}`)
	dc.HandleAgentOnline(ean.TopicDiscoveryAgent, agentPayload, "mqtt")
	_, ok := dc.GetAgent("linked-node")
	require.True(t, ok)

	app.Delete("/api/nodes/:nodeId", handleDeleteNode(regSvc, bus))

	req := httptest.NewRequest("DELETE", "/api/nodes/linked-node", nil)
	setAuthHeader(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// 节点已删除
	_, err = regSvc.GetNode("linked-node")
	assert.Error(t, err)
	// 对应 Agent 也已删除（一致性）
	_, ok = dc.GetAgent("linked-node")
	assert.False(t, ok)
}

func TestHandleDeleteNode_WithoutEANBusStillWorks(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	regSvc := services.NewRegistryService(db)
	require.NoError(t, regSvc.EnsureNodeOnline("plain-node", "plain", "ean"))

	// nil eanBus 不应影响节点删除
	app.Delete("/api/nodes/:nodeId", handleDeleteNode(regSvc, nil))

	req := httptest.NewRequest("DELETE", "/api/nodes/plain-node", nil)
	setAuthHeader(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	_, err = regSvc.GetNode("plain-node")
	assert.Error(t, err)
}

func TestHandleEANDeleteAgent_Success(t *testing.T) {
	app := newTestApp()
	dc := ean.NewDiscoveryCenter(ean.DiscoveryConfig{}, zap.NewNop())
	bus := &ean.Bus{Discovery: dc}
	agentPayload := []byte(`{
		"header":{"message_id":"m1","timestamp":1,"source":"stale","message_type":"agent_descriptor","version":"2.0"},
		"body":{"agent":{"id":"stale","kind":"device","version":"2.0.0","status":"offline","transport":"mqtt","heartbeat_interval_sec":60}}
	}`)
	dc.HandleAgentOnline(ean.TopicDiscoveryAgent, agentPayload, "mqtt")
	_, ok := dc.GetAgent("stale")
	require.True(t, ok)

	app.Delete("/api/ean/agents/:id", handleEANDeleteAgent(bus))

	req := httptest.NewRequest("DELETE", "/api/ean/agents/stale", nil)
	setAuthHeader(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	_, ok = dc.GetAgent("stale")
	assert.False(t, ok)
}

func TestHandleEANDeleteAgent_NotFound(t *testing.T) {
	app := newTestApp()
	bus := &ean.Bus{Discovery: ean.NewDiscoveryCenter(ean.DiscoveryConfig{}, zap.NewNop())}
	app.Delete("/api/ean/agents/:id", handleEANDeleteAgent(bus))

	req := httptest.NewRequest("DELETE", "/api/ean/agents/nope", nil)
	setAuthHeader(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestHandleEANDeleteAgent_DisabledBus(t *testing.T) {
	app := newTestApp()
	app.Delete("/api/ean/agents/:id", handleEANDeleteAgent(nil))

	req := httptest.NewRequest("DELETE", "/api/ean/agents/nope", nil)
	setAuthHeader(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}