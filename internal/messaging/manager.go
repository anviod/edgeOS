package messaging

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/model"
	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/ws"
)

// HeartbeatMessage represents the new rich heartbeat message format
type HeartbeatMessage struct {
	Header struct {
		MessageID   string `json:"message_id"`
		Timestamp   int64  `json:"timestamp"`
		Source      string `json:"source"`
		MessageType string `json:"message_type"`
		Version     string `json:"version"`
	} `json:"header"`
	Body struct {
		NodeID          string             `json:"node_id"`
		Status          string             `json:"status"`
		Timestamp       int64              `json:"timestamp"`
		Sequence        int                `json:"sequence"`
		UptimeSeconds   int                `json:"uptime_seconds"`
		Version         string             `json:"version"`
		SystemMetrics   SystemMetrics      `json:"system_metrics"`
		DeviceSummary   DeviceSummary      `json:"device_summary"`
		ChannelSummary  ChannelSummary     `json:"channel_summary"`
		TaskSummary     TaskSummary        `json:"task_summary"`
		ConnectionStats ConnectionStats    `json:"connection_stats"`
		CustomMetrics   map[string]float64 `json:"custom_metrics"`
	} `json:"body"`
}

// SystemMetrics represents system metrics in the heartbeat message
type SystemMetrics struct {
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	MemoryTotal    int64   `json:"memory_total"`
	MemoryUsed     int64   `json:"memory_used"`
	DiskUsage      float64 `json:"disk_usage"`
	DiskTotal      int64   `json:"disk_total"`
	DiskUsed       int64   `json:"disk_used"`
	LoadAverage    float64 `json:"load_average"`
	NetworkRxBytes int64   `json:"network_rx_bytes"`
	NetworkTxBytes int64   `json:"network_tx_bytes"`
	ProcessCount   int     `json:"process_count"`
	ThreadCount    int     `json:"thread_count"`
}

// DeviceSummary represents device statistics in the heartbeat message
type DeviceSummary struct {
	TotalCount      int `json:"total_count"`
	OnlineCount     int `json:"online_count"`
	OfflineCount    int `json:"offline_count"`
	ErrorCount      int `json:"error_count"`
	DegradedCount   int `json:"degraded_count"`
	RecoveringCount int `json:"recovering_count"`
}

// ChannelSummary represents channel statistics in the heartbeat message
type ChannelSummary struct {
	TotalCount     int     `json:"total_count"`
	ConnectedCount int     `json:"connected_count"`
	ErrorCount     int     `json:"error_count"`
	AvgSuccessRate float64 `json:"avg_success_rate"`
}

// TaskSummary represents task statistics in the heartbeat message
type TaskSummary struct {
	TotalCount   int `json:"total_count"`
	RunningCount int `json:"running_count"`
	PausedCount  int `json:"paused_count"`
	ErrorCount   int `json:"error_count"`
}

// ConnectionStats represents MQTT connection statistics in the heartbeat message
type ConnectionStats struct {
	ReconnectCount  int    `json:"reconnect_count"`
	LastOnlineTime  int64  `json:"last_online_time"`
	LastOfflineTime int64  `json:"last_offline_time"`
	ConnectedSince  int64  `json:"connected_since"`
	PublishCount    int    `json:"publish_count"`
	ProtocolVersion string `json:"protocol_version"`
}

// Manager MQTT 消息管理器
// 负责：管理多个 MQTT 连接、主题订阅、消息路由到对应 Handler
// 支持：1) 启动时从配置/BoltDB 连接启用的中间件 2) 运行时动态添加/移除中间件
type Manager struct {
	middlewareSvc *services.MiddlewareService
	registrySvc   *services.RegistryService
	dataSvc       *services.DataService
	alertSvc      *services.AlertService
	controlSvc    *services.ControlService
	hub           *ws.Hub
	logger        *zap.Logger

	// 消息处理器
	nodeHandler    *NodeMQTTHandler
	deviceHandler  *DeviceMQTTHandler
	pointHandler   *PointMQTTHandler
	controlHandler *ControlMQTTHandler

	mu      sync.RWMutex
	clients map[string]*mqttClientEntry // middlewareID -> client
	stopped bool
}

// mqttClientEntry 单个 MQTT 客户端条目
type mqttClientEntry struct {
	client      pahomqtt.Client
	config      *model.MiddlewareConfig
	handlers    map[string]pahomqtt.MessageHandler
	handlerLock sync.RWMutex
	publishFn   func(topic string, payload []byte) error
}

// NodeMQTTHandler 节点消息处理器（内联避免循环依赖）
type NodeMQTTHandler struct {
	registrySvc *services.RegistryService
	hub         *ws.Hub
	logger      *zap.Logger
	publishFn   func(topic string, payload []byte) error
}

// DeviceMQTTHandler 设备消息处理器
type DeviceMQTTHandler struct {
	deviceSvc *services.DeviceService
	pointSvc  *services.PointService
	hub       *ws.Hub
	logger    *zap.Logger
}

// PointMQTTHandler 点位消息处理器
type PointMQTTHandler struct {
	pointSvc  *services.PointService
	deviceSvc *services.DeviceService
	hub       *ws.Hub
	logger    *zap.Logger
}

// ControlMQTTHandler 控制命令处理器
type ControlMQTTHandler struct {
	controlSvc *services.ControlService
	hub        *ws.Hub
	logger     *zap.Logger
}

