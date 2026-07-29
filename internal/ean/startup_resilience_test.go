package ean

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/config"
)

// TestNewBus_UnreachableMQTTBroker_DoesNotFail 验证 broker 不可用时 NewBus/Start 不报错、不 panic，
// 传输层仍注册并支持后台重连（与 messaging.Manager 降级行为一致）。
func TestNewBus_UnreachableMQTTBroker_DoesNotFail(t *testing.T) {
	cfg := BusConfig{
		PlannerID: "edgeos-planner-test",
		MQTT: config.EANMQTTConfig{
			Enabled:        true,
			Broker:         "tcp://ean-unreachable.test.invalid:19999", // 不可解析/不可达
			ClientID:       "edgeos-ean-resilience-test",
			QoS:            1,
			ConnectTimeout: 1, // 缩短首连等待
			KeepAlive:      30,
		},
		NATS: config.EANNATSConfig{Enabled: false},
		Heartbeat: config.EANHeartbeatConfig{
			TimeoutMultiplier: 3,
			CheckIntervalSec:  5,
		},
	}

	bus, err := NewBus(cfg, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, bus)

	// 即使未连上，MQTT 传输层也应已注册（后台 ConnectRetry）
	transports := bus.Transport().Transports()
	require.Contains(t, transports, "mqtt")

	err = bus.Start()
	require.NoError(t, err, "Start must not fail when broker is down")

	health := bus.Health()
	assert.Equal(t, true, health["started"])
	assert.Equal(t, 1, health["registered_transports"])
	// 不可达 broker：通常未连接；若环境异常连上也不应影响 Start 成功
	_ = health["transports"]
	assert.NotNil(t, health["invoke_metrics"])
	details, ok := health["transport_details"].([]TransportDetail)
	require.True(t, ok)
	require.Len(t, details, 1)
	assert.Equal(t, "mqtt", details[0].Name)
	assert.Equal(t, "tcp://ean-unreachable.test.invalid:19999", details[0].Endpoint)

	// 给后台重试一点时间，确认进程侧无异常
	time.Sleep(200 * time.Millisecond)
	bus.Stop()
}

// TestNewBus_UnreachableNATS_DoesNotFail 验证 NATS 不可达时 NewBus/Start 不 fatal，传输仍注册
func TestNewBus_UnreachableNATS_DoesNotFail(t *testing.T) {
	cfg := BusConfig{
		PlannerID: "edgeos-planner-nats-resilience",
		MQTT:      config.EANMQTTConfig{Enabled: false},
		NATS: config.EANNATSConfig{
			Enabled:        true,
			URL:            "nats://ean-unreachable.test.invalid:4222",
			ClientName:     "edgeos-ean-nats-resilience",
			ConnectTimeout: 1,
			ReconnectWait:  1,
			MaxReconnects:  2,
		},
		Heartbeat: config.EANHeartbeatConfig{
			TimeoutMultiplier: 3,
			CheckIntervalSec:  5,
		},
	}

	bus, err := NewBus(cfg, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, bus)
	require.Contains(t, bus.Transport().Transports(), "nats")

	err = bus.Start()
	require.NoError(t, err, "Start must not fail when NATS is down")

	health := bus.Health()
	assert.Equal(t, true, health["started"])
	assert.Equal(t, 1, health["registered_transports"])
	details, ok := health["transport_details"].([]TransportDetail)
	require.True(t, ok)
	require.Len(t, details, 1)
	assert.Equal(t, "nats", details[0].Name)
	assert.Contains(t, details[0].Endpoint, "nats://")

	time.Sleep(200 * time.Millisecond)
	bus.Stop()
}

// TestNewMQTTTransport_DeferredSubscribe 未连接时 Subscribe 仅登记，不返回错误
func TestNewMQTTTransport_DeferredSubscribe(t *testing.T) {
	tr, err := NewMQTTTransport(MQTTConfig{
		BrokerURL:      "tcp://ean-unreachable.test.invalid:19998",
		ClientID:       "edgeos-ean-deferred-sub",
		QoS:            1,
		ConnectTimeout: 1,
	}, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, tr)
	defer tr.Close()

	err = tr.Subscribe("$edgeos/discovery/agent", func(string, []byte, string) {})
	require.NoError(t, err, "deferred subscribe must not fail when disconnected")
}

// TestNewNATSTransport_DeferredSubscribe 未连接时 Subscribe 仅登记，不返回错误
func TestNewNATSTransport_DeferredSubscribe(t *testing.T) {
	tr, err := NewNATSTransport(NATSConfig{
		URL:            "nats://ean-unreachable.test.invalid:4222",
		Name:           "edgeos-ean-nats-deferred",
		ConnectTimeout: 1,
		ReconnectWait:  1,
		MaxReconnects:  1,
	}, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, tr)
	defer tr.Close()

	err = tr.Subscribe("$edgeos/discovery/agent", func(string, []byte, string) {})
	require.NoError(t, err, "nats deferred subscribe must not fail when disconnected")
	assert.Equal(t, "nats://ean-unreachable.test.invalid:4222", tr.Endpoint())
}

func TestInvokeMetricsPercentile(t *testing.T) {
	samples := []int64{10, 20, 30, 40, 100}
	assert.Equal(t, int64(10), percentileApprox(samples, 0))
	assert.Equal(t, int64(30), percentileApprox(samples, 50))
	assert.Equal(t, int64(100), percentileApprox(samples, 99))
	assert.Equal(t, int64(0), percentileApprox(nil, 50))
}
