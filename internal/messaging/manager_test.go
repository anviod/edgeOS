package messaging

import (
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"go.etcd.io/bbolt"
	"go.uber.org/zap"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/anviod/edgeOS/internal/model"
	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/ws"
)

// ─── 测试辅助 ────────────────────────────────

func openTestDB(t *testing.T) (*bbolt.DB, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "edgeos_messaging_test_*.db")
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

func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

func newTestHub() *ws.Hub {
	return ws.NewHub(zap.NewNop())
}

// fakeClient 实现 pahomqtt.Client 接口
type fakeClient struct {
	connected   bool
	capturePub  func(topic string, payload interface{})
	captureSub  func(topic string, qos byte, handler pahomqtt.MessageHandler)
}

func (f *fakeClient) IsConnected() bool                   { return f.connected }
func (f *fakeClient) IsConnectionOpen() bool              { return f.connected }
func (f *fakeClient) Connect() pahomqtt.Token            { f.connected = true; return &fakeToken{} }
func (f *fakeClient) Disconnect(quiesce uint)            { f.connected = false }
func (f *fakeClient) Publish(topic string, qos byte, retained bool, payload interface{}) pahomqtt.Token {
	if f.capturePub != nil {
		f.capturePub(topic, payload)
	}
	return &fakeToken{}
}
func (f *fakeClient) Subscribe(topic string, qos byte, handler pahomqtt.MessageHandler) pahomqtt.Token {
	if f.captureSub != nil {
		f.captureSub(topic, qos, handler)
	}
	return &fakeToken{}
}
func (f *fakeClient) SubscribeMultiple(_ map[string]byte, _ pahomqtt.MessageHandler) pahomqtt.Token { return &fakeToken{} }
func (f *fakeClient) Unsubscribe(_ ...string) pahomqtt.Token                                        { return &fakeToken{} }
func (f *fakeClient) AddRoute(_ string, _ pahomqtt.MessageHandler)                                 {}
func (f *fakeClient) OptionsReader() pahomqtt.ClientOptionsReader                                     { return pahomqtt.ClientOptionsReader{} }

// fakeToken 实现 pahomqtt.Token
type fakeToken struct{}

func (t *fakeToken) Wait() bool                               { return true }
func (t *fakeToken) WaitTimeout(time.Duration) bool           { return true }
func (t *fakeToken) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
func (t *fakeToken) Error() error       { return nil }
func (t *fakeToken) SessionPresent() bool { return false }

// newTestManager 创建测试用 Manager（无真实 MQTT 连接）
func newTestManager(t *testing.T) (*Manager, *bbolt.DB, func()) {
	t.Helper()
	db, cleanup := openTestDB(t)
	registrySvc := services.NewRegistryService(db)
	deviceSvc := services.NewDeviceService(db)
	pointSvc := services.NewPointService(db)
	controlSvc := services.NewControlService(db, newTestLogger())
	dataSvc := &services.DataService{DeviceSvc: deviceSvc, PointService: pointSvc}
	middlewareSvc := services.NewMiddlewareService(db, newTestLogger())
	hub := newTestHub()

	mgr := &Manager{
		middlewareSvc: middlewareSvc,
		registrySvc:   registrySvc,
		dataSvc:       dataSvc,
		alertSvc:      nil,
		controlSvc:    controlSvc,
		hub:           hub,
		logger:        newTestLogger(),
		clients:       make(map[string]*mqttClientEntry),
	}
	mgr.initHandlers()
	return mgr, db, cleanup
}

type pubRecord struct {
	topic   string
	payload []byte
}

// addFakeClient 添加一个 fake MQTT 客户端到 Manager
func addFakeClient(m *Manager, id string, captureCh chan<- pubRecord) {
	cfg := &model.MiddlewareConfig{ID: id, QoS: 1}
	client := &fakeClient{connected: true}
	if captureCh != nil {
		client.capturePub = func(topic string, payload interface{}) {
			if b, ok := payload.([]byte); ok {
				captureCh <- pubRecord{topic: topic, payload: b}
			}
		}
	}
	m.mu.Lock()
	m.clients[id] = &mqttClientEntry{
		client:    client,
		config:    cfg,
		handlers:  make(map[string]pahomqtt.MessageHandler),
		publishFn: func(topic string, payload []byte) error {
			if captureCh != nil {
				captureCh <- pubRecord{topic: topic, payload: payload}
			}
			return nil
		},
	}
	m.mu.Unlock()
}

