package handlers

import (
	"encoding/json"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/model"
	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/ws"
)

// DeviceHandler 处理设备列表同步消息
type DeviceHandler struct {
	deviceSvc *services.DeviceService
	hub       *ws.Hub
	logger    *zap.Logger
}

// NewDeviceHandler 创建设备消息处理器
func NewDeviceHandler(
	deviceSvc *services.DeviceService,
	hub *ws.Hub,
	logger *zap.Logger,
) *DeviceHandler {
	return &DeviceHandler{
		deviceSvc: deviceSvc,
		hub:       hub,
		logger:    logger,
	}
}

// HandleDeviceReport 处理设备列表上报
// Topic: edgeCore/devices/report
// Payload:
//
//	{
//	  "header": {"source": "<node_id>", ...},
//	  "body": {
//	    "node_id": "<node_id>",
//	    "devices": [EdgeCoreDeviceInfo, ...]
//	  }
//	}
func (h *DeviceHandler) HandleDeviceReport(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Header struct {
			Source string `json:"source"`
		} `json:"header"`
		Body struct {
			NodeID  string                  `json:"node_id"`
			Devices []model.EdgeCoreDeviceInfo `json:"devices"`
		} `json:"body"`
	}

	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		h.logger.Error("DeviceHandler.HandleDeviceReport: unmarshal failed",
			zap.Error(err), zap.String("topic", msg.Topic()))
		return
	}

	nodeID := envelope.Body.NodeID
	if nodeID == "" {
		nodeID = envelope.Header.Source
	}
	if nodeID == "" {
		h.logger.Warn("DeviceHandler.HandleDeviceReport: missing node_id")
		return
	}

	upserted, removed, err := h.deviceSvc.ReconcileDevices(nodeID, envelope.Body.Devices)
	if err != nil {
		h.logger.Error("DeviceHandler.HandleDeviceReport: reconcile failed",
			zap.String("node_id", nodeID), zap.Error(err))
		return
	}

	h.logger.Info("Devices synced",
		zap.String("node_id", nodeID),
		zap.Int("reported", len(envelope.Body.Devices)),
		zap.Int("upserted", upserted),
		zap.Int("removed", removed))

	if h.hub != nil {
		h.hub.BroadcastType(ws.EventDeviceSynced, map[string]interface{}{
			"node_id":  nodeID,
			"count":    upserted,
			"total":    len(envelope.Body.Devices),
			"removed":  removed,
		})
	}
}
