package server

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/anviod/edgeOS/internal/config"
	"github.com/anviod/edgeOS/internal/messaging"
	"github.com/anviod/edgeOS/internal/model"
	"github.com/anviod/edgeOS/internal/services"
)

// ===========================
// 认证
// ===========================

func handleLogin(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()
		if blocked, remain := IsIPBlocked(ip); blocked {
			return apiError(c, fiber.StatusTooManyRequests,
				fmt.Sprintf("登录已被锁定，请 %.0f 秒后再试", remain.Seconds()))
		}

		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BodyParser(&req); err != nil {
			return apiError(c, fiber.StatusBadRequest, "Invalid request body")
		}

		// 验证用户名密码（admin/admin）
		if req.Username != "admin" || req.Password != "admin" {
			AddLoginFail(ip)
			return apiError(c, fiber.StatusUnauthorized, "用户名或密码错误")
		}
		ClearLoginFail(ip)

		// 使用 NewJWT 与 JWTAuth 中间件共享同一密钥
		j := NewJWT()
		claims := CustomClaims{
			Name:  req.Username,
			Email: req.Username + "@edgeos.local",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Subject:   req.Username,
			},
		}
		tokenStr, err := j.CreateToken(claims)
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError, "Failed to generate token")
		}

		_ = cfg
		return apiSuccess(c, fiber.Map{
			"username":    req.Username,
			"token":       tokenStr,
			"permissions": []string{"admin"},
		})
	}
}

// ===========================
// 仪表盘
// ===========================

func handleDashboardStats(
	registrySvc *services.RegistryService,
	dataSvc *services.DataService,
	alertSvc *services.AlertService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		total, online := registrySvc.CountNodes()
		deviceCount := dataSvc.DeviceSvc.CountDevices()
		alertCount := alertSvc.CountAlerts()
		return apiSuccess(c, fiber.Map{
			"total_nodes":   total,
			"online_nodes":  online,
			"total_devices": deviceCount,
			"today_alerts":  alertCount,
		})
	}
}

// ===========================
// 中间件管理
// ===========================

func handleListMiddlewares(svc *services.MiddlewareService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		list, err := svc.List()
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError, err.Error())
		}
		return apiSuccess(c, fiber.Map{"middlewares": list})
	}
}

func handleCreateMiddleware(svc *services.MiddlewareService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var cfg model.MiddlewareConfig
		if err := c.BodyParser(&cfg); err != nil {
			return apiError(c, fiber.StatusBadRequest, "Invalid request body")
		}
		fmt.Printf("DEBUG handleCreateMiddleware: Host=%s Port=%d Broker=%s ClientID=%s\n",
			cfg.Host, cfg.Port, cfg.Broker, cfg.ClientID)
		if err := svc.Create(&cfg); err != nil {
			return apiError(c, fiber.StatusInternalServerError, err.Error())
		}
		fmt.Printf("DEBUG after Create: Host=%s Port=%d Broker=%s\n",
			cfg.Host, cfg.Port, cfg.Broker)
		return apiSuccess(c, fiber.Map{"middleware": cfg})
	}
}

func handleUpdateMiddleware(svc *services.MiddlewareService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		var cfg model.MiddlewareConfig
		if err := c.BodyParser(&cfg); err != nil {
			return apiError(c, fiber.StatusBadRequest, "Invalid request body")
		}
		cfg.ID = id
		if err := svc.Update(&cfg); err != nil {
			return apiError(c, fiber.StatusInternalServerError, err.Error())
		}
		return apiSuccess(c, fiber.Map{"middleware": cfg})
	}
}

func handleDeleteMiddleware(svc *services.MiddlewareService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		if err := svc.Delete(id); err != nil {
			return apiError(c, fiber.StatusInternalServerError, err.Error())
		}
		return apiSuccess(c, nil)
	}
}

