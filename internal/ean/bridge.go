package ean

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/model"
	"github.com/anviod/edgeOS/internal/services"
)

// V1ToEANBridge V1 数据同步桥接器
// 通过轮询 EdgeOS 内部服务将 V1 节点的 Agent 状态、心跳和点位数据同步到 EAN 子系统。
// 注意（OS-P3-01）：不再合成 V1 Capability（{node}/{device}/read-write），
// 写操作统一通过 EAN Invoke 下发到 EdgeX 北向原生 Capability。
type V1ToEANBridge struct {
	bus         *Bus
	registrySvc *services.RegistryService
	dataSvc     *services.DataService
	logger      *zap.Logger

	mu         sync.Mutex
	running    bool
	stopCh     chan struct{}
	interval   time.Duration

	// 缓存上一次的点位值，用于计算 previous_value
	pointCache map[string]interface{} // key: nodeID/deviceID/pointID
}

// NewV1ToEANBridge 创建 V1→EAN 桥接器
func NewV1ToEANBridge(
	bus *Bus,
	registrySvc *services.RegistryService,
	dataSvc *services.DataService,
	logger *zap.Logger,
) *V1ToEANBridge {
	return &V1ToEANBridge{
		bus:         bus,
		registrySvc: registrySvc,
		dataSvc:     dataSvc,
		logger:      logger.Named("ean-bridge"),
		stopCh:      make(chan struct{}),
		interval:    5 * time.Second,
		pointCache:  make(map[string]interface{}),
	}
}

// Start 启动桥接器，立即执行一次同步，然后定时轮询
func (b *V1ToEANBridge) Start() {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return
	}
	b.running = true
	b.mu.Unlock()

	b.logger.Info("V1→EAN bridge started",
		zap.Duration("sync_interval", b.interval))

	// 立即执行一次同步（让 EAN 立即可见现有节点）
	b.sync()

	// 启动后台轮询 goroutine
	go b.loop()
}

// Stop 停止桥接器
func (b *V1ToEANBridge) Stop() {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return
	}
	b.running = false
	b.mu.Unlock()

	close(b.stopCh)
	b.logger.Info("V1→EAN bridge stopped")
}

// loop 定时轮询循环
func (b *V1ToEANBridge) loop() {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for {
		select {
		case <-b.stopCh:
			return
		case <-ticker.C:
			b.sync()
		}
	}
}

// sync 执行一次完整的 V1→EAN 数据同步
func (b *V1ToEANBridge) sync() {
	nodes, err := b.registrySvc.ListNodes()
	if err != nil {
		b.logger.Warn("bridge sync: failed to list nodes", zap.Error(err))
		return
	}

	if len(nodes) == 0 {
		return // 无节点，无需同步
	}

	b.logger.Debug("bridge sync: syncing nodes",
		zap.Int("node_count", len(nodes)))

	for _, node := range nodes {
		if node == nil {
			continue
		}
		b.syncNode(node)
	}
}

// syncNode 同步单个节点到 EAN
// V1 Bridge 隔离规则（对接 EdgeX 北向 EAN Runtime，非 MCP Runtime）：
//   - 若 Agent 已有原生 EAN Capability / Agent 描述符 → 跳过 Agent 覆盖与设备级 Cap 合成
//   - 心跳与点位 Event 仍可同步（兼容北向心跳短暂缺失）
func (b *V1ToEANBridge) syncNode(node *model.EdgeXNodeInfo) {
	hasNative := b.bus.Discovery.HasNativeEANCaps(node.NodeID) ||
		b.bus.Discovery.HasNativeEANAgent(node.NodeID)

	// 1. 同步 Agent（已有北向原生 Agent 时由 DiscoveryCenter 拒绝覆盖）
	if !hasNative {
		agent := b.nodeToAgent(node)
		agentPayload, _ := json.Marshal(agent)
		b.bus.Discovery.HandleAgentOnline(TopicDiscoveryAgent, agentPayload, "v1-bridge")
	}

	// 2. 模拟心跳（维持 Agent 在线状态；原生心跳到达后仍可作兜底）
	hb := HeartbeatPayload{
		AgentID:   node.NodeID,
		Status:    node.Status,
		Timestamp: time.Now().UnixMilli(),
		Sequence:  0,
	}
	hbPayload, _ := json.Marshal(hb)
	b.bus.Heartbeat.HandleHeartbeat(HeartbeatTopic(node.NodeID), hbPayload, "v1-bridge")

	// 3. 同步设备和 Capability（V1 Bridge 隔离检查）
	devices, err := b.dataSvc.DeviceSvc.ListDevices(node.NodeID)
	if err != nil {
		b.logger.Warn("bridge sync: failed to list devices",
			zap.String("node_id", node.NodeID), zap.Error(err))
		return
	}

	if hasNative {
		b.logger.Debug("bridge sync: agent has native EAN, skipping device synthesis",
			zap.String("node_id", node.NodeID))
	} else {
		for _, dev := range devices {
			if dev == nil {
				continue
			}
			b.syncDevice(node.NodeID, dev)
		}
	}

	// 4. 同步点位数据（生成 EAN Event，含 previous_value）
	for _, dev := range devices {
		if dev == nil {
			continue
		}
		b.syncPointData(node.NodeID, dev)
	}
}

