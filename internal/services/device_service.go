package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.etcd.io/bbolt"

	"github.com/anviod/edgeOS/internal/model"
)

// NodeBrief 节点摘要信息（供 BuildSpatialTree 使用，避免直接依赖 RegistryService）
// NodeBrief: node summary for BuildSpatialTree (avoids direct RegistryService dependency)
type NodeBrief struct {
	NodeID   string `json:"node_id"`
	NodeName string `json:"node_name"`
	Status   string `json:"status"`
}

// NodeDevice 跨节点设备条目（含所属节点 ID）
// NodeDevice: cross-node device entry (includes owning node ID)
type NodeDevice struct {
	NodeID string
	Device *model.EdgeCoreDeviceInfo
}

const bucketDevices = "edgeCore_devices"

// DeviceService 设备管理服务
type DeviceService struct {
	db *bbolt.DB
}

// NewDeviceService 创建设备服务
func NewDeviceService(db *bbolt.DB) *DeviceService {
	return &DeviceService{db: db}
}

// deviceKey 构造设备存储key
func deviceKey(nodeID, deviceID string) string {
	return fmt.Sprintf("%s:%s", nodeID, deviceID)
}

// UpsertDevice 幂等更新设备
func (s *DeviceService) UpsertDevice(nodeID string, device *model.EdgeCoreDeviceInfo) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketDevices))
		if err != nil {
			return err
		}
		device.LastSync = time.Now().Unix()
		data, err := json.Marshal(device)
		if err != nil {
			return err
		}
		return b.Put([]byte(deviceKey(nodeID, device.DeviceID)), data)
	})
}

// GetDevice 获取设备
func (s *DeviceService) GetDevice(nodeID, deviceID string) (*model.EdgeCoreDeviceInfo, error) {
	var device model.EdgeCoreDeviceInfo
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketDevices))
		if b == nil {
			return fmt.Errorf("device not found: %s/%s", nodeID, deviceID)
		}
		v := b.Get([]byte(deviceKey(nodeID, deviceID)))
		if v == nil {
			return fmt.Errorf("device not found: %s/%s", nodeID, deviceID)
		}
		return json.Unmarshal(v, &device)
	})
	return &device, err
}

// ListDevices 列出节点下所有设备
func (s *DeviceService) ListDevices(nodeID string) ([]*model.EdgeCoreDeviceInfo, error) {
	prefix := nodeID + ":"
	var devices []*model.EdgeCoreDeviceInfo
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketDevices))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.Seek([]byte(prefix)); k != nil && len(k) > len(prefix) && string(k[:len(prefix)]) == prefix; k, v = c.Next() {
			var device model.EdgeCoreDeviceInfo
			if err := json.Unmarshal(v, &device); err != nil {
				continue
			}
			devices = append(devices, &device)
		}
		return nil
	})
	if devices == nil {
		devices = []*model.EdgeCoreDeviceInfo{}
	}
	return devices, err
}

// DeleteDevice 删除设备
func (s *DeviceService) DeleteDevice(nodeID, deviceID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketDevices))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(deviceKey(nodeID, deviceID)))
	})
}

// CountDevices 统计设备数量
func (s *DeviceService) CountDevices() int {
	count := 0
	s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketDevices))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			count++
			return nil
		})
	})
	return count
}

// ReconcileDevices 按节点对账全量设备上报：upsert 上报列表，并删除该节点下未再上报的设备。
// edgeCore `edgeCore/devices/report` 为全量快照；若只 upsert 不剪枝，历史已删除设备会残留，导致
// EdgeOS 设备数与 edgeCore 实际设备数不一致。
func (s *DeviceService) ReconcileDevices(nodeID string, reported []model.EdgeCoreDeviceInfo) (upserted, removed int, err error) {
	if nodeID == "" {
		return 0, 0, fmt.Errorf("nodeID is required")
	}

	keep := make(map[string]struct{}, len(reported))
	for i := range reported {
		dev := reported[i]
		if dev.DeviceID == "" {
			continue
		}
		keep[dev.DeviceID] = struct{}{}
		d := dev
		if err := s.UpsertDevice(nodeID, &d); err != nil {
			return upserted, removed, err
		}
		upserted++
	}

	existing, err := s.ListDevices(nodeID)
	if err != nil {
		return upserted, removed, err
	}
	for _, old := range existing {
		if old == nil || old.DeviceID == "" {
			continue
		}
		if _, ok := keep[old.DeviceID]; ok {
			continue
		}
		if err := s.DeleteDevice(nodeID, old.DeviceID); err != nil {
			return upserted, removed, err
		}
		removed++
	}
	return upserted, removed, nil
}

