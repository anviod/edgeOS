package ean

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ==================== 权限常量 ====================

// Capability 权限级别
const (
	PermissionRead   = "read"   // 只读能力（如 read_points、list_points）
	PermissionWrite  = "write"  // 写入能力（如 write_point）- 需要额外权限检查
	PermissionAdmin  = "admin"  // 管理能力（如 scan_devices、diagnostics）- 需要管理员权限
	PermissionAI     = "ai"     // AI 能力（如 protocol_reverse、doc_parse）- 需要显式授权
)

// ==================== 租户策略 ====================

// TenantPolicy 租户策略，控制租户可访问的 Capability 范围
// 采用 allow/deny 列表模式: 先检查 deny，再检查 allow
type TenantPolicy struct {
	TenantID string   // 租户标识
	AllowCap []string // 允许调用的 capability 前缀列表（空表示允许全部，被 deny 优先级覆盖）
	DenyCap  []string // 禁止调用的 capability 前缀列表
	AllowTarget []string // 允许调用的目标 Agent ID 列表（空表示允许全部）
	DenyTarget  []string // 禁止调用的目标 Agent ID 列表
}

// ==================== Governance ====================

// Governance 治理模块，负责权限检查与审计记录
// 核心职责:
//  - 检查跨节点 Invoke 的权限（尤其 write/admin/AI 类 Capability）
//  - 记录审计日志
//  - 管理租户策略
type Governance struct {
	// 租户策略表
	policies map[string]*TenantPolicy // tenantID -> TenantPolicy
	policyMu  sync.RWMutex

	// 审计记录存储
	auditRecords []AuditRecord
	auditMu      sync.Mutex
	maxAudit     int // 审计记录上限，超出后丢弃最早的

	// 审计回调（可选，用于异步写外部存储）
	onAudit func(record *AuditRecord)

	logger *zap.Logger
}

// GovernanceConfig 治理模块配置
type GovernanceConfig struct {
	// MaxAudit 审计记录最大缓存条数，默认 10000
	MaxAudit int
	// OnAudit 审计记录回调（可选，用于异步落库）
	OnAudit func(record *AuditRecord)
}

// NewGovernance 创建治理模块
func NewGovernance(cfg GovernanceConfig, logger *zap.Logger) *Governance {
	maxAudit := cfg.MaxAudit
	if maxAudit <= 0 {
		maxAudit = 10000
	}

	return &Governance{
		policies:     make(map[string]*TenantPolicy),
		auditRecords: make([]AuditRecord, 0, maxAudit),
		maxAudit:     maxAudit,
		onAudit:      cfg.OnAudit,
		logger:       logger.Named("governance"),
	}
}

// ==================== 权限检查 ====================

// PermissionCheckResult 权限检查结果
type PermissionCheckResult struct {
	Allowed bool   // 是否允许
	Reason  string // 拒绝原因（Allowed=false 时有值）
}

// CheckInvokePermission 检查指定 Invoke 是否被允许
// tenantID: 调用方租户标识
// target: 目标 Agent ID
// capabilityID: 要调用的 Capability ID
// capabilityPermission: Capability 自身声明的权限级别（read/write/admin/ai）
//
// 检查规则:
//  1. read 能力默认允许所有租户
//  2. write/admin 能力需要租户策略中未被 deny
//  3. ai 能力需要租户策略中显式 allow
//  4. 先检查 deny 列表（deny 优先），再检查 allow 列表
func (g *Governance) CheckInvokePermission(tenantID, target, capabilityID, capabilityPermission string) *PermissionCheckResult {
	// read 能力默认放行
	if capabilityPermission == PermissionRead {
		return &PermissionCheckResult{Allowed: true}
	}

	g.policyMu.RLock()
	policy, exists := g.policies[tenantID]
	g.policyMu.RUnlock()

	// 无策略配置时: default 租户放行全部（本地开发/单租户场景）
	// 其他租户: write/admin/ai 拒绝，需显式配置策略
	if !exists {
		if tenantID == "default" || tenantID == "" {
			return &PermissionCheckResult{Allowed: true}
		}
		switch capabilityPermission {
		case PermissionWrite, PermissionAdmin, PermissionAI:
			return &PermissionCheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("tenant %q has no policy, %s capability denied", tenantID, capabilityPermission),
			}
		default:
			return &PermissionCheckResult{Allowed: true}
		}
	}

	// 检查目标 Agent 的 deny 列表
	for _, denied := range policy.DenyTarget {
		if matchPrefix(target, denied) {
			return &PermissionCheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("target agent %q matches deny list %q", target, denied),
			}
		}
	}

	// 检查 Capability 的 deny 列表
	for _, denied := range policy.DenyCap {
		if matchPrefix(capabilityID, denied) {
			return &PermissionCheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("capability %q matches deny list %q", capabilityID, denied),
			}
		}
	}

	// AI 能力需要显式 allow
	if capabilityPermission == PermissionAI {
		if len(policy.AllowCap) == 0 {
			return &PermissionCheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("ai capability %q requires explicit allow in tenant policy", capabilityID),
			}
		}
		allowed := false
		for _, allowedCap := range policy.AllowCap {
			if matchPrefix(capabilityID, allowedCap) {
				allowed = true
				break
			}
		}
		if !allowed {
			return &PermissionCheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("ai capability %q not in tenant allow list", capabilityID),
			}
		}
	}

	// 检查 allow target（非空时必须匹配）
	if len(policy.AllowTarget) > 0 {
		matched := false
		for _, allowed := range policy.AllowTarget {
			if matchPrefix(target, allowed) {
				matched = true
				break
			}
		}
		if !matched {
			return &PermissionCheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("target agent %q not in tenant allow target list", target),
			}
		}
	}

	// 检查 allow capability（非空时必须匹配，对 write/admin 也生效）
	if len(policy.AllowCap) > 0 && capabilityPermission != PermissionAI {
		matched := false
		for _, allowedCap := range policy.AllowCap {
			if matchPrefix(capabilityID, allowedCap) {
				matched = true
				break
			}
		}
		if !matched {
			return &PermissionCheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("capability %q not in tenant allow list", capabilityID),
			}
		}
	}

	return &PermissionCheckResult{Allowed: true}
}