// NewManager 创建消息管理器
func NewManager(
	middlewareSvc *services.MiddlewareService,
	registrySvc *services.RegistryService,
	dataSvc *services.DataService,
	alertSvc *services.AlertService,
	controlSvc *services.ControlService,
	hub *ws.Hub,
	logger *zap.Logger,
) *Manager {
	m := &Manager{
		middlewareSvc: middlewareSvc,
		registrySvc:   registrySvc,
		dataSvc:       dataSvc,
		alertSvc:      alertSvc,
		controlSvc:    controlSvc,
		hub:           hub,
		logger:        logger,
		clients:       make(map[string]*mqttClientEntry),
	}
	m.initHandlers()
	return m
}

// initHandlers 初始化消息处理器
func (m *Manager) initHandlers() {
	// publishFn 使用默认客户端（如果有）
	m.nodeHandler = NewNodeMQTTHandler(m.registrySvc, m.hub, m.logger, m.publishToFirstClient)
	m.deviceHandler = NewDeviceMQTTHandler(m.dataSvc.DeviceSvc, m.dataSvc.PointService, m.hub, m.logger)
	m.pointHandler = NewPointMQTTHandler(m.dataSvc.PointService, m.dataSvc.DeviceSvc, m.hub, m.logger)
	m.controlHandler = NewControlMQTTHandler(m.controlSvc, m.hub, m.logger)
}

// publishToFirstClient 发布到第一个已连接的客户端
func (m *Manager) publishToFirstClient(topic string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, entry := range m.clients {
		if entry.client.IsConnected() {
			token := entry.client.Publish(topic, entry.config.QoS, false, payload)
			token.Wait()
			return token.Error()
		}
	}
	return fmt.Errorf("no MQTT client connected")
}

// Start 启动消息管理器，连接所有启用的中间件
func (m *Manager) Start() error {
	if m.middlewareSvc == nil {
		m.logger.Warn("MiddlewareService not provided, messaging disabled")
		return nil
	}

	// 加载所有启用的中间件并连接
	enabled, err := m.middlewareSvc.ListEnabled()
	if err != nil {
		m.logger.Error("Failed to list enabled middlewares", zap.Error(err))
	}

	for _, cfg := range enabled {
		if err := m.Connect(cfg.ID); err != nil {
			m.logger.Error("Failed to connect middleware on startup",
				zap.String("id", cfg.ID), zap.Error(err))
		}
	}

	if len(enabled) == 0 {
		m.logger.Info("No enabled middlewares found, messaging idle")
	} else {
		m.logger.Info("Messaging manager started",
			zap.Int("connected", m.connectedCount()))
	}
	return nil
}

// Connect 连接指定 ID 的中间件
func (m *Manager) Connect(id string) error {
	cfg, err := m.middlewareSvc.Get(id)
	if err != nil {
		return fmt.Errorf("middleware not found: %s", id)
	}

	m.logger.Info("DEBUG Connect: retrieved from DB",
		zap.String("id", id),
		zap.String("Host", cfg.Host),
		zap.Int("Port", cfg.Port),
		zap.String("Broker", cfg.Broker),
		zap.String("ClientID", cfg.ClientID))

	m.mu.Lock()
	if _, exists := m.clients[id]; exists {
		m.mu.Unlock()
		m.logger.Info("Middleware already connected", zap.String("id", id))
		return nil
	}

	// 创建 MQTT 客户端选项
	opts := pahomqtt.NewClientOptions()
	broker := cfg.Broker
	if broker == "" {
		if cfg.SSL {
			broker = fmt.Sprintf("ssl://%s:%d", cfg.Host, cfg.Port)
		} else {
			broker = fmt.Sprintf("tcp://%s:%d", cfg.Host, cfg.Port)
		}
	}
	// 如果 SSL 启用，替换 broker 协议为 ssl://
	if cfg.SSL && strings.HasPrefix(broker, "tcp://") {
		broker = "ssl://" + broker[6:]
	}

	// 设置 MQTT 版本
	mqttVersion := cfg.MQTTVersion
	if mqttVersion == 0 {
		mqttVersion = 4 // 默认 MQTT 3.1.1
	}

	m.logger.Info("MQTT connecting",
		zap.String("id", id),
		zap.String("broker", broker),
		zap.String("client_id", cfg.ClientID),
		zap.String("username", cfg.Username),
		zap.Int("connect_timeout", cfg.ConnectTimeout),
		zap.Int("keep_alive", cfg.KeepAlive),
		zap.Bool("clean_session", cfg.CleanSession),
		zap.Bool("ssl", cfg.SSL),
		zap.Int("mqtt_version", mqttVersion))

	opts.AddBroker(broker)
	opts.SetProtocolVersion(uint(mqttVersion))
	if cfg.ClientID != "" {
		opts.SetClientID(cfg.ClientID)
	}
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)
	opts.SetCleanSession(cfg.CleanSession)
	opts.SetKeepAlive(time.Duration(cfg.KeepAlive) * time.Second)
	opts.SetAutoReconnect(cfg.AutoReconnect)
	opts.SetConnectTimeout(time.Duration(cfg.ConnectTimeout) * time.Second)

	// 配置 SSL/TLS
	if cfg.SSL {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.CAFile == "", // 如果没有 CA 证书，跳过验证
		}
		// 如果有 CA 证书文件，可以加载自定义 CA
		// 注意：paho.mqtt.golang 目前不支持直接从文件加载 CA，
		// 需要使用系统证书或自定义加载
		opts.SetTLSConfig(tlsConfig)
		m.logger.Info("SSL/TLS enabled for MQTT connection",
			zap.String("ca_file", cfg.CAFile))
	}

	client := pahomqtt.NewClient(opts)
	token := client.Connect()
	if token.WaitTimeout(time.Duration(cfg.ConnectTimeout+5)*time.Second) && token.Error() != nil {
		m.mu.Unlock()
		m.middlewareSvc.UpdateStatus(id, "error", token.Error().Error())
		return fmt.Errorf("connect failed: %w", token.Error())
	}

	// 保存客户端
	entry := &mqttClientEntry{
		client:   client,
		config:   cfg,
		handlers: make(map[string]pahomqtt.MessageHandler),
		publishFn: func(topic string, payload []byte) error {
			token := client.Publish(topic, cfg.QoS, false, payload)
			token.Wait()
			return token.Error()
		},
	}
	m.clients[id] = entry
	m.mu.Unlock()

	// 订阅所有主题
	m.subscribeAllTopics(entry)

	m.middlewareSvc.UpdateStatus(id, "connected", "")
	m.logger.Info("Middleware connected",
		zap.String("id", id),
		zap.String("broker", broker),
		zap.Strings("topics", cfg.Subscriptions))

	// WebSocket 广播
	if m.hub != nil {
		m.hub.BroadcastType(ws.EventMiddlewareStatus, map[string]interface{}{
			"id":     id,
			"name":   cfg.Name,
			"status": "connected",
			"broker": broker,
		})
	}

	return nil
}

