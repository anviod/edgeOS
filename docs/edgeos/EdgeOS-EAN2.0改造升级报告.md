# EdgeOS EAN 2.0 改造升级报告

> **日期**: 2026-07-28（v6 设备数对账 / 全量测试）  
> **对照基线**: [EAN2.0-EdgeX-EdgeOS改造指南.md](./EAN2.0-EdgeX-EdgeOS改造指南.md) §3 / §5 / §7.2  
> **共识文档**: `D:\code\edgex\docs\TODO\V1-to-EAN-Migration-Assessment.md`（v2.12）  
> **结论**: **Phase 1/2 代码完成并实机复验通过（MQTT + NATS）**；**全量 `go test ./...` 通过**；**V1 设备全量上报对账剪枝已修复（EdgeX 4 ↔ EdgeOS 4）**；**EAN Agent→节点注册表镜像已修复（nodes 不再为 0）**；Phase 3 部分完成（OS-P3-01/02 已完成）。

---

## 1. 摘要

EdgeOS 已落地完整 EAN 协调层：`internal/ean`（Discovery / Invoke+Reply / Event+previous_value / Heartbeat / Governance / DualTransport）、`/api/ean/*`、UI（Overview / Agents / Invoke / Events / DebugHelp）、V1 过渡期并行（Bridge；写操作走原生 EAN Invoke）。

本轮关键进展（v6）：排查并修复「EdgeX 设备数与 EdgeOS 上报不一致」——根因在 **V1 `edgex/devices/report` 只 Upsert 不剪枝**，历史设备残留（实机曾 **EdgeX=4 / EdgeOS=14**）；同时 **EAN 在线但 `/api/nodes` 为空**（V1 节点注册缺失）。已实现全量对账剪枝、EAN→Registry 镜像、HTTP 对账 API，实机恢复为 **devices=4 / nodes=1 / agents=1 / caps=63**。

| 维度 | 结论 |
|------|------|
| 代码能力（§3 OS-1～22，除 OS-13 status） | **已完成** |
| 设备数一致性（V1 report 对账） | **已修复并实机验证** |
| 节点/Agent 展示一致性 | **已修复**（EAN mirror + EnsureNodeOnline） |
| 全量测试 `go test ./...` | **全绿** |
| UI build | **通过** |
| Phase 3 | **部分**（见共识文档勾选） |

硬约束：未实现 `$edgeos/state/*`；NATS 保留斜杠；不直接驱动南向；AI Invoke 无自动落库。

---

## 2. 必做项完成度

### 2.1 能力对照

| 必做 | 状态 | 说明 |
|------|------|------|
| 双传输 `$edgeos/#`（MQTT + NATS，NATS 保留斜杠） | **代码完成 + 实机验证** | `DualTransport`；NATS 已启用 |
| Discovery 索引（agent / capability / offline） | **代码完成** | 含原生/V1 source、purge、主动 Query |
| Invoke + Reply 关联 | **代码完成** | correlation_id / invoke_id；原生 Cap 禁 V1 Fallback |
| Event + `previous_value` | **代码完成** | 单测覆盖；实机 Event 流有数据 |
| 心跳超时 / 权限 / 审计 | **代码完成** | HeartbeatMonitor + Governance |
| V1 过渡期并行 | **代码完成** | V1 Topic 保留；新控制面走 EAN |
| V1 设备全量对账 | **已修复（v6）** | `ReconcileDevices`：upsert + 删除未再上报设备 |
| EAN→节点注册镜像 | **已修复（v6）** | Agent online/offline 同步 `edgex_nodes` |
| 启动韧性 | **已修复** | MQTT/NATS 不可用 → Warn + 重连 + 延迟订阅 |
| Invoke 监控指标 | **代码完成** | `Health.invoke_metrics` + `transport_details` |

### 2.2 §7.2 验收清单

- [x] 双传输订阅 `$edgeos/discovery/*` 并建立索引（63 原生 Cap，0 V1 Bridge）
- [x] Invoke + Reply（`system.diagnostics` / `modbus_tcp.list_points` completed）
- [x] Event + `previous_value`（代码完成；实机 recent events 有点位变化流）
- [x] Agent 心跳超时标记 offline
- [x] 权限限制 write/admin/AI（v2.8 已修 default 租户）
- [x] 与 V1 并存无 Topic 冲突
- [x] V1 设备上报数量与 EdgeX 实际设备一致（**v6：4=4**）

---

## 3. 本轮改动摘要（v6 / 设备数对账）

1. **根因**  
   - EdgeX `publishDeviceReport` 发的是**全量快照**（当前 4 台：BACnet×3 + Modbus×1）。  
   - EdgeOS 原逻辑只 `UpsertDevice`，**从不删除**已移除设备 → BoltDB 残留（曾达 14）。  
   - **不是** Discovery 索引错误、不是 EAN Event 重复计数、也不是双传输把同一 `device_id` 存两份（key=`nodeID:deviceID`）。  
   - 另：EAN Agent 在线时 V1 `/api/nodes` 可为 0 → Dashboard `total_nodes=0` 与 `total_devices>0` 并存。  
2. **修复**  
   - `DeviceService.ReconcileDevices` + MQTT/Handler 全量上报路径改用对账。  
   - `Bus.AttachRegistryMirror`：EAN Agent ↔ V1 节点表；设备上报亦 `EnsureNodeOnline`。  
   - `POST /api/nodes/:nodeId/devices/reconcile`；`cmd/reconcile-devices` 离线工具。  
3. **实机数字（修复后）**  

| 侧 | 指标 | 数量 |
|----|------|------|
| EdgeX | channels devices + diagnostics | **4** |
| EdgeOS | Dashboard / devices API | **4** |
| EdgeOS | nodes | **1** online |
| EdgeOS EAN | agents / native caps | **1 / 63** |

### 3.1～3.3 前序

v5 NATS 对称、v4 NATS 启用、v3 Governance 权限修复——见历史记录与共识文档 v2.8～v2.11。

---

## 4. 测试结果

```bash
go test ./... -count=1
# 全部 ok（含 Reconcile / RegistryMirror / reconcile API 单测）

cd ui && npm run build
# ok
```

| 包/命令 | 结果 |
|---------|------|
| `go test ./...` | **全绿** |
| `ui npm run build` | ok |

---

## 5. 残留风险 / 待联调

1. **Phase 3 未做**：OS-P3-04/05/06（V1 cmd responses、配置合并、移除 V1 节点 Topic）。  
2. **OS-13** Invoke status 进度订阅仍未做。  
3. **审计**仍为内存环形缓存。  
4. **双传输 Invoke 双边执行**：EdgeOS 双发，EdgeX 可能执行两次；EdgeOS 只收首个 Reply。  
5. **V1 数据面仅 MQTT**：messaging **未订** NATS `edgex.devices.report`；仅 NATS 北向时需 MQTT middleware 或 HTTP reconcile。  
6. **EdgeX 重注册后 device_report 未稳定触发**：`POST /api/edgex/discover` 可见节点注册，未见随后 devices/report（建议 EdgeX 确认 register_response→publishDeviceReport）。  
7. **启动 retained offline**：MQTT 启动瞬间 agent online→offline 闪断，随后 Query/心跳恢复。  
8. **历史 Invoke E001 失败计数**：与设备对账无关，属既有 Invoke/参数问题沉淀。

### 文档路径

- 共识：`D:\code\edgex\docs\TODO\V1-to-EAN-Migration-Assessment.md`（v2.12）  
- 本报告：`docs/edgeos/EdgeOS-EAN2.0改造升级报告.md`
