package ean

import (
	"time"

	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/services"
)

// AttachEventBridge 将 EAN EventCenter 的事件回调连接到 V1 设备/点位服务。
// 当 EAN Bus 收到 $edgeos/event/# 消息时，EventCenter 解析事件并调用 V1 服务：
//   - device.online → DeviceService.UpdateDeviceStatus(online) + RegistryService.EnsureNodeOnline
//   - device.offline → DeviceService.UpdateDeviceStatus(offline)
//   - point.change → PointService.SaveSnapshot + DeviceService.UpdateDeviceStatus(online)
//
// 注意：V1 messaging.Manager 也订阅 $edgeos/event/# 并处理相同事件。
// 两条路径均使用幂等操作（Upsert/Update），不会产生数据冲突。
// AttachEventBridge wires EAN EventCenter callbacks to V1 device/point services.
func (b *Bus) AttachEventBridge(
	deviceSvc *services.DeviceService,
	pointSvc *services.PointService,
	registrySvc *services.RegistryService,
) {
	if b == nil || b.Event == nil {
		return
	}

	logger := b.logger.Named("event-bridge")

	// 设备上线 → 更新设备状态 + 确保节点存在
	// Device online → update device status + ensure node exists
	b.Event.SetDeviceOnlineHandler(func(event *DeviceStatusEvent) {
		if event == nil || event.AgentID == "" || event.DeviceID == "" {
			return
		}
		// 确保节点存在 | ensure node exists
		if registrySvc != nil {
			_ = registrySvc.EnsureNodeOnline(event.AgentID, event.AgentID, "ean")
		}
		// 更新设备状态 | update device status
		if deviceSvc != nil {
			if err := deviceSvc.UpdateDeviceStatus(event.AgentID, event.DeviceID, "online"); err != nil {
				logger.Debug("event bridge: update device online failed",
					zap.String("agent_id", event.AgentID),
					zap.String("device_id", event.DeviceID),
					zap.Error(err))
			}
		}
	})

	// 设备下线 → 更新设备状态
	// Device offline → update device status
	b.Event.SetDeviceOfflineHandler(func(event *DeviceStatusEvent) {
		if event == nil || event.AgentID == "" || event.DeviceID == "" {
			return
		}
		if deviceSvc != nil {
			if err := deviceSvc.UpdateDeviceStatus(event.AgentID, event.DeviceID, "offline"); err != nil {
				logger.Debug("event bridge: update device offline failed",
					zap.String("agent_id", event.AgentID),
					zap.String("device_id", event.DeviceID),
					zap.Error(err))
			}
		}
	})

	// 点位变化 → 保存快照 + 标记设备在线
	// Point change → save snapshot + mark device online
	b.Event.SetPointChangeHandler(func(event *PointChangeEvent) {
		if event == nil || event.AgentID == "" || event.DeviceID == "" || event.PointID == "" {
			return
		}
		// 确保节点存在 | ensure node exists
		if registrySvc != nil {
			_ = registrySvc.EnsureNodeOnline(event.AgentID, event.AgentID, "ean")
		}
		// 保存点位快照 | save point snapshot
		if pointSvc != nil {
			points := map[string]interface{}{
				event.PointID: event.Value,
			}
			quality := ""
			if event.Metadata != nil && event.Metadata.Quality != "" {
				quality = event.Metadata.Quality
			}
			ts := event.Timestamp
			if ts == 0 {
				ts = time.Now().UnixMilli()
			}
			pointSvc.SaveSnapshot(event.AgentID, event.DeviceID, points, quality, ts/1000, false)
		}
		// 收到数据表示设备在线 | receiving data means device is online
		if deviceSvc != nil {
			_ = deviceSvc.UpdateDeviceStatus(event.AgentID, event.DeviceID, "online")
		}
	})

	logger.Info("EAN→V1 event bridge attached")
}
