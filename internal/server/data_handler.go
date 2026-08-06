package server

import (
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"

	"github.com/anviod/edgeOS/internal/storage"
)

// ===========================
// 数据管理 | Database management
// ===========================

func dataStatsPayload(store *storage.Storage) (fiber.Map, error) {
	stats, totalSize, err := store.GetBucketStats()
	if err != nil {
		return nil, err
	}
	return fiber.Map{
		"config_db": fiber.Map{
			"path": store.GetConfigPath(),
		},
		"runtime_db": fiber.Map{
			"path": store.GetRuntimePath(),
		},
		"total_size": totalSize,
		"buckets":    stats,
	}, nil
}

// handleDataStats 数据库概览：config.db + runtime.db 双库统计与全部 bucket 明细（只读）
// GET /api/data/stats
func handleDataStats(store *storage.Storage) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if store == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "数据库未初始化 | Storage not available")
		}
		payload, err := dataStatsPayload(store)
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError, fmt.Sprintf("获取数据库统计失败: %v", err))
		}
		return apiSuccess(c, payload)
	}
}

// handleBackupConfig 备份配置库到本地目录（配置库优先备份，不包含运行时数据）
// POST /api/data/backup-config?dir=data/backups
func handleBackupConfig(store *storage.Storage) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if store == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "数据库未初始化 | Storage not available")
		}
		backupDir := c.Query("dir", "data/backups")
		info, err := store.BackupConfigDB(backupDir)
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError, fmt.Sprintf("备份配置库失败: %v", err))
		}
		return apiSuccess(c, fiber.Map{
			"status":       "success",
			"message":      "配置库已备份，运行时数据不受影响",
			"backup_path":  info.BackupPath,
			"backup_time":  info.BackupTime,
			"original":     info.OriginalPath,
			"size_bytes":   info.FileSizeBytes,
			"size_display": formatBytes(info.FileSizeBytes),
		})
	}
}

// handleClearRuntimeBuckets 清空指定运行时 bucket（配置库 bucket 受保护，返回 403）
// POST /api/data/clear-cache  body: { "buckets": ["edgeCore_alerts"] } | { "mode": "all" }
func handleClearRuntimeBuckets(store *storage.Storage) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if store == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "数据库未初始化 | Storage not available")
		}
		var req struct {
			Mode    string   `json:"mode"`
			Buckets []string `json:"buckets"`
		}
		if err := c.BodyParser(&req); err != nil {
			return apiError(c, fiber.StatusBadRequest, "无效的请求体 | Invalid request body")
		}

		var targets []string
		switch req.Mode {
		case "all":
			stats, _, err := store.GetBucketStats()
			if err != nil {
				return apiError(c, fiber.StatusInternalServerError, fmt.Sprintf("读取数据库统计失败: %v", err))
			}
			for _, st := range stats {
				if st.Database == "runtime" && st.Clearable {
					targets = append(targets, st.Name)
				}
			}
		case "":
			targets = req.Buckets
		default:
			return apiError(c, fiber.StatusBadRequest, "无效的 mode: "+req.Mode)
		}

		if len(targets) == 0 {
			return apiError(c, fiber.StatusBadRequest, "mode 或 buckets 至少提供一个 | mode or buckets is required")
		}

		var cleared []string
		for _, name := range targets {
			if err := store.ClearBucket(name); err != nil {
				return apiError(c, fiber.StatusForbidden, fmt.Sprintf("清理失败: %v", err))
			}
			cleared = append(cleared, name)
		}

		payload, err := dataStatsPayload(store)
		if err != nil {
			payload = fiber.Map{"message": "已清理，刷新统计失败"}
		}
		return apiSuccess(c, fiber.Map{
			"status":  "success",
			"cleared": cleared,
			"stats":   payload,
		})
	}
}

// handleClearAllRuntime 清空运行时库全部 bucket（配置库不受影响），并自动压缩
// POST /api/data/clear-all-runtime
func handleClearAllRuntime(store *storage.Storage) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if store == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "数据库未初始化 | Storage not available")
		}
		cleared, err := store.ClearAllRuntimeBuckets()
		if err != nil {
			return apiError(c, fiber.StatusInternalServerError, fmt.Sprintf("清空运行时库失败: %v", err))
		}

		compact := fiber.Map{}
		if cerr := store.CompactRuntimeDB(); cerr != nil {
			compact["error"] = cerr.Error()
		} else {
			compact["status"] = "ok"
		}

		payload, _ := dataStatsPayload(store)
		return apiSuccess(c, fiber.Map{
			"status":  "success",
			"cleared": cleared,
			"compact": compact,
			"stats":   payload,
		})
	}
}

// handleCompactRuntime 压缩运行时库，回收已删除数据的磁盘空间
// POST /api/data/compact-runtime
func handleCompactRuntime(store *storage.Storage) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if store == nil {
			return apiError(c, fiber.StatusServiceUnavailable, "数据库未初始化 | Storage not available")
		}
		runtimePath := store.GetRuntimePath()
		beforeSize := fileSize(runtimePath)

		if err := store.CompactRuntimeDB(); err != nil {
			return apiError(c, fiber.StatusInternalServerError, fmt.Sprintf("压缩运行时库失败: %v", err))
		}

		afterSize := fileSize(runtimePath)
		saved := beforeSize - afterSize
		if saved < 0 {
			saved = 0
		}

		return apiSuccess(c, fiber.Map{
			"status":        "success",
			"message":       "运行时库压缩完成",
			"before_bytes":  beforeSize,
			"after_bytes":   afterSize,
			"saved_bytes":   saved,
			"before_size":   formatBytes(beforeSize),
			"after_size":    formatBytes(afterSize),
			"saved_size":    formatBytes(saved),
			"runtime_db":    fiber.Map{"path": runtimePath},
		})
	}
}

// formatBytes 人类可读文件大小 | Human-readable file size
func formatBytes(bytes int64) string {
	if bytes <= 0 {
		return "0 MB"
	}
	const unit = 1024 * 1024
	mb := float64(bytes) / unit
	if mb < 0.01 {
		return "0.01 MB"
	}
	return fmt.Sprintf("%.2f MB", mb)
}

// fileSize 返回文件大小，失败返回 0 | Return file size, 0 on error
func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
