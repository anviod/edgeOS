package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────── HandleCommandResponse ────────────────────

func TestControlHandler_HandleCommandResponse_Success(t *testing.T) {
	svc, cleanup := newTestControlService(t)
	defer cleanup()

	h := NewControlHandler(svc, newTestHub(), newTestLogger())

	// 先创建一条命令
	cmd, err := svc.CreateCommand("n1", "d1", "p1", 100)
	require.NoError(t, err)

	payload := map[string]interface{}{
		"header": map[string]interface{}{
			"message_type": "command_response",
			"request_id":   cmd.ID,
		},
		"body": map[string]interface{}{
			"request_id": cmd.ID,
			"status":     "success",
			"node_id":    "n1",
			"device_id":  "d1",
			"point_id":   "p1",
		},
	}
	h.HandleCommandResponse(nil, buildMsg("edgex/responses/n1/d1", payload))

	// 验证命令状态已更新
	got, err := svc.GetCommand(cmd.ID)
	require.NoError(t, err)
	assert.Equal(t, "success", got.Status)
}

func TestControlHandler_HandleCommandResponse_ErrorStatus(t *testing.T) {
	svc, cleanup := newTestControlService(t)
	defer cleanup()

	h := NewControlHandler(svc, nil, newTestLogger())

	cmd, err := svc.CreateCommand("n1", "d1", "p1", 0)
	require.NoError(t, err)

	payload := map[string]interface{}{
		"header": map[string]interface{}{"request_id": cmd.ID},
		"body": map[string]interface{}{
			"request_id": cmd.ID,
			"status":     "error",
			"error":      "device timeout",
		},
	}
	h.HandleCommandResponse(nil, buildMsg("edgex/responses/n1", payload))

	got, err := svc.GetCommand(cmd.ID)
	require.NoError(t, err)
	assert.Equal(t, "error", got.Status)
}

func TestControlHandler_HandleCommandResponse_RequestIDFromHeader(t *testing.T) {
	svc, cleanup := newTestControlService(t)
	defer cleanup()

	h := NewControlHandler(svc, nil, newTestLogger())

	cmd, err := svc.CreateCommand("n1", "d1", "p1", 0)
	require.NoError(t, err)

	// body.request_id 为空，应从 header.request_id 补充
	payload := map[string]interface{}{
		"header": map[string]interface{}{
			"message_type": "command_response",
			"request_id":   cmd.ID,
		},
		"body": map[string]interface{}{
			"request_id": "",
			"status":     "success",
		},
	}
	h.HandleCommandResponse(nil, buildMsg("edgex/responses/n1", payload))

	got, err := svc.GetCommand(cmd.ID)
	require.NoError(t, err)
	assert.Equal(t, "success", got.Status)
}

func TestControlHandler_HandleCommandResponse_MissingRequestID(t *testing.T) {
	svc, cleanup := newTestControlService(t)
	defer cleanup()

	h := NewControlHandler(svc, nil, newTestLogger())

	// body 和 header 都没有 request_id，应静默返回
	payload := map[string]interface{}{
		"header": map[string]interface{}{},
		"body":   map[string]interface{}{"status": "success"},
	}
	h.HandleCommandResponse(nil, buildMsg("edgex/responses/unknown", payload))
}

func TestControlHandler_HandleCommandResponse_DefaultStatusSuccess(t *testing.T) {
	svc, cleanup := newTestControlService(t)
	defer cleanup()

	h := NewControlHandler(svc, nil, newTestLogger())

	cmd, err := svc.CreateCommand("n1", "d1", "p1", 0)
	require.NoError(t, err)

	// status 字段为空，handler 应默认填 "success"
	payload := map[string]interface{}{
		"header": map[string]interface{}{},
		"body": map[string]interface{}{
			"request_id": cmd.ID,
			"status":     "", // 留空
		},
	}
	h.HandleCommandResponse(nil, buildMsg("edgex/responses/n1", payload))

	got, err := svc.GetCommand(cmd.ID)
	require.NoError(t, err)
	assert.Equal(t, "success", got.Status)
}

func TestControlHandler_HandleCommandResponse_InvalidPayload(t *testing.T) {
	svc, cleanup := newTestControlService(t)
	defer cleanup()

	h := NewControlHandler(svc, nil, newTestLogger())
	msg := &mockMessage{topic: "edgex/responses/n1", payload: []byte(`bad json`)}
	h.HandleCommandResponse(nil, msg)
}

func TestControlHandler_HandleCommandResponse_NilControlService(t *testing.T) {
	// controlSvc 为 nil，不应 panic
	h := NewControlHandler(nil, nil, newTestLogger())
	payload := map[string]interface{}{
		"header": map[string]interface{}{},
		"body":   map[string]interface{}{"request_id": "req-nil", "status": "success"},
	}
	h.HandleCommandResponse(nil, buildMsg("edgex/responses/x", payload))
}

// ──────────────────── publishFn 闭包断言 ────────────────────

// TestControlHandler_PublishFnClosure 验证 publishFn 闭包捕获 topic/payload 的能力
// 这里通过 NodeHandler 的 publishFn 机制来测试闭包逻辑
func TestControlHandler_PublishFnClosure(t *testing.T) {
	type pub struct{ topic string; payload []byte }
	results := make([]pub, 0)
	publishFn := func(topic string, payload []byte) error {
		results = append(results, pub{topic: topic, payload: payload})
		return nil
	}

	svc, cleanup := newTestRegistryService(t)
	defer cleanup()
	h := NewNodeHandler(svc, nil, newTestLogger(), publishFn)

	// 注册节点，触发 publishRegisterResponse
	regPayload := map[string]interface{}{
		"header": map[string]interface{}{},
		"body":   map[string]interface{}{"node_id": "closure-test"},
	}
	h.HandleRegister(nil, buildMsg("edgex/nodes/register", regPayload))

	require.Len(t, results, 1)
	assert.Equal(t, "edgex/nodes/closure-test/response", results[0].topic)

	// 验证 payload 是合法 JSON 且包含 node_id
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(results[0].payload, &resp))
	body, ok := resp["body"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "closure-test", body["node_id"])
}

// ──────────────────── WaitResponse 集成 ────────────────────

func TestControlHandler_WaitResponse_ViaHandleResponse(t *testing.T) {
	svc, cleanup := newTestControlService(t)
	defer cleanup()

	h := NewControlHandler(svc, nil, newTestLogger())

	cmd, err := svc.CreateCommand("n1", "d1", "p1", 99)
	require.NoError(t, err)

	// 异步发送响应
	go func() {
		time.Sleep(50 * time.Millisecond)
		payload := map[string]interface{}{
			"header": map[string]interface{}{},
			"body":   map[string]interface{}{"request_id": cmd.ID, "status": "success"},
		}
		h.HandleCommandResponse(nil, buildMsg("edgex/responses/n1", payload))
	}()

	// WaitResponse 应在 500ms 内返回
	result, err := svc.WaitResponse(cmd.ID, 500*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
}
