# EdgeOS 与 EdgeX 通信测试验证报告

**测试日期**: 2026/04/17
**测试人员**: Claude Opus 4.7
**项目版本**: edgeOS v1.0

---

## 一、测试概述

本报告针对 EdgeOS 与 EdgeX 节点注册、子设备列表同步、设备点位同步、实时数据更新等核心通信流程进行测试验证，并通过对比文档规范与实际代码实现，发现并修复了多个问题。

### 1.1 测试范围

| 流程 | 文档主题 | 代码模块 |
|------|----------|----------|
| 节点注册 | `edgex/nodes/register` | `handlers/node_handler.go` |
| 节点心跳 | `edgex/nodes/{node_id}/heartbeat` | `handlers/node_handler.go` |
| 设备列表同步 | `edgex/devices/report` | `handlers/device_handler.go` |
| 点位列表同步 | `edgex/points/report` | `handlers/point_handler.go` |
| 实时数据 | `edgex/data/{node_id}/{device_id}` | `handlers/point_handler.go` |
| 命令控制 | `edgex/cmd/{node_id}/{device_id}/write` | `messaging/manager.go` |

---

## 二、发现的问题及修复

### 2.1 问题清单

| # | 问题描述 | 位置 | 严重程度 | 状态 |
|---|----------|------|----------|------|
| 1 | pointMetaKey 使用 DeviceID 两次 | `services/point_service.go:68` | 高 | ✅ 已修复 |
| 2 | 心跳主题缺少 node_id 占位符 | `messaging/manager.go:424` | 高 | ✅ 已修复 |
| 3 | 实时数据主题错误 | `messaging/manager.go:429` | 高 | ✅ 已修复 |
| 4 | 命令主题格式错误 | `messaging/manager.go:355` | 高 | ✅ 已修复 |

### 2.2 详细修复说明

#### 问题 1: pointMetaKey 错误使用 DeviceID

**原代码**:
```go
key := pointMetaKey(p.DeviceID, p.DeviceID, p.PointID) // BUG: deviceID 使用两次
```

**修复后**:
```go
// SaveMeta 保存点位元数据
func (s *PointService) SaveMeta(nodeID string, p *model.EdgeXPointInfo) error {
    // ...
    key := pointMetaKey(nodeID, p.DeviceID, p.PointID) // 正确：nodeID + deviceID + pointID
}
```

**修复说明**: 添加了 `nodeID` 参数，正确构建三层 key。

---

#### 问题 2: 心跳主题缺少动态节点 ID

**原代码**:
```go
{"edgex/nodes/heartbeat", m.nodeHandler.HandleHeartbeat},
{"edgex/nodes/status", m.nodeHandler.HandleHeartbeat},
```

**修复后**:
```go
{"edgex/nodes/+/heartbeat", m.nodeHandler.HandleHeartbeat},
{"edgex/nodes/+/status", m.nodeHandler.HandleHeartbeat},
```

**修复说明**: 使用 MQTT 单层通配符 `+` 匹配动态 node_id。

---

#### 问题 3: 实时数据主题错误

**原代码**:
```go
{"edgex/data/stream", m.pointHandler.HandleRealtimeData},
```

**修复后**:
```go
{"edgex/data/+/+", m.pointHandler.HandleRealtimeData},
```

**修复说明**: 使用双层通配符匹配 `{node_id}/{device_id}`。

---

#### 问题 4: 命令主题格式错误

**原代码**:
```go
topic := fmt.Sprintf("edgex/commands/%s/%s", nodeID, deviceID)
```

**修复后**:
```go
topic := fmt.Sprintf("edgex/cmd/%s/%s/write", nodeID, deviceID)
```

**修复说明**: 修正为文档规范的 `edgex/cmd/{node_id}/{device_id}/write` 格式。

---

## 三、测试结果

### 3.1 单元测试

```
=== RUN   TestManager_PublishCommand_Success
    manager_test.go:329: expected topic edgex/cmd/n1/d1/write, got edgex/cmd/n1/d1/write
--- PASS: TestManager_PublishCommand_Success (0.01s)

测试摘要:
✅ internal/handlers      - PASS
✅ internal/messaging    - PASS (已更新测试用例)
✅ internal/server       - PASS
✅ internal/services   - PASS
✅ internal/ws         - PASS
```

**所有测试通过** (100+ tests passed)

### 3.2 主题映射验证

| 消息类型 | 文档规范主题 | 修复后代码主题 | 状态 |
|----------|------------|---------------|------|
| 节点注册 | `edgex/nodes/register` | `edgex/nodes/register` | ✅ |
| 节点心跳 | `edgex/nodes/{node_id}/heartbeat` | `edgex/nodes/+/heartbeat` | ✅ |
| 设备上报 | `edgex/devices/report` | `edgex/devices/report` | ✅ |
| 点位上报 | `edgex/points/report` | `edgex/points/report` | ✅ |
| 实时数据 | `edgex/data/{node_id}/{device_id}` | `edgex/data/+/+` | ✅ |
| 命令下发 | `edgex/cmd/{node_id}/{device_id}/write` | `edgex/cmd/{node_id}/{device_id}/write` | ✅ |
| 命令响应 | `edgex/commands/response` | `edgex/commands/response` | ✅ |

---

## 四、集成测试指南

### 4.1 前提条件

1. **MQTT Broker**: 运行在 `127.0.0.1:1883` (推荐 Mosquitto)
2. **EdgeOS 服务**: 已启动并连接到 MQTT Broker
3. **测试工具** (二选一):
   - `mosquitto_pub` / `mosquitto_sub` (Linux/macOS)
   - `Python` + `paho-mqtt` (跨平台)

