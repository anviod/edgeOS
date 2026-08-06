package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestStorage(t *testing.T) (*Storage, func()) {
	t.Helper()
	store, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	return store, func() { store.Close() }
}

// TestGetBucketStats_Classify 验证双库 bucket 统计与分类：config bucket 受保护、runtime bucket 可清理。
func TestGetBucketStats_Classify(t *testing.T) {
	store, cleanup := newTestStorage(t)
	defer cleanup()

	// 写入一条运行时数据到 edgeCore_nodes 与一条配置到 config.db Node
	if err := store.SaveData("edgeCore_nodes", "n1", map[string]string{"id": "n1"}); err != nil {
		t.Fatalf("SaveData failed: %v", err)
	}
	cs, err := NewConfigStore(store.GetConfigDB())
	if err != nil {
		t.Fatalf("NewConfigStore failed: %v", err)
	}
	if err := cs.SaveNodeConfig(NodeConfigData{NodeID: "node-001", NodeType: "primary", Listen: ":8000"}); err != nil {
		t.Fatalf("SaveNodeConfig failed: %v", err)
	}

	stats, totalSize, err := store.GetBucketStats()
	if err != nil {
		t.Fatalf("GetBucketStats failed: %v", err)
	}
	if totalSize <= 0 {
		t.Fatal("expected total size > 0")
	}

	var configSeen, runtimeSeen bool
	for _, st := range stats {
		if st.Database == "config" {
			configSeen = true
			if st.Clearable {
				t.Errorf("config bucket %s must not be clearable", st.Name)
			}
			if st.Category != "config" {
				t.Errorf("config bucket %s category = %q, want config", st.Name, st.Category)
			}
		} else if st.Database == "runtime" {
			runtimeSeen = true
			if st.Name == "edgeCore_nodes" && st.RecordCount != 1 {
				t.Errorf("edgeCore_nodes record_count = %d, want 1", st.RecordCount)
			}
		}
	}
	if !configSeen {
		t.Error("expected config buckets in stats")
	}
	if !runtimeSeen {
		t.Error("expected runtime buckets in stats")
	}
}

// TestClearBucket_RejectsConfigBucket 验证配置库 bucket 受保护。
func TestClearBucket_RejectsConfigBucket(t *testing.T) {
	store, cleanup := newTestStorage(t)
	defer cleanup()

	if err := store.ClearBucket(BucketNode); err == nil {
		t.Fatal("expected ClearBucket on config bucket to fail")
	}
}

// TestClearBucket_ClearsRuntimeBucket 验证运行时 bucket 可清理。
func TestClearBucket_ClearsRuntimeBucket(t *testing.T) {
	store, cleanup := newTestStorage(t)
	defer cleanup()

	if err := store.SaveData("edgeCore_alerts", "a1", map[string]string{"msg": "x"}); err != nil {
		t.Fatalf("SaveData failed: %v", err)
	}
	if err := store.ClearBucket("edgeCore_alerts"); err != nil {
		t.Fatalf("ClearBucket failed: %v", err)
	}

	var result map[string]string
	if err := store.GetData("edgeCore_alerts", "a1", &result); err == nil {
		t.Fatal("expected key to be gone after clear")
	}
}

// TestClearAllRuntimeBuckets_KeepsConfig 验证清空运行时库不影响配置库。
func TestClearAllRuntimeBuckets_KeepsConfig(t *testing.T) {
	store, cleanup := newTestStorage(t)
	defer cleanup()

	if err := store.SaveData("edgeCore_nodes", "n1", map[string]string{"id": "n1"}); err != nil {
		t.Fatalf("SaveData failed: %v", err)
	}
	cs, err := NewConfigStore(store.GetConfigDB())
	if err != nil {
		t.Fatalf("NewConfigStore failed: %v", err)
	}
	if err := cs.SaveNodeConfig(NodeConfigData{NodeID: "node-001", NodeType: "primary", Listen: ":8000"}); err != nil {
		t.Fatalf("SaveNodeConfig failed: %v", err)
	}

	cleared, err := store.ClearAllRuntimeBuckets()
	if err != nil {
		t.Fatalf("ClearAllRuntimeBuckets failed: %v", err)
	}
	if len(cleared) == 0 {
		t.Fatal("expected at least one runtime bucket cleared")
	}

	// 配置库仍可读
	node, err := cs.LoadNodeConfig()
	if err != nil {
		t.Fatalf("LoadNodeConfig after clear failed: %v", err)
	}
	if node.NodeID != "node-001" {
		t.Errorf("config lost after clear-all-runtime: %+v", node)
	}

	// 运行时数据已清空
	var result map[string]string
	if err := store.GetData("edgeCore_nodes", "n1", &result); err == nil {
		t.Fatal("expected runtime data gone after clear-all-runtime")
	}
}

// TestBackupConfigDB 验证配置库备份生成有效文件。
func TestBackupConfigDB(t *testing.T) {
	store, cleanup := newTestStorage(t)
	defer cleanup()

	cs, err := NewConfigStore(store.GetConfigDB())
	if err != nil {
		t.Fatalf("NewConfigStore failed: %v", err)
	}
	if err := cs.SaveNodeConfig(NodeConfigData{NodeID: "node-001", NodeType: "primary", Listen: ":8000"}); err != nil {
		t.Fatalf("SaveNodeConfig failed: %v", err)
	}

	backupDir := filepath.Join(t.TempDir(), "backups")
	info, err := store.BackupConfigDB(backupDir)
	if err != nil {
		t.Fatalf("BackupConfigDB failed: %v", err)
	}
	if _, err := os.Stat(info.BackupPath); err != nil {
		t.Fatalf("backup file does not exist: %v", err)
	}
	if info.FileSizeBytes <= 0 {
		t.Fatal("backup file size must be > 0")
	}
}