func handleConnectMiddleware(svc *services.MiddlewareService, mgr *messaging.Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		_, err := svc.Get(id)
		if err != nil {
			return apiError(c, fiber.StatusNotFound, "Middleware not found")
		}
		if mgr != nil {
			if err := mgr.Connect(id); err != nil {
				svc.UpdateStatus(id, "error", err.Error())
				return apiError(c, fiber.StatusBadGateway, err.Error())
			}
			svc.UpdateStatus(id, "connected", "")
			return apiSuccess(c, fiber.Map{"status": "connected"})
		} else {
			return apiError(c, fiber.StatusServiceUnavailable, "Messaging manager not initialized")
		}
	}
}

func handleDisconnectMiddleware(svc *services.MiddlewareService, mgr *messaging.Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		if mgr != nil {
			if err := mgr.Disconnect(id); err != nil {
				svc.UpdateStatus(id, "disconnected", err.Error())
				return apiError(c, fiber.StatusInternalServerError, err.Error())
			}
		}
		svc.UpdateStatus(id, "disconnected", "")
		return apiSuccess(c, fiber.Map{"status": "disconnected"})
	}
}

func handleGetMiddlewareStatus(svc *services.MiddlewareService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		cfg, err := svc.Get(id)
		if err != nil {
			return apiError(c, fiber.StatusNotFound, "Middleware not found")
		}
		return apiSuccess(c, fiber.Map{
			"id":         cfg.ID,
			"name":       cfg.Name,
			"type":       cfg.Type,
			"status":     cfg.Status,
			"last_error": cfg.LastError,
			"host":       cfg.Host,
			"port":       cfg.Port,
			"client_id":  cfg.ClientID,
			"enabled":    cfg.Enabled,
		})
	}
}

// ===========================
// 节点管理
// ===========================

func handleListNodes(svc *services.RegistryService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		nodes, err := svc.ListNodes()
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError, err.Error())
		}
		return apiSuccess(c, fiber.Map{"nodes": nodes})
	}
}

func handleGetNode(svc *services.RegistryService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		nodeID := c.Params("nodeId")
		node, err := svc.GetNode(nodeID)
		if err != nil {
			return apiError(c, fiber.StatusNotFound, "Node not found")
		}
		return apiSuccess(c, fiber.Map{"node": node})
	}
}

func handleDeleteNode(svc *services.RegistryService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		nodeID := c.Params("nodeId")
		if err := svc.DeleteNode(nodeID); err != nil {
			return apiError(c, fiber.StatusInternalServerError, err.Error())
		}
		return apiSuccess(c, nil)
	}
}

// ===========================
// 设备管理
// ===========================

func handleListDevices(dataSvc *services.DataService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		nodeID := c.Params("nodeId")
		devices, err := dataSvc.DeviceSvc.ListDevices(nodeID)
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError, err.Error())
		}
		return apiSuccess(c, fiber.Map{"devices": devices})
	}
}

func handleGetDevice(dataSvc *services.DataService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		nodeID := c.Params("nodeId")
		deviceID := c.Params("deviceId")
		device, err := dataSvc.DeviceSvc.GetDevice(nodeID, deviceID)
		if err != nil {
			return apiError(c, fiber.StatusNotFound, "Device not found")
		}
		return apiSuccess(c, fiber.Map{"device": device})
	}
}

// ===========================
// 点位管理
// ===========================

func handleListPoints(dataSvc *services.DataService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		nodeID := c.Params("nodeId")
		deviceID := c.Params("deviceId")
		points, err := dataSvc.PointService.ListByDevice(nodeID, deviceID)
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError, err.Error())
		}
		// 尝试附加快照值
		snapshot, _ := dataSvc.PointService.GetSnapshot(nodeID, deviceID)
		snapshotData := map[string]interface{}{}
		if snapshot != nil {
			snapshotData = snapshot.Points
		}
		return apiSuccess(c, fiber.Map{
			"points":   points,
			"snapshot": snapshotData,
		})
	}
}

func handleGetSnapshot(dataSvc *services.DataService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		nodeID := c.Params("nodeId")
		deviceID := c.Params("deviceId")
		snapshot, err := dataSvc.PointService.GetSnapshot(nodeID, deviceID)
		if err != nil {
			return apiSuccess(c, fiber.Map{"snapshot": nil})
		}
		return apiSuccess(c, fiber.Map{"snapshot": snapshot})
	}
}

// ===========================
// 命令控制
// ===========================

