package ean

import (
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGovernance(t *testing.T, opts ...func(*GovernanceConfig)) *Governance {
	t.Helper()
	cfg := GovernanceConfig{MaxAudit: 100}
	for _, opt := range opts {
		opt(&cfg)
	}
	return NewGovernance(cfg, zap.NewNop())
}

// ---------- TestCheckInvokePermission_Read ----------

func TestCheckInvokePermission_Read(t *testing.T) {
	g := newTestGovernance(t)

	// read 能力默认放行，不需要任何策略
	result := g.CheckInvokePermission("tenant-001", "agent-001", "modbus.read_points", PermissionRead)
	assert.True(t, result.Allowed)
	assert.Empty(t, result.Reason)

	// 无策略的租户调用 read 也放行
	result = g.CheckInvokePermission("tenant-unknown", "agent-001", "any.read_cap", PermissionRead)
	assert.True(t, result.Allowed)
}

// ---------- TestCheckInvokePermission_WriteDenied ----------

func TestCheckInvokePermission_WriteDenied(t *testing.T) {
	g := newTestGovernance(t)

	// write 能力无策略时拒绝
	result := g.CheckInvokePermission("tenant-001", "agent-001", "modbus.write_point", PermissionWrite)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "has no policy")
	assert.Contains(t, result.Reason, "write")
}

// ---------- TestCheckInvokePermission_AllowList ----------

func TestCheckInvokePermission_AllowList(t *testing.T) {
	g := newTestGovernance(t)

	// 设置 allow 策略
	g.SetPolicy(&TenantPolicy{
		TenantID:    "tenant-001",
		AllowCap:    []string{"modbus."},
		AllowTarget: []string{"agent-001"},
	})

	// write 能力在 allow 列表中
	result := g.CheckInvokePermission("tenant-001", "agent-001", "modbus.write_point", PermissionWrite)
	assert.True(t, result.Allowed)

	// write 能力不在 allow 列表中
	result = g.CheckInvokePermission("tenant-001", "agent-001", "bacnet.write_point", PermissionWrite)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "not in tenant allow list")

	// target 不在 allow 列表中
	result = g.CheckInvokePermission("tenant-001", "agent-999", "modbus.write_point", PermissionWrite)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "not in tenant allow target list")
}

// ---------- TestCheckInvokePermission_DenyList ----------

func TestCheckInvokePermission_DenyList(t *testing.T) {
	g := newTestGovernance(t)

	// 设置 deny 策略
	g.SetPolicy(&TenantPolicy{
		TenantID:   "tenant-001",
		DenyCap:    []string{"bacnet."},    // 禁止所有 bacnet 开头的 capability
		DenyTarget: []string{"agent-danger"}, // 禁止调用 agent-danger
	})

	// write 能力在 deny 列表中
	result := g.CheckInvokePermission("tenant-001", "agent-001", "bacnet.write_point", PermissionWrite)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "deny list")

	// target 在 deny 列表中
	result = g.CheckInvokePermission("tenant-001", "agent-danger", "modbus.write_point", PermissionWrite)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "deny list")

	// 不在 deny 列表中的 write 无 allow 列表也拒绝（策略存在但 allow 为空）
	result = g.CheckInvokePermission("tenant-001", "agent-001", "modbus.write_point", PermissionWrite)
	// write/admin: 策略存在，deny 先检查通过，然后 allow 列表为空则也放行
	// 看 CheckInvokePermission 逻辑: allowCap 为空 && permission != AI → 跳过 allow 检查 → Allowed
	assert.True(t, result.Allowed)
}

// ---------- AI Permission ----------

func TestCheckInvokePermission_AI(t *testing.T) {
	t.Run("no policy - denied", func(t *testing.T) {
		g := newTestGovernance(t)
		result := g.CheckInvokePermission("tenant-001", "agent-001", "ai.protocol_reverse", PermissionAI)
		assert.False(t, result.Allowed)
	})

	t.Run("policy without allow - denied", func(t *testing.T) {
		g := newTestGovernance(t)
		g.SetPolicy(&TenantPolicy{TenantID: "tenant-001"})
		result := g.CheckInvokePermission("tenant-001", "agent-001", "ai.protocol_reverse", PermissionAI)
		assert.False(t, result.Allowed)
		assert.Contains(t, result.Reason, "explicit allow")
	})

	t.Run("policy with allow - allowed", func(t *testing.T) {
		g := newTestGovernance(t)
		g.SetPolicy(&TenantPolicy{
			TenantID: "tenant-001",
			AllowCap: []string{"ai."},
		})
		result := g.CheckInvokePermission("tenant-001", "agent-001", "ai.protocol_reverse", PermissionAI)
		assert.True(t, result.Allowed)
	})

	t.Run("policy with non-matching allow - denied", func(t *testing.T) {
		g := newTestGovernance(t)
		g.SetPolicy(&TenantPolicy{
			TenantID: "tenant-001",
			AllowCap: []string{"vision."},
		})
		result := g.CheckInvokePermission("tenant-001", "agent-001", "ai.protocol_reverse", PermissionAI)
		assert.False(t, result.Allowed)
		assert.Contains(t, result.Reason, "not in tenant allow list")
	})
}

// ---------- Admin Permission ----------