// ─── PublishNodeDiscovery 测试 ────────────────────────────────

func TestManager_PublishNodeDiscovery_NoClient(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	err := mgr.PublishNodeDiscovery()
	if err == nil {
		t.Fatal("expected error when no client connected, got nil")
	}
}

func TestManager_PublishNodeDiscovery_Success(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	publishCh := make(chan pubRecord, 4)
	addFakeClient(mgr, "mqtt-1", publishCh)

	err := mgr.PublishNodeDiscovery()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case rec := <-publishCh:
		if rec.topic != "edgex/cmd/nodes/register" {
			t.Fatalf("expected topic edgex/cmd/nodes/register, got %s", rec.topic)
		}
		var env map[string]interface{}
		if err := json.Unmarshal(rec.payload, &env); err != nil {
			t.Fatalf("invalid JSON payload: %v", err)
		}
		header, ok := env["header"].(map[string]interface{})
		if !ok {
			t.Fatal("missing header in envelope")
		}
		if header["message_type"] != "discovery_request" {
			t.Fatalf("expected message_type=discovery_request, got %v", header["message_type"])
		}
		if header["source"] != "edgeos" {
			t.Fatalf("expected source=edgeos, got %v", header["source"])
		}
		if header["timestamp"] == nil {
			t.Fatal("expected timestamp in header")
		}
	case <-time.After(time.Second):
		t.Fatal("no publish received within timeout")
	}
}

func TestManager_PublishNodeDiscovery_MultipleClients(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	ch1 := make(chan pubRecord, 1)
	ch2 := make(chan pubRecord, 1)
	addFakeClient(mgr, "mqtt-1", ch1)
	addFakeClient(mgr, "mqtt-2", ch2)

	err := mgr.PublishNodeDiscovery()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify exactly one channel receives the message (publishToFirstClient)
	received := 0
	timeout := time.After(time.Second)
	for received < 2 {
		select {
		case <-ch1:
			received++
		case <-ch2:
			received++
		case <-timeout:
			if received == 0 {
				t.Fatal("no publish received within timeout")
			}
			// Got at least one message - exit
			goto done
		}
	}
	// If we get a second message, it's a bug
	t.Fatal("second client should not receive publish (publishToFirstClient)")
done:
	// Drain remaining channel to avoid leaking state to next test
	select {
	case <-ch1:
	case <-time.After(time.Millisecond):
	}
	select {
	case <-ch2:
	case <-time.After(time.Millisecond):
	}
}

// ─── PublishNodeDiscoveryTo 测试 ────────────────────────────────

func TestManager_PublishNodeDiscoveryTo_NotConnected(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	err := mgr.PublishNodeDiscoveryTo("non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent middleware")
	}
}

func TestManager_PublishNodeDiscoveryTo_Success(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	publishCh := make(chan pubRecord, 4)
	addFakeClient(mgr, "mqtt-target", publishCh)

	err := mgr.PublishNodeDiscoveryTo("mqtt-target")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case rec := <-publishCh:
		if rec.topic != "edgex/cmd/nodes/register" {
			t.Fatalf("expected topic edgex/cmd/nodes/register, got %s", rec.topic)
		}
		var env map[string]interface{}
		json.Unmarshal(rec.payload, &env)
		header := env["header"].(map[string]interface{})
		if header["message_type"] != "discovery_request" {
			t.Fatalf("expected message_type=discovery_request, got %v", header["message_type"])
		}
	case <-time.After(time.Second):
		t.Fatal("no publish received")
	}
}

func TestManager_PublishNodeDiscoveryTo_Disconnected(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	cfg := &model.MiddlewareConfig{ID: "mqtt-offline", QoS: 1}
	client := &fakeClient{connected: false} // 默认 false
	mgr.mu.Lock()
	mgr.clients["mqtt-offline"] = &mqttClientEntry{
		client:   client,
		config:   cfg,
		handlers: make(map[string]pahomqtt.MessageHandler),
	}
	mgr.mu.Unlock()

	err := mgr.PublishNodeDiscoveryTo("mqtt-offline")
	if err == nil {
		t.Fatal("expected error for disconnected middleware")
	}
}

