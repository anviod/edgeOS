package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/config"
	"github.com/anviod/edgeOS/internal/core"
	"github.com/anviod/edgeOS/internal/discovery"
	"github.com/anviod/edgeOS/internal/ean"
	"github.com/anviod/edgeOS/internal/messaging"
	"github.com/anviod/edgeOS/internal/server"
	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/storage"
	"github.com/anviod/edgeOS/internal/ws"
)

func main() {
	// 初始化日志 | Initialize logger
	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer zapLogger.Sync()

	zapLogger.Info("Starting EdgeOS (database-backed configuration)...")

	// ── 1. 初始化存储（config.db + edgeos.db）| Initialize storage ──────────
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		zapLogger.Fatal("Failed to create data directory", zap.String("dir", dataDir), zap.Error(err))
	}

	configPath := filepath.Join(dataDir, "config.db")
	configExists := false
	if _, err := os.Stat(configPath); err == nil {
		configExists = true
	}

	zapLogger.Info("Database check",
		zap.String("config_db", configPath),
		zap.Bool("config_exists", configExists))

	var store *storage.Storage
	var cfg *config.Config
	var cfgManager *config.ConfigManager
	var runtimeReady bool

	if configExists {
		// config.db 已存在，从数据库加载 | config.db exists, load from database
		zapLogger.Info("Initializing storage...", zap.String("dir", dataDir))
		store, err = storage.NewStorage(dataDir)
		if err != nil {
			zapLogger.Fatal("Failed to init storage", zap.Error(err))
		}
		zapLogger.Info("Storage initialized successfully",
			zap.String("config_db", store.GetConfigPath()),
			zap.String("runtime_db", store.GetRuntimePath()))

		// 检查配置库是否有业务数据 | Check if config DB has business data
		configStore, _ := storage.NewConfigStore(store.GetConfigDB())
		runtimeReady, _ = configStore.HasConfigData()
		if runtimeReady {
			zapLogger.Info("Configuration data found in config.db")
		} else {
			// config.db 存在但为空：进入安装引导模式（仅启动 Web 服务，等待安装向导写入配置）
			// Empty config.db: enter install mode (web UI only, wait for install wizard to persist config)
			zapLogger.Info("config.db is empty, entering install mode (web UI only)")
		}

		zapLogger.Info("Loading configuration from database...")
		cfgManager, err = config.NewConfigManagerWithDB(store.GetConfigDB())
		if err != nil {
			zapLogger.Fatal("Failed to load config with DB", zap.Error(err))
		}
		cfg = cfgManager.GetConfig()
		zapLogger.Info("Configuration loaded successfully from database")
	} else {
		// config.db 不存在：创建存储并进入安装引导模式（无 YAML 配置文件依赖，
		// 不使用默认配置落库，必须由安装向导写入后才进入运行模式）
		// config.db not found: create storage and enter install mode (no YAML file dependency;
		// defaults are NOT persisted — runtime only starts after the install wizard writes config)
		zapLogger.Info("config.db not found, creating new storage...")
		store, err = storage.NewStorage(dataDir)
		if err != nil {
			zapLogger.Fatal("Failed to init storage", zap.Error(err))
		}
		zapLogger.Info("Storage created",
			zap.String("config_db", store.GetConfigPath()),
			zap.String("runtime_db", store.GetRuntimePath()))

		zapLogger.Info("No configuration data, entering install mode (web UI only)")
		runtimeReady = false

		// 从数据库加载配置（无数据时返回默认配置，供安装向导预填）| Load config from DB (defaults when empty)
		cfgManager, err = config.NewConfigManagerWithDB(store.GetConfigDB())
		if err != nil {
			zapLogger.Fatal("Failed to load config with DB", zap.Error(err))
		}
		cfg = cfgManager.GetConfig()
		zapLogger.Info("Default config loaded for installation wizard")
	}

	defer store.Close()

	// 运行时数据库句柄（用于节点和服务层）| Runtime DB handle (for node and services)
	runtimeDB := store.GetRuntimeDB()

	// ── 2. 创建节点 | Create node ────────────────────────────────────────────
	node, err := createNode(cfg, runtimeDB)
	if err != nil {
		zapLogger.Fatal("Failed to create node", zap.Error(err))
	}

	// 安装模式下不启动节点/消息总线/EAN 运行时（仅 Web 服务），等待安装向导写入配置后重启生效
	// In install mode the node/messaging/EAN runtime is NOT started (web UI only); it starts
	// after the install wizard persists config and the process restarts.
	if runtimeReady {
		// 启动节点 | Start node
		ctx, cancel := context.WithCancel(context.Background())
		if err := node.Start(ctx); err != nil {
			zapLogger.Fatal("Failed to start node", zap.Error(err))
		}
		defer cancel()
	}

	// ── 3. 初始化服务层 | Initialize service layer ──────────────────────────
	hub := ws.NewHub(zapLogger)

	registrySvc := services.NewRegistryService(runtimeDB)
	dataSvc := services.NewDataService(runtimeDB)
	alertSvc := services.NewAlertService(runtimeDB)
	middlewareSvc := services.NewMiddlewareService(runtimeDB, zapLogger)
	controlSvc := services.NewControlService(runtimeDB, zapLogger)

	// 初始化消息管理器 | Initialize messaging manager
	messagingManager := messaging.NewManager(middlewareSvc, registrySvc, dataSvc, alertSvc, controlSvc, hub, zapLogger)
	var eanBus *ean.Bus

	if runtimeReady {
		// 从数据库配置初始化中间件 | Initialize middlewares from DB-backed config
		if err := middlewareSvc.InitFromConfig(cfg.Middlewares); err != nil {
			zapLogger.Warn("Failed to init middlewares from config", zap.Error(err))
		}

		// Phase 4 (OS-P4): V1 命令面开关——过渡期默认 true 保留运行；测试通过后配置 v1_command_enabled=false 全面下线
		// Phase 4 (OS-P4): V1 command plane switch — default true during transition; set false after tests pass
		messagingManager.SetV1CommandEnabled(cfg.EAN.V1CommandEnabled)
		if err := messagingManager.Start(); err != nil {
			zapLogger.Fatal("Failed to start messaging manager", zap.Error(err))
		}
		zapLogger.Info("Messaging manager started",
			zap.Bool("v1_command_enabled", cfg.EAN.V1CommandEnabled))

		// ── 4. 初始化 EAN 2.0 Bus（如果启用）| Initialize EAN Bus ────────────
		if cfg.EAN.Enabled {
			// 修正 EAN broker 地址：如果 EAN MQTT/NATS 地址为 127.0.0.1，则使用 V1 中间件的 broker 主机
			// Fix EAN broker address: if EAN MQTT/NATS is localhost, use V1 middleware's broker host
			eanHost := extractBrokerHost(cfg.Middlewares)
			if eanHost != "" && eanHost != "127.0.0.1" && eanHost != "localhost" {
				if cfg.EAN.MQTT.Enabled && isLocalhostBroker(cfg.EAN.MQTT.Broker) {
					zapLogger.Info("Fixing EAN MQTT broker address",
						zap.String("original", cfg.EAN.MQTT.Broker),
						zap.String("fixed_host", eanHost))
					cfg.EAN.MQTT.Broker = replaceBrokerHost(cfg.EAN.MQTT.Broker, eanHost)
				}
				if cfg.EAN.NATS.Enabled && isLocalhostBroker(cfg.EAN.NATS.URL) {
					zapLogger.Info("Fixing EAN NATS URL address",
						zap.String("original", cfg.EAN.NATS.URL),
						zap.String("fixed_host", eanHost))
					cfg.EAN.NATS.URL = replaceBrokerHost(cfg.EAN.NATS.URL, eanHost)
				}
			}

			busCfg := ean.BusConfig{
				PlannerID: cfg.EAN.PlannerID,
				MQTT:      cfg.EAN.MQTT,
				NATS:      cfg.EAN.NATS,
				Heartbeat: cfg.EAN.Heartbeat,
			}
			// 如果 PlannerID 为空则使用节点 ID | Use node ID if PlannerID is empty
			if busCfg.PlannerID == "" {
				busCfg.PlannerID = cfg.Node.NodeID
			}
			eanBus, err = ean.NewBus(busCfg, zapLogger)
			if err != nil {
				// EAN 创建失败不 fatal，进程降级继续 | EAN failure is not fatal, degraded mode
				zapLogger.Warn("Failed to create EAN Bus, running in degraded mode", zap.Error(err))
			}
			if eanBus != nil {
				if err := eanBus.Start(); err != nil {
					zapLogger.Warn("Failed to start EAN Bus, running in degraded mode", zap.Error(err))
				}
				zapLogger.Info("EAN 2.0 Bus started",
					zap.String("planner_id", busCfg.PlannerID),
					zap.Bool("mqtt_enabled", busCfg.MQTT.Enabled),
					zap.Bool("nats_enabled", busCfg.NATS.Enabled),
					zap.Strings("connected_transports", eanBus.Transport().ConnectedNames()))

				// Phase 4: V1→EAN 桥接器已下线（OS-P4）——原生 EAN Discovery/Heartbeat/Event 已完全覆盖，
				// 不再轮询 V1 节点合成 Agent/心跳/点位 Event；V1 数据面由 messaging 中间件与 V1 NATS 数据面桥接保留。
				// | Phase 4: V1→EAN bridge retired; native EAN flows fully replace it.

				// EAN Agent → V1 节点注册表镜像 | EAN Agent registry mirror
				eanBus.AttachRegistryMirror(registrySvc)

				// EAN Event → V1 设备/点位服务桥接 | EAN Event → V1 device/point service bridge
				eanBus.AttachEventBridge(dataSvc.DeviceSvc, dataSvc.PointService, registrySvc)

				// V1 NATS 数据面桥接（OS-23：MQTT/NATS 双传输对称订阅 V1 数据面）
				// V1 NATS data-plane bridge (OS-23: symmetric MQTT/NATS V1 data-plane subscription)
				eanBus.AttachV1NATSDataPlane(registrySvc, dataSvc.DeviceSvc, dataSvc.PointService, alertSvc, hub)
			}
		}
	} else {
		zapLogger.Info("Install mode: node/messaging/EAN runtime deferred until installation completes")
	}

	if messagingManager != nil {
		defer messagingManager.Stop()
	}
	if eanBus != nil {
		defer eanBus.Stop()
	}

	// ── 5. 初始化 HTTP 服务器 | Initialize HTTP server ──────────────────────
	app := initServer(cfg, node, runtimeDB, store.GetConfigDB(), store, hub, registrySvc, dataSvc, alertSvc, middlewareSvc, controlSvc, messagingManager, zapLogger, eanBus)

	// 启动 HTTP 服务器 | Start HTTP server
	serverAddr := cfg.Node.Listen
	if serverAddr == "" {
		serverAddr = ":8000"
	}

	// 优雅关闭 | Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		zapLogger.Info("Server starting", zap.String("addr", serverAddr))
		if err := app.Listen(serverAddr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-c
	zapLogger.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		zapLogger.Error("Server shutdown error", zap.Error(err))
	}

	if err := node.Stop(); err != nil {
		zapLogger.Error("Node stop error", zap.Error(err))
	}

	zapLogger.Info("Server stopped gracefully")

	// 避免未使用变量警告 | Avoid unused variable warning
	_ = cfgManager
}

