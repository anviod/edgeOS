package server

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/anviod/edgeOS/internal/ean"
)

// ===========================
// EAN API 路由注册
// ===========================

// RegisterEANRoutes 注册 EAN 2.0 API 路由
// eanBus 为 nil 时所有 EAN 路由返回 503 Service Unavailable
func RegisterEANRoutes(router fiber.Router, eanBus *ean.Bus) {
	// EAN 路由组（受 JWT 保护）
	eanGroup := router.Group("/ean")

	// ---- Agent 管理 ----
	eanGroup.Get("/agents", handleEANListAgents(eanBus))
	eanGroup.Get("/agents/:id", handleEANGetAgent(eanBus))
	eanGroup.Get("/agents/:id/capabilities", handleEANListCapabilities(eanBus))

	// ---- Invoke 编排 ----
	eanGroup.Post("/invoke", handleEANInvoke(eanBus))

	// ---- 事件查询 ----
	eanGroup.Get("/events/recent", handleEANRecentEvents(eanBus))

	// ---- 审计查询 ----
	eanGroup.Get("/audit", handleEANAuditRecords(eanBus))

	// ---- 治理策略 ----
	eanGroup.Post("/governance/policies", handleEANSetPolicy(eanBus))

	// ---- 健康检查 ----
	eanGroup.Get("/health", handleEANHealth(eanBus))
}

// ===========================
// Agent 管理 Handler
// ===========================

// handleEANListAgents 列出所有 Agent
// GET /api/ean/agents
func handleEANListAgents(eanBus *ean.Bus) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if eanBus == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "EAN Bus 未启用")
		}
		agents := eanBus.GetDiscovery().ListAgents()
		return apiSuccess(c, fiber.Map{"agents": agents, "total": len(agents)})
	}
}

// handleEANGetAgent 获取指定 Agent 详情
// GET /api/ean/agents/:id
func handleEANGetAgent(eanBus *ean.Bus) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if eanBus == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "EAN Bus 未启用")
		}
		agentID := c.Params("id")
		agent, ok := eanBus.GetDiscovery().GetAgent(agentID)
		if !ok {
			return apiError(c, fiber.StatusNotFound, "Agent 未找到")
		}
		return apiSuccess(c, agent)
	}
}

// handleEANListCapabilities 列出指定 Agent 的所有 Capability
// GET /api/ean/agents/:id/capabilities
func handleEANListCapabilities(eanBus *ean.Bus) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if eanBus == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "EAN Bus 未启用")
		}
		agentID := c.Params("id")
		dc := eanBus.GetDiscovery()
		caps := dc.GetCapabilitiesByAgent(agentID)
		native, _ := dc.CountCapabilitiesBySource(agentID)

		// 附带来源标记（OS-P3-01 后 V1 Bridge Cap 不再生成，source 始终为 native-ean）
		enriched := make([]fiber.Map, 0, len(caps))
		for _, cap := range caps {
			src, _ := dc.GetCapabilitySource(cap.ID)
			enriched = append(enriched, fiber.Map{
				"id":            cap.ID,
				"agent_id":      cap.AgentID,
				"description":   cap.Description,
				"category":      cap.Category,
				"input_schema":  cap.InputSchema,
				"output_schema": cap.OutputSchema,
				"timeout_sec":   cap.TimeoutSec,
				"permission":    cap.Permission,
				"metadata":      cap.Metadata,
				"source":        string(src),
			})
		}
		return apiSuccess(c, fiber.Map{
			"agent_id":         agentID,
			"capabilities":     enriched,
			"total":            len(enriched),
			"native_ean_caps":  native,
		})
	}
}

// ===========================
// Invoke 编排 Handler
// ===========================

