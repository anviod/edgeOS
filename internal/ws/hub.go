package ws

import (
	"encoding/json"
	"sync"
	"time"

	fiberws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// EventType WebSocket 事件类型
type EventType string

const (
	EventDataUpdate       EventType = "data_update"
	EventNodeStatus       EventType = "node_status"
	EventDeviceSynced     EventType = "device_synced"
	EventDeviceOnline     EventType = "device_online"
	EventDeviceOffline    EventType = "device_offline"
	EventCommandResp      EventType = "command_response"
	EventAlert            EventType = "alert"
	EventMiddlewareStatus EventType = "middleware_status"
	EventPointReport      EventType = "point_report"
	EventPointSynced      EventType = "point_synced"
)

// RealtimeEvent WebSocket 实时事件
type RealtimeEvent struct {
	Type      EventType   `json:"type"`
	Timestamp int64       `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// DataUpdatePayload 数据更新事件载荷
type DataUpdatePayload struct {
	NodeID         string                 `json:"node_id"`
	DeviceID       string                 `json:"device_id"`
	Points         map[string]interface{} `json:"points"`
	Timestamp      int64                  `json:"timestamp"`
	Quality        string                 `json:"quality"`
	IsFullSnapshot bool                   `json:"is_full_snapshot"`
}

// NodeStatusPayload 节点状态事件载荷
type NodeStatusPayload struct {
	NodeID   string `json:"node_id"`
	NodeName string `json:"node_name"`
	Status   string `json:"status"`
	LastSeen int64  `json:"last_seen"`
}

// client WebSocket 客户端
type client struct {
	conn *fiberws.Conn
	send chan []byte
	done chan struct{}
}

// Hub WebSocket Hub
type Hub struct {
	mu      sync.RWMutex
	clients map[*client]bool
	logger  *zap.Logger
}

// NewHub 创建 Hub
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients: make(map[*client]bool),
		logger:  logger,
	}
}

// Broadcast 广播事件到所有客户端
func (h *Hub) Broadcast(event RealtimeEvent) {
	event.Timestamp = time.Now().UnixMilli()
	data, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("Failed to marshal event", zap.Error(err))
		return
	}

	h.mu.RLock()
	clients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		select {
		case c.send <- data:
		default:
			h.removeClient(c)
		}
	}
}

// BroadcastType 广播指定类型事件
func (h *Hub) BroadcastType(eventType EventType, payload interface{}) {
	h.Broadcast(RealtimeEvent{
		Type:    eventType,
		Payload: payload,
	})
}

// IsWebSocketUpgrade 检查是否需要升级 WebSocket
func IsWebSocketUpgrade(c *fiber.Ctx) bool {
	return fiberws.IsWebSocketUpgrade(c)
}

// NewHandler 返回 Fiber WebSocket 中间件和处理器
func (h *Hub) NewHandler() fiber.Handler {
	return fiberws.New(func(conn *fiberws.Conn) {
		cli := &client{
			conn: conn,
			send: make(chan []byte, 256),
			done: make(chan struct{}),
		}

		h.mu.Lock()
		h.clients[cli] = true
		h.mu.Unlock()

		h.logger.Info("WebSocket client connected",
			zap.String("remote", conn.RemoteAddr().String()))

		go h.writePump(cli)
		h.readPump(cli)
	})
}

func (h *Hub) writePump(c *client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.conn.WriteMessage(1, msg); err != nil { // 1 = TextMessage
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteMessage(9, nil); err != nil { // 9 = PingMessage
				return
			}
		case <-c.done:
			return
		}
	}
}

func (h *Hub) readPump(c *client) {
	defer func() {
		close(c.done)
		h.removeClient(c)
	}()
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (h *Hub) removeClient(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		select {
		case <-c.send:
		default:
			close(c.send)
		}
	}
}

// ClientCount 返回当前连接客户端数量
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