// createNode 创建节点 | Create node
func createNode(cfg *config.Config, db *bbolt.DB) (core.Node, error) {
	nodeType := cfg.Node.NodeType
	nodeID := cfg.Node.NodeID

	switch nodeType {
	case "primary":
		return core.NewPrimaryQueen(nodeID, db), nil
	case "secondary":
		return core.NewSecondaryQueen(nodeID, db), nil
	case "collector":
		return core.NewEdgeCollector(nodeID, db), nil
	default:
		return nil, fmt.Errorf("invalid node type: %s", nodeType)
	}
}

// initServer 初始化 HTTP 服务器 | Initialize HTTP server
func initServer(
	cfg *config.Config,
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
	logger *zap.Logger,
	eanBus *ean.Bus,
) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	// 中间件 | Middleware
	app.Use(fiberlogger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// 注册所有路由 | Register all routes
	discoveryService := discovery.NewDiscoveryService(db, node.GetNodeID(), "edgeos-queen.local", "edgeos-shared-secret")
	if err := discoveryService.Start(); err != nil {
		logger.Error("Failed to start discovery service", zap.Error(err))
	}

	server.RegisterAllRoutes(app, node, db, configDB, store, hub, registrySvc, dataSvc, alertSvc, middlewareSvc, controlSvc, messagingManager, discoveryService, cfg, logger, eanBus)

	// 健康检查 | Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":      "ok",
			"node_id":     node.GetNodeID(),
			"node_type":   node.GetNodeType(),
			"node_status": node.GetStatus(),
		})
	})

	// 静态资源（优先相对可执行文件目录，兼容未设置 WorkingDirectory 的部署）
	// Static assets: prefer executable dir, fall back to relative ./ui/dist
	uiDist := uiDistDir()
	logger.Info("Static file serving", zap.String("ui_dist", uiDist))
	app.Static("/", uiDist)

	// SPA Fallback: 所有未匹配的路由都返回 index.html | SPA fallback
	app.Get("*", func(c *fiber.Ctx) error {
		return c.SendFile(filepath.Join(uiDist, "index.html"))
	})

	return app
}