func handleSendCommand(controlSvc *services.ControlService, mgr *messaging.Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		nodeID := c.Params("nodeId")
		deviceID := c.Params("deviceId")
		var req struct {
			PointID string      `json:"point_id"`
			Value   interface{} `json:"value"`
		}
		if err := c.BodyParser(&req); err != nil {
			return apiError(c, fiber.StatusBadRequest, "Invalid request body")
		}
		cmd, err := controlSvc.CreateCommand(nodeID, deviceID, req.PointID, req.Value)
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError, err.Error())
		}
		if mgr != nil {
			if err := mgr.PublishCommand(nodeID, deviceID, req.PointID, req.Value, cmd.ID); err != nil {
				controlSvc.UpdateCommandStatus(cmd.ID, "error", err.Error())
				return apiError(c, fiber.StatusBadGateway, err.Error())
			}
		}
		return apiSuccess(c, fiber.Map{"command": cmd})
	}
}

func handleListCommands(controlSvc *services.ControlService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		nodeID := c.Params("nodeId")
		deviceID := c.Params("deviceId")
		cmds, err := controlSvc.ListCommands(nodeID, deviceID, 1000)
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError, err.Error())
		}
		return apiSuccess(c, fiber.Map{"commands": cmds})
	}
}

func handleGetCommand(controlSvc *services.ControlService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cmdID := c.Params("cmdId")
		cmd, err := controlSvc.GetCommand(cmdID)
		if err != nil {
			return apiError(c, fiber.StatusNotFound, "Command not found")
		}
		return apiSuccess(c, fiber.Map{"command": cmd})
	}
}

func handleClearCommands(controlSvc *services.ControlService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := controlSvc.ClearCommands()
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError, err.Error())
		}
		return apiSuccess(c, fiber.Map{"message": "Commands cleared successfully"})
	}
}

// ===========================
// 节点主动发现（Stage 2）
// ===========================

func handleNodeDiscovery(mgr *messaging.Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if mgr == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "Messaging manager not initialized")
		}
		if err := mgr.PublishNodeDiscovery(); err != nil {
			return apiError(c, fiber.StatusBadGateway, err.Error())
		}
		return apiSuccess(c, fiber.Map{
			"message": "Node discovery request published",
			"topic":   "edgex/cmd/nodes/register",
		})
	}
}

func handleNodeDiscoveryTo(mgr *messaging.Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if mgr == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "Messaging manager not initialized")
		}
		middlewareID := c.Params("middlewareId")
		if err := mgr.PublishNodeDiscoveryTo(middlewareID); err != nil {
			return apiError(c, fiber.StatusBadGateway, err.Error())
		}
		return apiSuccess(c, fiber.Map{
			"message":    "Node discovery request published",
			"middleware": middlewareID,
			"topic":      "edgex/cmd/nodes/register",
		})
	}
}

// ===========================
// 告警管理
// ===========================

func handleListAlerts(svc *services.AlertService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		status := c.Query("status", "")
		limit := 100
		alerts, err := svc.ListAlerts(status, limit)
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError, err.Error())
		}
		return apiSuccess(c, fiber.Map{"alerts": alerts})
	}
}

func handleAcknowledgeAlert(svc *services.AlertService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		username := "admin" // TODO: 从 JWT claims 获取
		if err := svc.AcknowledgeAlert(id, username); err != nil {
			return apiError(c, fiber.StatusInternalServerError, err.Error())
		}
		return apiSuccess(c, nil)
	}
}

// ===========================
// 通用响应辅助
// ===========================

func apiSuccess(c *fiber.Ctx, data interface{}) error {
	var response map[string]interface{}
	if data == nil {
		response = fiber.Map{"code": "0", "msg": "success"}
	} else {
		response = fiber.Map{"code": "0", "msg": "success", "data": data}
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "1", "msg": "Failed to marshal response"})
	}

	c.Set("Content-Type", "application/json; charset=utf-8")
	return c.Send(jsonData)
}

func apiError(c *fiber.Ctx, status int, msg string) error {
	c.Set("Content-Type", "application/json; charset=utf-8")
	return c.Status(status).JSON(fiber.Map{"code": "1", "msg": msg})
}
