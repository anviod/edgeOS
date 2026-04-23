package handlers

import (
	"encoding/json"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/ws"
)

// ControlHandler 处理命令响应消息，追踪命令执行状态
type ControlHandler struct {
	controlSvc *services.ControlService
	hub        *ws.Hub
	logger     *zap.Logger
}

// NewControlHandler 创建控制命令处理器
func NewControlHandler(
	controlSvc *services.ControlService,
	hub *ws.Hub,
	logger *zap.Logger,
) *ControlHandler {
	return &ControlHandler{
		controlSvc: controlSvc,
		hub:        hub,
		logger:     logger,
	}
}

// commandResponseBody 命令响应体
type commandResponseBody struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"` // success | error | timeout
	Error     string `json:"error,omitempty"`
	NodeID    string `json:"node_id,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
	PointID   string `json:"point_id,omitempty"`
}

// HandleCommandResponse 处理命令响应
// Topic: edgex/responses/# (通配符)
// Payload:
//
//	{
//	  "header": {"message_type": "command_response", "request_id": "<id>", ...},
//	  "body": {
//	    "request_id": "<id>",
//	    "status": "success|error",
//	    "error": "<error message>",
//	    "node_id": "<node_id>",
//	    "device_id": "<device_id>",
//	    "point_id": "<point_id>"
//	  }
//	}
func (h *ControlHandler) HandleCommandResponse(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Header struct {
			MessageType string `json:"message_type"`
			RequestID   string `json:"request_id"`
		} `json:"header"`
		Body commandResponseBody `json:"body"`
	}

	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		h.logger.Error("ControlHandler.HandleCommandResponse: unmarshal failed",
			zap.Error(err), zap.String("topic", msg.Topic()))
		return
	}

	// request_id 优先从 body 取，兜底从 header 取
	body := envelope.Body
	if body.RequestID == "" {
		body.RequestID = envelope.Header.RequestID
	}
	if body.RequestID == "" {
		h.logger.Warn("ControlHandler.HandleCommandResponse: missing request_id",
			zap.String("topic", msg.Topic()))
		return
	}
	if body.Status == "" {
		body.Status = "success"
	}

	h.logger.Info("Command response received",
		zap.String("request_id", body.RequestID),
		zap.String("status", body.Status),
		zap.String("error", body.Error))

	// 回填 ControlService pending channel，触发 WaitResponse 返回
	if h.controlSvc != nil {
		h.controlSvc.HandleResponse(body.RequestID, body.Status, body.Error)
	}

	// WebSocket 广播命令响应事件到前端
	if h.hub != nil {
		h.hub.BroadcastType(ws.EventCommandResp, map[string]interface{}{
			"request_id": body.RequestID,
			"status":     body.Status,
			"error":      body.Error,
			"node_id":    body.NodeID,
			"device_id":  body.DeviceID,
			"point_id":   body.PointID,
		})
	}
}

// HandleAlertMessage 处理告警消息（可选，统一在 control handler 或单独 alert handler 中）
// 此方法留空，告警由 messaging.Manager 直接处理