// uiDistDir 按候选顺序查找 UI 静态资源目录 | Find UI dist dir by candidate order
// 优先可执行文件所在目录的 ui/dist，回退到相对路径 ./ui/dist
func uiDistDir() string {
	candidates := []string{"./ui/dist"}
	if exe, err := os.Executable(); err == nil {
		candidates = append([]string{filepath.Join(filepath.Dir(exe), "ui", "dist")}, candidates...)
	}
	for _, dir := range candidates {
		if info, err := os.Stat(filepath.Join(dir, "index.html")); err == nil && !info.IsDir() {
			return dir
		}
	}
	return "./ui/dist"
}

// extractBrokerHost 从 V1 中间件配置中提取第一个非 localhost 的 broker 主机地址
// Extract first non-localhost broker host from V1 middleware configs
func extractBrokerHost(middlewares []config.MiddlewareMiddlewareConfig) string {
	for _, mw := range middlewares {
		if !mw.Enabled || mw.Broker == "" {
			continue
		}
		host := parseBrokerHost(mw.Broker)
		if host != "" && host != "127.0.0.1" && host != "localhost" {
			return host
		}
	}
	return ""
}

// parseBrokerHost 从 broker URL 中提取主机地址 | Extract host from broker URL
func parseBrokerHost(broker string) string {
	for _, prefix := range []string{"tcp://", "ssl://", "mqtt://", "mqtts://", "nats://"} {
		broker = strings.TrimPrefix(broker, prefix)
	}
	if idx := strings.LastIndex(broker, ":"); idx != -1 {
		return broker[:idx]
	}
	return broker
}

// isLocalhostBroker 检查 broker URL 是否指向 localhost | Check if broker URL points to localhost
func isLocalhostBroker(broker string) bool {
	if broker == "" {
		return false
	}
	return strings.Contains(broker, "127.0.0.1") || strings.Contains(broker, "localhost")
}

// replaceBrokerHost 替换 broker URL 中的 localhost 主机为指定主机
// Replace localhost host in broker URL with the specified host
func replaceBrokerHost(broker, newHost string) string {
	broker = strings.ReplaceAll(broker, "127.0.0.1", newHost)
	broker = strings.ReplaceAll(broker, "localhost", newHost)
	return broker
}