// Disconnect 断开指定 ID 的中间件
func (m *Manager) Disconnect(id string) error {
	m.mu.Lock()
	entry, exists := m.clients[id]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("middleware not connected: %s", id)
	}
	delete(m.clients, id)
	m.mu.Unlock()

	if entry.client.IsConnected() {
		entry.client.Disconnect(500)
	}
	m.middlewareSvc.UpdateStatus(id, "disconnected", "")

	m.logger.Info("Middleware disconnected", zap.String("id", id))

	if m.hub != nil {
		m.hub.BroadcastType(ws.EventMiddlewareStatus, map[string]interface{}{
			"id":     id,
			"status": "disconnected",
		})
	}
	return nil
}

// IsConnected 检查指定中间件是否已连接
func (m *Manager) IsConnected(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry, ok := m.clients[id]; ok {
		return entry.client.IsConnected()
	}
	return false
}

// Stop 停止所有消息总线
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = true
	for id, entry := range m.clients {
		if entry.client.IsConnected() {
			entry.client.Disconnect(500)
		}
		m.middlewareSvc.UpdateStatus(id, "disconnected", "")
	}
	m.clients = make(map[string]*mqttClientEntry)
}

// PublishCommand 发布设备控制命令（发送到第一个连接的客户端）
func (m *Manager) PublishCommand(nodeID, deviceID, pointID string, value interface{}, requestID string) error {
	if requestID == "" {
		requestID = uuid.New().String()
	}

	payload, err := json.Marshal(map[string]interface{}{
		"header": map[string]interface{}{
			"message_id":   uuid.New().String(),
			"message_type": "command_write",
			"timestamp":    time.Now().UnixMilli(),
			"request_id":   requestID,
		},
		"body": map[string]interface{}{
			"node_id":   nodeID,
			"device_id": deviceID,
			"point_id":  pointID,
			"value":     value,
		},
	})
	if err != nil {
		return fmt.Errorf("marshal command failed: %w", err)
	}

	topic := fmt.Sprintf("edgex/cmd/%s/%s/write", nodeID, deviceID)

	m.mu.Lock()
	defer m.mu.Unlock()
	for _, entry := range m.clients {
		if entry.client.IsConnected() {
			token := entry.client.Publish(topic, entry.config.QoS, false, payload)
			token.Wait()
			m.logger.Debug("Command published",
				zap.String("topic", topic),
				zap.String("request_id", requestID))
			return token.Error()
		}
	}
	return fmt.Errorf("no MQTT client connected")
}

// PublishNodeDiscovery 主动向第一个已连接的中间件发布节点发现请求
// 触发 EdgeX 节点重新注册：edgex/cmd/nodes/register
// 典型流程：EdgeOS → MQTT → EdgeX 节点 → edgex/nodes/register → MQTT → EdgeOS
func (m *Manager) PublishNodeDiscovery() error {
	msg := map[string]interface{}{
		"header": map[string]interface{}{
			"message_type": "discovery_request",
			"source":       "edgeos",
			"timestamp":    time.Now().UnixMilli(),
		},
		"body": map[string]interface{}{},
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal discovery request failed: %w", err)
	}
	return m.publishToFirstClient("edgex/cmd/nodes/register", payload)
}

// PublishNodeDiscoveryTo 向指定 middlewareID 发布节点发现请求
func (m *Manager) PublishNodeDiscoveryTo(middlewareID string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.clients[middlewareID]
	if !ok || entry.client == nil || !entry.client.IsConnected() {
		return fmt.Errorf("middleware %s not connected", middlewareID)
	}
	msg := map[string]interface{}{
		"header": map[string]interface{}{
			"message_type": "discovery_request",
			"source":       "edgeos",
			"timestamp":    time.Now().UnixMilli(),
		},
		"body": map[string]interface{}{},
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal discovery request failed: %w", err)
	}
	return entry.publishFn("edgex/cmd/nodes/register", payload)
}