// handleEANInvoke 调用指定 Agent 的 Capability
// POST /api/ean/invoke
// Body: { "target": "agent-id", "capability": "cap-id", "arguments": {}, "timeout_sec": 30 }
func handleEANInvoke(eanBus *ean.Bus) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if eanBus == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "EAN Bus 未启用")
		}
		var req struct {
			Target      string                 `json:"target"`
			Capability  string                 `json:"capability"`
			Arguments   map[string]interface{} `json:"arguments"`
			TimeoutSec  int                    `json:"timeout_sec"`
			TenantID    string                 `json:"tenant_id"`
		}
		if err := c.BodyParser(&req); err != nil {
			return apiError(c, fiber.StatusBadRequest, "请求体解析失败")
		}
		if req.Target == "" {
			return apiError(c, fiber.StatusBadRequest, "target 不能为空")
		}
		if req.Capability == "" {
			return apiError(c, fiber.StatusBadRequest, "capability 不能为空")
		}

		opts := &ean.InvokeOptions{
			TenantID:  req.TenantID,
			TimeoutSec: req.TimeoutSec,
		}

		result, err := eanBus.InvokeCapability(c.Context(), req.Target, req.Capability, req.Arguments, opts)
		if err != nil {
			return apiError(c, fiber.StatusBadGateway, err.Error())
		}

		return apiSuccess(c, result)
	}
}

// ===========================
// 事件查询 Handler
// ===========================

// handleEANRecentEvents 查询最近 N 条事件
// GET /api/ean/events/recent?n=100
func handleEANRecentEvents(eanBus *ean.Bus) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if eanBus == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "EAN Bus 未启用")
		}
		nStr := c.Query("n", "100")
		n, err := strconv.Atoi(nStr)
		if err != nil || n <= 0 {
			n = 100
		}
		events := eanBus.GetEvent().RecentEvents(n)
		return apiSuccess(c, fiber.Map{
			"events": events,
			"count":  len(events),
		})
	}
}

// ===========================
// 审计查询 Handler
// ===========================

// handleEANAuditRecords 查询审计记录
// GET /api/ean/audit?limit=100
func handleEANAuditRecords(eanBus *ean.Bus) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if eanBus == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "EAN Bus 未启用")
		}
		limitStr := c.Query("limit", "100")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 100
		}
		records := eanBus.GetGovernance().QueryAuditRecords("", "", "", limit)
		return apiSuccess(c, fiber.Map{
			"records": records,
			"count":   len(records),
		})
	}
}

// ===========================
// 治理策略 Handler
// ===========================

// handleEANSetPolicy 设置租户策略
// POST /api/ean/governance/policies
// Body: { "tenant_id": "...", "allow_cap": [...], "deny_cap": [...], "allow_target": [...], "deny_target": [...] }
func handleEANSetPolicy(eanBus *ean.Bus) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if eanBus == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "EAN Bus 未启用")
		}
		var req struct {
			TenantID     string   `json:"tenant_id"`
			AllowCap     []string `json:"allow_cap"`
			DenyCap      []string `json:"deny_cap"`
			AllowTarget  []string `json:"allow_target"`
			DenyTarget   []string `json:"deny_target"`
		}
		if err := c.BodyParser(&req); err != nil {
			return apiError(c, fiber.StatusBadRequest, "请求体解析失败")
		}
		if req.TenantID == "" {
			return apiError(c, fiber.StatusBadRequest, "tenant_id 不能为空")
		}

		policy := &ean.TenantPolicy{
			TenantID:     req.TenantID,
			AllowCap:     req.AllowCap,
			DenyCap:      req.DenyCap,
			AllowTarget:  req.AllowTarget,
			DenyTarget:   req.DenyTarget,
		}
		eanBus.GetGovernance().SetPolicy(policy)

		return apiSuccess(c, fiber.Map{
			"tenant_id":    req.TenantID,
			"policy_set":   true,
		})
	}
}

// ===========================
// 健康检查 Handler
// ===========================

// handleEANHealth 返回 EAN Bus 健康状态
// GET /api/ean/health
func handleEANHealth(eanBus *ean.Bus) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if eanBus == nil {
			return apiSuccess(c, fiber.Map{
				"status":  "disabled",
				"message": "EAN Bus 未启用",
			})
		}
		health := eanBus.Health()
		health["status"] = "ok"
		return apiSuccess(c, health)
	}
}
