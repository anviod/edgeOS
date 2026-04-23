package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/model"
)

// ======================== ControlService ========================

func newTestControlService(t *testing.T) (*ControlService, func()) {
	t.Helper()
	db, cleanup := openTestDB(t)
	return NewControlService(db, zap.NewNop()), cleanup
}

func TestControlService_CreateAndGet(t *testing.T) {
	svc, cleanup := newTestControlService(t)
	defer cleanup()

	cmd, err := svc.CreateCommand("n1", "d1", "temp", 25.0)
	require.NoError(t, err)
	assert.NotEmpty(t, cmd.ID)
	assert.Equal(t, "pending", cmd.Status)
	assert.Equal(t, "n1", cmd.NodeID)
	assert.Equal(t, "d1", cmd.DeviceID)
	assert.Equal(t, "temp", cmd.PointID)

	got, err := svc.GetCommand(cmd.ID)
	require.NoError(t, err)
	assert.Equal(t, cmd.ID, got.ID)
}

func TestControlService_GetCommand_NotFound(t *testing.T) {
	svc, cleanup := newTestControlService(t)
	defer cleanup()

	_, err := svc.GetCommand("ghost-id")
	assert.Error(t, err)
}

func TestControlService_ListCommands(t *testing.T) {
	svc, cleanup := newTestControlService(t)
	defer cleanup()

	// 空时返回空切片
	cmds, err := svc.ListCommands("", "", 0)
	require.NoError(t, err)
	assert.Empty(t, cmds)

	_, err = svc.CreateCommand("n1", "d1", "p1", 1)
	require.NoError(t, err)
	_, err = svc.CreateCommand("n1", "d2", "p2", 2)
	require.NoError(t, err)
	_, err = svc.CreateCommand("n2", "d3", "p3", 3)
	require.NoError(t, err)

	// 全部
	all, err := svc.ListCommands("", "", 0)
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// 按 nodeID 过滤
	n1cmds, err := svc.ListCommands("n1", "", 0)
	require.NoError(t, err)
	assert.Len(t, n1cmds, 2)

	// 按 nodeID + deviceID 过滤
	d1cmds, err := svc.ListCommands("n1", "d1", 0)
	require.NoError(t, err)
	assert.Len(t, d1cmds, 1)
}

func TestControlService_ListCommands_Limit(t *testing.T) {
	svc, cleanup := newTestControlService(t)
	defer cleanup()

	for i := 0; i < 10; i++ {
		_, err := svc.CreateCommand("n1", "d1", "p", i)
		require.NoError(t, err)
	}

	cmds, err := svc.ListCommands("", "", 3)
	require.NoError(t, err)
	assert.Len(t, cmds, 3)
}

func TestControlService_UpdateCommandStatus(t *testing.T) {
	svc, cleanup := newTestControlService(t)
	defer cleanup()

	cmd, err := svc.CreateCommand("n1", "d1", "p", 0)
	require.NoError(t, err)

	require.NoError(t, svc.UpdateCommandStatus(cmd.ID, "success", ""))

	got, err := svc.GetCommand(cmd.ID)
	require.NoError(t, err)
	assert.Equal(t, "success", got.Status)
}

func TestControlService_HandleResponse_FillsChannel(t *testing.T) {
	svc, cleanup := newTestControlService(t)
	defer cleanup()

	cmd, err := svc.CreateCommand("n1", "d1", "p", 42)
	require.NoError(t, err)

	// 注册 pending channel（类型需与 ControlService 内部一致）
	ch := make(chan *model.CommandRecord, 1)
	svc.pending.Store(cmd.ID, ch)

	// HandleResponse 应更新状态并向 channel 推送
	svc.HandleResponse(cmd.ID, "success", "")

	// 验证 channel 已收到命令
	select {
	case received := <-ch:
		assert.Equal(t, "success", received.Status)
	default:
		t.Fatal("channel should have received a command record")
	}

	got, err := svc.GetCommand(cmd.ID)
	require.NoError(t, err)
	assert.Equal(t, "success", got.Status)
}

func TestControlService_WaitResponse_Timeout(t *testing.T) {
	svc, cleanup := newTestControlService(t)
	defer cleanup()

	cmd, err := svc.CreateCommand("n1", "d1", "p", 0)
	require.NoError(t, err)

	start := time.Now()
	result, err := svc.WaitResponse(cmd.ID, 100*time.Millisecond)
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.Greater(t, elapsed, 90*time.Millisecond)

	// 状态应更新为 timeout
	assert.Equal(t, "timeout", result.Status)
}

func TestControlService_WaitResponse_Success(t *testing.T) {
	svc, cleanup := newTestControlService(t)
	defer cleanup()

	cmd, err := svc.CreateCommand("n1", "d1", "p", 0)
	require.NoError(t, err)

	// 异步触发响应
	go func() {
		time.Sleep(20 * time.Millisecond)
		svc.HandleResponse(cmd.ID, "success", "")
	}()

	result, err := svc.WaitResponse(cmd.ID, 2*time.Second)
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
}