// subscribeAllTopics 订阅中间件配置的所有主题
func (m *Manager) subscribeAllTopics(entry *mqttClientEntry) {
	client := entry.client
	cfg := entry.config

	bindings := []struct {
		topic   string
		handler pahomqtt.MessageHandler
	}{
		{"edgex/nodes/register", m.nodeHandler.HandleRegister},
		{"edgex/nodes/+/heartbeat", m.nodeHandler.HandleHeartbeat},
		{"edgex/nodes/+/status", m.nodeHandler.HandleHeartbeat},
		{"edgex/nodes/unregister", m.nodeHandler.HandleUnregister},
		{"edgex/devices/report", m.deviceHandler.HandleDeviceReport},
		{"edgex/devices/+/+/online", m.deviceHandler.HandleDeviceOnline},
		{"edgex/devices/+/+/offline", m.deviceHandler.HandleDeviceOffline},
		{"edgex/points/report", m.pointHandler.HandlePointReport},
		{"edgex/points/+/+", m.pointHandler.HandlePointSync},
		{"edgex/data/+/+", m.pointHandler.HandleRealtimeData},
		{"edgex/events/alert", m.handleAlert},
		{"edgex/events/error", m.handleAlert},
		{"edgex/events/info", m.handleAlert},
		{"edgex/cmd/responses/#", m.controlHandler.HandleCommandResponse},
	}

	// 动态订阅配置的主题
	for _, topic := range cfg.Subscriptions {
		topic = strings.TrimSpace(topic)
		if topic == "" {
			continue
		}
		// 检查是否已绑定
		found := false
		for _, b := range bindings {
			if b.topic == topic {
				found = true
				break
			}
		}
		if !found {
			bindings = append(bindings, struct {
				topic   string
				handler pahomqtt.MessageHandler
			}{topic, m.pointHandler.HandleRealtimeData})
		}
	}

	for _, b := range bindings {
		if b.topic == "" {
			continue
		}
		token := client.Subscribe(b.topic, cfg.QoS, b.handler)
		token.Wait()
		if token.Error() != nil {
			m.logger.Error("Failed to subscribe",
				zap.String("topic", b.topic),
				zap.String("middleware", cfg.ID),
				zap.Error(token.Error()))
		} else {
			m.logger.Info("Subscribed",
				zap.String("topic", b.topic),
				zap.String("middleware", cfg.ID))
			entry.handlerLock.Lock()
			entry.handlers[b.topic] = b.handler
			entry.handlerLock.Unlock()
		}
	}
}

// handleAlert 处理告警消息
func (m *Manager) handleAlert(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Body model.AlertInfo `json:"body"`
	}
	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		m.logger.Error("Manager.handleAlert: unmarshal failed",
			zap.Error(err), zap.String("topic", msg.Topic()))
		return
	}
	alert := &envelope.Body
	if err := m.alertSvc.AddAlert(alert); err != nil {
		m.logger.Error("Manager.handleAlert: add alert failed", zap.Error(err))
		return
	}
	m.logger.Info("Alert received",
		zap.String("alert_id", alert.ID),
		zap.String("level", alert.Level))
	if m.hub != nil {
		m.hub.BroadcastType(ws.EventAlert, alert)
	}
}

// connectedCount 返回已连接的客户端数量
func (m *Manager) connectedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, entry := range m.clients {
		if entry.client.IsConnected() {
			n++
		}
	}
	return n
}

// ==================== 节点消息处理器 ====================

func NewNodeMQTTHandler(
	registrySvc *services.RegistryService,
	hub *ws.Hub,
	logger *zap.Logger,
	publishFn func(topic string, payload []byte) error,
) *NodeMQTTHandler {
	return &NodeMQTTHandler{
		registrySvc: registrySvc,
		hub:         hub,
		logger:      logger,
		publishFn:   publishFn,
	}
}

func (h *NodeMQTTHandler) HandleRegister(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Header map[string]interface{} `json:"header"`
		Body   model.EdgeXNodeInfo    `json:"body"`
	}
	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		h.logger.Error("HandleRegister: unmarshal failed", zap.Error(err))
		return
	}

	node := &envelope.Body
	node.Status = "online"
	if node.AccessToken == "" {
		node.AccessToken = uuid.New().String()
		node.ExpiresAt = time.Now().Add(24 * time.Hour).Unix()
	}

	if err := h.registrySvc.UpsertNode(node); err != nil {
		h.logger.Error("HandleRegister: upsert failed",
			zap.String("node_id", node.NodeID), zap.Error(err))
		return
	}

	h.logger.Info("Node registered",
		zap.String("node_id", node.NodeID),
		zap.String("node_name", node.NodeName))

	h.publishRegisterResponse(node)

	if h.hub != nil {
		h.hub.BroadcastType(ws.EventNodeStatus, ws.NodeStatusPayload{
			NodeID:   node.NodeID,
			NodeName: node.NodeName,
			Status:   "online",
			LastSeen: node.LastSeen,
		})
	}
}

func (h *NodeMQTTHandler) publishRegisterResponse(node *model.EdgeXNodeInfo) {
	if h.publishFn == nil {
		return
	}
	resp, _ := json.Marshal(map[string]interface{}{
		"header": map[string]interface{}{
			"message_id":   uuid.New().String(),
			"message_type": "register_response",
			"timestamp":    time.Now().UnixMilli(),
		},
		"body": map[string]interface{}{
			"node_id":      node.NodeID,
			"status":       "success",
			"access_token": node.AccessToken,
			"expires_at":   node.ExpiresAt,
		},
	})
	topic := fmt.Sprintf("edgex/nodes/%s/response", node.NodeID)
	if err := h.publishFn(topic, resp); err != nil {
		h.logger.Error("HandleRegister: publish response failed",
			zap.String("topic", topic), zap.Error(err))
	}
}

