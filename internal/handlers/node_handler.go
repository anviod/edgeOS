package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/model"
	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/ws"
)

// NodeHandler 处理节点注册、心跳、注销消息
type NodeHandler struct {
	registrySvc *services.RegistryService
	hub         *ws.Hub
	logger      *zap.Logger
	publishFn   func(topic string, payload []byte) error
}

// NewNodeHandler 创建节点消息处理器
func NewNodeHandler(
	registrySvc *services.RegistryService,
	hub *ws.Hub,
	logger *zap.Logger,
	publishFn func(topic string, payload []byte) error,
) *NodeHandler {
	return &NodeHandler{
		registrySvc: registrySvc,
		hub:         hub,
		logger:      logger,
		publishFn:   publishFn,
	}
}

// HandleRegister 处理节点注册消息
// Topic: edgex/nodes/register
func (h *NodeHandler) HandleRegister(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Header map[string]interface{} `json:"header"`
		Body   model.EdgeXNodeInfo    `json:"body"`
	}
	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		h.logger.Error("NodeHandler.HandleRegister: unmarshal failed",
			zap.Error(err), zap.String("topic", msg.Topic()))
		return
	}

	node := &envelope.Body
	node.Status = "online"

	// 生成或保留 access_token
	if node.AccessToken == "" {
		node.AccessToken = uuid.New().String()
		node.ExpiresAt = time.Now().Add(24 * time.Hour).Unix()
	}

	if err := h.registrySvc.UpsertNode(node); err != nil {
		h.logger.Error("NodeHandler.HandleRegister: upsert failed",
			zap.Error(err), zap.String("node_id", node.NodeID))
		return
	}

	h.logger.Info("Node registered",
		zap.String("node_id", node.NodeID),
		zap.String("node_name", node.NodeName),
		zap.String("protocol", node.Protocol))

	// 回复注册响应
	h.publishRegisterResponse(node)

	// WebSocket 广播
	if h.hub != nil {
		h.hub.BroadcastType(ws.EventNodeStatus, ws.NodeStatusPayload{
			NodeID:   node.NodeID,
			NodeName: node.NodeName,
			Status:   "online",
			LastSeen: node.LastSeen,
		})
	}
}

// publishRegisterResponse 发布注册响应
func (h *NodeHandler) publishRegisterResponse(node *model.EdgeXNodeInfo) {
	if h.publishFn == nil {
		return
	}
	resp, err := json.Marshal(map[string]interface{}{
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
	if err != nil {
		h.logger.Error("NodeHandler: marshal register response failed", zap.Error(err))
		return
	}
	topic := fmt.Sprintf("edgex/nodes/%s/response", node.NodeID)
	if err := h.publishFn(topic, resp); err != nil {
		h.logger.Error("NodeHandler: publish register response failed",
			zap.Error(err), zap.String("topic", topic))
	}
}

// HandleHeartbeat 处理节点心跳消息
// Topic: edgex/nodes/heartbeat
func (h *NodeHandler) HandleHeartbeat(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Header map[string]interface{} `json:"header"`
		Body   struct {
			NodeID string `json:"node_id"`
		} `json:"body"`
	}
	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		h.logger.Warn("NodeHandler.HandleHeartbeat: unmarshal failed", zap.Error(err))
		return
	}

	nodeID := envelope.Body.NodeID
	if nodeID == "" {
		// 尝试从 header 中读取
		if src, ok := envelope.Header["source"].(string); ok {
			nodeID = src
		}
	}
	if nodeID == "" {
		return
	}

	if err := h.registrySvc.UpdateNodeStatus(nodeID, "online"); err != nil {
		h.logger.Debug("NodeHandler.HandleHeartbeat: update status failed",
			zap.Error(err), zap.String("node_id", nodeID))
	}

	if h.hub != nil {
		h.hub.BroadcastType(ws.EventNodeStatus, ws.NodeStatusPayload{
			NodeID:   nodeID,
			Status:   "online",
			LastSeen: time.Now().Unix(),
		})
	}
}

// HandleUnregister 处理节点注销消息
// Topic: edgex/nodes/unregister
func (h *NodeHandler) HandleUnregister(_ pahomqtt.Client, msg pahomqtt.Message) {
	var envelope struct {
		Header map[string]interface{} `json:"header"`
		Body   struct {
			NodeID string `json:"node_id"`
		} `json:"body"`
	}
	if err := json.Unmarshal(msg.Payload(), &envelope); err != nil {
		h.logger.Warn("NodeHandler.HandleUnregister: unmarshal failed", zap.Error(err))
		return
	}

	nodeID := envelope.Body.NodeID
	if nodeID == "" {
		return
	}

	if err := h.registrySvc.UpdateNodeStatus(nodeID, "offline"); err != nil {
		h.logger.Debug("NodeHandler.HandleUnregister: update status failed",
			zap.Error(err), zap.String("node_id", nodeID))
	}

	h.logger.Info("Node unregistered", zap.String("node_id", nodeID))

	if h.hub != nil {
		h.hub.BroadcastType(ws.EventNodeStatus, ws.NodeStatusPayload{
			NodeID:   nodeID,
			Status:   "offline",
			LastSeen: time.Now().Unix(),
		})
	}
}
