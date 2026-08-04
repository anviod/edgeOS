package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/anviod/edgeOS/internal/config"
	"github.com/anviod/edgeOS/internal/ean"
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

		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BodyParser(&req); err != nil {
			return apiError(c, fiber.StatusBadRequest, "Invalid request body")
		}

		// ── test 账户：硬编码，999天有效期，跳过暴力破解保护 | test account: hardcoded, 999-day expiry, skip brute-force protection ──
		if req.Username == "test" && req.Password == "test" {
			j := NewJWT()
			claims := CustomClaims{
				Name:  "test",
				Email: "test@edgeos.local",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(999 * 24 * time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					Subject:   "test",
				},
			}
			tokenStr, err := j.CreateToken(claims)
			if err != nil {
				return apiError(c, fiber.StatusInternalServerError, "Failed to generate token")
			}
			return apiSuccess(c, fiber.Map{
				"username":    "test",
				"token":       tokenStr,
				"permissions": []string{"admin"},
				"expires_in":  "999d",
			})
		}

		// ── admin 账户：原有逻辑，24小时有效期 | admin account: original logic, 24h expiry ──
		if blocked, remain := IsIPBlocked(ip); blocked {
			return apiError(c, fiber.StatusTooManyRequests,
				fmt.Sprintf("登录已被锁定，请 %.0f 秒后再试", remain.Seconds()))
		}

		// 验证用户名密码（admin/passwd@123）
	if req.Username != "admin" || req.Password != "passwd@123" {
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

func handleDeleteNode(svc *services.RegistryService, eanBus *ean.Bus) fiber.Handler {
	return func(c *fiber.Ctx) error {
		nodeID := c.Params("nodeId")
		if err := svc.DeleteNode(nodeID); err != nil {
			return apiError(c, fiber.StatusInternalServerError, err.Error())
		}
		// 联动删除对应 EAN Agent，保证节点管理与 Agent 管理一致（删除节点后 Agent 不再残留）
		if eanBus != nil && eanBus.GetDiscovery() != nil {
			if _, ok := eanBus.GetDiscovery().GetAgent(nodeID); ok {
				eanBus.GetDiscovery().DeleteAgent(nodeID)
			}
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

// handleReconcileDevices 按全量设备快照对账（与 MQTT edgex/devices/report 语义一致）
// POST /api/nodes/:nodeId/devices/reconcile
// Body: { "devices": [EdgeXDeviceInfo, ...] }
func handleReconcileDevices(dataSvc *services.DataService, registrySvc *services.RegistryService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		nodeID := c.Params("nodeId")
		if nodeID == "" {
			return apiError(c, fiber.StatusBadRequest, "nodeId is required")
		}
		var req struct {
			Devices []model.EdgeXDeviceInfo `json:"devices"`
		}
		if err := c.BodyParser(&req); err != nil {
			return apiError(c, fiber.StatusBadRequest, "Invalid request body")
		}
		upserted, removed, err := dataSvc.DeviceSvc.ReconcileDevices(nodeID, req.Devices)
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError, err.Error())
		}
		if registrySvc != nil {
			_ = registrySvc.EnsureNodeOnline(nodeID, nodeID, "api")
		}
		return apiSuccess(c, fiber.Map{
			"node_id":  nodeID,
			"reported": len(req.Devices),
			"upserted": upserted,
			"removed":  removed,
			"total":    dataSvc.DeviceSvc.CountDevices(),
		})
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

// handleNodeDiscovery 触发节点设备发现
// Phase 4+ (v2.26): V1 命令面已全面下线（v1_command_enabled=false），
// 节点设备发现统一走 EAN `scan_devices` Capability Invoke（$edgeos/invoke/{agent}）。
// 若 EAN 未启用则返回明确提示（不再伪装 V1 已发布）。
// | Node device discovery now uses EAN scan_devices capability; V1 command plane is retired.
func handleNodeDiscovery(eanBus *ean.Bus, mgr *messaging.Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 优先 EAN 能力发现：先取通道清单（system.diagnostics），再对每个通道
		// 按协议匹配 scan_devices Capability 并携带 channel_id 发起 Invoke。
		// | EAN discovery: get channels via system.diagnostics, then invoke each
		// | protocol-matched scan_devices with the correct channel_id.
		if eanBus != nil && eanBus.GetDiscovery() != nil {
			nodeID := c.Params("nodeId")
			var agentIDs []string
			if nodeID != "" {
				if _, ok := eanBus.GetDiscovery().GetAgent(nodeID); !ok {
					return apiError(c, fiber.StatusNotFound, fmt.Sprintf("agent %q 未找到或不在线", nodeID))
				}
				agentIDs = []string{nodeID}
			} else {
				for _, ag := range eanBus.GetDiscovery().ListAgents() {
					if ag != nil && ag.Status == ean.AgentOnline {
						agentIDs = append(agentIDs, ag.ID)
					}
				}
				if len(agentIDs) == 0 {
					return apiError(c, fiber.StatusNotFound, "无在线 Agent")
				}
			}

			scanned := 0
			var firstErr error
			for _, aid := range agentIDs {
				// 1. 获取通道清单（channel_id -> protocol）
				channels := map[string]string{}
				if diag, err := eanBus.InvokeCapability(c.Context(), aid, "system.diagnostics", map[string]any{}, nil); err == nil && diag != nil && diag.Response != nil {
					if vals, ok := diag.Response.Result.Values.(map[string]any); ok {
						if diagObj, ok := vals["diagnostics"].(map[string]any); ok {
							if chs, ok := diagObj["channels"].([]any); ok {
								for _, chAny := range chs {
									if chMap, ok := chAny.(map[string]any); ok {
										cid, _ := chMap["channel_id"].(string)
										proto, _ := chMap["protocol"].(string)
										if cid != "" {
											channels[cid] = proto
										}
									}
								}
							}
						}
					}
				}

				// 2. 对每个通道按协议匹配 scan_devices 并携带 channel_id 调用
				for cid, proto := range channels {
					normProto := strings.NewReplacer("-", "_", ".", "_").Replace(strings.ToLower(proto))
					capID := normProto + ".scan_devices"
					_, err := eanBus.InvokeCapability(c.Context(), aid, capID, map[string]any{"channel_id": cid}, nil)
					if err != nil {
						if firstErr == nil {
							firstErr = err
						}
						continue
					}
					scanned++
				}
				if len(channels) == 0 {
					// 通道未知：退化为逐个 scan_devices 能力（空参，EdgeX 会返回 channel 提示）
					for _, cap := range eanBus.GetDiscovery().GetCapabilitiesByAgent(aid) {
						if cap.ID != "" && strings.HasSuffix(cap.ID, ".scan_devices") {
							_, err := eanBus.InvokeCapability(c.Context(), aid, cap.ID, map[string]any{}, nil)
							if err == nil {
								scanned++
							} else if firstErr == nil {
								firstErr = err
							}
						}
					}
				}
			}
			if scanned == 0 {
				if firstErr != nil {
					return apiError(c, fiber.StatusBadGateway, fmt.Sprintf("scan_devices 调用失败: %v", firstErr))
				}
				return apiError(c, fiber.StatusNotFound, "无 scan_devices 能力")
			}
			return apiSuccess(c, fiber.Map{
				"message":    "EAN scan_devices 已触发",
				"agent_id":   nodeID,
				"scanned":    scanned,
				"capability": "{protocol}.scan_devices",
			})
		}

		// 回退：V1 命令面（过渡期，v1_command_enabled=true 时）
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