func (h *NodeMQTTHandler) HandleHeartbeat(_ pahomqtt.Client, msg pahomqtt.Message) {
	var heartbeat HeartbeatMessage
	if err := json.Unmarshal(msg.Payload(), &heartbeat); err != nil {
		h.logger.Warn("HandleHeartbeat: unmarshal failed", zap.Error(err))
		return
	}

	nodeID := heartbeat.Body.NodeID
	if nodeID == "" {
		if heartbeat.Header.Source != "" {
			nodeID = heartbeat.Header.Source
		} else {
			return
		}
	}

	// Update node status in registry
	h.registrySvc.UpdateNodeStatus(nodeID, heartbeat.Body.Status)

	// Log heartbeat information
	h.logger.Info("Received heartbeat",
		zap.String("node_id", nodeID),
		zap.String("status", heartbeat.Body.Status),
		zap.Int64("timestamp", heartbeat.Body.Timestamp),
		zap.Int("sequence", heartbeat.Body.Sequence),
		zap.Int("uptime_seconds", heartbeat.Body.UptimeSeconds),
		zap.String("version", heartbeat.Body.Version),
		zap.Float64("cpu_usage", heartbeat.Body.SystemMetrics.CPUUsage),
		zap.Float64("memory_usage", heartbeat.Body.SystemMetrics.MemoryUsage),
		zap.Int("total_devices", heartbeat.Body.DeviceSummary.TotalCount),
		zap.Int("online_devices", heartbeat.Body.DeviceSummary.OnlineCount),
		zap.Int("total_channels", heartbeat.Body.ChannelSummary.TotalCount),
		zap.Int("connected_channels", heartbeat.Body.ChannelSummary.ConnectedCount),
	)

	// Broadcast node status event
	if h.hub != nil {
		h.hub.BroadcastType(ws.EventNodeStatus, ws.NodeStatusPayload{
			NodeID:   nodeID,
			NodeName: nodeID, // Use nodeID as name if not provided
			Status:   heartbeat.Body.Status,
			LastSeen: heartbeat.Body.Timestamp / 1000, // Convert ms to seconds
		})
	}
}

func (h *NodeMQTTHandler) HandleUnregister(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Header map[string]interface{} `json:"header"`
		Body   struct {
			NodeID string `json:"node_id"`
		} `json:"body"`
	}
	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		return
	}
	nodeID := envelope.Body.NodeID
	if nodeID == "" {
		return
	}
	h.registrySvc.UpdateNodeStatus(nodeID, "offline")
	h.logger.Info("Node unregistered", zap.String("node_id", nodeID))
	if h.hub != nil {
		h.hub.BroadcastType(ws.EventNodeStatus, ws.NodeStatusPayload{
			NodeID: nodeID, Status: "offline",
		})
	}
}

// ==================== 设备消息处理器 ====================

func NewDeviceMQTTHandler(
	deviceSvc *services.DeviceService,
	pointSvc *services.PointService,
	hub *ws.Hub,
	logger *zap.Logger,
) *DeviceMQTTHandler {
	return &DeviceMQTTHandler{
		deviceSvc: deviceSvc,
		pointSvc:  pointSvc,
		hub:       hub,
		logger:    logger,
	}
}

func (h *DeviceMQTTHandler) HandleDeviceReport(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Header map[string]interface{} `json:"header"`
		Body   struct {
			NodeID  string                  `json:"node_id"`
			Devices []model.EdgeXDeviceInfo `json:"devices"`
		} `json:"body"`
	}
	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		h.logger.Error("HandleDeviceReport: unmarshal failed", zap.Error(err))
		return
	}

	nodeID := envelope.Body.NodeID
	// 如果 node_id 为空，尝试从 header 的 source 字段获取
	if nodeID == "" {
		if src, ok := envelope.Header["source"].(string); ok {
			nodeID = src
		}
	}
	// 如果 node_id 仍然为空，记录错误并返回
	if nodeID == "" {
		h.logger.Error("HandleDeviceReport: missing node_id", zap.String("topic", msg.Topic()))
		return
	}

	for _, dev := range envelope.Body.Devices {
		dev.LastSync = time.Now().Unix()
		if err := h.deviceSvc.UpsertDevice(nodeID, &dev); err != nil {
			h.logger.Error("HandleDeviceReport: upsert device failed",
				zap.String("device_id", dev.DeviceID), zap.Error(err))
		}

		// 从 Properties 中提取点位信息并同步
		if dev.Properties != nil && h.pointSvc != nil {
			h.syncPointsFromProperties(nodeID, dev.DeviceID, dev.Properties)
		}
	}

	if h.hub != nil {
		h.hub.BroadcastType(ws.EventDeviceSynced, map[string]interface{}{
			"node_id": nodeID,
			"count":   len(envelope.Body.Devices),
		})
	}
}