// UpdateDeviceStatus 更新设备状态
func (s *DeviceService) UpdateDeviceStatus(nodeID, deviceID, status string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketDevices))
		if b == nil {
			return fmt.Errorf("device bucket not found")
		}
		key := []byte(deviceKey(nodeID, deviceID))
		v := b.Get(key)
		if v == nil {
			return fmt.Errorf("device not found: %s/%s", nodeID, deviceID)
		}
		var device model.EdgeCoreDeviceInfo
		if err := json.Unmarshal(v, &device); err != nil {
			return err
		}
		device.OperatingState = status
		device.LastSync = time.Now().Unix()
		data, err := json.Marshal(&device)
		if err != nil {
			return err
		}
		return b.Put(key, data)
	})
}

// ==================== 空间属性查询与空间树 | Spatial queries & tree ====================

// ListAllDevices 列出所有节点的所有设备（跨节点扫描）
// ListAllDevices: list all devices across all nodes (cross-node scan)
func (s *DeviceService) ListAllDevices() ([]NodeDevice, error) {
	var result []NodeDevice
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketDevices))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			key := string(k)
			parts := strings.SplitN(key, ":", 2)
			if len(parts) != 2 {
				return nil
			}
			var device model.EdgeCoreDeviceInfo
			if err := json.Unmarshal(v, &device); err != nil {
				return nil // 跳过损坏记录 | skip corrupted records
			}
			result = append(result, NodeDevice{
				NodeID: parts[0],
				Device: &device,
			})
			return nil
		})
	})
	if result == nil {
		result = []NodeDevice{}
	}
	return result, err
}

// ListByStationCode 按局站编码检索设备（跨所有节点）
// ListByStationCode: query devices by station code (across all nodes)
func (s *DeviceService) ListByStationCode(stationCode string) ([]NodeDevice, error) {
	all, err := s.ListAllDevices()
	if err != nil {
		return nil, err
	}
	var result []NodeDevice
	for _, nd := range all {
		if nd.Device != nil && nd.Device.StationCode == stationCode {
			result = append(result, nd)
		}
	}
	if result == nil {
		result = []NodeDevice{}
	}
	return result, nil
}

// ListByRoomCode 按机房编码检索设备（跨所有节点）
// ListByRoomCode: query devices by room code (across all nodes)
func (s *DeviceService) ListByRoomCode(roomCode string) ([]NodeDevice, error) {
	all, err := s.ListAllDevices()
	if err != nil {
		return nil, err
	}
	var result []NodeDevice
	for _, nd := range all {
		if nd.Device != nil && nd.Device.RoomCode == roomCode {
			result = append(result, nd)
		}
	}
	if result == nil {
		result = []NodeDevice{}
	}
	return result, nil
}

// BuildSpatialTree 构建空间结构树（节点根）：Node → Station → Building → Room → Device
// nodes 参数提供节点摘要信息（来自 RegistryService），设备数据从 BoltDB 读取。
// 未配置空间属性的设备归入「未分配」分组。
// | Build spatial tree (node-rooted): Node → Station → Building → Room → Device.
// | Devices without spatial attributes are grouped under "未分配" (unassigned).
func (s *DeviceService) BuildSpatialTree(nodes []NodeBrief) ([]model.SpatialNode, error) {
	allDevices, err := s.ListAllDevices()
	if err != nil {
		return nil, err
	}

	// 按节点 ID 分组设备 | group devices by node ID
	devicesByNode := make(map[string][]*model.EdgeCoreDeviceInfo)
	for _, nd := range allDevices {
		if nd.Device == nil {
			continue
		}
		devicesByNode[nd.NodeID] = append(devicesByNode[nd.NodeID], nd.Device)
	}

	tree := make([]model.SpatialNode, 0, len(nodes))
	for _, n := range nodes {
		devices := devicesByNode[n.NodeID]
		stations := buildSpatialStations(devices)
		tree = append(tree, model.SpatialNode{
			NodeID:      n.NodeID,
			NodeName:    n.NodeName,
			Status:      n.Status,
			Stations:    stations,
			DeviceCount: len(devices),
		})
	}
	return tree, nil
}