// ─── PublishCommand 测试 ────────────────────────────────

func TestManager_PublishCommand_NoClient(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	err := mgr.PublishCommand("n1", "d1", "p1", float64(42), "req-1")
	if err == nil {
		t.Fatal("expected error when no client connected")
	}
}

func TestManager_PublishCommand_Success(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	publishCh := make(chan pubRecord, 4)
	addFakeClient(mgr, "mqtt-1", publishCh)

	err := mgr.PublishCommand("n1", "d1", "p1", float64(42), "req-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case rec := <-publishCh:
		if rec.topic != "edgex/cmd/n1/d1/write" {
			t.Fatalf("expected topic edgex/cmd/n1/d1/write, got %s", rec.topic)
		}
		var env map[string]interface{}
		if err := json.Unmarshal(rec.payload, &env); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		body, ok := env["body"].(map[string]interface{})
		if !ok {
			t.Fatal("missing body")
		}
		if body["request_id"] != "req-123" {
			t.Fatalf("expected request_id=req-123, got %v", body["request_id"])
		}
		if body["value"] != float64(42) {
			t.Fatalf("expected value=42, got %v", body["value"])
		}
	case <-time.After(time.Second):
		t.Fatal("no publish received")
	}
}

func TestManager_PublishCommand_GeneratesRequestID(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	publishCh := make(chan pubRecord, 4)
	addFakeClient(mgr, "mqtt-1", publishCh)

	err := mgr.PublishCommand("n1", "d1", "p1", "hello", "") // empty requestID
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case rec := <-publishCh:
		var env map[string]interface{}
		json.Unmarshal(rec.payload, &env)
		body := env["body"].(map[string]interface{})
		if body["request_id"] == "" {
			t.Fatal("request_id should be auto-generated")
		}
	case <-time.After(time.Second):
		t.Fatal("no publish received")
	}
}

// ─── Connect/Disconnect 测试 ────────────────────────────────

func TestManager_IsConnected(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	if mgr.IsConnected("any") {
		t.Fatal("new manager should not report any connected clients")
	}

	addFakeClient(mgr, "test", nil)

	if !mgr.IsConnected("test") {
		t.Fatal("test client should be reported as connected")
	}
}

// ─── 并发安全测试 ────────────────────────────────

func TestManager_ConcurrentPublish(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	publishCh := make(chan pubRecord, 20)
	addFakeClient(mgr, "mqtt-1", publishCh)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mgr.PublishNodeDiscovery()
			mgr.PublishCommand("n1", "d1", "p1", float64(i), "")
		}()
	}
	wg.Wait()

	// 不 panic 即通过
}

// ─── PointMQTTHandler 测试 ────────────────────────────────

// fakeMessage 实现 pahomqtt.Message 接口
type fakeMessage struct {
	topic   string
	payload []byte
	retain  bool
}

func (f *fakeMessage) Topic() string             { return f.topic }
func (f *fakeMessage) Payload() []byte           { return f.payload }
func (f *fakeMessage) Qos() byte                 { return 1 }
func (f *fakeMessage) Retained() bool             { return f.retain }
func (f *fakeMessage) MessageID() uint16          { return 0 }
func (f *fakeMessage) Ack()                      {}
func (f *fakeMessage) Duplicate() bool           { return false }

func TestPointMQTTHandler_HandlePointReport(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	// 创建测试点位数据
	pointData := map[string]interface{}{
		"header": map[string]interface{}{
			"message_type": "point_report",
			"timestamp":    time.Now().UnixMilli(),
		},
		"body": map[string]interface{}{
			"node_id":   "test-node",
			"device_id": "test-device",
			"points": []map[string]interface{}{
				{
					"point_id":   "Temperature",
					"point_name": "温度",
					"data_type":  "Float32",
				},
				{
					"point_id":   "Humidity",
					"point_name": "湿度",
					"data_type":  "Float32",
				},
			},
		},
	}
	payload, _ := json.Marshal(pointData)

	// 调用 Handler
	msg := &fakeMessage{topic: "edgex/points/report", payload: payload}
	mgr.pointHandler.HandlePointReport(nil, msg)

	// 验证点位已保存
	points, err := mgr.dataSvc.PointService.ListByDevice("test-node", "test-device")
	if err != nil {
		t.Fatalf("failed to list points: %v", err)
	}

	if len(points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(points))
	}
}