// syncPointsFromProperties 从设备属性中提取点位信息并同步
func (h *DeviceMQTTHandler) syncPointsFromProperties(nodeID, deviceID string, properties map[string]interface{}) {
	// 检查常见的点位数组字段名
	pointKeys := []string{"points", "measurements", "tags", "resources"}

	for _, key := range pointKeys {
		if pointsRaw, ok := properties[key]; ok {
			points, ok := pointsRaw.([]interface{})
			if !ok {
				continue
			}

			for _, p := range points {
				pMap, ok := p.(map[string]interface{})
				if !ok {
					continue
				}

				pt := h.convertToEdgeXPoint(deviceID, pMap)
				if pt.PointID != "" {
					if err := h.pointSvc.UpsertPoint(nodeID, deviceID, pt); err != nil {
						h.logger.Warn("HandleDeviceReport: upsert point failed",
							zap.String("point_id", pt.PointID), zap.Error(err))
					}
				}
			}

			h.logger.Info("HandleDeviceReport: synced points from properties",
				zap.String("node_id", nodeID),
				zap.String("device_id", deviceID),
				zap.String("key", key),
				zap.Int("count", len(points)))
			return
		}
	}
}

// convertToEdgeXPoint 将 map 转换为 EdgeXPointInfo
func (h *DeviceMQTTHandler) convertToEdgeXPoint(deviceID string, p map[string]interface{}) *model.EdgeXPointInfo {
	pt := &model.EdgeXPointInfo{
		DeviceID:   deviceID,
		Properties: make(map[string]interface{}),
	}

	// point_id (必需)
	if v, ok := p["point_id"].(string); ok && v != "" {
		pt.PointID = v
	} else if v, ok := p["name"].(string); ok && v != "" {
		pt.PointID = v
	} else if v, ok := p["id"].(string); ok && v != "" {
		pt.PointID = v
	}

	// point_name
	if v, ok := p["point_name"].(string); ok {
		pt.PointName = v
	} else if v, ok := p["description"].(string); ok {
		pt.PointName = v
	} else if v, ok := p["label"].(string); ok {
		pt.PointName = v
	}

	// data_type
	if v, ok := p["data_type"].(string); ok {
		pt.DataType = v
	} else if v, ok := p["value_type"].(string); ok {
		pt.DataType = v
	} else if v, ok := p["type"].(string); ok {
		pt.DataType = v
	}

	// point_type / read_write
	if v, ok := p["point_type"].(string); ok {
		pt.PointType = v
		pt.ReadWrite = v == "readwrite" || v == "R/W"
	} else if v, ok := p["access_mode"].(string); ok {
		pt.PointType = v
		pt.ReadWrite = v == "R/W" || v == "ReadWrite"
	} else if v, ok := p["accessMode"].(string); ok {
		pt.PointType = v
		pt.ReadWrite = v == "R/W" || v == "ReadWrite"
	}

	// units
	if v, ok := p["units"].(string); ok {
		pt.Units = v
	} else if v, ok := p["unit"].(string); ok {
		pt.Units = v
	}

	// description
	if v, ok := p["description"].(string); ok {
		pt.Description = v
	}

	// default_value
	pt.DefaultValue = p["default_value"]

	// 复制其他属性
	for k, v := range p {
		if k != "point_id" && k != "name" && k != "id" &&
			k != "point_name" && k != "description" && k != "label" &&
			k != "data_type" && k != "value_type" && k != "type" &&
			k != "point_type" && k != "access_mode" && k != "accessMode" &&
			k != "units" && k != "unit" && k != "default_value" {
			pt.Properties[k] = v
		}
	}

	return pt
}

// ==================== 设备上下线消息处理器 ====================

func (h *DeviceMQTTHandler) HandleDeviceOnline(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Header map[string]interface{} `json:"header"`
		Body   struct {
			NodeID     string                 `json:"node_id"`
			DeviceID   string                 `json:"device_id"`
			DeviceName string                 `json:"device_name"`
			OnlineTime int64                  `json:"online_time"`
			Status     string                 `json:"status"`
			Details    map[string]interface{} `json:"details"`
		} `json:"body"`
	}
	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		h.logger.Error("HandleDeviceOnline: unmarshal failed", zap.Error(err), zap.String("topic", msg.Topic()))
		return
	}

	nodeID := envelope.Body.NodeID
	deviceID := envelope.Body.DeviceID
	if nodeID == "" || deviceID == "" {
		h.logger.Warn("HandleDeviceOnline: missing node_id or device_id", zap.String("topic", msg.Topic()))
		return
	}

	if err := h.deviceSvc.UpdateDeviceStatus(nodeID, deviceID, "online"); err != nil {
		h.logger.Error("HandleDeviceOnline: update device status failed",
			zap.String("node_id", nodeID),
			zap.String("device_id", deviceID),
			zap.Error(err))
		return
	}

	h.logger.Info("Device online",
		zap.String("node_id", nodeID),
		zap.String("device_id", deviceID),
		zap.String("device_name", envelope.Body.DeviceName))

	if h.hub != nil {
		h.hub.BroadcastType(ws.EventDeviceOnline, map[string]interface{}{
			"node_id":     nodeID,
			"device_id":   deviceID,
			"device_name": envelope.Body.DeviceName,
			"status":      "online",
			"timestamp":   envelope.Body.OnlineTime,
		})
	}
}

