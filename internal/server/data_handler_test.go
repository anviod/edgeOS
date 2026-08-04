package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anviod/edgeOS/internal/storage"
)

func newTestDataStore(t *testing.T) (*storage.Storage, func()) {
	t.Helper()
	store, err := storage.NewStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	return store, func() { store.Close() }
}

func decodeDataResp(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return body
}

// TestHandleDataStats 验证数据库概览返回双库 bucket 统计。
func TestHandleDataStats(t *testing.T) {
	app := newTestApp()
	store, cleanup := newTestDataStore(t)
	defer cleanup()

	// 写入配置与运行时数据
	cs, err := storage.NewConfigStore(store.GetConfigDB())
	require.NoError(t, err)
	require.NoError(t, cs.SaveNodeConfig(storage.NodeConfigData{NodeID: "node-001", NodeType: "primary", Listen: ":8000"}))
	require.NoError(t, store.SaveData("edgex_nodes", "n1", map[string]string{"id": "n1"}))

	app.Get("/api/data/stats", handleDataStats(store))

	req := httptest.NewRequest("GET", "/api/data/stats", nil)
	setAuthHeader(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body := decodeDataResp(t, resp)
	assert.Equal(t, "0", body["code"])
	data := body["data"].(map[string]interface{})
	assert.NotNil(t, data["config_db"])
	assert.NotNil(t, data["runtime_db"])
	assert.True(t, data["total_size"].(float64) > 0)

	buckets := data["buckets"].([]interface{})
	assert.NotEmpty(t, buckets)
	// 配置 bucket 标记为 config 且不可清理
	var configSeen bool
	for _, b := range buckets {
		bk := b.(map[string]interface{})
		if bk["database"] == "config" && bk["name"] == "Node" {
			configSeen = true
			assert.False(t, bk["clearable"].(bool))
		}
	}
	assert.True(t, configSeen, "expected Node config bucket")
}

// TestHandleBackupConfig 验证配置库备份。
func TestHandleBackupConfig(t *testing.T) {
	app := newTestApp()
	store, cleanup := newTestDataStore(t)
	defer cleanup()

	app.Post("/api/data/backup-config", handleBackupConfig(store))

	req := httptest.NewRequest("POST", "/api/data/backup-config?dir=data/backups", nil)
	setAuthHeader(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body := decodeDataResp(t, resp)
	assert.Equal(t, "0", body["code"])
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "success", data["status"])
	assert.NotEmpty(t, data["backup_path"])
}

// TestHandleClearRuntimeBuckets 验证清理运行时 bucket，且配置 bucket 被拒绝。
func TestHandleClearRuntimeBuckets(t *testing.T) {
	app := newTestApp()
	store, cleanup := newTestDataStore(t)
	defer cleanup()

	require.NoError(t, store.SaveData("edgex_alerts", "a1", map[string]string{"msg": "x"}))

	app.Post("/api/data/clear-cache", handleClearRuntimeBuckets(store))

	// 清理运行时 bucket → 成功
	req := httptest.NewRequest("POST", "/api/data/clear-cache",
		bytes.NewReader([]byte(`{"buckets":["edgex_alerts"]}`)))
	req.Header.Set("Content-Type", "application/json")
	setAuthHeader(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]string
	require.Error(t, store.GetData("edgex_alerts", "a1", &result), "runtime data should be cleared")

	// 清理配置 bucket → 403 拒绝
	req = httptest.NewRequest("POST", "/api/data/clear-cache",
		bytes.NewReader([]byte(`{"buckets":["Node"]}`)))
	req.Header.Set("Content-Type", "application/json")
	setAuthHeader(req)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

// TestHandleClearAllRuntime 验证清空运行时库全部 bucket。
func TestHandleClearAllRuntime(t *testing.T) {
	app := newTestApp()
	store, cleanup := newTestDataStore(t)
	defer cleanup()

	require.NoError(t, store.SaveData("edgex_nodes", "n1", map[string]string{"id": "n1"}))
	require.NoError(t, store.SaveData("edgex_alerts", "a1", map[string]string{"msg": "x"}))

	app.Post("/api/data/clear-all-runtime", handleClearAllRuntime(store))

	req := httptest.NewRequest("POST", "/api/data/clear-all-runtime", nil)
	setAuthHeader(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body := decodeDataResp(t, resp)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "success", data["status"])
	cleared := data["cleared"].([]interface{})
	assert.Contains(t, cleared, "edgex_nodes")

	var result map[string]string
	require.Error(t, store.GetData("edgex_nodes", "n1", &result))
}

// TestHandleCompactRuntime 验证压缩运行时库。
func TestHandleCompactRuntime(t *testing.T) {
	app := newTestApp()
	store, cleanup := newTestDataStore(t)
	defer cleanup()

	require.NoError(t, store.SaveData("edgex_nodes", "n1", map[string]string{"id": "n1"}))

	app.Post("/api/data/compact-runtime", handleCompactRuntime(store))

	req := httptest.NewRequest("POST", "/api/data/compact-runtime", nil)
	setAuthHeader(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body := decodeDataResp(t, resp)
	assert.Equal(t, "0", body["code"])
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "success", data["status"])

	// 压缩后运行时数据仍可写
	require.NoError(t, store.SaveData("edgex_nodes", "n2", map[string]string{"id": "n2"}))
}
