package model

import (
	"encoding/json"
)

// EndpointInfo represents endpoint information in a node registration message
type EndpointInfo struct {
	Host string `json:"host"`
	Port string `json:"port"` // actual field is string per real message
}

// NodeMetadata represents metadata in a node registration message
type NodeMetadata struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
}

// EdgeCoreNodeInfo represents edgeCore node information
type EdgeCoreNodeInfo struct {
	NodeID       string        `json:"node_id"`
	NodeName     string        `json:"node_name"`
	Model        string        `json:"model"`
	Version      string        `json:"version"`
	APIVersion   string        `json:"api_version"`
	Capabilities []string      `json:"capabilities"`
	Protocol     string        `json:"protocol"`
	Endpoint     *EndpointInfo `json:"endpoint,omitempty"`
	Metadata     *NodeMetadata `json:"metadata,omitempty"`
	AccessToken  string        `json:"access_token"`
	ExpiresAt    int64         `json:"expires_at"`
	LastSeen     int64         `json:"last_seen"`
	Status       string        `json:"status"`
}

// EdgeCoreDeviceInfo represents edgeCore device information
type EdgeCoreDeviceInfo struct {
	DeviceID       string                 `json:"device_id"`
	DeviceName     string                 `json:"device_name"`
	DeviceProfile  string                 `json:"device_profile"`
	ServiceName    string                 `json:"service_name"`
	Labels         []string               `json:"labels"`
	Description    string                 `json:"description"`
	AdminState     string                 `json:"admin_state"`
	OperatingState string                 `json:"operating_state"`
	Properties     map[string]interface{} `json:"properties"`
	LastSync       int64                  `json:"last_sync"`
	// 空间属性（device_report 顶级字段，omitempty 向后兼容）
	// Spatial attributes (top-level fields in device_report, omitempty for backward compat)
	StationName  string `json:"station_name,omitempty"`  // 局站名称 | Station name
	StationCode  string `json:"station_code,omitempty"`  // 局站编码 | Station code
	BuildingName string `json:"building_name,omitempty"` // 机楼名称 | Building name
	BuildingCode string `json:"building_code,omitempty"` // 机楼编码 | Building code
	RoomName     string `json:"room_name,omitempty"`     // 机房名称 | Room name
	RoomCode     string `json:"room_code,omitempty"`     // 机房编码 | Room code
	// NodeID 仅在局站根树视图下发填充（标记设备归属的 EdgeCore 节点），存储时不落库
	// NodeID is populated only in station-rooted tree responses (marks the owning EdgeCore node); not persisted
	NodeID string `json:"node_id,omitempty"`
}

// ==================== 空间结构树类型 | Spatial tree types ====================

// SpatialRoom 机房层级——空间树最底层，包含设备列表
// SpatialRoom: room level (leaf tier of spatial tree), contains device list
type SpatialRoom struct {
	RoomName    string                `json:"room_name"`
	RoomCode    string                `json:"room_code"`
	Devices     []EdgeCoreDeviceInfo  `json:"devices"`
	DeviceCount int                   `json:"device_count"`
}

// SpatialBuilding 机楼层级——空间树中间层，包含机房列表
// SpatialBuilding: building level (middle tier of spatial tree), contains room list
type SpatialBuilding struct {
	BuildingName string         `json:"building_name"`
	BuildingCode string         `json:"building_code"`
	Rooms        []SpatialRoom  `json:"rooms"`
	DeviceCount  int            `json:"device_count"`
}

// SpatialStation 局站层级——空间树第二层，包含机楼列表
// SpatialStation: station level (2nd tier of spatial tree), contains building list
type SpatialStation struct {
	StationName  string             `json:"station_name"`
	StationCode  string             `json:"station_code"`
	Buildings    []SpatialBuilding  `json:"buildings"`
	DeviceCount  int                `json:"device_count"`
}

// SpatialNode 节点层级——空间树第一层（EdgeCore 节点），包含局站列表
// SpatialNode: node level (1st tier, EdgeCore node), contains station list
type SpatialNode struct {
	NodeID      string           `json:"node_id"`
	NodeName    string           `json:"node_name"`
	Status      string           `json:"status"`
	Stations    []SpatialStation `json:"stations"`
	DeviceCount int              `json:"device_count"`
}

// SpatialStationRoot 局站根树——以局站为根的空间树（跨节点汇聚）
// 设备的 NodeID 字段标记其归属的 EdgeCore 节点。
// SpatialStationRoot: station-rooted spatial tree (aggregated across nodes).
// Device.NodeID marks the owning EdgeCore node.
type SpatialStationRoot struct {
	StationName  string             `json:"station_name"`
	StationCode  string             `json:"station_code"`
	Buildings    []SpatialBuilding  `json:"buildings"`
	DeviceCount  int                `json:"device_count"`
}

// EdgeCorePointInfo represents edgeCore point information
type EdgeCorePointInfo struct {
	PointID      string                 `json:"point_id"`
	PointName    string                 `json:"point_name"`
	DeviceID     string                 `json:"device_id"`
	ServiceName  string                 `json:"service_name"`
	ProfileName  string                 `json:"profile_name"`
	PointType    string                 `json:"point_type"`
	DataType     string                 `json:"data_type"`
	ReadWrite    bool                   `json:"read_write"`
	DefaultValue interface{}            `json:"default_value"`
	Units        string                 `json:"units"`
	Description  string                 `json:"description"`
	Properties   map[string]interface{} `json:"properties"`
	LastSync     int64                  `json:"last_sync"`
}

// EncodeNodeInfo encodes EdgeCoreNodeInfo to JSON bytes
func EncodeNodeInfo(node *EdgeCoreNodeInfo) ([]byte, error) {
	return json.Marshal(node)
}

// DecodeNodeInfo decodes JSON bytes to EdgeCoreNodeInfo
func DecodeNodeInfo(data []byte) (*EdgeCoreNodeInfo, error) {
	var node EdgeCoreNodeInfo
	err := json.Unmarshal(data, &node)
	return &node, err
}

// EncodeDeviceInfo encodes EdgeCoreDeviceInfo to JSON bytes
func EncodeDeviceInfo(device *EdgeCoreDeviceInfo) ([]byte, error) {
	return json.Marshal(device)
}

// DecodeDeviceInfo decodes JSON bytes to EdgeCoreDeviceInfo
func DecodeDeviceInfo(data []byte) (*EdgeCoreDeviceInfo, error) {
	var device EdgeCoreDeviceInfo
	err := json.Unmarshal(data, &device)
	return &device, err
}

// EncodePointInfo encodes EdgeCorePointInfo to JSON bytes
func EncodePointInfo(point *EdgeCorePointInfo) ([]byte, error) {
	return json.Marshal(point)
}

// DecodePointInfo decodes JSON bytes to EdgeCorePointInfo
func DecodePointInfo(data []byte) (*EdgeCorePointInfo, error) {
	var point EdgeCorePointInfo
	err := json.Unmarshal(data, &point)
	return &point, err
}