// ==================== 审计记录 ====================

// RecordAudit 记录一条审计日志
// 用于跨节点 Invoke 的审计追踪
func (g *Governance) RecordAudit(initiator, target, capability, invokeID, status, tenantID string) {
	record := AuditRecord{
		ID:         uuid.New().String(),
		Initiator:  initiator,
		Target:     target,
		Capability: capability,
		InvokeID:   invokeID,
		Status:     status,
		TenantID:   tenantID,
		Timestamp:  time.Now().UnixMilli(),
	}

	// 缓存审计记录
	g.auditMu.Lock()
	if len(g.auditRecords) >= g.maxAudit {
		// 丢弃最早的记录
		g.auditRecords = g.auditRecords[1:]
	}
	g.auditRecords = append(g.auditRecords, record)
	g.auditMu.Unlock()

	g.logger.Info("audit record",
		zap.String("audit_id", record.ID),
		zap.String("initiator", initiator),
		zap.String("target", target),
		zap.String("capability", capability),
		zap.String("invoke_id", invokeID),
		zap.String("status", status),
		zap.String("tenant_id", tenantID))

	// 异步回调
	if g.onAudit != nil {
		go g.onAudit(&record)
	}
}

// ==================== 审计查询 ====================

// QueryAuditRecords 查询审计记录
// 按条件过滤，返回匹配的记录（倒序，最新在前）
func (g *Governance) QueryAuditRecords(initiator, target, capability string, limit int) []AuditRecord {
	g.auditMu.Lock()
	defer g.auditMu.Unlock()

	if limit <= 0 || limit > len(g.auditRecords) {
		limit = len(g.auditRecords)
	}

	result := make([]AuditRecord, 0, limit)
	// 倒序遍历
	for i := len(g.auditRecords) - 1; i >= 0 && len(result) < limit; i-- {
		record := g.auditRecords[i]
		if initiator != "" && record.Initiator != initiator {
			continue
		}
		if target != "" && record.Target != target {
			continue
		}
		if capability != "" && record.Capability != capability {
			continue
		}
		result = append(result, record)
	}
	return result
}

// AuditCount 返回当前缓存中的审计记录总数
func (g *Governance) AuditCount() int {
	g.auditMu.Lock()
	defer g.auditMu.Unlock()
	return len(g.auditRecords)
}

// ==================== 策略管理 ====================

// SetPolicy 设置/更新租户策略
func (g *Governance) SetPolicy(policy *TenantPolicy) {
	g.policyMu.Lock()
	defer g.policyMu.Unlock()

	policyCopy := *policy
	g.policies[policy.TenantID] = &policyCopy

	g.logger.Info("tenant policy set",
		zap.String("tenant_id", policy.TenantID),
		zap.Int("allow_cap_count", len(policy.AllowCap)),
		zap.Int("deny_cap_count", len(policy.DenyCap)))
}

// RemovePolicy 移除租户策略
func (g *Governance) RemovePolicy(tenantID string) {
	g.policyMu.Lock()
	defer g.policyMu.Unlock()

	delete(g.policies, tenantID)
	g.logger.Info("tenant policy removed", zap.String("tenant_id", tenantID))
}

// GetPolicy 获取指定租户的策略
func (g *Governance) GetPolicy(tenantID string) (*TenantPolicy, bool) {
	g.policyMu.RLock()
	defer g.policyMu.RUnlock()

	p, ok := g.policies[tenantID]
	if !ok {
		return nil, false
	}
	cp := *p
	return &cp, true
}

// ListPolicies 列出所有租户策略
func (g *Governance) ListPolicies() []*TenantPolicy {
	g.policyMu.RLock()
	defer g.policyMu.RUnlock()

	result := make([]*TenantPolicy, 0, len(g.policies))
	for _, p := range g.policies {
		cp := *p
		result = append(result, &cp)
	}
	return result
}

// ==================== 辅助函数 ====================

// matchPrefix 检查字符串是否匹配指定模式
// 支持：精确匹配、"*" 通配全部、前缀匹配（如 "ai." 匹配 "ai.protocol_reverse"）
func matchPrefix(s, prefix string) bool {
	if prefix == "" {
		return true
	}
	if prefix == "*" {
		return true
	}
	if s == prefix {
		return true
	}
	// 前缀匹配: "ai." 能匹配 "ai.protocol_reverse"
	if len(s) > len(prefix) && s[:len(prefix)] == prefix {
		return true
	}
	return false
}