func TestPointMQTTHandler_HandlePointReport_InvalidJSON(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	// 发送无效 JSON
	msg := &fakeMessage{topic: "edgex/points/report", payload: []byte("invalid json")}
	mgr.pointHandler.HandlePointReport(nil, msg) // 不应 panic
}

func TestPointMQTTHandler_HandlePointSync(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	// 创建点位全量同步数据
	syncData := map[string]interface{}{
		"header": map[string]interface{}{
			"message_type": "point_sync",
			"timestamp":    time.Now().UnixMilli(),
		},
		"body": map[string]interface{}{
			"node_id":   "test-node",
			"device_id": "test-device",
			"points": []map[string]interface{}{
				{
					"point_id":   "Pressure",
					"point_name": "压力",
					"data_type":  "Float64",
				},
				{
					"point_id":   "FlowRate",
					"point_name": "流量",
					"data_type":  "Float64",
				},
			},
		},
	}
	payload, _ := json.Marshal(syncData)

	// 调用 Handler
	msg := &fakeMessage{topic: "edgex/points/test-node/test-device", payload: payload}
	mgr.pointHandler.HandlePointSync(nil, msg)

	// 验证点位已保存
	points, err := mgr.dataSvc.PointService.ListByDevice("test-node", "test-device")
	if err != nil {
		t.Fatalf("failed to list points: %v", err)
	}

	if len(points) != 2 {
		t.Fatalf("expected 2 points after sync, got %d", len(points))
	}
}

func TestPointMQTTHandler_HandlePointSync_MissingNodeOrDeviceID(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	// 缺少 node_id
	syncData := map[string]interface{}{
		"header": map[string]interface{}{},
		"body": map[string]interface{}{
			"device_id": "test-device",
			"points":    []map[string]interface{}{},
		},
	}
	payload, _ := json.Marshal(syncData)
	msg := &fakeMessage{topic: "edgex/points/test-node/test-device", payload: payload}
	mgr.pointHandler.HandlePointSync(nil, msg) // 不应 panic，应提前返回

	// 缺少 device_id
	syncData2 := map[string]interface{}{
		"header": map[string]interface{}{},
		"body": map[string]interface{}{
			"node_id": "test-node",
			"points":  []map[string]interface{}{},
		},
	}
	payload2, _ := json.Marshal(syncData2)
	msg2 := &fakeMessage{topic: "edgex/points/test-node/test-device", payload: payload2}
	mgr.pointHandler.HandlePointSync(nil, msg2) // 不应 panic，应提前返回
}

func TestPointMQTTHandler_HandlePointSync_InvalidJSON(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	// 发送无效 JSON
	msg := &fakeMessage{topic: "edgex/points/test-node/test-device", payload: []byte("invalid json")}
	mgr.pointHandler.HandlePointSync(nil, msg) // 不应 panic
}

func TestPointMQTTHandler_HandlePointSync_UpdatesExistingPoint(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()
	pointSvc := mgr.dataSvc.PointService

	// 先插入一个点位
	initialPoint := &model.EdgeXPointInfo{
		PointID:   "Temperature",
		PointName: "温度传感器",
		DataType:  "Float32",
	}
	err := pointSvc.UpsertPoint("test-node", "test-device", initialPoint)
	if err != nil {
		t.Fatalf("failed to insert initial point: %v", err)
	}

	// 全量同步更新该点位（使用不同的 data_type）
	syncData := map[string]interface{}{
		"header": map[string]interface{}{},
		"body": map[string]interface{}{
			"node_id":   "test-node",
			"device_id": "test-device",
			"points": []map[string]interface{}{
				{
					"point_id":   "Temperature",
					"point_name": "温度传感器(已更新)",
					"data_type":  "Float64", // changed
				},
			},
		},
	}
	payload, _ := json.Marshal(syncData)
	msg := &fakeMessage{topic: "edgex/points/test-node/test-device", payload: payload}
	mgr.pointHandler.HandlePointSync(nil, msg)

	// 验证点位已更新
	updatedPoint, err := pointSvc.GetMeta("test-node", "test-device", "Temperature")
	if err != nil {
		t.Fatalf("failed to get updated point: %v", err)
	}

	if updatedPoint.DataType != "Float64" {
		t.Fatalf("expected data_type=Float64, got %s", updatedPoint.DataType)
	}
	if updatedPoint.PointName != "温度传感器(已更新)" {
		t.Fatalf("expected point_name=温度传感器(已更新), got %s", updatedPoint.PointName)
	}
}

