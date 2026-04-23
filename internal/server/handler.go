package server

import (
	fiberws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/config"
	"github.com/anviod/edgeOS/internal/core"
	"github.com/anviod/edgeOS/internal/discovery"
	"github.com/anviod/edgeOS/internal/messaging"
	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/ws"
)

// RegisterAllRoutes 注册所有路由
func RegisterAllRoutes(
	app *fiber.App,
	node core.Node,
	db *bbolt.DB,
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
) {
	api := app.Group("/api")

	// ===========================
	// 公开路由（无需JWT）
	// ===========================

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
	nodes.Delete("/:nodeId", handleDeleteNode(registrySvc))
	nodes.Post("/:nodeId/discover", handleNodeDiscovery(messagingManager))

	// 设备管理
	nodes.Get("/:nodeId/devices", handleListDevices(dataSvc))
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
	edgex.Post("/discover", handleNodeDiscovery(messagingManager))
	edgex.Post("/discover/:middlewareId", handleNodeDiscoveryTo(messagingManager))
}
