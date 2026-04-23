package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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
	"github.com/anviod/edgeOS/internal/messaging"
	"github.com/anviod/edgeOS/internal/server"
	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/ws"
)

func main() {
	// 初始化日志
	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer zapLogger.Sync()

	// 加载配置
	cfg, err := config.LoadConfig("./config/config.yaml")
	if err != nil {
		zapLogger.Fatal("Failed to load config", zap.Error(err))
	}

	// 初始化数据库
	db, err := initDatabase(cfg)
	if err != nil {
		zapLogger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// 创建节点
	node, err := createNode(cfg, db)
	if err != nil {
		zapLogger.Fatal("Failed to create node", zap.Error(err))
	}

	// 启动节点
	ctx, cancel := context.WithCancel(context.Background())
	if err := node.Start(ctx); err != nil {
		zapLogger.Fatal("Failed to start node", zap.Error(err))
	}
	defer cancel()

	// 初始化 WebSocket Hub
	hub := ws.NewHub(zapLogger)

	// 初始化服务层
	registrySvc := services.NewRegistryService(db)
	dataSvc := services.NewDataService(db)
	alertSvc := services.NewAlertService(db)
	middlewareSvc := services.NewMiddlewareService(db, zapLogger)
	controlSvc := services.NewControlService(db, zapLogger)

	// 从配置文件初始化中间件配置
	if err := middlewareSvc.InitFromConfig(cfg.Middlewares); err != nil {
		zapLogger.Warn("Failed to init middlewares from config", zap.Error(err))
	}

	// 初始化消息管理器
	messagingManager := messaging.NewManager(middlewareSvc, registrySvc, dataSvc, alertSvc, controlSvc, hub, zapLogger)
	if err := messagingManager.Start(); err != nil {
		zapLogger.Fatal("Failed to start messaging manager", zap.Error(err))
	}
	defer messagingManager.Stop()
	zapLogger.Info("Messaging manager started")

	// 初始化 HTTP 服务器
	app := initServer(cfg, node, db, hub, registrySvc, dataSvc, alertSvc, middlewareSvc, controlSvc, messagingManager, zapLogger)

	// 启动 HTTP 服务器
	serverAddr := cfg.Node.Listen
	if serverAddr == "" {
		serverAddr = ":8000"
	}

	// 优雅关闭
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Server starting on %s", serverAddr)
		if err := app.Listen(serverAddr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待信号
	<-c
	log.Println("Shutting down server...")

	// 给服务器 5 秒时间完成现有请求
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// 关闭 HTTP 服务器
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// 停止节点
	if err := node.Stop(); err != nil {
		log.Printf("Node stop error: %v", err)
	}

	log.Println("Server stopped gracefully")
}

// initDatabase 初始化数据库
func initDatabase(cfg *config.Config) (*bbolt.DB, error) {
	dbPath := cfg.Database.Path
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %v", err)
	}

	db, err := bbolt.Open(dbPath+"/edgeos.db", 0600, &bbolt.Options{
		Timeout:      30 * time.Second,
		NoGrowSync:   false,
		FreelistType: bbolt.FreelistArrayType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database at %s: %v", dbPath+"/edgeos.db", err)
	}

	if err := initDatabaseBuckets(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database buckets: %v", err)
	}

	log.Printf("Database initialized successfully at %s", dbPath+"/edgeos.db")
	return db, nil
}

// initDatabaseBuckets 初始化数据库桶
func initDatabaseBuckets(db *bbolt.DB) error {
	return db.Update(func(tx *bbolt.Tx) error {
		buckets := []string{
			"devices", "tasks", "state", "stats",
			"edgex_nodes", "edgex_devices", "edgex_points",
			"edgex_data", "edgex_alerts", "middlewares", "edgex_commands",
		}
		for _, b := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(b)); err != nil {
				return err
			}
		}
		return nil
	})
}

// createNode 创建节点
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

// initServer 初始化 HTTP 服务器
func initServer(
	cfg *config.Config,
	node core.Node,
	db *bbolt.DB,
	hub *ws.Hub,
	registrySvc *services.RegistryService,
	dataSvc *services.DataService,
	alertSvc *services.AlertService,
	middlewareSvc *services.MiddlewareService,
	controlSvc *services.ControlService,
	messagingManager *messaging.Manager,
	logger *zap.Logger,
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

	// 中间件
	app.Use(fiberlogger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// 静态文件服务
	app.Static("/", "./ui/dist")

	// 注册所有路由
	discoveryService := discovery.NewDiscoveryService(db, node.GetNodeID(), "edgeos-queen.local", "edgeos-shared-secret")
	if err := discoveryService.Start(); err != nil {
		logger.Error("Failed to start discovery service", zap.Error(err))
	}

	server.RegisterAllRoutes(app, node, db, hub, registrySvc, dataSvc, alertSvc, middlewareSvc, controlSvc, messagingManager, discoveryService, cfg, logger)

	// 健康检查
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":      "ok",
			"node_id":     node.GetNodeID(),
			"node_type":   node.GetNodeType(),
			"node_status": node.GetStatus(),
		})
	})

	// SPA Fallback
	app.All("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./ui/dist/index.html")
	})

	return app
}