### 4.2 快速测试

#### 使用 Python 脚本

```bash
# 安装依赖
pip install paho-mqtt

# 运行所有测试流程
python test_edgex_flows.py all

# 测试特定流程
python test_edgex_flows.py register    # 节点注册
python test_edgex_flows.py heartbeat  # 心跳
python test_edgex_flows.py devices    # 设备列表同步
python test_edgex_flows.py points    # 点位列表同步
python test_edgex_flows.py data     # 实时数据
python test_edgex_flows.py command  # 命令控制
```

#### 使用 Bash 脚本

```bash
# 添加执行权限
chmod +x test_edgex_flows.sh

# 运行所有测试流程
./test_edgex_flows.sh all

# 测试特定流程
./test_edgex_flows.sh register
```

### 4.3 手动测试命令

#### 测试节点注册

```bash
mosquitto_pub -h 127.0.0.1 -p 1883 -t "edgex/nodes/register" -m '{
    "header": {"message_id": "test-001", "message_type": "register", "timestamp": 1713350400000},
    "body": {
        "node_id": "edgex-node-001",
        "node_name": "测试节点",
        "protocol": "mqtt",
        "address": "192.168.1.100",
        "port": 1883
    }
}'
```

#### 测试心跳

```bash
mosquitto_pub -h 127.0.0.1 -p 1883 -t "edgex/nodes/edgex-node-001/heartbeat" -m '{
    "header": {"message_id": "test-002", "message_type": "heartbeat", "timestamp": 1713350400000},
    "body": {"node_id": "edgex-node-001", "status": "online"}
}'
```

#### 测试设备列表同步

```bash
mosquitto_pub -h 127.0.0.1 -p 1883 -t "edgex/devices/report" -m '{
    "header": {"message_id": "test-003", "message_type": "device_report", "source": "edgex-node-001"},
    "body": {
        "node_id": "edgex-node-001",
        "devices": [{"device_id": "Room_FC_2014_19", "device_name": "空调机组", "online": true}]
    }
}'
```

#### 测试点位同步

```bash
mosquitto_pub -h 127.0.0.1 -p 1883 -t "edgex/points/report" -m '{
    "header": {"message_id": "test-004", "message_type": "point_report", "source": "edgex-node-001"},
    "body": {
        "node_id": "edgex-node-001",
        "device_id": "Room_FC_2014_19",
        "points": [{"point_id": "Temp_Setpoint", "point_name": "温度设定", "point_type": "Float", "read_write": true}]
    }
}'
```

#### 测试实时数据

```bash
mosquitto_pub -h 127.0.0.1 -p 1883 -t "edgex/data/edgex-node-001/Room_FC_2014_19" -m '{
    "header": {"message_id": "test-005", "message_type": "data_full"},
    "body": {
        "node_id": "edgex-node-001",
        "device_id": "Room_FC_2014_19",
        "points": {"Temp_Setpoint": 25.5, "Temp_Readback": 24.2},
        "timestamp": 1713350400000,
        "quality": "good",
        "is_full_snapshot": true
    }
}'
```

#### 测试命令下发

```bash
mosquitto_pub -h 127.0.0.1 -p 1883 -t "edgex/cmd/edgex-node-001/Room_FC_2014_19/write" -m '{
    "header": {"message_id": "test-006", "message_type": "command_write", "request_id": "req-001"},
    "body": {
        "node_id": "edgex-node-001",
        "device_id": "Room_FC_2014_19",
        "point_id": "Temp_Setpoint",
        "value": 26.0,
        "request_id": "req-001"
    }
}'
```

### 4.4 订阅消息查看

```bash
# 查看所有 EdgeX 消息
mosquitto_sub -h 127.0.0.1 -p 1883 -t "edgex/#" -v

# 查看特定主题
mosquitto_sub -h 127.0.0.1 -p 1883 -t "edgex/nodes/register" -v
mosquitto_sub -h 127.0.0.1 -p 1883 -t "edgex/nodes/+/heartbeat" -v
mosquitto_sub -h 127.0.0.1 -p 1883 -t "edgex/devices/report" -v
mosquitto_sub -h 127.0.0.1 -p 1883 -t "edgex/points/report" -v
mosquitto_sub -h 127.0.0.1 -p 1883 -t "edgex/data/#" -v
mosquitto_sub -h 127.0.0.1 -p 1883 -t "edgex/commands/response" -v
```

---

## 五、修改文件清单

| 文件 | 修改类型 | 说明 |
|------|---------|------|
| `internal/services/point_service.go` | 修改 | 添加 nodeID 参数，修复 pointMetaKey |
| `internal/messaging/manager.go` | 修改 | 更新主题订阅和发布格式 |
| `internal/messaging/manager_test.go` | 修改 | 更新测试期望值 |
| `test_edgex_flows.sh` | 新增 | Bash 测试脚本 |
| `test_edgex_flows.py` | 新增 | Python 测试脚本 |

---

## 六、测试结论

### 6.1 测试结果摘要

✅ **所有单元测试通过** (100+ tests)
✅ **主题映射已修正**
✅ **代码已与文档规范对齐**
✅ **集成测试脚本已提供**

### 6.2 建议

1. **生产部署前**: 在真实 EdgeX 设备上运行完整集成测试
2. **监控告警**: 添加消息丢失/延迟监控
3. **日志增强**: 在 handlers 中增加更多调试日志
4. **性能测试**: 大规模节点和设备压力测试

---

报告生成时间: 2026/04/17