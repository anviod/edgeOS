package ean

import (
	"encoding/json"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/model"
	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/ws"
)

// V1NATSDataPlane 桥接 V1 NATS 数据面 Subject（edgex.*）到 V1 服务。
// 对齐改造指南 §2.7 / §7.2 OS-23：V1 设备清单须同时订阅 MQTT `edgex/devices/report`
// 与 NATS `edgex.devices.report`（双传输对称）。MQTT 侧由 messaging.Manager 订阅，
// 本桥接在 EAN NATS 传输层上订阅同语义的 NATS subject，并复用同一套 V1 服务。
//
// Subject 采用 V1 点分形式（edgex.devices.report 等），通配符 * / > 与 MQTT + / # 对应。
type V1NATSDataPlane struct {
	registrySvc *services.RegistryService
	deviceSvc   *services.DeviceService
	pointSvc    *services.PointService
	alertSvc    *services.AlertService
	hub         *ws.Hub
	logger      *zap.Logger
}

// NewV1NATSDataPlane 创建 V1 NATS 数据面桥接器。
func NewV1NATSDataPlane(
	registrySvc *services.RegistryService,
	deviceSvc *services.DeviceService,
	pointSvc *services.PointService,
	alertSvc *services.AlertService,
	hub *ws.Hub,
	logger *zap.Logger,
) *V1NATSDataPlane {
	return &V1NATSDataPlane{
		registrySvc: registrySvc,
		deviceSvc:   deviceSvc,
		pointSvc:    pointSvc,
		alertSvc:    alertSvc,
		hub:         hub,
		logger:      logger.Named("v1-nats"),
	}
}

// v1SubjectBindings 返回 V1 NATS 数据面订阅绑定（subject -> handler）。
// MQTT 通配符 + / # 在 NATS 中分别对应 * / >，其余 token 点分。
func (p *V1NATSDataPlane) v1SubjectBindings() []struct {
	subject string
	handler MessageHandler
} {
	return []struct {
		subject string
		handler MessageHandler
	}{
		{"edgex.devices.report", p.handleDeviceReport},
		{"edgex.devices.*.*.online", p.handleDeviceOnline},
		{"edgex.devices.*.*.offline", p.handleDeviceOffline},
		{"edgex.points.report", p.handlePointReport},
		{"edgex.points.*.*", p.handlePointSync},
		{"edgex.data.*.*", p.handleRealtimeData},
		{"edgex.events.alert", p.handleAlert},
		{"edgex.events.error", p.handleAlert},
		{"edgex.events.info", p.handleAlert},
		{"edgex.nodes.register", p.handleNodeRegister},
		{"edgex.nodes.*.heartbeat", p.handleHeartbeat},
		{"edgex.nodes.*.status", p.handleHeartbeat},
	}
}

// Subscribe 在给定 NATS 传输层上注册 V1 数据面订阅。
// 传输层未连接时由 NATS Transport 本地登记，Connect/Reconnect 时补订。
func (p *V1NATSDataPlane) Subscribe(t interface {
	Subscribe(string, MessageHandler) error
}) error {
	for _, b := range p.v1SubjectBindings() {
		if err := t.Subscribe(b.subject, b.handler); err != nil {
			p.logger.Warn("subscribe v1 nats subject failed",
				zap.String("subject", b.subject), zap.Error(err))
		} else {
			p.logger.Info("v1 nats subject subscribed", zap.String("subject", b.subject))
		}
	}
	return nil
}

// ==================== 消息解析辅助 ====================

// parseHeaderBody 解析 V1 信封 {header, body}，返回 header 与 body bytes。
func parseV1Envelope(payload []byte) (map[string]interface{}, json.RawMessage, error) {
	var env struct {
		Header map[string]interface{} `json:"header"`
		Body   json.RawMessage        `json:"body"`
	}
	if err := json.Unmarshal(payload, &env); err != nil {
		return nil, nil, err
	}
	return env.Header, env.Body, nil
}

// sourceFromHeader 从 header.source 提取 node_id（body.node_id 缺失时兜底）。
func sourceFromHeader(header map[string]interface{}) string {
	if src, ok := header["source"].(string); ok {
		return src
	}
	return ""
}

// ==================== 节点处理 ====================

