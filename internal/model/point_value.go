package model

// PointValue 点位实时值（扩展 EdgeXPointInfo）
type PointValue struct {
	NodeID       string      `json:"node_id"`
	DeviceID     string      `json:"device_id"`
	PointID      string      `json:"point_id"`
	CurrentValue interface{} `json:"current_value"`
	Quality      string      `json:"quality"` // good, bad, uncertain
	LastUpdated  int64       `json:"last_updated"`
}

// DeviceSnapshot 设备物模型快照（全量）
type DeviceSnapshot struct {
	NodeID    string                 `json:"node_id"`
	DeviceID  string                 `json:"device_id"`
	Points    map[string]interface{} `json:"points"` // pointId -> value
	Quality   string                 `json:"quality"`
	Timestamp int64                  `json:"timestamp"`
}
