package server

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.etcd.io/bbolt"

	"github.com/anviod/edgeOS/internal/config"
	"github.com/anviod/edgeOS/internal/storage"
)

// ===========================
// 安装引导 | Install wizard
// ===========================

// InstallStatusResponse 安装状态响应 | Install status response
type InstallStatusResponse struct {
	IsInstalled bool   `json:"is_installed"`
	Initialized bool   `json:"initialized"`
	Message     string `json:"message"`
}

// handleInstallStatus 检查系统是否已初始化安装
// GET /api/install/status
// 无数据库配置（config.db 不存在或为空）时返回 is_installed=false，前端必须进入安装引导；
// 已有配置时返回 is_installed=true，前端不得再次进入安装引导。
// Check whether the system has been installed. Returns is_installed=false when config.db
// is missing/empty (frontend MUST show the install wizard); true when config exists (frontend
// MUST NOT show the install wizard again).
func handleInstallStatus(configDB *bbolt.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if configDB == nil {
			return apiError(c, fiber.StatusInternalServerError, "配置数据库未初始化 | Config database not initialized")
		}
		configStore, err := storage.NewConfigStore(configDB)
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError,
				fmt.Sprintf("创建配置存储失败 | Failed to create config store: %v", err))
		}
		has, err := configStore.HasConfigData()
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError,
				fmt.Sprintf("检查配置失败 | Failed to check config: %v", err))
		}

		msg := "系统未初始化，请完成安装引导 | System not initialized, please complete the install wizard"
		if has {
			msg = "系统已初始化 | System already initialized"
		}
		return apiSuccess(c, InstallStatusResponse{
			IsInstalled: has,
			Initialized: has,
			Message:     msg,
		})
	}
}

// handleInstall 执行安装引导：写入配置到 config.db 并触发服务重启
// POST /api/install
// Body 为 config.Config 的 JSON 结构（可部分提交，缺省字段使用默认值）。
// 仅允许在未初始化（无配置数据）时调用；已初始化时返回 409，确保不重复安装引导。
// Performs install: persists submitted config to config.db and triggers a service restart.
// Only allowed when no configuration data exists; returns 409 when already installed.
// restart 为重启回调（生产环境为 os.Interrupt 触发优雅重启；测试注入 no-op）。
func handleInstall(configDB *bbolt.DB, restart func()) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if configDB == nil {
			return apiError(c, fiber.StatusInternalServerError, "配置数据库未初始化 | Config database not initialized")
		}

		configStore, err := storage.NewConfigStore(configDB)
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError,
				fmt.Sprintf("创建配置存储失败 | Failed to create config store: %v", err))
		}
		has, err := configStore.HasConfigData()
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError,
				fmt.Sprintf("检查配置失败 | Failed to check config: %v", err))
		}
		if has {
			return apiError(c, fiber.StatusConflict, "系统已初始化，请勿重复安装 | System already installed, skip the wizard")
		}

		var submitted config.Config
		if err := c.BodyParser(&submitted); err != nil {
			return apiError(c, fiber.StatusBadRequest, "无效的请求体 | Invalid request body: "+err.Error())
		}

		if err := validateInstallConfig(&submitted); err != nil {
			return apiError(c, fiber.StatusBadRequest, err.Error())
		}

		cfg := mergeInstallConfig(&submitted)

		if err := config.SaveConfigToDB(configDB, cfg); err != nil {
			return apiError(c, fiber.StatusInternalServerError,
				fmt.Sprintf("保存配置失败 | Failed to save config: %v", err))
		}

		// 异步触发优雅重启：以新配置启动完整运行时（systemd Restart=always 会自动拉起）
		// Async graceful restart so the runtime boots with the new config (systemd restarts it)
		if restart != nil {
			go restart()
		}

		return apiSuccess(c, fiber.Map{
			"status":  "installed",
			"message": "安装完成，系统将自动重启 | Installation complete, the system will restart",
		})
	}
}

