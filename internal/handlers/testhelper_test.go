package handlers

import (
	"os"
	"testing"

	"go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/ws"
)

// openTestDB 创建临时 BoltDB，返回清理函数
func openTestDB(t *testing.T) (*bbolt.DB, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "edgeos_handler_test_*.db")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	f.Close()

	db, err := bbolt.Open(f.Name(), 0600, nil)
	if err != nil {
		os.Remove(f.Name())
		t.Fatalf("open bbolt: %v", err)
	}

	return db, func() {
		db.Close()
		os.Remove(f.Name())
	}
}

// newTestLogger 返回 nop logger
func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

// newTestHub 返回无客户端的 Hub（用于测试广播不 panic）
func newTestHub() *ws.Hub {
	return ws.NewHub(zap.NewNop())
}

// mockMessage 实现 paho.mqtt.golang.Message 接口的最小 mock
type mockMessage struct {
	topic   string
	payload []byte
}

func (m *mockMessage) Duplicate() bool          { return false }
func (m *mockMessage) Qos() byte                { return 0 }
func (m *mockMessage) Retained() bool           { return false }
func (m *mockMessage) Topic() string             { return m.topic }
func (m *mockMessage) MessageID() uint16         { return 0 }
func (m *mockMessage) Payload() []byte           { return m.payload }
func (m *mockMessage) Ack()                      {}

// capturePublish 构建一个 publishFn，将发布的 topic/payload 写入 channel
func capturePublish(ch chan<- struct{ topic string; payload []byte }) func(string, []byte) error {
	return func(topic string, payload []byte) error {
		ch <- struct{ topic string; payload []byte }{topic: topic, payload: payload}
		return nil
	}
}

// newTestRegistryService 创建 RegistryService + 清理函数
func newTestRegistryService(t *testing.T) (*services.RegistryService, func()) {
	t.Helper()
	db, cleanup := openTestDB(t)
	return services.NewRegistryService(db), cleanup
}

// newTestDeviceService 创建 DeviceService + 清理函数
func newTestDeviceService(t *testing.T) (*services.DeviceService, func()) {
	t.Helper()
	db, cleanup := openTestDB(t)
	return services.NewDeviceService(db), cleanup
}

// newTestPointService 创建 PointService + 清理函数
func newTestPointService(t *testing.T) (*services.PointService, func()) {
	t.Helper()
	db, cleanup := openTestDB(t)
	return services.NewPointService(db), cleanup
}

// newTestControlService 创建 ControlService + 清理函数
func newTestControlService(t *testing.T) (*services.ControlService, func()) {
	t.Helper()
	db, cleanup := openTestDB(t)
	return services.NewControlService(db, newTestLogger()), cleanup
}
