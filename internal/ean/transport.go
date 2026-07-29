package ean

import (
	"fmt"
	"strings"
	"sync"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// ==================== 接口定义 ====================

// MessageHandler 消息回调签名，所有传输层共享
// topic: 原始主题（MQTT 保持 $edgeos/...，NATS 同样保持 $edgeos/...）
// payload: 原始消息体
// transport: 传输层标识（"mqtt" / "nats"）
type MessageHandler func(topic string, payload []byte, transport string)

// Transport 传输层抽象接口，MQTT/NATS 均须实现
type Transport interface {
	// Name 返回传输层唯一标识，如 "mqtt"、"nats"
	Name() string

	// Endpoint 返回连接端点（broker URL / NATS URL），供健康状态展示
	Endpoint() string

	// Subscribe 订阅指定主题，收到的消息回调 handler
	Subscribe(topic string, handler MessageHandler) error

	// Publish 向指定主题发布消息
	Publish(topic string, payload []byte) error

	// IsConnected 当前传输层是否已连接
	IsConnected() bool

	// Close 关闭连接、释放资源
	Close() error
}

// TransportDetail 单传输层健康快照（供 /api/ean/health）
type TransportDetail struct {
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
	Endpoint  string `json:"endpoint"`
}

// ==================== MQTT 适配器 ====================

// MQTTConfig MQTT 连接配置
type MQTTConfig struct {
	BrokerURL      string // 例: tcp://127.0.0.1:18083
	ClientID       string // 客户端标识，建议全局唯一
	QoS            byte   // 默认 1，Discovery/Invoke/Event 建议 QoS1
	Username       string // 可选认证
	Password       string
	ConnectTimeout int    // 单次连接尝试超时（秒），默认 10
	KeepAlive      int    // 保活间隔（秒），默认 30
}

// MQTTTransport MQTT 传输适配器，实现 Transport 接口
// 启动韧性：broker 不可用时仍创建实例，依赖 ConnectRetry/AutoReconnect 后台重连；
// 订阅登记在本地，OnConnect 时统一补订。
type MQTTTransport struct {
	cfg     MQTTConfig
	client  pahomqtt.Client
	logger  *zap.Logger
	running bool
	mu      sync.RWMutex
	subs    map[string]MessageHandler // topic -> handler，供重连后补订
}

// NewMQTTTransport 创建 MQTT 传输适配器
// 始终返回可用实例（error 恒为 nil，除非配置非法）；初始连接失败仅 Warn，后台持续重试。
func NewMQTTTransport(cfg MQTTConfig, logger *zap.Logger) (*MQTTTransport, error) {
	if cfg.BrokerURL == "" {
		return nil, fmt.Errorf("mqtt broker url is required")
	}
	if cfg.QoS == 0 {
		cfg.QoS = 1 // EAN 默认 QoS1
	}
	connectTimeout := time.Duration(cfg.ConnectTimeout) * time.Second
	if connectTimeout <= 0 {
		connectTimeout = 10 * time.Second
	}
	keepAlive := time.Duration(cfg.KeepAlive) * time.Second
	if keepAlive <= 0 {
		keepAlive = 30 * time.Second
	}

	t := &MQTTTransport{
		cfg:     cfg,
		logger:  logger.Named("mqtt"),
		running: true,
		subs:    make(map[string]MessageHandler),
	}

	opts := pahomqtt.NewClientOptions().
		AddBroker(cfg.BrokerURL).
		SetClientID(cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true). // 初始连接失败也持续重试，不拖垮进程
		SetConnectRetryInterval(5 * time.Second).
		SetMaxReconnectInterval(30 * time.Second).
		SetConnectTimeout(connectTimeout).
		SetKeepAlive(keepAlive).
		SetCleanSession(true).
		SetOnConnectHandler(func(_ pahomqtt.Client) {
			t.logger.Info("mqtt transport connected/reconnected", zap.String("broker", cfg.BrokerURL))
			t.resubscribeAll()
		}).
		SetConnectionLostHandler(func(_ pahomqtt.Client, err error) {
			t.logger.Warn("mqtt connection lost, will auto-reconnect",
				zap.String("broker", cfg.BrokerURL), zap.Error(err))
		})

	if cfg.Username != "" {
		opts.SetUsername(cfg.Username).SetPassword(cfg.Password)
	}

	client := pahomqtt.NewClient(opts)
	t.client = client

	// 触发连接；ConnectRetry=true 时后台持续重试。
	// WaitTimeout 仅用于打日志，超时/失败都不返回 error。
	token := client.Connect()
	if token.WaitTimeout(connectTimeout) {
		if err := token.Error(); err != nil {
			t.logger.Warn("mqtt initial connect failed, will keep retrying in background",
				zap.String("broker", cfg.BrokerURL), zap.Error(err))
		} else {
			t.logger.Info("mqtt transport connected", zap.String("broker", cfg.BrokerURL))
		}
	} else {
		t.logger.Warn("mqtt initial connect still pending, will keep retrying in background",
			zap.String("broker", cfg.BrokerURL),
			zap.Duration("waited", connectTimeout))
	}

	return t, nil
}

// Name 返回 "mqtt"
func (t *MQTTTransport) Name() string { return "mqtt" }

// Endpoint 返回 MQTT broker URL
func (t *MQTTTransport) Endpoint() string { return t.cfg.BrokerURL }

// Subscribe 订阅 MQTT 主题；未连接或订阅失败时仅登记，待 OnConnect 补订（永不因未连接而失败）
func (t *MQTTTransport) Subscribe(topic string, handler MessageHandler) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running {
		return fmt.Errorf("mqtt transport is closed")
	}
	t.subs[topic] = handler

	if t.client == nil || !t.client.IsConnected() {
		t.logger.Debug("mqtt subscribe deferred until connected", zap.String("topic", topic))
		return nil
	}
	if err := t.subscribeLocked(topic, handler); err != nil {
		t.logger.Debug("mqtt subscribe deferred after error",
			zap.String("topic", topic), zap.Error(err))
		return nil
	}
	return nil
}