func (h *DeviceMQTTHandler) HandleDeviceOffline(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Header map[string]interface{} `json:"header"`
		Body   struct {
			NodeID      string                 `json:"node_id"`
			DeviceID    string                 `json:"device_id"`
			DeviceName  string                 `json:"device_name"`
			OfflineTime int64                  `json:"offline_time"`
			Status      string                 `json:"status"`
			Reason      string                 `json:"reason"`
			Details     map[string]interface{} `json:"details"`
		} `json:"body"`
	}
	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		h.logger.Error("HandleDeviceOffline: unmarshal failed", zap.Error(err), zap.String("topic", msg.Topic()))
		return
	}

	nodeID := envelope.Body.NodeID
	deviceID := envelope.Body.DeviceID
	if nodeID == "" || deviceID == "" {
		h.logger.Warn("HandleDeviceOffline: missing node_id or device_id", zap.String("topic", msg.Topic()))
		return
	}

	if err := h.deviceSvc.UpdateDeviceStatus(nodeID, deviceID, "offline"); err != nil {
		h.logger.Error("HandleDeviceOffline: update device status failed",
			zap.String("node_id", nodeID),
			zap.String("device_id", deviceID),
			zap.Error(err))
		return
	}

	h.logger.Info("Device offline",
		zap.String("node_id", nodeID),
		zap.String("device_id", deviceID),
		zap.String("device_name", envelope.Body.DeviceName),
		zap.String("reason", envelope.Body.Reason))

	if h.hub != nil {
		h.hub.BroadcastType(ws.EventDeviceOffline, map[string]interface{}{
			"node_id":     nodeID,
			"device_id":   deviceID,
			"device_name": envelope.Body.DeviceName,
			"status":      "offline",
			"reason":      envelope.Body.Reason,
			"timestamp":   envelope.Body.OfflineTime,
		})
	}
}

// ==================== 点位消息处理器 ====================

func NewPointMQTTHandler(
	pointSvc *services.PointService,
	deviceSvc *services.DeviceService,
	hub *ws.Hub,
	logger *zap.Logger,
) *PointMQTTHandler {
	return &PointMQTTHandler{pointSvc: pointSvc, deviceSvc: deviceSvc, hub: hub, logger: logger}
}

func (h *PointMQTTHandler) HandlePointReport(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Header map[string]interface{} `json:"header"`
		Body   struct {
			NodeID   string `json:"node_id"`
			DeviceID string `json:"device_id"`
			Points   []struct {
				PointID   string `json:"point_id"`
				PointName string `json:"point_name"`
				Address   string `json:"address"`
				DataType  string `json:"data_type"`
				RW        string `json:"rw"`
				Unit      string `json:"unit"`
			} `json:"points"`
		} `json:"body"`
	}
	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		h.logger.Error("HandlePointReport: unmarshal failed", zap.Error(err))
		return
	}

	nodeID := envelope.Body.NodeID
	// 如果 node_id 为空，尝试从 header 的 source 字段获取
	if nodeID == "" {
		if src, ok := envelope.Header["source"].(string); ok {
			nodeID = src
		}
	}
	// 如果 node_id 仍然为空，记录错误并返回
	if nodeID == "" {
		h.logger.Error("HandlePointReport: missing node_id", zap.String("topic", msg.Topic()))
		return
	}

	deviceID := envelope.Body.DeviceID
	if deviceID == "" {
		h.logger.Error("HandlePointReport: missing device_id", zap.String("topic", msg.Topic()))
		return
	}

	for _, pt := range envelope.Body.Points {
		point := model.EdgeXPointInfo{
			PointID:   pt.PointID,
			PointName: pt.PointName,
			DeviceID:  deviceID,
			DataType:  pt.DataType,
			ReadWrite: pt.RW == "RW",
			Units:     pt.Unit,
			LastSync:  time.Now().Unix(),
		}
		if err := h.pointSvc.UpsertPoint(nodeID, deviceID, &point); err != nil {
			h.logger.Error("HandlePointReport: upsert failed",
				zap.String("point_id", pt.PointID), zap.Error(err))
		}
	}

	h.logger.Info("HandlePointReport: points synced",
		zap.String("node_id", nodeID),
		zap.String("device_id", deviceID),
		zap.Int("count", len(envelope.Body.Points)))

	if h.hub != nil {
		h.hub.BroadcastType(ws.EventPointSynced, map[string]interface{}{
			"node_id":   nodeID,
			"device_id": deviceID,
			"count":     len(envelope.Body.Points),
		})
	}
}

// HandlePointSync 处理点位全量同步消息
// Topic: edgex/points/{node_id}/{device_id} (对应订阅 edgex/points/+/+)
func (h *PointMQTTHandler) HandlePointSync(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Header map[string]interface{} `json:"header"`
		Body   struct {
			NodeID   string                 `json:"node_id"`
			DeviceID string                 `json:"device_id"`
			Points   []model.EdgeXPointInfo `json:"points"`
		} `json:"body"`
	}
	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		h.logger.Error("HandlePointSync: unmarshal failed", zap.Error(err), zap.String("topic", msg.Topic()))
		return
	}

	nodeID := envelope.Body.NodeID
	deviceID := envelope.Body.DeviceID
	if nodeID == "" || deviceID == "" {
		h.logger.Warn("HandlePointSync: missing node_id or device_id", zap.String("topic", msg.Topic()))
		return
	}

	// 全量同步：更新所有点位（Upsert 操作）
	if h.pointSvc != nil {
		for i := range envelope.Body.Points {
			pt := &envelope.Body.Points[i]
			pt.LastSync = time.Now().Unix()
			if err := h.pointSvc.UpsertPoint(nodeID, deviceID, pt); err != nil {
				h.logger.Error("HandlePointSync: upsert failed",
					zap.String("point_id", pt.PointID), zap.Error(err))
			}
		}
	}

	h.logger.Info("HandlePointSync: full sync completed",
		zap.String("node_id", nodeID),
		zap.String("device_id", deviceID),
		zap.Int("count", len(envelope.Body.Points)))

	if h.hub != nil {
		h.hub.BroadcastType(ws.EventPointSynced, map[string]interface{}{
			"node_id":   nodeID,
			"device_id": deviceID,
			"count":     len(envelope.Body.Points),
			"sync_type": "full",
		})
	}
}

