package ean

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestOrchestrator(t *testing.T, publishFn PublishFunc) *InvokeOrchestrator {
	t.Helper()
	cfg := InvokeConfig{
		SourceID:  "test-planner",
		PublishFn: publishFn,
	}
	return NewInvokeOrchestrator(cfg, zap.NewNop())
}

// ---------- TestInvokeOrchestratorReply ----------

func TestInvokeOrchestratorReply(t *testing.T) {
	// 捕获 publish 调用
	var captured struct {
		Topic   string
		Payload []byte
	}
	publishCh := make(chan struct{}, 1)
	publishFn := func(topic string, payload []byte) error {
		captured.Topic = topic
		captured.Payload = payload
		publishCh <- struct{}{}
		return nil
	}

	orch := newTestOrchestrator(t, publishFn)

	// 在另一个 goroutine 中调用 Invoke 并等待回复
	resultCh := make(chan *InvokeCall, 1)
	go func() {
		call := orch.Invoke(context.Background(), "agent-001", "modbus.read_points",
			map[string]interface{}{"device": "PLC-1"}, 5*time.Second)
		resultCh <- call
	}()

	// 等待 publish
	<-publishCh

	// 验证 publish 的 topic
	assert.Equal(t, "$edgeos/invoke/agent-001", captured.Topic)

	// 解析 payload 获取 invoke_id
	var msg Message
	require.NoError(t, json.Unmarshal(captured.Payload, &msg))
	var req InvokeRequest
	require.NoError(t, json.Unmarshal(msg.Body, &req))
	invokeID := req.InvokeID
	assert.NotEmpty(t, invokeID)
	assert.Equal(t, "agent-001", req.Target)
	assert.Equal(t, "modbus.read_points", req.Capability)

	// 构造 Reply 消息
	resp := InvokeResponse{
		InvokeID: invokeID,
		Status:   "success",
		Result: InvokeResult{
			Success: true,
			Values:  map[string]interface{}{"temperature": float64(25.5)},
		},
	}
	respBody, _ := json.Marshal(resp)
	replyMsg := Message{
		Header: MessageHeader{
			MessageID:   "reply-001",
			Timestamp:   time.Now().UnixMilli(),
			Source:      "agent-001",
			MessageType:  "invoke_reply",
			Version:      "2.0",
			CorrelationID: invokeID,
		},
		Body: respBody,
	}
	replyPayload, _ := json.Marshal(replyMsg)

	// 模拟收到 reply
	orch.HandleReply("$edgeos/reply/test-planner", replyPayload, "mqtt")

	// 验证 Invoke 返回了正确的 response
	select {
	case call := <-resultCh:
		require.NoError(t, call.Error)
		require.NotNil(t, call.Response)
		assert.Equal(t, "success", call.Response.Status)
		assert.True(t, call.Response.Result.Success)
		assert.Equal(t, float64(25.5), call.Response.Result.Values["temperature"])
	case <-time.After(time.Second):
		t.Fatal("invoke did not complete within timeout")
	}
}

// ---------- TestInvokeTimeout ----------

func TestInvokeTimeout(t *testing.T) {
	publishFn := func(topic string, payload []byte) error {
		return nil // 发布成功但不回复
	}

	orch := newTestOrchestrator(t, publishFn)

	start := time.Now()
	call := orch.Invoke(context.Background(), "agent-001", "modbus.read_points", nil, 50*time.Millisecond)
	elapsed := time.Since(start)

	require.Error(t, call.Error)
	assert.Contains(t, call.Error.Error(), "timed out")
	assert.Nil(t, call.Response)
	assert.True(t, elapsed < 500*time.Millisecond, "timeout took too long: %v", elapsed)

	// pending 应被清理
	assert.Equal(t, 0, orch.PendingCount())
}

// ---------- TestInvokeValidation ----------

func TestInvokeValidation_EmptyPublishFn(t *testing.T) {
	orch := NewInvokeOrchestrator(InvokeConfig{
		SourceID:  "test",
		PublishFn: nil,
	}, zap.NewNop())

	// nil publishFn → Invoke 内部调用时会 panic 或报错
	// 因为 io.publishFn(topic, payload) 调用 nil 函数会 panic
	assert.Panics(t, func() {
		orch.Invoke(context.Background(), "agent-001", "cap", nil, time.Second)
	})
}

