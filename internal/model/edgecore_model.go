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
