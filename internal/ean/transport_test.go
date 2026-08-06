package ean

import (
	"fmt"
	"sync"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTransport 实现 Transport 接口的 mock
type mockTransport struct {
	name       string
	connected  bool
	closed     bool
	subscribeFn func(topic string, handler MessageHandler) error
	publishFn  func(topic string, payload []byte) error

	mu sync.Mutex
}

func (m *mockTransport) Name() string { return m.name }

func (m *mockTransport) Endpoint() string {
	if m.name == "nats" {
		return "nats://mock"
	}
	return "tcp://mock"
}

func (m *mockTransport) Subscribe(topic string, handler MessageHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.subscribeFn != nil {
		return m.subscribeFn(topic, handler)
	}
	return nil
}

func (m *mockTransport) Publish(topic string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.publishFn != nil {
		return m.publishFn(topic, payload)
	}
	return nil
}

func (m *mockTransport) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *mockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	m.connected = false
	return nil
}

func TestMqttTopicToNatsSubject(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "exact topic unchanged",
			input:    "$edgeos/discovery/agent",
			expected: "$edgeos/discovery/agent",
		},
		{
			name:     "single level wildcard + to *",
			input:    "$edgeos/event/+/status",
			expected: "$edgeos/event/*/status",
		},
		{
			name:     "multi-level wildcard # to >",
			input:    "$edgeos/event/#",
			expected: "$edgeos/event/>",
		},
		{
			name:     "multi-level with invoke",
			input:    "$edgeos/invoke/edgeCore-node-001",
			expected: "$edgeos/invoke/edgeCore-node-001",
		},
		{
			name:     "multiple single-level wildcards",
			input:    "$edgeos/+/+/capability",
			expected: "$edgeos/*/*/capability",
		},
		{
			name:     "trailing wildcard",
			input:    "$edgeos/heartbeat/#",
			expected: "$edgeos/heartbeat/>",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no wildcards simple topic",
			input:    "a/b/c/d",
			expected: "a/b/c/d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mqttTopicToNatsSubject(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDualTransport_Add(t *testing.T) {
	dt := NewDualTransport(zap.NewNop())
	m := &mockTransport{name: "mqtt", connected: true}

	err := dt.Add(m)
	require.NoError(t, err)

	// 重复添加应报错
	err = dt.Add(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestDualTransport_Remove(t *testing.T) {
	dt := NewDualTransport(zap.NewNop())
	m := &mockTransport{name: "mqtt", connected: true}
	_ = dt.Add(m)

	err := dt.Remove("mqtt")
	require.NoError(t, err)
	assert.True(t, m.closed) // Remove 会调用 Close

	// 移除不存在的 transport 应报错
	err = dt.Remove("nats")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDualTransport_Subscribe(t *testing.T) {
	dt := NewDualTransport(zap.NewNop())
	var subscribedTopics []string
	m := &mockTransport{
		name:      "mqtt",
		connected: true,
		subscribeFn: func(topic string, handler MessageHandler) error {
			subscribedTopics = append(subscribedTopics, topic)
			return nil
		},
	}
	_ = dt.Add(m)

	err := dt.Subscribe("$edgeos/discovery/agent", func(topic string, payload []byte, transport string) {})
	require.NoError(t, err)
	assert.Equal(t, []string{"$edgeos/discovery/agent"}, subscribedTopics)
}

func TestDualTransport_SubscribeError(t *testing.T) {
	dt := NewDualTransport(zap.NewNop())
	m := &mockTransport{
		name:      "mqtt",
		connected: true,
		subscribeFn: func(topic string, handler MessageHandler) error {
			return fmt.Errorf("subscribe failed")
		},
	}
	_ = dt.Add(m)

	err := dt.Subscribe("$edgeos/test", func(topic string, payload []byte, transport string) {})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subscribe failed")
}

func TestDualTransport_Publish(t *testing.T) {
	dt := NewDualTransport(zap.NewNop())

	var published []struct {
		topic   string
		payload []byte
	}
	m := &mockTransport{
		name:      "mqtt",
		connected: true,
		publishFn: func(topic string, payload []byte) error {
			published = append(published, struct {
				topic   string
				payload []byte
			}{topic: topic, payload: payload})
			return nil
		},
	}
	_ = dt.Add(m)

	err := dt.Publish("$edgeos/test", []byte("hello"))
	require.NoError(t, err)
	require.Len(t, published, 1)
	assert.Equal(t, "$edgeos/test", published[0].topic)
	assert.Equal(t, []byte("hello"), published[0].payload)
}

func TestDualTransport_PublishSkipDisconnected(t *testing.T) {
	dt := NewDualTransport(zap.NewNop())
	m := &mockTransport{
		name:      "mqtt",
		connected: false, // 未连接
	}
	_ = dt.Add(m)

	// 未连接的 transport 不应被调用，且返回 nil（无可用传输）
	err := dt.Publish("$edgeos/test", []byte("hello"))
	assert.NoError(t, err)
}

func TestDualTransport_IsAnyConnected(t *testing.T) {
	dt := NewDualTransport(zap.NewNop())

	// 无 transport 时返回 false
	assert.False(t, dt.IsConnected())

	m1 := &mockTransport{name: "mqtt", connected: false}
	_ = dt.Add(m1)
	assert.False(t, dt.IsConnected())

	m2 := &mockTransport{name: "nats", connected: true}
	_ = dt.Add(m2)
	assert.True(t, dt.IsConnected())
}

func TestDualTransport_ConnectedNames(t *testing.T) {
	dt := NewDualTransport(zap.NewNop())

	_ = dt.Add(&mockTransport{name: "mqtt", connected: true})
	_ = dt.Add(&mockTransport{name: "nats", connected: false})

	names := dt.ConnectedNames()
	assert.Equal(t, []string{"mqtt"}, names)
}

func TestDualTransport_Transports(t *testing.T) {
	dt := NewDualTransport(zap.NewNop())
	m1 := &mockTransport{name: "mqtt", connected: true}
	m2 := &mockTransport{name: "nats", connected: true}
	_ = dt.Add(m1)
	_ = dt.Add(m2)

	transports := dt.Transports()
	assert.Len(t, transports, 2)
	_, ok := transports["mqtt"]
	assert.True(t, ok)
	_, ok = transports["nats"]
	assert.True(t, ok)
}

func TestDualTransport_Get(t *testing.T) {
	dt := NewDualTransport(zap.NewNop())
	m := &mockTransport{name: "mqtt", connected: true}
	_ = dt.Add(m)

	got, ok := dt.Get("mqtt")
	assert.True(t, ok)
	assert.Equal(t, m, got)

	_, ok = dt.Get("nonexistent")
	assert.False(t, ok)
}

func TestDualTransport_Close(t *testing.T) {
	dt := NewDualTransport(zap.NewNop())
	m1 := &mockTransport{name: "mqtt", connected: true}
	m2 := &mockTransport{name: "nats", connected: true}
	_ = dt.Add(m1)
	_ = dt.Add(m2)

	err := dt.Close()
	assert.NoError(t, err)
	assert.True(t, m1.closed)
	assert.True(t, m2.closed)

	// 关闭后 transports 应为空
	transports := dt.Transports()
	assert.Empty(t, transports)
}
