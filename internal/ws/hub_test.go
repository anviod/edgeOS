package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newTestHub 创建用于测试的 Hub（使用 nop logger）
func newTestHub() *Hub {
	logger, _ := zap.NewDevelopment()
	return NewHub(logger)
}

// addFakeClient 向 Hub 注册一个不依赖真实 WebSocket 连接的 fake client，
// 返回 client 指针和其 send channel（用于断言接收到的消息）。
func addFakeClient(h *Hub) (*client, chan []byte) {
	ch := make(chan []byte, 16)
	c := &client{
		conn: nil,
		send: ch,
		done: make(chan struct{}),
	}
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
	return c, ch
}

// drainClient 从 send channel 读取所有待处理消息，返回列表。
func drainClient(ch chan []byte, wait time.Duration) [][]byte {
	var msgs [][]byte
	deadline := time.After(wait)
	for {
		select {
		case msg := <-ch:
			msgs = append(msgs, msg)
		case <-deadline:
			return msgs
		}
	}
}

// ---------- ClientCount ----------

func TestHub_ClientCount_Empty(t *testing.T) {
	h := newTestHub()
	assert.Equal(t, 0, h.ClientCount())
}

func TestHub_ClientCount_AfterAdd(t *testing.T) {
	h := newTestHub()
	addFakeClient(h)
	addFakeClient(h)
	assert.Equal(t, 2, h.ClientCount())
}

func TestHub_ClientCount_AfterRemove(t *testing.T) {
	h := newTestHub()
	c, _ := addFakeClient(h)
	addFakeClient(h)
	require.Equal(t, 2, h.ClientCount())

	h.removeClient(c)
	assert.Equal(t, 1, h.ClientCount())
}

// ---------- removeClient ----------

func TestHub_RemoveClient_Idempotent(t *testing.T) {
	h := newTestHub()
	c, _ := addFakeClient(h)

	h.removeClient(c)
	assert.Equal(t, 0, h.ClientCount())

	// 重复调用不应 panic（channel 已关闭，select default 分支保护）
	assert.NotPanics(t, func() {
		h.removeClient(c)
	})
}

func TestHub_RemoveClient_ClosesChannel(t *testing.T) {
	h := newTestHub()
	c, ch := addFakeClient(h)
	h.removeClient(c)

	// channel 关闭后从中读取会立即返回零值 + false
	_, open := <-ch
	assert.False(t, open, "send channel should be closed after removeClient")
}

// ---------- Broadcast ----------

func TestHub_Broadcast_DeliveredToAllClients(t *testing.T) {
	h := newTestHub()
	_, ch1 := addFakeClient(h)
	_, ch2 := addFakeClient(h)

	event := RealtimeEvent{
		Type:    EventNodeStatus,
		Payload: map[string]string{"node_id": "n1", "status": "online"},
	}
	h.Broadcast(event)

	// 两个客户端都应收到消息
	msgs1 := drainClient(ch1, 200*time.Millisecond)
	msgs2 := drainClient(ch2, 200*time.Millisecond)

	require.Len(t, msgs1, 1)
	require.Len(t, msgs2, 1)

	// 解码验证内容
	var got1 RealtimeEvent
	require.NoError(t, json.Unmarshal(msgs1[0], &got1))
	assert.Equal(t, EventNodeStatus, got1.Type)
	assert.NotZero(t, got1.Timestamp, "Broadcast should set Timestamp")
}

func TestHub_Broadcast_TimestampAutoSet(t *testing.T) {
	h := newTestHub()
	_, ch := addFakeClient(h)

	before := time.Now().UnixMilli()
	h.Broadcast(RealtimeEvent{Type: EventAlert, Payload: "test"})
	after := time.Now().UnixMilli()

	msgs := drainClient(ch, 200*time.Millisecond)
	require.Len(t, msgs, 1)

	var got RealtimeEvent
	require.NoError(t, json.Unmarshal(msgs[0], &got))
	assert.GreaterOrEqual(t, got.Timestamp, before)
	assert.LessOrEqual(t, got.Timestamp, after)
}

func TestHub_Broadcast_NoClients_NoPanic(t *testing.T) {
	h := newTestHub()
	assert.NotPanics(t, func() {
		h.Broadcast(RealtimeEvent{Type: EventDataUpdate, Payload: nil})
	})
}

func TestHub_Broadcast_SlowClient_Removed(t *testing.T) {
	h := newTestHub()
	// 创建一个容量为 0 的 channel 模拟"满"客户端（select default 立即触发）
	slowCh := make(chan []byte) // 无缓冲，发送必然走 default
	slowClient := &client{
		conn: nil,
		send: slowCh,
		done: make(chan struct{}),
	}
	h.mu.Lock()
	h.clients[slowClient] = true
	h.mu.Unlock()

	// 正常客户端应正常收到
	_, normalCh := addFakeClient(h)

	h.Broadcast(RealtimeEvent{Type: EventDataUpdate, Payload: "hello"})

	// 慢客户端应被踢出
	assert.Equal(t, 1, h.ClientCount(), "slow client should be removed")

	// 正常客户端应收到消息
	msgs := drainClient(normalCh, 200*time.Millisecond)
	assert.Len(t, msgs, 1)
}

// ---------- BroadcastType ----------

func TestHub_BroadcastType_SetsTypeAndPayload(t *testing.T) {
	h := newTestHub()
	_, ch := addFakeClient(h)

	payload := DataUpdatePayload{
		NodeID:   "n1",
		DeviceID: "d1",
		Points:   map[string]interface{}{"temp": 25.5},
	}
	h.BroadcastType(EventDataUpdate, payload)

	msgs := drainClient(ch, 200*time.Millisecond)
	require.Len(t, msgs, 1)

	var got RealtimeEvent
	require.NoError(t, json.Unmarshal(msgs[0], &got))
	assert.Equal(t, EventDataUpdate, got.Type)
	assert.NotNil(t, got.Payload)
}

func TestHub_BroadcastType_AllEventTypes(t *testing.T) {
	types := []EventType{
		EventDataUpdate,
		EventNodeStatus,
		EventDeviceSynced,
		EventCommandResp,
		EventAlert,
		EventMiddlewareStatus,
		EventPointReport,
	}
	for _, et := range types {
		t.Run(string(et), func(t *testing.T) {
			h := newTestHub()
			_, ch := addFakeClient(h)

			h.BroadcastType(et, map[string]string{"k": "v"})

			msgs := drainClient(ch, 200*time.Millisecond)
			require.Len(t, msgs, 1)

			var got RealtimeEvent
			require.NoError(t, json.Unmarshal(msgs[0], &got))
			assert.Equal(t, et, got.Type)
		})
	}
}

// ---------- 并发安全 ----------

func TestHub_Broadcast_ConcurrentSafe(t *testing.T) {
	h := newTestHub()
	for i := 0; i < 5; i++ {
		addFakeClient(h)
	}

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			h.BroadcastType(EventAlert, "concurrent")
		}()
	}
	// 并发 removeClient 同时进行
	go func() {
		h.mu.RLock()
		var cs []*client
		for c := range h.clients {
			cs = append(cs, c)
		}
		h.mu.RUnlock()
		for _, c := range cs {
			h.removeClient(c)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent test timed out")
	}
	// 只要不 panic、不死锁即通过
}