func TestInvokeValidation_PublishError(t *testing.T) {
	publishFn := func(topic string, payload []byte) error {
		return fmt.Errorf("network error")
	}

	orch := newTestOrchestrator(t, publishFn)

	call := orch.Invoke(context.Background(), "agent-001", "cap", nil, time.Second)
	require.Error(t, call.Error)
	assert.Contains(t, call.Error.Error(), "publish invoke failed")
	assert.Equal(t, 0, orch.PendingCount())
}

// ---------- TestHandleReply ----------

func TestHandleReply_NormalReply(t *testing.T) {
	publishFn := func(topic string, payload []byte) error { return nil }
	orch := newTestOrchestrator(t, publishFn)

	// 手动注册一个 pending invoke
	invokeID := "test-invoke-001"
	respCh := make(chan *InvokeResponse, 1)
	orch.mu.Lock()
	orch.pending[invokeID] = &pendingInvoke{
		invokeID:   invokeID,
		responseCh: respCh,
		cancel:     func() {},
	}
	orch.mu.Unlock()

	// 构造 reply 消息
	resp := InvokeResponse{
		InvokeID: invokeID,
		Status:   "success",
		Result:   InvokeResult{Success: true},
	}
	respBody, _ := json.Marshal(resp)
	replyMsg := Message{
		Header: MessageHeader{
			MessageID:      "reply-001",
			Source:         "agent-001",
			CorrelationID:  invokeID,
			Version:        "2.0",
			MessageType:    "invoke_reply",
		},
		Body: respBody,
	}
	replyPayload, _ := json.Marshal(replyMsg)

	orch.HandleReply("$edgeos/reply/test-planner", replyPayload, "mqtt")

	select {
	case r := <-respCh:
		assert.Equal(t, "success", r.Status)
		assert.True(t, r.Result.Success)
	case <-time.After(time.Second):
		t.Fatal("reply not delivered to channel")
	}
}

func TestHandleReply_MissingInvokeID(t *testing.T) {
	publishFn := func(topic string, payload []byte) error { return nil }
	orch := newTestOrchestrator(t, publishFn)

	// Reply 消息中既没有 invoke_id 也没有 correlation_id
	resp := InvokeResponse{Status: "success"}
	respBody, _ := json.Marshal(resp)
	replyMsg := Message{
		Header: MessageHeader{
			MessageID: "reply-001",
			Version:   "2.0",
		},
		Body: respBody,
	}
	replyPayload, _ := json.Marshal(replyMsg)

	// 不应 panic
	orch.HandleReply("$edgeos/reply/test-planner", replyPayload, "mqtt")
	assert.Equal(t, 0, orch.PendingCount())
}

func TestHandleReply_UnknownInvoke(t *testing.T) {
	publishFn := func(topic string, payload []byte) error { return nil }
	orch := newTestOrchestrator(t, publishFn)

	resp := InvokeResponse{InvokeID: "nonexistent-invoke", Status: "success"}
	respBody, _ := json.Marshal(resp)
	replyMsg := Message{
		Header: MessageHeader{Version: "2.0"},
		Body:    respBody,
	}
	replyPayload, _ := json.Marshal(replyMsg)

	// 不应 panic，仅 debug log
	orch.HandleReply("$edgeos/reply/test-planner", replyPayload, "mqtt")
}

func TestHandleReply_InvalidJSON(t *testing.T) {
	publishFn := func(topic string, payload []byte) error { return nil }
	orch := newTestOrchestrator(t, publishFn)

	// 无效 JSON，不应 panic
	orch.HandleReply("$edgeos/reply/test-planner", []byte("invalid"), "mqtt")
}

func TestInvokeOrchestrator_SourceID(t *testing.T) {
	orch := newTestOrchestrator(t, func(string, []byte) error { return nil })
	assert.Equal(t, "test-planner", orch.SourceID())
	assert.Equal(t, "$edgeos/reply/test-planner", orch.ReplyTopic())
}

func TestInvokeOrchestrator_DefaultSourceID(t *testing.T) {
	orch := NewInvokeOrchestrator(InvokeConfig{
		PublishFn: func(string, []byte) error { return nil },
	}, zap.NewNop())
	assert.Equal(t, "edgeos-planner", orch.SourceID())
}