// handleNodeRegister 处理 edgex.nodes.register（V1 NATS 节点注册）。
func (p *V1NATSDataPlane) handleNodeRegister(topic string, payload []byte, transport string) {
	header, body, err := parseV1Envelope(payload)
	if err != nil {
		p.logger.Error("handleNodeRegister: unmarshal failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	var node model.EdgeXNodeInfo
	if err := json.Unmarshal(body, &node); err != nil {
		p.logger.Error("handleNodeRegister: parse body failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	if node.NodeID == "" {
		node.NodeID = sourceFromHeader(header)
	}
	if node.NodeID == "" {
		p.logger.Warn("handleNodeRegister: missing node_id", zap.String("subject", topic))
		return
	}
	node.Status = "online"
	if err := p.registrySvc.UpsertNode(&node); err != nil {
		p.logger.Error("handleNodeRegister: upsert failed", zap.Error(err), zap.String("node_id", node.NodeID))
		return
	}
	p.logger.Info("v1 nats node registered", zap.String("node_id", node.NodeID))
	if p.hub != nil {
		p.hub.BroadcastType(ws.EventNodeStatus, ws.NodeStatusPayload{
			NodeID:   node.NodeID,
			NodeName: node.NodeName,
			Status:   "online",
			LastSeen: node.LastSeen,
		})
	}
}

// handleHeartbeat 处理 edgex.nodes.*.heartbeat / edgex.nodes.*.status。
func (p *V1NATSDataPlane) handleHeartbeat(topic string, payload []byte, transport string) {
	header, body, err := parseV1Envelope(payload)
	if err != nil {
		p.logger.Warn("handleHeartbeat: unmarshal failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	var envelope struct {
		NodeID string `json:"node_id"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		p.logger.Warn("handleHeartbeat: parse body failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	nodeID := envelope.NodeID
	if nodeID == "" {
		nodeID = sourceFromHeader(header)
	}
	if nodeID == "" {
		// 从 subject 尾部取 node token，如 edgex.nodes.node-001.heartbeat
		parts := strings.Split(topic, ".")
		if len(parts) >= 3 && parts[1] == "nodes" {
			nodeID = parts[2]
		}
	}
	if nodeID == "" {
		return
	}
	if err := p.registrySvc.UpdateNodeStatus(nodeID, "online"); err != nil {
		p.logger.Debug("handleHeartbeat: update status failed", zap.Error(err), zap.String("node_id", nodeID))
	}
	if p.hub != nil {
		p.hub.BroadcastType(ws.EventNodeStatus, ws.NodeStatusPayload{
			NodeID:   nodeID,
			Status:   "online",
			LastSeen: time.Now().Unix(),
		})
	}
}

// ==================== 设备处理 ====================

// handleDeviceReport 处理 edgex.devices.report（V1 NATS 设备清单对账，OS-23 核心）。
func (p *V1NATSDataPlane) handleDeviceReport(topic string, payload []byte, transport string) {
	header, body, err := parseV1Envelope(payload)
	if err != nil {
		p.logger.Error("handleDeviceReport: unmarshal failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	var envelope struct {
		NodeID  string                  `json:"node_id"`
		Devices []model.EdgeXDeviceInfo `json:"devices"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		p.logger.Error("handleDeviceReport: parse body failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	nodeID := envelope.NodeID
	if nodeID == "" {
		nodeID = sourceFromHeader(header)
	}
	if nodeID == "" {
		p.logger.Error("handleDeviceReport: missing node_id", zap.String("subject", topic))
		return
	}

	upserted, removed, err := p.deviceSvc.ReconcileDevices(nodeID, envelope.Devices)
	if err != nil {
		p.logger.Error("handleDeviceReport: reconcile failed", zap.String("node_id", nodeID), zap.Error(err))
		return
	}
	if p.registrySvc != nil {
		if err := p.registrySvc.EnsureNodeOnline(nodeID, nodeID, "nats"); err != nil {
			p.logger.Warn("handleDeviceReport: ensure node online failed", zap.String("node_id", nodeID), zap.Error(err))
		}
	}
	p.logger.Info("v1 nats device report reconciled",
		zap.String("node_id", nodeID),
		zap.Int("reported", len(envelope.Devices)),
		zap.Int("upserted", upserted),
		zap.Int("removed", removed))
	if p.hub != nil {
		p.hub.BroadcastType(ws.EventDeviceSynced, map[string]interface{}{
			"node_id": nodeID,
			"count":   upserted,
			"removed": removed,
		})
	}
}

// handleDeviceOnline 处理 edgex.devices.*.*.online。
func (p *V1NATSDataPlane) handleDeviceOnline(topic string, payload []byte, transport string) {
	p.handleDeviceStatus(topic, payload, "online")
}

// handleDeviceOffline 处理 edgex.devices.*.*.offline。
func (p *V1NATSDataPlane) handleDeviceOffline(topic string, payload []byte, transport string) {
	p.handleDeviceStatus(topic, payload, "offline")
}

func (p *V1NATSDataPlane) handleDeviceStatus(topic string, payload []byte, status string) {
	_, body, err := parseV1Envelope(payload)
	if err != nil {
		p.logger.Warn("handleDeviceStatus: unmarshal failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	var envelope struct {
		NodeID      string `json:"node_id"`
		DeviceID    string `json:"device_id"`
		DeviceName  string `json:"device_name"`
		Reason      string `json:"reason"`
		StatusTime  int64  `json:"online_time"`
		OfflineTime int64  `json:"offline_time"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		p.logger.Warn("handleDeviceStatus: parse body failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	if envelope.NodeID == "" || envelope.DeviceID == "" {
		p.logger.Warn("handleDeviceStatus: missing node_id or device_id", zap.String("subject", topic))
		return
	}
	if err := p.deviceSvc.UpdateDeviceStatus(envelope.NodeID, envelope.DeviceID, status); err != nil {
		p.logger.Error("handleDeviceStatus: update status failed",
			zap.String("node_id", envelope.NodeID), zap.String("device_id", envelope.DeviceID), zap.Error(err))
		return
	}
	p.logger.Info("v1 nats device status",
		zap.String("node_id", envelope.NodeID),
		zap.String("device_id", envelope.DeviceID),
		zap.String("status", status))
	if p.hub != nil {
		evt := ws.EventDeviceOnline
		payloadMap := map[string]interface{}{"node_id": envelope.NodeID, "device_id": envelope.DeviceID, "status": status}
		if status == "offline" {
			evt = ws.EventDeviceOffline
			payloadMap["reason"] = envelope.Reason
		}
		p.hub.BroadcastType(evt, payloadMap)
	}
}

// ==================== 点位处理 ====================

// handlePointReport 处理 edgex.points.report（点位元数据上报）。
func (p *V1NATSDataPlane) handlePointReport(topic string, payload []byte, transport string) {
	header, body, err := parseV1Envelope(payload)
	if err != nil {
		p.logger.Error("handlePointReport: unmarshal failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	var envelope struct {
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
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		p.logger.Error("handlePointReport: parse body failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	nodeID := envelope.NodeID
	if nodeID == "" {
		nodeID = sourceFromHeader(header)
	}
	if nodeID == "" || envelope.DeviceID == "" {
		p.logger.Warn("handlePointReport: missing node_id or device_id", zap.String("subject", topic))
		return
	}
	for _, pt := range envelope.Points {
		point := model.EdgeXPointInfo{
			PointID:   pt.PointID,
			PointName: pt.PointName,
			DeviceID:  envelope.DeviceID,
			DataType:  pt.DataType,
			ReadWrite: pt.RW == "RW",
			Units:     pt.Unit,
			LastSync:  time.Now().Unix(),
		}
		if err := p.pointSvc.UpsertPoint(nodeID, envelope.DeviceID, &point); err != nil {
			p.logger.Error("handlePointReport: upsert failed", zap.String("point_id", pt.PointID), zap.Error(err))
		}
	}
	p.logger.Info("v1 nats point report synced",
		zap.String("node_id", nodeID), zap.String("device_id", envelope.DeviceID),
		zap.Int("count", len(envelope.Points)))
	if p.hub != nil {
		p.hub.BroadcastType(ws.EventPointSynced, map[string]interface{}{
			"node_id": nodeID, "device_id": envelope.DeviceID, "count": len(envelope.Points),
		})
	}
}

// handlePointSync 处理 edgex.points.{node}.{device}（点位全量同步）。
func (p *V1NATSDataPlane) handlePointSync(topic string, payload []byte, transport string) {
	header, body, err := parseV1Envelope(payload)
	if err != nil {
		p.logger.Error("handlePointSync: unmarshal failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	var envelope struct {
		NodeID   string                 `json:"node_id"`
		DeviceID string                 `json:"device_id"`
		Points   []model.EdgeXPointInfo `json:"points"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		p.logger.Error("handlePointSync: parse body failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	nodeID := envelope.NodeID
	if nodeID == "" {
		nodeID = sourceFromHeader(header)
	}
	if nodeID == "" || envelope.DeviceID == "" {
		p.logger.Warn("handlePointSync: missing node_id or device_id", zap.String("subject", topic))
		return
	}
	for i := range envelope.Points {
		pt := &envelope.Points[i]
		pt.DeviceID = envelope.DeviceID
		pt.LastSync = time.Now().Unix()
		if err := p.pointSvc.UpsertPoint(nodeID, envelope.DeviceID, pt); err != nil {
			p.logger.Error("handlePointSync: upsert failed", zap.String("point_id", pt.PointID), zap.Error(err))
		}
	}
	p.logger.Info("v1 nats point sync completed",
		zap.String("node_id", nodeID), zap.String("device_id", envelope.DeviceID),
		zap.Int("count", len(envelope.Points)))
	if p.hub != nil {
		p.hub.BroadcastType(ws.EventPointSynced, map[string]interface{}{
			"node_id": nodeID, "device_id": envelope.DeviceID, "count": len(envelope.Points), "sync_type": "full",
		})
	}
}

// handleRealtimeData 处理 edgex.data.{node}.{device}（实时数据）。
func (p *V1NATSDataPlane) handleRealtimeData(topic string, payload []byte, transport string) {
	header, body, err := parseV1Envelope(payload)
	if err != nil {
		p.logger.Error("handleRealtimeData: unmarshal failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	var envelope struct {
		NodeID         string                 `json:"node_id"`
		DeviceID       string                 `json:"device_id"`
		Points         map[string]interface{} `json:"points"`
		Quality        string                 `json:"quality"`
		Timestamp      int64                  `json:"timestamp"`
		IsFullSnapshot bool                   `json:"is_full_snapshot"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		p.logger.Error("handleRealtimeData: parse body failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	nodeID := envelope.NodeID
	if nodeID == "" {
		nodeID = sourceFromHeader(header)
	}
	if nodeID == "" || envelope.DeviceID == "" {
		p.logger.Warn("handleRealtimeData: missing node_id or device_id", zap.String("subject", topic))
		return
	}
	ts := envelope.Timestamp
	if ts == 0 {
		ts = time.Now().Unix()
	}
	if p.pointSvc != nil {
		p.pointSvc.SaveSnapshot(nodeID, envelope.DeviceID, envelope.Points, envelope.Quality, ts, envelope.IsFullSnapshot)
	}
	if p.deviceSvc != nil {
		if err := p.deviceSvc.UpdateDeviceStatus(nodeID, envelope.DeviceID, "online"); err != nil {
			p.logger.Warn("handleRealtimeData: update device status failed",
				zap.String("node_id", nodeID), zap.String("device_id", envelope.DeviceID), zap.Error(err))
		}
	}
	if p.hub != nil {
		p.hub.BroadcastType(ws.EventDataUpdate, map[string]interface{}{
			"node_id":          nodeID,
			"device_id":        envelope.DeviceID,
			"points":           envelope.Points,
			"quality":          envelope.Quality,
			"timestamp":        ts,
			"is_full_snapshot": envelope.IsFullSnapshot,
		})
	}
}

// ==================== 告警处理 ====================

// handleAlert 处理 edgex.events.alert / error / info。
func (p *V1NATSDataPlane) handleAlert(topic string, payload []byte, transport string) {
	_, body, err := parseV1Envelope(payload)
	if err != nil {
		p.logger.Error("handleAlert: unmarshal failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	var alert model.AlertInfo
	if err := json.Unmarshal(body, &alert); err != nil {
		p.logger.Error("handleAlert: parse body failed", zap.String("subject", topic), zap.Error(err))
		return
	}
	if err := p.alertSvc.AddAlert(&alert); err != nil {
		p.logger.Error("handleAlert: add alert failed", zap.Error(err))
		return
	}
	p.logger.Info("v1 nats alert received", zap.String("alert_id", alert.ID), zap.String("level", alert.Level))
	if p.hub != nil {
		p.hub.BroadcastType(ws.EventAlert, &alert)
	}
}
