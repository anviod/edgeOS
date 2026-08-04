package server

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anviod/edgeOS/internal/config"
	"github.com/anviod/edgeOS/internal/storage"
)

// ─── handleInstallStatus 测试 ────────────────────────────────

func TestHandleInstallStatus_NotInstalled(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	// 创建 config store（仅初始化 bucket，无业务数据）
	_, err := storage.NewConfigStore(db)
	require.NoError(t, err)

	app.Get("/api/install/status", handleInstallStatus(db))

	req := httptest.NewRequest("GET", "/api/install/status", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body struct {
		Code string              `json:"code"`
		Data InstallStatusResponse `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "0", body.Code)
	assert.False(t, body.Data.IsInstalled, "no config data => is_installed must be false")
}

func TestHandleInstallStatus_Installed(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	// 已初始化配置（模拟安装完成）| Install config first
	cs, err := storage.NewConfigStore(db)
	require.NoError(t, err)
	require.NoError(t, cs.SaveNodeConfig(storage.NodeConfigData{
		NodeID:   "node-001",
		NodeType: "primary",
		Listen:   ":8000",
	}))

	app.Get("/api/install/status", handleInstallStatus(db))

	req := httptest.NewRequest("GET", "/api/install/status", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body struct {
		Code string              `json:"code"`
		Data InstallStatusResponse `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "0", body.Code)
	assert.True(t, body.Data.IsInstalled, "config exists => is_installed must be true")
}

// ─── handleInstall 测试 ────────────────────────────────

func TestHandleInstall_Success(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	restartCalled := false
	app.Post("/api/install", handleInstall(db, func() { restartCalled = true }))

	payload := `{
		"node": { "node_id": "node-001", "node_type": "primary", "listen": ":8000" },
		"user": { "username": "admin", "password": "Passw0rd!", "role": "admin" },
		"ean": { "enabled": false, "planner_id": "edgeos-planner" },
		"middlewares": []
	}`

	req := httptest.NewRequest("POST", "/api/install", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body struct {
		Code string `json:"code"`
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "0", body.Code)
	assert.Equal(t, "installed", body.Data.Status)
	assert.True(t, restartCalled, "restart callback must be triggered after install")

	// 安装后 config.db 应有业务数据（后续重启不再进入安装引导）
	cs, err := storage.NewConfigStore(db)
	require.NoError(t, err)
	has, err := cs.HasConfigData()
	require.NoError(t, err)
	assert.True(t, has, "config must be persisted after install")

	// 验证已持久化的配置
	cfg, err := config.LoadConfigFromDB(db)
	require.NoError(t, err)
	assert.Equal(t, "node-001", cfg.Node.NodeID)
	assert.Equal(t, "primary", cfg.Node.NodeType)
	assert.Equal(t, ":8000", cfg.Node.Listen)
	assert.Equal(t, "admin", cfg.User.Username)
	assert.Equal(t, "Passw0rd!", cfg.User.Password)
	assert.Equal(t, "admin", cfg.User.Role)
	assert.False(t, cfg.EAN.Enabled)
	assert.Equal(t, "edgeos-planner", cfg.EAN.PlannerID)
}

func TestHandleInstall_RejectWhenAlreadyInstalled(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	// 先完成一次安装 | Complete an install first
	cs, err := storage.NewConfigStore(db)
	require.NoError(t, err)
	require.NoError(t, cs.SaveNodeConfig(storage.NodeConfigData{
		NodeID:   "node-001",
		NodeType: "primary",
		Listen:   ":8000",
	}))

	app.Post("/api/install", handleInstall(db, func() {}))

	payload := `{
		"node": { "node_id": "node-002", "node_type": "secondary", "primary_node_id": "node-primary", "listen": ":8001" },
		"user": { "username": "admin", "password": "Passw0rd!", "role": "admin" }
	}`
	req := httptest.NewRequest("POST", "/api/install", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode, "repeated install must be rejected")

	// 已安装配置不应被重复安装覆盖
	cfg, err := config.LoadConfigFromDB(db)
	require.NoError(t, err)
	assert.Equal(t, "node-001", cfg.Node.NodeID)
}

func TestHandleInstall_ValidationErrors(t *testing.T) {
	app := newTestApp()
	db, cleanup := openTestDB(t)
	defer cleanup()

	app.Post("/api/install", handleInstall(db, func() {}))

	// 缺少 node_id
	req := httptest.NewRequest("POST", "/api/install", bytes.NewReader([]byte(`{
		"user": { "username": "admin", "password": "Passw0rd!" }
	}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	// 密码过短
	req = httptest.NewRequest("POST", "/api/install", bytes.NewReader([]byte(`{
		"node": { "node_id": "n1", "node_type": "primary" },
		"user": { "username": "admin", "password": "short" }
	}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	// 无效节点类型
	req = httptest.NewRequest("POST", "/api/install", bytes.NewReader([]byte(`{
		"node": { "node_id": "n1", "node_type": "unknown" },
		"user": { "username": "admin", "password": "Passw0rd!" }
	}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	// secondary 必须配置 primary_node_id
	req = httptest.NewRequest("POST", "/api/install", bytes.NewReader([]byte(`{
		"node": { "node_id": "n2", "node_type": "secondary" },
		"user": { "username": "admin", "password": "Passw0rd!" }
	}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