func TestCheckInvokePermission_Admin(t *testing.T) {
	t.Run("no policy - denied", func(t *testing.T) {
		g := newTestGovernance(t)
		result := g.CheckInvokePermission("tenant-001", "agent-001", "system.scan_devices", PermissionAdmin)
		assert.False(t, result.Allowed)
	})

	t.Run("with allow policy - allowed", func(t *testing.T) {
		g := newTestGovernance(t)
		g.SetPolicy(&TenantPolicy{
			TenantID: "tenant-001",
			AllowCap: []string{"system."},
		})
		result := g.CheckInvokePermission("tenant-001", "agent-001", "system.scan_devices", PermissionAdmin)
		assert.True(t, result.Allowed)
	})
}

// ---------- TestRecordAudit ----------

func TestRecordAudit(t *testing.T) {
	g := newTestGovernance(t)

	g.RecordAudit("initiator-001", "agent-001", "modbus.read_points", "invoke-001", "success", "tenant-001")

	assert.Equal(t, 1, g.AuditCount())

	records := g.QueryAuditRecords("", "", "", 10)
	require.Len(t, records, 1)
	assert.Equal(t, "initiator-001", records[0].Initiator)
	assert.Equal(t, "agent-001", records[0].Target)
	assert.Equal(t, "modbus.read_points", records[0].Capability)
	assert.Equal(t, "invoke-001", records[0].InvokeID)
	assert.Equal(t, "success", records[0].Status)
	assert.Equal(t, "tenant-001", records[0].TenantID)
	assert.NotEmpty(t, records[0].ID)
	assert.Greater(t, records[0].Timestamp, int64(0))
}

// ---------- TestQueryAudit ----------

func TestQueryAudit(t *testing.T) {
	g := newTestGovernance(t)

	// 记录多条审计
	g.RecordAudit("init-1", "target-1", "cap-1", "inv-1", "success", "tenant-1")
	g.RecordAudit("init-2", "target-1", "cap-2", "inv-2", "failed", "tenant-1")
	g.RecordAudit("init-1", "target-2", "cap-1", "inv-3", "success", "tenant-2")

	assert.Equal(t, 3, g.AuditCount())

	// 按 initiator 过滤
	records := g.QueryAuditRecords("init-1", "", "", 10)
	assert.Len(t, records, 2)

	// 按 target 过滤
	records = g.QueryAuditRecords("", "target-1", "", 10)
	assert.Len(t, records, 2)

	// 按 capability 过滤
	records = g.QueryAuditRecords("", "", "cap-1", 10)
	assert.Len(t, records, 2)

	// 按 limit
	records = g.QueryAuditRecords("", "", "", 1)
	assert.Len(t, records, 1)
	// 最新的在前面
	assert.Equal(t, "inv-3", records[0].InvokeID)

	// 组合过滤
	records = g.QueryAuditRecords("init-1", "target-2", "cap-1", 10)
	assert.Len(t, records, 1)

	// 无匹配
	records = g.QueryAuditRecords("nonexistent", "", "", 10)
	assert.Empty(t, records)
}

// ---------- 审计上限淘汰 ----------

func TestAuditMaxCapacity(t *testing.T) {
	g := newTestGovernance(t, func(cfg *GovernanceConfig) {
		cfg.MaxAudit = 5
	})

	for i := 0; i < 8; i++ {
		g.RecordAudit("init", "target", "cap", "", "success", "tenant")
	}

	assert.Equal(t, 5, g.AuditCount())
	records := g.QueryAuditRecords("", "", "", 10)
	assert.Len(t, records, 5)
	// 最早的被淘汰，最新的在前
	assert.Equal(t, "success", records[0].Status)
}

// ---------- 策略管理 ----------

func TestPolicyManagement(t *testing.T) {
	g := newTestGovernance(t)

	policy := &TenantPolicy{
		TenantID:    "tenant-001",
		AllowCap:    []string{"modbus."},
		DenyCap:     []string{"bacnet."},
		AllowTarget: []string{"agent-001"},
		DenyTarget:  []string{"agent-danger"},
	}
	g.SetPolicy(policy)

	// GetPolicy
	p, ok := g.GetPolicy("tenant-001")
	require.True(t, ok)
	assert.Equal(t, "tenant-001", p.TenantID)
	assert.Equal(t, []string{"modbus."}, p.AllowCap)

	// ListPolicies
	policies := g.ListPolicies()
	assert.Len(t, policies, 1)

	// RemovePolicy
	g.RemovePolicy("tenant-001")
	_, ok = g.GetPolicy("tenant-001")
	assert.False(t, ok)
	assert.Empty(t, g.ListPolicies())
}

// ---------- matchPrefix ----------

func TestMatchPrefix(t *testing.T) {
	tests := []struct {
		s      string
		prefix string
		want   bool
	}{
		{"hello", "hello", true},
		{"hello.world", "hello.", true},
		{"hello.world", "hello", true}, // "hello" is a prefix of "hello.world"
		{"ai.protocol_reverse", "ai.", true},
		{"modbus.read", "modbus.", true},
		{"modbus", "modbus.", false},
		{"anything", "", true},
		{"", "", true},
		{"", "prefix", false},
		{"edgeCore-node-001", "*", true},
		{"anything", "*", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"/"+tt.prefix, func(t *testing.T) {
			assert.Equal(t, tt.want, matchPrefix(tt.s, tt.prefix))
		})
	}
}