func (h *PointMQTTHandler) HandleRealtimeData(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Header map[string]interface{} `json:"header"`
		Body   struct {
			NodeID         string                 `json:"node_id"`
			DeviceID       string                 `json:"device_id"`
			Points         map[string]interface{} `json:"points"`
			Quality        string                 `json:"quality"`
			Timestamp      int64                  `json:"timestamp"`
			IsFullSnapshot bool                   `json:"is_full_snapshot"`
		} `json:"body"`
	}
	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		h.logger.Error("HandleRealtimeData: unmarshal failed", zap.Error(err))
		return
	}

	nodeID := envelope.Body.NodeID
	// 如果 node_id 为空，尝试从 header 的 source 字段获取
	if nodeID == "" {
		if src, ok := envelope.Header["source"].(string); ok {
			nodeID = src
		}
	}
	// 如果 node_id 仍然为空，记录错误并返回
	if nodeID == "" {
		h.logger.Error("HandleRealtimeData: missing node_id", zap.String("topic", msg.Topic()))
		return
	}

	deviceID := envelope.Body.DeviceID
	if deviceID == "" {
		h.logger.Error("HandleRealtimeData: missing device_id", zap.String("topic", msg.Topic()))
		return
	}

	ts := envelope.Body.Timestamp
	if ts == 0 {
		ts = time.Now().Unix()
	}
	isFull := envelope.Body.IsFullSnapshot

	// 存储点位数据（全量替换或差量合并）
	if h.pointSvc != nil {
		h.pointSvc.SaveSnapshot(nodeID, deviceID,
			envelope.Body.Points, envelope.Body.Quality, ts, isFull)
	}

	// 当收到实时数据时，将设备状态设置为在线
	if h.deviceSvc != nil {
		err := h.deviceSvc.UpdateDeviceStatus(nodeID, deviceID, "online")
		if err != nil {
			h.logger.Warn("HandleRealtimeData: update device status failed",
				zap.String("node_id", nodeID),
				zap.String("device_id", deviceID),
				zap.Error(err))
		}
	}

	if h.hub != nil {
		h.hub.BroadcastType(ws.EventDataUpdate, map[string]interface{}{
			"node_id":          nodeID,
			"device_id":        deviceID,
			"points":           envelope.Body.Points,
			"quality":          envelope.Body.Quality,
			"timestamp":        ts,
			"is_full_snapshot": isFull,
		})
	}
}

// ==================== 控制命令处理器 ====================

func NewControlMQTTHandler(
	controlSvc *services.ControlService,
	hub *ws.Hub,
	logger *zap.Logger,
) *ControlMQTTHandler {
	return &ControlMQTTHandler{controlSvc: controlSvc, hub: hub, logger: logger}
}

func (h *ControlMQTTHandler) HandleCommandResponse(_ pahomqtt.Client, msg pahomqtt.Message) {
	h.logger.Info("HandleCommandResponse: received message", zap.String("topic", msg.Topic()), zap.String("payload", string(msg.Payload())))
	var envelope struct {
		Header struct {
			MessageID   string `json:"message_id"`
			Timestamp   int64  `json:"timestamp"`
			Source      string `json:"source"`
			Destination string `json:"destination"`
			MessageType string `json:"message_type"`
			Version     string `json:"version"`
			RequestID   string `json:"request_id"`
		} `json:"header"`
		Body struct {
			Success bool                   `json:"success"`
			Message string                 `json:"message"`
			Data    map[string]interface{} `json:"data"`
		} `json:"body"`
	}
	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		h.logger.Error("HandleCommandResponse: unmarshal failed", zap.Error(err))
		return
	}

	// 从 header 中获取 request_id
	requestID := envelope.Header.RequestID
	if requestID == "" {
		h.logger.Warn("HandleCommandResponse: missing request_id in header, message may not be processed correctly")
	}

	h.logger.Info("HandleCommandResponse: parsed envelope",
		zap.String("request_id", requestID),
		zap.Bool("success", envelope.Body.Success),
		zap.String("message", envelope.Body.Message))

	// 更新命令状态
	status := "success"
	if !envelope.Body.Success {
		status = "error"
	}
	h.controlSvc.UpdateCommandStatus(requestID, status, envelope.Body.Message)

	h.logger.Info("HandleCommandResponse: broadcasting WebSocket event",
		zap.String("request_id", requestID),
		zap.Bool("hub_not_nil", h.hub != nil))

	if h.hub != nil {
		// 从 data 中提取 device_id 和 point_id
		deviceID := ""
		pointID := ""
		if envelope.Body.Data != nil {
			if d, ok := envelope.Body.Data["device_id"].(string); ok {
				deviceID = d
			}
			if p, ok := envelope.Body.Data["point_id"].(string); ok {
				pointID = p
			}
		}
		h.hub.BroadcastType(ws.EventCommandResp, map[string]interface{}{
			"request_id": requestID,
			"node_id":    envelope.Header.Source,
			"device_id":  deviceID,
			"point_id":   pointID,
			"success":    envelope.Body.Success,
			"error":      envelope.Body.Message,
			"timestamp":  time.Now().UnixMilli(),
		})
		h.logger.Info("HandleCommandResponse: WebSocket event broadcasted",
			zap.String("request_id", requestID))
	}

}