func (t *MQTTTransport) subscribeLocked(topic string, handler MessageHandler) error {
	token := t.client.Subscribe(topic, t.cfg.QoS, func(_ pahomqtt.Client, msg pahomqtt.Message) {
		payload := make([]byte, len(msg.Payload()))
		copy(payload, msg.Payload())
		handler(msg.Topic(), payload, "mqtt")
	})
	token.Wait()
	return token.Error()
}

// resubscribeAll 连接建立后补订所有已登记主题（不持锁等待 MQTT token，避免与 Subscribe 死锁）
func (t *MQTTTransport) resubscribeAll() {
	t.mu.RLock()
	if !t.running || t.client == nil || !t.client.IsConnected() {
		t.mu.RUnlock()
		return
	}
	snapshot := make(map[string]MessageHandler, len(t.subs))
	for topic, handler := range t.subs {
		snapshot[topic] = handler
	}
	client := t.client
	qos := t.cfg.QoS
	t.mu.RUnlock()

	for topic, handler := range snapshot {
		h := handler
		token := client.Subscribe(topic, qos, func(_ pahomqtt.Client, msg pahomqtt.Message) {
			payload := make([]byte, len(msg.Payload()))
			copy(payload, msg.Payload())
			h(msg.Topic(), payload, "mqtt")
		})
		token.Wait()
		if err := token.Error(); err != nil {
			t.logger.Warn("mqtt resubscribe failed",
				zap.String("topic", topic), zap.Error(err))
		} else {
			t.logger.Debug("mqtt resubscribed", zap.String("topic", topic))
		}
	}
}

// Publish 发布 MQTT 消息
func (t *MQTTTransport) Publish(topic string, payload []byte) error {
	t.mu.RLock()
	running := t.running
	client := t.client
	t.mu.RUnlock()
	if !running || client == nil {
		return fmt.Errorf("mqtt transport is closed")
	}
	if !client.IsConnected() {
		return fmt.Errorf("mqtt transport not connected")
	}
	token := client.Publish(topic, t.cfg.QoS, false, payload)
	token.Wait()
	return token.Error()
}

// IsConnected 检查 MQTT 连接状态
func (t *MQTTTransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.client != nil && t.client.IsConnected()
}

