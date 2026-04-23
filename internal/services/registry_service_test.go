package services

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"

	"github.com/anviod/edgeOS/internal/model"
)

func openTestDB(t *testing.T) (*bbolt.DB, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "edgeos_test_*.db")
	require.NoError(t, err)
	f.Close()

	db, err := bbolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	return db, func() {
		db.Close()
		os.Remove(f.Name())
	}
}

// ======================== RegistryService ========================

func TestRegistryService_UpsertAndGet(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewRegistryService(db)

	node := &model.EdgeXNodeInfo{
		NodeID:      "node-1",
		NodeName:    "Test Node",
		Status:      "online",
		AccessToken: "token-abc",
	}
	require.NoError(t, svc.UpsertNode(node))

	got, err := svc.GetNode("node-1")
	require.NoError(t, err)
	assert.Equal(t, "node-1", got.NodeID)
	assert.Equal(t, "Test Node", got.NodeName)
	assert.Equal(t, "token-abc", got.AccessToken)
}

func TestRegistryService_UpsertPreservesToken(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewRegistryService(db)

	// 首次插入，带 token
	n1 := &model.EdgeXNodeInfo{NodeID: "node-1", AccessToken: "original-token"}
	require.NoError(t, svc.UpsertNode(n1))

	// 第二次 upsert 不带 token，应保留原 token
	n2 := &model.EdgeXNodeInfo{NodeID: "node-1", NodeName: "Updated", AccessToken: ""}
	require.NoError(t, svc.UpsertNode(n2))

	got, err := svc.GetNode("node-1")
	require.NoError(t, err)
	assert.Equal(t, "original-token", got.AccessToken)
	assert.Equal(t, "Updated", got.NodeName)
}

func TestRegistryService_UpdateNodeStatus(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewRegistryService(db)

	require.NoError(t, svc.UpsertNode(&model.EdgeXNodeInfo{NodeID: "node-1", Status: "online"}))
	require.NoError(t, svc.UpdateNodeStatus("node-1", "offline"))

	got, err := svc.GetNode("node-1")
	require.NoError(t, err)
	assert.Equal(t, "offline", got.Status)
}

func TestRegistryService_UpdateNodeStatus_NotFound(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewRegistryService(db)
	// bucket 不存在时应返回错误
	err := svc.UpdateNodeStatus("ghost", "online")
	assert.Error(t, err)
}

func TestRegistryService_GetNode_NotFound(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewRegistryService(db)
	_, err := svc.GetNode("nonexistent")
	assert.Error(t, err)
}

func TestRegistryService_ListNodes(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewRegistryService(db)

	// 空列表应返回空切片而非 nil
	nodes, err := svc.ListNodes()
	require.NoError(t, err)
	assert.Empty(t, nodes)

	require.NoError(t, svc.UpsertNode(&model.EdgeXNodeInfo{NodeID: "n1"}))
	require.NoError(t, svc.UpsertNode(&model.EdgeXNodeInfo{NodeID: "n2"}))

	nodes, err = svc.ListNodes()
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
}

func TestRegistryService_DeleteNode(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewRegistryService(db)
	require.NoError(t, svc.UpsertNode(&model.EdgeXNodeInfo{NodeID: "node-del"}))

	require.NoError(t, svc.DeleteNode("node-del"))

	_, err := svc.GetNode("node-del")
	assert.Error(t, err)
}

func TestRegistryService_CountNodes(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewRegistryService(db)

	total, online := svc.CountNodes()
	assert.Equal(t, 0, total)
	assert.Equal(t, 0, online)

	require.NoError(t, svc.UpsertNode(&model.EdgeXNodeInfo{NodeID: "n1", Status: "online"}))
	require.NoError(t, svc.UpsertNode(&model.EdgeXNodeInfo{NodeID: "n2", Status: "offline"}))
	require.NoError(t, svc.UpsertNode(&model.EdgeXNodeInfo{NodeID: "n3", Status: "online"}))

	total, online = svc.CountNodes()
	assert.Equal(t, 3, total)
	assert.Equal(t, 2, online)
}
