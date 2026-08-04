package server

import (
	fiberws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/config"
	"github.com/anviod/edgeOS/internal/core"
	"github.com/anviod/edgeOS/internal/discovery"
	"github.com/anviod/edgeOS/internal/ean"
	"github.com/anviod/edgeOS/internal/messaging"
	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/storage"
	"github.com/anviod/edgeOS/internal/ws"
)

// RegisterAllRoutes 注册所有路由
// eanBus: EAN Bus 实例，可为 nil（EAN 未启用时）
// configDB: 配置数据库句柄（用于配置导出）
// store: 双库存储管理器（用于数据管理）
func RegisterAllRoutes(
	app *fiber.App,
	node core.Node,
	db *bbolt.DB,
	configDB *bbolt.DB,
	store *storage.Storage,
	hub *ws.Hub,
	registrySvc *services.RegistryService,
	dataSvc *services.DataService,
	alertSvc *services.AlertService,
	middlewareSvc *services.MiddlewareService,
	controlSvc *services.ControlService,
	messagingManager *messaging.Manager,
	discoveryService *discovery.DiscoveryService,
	cfg *config.Config,
	logger *zap.Logger,
	eanBus *ean.Bus,
) {
	api := app.Group("/api")

	// ===========================
	// 公开路由（无需JWT）
	// ===========================

	// 安装引导（无配置数据时必须先完成安装）| Install wizard (required when no config data)
	install := api.Group("/install")
	install.Get("/status", handleInstallStatus(configDB))
	install.Post("/", handleInstall(configDB, triggerServiceRestart))

	// 认证路由
	auth := api.Group("/auth")
	auth.Post("/login", handleLogin(cfg))
	auth.Post("/logout", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"code": "0", "msg": "Logged out"})
	})
	auth.Get("/system-info", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"code": "0",
			"data": fiber.Map{"name": "EdgeOS", "softVer": "v1.0.0"},
		})
	})

	// WebSocket 路由
	app.Use("/ws", func(c *fiber.Ctx) error {
		if fiberws.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	// 应用JWT认证到WebSocket路由
	app.Get("/ws", JWTAuth(), hub.NewHandler())

	// ===========================
	// JWT 保护路由
	// ===========================
	protected := api.Group("")
	protected.Use(JWTAuth())

	// 仪表盘统计
	protected.Get("/dashboard/stats", handleDashboardStats(registrySvc, dataSvc, alertSvc))

	// 中间件管理
	mw := protected.Group("/middlewares")
	mw.Get("/", handleListMiddlewares(middlewareSvc))
	mw.Post("/", handleCreateMiddleware(middlewareSvc))
	mw.Put("/:id", handleUpdateMiddleware(middlewareSvc))
	mw.Delete("/:id", handleDeleteMiddleware(middlewareSvc))
	mw.Post("/:id/connect", handleConnectMiddleware(middlewareSvc, messagingManager))
	mw.Post("/:id/disconnect", handleDisconnectMiddleware(middlewareSvc, messagingManager))
	mw.Get("/:id/status", handleGetMiddlewareStatus(middlewareSvc))

	// 节点管理
	nodes := protected.Group("/nodes")
	nodes.Get("/", handleListNodes(registrySvc))
	nodes.Get("/:nodeId", handleGetNode(registrySvc))
	nodes.Delete("/:nodeId", handleDeleteNode(registrySvc, eanBus))
	nodes.Post("/:nodeId/discover", handleNodeDiscovery(eanBus, messagingManager))

	// 设备管理
	nodes.Get("/:nodeId/devices", handleListDevices(dataSvc))
	nodes.Post("/:nodeId/devices/reconcile", handleReconcileDevices(dataSvc, registrySvc))
	nodes.Get("/:nodeId/devices/:deviceId", handleGetDevice(dataSvc))

	// 点位管理
	nodes.Get("/:nodeId/devices/:deviceId/points", handleListPoints(dataSvc))
	nodes.Get("/:nodeId/devices/:deviceId/snapshot", handleGetSnapshot(dataSvc))

	// 命令控制
	nodes.Post("/:nodeId/devices/:deviceId/commands", handleSendCommand(controlSvc, messagingManager))
	nodes.Get("/:nodeId/devices/:deviceId/commands", handleListCommands(controlSvc))
	nodes.Get("/:nodeId/devices/:deviceId/commands/:cmdId", handleGetCommand(controlSvc))

	// 全局命令列表
	protected.Get("/commands", handleListCommands(controlSvc))
	// 清空命令记录
	protected.Delete("/commands", handleClearCommands(controlSvc))

	// 告警管理
	alerts := protected.Group("/alerts")
	alerts.Get("/", handleListAlerts(alertSvc))
	alerts.Post("/:id/acknowledge", handleAcknowledgeAlert(alertSvc))

	// EdgeX 节点发现
	edgex := protected.Group("/edgex")
	edgex.Get("/nodes", GetEdgeXNodes(discoveryService))
	edgex.Get("/nodes/:id", GetEdgeXNode(discoveryService))
	edgex.Post("/nodes", AddEdgeXNode(discoveryService))
	edgex.Post("/scan", ScanEdgeXNodes(discoveryService))
	// Stage 2: EdgeOS 主动触发 EdgeX 节点重新注册
	edgex.Post("/discover", handleNodeDiscovery(eanBus, messagingManager))
	edgex.Post("/discover/:middlewareId", handleNodeDiscoveryTo(messagingManager))

	// ===========================
	// 系统管理（服务重启 / 配置导出 / 数据管理）
	// ===========================
	system := protected.Group("/system")
	system.Post("/restart", handleServiceRestart())
	system.Get("/export-config", handleExportConfig(configDB, registrySvc, dataSvc))

	// 数据管理（数据库概览 / 配置库备份 / 运行时库清理与压缩）
	if store != nil {
		data := protected.Group("/data")
		data.Get("/stats", handleDataStats(store))
		data.Post("/backup-config", handleBackupConfig(store))
		data.Post("/clear-cache", handleClearRuntimeBuckets(store))
		data.Post("/clear-all-runtime", handleClearAllRuntime(store))
		data.Post("/compact-runtime", handleCompactRuntime(store))
	}

	// ===========================
	// EAN 2.0 API 路由（与 V1 edgex/* 主题并存，互不影响）
	// ===========================
	RegisterEANRoutes(protected, eanBus)
}
