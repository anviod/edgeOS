package server

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/anviod/edgeOS/internal/core"
	"github.com/anviod/edgeOS/internal/discovery"
)

// RegisterRoutes 保留向后兼容的路由注册（仅 EdgeX 发现）
func RegisterRoutes(app *fiber.App, node core.Node, discoveryService *discovery.DiscoveryService) {
	api := app.Group("/api")
	edgex := api.Group("/edgex")
	edgex.Get("/nodes", getEdgeXNodes(discoveryService))
	edgex.Get("/nodes/:id", getEdgeXNode(discoveryService))
	edgex.Post("/nodes", addEdgeXNode(discoveryService))
	edgex.Post("/scan", scanEdgeXNodes(discoveryService))
	_ = node
}

// ── EdgeX Discovery helpers ───────────────────────────────────────────────────

func getEdgeXNodes(discoveryService *discovery.DiscoveryService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if discoveryService == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "Discovery service is unavailable")
		}
		nodes, err := discoveryService.ListNodes()
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError, fmt.Sprintf("List EdgeX nodes failed: %v", err))
		}
		return apiSuccess(c, fiber.Map{"nodes": nodes})
	}
}

func getEdgeXNode(discoveryService *discovery.DiscoveryService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if discoveryService == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "Discovery service is unavailable")
		}
		id := strings.TrimSpace(c.Params("id"))
		if id == "" {
			return apiError(c, fiber.StatusBadRequest, "node id is required")
		}
		node, err := discoveryService.GetNode(id)
		if err != nil {
			return apiError(c, fiber.StatusNotFound, "EdgeX node not found")
		}
		return apiSuccess(c, fiber.Map{"node": node})
	}
}

func addEdgeXNode(discoveryService *discovery.DiscoveryService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if discoveryService == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "Discovery service is unavailable")
		}
		var req struct {
			IP       string `json:"ip"`
			Port     string `json:"port"`
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BodyParser(&req); err != nil {
			return apiError(c, fiber.StatusBadRequest, "Invalid request body")
		}
		req.IP = strings.TrimSpace(req.IP)
		if req.IP == "" {
			return apiError(c, fiber.StatusBadRequest, "ip is required")
		}
		node, err := discoveryService.AddNode(req.IP, req.Port, req.Username, req.Password)
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError, fmt.Sprintf("Add EdgeX node failed: %v", err))
		}
		return apiSuccess(c, fiber.Map{"node": node})
	}
}

func scanEdgeXNodes(discoveryService *discovery.DiscoveryService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if discoveryService == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "Discovery service is unavailable")
		}
		targets := []string{"127.0.0.1:8082", "localhost:8082"}
		var foundNodes []interface{}
		for _, target := range targets {
			node, err := discoveryService.ScanIP(target)
			if err == nil && node != nil {
				foundNodes = append(foundNodes, map[string]interface{}{
					"nodeId":   node.NodeID,
					"nodeName": node.NodeName,
					"status":   "found",
				})
			}
		}
		if foundNodes == nil {
			foundNodes = []interface{}{}
		}
		if len(foundNodes) == 0 {
			return apiSuccess(c, fiber.Map{
				"status":  "scanning",
				"message": "No EdgeX nodes found",
				"nodes":   foundNodes,
			})
		}
		return apiSuccess(c, fiber.Map{
			"status":  "success",
			"message": fmt.Sprintf("Found %d EdgeX node(s)", len(foundNodes)),
			"nodes":   foundNodes,
		})
	}
}

// Exported aliases
func GetEdgeXNodes(d *discovery.DiscoveryService) fiber.Handler  { return getEdgeXNodes(d) }
func GetEdgeXNode(d *discovery.DiscoveryService) fiber.Handler   { return getEdgeXNode(d) }
func AddEdgeXNode(d *discovery.DiscoveryService) fiber.Handler   { return addEdgeXNode(d) }
func ScanEdgeXNodes(d *discovery.DiscoveryService) fiber.Handler { return scanEdgeXNodes(d) }