// BuildSpatialTreeByStation 构建空间结构树（局站根）：Station → Building → Room → Device
// 跨节点汇聚所有设备，按局站分组。每个设备的 NodeID 字段标记其归属的 EdgeCore 节点。
// | Build spatial tree (station-rooted): Station → Building → Room → Device.
// | Aggregates devices across all nodes, grouped by station. Each device's NodeID marks the owning node.
func (s *DeviceService) BuildSpatialTreeByStation() ([]model.SpatialStationRoot, error) {
	allDevices, err := s.ListAllDevices()
	if err != nil {
		return nil, err
	}

	// 按局站编码分组设备（跨节点）| group devices by station code (cross-node)
	type stationGroup struct {
		name    string
		code    string
		devices []*model.EdgeCoreDeviceInfo
	}
	stMap := make(map[string]*stationGroup)
	var assignedOrder []string
	hasUnassigned := false
	var unassignedDevs []*model.EdgeCoreDeviceInfo

	for _, nd := range allDevices {
		if nd.Device == nil {
			continue
		}
		// 填充 NodeID 供前端显示设备归属节点 | populate NodeID for frontend display
		nd.Device.NodeID = nd.NodeID

		if nd.Device.StationCode == "" {
			unassignedDevs = append(unassignedDevs, nd.Device)
			hasUnassigned = true
			continue
		}
		g, ok := stMap[nd.Device.StationCode]
		if !ok {
			g = &stationGroup{name: nd.Device.StationName, code: nd.Device.StationCode}
			stMap[nd.Device.StationCode] = g
			assignedOrder = append(assignedOrder, nd.Device.StationCode)
		}
		if nd.Device.StationName != "" && g.name == "" {
			g.name = nd.Device.StationName
		}
		g.devices = append(g.devices, nd.Device)
	}

	tree := make([]model.SpatialStationRoot, 0, len(assignedOrder)+1)
	for _, code := range assignedOrder {
		g := stMap[code]
		tree = append(tree, model.SpatialStationRoot{
			StationName:  g.name,
			StationCode:  g.code,
			Buildings:    buildSpatialBuildings(g.devices),
			DeviceCount:  len(g.devices),
		})
	}
	if hasUnassigned {
		tree = append(tree, model.SpatialStationRoot{
			StationName: "未分配",
			StationCode: "",
			Buildings:   buildSpatialBuildings(unassignedDevs),
			DeviceCount: len(unassignedDevs),
		})
	}
	return tree, nil
}

// buildSpatialStations 将设备列表按 局站 → 机楼 → 机房 → 设备 分组
// | Group device list into station → building → room → device hierarchy.
func buildSpatialStations(devices []*model.EdgeCoreDeviceInfo) []model.SpatialStation {
	type stationGroup struct {
		name    string
		code    string
		devices []*model.EdgeCoreDeviceInfo
	}

	stMap := make(map[string]*stationGroup)
	var assignedOrder []string // 有 station_code 的顺序
	hasUnassigned := false
	var unassignedDevs []*model.EdgeCoreDeviceInfo

	for _, dev := range devices {
		if dev.StationCode == "" {
			unassignedDevs = append(unassignedDevs, dev)
			hasUnassigned = true
			continue
		}
		g, ok := stMap[dev.StationCode]
		if !ok {
			g = &stationGroup{name: dev.StationName, code: dev.StationCode}
			stMap[dev.StationCode] = g
			assignedOrder = append(assignedOrder, dev.StationCode)
		}
		if dev.StationName != "" && g.name == "" {
			g.name = dev.StationName
		}
		g.devices = append(g.devices, dev)
	}

	stations := make([]model.SpatialStation, 0, len(assignedOrder)+1)
	for _, code := range assignedOrder {
		g := stMap[code]
		stations = append(stations, model.SpatialStation{
			StationName:  g.name,
			StationCode:  g.code,
			Buildings:    buildSpatialBuildings(g.devices),
			DeviceCount:  len(g.devices),
		})
	}
	if hasUnassigned {
		stations = append(stations, model.SpatialStation{
			StationName: "未分配",
			StationCode: "",
			Buildings:   buildSpatialBuildings(unassignedDevs),
			DeviceCount: len(unassignedDevs),
		})
	}
	return stations
}