// Close 关闭 MQTT 连接
func (t *MQTTTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running {
		return nil
	}
	t.running = false
	if t.client != nil && t.client.IsConnected() {
		t.client.Disconnect(1000)
	} else if t.client != nil {
		// 取消后台 ConnectRetry
		t.client.Disconnect(250)
	}
	t.logger.Info("mqtt transport closed")
	return nil
}

// ==================== NATS 适配器 ====================

// NATSConfig NATS 连接配置
type NATSConfig struct {
	URL            string // 例: nats://127.0.0.1:4222
	Name           string // 连接名称，用于服务端日志
	Cluster        string // 可选集群名
	ConnectTimeout int    // 单次连接超时（秒）
	ReconnectWait  int    // 重连等待（秒）
	MaxReconnects  int    // 最大重连次数；<=0 表示无限
}

// NATSTransport NATS 传输适配器，实现 Transport 接口
// 启动韧性：服务器不可用时仍创建实例（RetryOnFailedConnect），订阅本地登记，
// Connect/Reconnect 时补订。Subject 保持 $edgeos/... 斜杠形式。
type NATSTransport struct {
	cfg      NATSConfig
	conn     *nats.Conn
	jsCtx    nats.JetStreamContext // 可选，按需创建
	logger   *zap.Logger
	pending  map[string]MessageHandler // MQTT 风格 topic -> handler，供延迟/补订
	active   []*nats.Subscription
	mu       sync.Mutex
	closed   bool
}

// NewNATSTransport 创建 NATS 传输适配器
// 注意: NATS subject 保持 $edgeos/... 斜杠形式，不转换为点分形式
// 服务器不可用时启用 RetryOnFailedConnect，不拖垮进程。
func NewNATSTransport(cfg NATSConfig, logger *zap.Logger) (*NATSTransport, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("nats url is required")
	}
	reconnectWait := time.Duration(cfg.ReconnectWait) * time.Second
	if reconnectWait <= 0 {
		reconnectWait = 2 * time.Second
	}
	maxReconnects := cfg.MaxReconnects
	if maxReconnects == 0 {
		maxReconnects = -1 // 无限重连
	}
	connectTimeout := time.Duration(cfg.ConnectTimeout) * time.Second
	if connectTimeout <= 0 {
		connectTimeout = 10 * time.Second
	}

	t := &NATSTransport{
		cfg:     cfg,
		logger:  logger.Named("nats"),
		pending: make(map[string]MessageHandler),
	}

	opts := []nats.Option{
		nats.Name(cfg.Name),
		nats.ReconnectWait(reconnectWait),
		nats.MaxReconnects(maxReconnects),
		nats.Timeout(connectTimeout),
		nats.RetryOnFailedConnect(true),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			t.logger.Warn("nats disconnected, will reconnect", zap.Error(err))
		}),
		nats.ConnectHandler(func(_ *nats.Conn) {
			t.logger.Info("nats transport connected", zap.String("url", cfg.URL))
			t.resubscribeAll()
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			t.logger.Info("nats reconnected", zap.String("url", cfg.URL))
			t.resubscribeAll()
		}),
	}

	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		// RetryOnFailedConnect 通常不会走到这里；保留兜底
		return nil, fmt.Errorf("nats connect failed: %w", err)
	}
	t.conn = conn

	if conn.IsConnected() {
		t.logger.Info("nats transport connected", zap.String("url", cfg.URL))
	} else {
		t.logger.Warn("nats initial connect pending, will keep retrying in background",
			zap.String("url", cfg.URL))
	}
	return t, nil
}

// Name 返回 "nats"
func (t *NATSTransport) Name() string { return "nats" }

// Endpoint 返回 NATS URL
func (t *NATSTransport) Endpoint() string { return t.cfg.URL }

// Subscribe 订阅 NATS subject；未连接或订阅失败时仅登记，待 Connect/Reconnect 补订
// topic 参数为 MQTT 风格的斜杠主题，自动转换通配符（+→* / #→>），分隔符保持斜杠
func (t *NATSTransport) Subscribe(topic string, handler MessageHandler) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("nats transport is closed")
	}
	t.pending[topic] = handler

	if t.conn == nil || !t.conn.IsConnected() {
		t.logger.Debug("nats subscribe deferred until connected", zap.String("topic", topic))
		return nil
	}
	if err := t.subscribeLocked(topic, handler); err != nil {
		t.logger.Debug("nats subscribe deferred after error",
			zap.String("topic", topic), zap.Error(err))
		return nil
	}
	return nil
}