// nodeToAgent 将 EdgeX 节点转换为 EAN Agent 描述符
func (b *V1ToEANBridge) nodeToAgent(node *model.EdgeXNodeInfo) AgentDescriptor {
	status := AgentOffline
	if node.Status == "online" {
		status = AgentOnline
	}

	metadata := FlexibleStringMap{
		"protocol":  node.Protocol,
		"model":     node.Model,
		"node_name": node.NodeName,
	}
	if node.Metadata != nil {
		metadata["hostname"] = node.Metadata.Hostname
		metadata["os"] = node.Metadata.OS
		metadata["arch"] = node.Metadata.Arch
	}
	if node.Endpoint != nil {
		metadata["endpoint_host"] = node.Endpoint.Host
		metadata["endpoint_port"] = node.Endpoint.Port
	}

	return AgentDescriptor{
		ID:                   node.NodeID,
		Kind:                 "edgex-gateway",
		Version:              node.Version,
		Status:               status,
		Transport:            TransportList{"mqtt"},
		HeartbeatIntervalSec: 30,
		Metadata:             metadata,
	}
}

// syncDevice 同步设备上下线状态到 EAN Event（OS-P3-01: 不再合成 V1 Capability）
func (b *V1ToEANBridge) syncDevice(nodeID string, device *model.EdgeXDeviceInfo) {
	// 仅同步设备上下线事件，不再合成 {node}/{device}/read-write Capability
	// 写操作统一通过 EAN Invoke 调用 EdgeX 北向原生 Capability（如 modbus_tcp.write_register）
	if device.OperatingState == "UP" || device.OperatingState == "online" {
		event := DeviceStatusEvent{
			EventType:  "device.online",
			AgentID:    nodeID,
			DeviceID:   device.DeviceID,
			DeviceName: device.DeviceName,
			Timestamp:  time.Now().UnixMilli(),
		}
		evtPayload, _ := json.Marshal(event)
		b.bus.Event.HandleEvent(EventTopic(nodeID), evtPayload, "v1-bridge")
	}
}

// syncPointData 同步点位数据变化到 EAN Event（含 previous_value）
func (b *V1ToEANBridge) syncPointData(nodeID string, device *model.EdgeXDeviceInfo) {
	snapshot, err := b.dataSvc.PointService.GetSnapshot(nodeID, device.DeviceID)
	if err != nil {
		// 无快照数据，正常情况
		return
	}

	for pointID, value := range snapshot.Points {
		cacheKey := nodeID + "/" + device.DeviceID + "/" + pointID

		b.mu.Lock()
		prevValue, hasPrev := b.pointCache[cacheKey]
		b.mu.Unlock()

		// 首次或值变化时才生成事件
		if hasPrev && fmt.Sprintf("%v", prevValue) == fmt.Sprintf("%v", value) {
			continue
		}

		event := PointChangeEvent{
			EventType:     "point.change",
			AgentID:       nodeID,
			DeviceID:      device.DeviceID,
			PointID:       pointID,
			Value:         value,
			PreviousValue: prevValue,
			Timestamp:     snapshot.Timestamp,
			Metadata: &PointEventMetadata{
				Quality: snapshot.Quality,
			},
		}

		evtPayload, _ := json.Marshal(event)
		b.bus.Event.HandleEvent(EventTopic(nodeID), evtPayload, "v1-bridge")

		b.mu.Lock()
		b.pointCache[cacheKey] = value
		b.mu.Unlock()
	}
}

// SetSyncInterval 动态调整同步间隔（调试用）
func (b *V1ToEANBridge) SetSyncInterval(d time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if d > 0 {
		b.interval = d
		b.logger.Info("bridge sync interval updated", zap.Duration("interval", d))
	}
}