func TestPointMQTTHandler_HandlePointSync_AddNewPoints(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()
	pointSvc := mgr.dataSvc.PointService

	// 先插入一个点位
	initialPoint := &model.EdgeXPointInfo{
		PointID:   "Temperature",
		PointName: "温度",
	}
	pointSvc.UpsertPoint("test-node", "test-device", initialPoint)

	// 全量同步添加新点位
	syncData := map[string]interface{}{
		"header": map[string]interface{}{},
		"body": map[string]interface{}{
			"node_id":   "test-node",
			"device_id": "test-device",
			"points": []map[string]interface{}{
				{"point_id": "Temperature", "point_name": "温度"},
				{"point_id": "Humidity", "point_name": "湿度"},
			},
		},
	}
	payload, _ := json.Marshal(syncData)
	msg := &fakeMessage{topic: "edgex/points/test-node/test-device", payload: payload}
	mgr.pointHandler.HandlePointSync(nil, msg)

	// 验证两个点位都存在
	points, err := pointSvc.ListByDevice("test-node", "test-device")
	if err != nil {
		t.Fatalf("failed to list points: %v", err)
	}

	if len(points) != 2 {
		t.Fatalf("expected 2 points after sync, got %d", len(points))
	}
}

func TestPointMQTTHandler_HandlePointReport_WithHubBroadcast(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	// 注册一个 WebSocket 客户端用于接收广播
	done := make(chan struct{})
	mgr.hub.Broadcast(ws.RealtimeEvent{
		Type:      "test_setup",
		Timestamp: time.Now().UnixMilli(),
		Payload:   nil,
	})
	close(done) // 确认 hub 可用

	// 发送点位报告
	pointData := map[string]interface{}{
		"header": map[string]interface{}{},
		"body": map[string]interface{}{
			"node_id":   "test-node",
			"device_id": "test-device",
			"points": []map[string]interface{}{
				{"point_id": "Temp", "point_name": "温度"},
			},
		},
	}
	payload, _ := json.Marshal(pointData)
	msg := &fakeMessage{topic: "edgex/points/report", payload: payload}
	mgr.pointHandler.HandlePointReport(nil, msg)

	// 验证点位已保存
	points, _ := mgr.dataSvc.PointService.ListByDevice("test-node", "test-device")
	if len(points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(points))
	}
}

func TestPointMQTTHandler_HandleRealtimeData(t *testing.T) {
	mgr, _, cleanup := newTestManager(t)
	defer cleanup()

	// 发送实时数据
	dataMsg := map[string]interface{}{
		"header": map[string]interface{}{},
		"body": map[string]interface{}{
			"node_id":   "test-node",
			"device_id": "test-device",
			"points": map[string]interface{}{
				"Temperature": float64(25.5),
				"Humidity":   float64(65.0),
			},
			"quality":         "good",
			"timestamp":       time.Now().Unix(),
			"is_full_snapshot": true,
		},
	}
	payload, _ := json.Marshal(dataMsg)
	msg := &fakeMessage{topic: "edgex/data/test-node/test-device", payload: payload}
	mgr.pointHandler.HandleRealtimeData(nil, msg)

	// 验证快照已保存
	snapshot, err := mgr.dataSvc.PointService.GetSnapshot("test-node", "test-device")
	if err != nil {
		t.Fatalf("failed to get snapshot: %v", err)
	}

	if snapshot.Points["Temperature"] != float64(25.5) {
		t.Fatalf("expected Temperature=25.5, got %v", snapshot.Points["Temperature"])
	}
}