func (t *NATSTransport) subscribeLocked(topic string, handler MessageHandler) error {
	subject := mqttTopicToNatsSubject(topic)
	sub, err := t.conn.Subscribe(subject, func(msg *nats.Msg) {
		handler(msg.Subject, msg.Data, "nats")
	})
	if err != nil {
		return fmt.Errorf("nats subscribe %s failed: %w", subject, err)
	}
	t.active = append(t.active, sub)
	return nil
}

// resubscribeAll 连接建立后补订所有已登记主题
func (t *NATSTransport) resubscribeAll() {
	t.mu.Lock()
	if t.closed || t.conn == nil || !t.conn.IsConnected() {
		t.mu.Unlock()
		return
	}
	// 清空旧订阅句柄（重连后库内订阅可能已失效；按 pending 重建）
	for _, sub := range t.active {
		if sub != nil {
			_ = sub.Unsubscribe()
		}
	}
	t.active = nil
	snapshot := make(map[string]MessageHandler, len(t.pending))
	for topic, handler := range t.pending {
		snapshot[topic] = handler
	}
	conn := t.conn
	t.mu.Unlock()

	for topic, handler := range snapshot {
		h := handler
		subject := mqttTopicToNatsSubject(topic)
		sub, err := conn.Subscribe(subject, func(msg *nats.Msg) {
			h(msg.Subject, msg.Data, "nats")
		})
		t.mu.Lock()
		if t.closed {
			t.mu.Unlock()
			if sub != nil {
				_ = sub.Unsubscribe()
			}
			return
		}
		if err != nil {
			t.logger.Warn("nats resubscribe failed",
				zap.String("topic", topic), zap.String("subject", subject), zap.Error(err))
		} else {
			t.active = append(t.active, sub)
			t.logger.Debug("nats resubscribed", zap.String("topic", topic), zap.String("subject", subject))
		}
		t.mu.Unlock()
	}
}

// Publish 发布 NATS 消息
func (t *NATSTransport) Publish(topic string, payload []byte) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return fmt.Errorf("nats transport is closed")
	}
	conn := t.conn
	t.mu.Unlock()

	if conn == nil || !conn.IsConnected() {
		return fmt.Errorf("nats transport not connected")
	}

	subject := mqttTopicToNatsSubject(topic)
	return conn.Publish(subject, payload)
}

// IsConnected 检查 NATS 连接状态
func (t *NATSTransport) IsConnected() bool {
	return t.conn != nil && t.conn.IsConnected()
}

// Close 关闭 NATS 连接并取消所有订阅
func (t *NATSTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true

	for _, sub := range t.active {
		if sub != nil {
			_ = sub.Unsubscribe()
		}
	}
	t.active = nil
	t.pending = make(map[string]MessageHandler)
	if t.conn != nil {
		t.conn.Close()
	}
	t.logger.Info("nats transport closed")
	return nil
}

// JetStream 返回 JetStream 上下文（可选），按需创建
func (t *NATSTransport) JetStream() (nats.JetStreamContext, error) {
	if t.jsCtx != nil {
		return t.jsCtx, nil
	}
	js, err := t.conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("jetstream init failed: %w", err)
	}
	t.jsCtx = js
	return js, nil
}

// ==================== 主题转换工具 ====================

// mqttTopicToNatsSubject 将 MQTT 风格主题转为 NATS subject
// 规则: + -> *, # -> >，其余保持斜杠形式不变
//
// 示例:
//
//	$edgeos/discovery/agent          -> $edgeos/discovery/agent
//	$edgeos/event/+/status           -> $edgeos/event/*/status
//	$edgeos/event/#                  -> $edgeos/event/>
//	$edgeos/invoke/edgex-node-001    -> $edgeos/invoke/edgex-node-001
func mqttTopicToNatsSubject(topic string) string {
	r := strings.NewReplacer("+", "*", "#", ">")
	return r.Replace(topic)
}

