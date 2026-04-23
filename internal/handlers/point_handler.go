package handlers

import (
	"encoding/json"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/model"
	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/ws"
)

// PointHandler 处理物模型点位上报和实时数据（差量Merge）
type PointHandler struct {
	pointSvc *services.PointService
	hub      *ws.Hub
	logger   *zap.Logger
}

// NewPointHandler 创建点位消息处理器
func NewPointHandler(
	pointSvc *services.PointService,
	hub *ws.Hub,
	logger *zap.Logger,
) *PointHandler {
	return &PointHandler{
		pointSvc: pointSvc,
		hub:      hub,
		logger:   logger,
	}
}

// HandlePointReport 处理点位元数据上报（物模型定义）
// Topic: edgex/points/report
// Payload:
//
//	{
//	  "body": {
//	    "node_id": "<node_id>",
//	    "device_id": "<device_id>",
//	    "points": [EdgeXPointInfo, ...]
//	  }
//	}

// mqttPointRaw 捕获 MQTT 点位原始数据（字段名与 MQTT payload 对应）
type mqttPointRaw struct {
	PointID    string `json:"point_id"`
	PointName  string `json:"point_name"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	DataType   string `json:"data_type"`
	Rw         string `json:"rw"`        // R/RW/W
	Unit       string `json:"unit"`
	Address    string `json:"address"`
	ChannelID  string `json:"channel_id"`
}

func (h *PointHandler) HandlePointReport(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Header struct {
			Source string `json:"source"`
		} `json:"header"`
		Body struct {
			NodeID   string            `json:"node_id"`
			DeviceID string            `json:"device_id"`
			Points   []mqttPointRaw    `json:"points"`
		} `json:"body"`
	}

	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		h.logger.Error("PointHandler.HandlePointReport: unmarshal failed",
			zap.Error(err), zap.String("topic", msg.Topic()))
		return
	}

	nodeID := envelope.Body.NodeID
	if nodeID == "" {
		nodeID = envelope.Header.Source
	}
	if nodeID == "" {
		h.logger.Warn("PointHandler.HandlePointReport: missing node_id")
		return
	}

	saved := 0
	for _, raw := range envelope.Body.Points {
		p := model.EdgeXPointInfo{
			PointID:     raw.PointID,
			PointName:   raw.PointName,
			DeviceID:    raw.DeviceID,
			ServiceName: "edgex-service",
			ProfileName: "default",
			DataType:    raw.DataType,
			Units:       raw.Unit,
			Description: "Auto-discovered point",
			Properties:  make(map[string]interface{}),
		}

		// 根据 rw 字段转换 ReadWrite 和 PointType
		switch raw.Rw {
		case "R":
			p.ReadWrite = false
			p.PointType = "read"
		case "W":
			p.ReadWrite = true
			p.PointType = "write"
		default: // "RW" or empty
			p.ReadWrite = true
			p.PointType = "readwrite"
		}

		if err := h.pointSvc.SaveMetaWithNode(nodeID, &p); err != nil {
			h.logger.Error("PointHandler.HandlePointReport: save meta failed",
				zap.String("node_id", nodeID),
				zap.String("point_id", p.PointID),
				zap.Error(err))
			continue
		}
		saved++
	}

	h.logger.Info("Points meta synced",
		zap.String("node_id", nodeID),
		zap.Int("total", len(envelope.Body.Points)),
		zap.Int("saved", saved))

	// 使用第一个点的 device_id（如果 body 的 device_id 为空）
	deviceID := envelope.Body.DeviceID
	if deviceID == "" && len(envelope.Body.Points) > 0 {
		deviceID = envelope.Body.Points[0].DeviceID
	}

	if h.hub != nil {
		h.hub.BroadcastType(ws.EventPointReport, map[string]interface{}{
			"node_id":   nodeID,
			"device_id": deviceID,
			"count":     saved,
		})
	}
}

// HandleRealtimeData 处理实时数据流（支持全量/差量Merge）
// Topic: edgex/data/stream
// Payload:
//
//	{
//	  "header": {"message_type": "data_full|data_delta"},
//	  "body": {
//	    "node_id":          "<node_id>",
//	    "device_id":        "<device_id>",
//	    "points":           {"pointId": value, ...},
//	    "timestamp":        <unix_ms>,
//	    "quality":          "good|bad|uncertain",
//	    "is_full_snapshot": true|false
//	  }
//	}
//
// 差量Merge逻辑：
//   - 若 is_full_snapshot=true 或 设备无快照缓存 → 全量保存，建立快照
//   - 否则 → 从 BoltDB 读取已有快照，合并差量点位，写回
func (h *PointHandler) HandleRealtimeData(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Header struct {
			MessageType string `json:"message_type"`
		} `json:"header"`
		Body struct {
			NodeID         string                 `json:"node_id"`
			DeviceID       string                 `json:"device_id"`
			Points         map[string]interface{} `json:"points"`
			Timestamp      int64                  `json:"timestamp"`
			Quality        string                 `json:"quality"`
			IsFullSnapshot bool                   `json:"is_full_snapshot"`
		} `json:"body"`
	}

	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		h.logger.Error("PointHandler.HandleRealtimeData: unmarshal failed",
			zap.Error(err), zap.String("topic", msg.Topic()))
		return
	}

	body := envelope.Body
	if body.NodeID == "" || body.DeviceID == "" {
		h.logger.Warn("PointHandler.HandleRealtimeData: missing node_id or device_id")
		return
	}
	if body.Timestamp == 0 {
		body.Timestamp = time.Now().UnixMilli()
	}
	if body.Quality == "" {
		body.Quality = "good"
	}

	// 差量Merge策略：
	// 1. 设备无快照 → 强制全量（建立基线）
	// 2. is_full_snapshot=true → 全量替换
	// 3. 其他 → 差量Merge（只更新出现的点位）
	isFull := body.IsFullSnapshot || !h.pointSvc.HasCache(body.NodeID, body.DeviceID)

	if err := h.pointSvc.SaveSnapshot(
		body.NodeID, body.DeviceID,
		body.Points, body.Quality, body.Timestamp,
		isFull,
	); err != nil {
		h.logger.Error("PointHandler.HandleRealtimeData: save snapshot failed",
			zap.String("node_id", body.NodeID),
			zap.String("device_id", body.DeviceID),
			zap.Error(err))
		return
	}

	h.logger.Debug("Realtime data processed",
		zap.String("node_id", body.NodeID),
		zap.String("device_id", body.DeviceID),
		zap.Int("point_count", len(body.Points)),
		zap.Bool("is_full", isFull))

	if h.hub != nil {
		h.hub.BroadcastType(ws.EventDataUpdate, ws.DataUpdatePayload{
			NodeID:         body.NodeID,
			DeviceID:       body.DeviceID,
			Points:         body.Points,
			Timestamp:      body.Timestamp,
			Quality:        body.Quality,
			IsFullSnapshot: isFull,
		})
	}
}
