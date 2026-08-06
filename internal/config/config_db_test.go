package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/anviod/edgeOS/internal/storage"
)

// TestDefaultInstallInitialization 验证「默认安装初始化」：
// 首次启动（无 data/config.db）时，InitializeDefaultConfig 将默认配置写入 bboltDB，
// HasConfigData 判定为已初始化，可从数据库重新加载出完整默认配置。
func TestDefaultInstallInitialization(t *testing.T) {
	store, err := storage.NewStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer store.Close()

	configStore, err := storage.NewConfigStore(store.GetConfigDB())
	if err != nil {
		t.Fatalf("NewConfigStore failed: %v", err)
	}

	// 初始化前不应有业务数据
	has, err := configStore.HasConfigData()
	if err != nil {
		t.Fatalf("HasConfigData failed: %v", err)
	}
	if has {
		t.Fatal("expected no config data before default initialization")
	}

	if err := InitializeDefaultConfig(store.GetConfigDB()); err != nil {
		t.Fatalf("InitializeDefaultConfig failed: %v", err)
	}

	has, err = configStore.HasConfigData()
	if err != nil {
		t.Fatalf("HasConfigData failed: %v", err)
	}
	if !has {
		t.Fatal("expected config data after default initialization")
	}

	cfg, err := LoadConfigFromDB(store.GetConfigDB())
	if err != nil {
		t.Fatalf("LoadConfigFromDB failed: %v", err)
	}

	def := DefaultConfig()
	if cfg.Node.NodeID != def.Node.NodeID {
		t.Errorf("node_id mismatch: got %q want %q", cfg.Node.NodeID, def.Node.NodeID)
	}
	if cfg.Node.NodeType != def.Node.NodeType {
		t.Errorf("node_type mismatch: got %q want %q", cfg.Node.NodeType, def.Node.NodeType)
	}
	if cfg.Node.Listen != def.Node.Listen {
		t.Errorf("listen mismatch: got %q want %q", cfg.Node.Listen, def.Node.Listen)
	}
	if cfg.User.Username != def.User.Username || cfg.User.Password != def.User.Password {
		t.Errorf("default user mismatch: got %+v", cfg.User)
	}
	if cfg.Security.JWTSecret != def.Security.JWTSecret {
		t.Errorf("jwt_secret mismatch: got %q want %q", cfg.Security.JWTSecret, def.Security.JWTSecret)
	}
	if cfg.EAN.Enabled != def.EAN.Enabled {
		t.Errorf("ean.enabled mismatch: got %v want %v", cfg.EAN.Enabled, def.EAN.Enabled)
	}
	if cfg.EAN.PlannerID != def.EAN.PlannerID {
		t.Errorf("ean.planner_id mismatch: got %q want %q", cfg.EAN.PlannerID, def.EAN.PlannerID)
	}
	if cfg.EAN.MQTT.Broker != def.EAN.MQTT.Broker {
		t.Errorf("ean.mqtt.broker mismatch: got %q want %q", cfg.EAN.MQTT.Broker, def.EAN.MQTT.Broker)
	}
	if cfg.EAN.V1CommandEnabled != def.EAN.V1CommandEnabled {
		t.Errorf("ean.v1_command_enabled mismatch: got %v want %v", cfg.EAN.V1CommandEnabled, def.EAN.V1CommandEnabled)
	}
}