// ==================== 双传输管理器 ====================

// DualTransport 多传输层管理器，同时管理 MQTT/NATS 等多个 Transport
// 订阅时向所有传输层注册，发布时向所有已连接的传输层发送
type DualTransport struct {
	transports map[string]Transport // name -> Transport
	logger     *zap.Logger
	mu         sync.RWMutex
}

// NewDualTransport 创建双传输管理器
func NewDualTransport(logger *zap.Logger) *DualTransport {
	return &DualTransport{
		transports: make(map[string]Transport),
		logger:     logger.Named("dual-transport"),
	}
}

// Add 注册一个传输层实例
func (d *DualTransport) Add(t Transport) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	name := t.Name()
	if _, exists := d.transports[name]; exists {
		return fmt.Errorf("transport %q already registered", name)
	}
	d.transports[name] = t
	d.logger.Info("transport added", zap.String("name", name))
	return nil
}

// Remove 移除并关闭指定传输层
func (d *DualTransport) Remove(name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	t, ok := d.transports[name]
	if !ok {
		return fmt.Errorf("transport %q not found", name)
	}
	_ = t.Close()
	delete(d.transports, name)
	d.logger.Info("transport removed", zap.String("name", name))
	return nil
}

// Subscribe 向所有传输层订阅同一主题
// 任一传输层订阅失败不影响其他传输层，仅记录警告
func (d *DualTransport) Subscribe(topic string, handler MessageHandler) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var firstErr error
	for name, t := range d.transports {
		if err := t.Subscribe(topic, handler); err != nil {
			d.logger.Warn("subscribe failed on transport",
				zap.String("topic", topic),
				zap.String("transport", name),
				zap.Error(err))
			if firstErr == nil {
				firstErr = err
			}
		} else {
			d.logger.Debug("subscribe ok",
				zap.String("topic", topic),
				zap.String("transport", name))
		}
	}
	return firstErr
}

// Publish 向所有已连接的传输层发布消息
// 无已连接传输时跳过并返回 nil（由上层 ConnectedNames 判断是否可发）
func (d *DualTransport) Publish(topic string, payload []byte) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var firstErr error
	for name, t := range d.transports {
		if !t.IsConnected() {
			d.logger.Warn("publish skipped: transport not connected",
				zap.String("topic", topic),
				zap.String("transport", name))
			continue
		}
		if err := t.Publish(topic, payload); err != nil {
			d.logger.Warn("publish failed on transport",
				zap.String("topic", topic),
				zap.String("transport", name),
				zap.Error(err))
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// IsConnected 任一传输层已连接即返回 true
func (d *DualTransport) IsConnected() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, t := range d.transports {
		if t.IsConnected() {
			return true
		}
	}
	return false
}

// ConnectedNames 返回所有已连接传输层名称
func (d *DualTransport) ConnectedNames() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var names []string
	for name, t := range d.transports {
		if t.IsConnected() {
			names = append(names, name)
		}
	}
	return names
}

// Details 返回所有已注册传输层的健康快照（含未连接）
func (d *DualTransport) Details() []TransportDetail {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]TransportDetail, 0, len(d.transports))
	for name, t := range d.transports {
		out = append(out, TransportDetail{
			Name:      name,
			Connected: t.IsConnected(),
			Endpoint:  t.Endpoint(),
		})
	}
	return out
}

// Get 获取指定名称的传输层
func (d *DualTransport) Get(name string) (Transport, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	t, ok := d.transports[name]
	return t, ok
}

// Transports 返回所有传输层（只读快照）
func (d *DualTransport) Transports() map[string]Transport {
	d.mu.RLock()
	defer d.mu.RUnlock()
	cp := make(map[string]Transport, len(d.transports))
	for k, v := range d.transports {
		cp[k] = v
	}
	return cp
}

// Close 关闭所有传输层
func (d *DualTransport) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var lastErr error
	for name, t := range d.transports {
		if err := t.Close(); err != nil {
			d.logger.Warn("close transport failed",
				zap.String("name", name), zap.Error(err))
			lastErr = err
		}
	}
	d.transports = make(map[string]Transport)
	d.logger.Info("dual transport closed")
	return lastErr
}
