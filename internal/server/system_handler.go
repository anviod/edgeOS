package server

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/anviod/edgeOS/internal/model"
	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/storage"
	"go.etcd.io/bbolt"
)

// ===========================
// 系统管理（服务重启 / 配置导出）
// System management (service restart / config export)
// ===========================

// RuntimeExport 运行时映射关系导出结构 | Runtime mapping export structure
type RuntimeExport struct {
	Nodes   []*model.EdgeCoreNodeInfo   `json:"nodes"`
	Devices []*model.EdgeCoreDeviceInfo `json:"devices"`
	Points  []*model.EdgeCorePointInfo  `json:"points"`
}

// FullConfigExport 完整配置导出（含节点和映射关系）
// FullConfigExport includes all config + runtime node/device/point mappings
type FullConfigExport struct {
	Config     *storage.ConfigExport `json:"config"`
	Runtime    RuntimeExport         `json:"runtime"`
	ExportedAt string                `json:"exported_at"`
	Version    string                `json:"version"`
}

// handleServiceRestart 重启服务 | Restart service
// POST /api/system/restart
// 通过发送 os.Interrupt 信号触发优雅关闭，systemd (Restart=always) 会自动重启进程
// Sends os.Interrupt to trigger graceful shutdown; systemd restarts automatically
func handleServiceRestart() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 异步触发重启，确保 HTTP 响应先返回客户端
		// Trigger restart asynchronously to ensure HTTP response is sent first
		go func() {
			time.Sleep(500 * time.Millisecond)
			p, err := os.FindProcess(os.Getpid())
			if err == nil {
				_ = p.Signal(os.Interrupt)
			}
		}()
		return apiSuccess(c, fiber.Map{
			"message": "服务正在重启，请稍后刷新页面 | Service is restarting, please refresh shortly",
			"delay":   "5s",
		})
	}
}

// handleExportConfig 导出完整配置（包含所有节点和映射关系）
// GET /api/system/export-config
// 导出 config.db 全部配置 + 运行时节点/设备/点位映射关系
// Exports config.db data + runtime node/device/point mappings
func handleExportConfig(
	configDB *bbolt.DB,
	registrySvc *services.RegistryService,
	dataSvc *services.DataService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if configDB == nil {
			return apiError(c, fiber.StatusInternalServerError,
				"配置数据库未初始化 | Config database not initialized")
		}

		// 1. 导出配置库数据 | Export config DB data
		configStore, err := storage.NewConfigStore(configDB)
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError,
				fmt.Sprintf("创建配置存储失败 | Failed to create config store: %v", err))
		}
		configExport, err := configStore.ExportAllConfig()
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError,
				fmt.Sprintf("导出配置失败 | Failed to export config: %v", err))
		}

		// 2. 导出运行时映射关系 | Export runtime mappings
		runtimeExport := RuntimeExport{
			Nodes:   []*model.EdgeCoreNodeInfo{},
			Devices: []*model.EdgeCoreDeviceInfo{},
			Points:  []*model.EdgeCorePointInfo{},
		}

		// 获取所有节点 | Get all nodes
		nodes, err := registrySvc.ListNodes()
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError,
				fmt.Sprintf("获取节点列表失败 | Failed to list nodes: %v", err))
		}
		runtimeExport.Nodes = nodes

		// 遍历节点获取设备和点位 | Iterate nodes for devices and points
		for _, node := range nodes {
			// 获取节点下所有设备 | Get devices for this node
			devices, err := dataSvc.DeviceSvc.ListDevices(node.NodeID)
			if err != nil {
				continue
			}
			runtimeExport.Devices = append(runtimeExport.Devices, devices...)

			// 获取每台设备的点位 | Get points for each device
			for _, device := range devices {
				points, err := dataSvc.PointService.ListByDevice(node.NodeID, device.DeviceID)
				if err != nil {
					continue
				}
				runtimeExport.Points = append(runtimeExport.Points, points...)
			}
		}

		// 3. 组装完整导出 | Assemble full export
		fullExport := FullConfigExport{
			Config:     configExport,
			Runtime:    runtimeExport,
			ExportedAt: time.Now().Format(time.RFC3339),
			Version:    "1.0",
		}

		// 4. 返回 JSON 文件下载 | Return as downloadable JSON file
		filename := fmt.Sprintf("edgeos-config-%s.json", time.Now().Format("20060102-150405"))
		c.Set("Content-Type", "application/json; charset=utf-8")
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

		jsonData, err := json.MarshalIndent(fullExport, "", "  ")
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError,
				fmt.Sprintf("序列化导出数据失败 | Failed to marshal export: %v", err))
		}
		return c.Send(jsonData)
	}
}