// triggerServiceRestart 异步发送 os.Interrupt 触发优雅关闭（systemd Restart=always 自动重启）
// Async os.Interrupt for graceful shutdown; systemd restarts the process automatically
func triggerServiceRestart() {
	time.Sleep(500 * time.Millisecond)
	if p, err := os.FindProcess(os.Getpid()); err == nil {
		_ = p.Signal(os.Interrupt)
	}
}

// mergeInstallConfig 用安装表单覆盖默认配置；仅覆盖非零字段
// Merge submitted install fields over DefaultConfig; only non-zero fields override
func mergeInstallConfig(in *config.Config) *config.Config {
	cfg := config.DefaultConfig()

	if in.User.Username != "" {
		cfg.User.Username = in.User.Username
	}
	if in.User.Password != "" {
		cfg.User.Password = in.User.Password
	}
	if in.User.Role != "" {
		cfg.User.Role = in.User.Role
	}

	if in.Node.NodeID != "" {
		cfg.Node.NodeID = in.Node.NodeID
	}
	if in.Node.NodeType != "" {
		cfg.Node.NodeType = in.Node.NodeType
	}
	if in.Node.PrimaryNodeID != "" {
		cfg.Node.PrimaryNodeID = in.Node.PrimaryNodeID
	}
	if in.Node.Listen != "" {
		cfg.Node.Listen = in.Node.Listen
	}

	if in.Security.JWTSecret != "" {
		cfg.Security.JWTSecret = in.Security.JWTSecret
	}
	if in.Security.TLSEnabled {
		cfg.Security.TLSEnabled = in.Security.TLSEnabled
	}

	if in.Monitoring.Enabled {
		cfg.Monitoring.Enabled = in.Monitoring.Enabled
	}
	if in.Monitoring.Prometheus != "" {
		cfg.Monitoring.Prometheus = in.Monitoring.Prometheus
	}

	// 中间件列表整体替换（安装表单提交的即为最终清单）
	cfg.Middlewares = in.Middlewares

	// EAN：任意子字段被提交则整体采用
	if in.EAN.Enabled || in.EAN.PlannerID != "" ||
		in.EAN.MQTT.Broker != "" || in.EAN.NATS.URL != "" ||
		in.EAN.Heartbeat.CheckIntervalSec != 0 || in.EAN.Heartbeat.TimeoutMultiplier != 0 {
		cfg.EAN = in.EAN
	}

	// 兜底默认值 | Fallback defaults
	if cfg.Node.Listen == "" {
		cfg.Node.Listen = ":8000"
	}
	if cfg.User.Role == "" {
		cfg.User.Role = "admin"
	}
	if cfg.EAN.Enabled && cfg.EAN.PlannerID == "" {
		cfg.EAN.PlannerID = cfg.Node.NodeID
	}

	return cfg
}

// validateInstallConfig 校验安装配置必填项（针对客户端提交值校验，禁止依赖默认值补齐必填项）
// Validate required install config fields (validates submitted values; required fields cannot be
// satisfied by defaults)
func validateInstallConfig(submitted *config.Config) error {
	if submitted.Node.NodeID == "" {
		return fmt.Errorf("节点ID不能为空 | node_id is required")
	}
	switch submitted.Node.NodeType {
	case "primary", "secondary", "collector":
	case "":
		return fmt.Errorf("节点类型不能为空 | node_type is required")
	default:
		return fmt.Errorf("无效的节点类型: %s（可选 primary/secondary/collector）", submitted.Node.NodeType)
	}
	if submitted.Node.NodeType == "secondary" && submitted.Node.PrimaryNodeID == "" {
		return fmt.Errorf("备用节点必须配置主节点ID | secondary node requires primary_node_id")
	}
	if submitted.User.Username == "" {
		return fmt.Errorf("管理员用户名不能为空 | username is required")
	}
	if len(submitted.User.Password) < 8 {
		return fmt.Errorf("管理员密码长度至少8位 | password must be at least 8 characters")
	}
	return nil
}