// TestRestartWithoutConfigFile 验证「无配置文件重启」：
// 写入配置到 config.db → 关闭数据库（模拟进程退出）→ 不提供任何配置文件重新打开
// （模拟重启）→ 全部配置（节点/安全/监控/中间件/EAN/用户）从 bboltDB 恢复。
func TestRestartWithoutConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "config.db")

	// ── 第一次启动：默认安装初始化 + 写入自定义配置 ──
	store, err := storage.NewStorage(tempDir)
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	if err := InitializeDefaultConfig(store.GetConfigDB()); err != nil {
		t.Fatalf("InitializeDefaultConfig failed: %v", err)
	}

	cfg := DefaultConfig()
	cfg.Node.NodeID = "node-restart"
	cfg.Node.NodeType = "primary"
	cfg.Node.Listen = ":8001"
	cfg.Security.JWTSecret = "restart-secret-key"
	cfg.Security.TLSEnabled = true
	cfg.Monitoring.Enabled = false
	cfg.Monitoring.Prometheus = ":9091"
	cfg.Middlewares = []MiddlewareMiddlewareConfig{
		{
			ID:                "mw-1",
			Name:              "restart-mqtt",
			Type:              "mqtt",
			Enabled:           true,
			Broker:            "tcp://10.0.0.10:1883",
			ClientID:          "edgeos-restart",
			Username:          "user",
			Password:          "pass",
			QoS:               1,
			Subscriptions:     []string{"edgeCore/nodes/register", "edgeCore/nodes/heartbeat"},
			ConnectTimeout:    10,
			KeepAlive:         30,
			AutoReconnect:     true,
			ReconnectInterval: 5,
		},
	}
	cfg.EAN.Enabled = true
	cfg.EAN.PlannerID = "planner-restart"
	cfg.EAN.MQTT.Broker = "tcp://10.0.0.11:18083"
	cfg.EAN.NATS.Enabled = true
	cfg.EAN.NATS.URL = "nats://10.0.0.12:4222"
	cfg.EAN.NATS.ClientName = "edgeos-ean-restart"
	cfg.EAN.Heartbeat.CheckIntervalSec = 10
	cfg.EAN.Heartbeat.TimeoutMultiplier = 4
	cfg.EAN.V1CommandEnabled = false

	if err := SaveConfigToDB(store.GetConfigDB(), cfg); err != nil {
		t.Fatalf("SaveConfigToDB failed: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("store.Close failed: %v", err)
	}

	// ── 第二次启动：无任何配置文件，仅从 config.db 恢复 ──
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("config.db should exist after first run: %v", err)
	}

	store2, err := storage.NewStorage(tempDir)
	if err != nil {
		t.Fatalf("reopen storage failed: %v", err)
	}
	defer store2.Close()

	configStore, err := storage.NewConfigStore(store2.GetConfigDB())
	if err != nil {
		t.Fatalf("NewConfigStore failed: %v", err)
	}
	has, err := configStore.HasConfigData()
	if err != nil {
		t.Fatalf("HasConfigData failed: %v", err)
	}
	if !has {
		t.Fatal("expected config data after restart (restart must recover from bboltDB)")
	}

	reloaded, err := LoadConfigFromDB(store2.GetConfigDB())
	if err != nil {
		t.Fatalf("LoadConfigFromDB after restart failed: %v", err)
	}

	if reloaded.Node.NodeID != "node-restart" {
		t.Errorf("node_id not recovered: got %q", reloaded.Node.NodeID)
	}
	if reloaded.Node.Listen != ":8001" {
		t.Errorf("listen not recovered: got %q", reloaded.Node.Listen)
	}
	if reloaded.Security.JWTSecret != "restart-secret-key" || !reloaded.Security.TLSEnabled {
		t.Errorf("security config not recovered: %+v", reloaded.Security)
	}
	if reloaded.Monitoring.Enabled != false || reloaded.Monitoring.Prometheus != ":9091" {
		t.Errorf("monitoring config not recovered: %+v", reloaded.Monitoring)
	}
	if len(reloaded.Middlewares) != 1 {
		t.Fatalf("expected 1 middleware after restart, got %d", len(reloaded.Middlewares))
	}
	mw := reloaded.Middlewares[0]
	if mw.ID != "mw-1" || mw.Broker != "tcp://10.0.0.10:1883" || mw.Username != "user" || mw.Password != "pass" {
		t.Errorf("middleware not recovered: %+v", mw)
	}
	if !reflect.DeepEqual(mw.Subscriptions, []string{"edgeCore/nodes/register", "edgeCore/nodes/heartbeat"}) {
		t.Errorf("middleware subscriptions not recovered: %v", mw.Subscriptions)
	}
	if !reloaded.EAN.Enabled || reloaded.EAN.PlannerID != "planner-restart" {
		t.Errorf("EAN enabled/planner not recovered: %+v", reloaded.EAN)
	}
	if reloaded.EAN.MQTT.Broker != "tcp://10.0.0.11:18083" {
		t.Errorf("EAN MQTT broker not recovered: %q", reloaded.EAN.MQTT.Broker)
	}
	if !reloaded.EAN.NATS.Enabled || reloaded.EAN.NATS.URL != "nats://10.0.0.12:4222" {
		t.Errorf("EAN NATS config not recovered: %+v", reloaded.EAN.NATS)
	}
	if reloaded.EAN.Heartbeat.CheckIntervalSec != 10 || reloaded.EAN.Heartbeat.TimeoutMultiplier != 4 {
		t.Errorf("EAN heartbeat config not recovered: %+v", reloaded.EAN.Heartbeat)
	}
	if reloaded.EAN.V1CommandEnabled != false {
		t.Errorf("ean.v1_command_enabled not recovered: %v", reloaded.EAN.V1CommandEnabled)
	}
	if reloaded.User.Username != "admin" {
		t.Errorf("default user not recovered: %+v", reloaded.User)
	}
}

// TestConvertEANFromDB_HeartbeatDefaultFallback 验证老版本 config.db（无 max_offline_retention_sec）
// 加载时回退默认离线保留期 600s，确保「edgeCore 关闭 EAN → Agent 自动清除」在旧库上升级后也生效。
func TestConvertEANFromDB_HeartbeatDefaultFallback(t *testing.T) {
	// 模拟老库数据：max_offline_retention_sec 缺省（0），timeout/check_interval 也缺省（0）
	cfg := convertEANFromDB(storage.EANConfigData{
		Enabled: true,
		Heartbeat: storage.EANHeartbeatData{
			TimeoutMultiplier:      0,
			CheckIntervalSec:       0,
			MaxOfflineRetentionSec: 0,
		},
	})
	if cfg.Heartbeat.TimeoutMultiplier != 3 {
		t.Errorf("timeout_multiplier fallback failed: got %d", cfg.Heartbeat.TimeoutMultiplier)
	}
	if cfg.Heartbeat.CheckIntervalSec != 5 {
		t.Errorf("check_interval_sec fallback failed: got %d", cfg.Heartbeat.CheckIntervalSec)
	}
	if cfg.Heartbeat.MaxOfflineRetentionSec != 600 {
		t.Errorf("max_offline_retention_sec fallback failed: got %d", cfg.Heartbeat.MaxOfflineRetentionSec)
	}
}