// buildSpatialBuildings 将设备列表按机楼 → 机房分组
// | Group device list by building → room.
func buildSpatialBuildings(devices []*model.EdgeCoreDeviceInfo) []model.SpatialBuilding {
	type buildingGroup struct {
		name    string
		code    string
		devices []*model.EdgeCoreDeviceInfo
	}

	bdMap := make(map[string]*buildingGroup)
	var assignedOrder []string
	hasUnassigned := false
	var unassignedDevs []*model.EdgeCoreDeviceInfo

	for _, dev := range devices {
		if dev.BuildingCode == "" {
			unassignedDevs = append(unassignedDevs, dev)
			hasUnassigned = true
			continue
		}
		g, ok := bdMap[dev.BuildingCode]
		if !ok {
			g = &buildingGroup{name: dev.BuildingName, code: dev.BuildingCode}
			bdMap[dev.BuildingCode] = g
			assignedOrder = append(assignedOrder, dev.BuildingCode)
		}
		if dev.BuildingName != "" && g.name == "" {
			g.name = dev.BuildingName
		}
		g.devices = append(g.devices, dev)
	}

	buildings := make([]model.SpatialBuilding, 0, len(assignedOrder)+1)
	for _, code := range assignedOrder {
		g := bdMap[code]
		buildings = append(buildings, model.SpatialBuilding{
			BuildingName: g.name,
			BuildingCode: g.code,
			Rooms:        buildSpatialRooms(g.devices),
			DeviceCount:  len(g.devices),
		})
	}
	if hasUnassigned {
		buildings = append(buildings, model.SpatialBuilding{
			BuildingName: "未分配",
			BuildingCode: "",
			Rooms:        buildSpatialRooms(unassignedDevs),
			DeviceCount:  len(unassignedDevs),
		})
	}
	return buildings
}

// buildSpatialRooms 将设备列表按机房分组
// | Group device list by room.
func buildSpatialRooms(devices []*model.EdgeCoreDeviceInfo) []model.SpatialRoom {
	type roomGroup struct {
		name    string
		code    string
		devices []*model.EdgeCoreDeviceInfo
	}

	rmMap := make(map[string]*roomGroup)
	var assignedOrder []string
	hasUnassigned := false
	var unassignedDevs []*model.EdgeCoreDeviceInfo

	for _, dev := range devices {
		if dev.RoomCode == "" {
			unassignedDevs = append(unassignedDevs, dev)
			hasUnassigned = true
			continue
		}
		g, ok := rmMap[dev.RoomCode]
		if !ok {
			g = &roomGroup{name: dev.RoomName, code: dev.RoomCode}
			rmMap[dev.RoomCode] = g
			assignedOrder = append(assignedOrder, dev.RoomCode)
		}
		if dev.RoomName != "" && g.name == "" {
			g.name = dev.RoomName
		}
		g.devices = append(g.devices, dev)
	}

	rooms := make([]model.SpatialRoom, 0, len(assignedOrder)+1)
	for _, code := range assignedOrder {
		g := rmMap[code]
		rooms = append(rooms, model.SpatialRoom{
			RoomName:    g.name,
			RoomCode:    g.code,
			Devices:     toDeviceInfoList(g.devices),
			DeviceCount: len(g.devices),
		})
	}
	if hasUnassigned {
		rooms = append(rooms, model.SpatialRoom{
			RoomName:    "未分配",
			RoomCode:    "",
			Devices:     toDeviceInfoList(unassignedDevs),
			DeviceCount: len(unassignedDevs),
		})
	}
	return rooms
}

// toDeviceInfoList 将指针切片转为值切片（用于 JSON 序列化）
// | Convert pointer slice to value slice (for JSON serialization).
func toDeviceInfoList(ptrs []*model.EdgeCoreDeviceInfo) []model.EdgeCoreDeviceInfo {
	result := make([]model.EdgeCoreDeviceInfo, 0, len(ptrs))
	for _, p := range ptrs {
		if p != nil {
			result = append(result, *p)
		}
	}
	return result
}
