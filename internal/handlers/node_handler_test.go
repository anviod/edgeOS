package handlers

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestNodeHandler(t *testing.T) (*NodeHandler, func()) {
	t.Helper()
	svc, cleanup := newTestRegistryService(t)
	hub := newTestHub()
	logger := newTestLogger()
	publishCh := make(chan struct{ topic string; payload []byte }, 4)
	h := NewNodeHandler(svc, hub, logger, capturePublish(publishCh))
	return h, cleanup
}

// buildMsg 构建测试消息
func buildMsg(topic string, v interface{}) *mockMessage {
	data, _ := json.Marshal(v)
	return &mockMessage{topic: topic, payload: data}
}

// ──────────────────── HandleRegister ────────────────────

func TestNodeHandler_HandleRegister_Success(t *testing.T) {
	svc, cleanup := newTestRegistryService(t)
	defer cleanup()

	publishCh := make(chan struct{ topic string; payload []byte }, 4)
	h := NewNodeHandler(svc, newTestHub(), newTestLogger(), capturePublish(publishCh))

	payload := map[string]interface{}{
		"header": map[string]interface{}{"message_type": "register"},
		"body": map[string]interface{}{
			"node_id":      "n1",
			"node_name":    "Edge Node 1",
			"protocol":     "mqtt",
			"access_token": "tok-abc",
		},
	}
	h.HandleRegister(nil, buildMsg("edgex/nodes/register", payload))

	node, err := svc.GetNode("n1")
	require.NoError(t, err)
	assert.Equal(t, "n1", node.NodeID)
	assert.Equal(t, "online", node.Status)
	assert.Equal(t, "tok-abc", node.AccessToken) // token 保留

	// 应有一条 publish 响应
	select {
	case pub := <-publishCh:
		assert.Equal(t, "edgex/nodes/n1/response", pub.topic)
	case <-time.After(time.Second):
		t.Fatal("expected publish response, got none")
	}
}

func TestNodeHandler_HandleRegister_GeneratesToken(t *testing.T) {
	svc, cleanup := newTestRegistryService(t)
	defer cleanup()

	publishCh := make(chan struct{ topic string; payload []byte }, 4)
	h := NewNodeHandler(svc, nil, newTestLogger(), capturePublish(publishCh))

	// 没有 access_token，handler 应自动生成
	payload := map[string]interface{}{
		"header": map[string]interface{}{},
		"body":   map[string]interface{}{"node_id": "n2", "node_name": "Auto Token Node"},
	}
	h.HandleRegister(nil, buildMsg("edgex/nodes/register", payload))

	node, err := svc.GetNode("n2")
	require.NoError(t, err)
	assert.NotEmpty(t, node.AccessToken, "token should be auto-generated")
}

func TestNodeHandler_HandleRegister_InvalidPayload(t *testing.T) {
	h, cleanup := newTestNodeHandler(t)
	defer cleanup()
	// 非法 JSON，不应 panic
	msg := &mockMessage{topic: "edgex/nodes/register", payload: []byte(`{invalid`)}
	h.HandleRegister(nil, msg)
}

// ──────────────────── HandleHeartbeat ────────────────────

func TestNodeHandler_HandleHeartbeat_UpdatesStatus(t *testing.T) {
	svc, cleanup := newTestRegistryService(t)
	defer cleanup()

	h := NewNodeHandler(svc, newTestHub(), newTestLogger(), nil)

	// 先注册节点
	payload := map[string]interface{}{
		"header": map[string]interface{}{},
		"body":   map[string]interface{}{"node_id": "n3", "node_name": "Heartbeat Node"},
	}
	h.HandleRegister(nil, buildMsg("edgex/nodes/register", payload))

	// 发送心跳
	hb := map[string]interface{}{
		"header": map[string]interface{}{},
		"body":   map[string]interface{}{"node_id": "n3"},
	}
	h.HandleHeartbeat(nil, buildMsg("edgex/nodes/heartbeat", hb))

	node, err := svc.GetNode("n3")
	require.NoError(t, err)
	assert.Equal(t, "online", node.Status)
}

func TestNodeHandler_HandleHeartbeat_FallbackHeader(t *testing.T) {
	svc, cleanup := newTestRegistryService(t)
	defer cleanup()

	// 先插入节点
	payload := map[string]interface{}{
		"header": map[string]interface{}{},
		"body":   map[string]interface{}{"node_id": "n4"},
	}
	h := NewNodeHandler(svc, nil, newTestLogger(), nil)
	h.HandleRegister(nil, buildMsg("edgex/nodes/register", payload))

	// 心跳 body.node_id 为空，从 header.source 读取
	hb := map[string]interface{}{
		"header": map[string]interface{}{"source": "n4"},
		"body":   map[string]interface{}{},
	}
	h.HandleHeartbeat(nil, buildMsg("edgex/nodes/heartbeat", hb))
	// 不 panic 即通过
}

func TestNodeHandler_HandleHeartbeat_InvalidPayload(t *testing.T) {
	h, cleanup := newTestNodeHandler(t)
	defer cleanup()
	msg := &mockMessage{topic: "edgex/nodes/heartbeat", payload: []byte(`bad`)}
	h.HandleHeartbeat(nil, msg)
}

// ──────────────────── HandleUnregister ────────────────────

func TestNodeHandler_HandleUnregister_SetsOffline(t *testing.T) {
	svc, cleanup := newTestRegistryService(t)
	defer cleanup()

	h := NewNodeHandler(svc, newTestHub(), newTestLogger(), nil)

	// 先注册
	payload := map[string]interface{}{
		"header": map[string]interface{}{},
		"body":   map[string]interface{}{"node_id": "n5"},
	}
	h.HandleRegister(nil, buildMsg("edgex/nodes/register", payload))

	// 注销
	unreg := map[string]interface{}{
		"header": map[string]interface{}{},
		"body":   map[string]interface{}{"node_id": "n5"},
	}
	h.HandleUnregister(nil, buildMsg("edgex/nodes/unregister", unreg))

	node, err := svc.GetNode("n5")
	require.NoError(t, err)
	assert.Equal(t, "offline", node.Status)
}

func TestNodeHandler_HandleUnregister_MissingNodeID(t *testing.T) {
	h, cleanup := newTestNodeHandler(t)
	defer cleanup()
	// body 无 node_id，应静默返回，不 panic
	payload := map[string]interface{}{
		"header": map[string]interface{}{},
		"body":   map[string]interface{}{},
	}
	h.HandleUnregister(nil, buildMsg("edgex/nodes/unregister", payload))
}

// ──────────────────── publishFn nil（无 broker）────────────────────

func TestNodeHandler_PublishFn_Nil(t *testing.T) {
	svc, cleanup := newTestRegistryService(t)
	defer cleanup()
	// publishFn 为 nil 时不应 panic
	h := NewNodeHandler(svc, nil, newTestLogger(), nil)

	payload := map[string]interface{}{
		"header": map[string]interface{}{},
		"body":   map[string]interface{}{"node_id": fmt.Sprintf("n-nil-%d", time.Now().UnixNano())},
	}
	h.HandleRegister(nil, buildMsg("edgex/nodes/register", payload))
}
