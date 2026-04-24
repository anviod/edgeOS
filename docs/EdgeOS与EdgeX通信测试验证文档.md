# EdgeOS与EdgeX  MQTT 通信测试验证文档

## 概述

本文档详细描述 EdgeX 边缘网关与 EdgeOS 蜂群网络之间的 MQTT 通信测试流程，包含完整的时序图、消息示例和验证步骤。

### 测试环境要求

| 组件 | 版本 | 说明 |
|------|------|------|
| MQTT Broker | Mosquitto/EMQX | 端口 1883 |
| EdgeOS | v1.0+ | 主控节点 |
| EdgeX | v1.0+ | 边缘网关节点 |
| 监控工具 | MQTTX / mosquitto_sub | 消息抓包 |

### 测试前提条件

1. MQTT Broker 已启动并运行在 `tcp://127.0.0.1:1883`
2. EdgeOS 已启动并配置好 MQTT 消息总线
3. EdgeX 节点已配置好 MQTT 连接信息

---

## 主题订阅/发布说明

### 消息订阅发布角色

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           MQTT/NATS Broker                             │
│                                                                         │
│   EdgeX ──订阅──> edgex/cmd/{node_id}/+                                │
│   EdgeX ──发布──> edgex/nodes/register         (节点注册请求)           │
│   EdgeX ──发布──> edgex/nodes/{node_id}/heartbeat  (心跳)              │
│   EdgeX ──发布──> edgex/devices/report           (设备上报)             │
│   EdgeX ──发布──> edgex/data/{node_id}/{device_id}  (实时数据)        │
│   EdgeX ──发布──> edgex/cmd/{node_id}/response  (命令响应)             │
│                                                                         │
│   edgeOS ──订阅──> edgex/nodes/register        (接收节点注册)           │
│   edgeOS ──订阅──> edgex/nodes/{node_id}/heartbeat  (接收心跳)        │
│   edgeOS ──订阅──> edgex/devices/report        (接收设备上报)          │
│   edgeOS ──订阅──> edgex/data/{node_id}/{device_id}  (接收实时数据)   │
│   edgeOS ──发布──> edgex/nodes/{node_id}/response  (注册响应)         │
│   edgeOS ──发布──> edgex/cmd/{node_id}/{command}  (下发命令)          │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 订阅/发布主题对照表

#### EdgeX 角色

| 操作 | 行为 | Topic | 说明 |
|------|------|-------|------|
| 连接时订阅 | **订阅** | `edgex/cmd/<node_id>/+` | 接收来自 edgeOS 的所有命令 |
| 节点注册 | **发布** | `edgex/nodes/register` | 发送节点注册请求 |
| 心跳 | **发布** | `edgex/nodes/<node_id>/heartbeat` | 定期发送心跳保活 |
| 设备上报 | **发布** | `edgex/devices/report` | 上报设备列表 |
| 实时数据 | **发布** | `edgex/data/<node_id>/<device_id>` | 上报点位数据 |
| 命令响应 | **发布** | `edgex/cmd/<node_id>/response` | 响应命令执行结果 |

#### edgeOS 角色

| 操作 | 行为 | Topic | 说明 |
|------|------|-------|------|
| 启动时订阅 | **订阅** | `edgex/nodes/register` | 接收节点注册请求 |
| 启动时订阅 | **订阅** | `edgex/nodes/<node_id>/heartbeat` | 接收心跳 |
| 启动时订阅 | **订阅** | `edgex/devices/report` | 接收设备上报 |
| 启动时订阅 | **订阅** | `edgex/data/+/+` | 接收所有设备数据 |
| 注册响应 | **发布** | `edgex/nodes/<node_id>/response` | 返回注册结果和 Token |
| 下发命令 | **发布** | `edgex/cmd/<node_id>/<command>` | 下发设备控制命令 |
| WebSocket | **推送** | WebSocket 连接 | 推送状态变更到 Web UI |

### 交互逻辑流程说明

#### 流程一：节点注册 (Node Registration)

```
EdgeX                              edgeOS                            Broker
 │                                   │                                 │
 │  1. 连接到 Broker                 │                                 │
 │──────────────────────────────────>│                                 │
 │                                   │                                 │
 │  2. 订阅 edgex/cmd/{node_id}/+   │                                 │
 │──────────────────────────────────>│                                 │
 │                                   │ 3. 订阅 edgex/nodes/register    │
 │                                   │<────────────────────────────────│
 │                                   │                                 │
 │  4. 发布 node_register           │                                 │
 │────────────────────────────────────────────────────────>│
 │                                   │ 5. 转发注册消息                  │
 │                                   │<────────────────────────────────│
 │                                   │                                 │
 │                                   │ 6. 解析并存储节点信息            │
 │                                   │    (UpsertNode → BoltDB)        │
 │                                   │                                 │
 │  7. 接收 register_response       │ 8. 发布响应                      │
 │<─────────────────────────────────────────────────────────│
 │                                   │                                 │
 │                                   │ 9. WebSocket 推送 node_status    │
 │                                   │────────────────────────────────>│ (Web UI)
```

**交互说明**:
1. EdgeX 首先连接到 MQTT Broker，并订阅命令接收主题
2. EdgeX 发布 `node_register` 到 `edgex/nodes/register`
3. Broker 转发消息给已订阅的 edgeOS
4. edgeOS 解析消息、验证节点信息、存储到 BoltDB
5. edgeOS 发布 `register_response` 到 `edgex/nodes/<node_id>/response`
6. EdgeX 接收响应，获取 `access_token` 和 `expires_at`
7. edgeOS 通过 WebSocket 通知前端更新节点状态

#### 流程二：设备同步 (Device Sync)

```
EdgeX                              edgeOS                            Broker
 │                                   │                                 │
 │  1. 发布 device_report           │                                 │
 │────────────────────────────────────────────────────────>│
 │                                   │ 2. 转发设备报告                  │
 │                                   │<────────────────────────────────│
 │                                   │                                 │
 │                                   │ 3. 解析并存储设备信息            │
 │                                   │    (UpsertDevices → BoltDB)     │
 │                                   │                                 │
 │                                   │ 4. WebSocket 推送 device_synced │
 │                                   │────────────────────────────────>│ (Web UI)
```

**交互说明**:
1. EdgeX 在节点注册成功后，自动或手动发布设备列表到 `edgex/devices/report`
2. edgeOS 接收并解析设备信息，按 `node_id` 关联存储
3. edgeOS 通过 WebSocket 推送设备同步事件，前端刷新设备列表

#### 流程三：实时数据推送 (Real-time Data)

```
EdgeX                              edgeOS                            Broker
 │                                   │                                 │
 │  1. 发布 data_report             │                                 │
 │  (定期/按需推送)                 │                                 │
 │────────────────────────────────────────────────────────>│
 │                                   │ 2. 转发数据消息                  │
 │                                   │<────────────────────────────────│
 │                                   │                                 │
 │                                   │ 3. 更新点位值                    │
 │                                   │    (UpdateValue → BoltDB)       │
 │                                   │                                 │
 │                                   │ 4. WebSocket 推送 data_update   │
 │                                   │────────────────────────────────>│ (Web UI)
```

**交互说明**:
1. EdgeX 持续采集设备数据，按配置周期或数据变化时发布到 `edgex/data/<node_id>/<device_id>`
2. edgeOS 接收数据，更新 BoltDB 中对应点位的 `current_value`
3. edgeOS 通过 WebSocket 推送数据更新事件，前端实时刷新显示

#### 流程四：心跳保活 (Heartbeat)

```
EdgeX                              edgeOS                            Broker
 │                                   │                                 │
 │  1. 发布 heartbeat (每30秒)       │                                 │
 │────────────────────────────────────────────────────────>│
 │                                   │ 2. 转发心跳消息                  │
 │                                   │<────────────────────────────────│
 │                                   │                                 │
 │                                   │ 3. 更新节点最后活跃时间          │
 │                                   │    (UpdateLastSeen)             │
 │                                   │                                 │
 │                                   │ 4. WebSocket 推送 node_status   │
 │                                   │    (status: online)             │
 │                                   │────────────────────────────────>│ (Web UI)
 │
 │  ... (正常心跳中，节点保持在线)    │                                 │
 │
 │                                   │                                 │
 │  5. 心跳超时 (90秒无心跳)         │                                 │
 │                                   │ 6. WebSocket 推送 node_status   │
 │                                   │    (status: offline)            │
 │                                   │────────────────────────────────>│ (Web UI)
```

**交互说明**:
1. EdgeX 每 30 秒发送一次心跳到 `edgex/nodes/<node_id>/heartbeat`
2. edgeOS 收到心跳后更新节点的 `last_seen` 时间戳
3. 若 90 秒内未收到心跳也未曾收到任何数据包，edgeOS 判定节点离线，更新状态并通知前端

#### 流程五：命令下发 (Command Dispatch)

```
edgeOS                             EdgeX                             Broker
 │                                   │                                 │
 │  1. 发布 discover_command        │                                 │
 │────────────────────────────────────────────────────────>│
 │                                   │ 2. 转发命令消息                  │
 │                                   │<────────────────────────────────│
 │                                   │                                 │
 │                                   │ 3. 订阅 edgex/cmd/{node_id}/+   │
 │                                   │    (已订阅，等待消息)            │
 │                                   │                                 │
 │                                   │ 4. 接收并解析命令                │
 │                                   │                                 │
 │                                   │ 5. 执行设备发现/数据写入         │
 │                                   │                                 │
 │  6. 发布 device_report           │ 7. 执行结果上报                  │
 │<─────────────────────────────────────────────────────────────────────│
 │                                   │                                 │
 │  8. 接收设备列表并同步            │                                 │
 │                                   │                                 │
 │  9. WebSocket 推送 device_synced  │                                 │
 │────────────────────────────────>│ (Web UI)                        │
```

**交互说明**:
1. edgeOS 通过 Web UI 或 API 触发设备发现，发布命令到 `edgex/cmd/<node_id>/discover`
2. EdgeX 订阅了命令主题，收到消息后执行相应操作
3. 执行完成后，EdgeX 发布 `device_report` 或响应消息
4. edgeOS 接收执行结果，更新设备列表或状态

---

## 测试一：节点注册流程

### 1.1 时序图

```
┌─────────┐     ┌──────────────┐     ┌────────────────┐     ┌─────────────┐
│  EdgeX  │     │  MQTT Broker │     │    EdgeOS      │     │   Web UI    │
└────┬────┘     └──────┬───────┘     └───────┬────────┘     └──────┬──────┘
     │                 │                      │                     │
     │  1. Connect     │                      │                     │
     │────────────────>│                      │                     │
     │                 │  2. Subscribe         │                     │
     │                 │<─────────────────────│                     │
     │                 │  (edgex/cmd/nodes/register)                 │
     │                 │                      │                     │
     │  3. Publish     │                      │                     │
     │  (node_register)│                      │                     │
     │────────────────>│  4. Forward          │                     │
     │                 │─────────────────────>│                     │
     │                 │                      │  5. UpsertNode()    │
     │                 │                      │  ─────────────────>│ (BoltDB)
     │                 │                      │                     │
     │                 │  6. Publish           │                     │
     │                 │  (register_response)  │                     │
     │                 │<─────────────────────│                     │
     │  7. Receive     │                      │                     │
     │<────────────────│                      │                     │
     │                 │                      │  8. WebSocket       │
     │                 │                      │  (node_status)      │
     │                 │                      │────────────────────>│
     │                 │                      │                     │
     │                 │                      │  9. HTTP GET /nodes │
     │                 │                      │<────────────────────│
     │                 │                      │ 10. Return nodes[]  │
     │                 │                      │────────────────────>│
     │                 │                      │                     │
```

### 1.2 测试步骤

#### 步骤 1：启动 MQTT 监控

```bash
# 方式一：使用 mosquitto_sub 监控所有 edgex 消息
mosquitto_sub -h 127.0.0.1 -p 1883 -t "edgex/#" -v

# 方式二：使用 MQTTX 图形化工具
# 连接地址: mqtt://127.0.0.1:1883
```

#### 步骤 2：确认 EdgeOS 已订阅主题

查看 EdgeOS 日志，确认已订阅以下主题：

```
edgex/nodes/register
edgex/nodes/heartbeat
edgex/nodes/status
```

#### 步骤 3：模拟 EdgeX 节点发送注册消息

发布到 Topic: `edgex/nodes/register`

```json
{
  "header": {
    "message_id": "msg-node-reg-001",
    "timestamp": 1744680000000,
    "source": "edgex-node-001",
    "destination": "edgeos",
    "message_type": "node_register",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "node_name": "EdgeX Gateway Node 001",
    "model": "edge-gateway",
    "version": "1.0.0",
    "api_version": "v1",
    "capabilities": [
      "shadow-sync",
      "heartbeat",
      "device-control",
      "task-execution"
    ],
    "protocol": "edgeOS(MQTT)",
    "endpoint": {
      "host": "192.168.1.100",
      "port": 8082
    },
    "metadata": {
      "os": "linux",
      "arch": "amd64",
      "hostname": "edgex-node-001.local"
    }
  }
}
```

**使用 mosquitto_pub 发布：**

```bash
mosquitto_pub -h 127.0.0.1 -p 1883 \
  -t "edgex/nodes/register" \
  -m '{
    "header": {
      "message_id": "msg-node-reg-001",
      "timestamp": 1744680000000,
      "source": "edgex-node-001",
      "destination": "edgeos",
      "message_type": "node_register",
      "version": "1.0"
    },
    "body": {
      "node_id": "edgex-node-001",
      "node_name": "EdgeX Gateway Node 001",
      "model": "edge-gateway",
      "version": "1.0.0",
      "api_version": "v1",
      "capabilities": ["shadow-sync", "heartbeat", "device-control", "task-execution"],
      "protocol": "edgeOS(MQTT)",
      "endpoint": {"host": "192.168.1.100", "port": 8082},
      "metadata": {"os": "linux", "arch": "amd64", "hostname": "edgex-node-001.local"}
    }
  }'
```

### 1.3 预期响应

#### EdgeOS 收到注册消息后的处理

**日志输出：**
```
[INFO] Node registered node_id=edgex-node-001 node_name="EdgeX Gateway Node 001"
```

#### EdgeOS 发布注册响应

Topic: `edgex/nodes/edgex-node-001/response`

```json
{
  "header": {
    "message_id": "msg-node-reg-resp-001",
    "timestamp": 1744680000100,
    "source": "edgeos",
    "destination": "edgex-node-001",
    "message_type": "register_response",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "status": "success",
    "access_token": "18c07e59-c2d9-40fb-92e9-8e39d27fbe03",
    "expires_at": 1744766400
  }
}
```

#### WebSocket 推送 (node_status)

```json
{
  "type": "node_status",
  "timestamp": 1744680000100,
  "payload": {
    "node_id": "edgex-node-001",
    "node_name": "EdgeX Gateway Node 001",
    "status": "online",
    "last_seen": 1744680000
  }
}
```

### 1.4 验证要点

| 验证项 | 预期结果 | 验证方法 |
|--------|---------|---------|
| MQTT 消息接收 | edgeOS 接收到 `edgex/nodes/register` 消息 | 检查日志 |
| 节点存储 | 节点信息保存到 BoltDB | API: GET /api/nodes |
| 注册响应 | EdgeOS 发布响应到 `edgex/nodes/{node_id}/response` | MQTT 监控 |
| WebSocket 推送 | 前端收到 `node_status` 事件 | 浏览器控制台 |
| UI 显示 | 节点列表显示 `edgex-node-001` | 访问 Web UI |

### 1.5 验证命令

```bash
# 验证节点已存储
curl -s http://localhost:8000/api/nodes | jq

# 预期输出：
{
  "code": "0",
  "data": {
    "nodes": [
      {
        "node_id": "edgex-node-001",
        "node_name": "EdgeX Gateway Node 001",
        "model": "edge-gateway",
        "status": "online",
        ...
      }
    ]
  }
}
```

---

## 测试二：子设备同步流程

### 2.1 时序图

```
┌─────────┐     ┌──────────────┐     ┌────────────────┐     ┌─────────────┐
│  EdgeX  │     │  MQTT Broker │     │    EdgeOS      │     │   Web UI    │
└────┬────┘     └──────┬───────┘     └───────┬────────┘     └──────┬──────┘
     │                 │                      │                     │
     │  (节点注册成功后)                     │                     │
     │  1. Publish     │                      │                     │
     │  (device_report)│                      │                     │
     │────────────────>│  2. Forward          │                     │
     │                 │─────────────────────>│                     │
     │                 │                      │  3. UpsertDevice()  │
     │                 │                      │  ─────────────────>│ (BoltDB)
     │                 │                      │                     │
     │                 │                      │  4. WebSocket       │
     │                 │                      │  (device_synced)    │
     │                 │                      │────────────────────>│
     │                 │                      │                     │
     │                 │                      │  5. HTTP GET         │
     │                 │                      │  /nodes/{id}/devices│
     │                 │                      │<────────────────────│
     │                 │                      │  6. Return devices[]│
     │                 │                      │────────────────────>│
     │                 │                      │                     │
```

### 2.2 测试步骤

#### 前提条件

- 测试一（节点注册）已完成
- 节点 `edgex-node-001` 状态为 `online`

#### 步骤 1：模拟 EdgeX 节点上报设备

发布到 Topic: `edgex/devices/report`

```json
{
  "header": {
    "message_id": "msg-dev-report-001",
    "timestamp": 1744680010000,
    "source": "edgex-node-001",
    "message_type": "device_report",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "devices": [
      {
        "device_id": "Room_FC_2014_19",
        "device_name": "Room FC 2014 HVAC Controller",
        "device_profile": "hvac-controller",
        "service_name": "bacnet-service",
        "labels": ["hvac", "room-control", "temperature"],
        "description": "HVAC Controller for Room 2014",
        "admin_state": "ENABLED",
        "operating_state": "ENABLED",
        "properties": {
          "protocol": "bacnet-ip",
          "address": "192.168.1.50:47808",
          "device_instance": 2014
        }
      },
      {
        "device_id": "Chiller_Plant_A",
        "device_name": "Chiller Plant A Controller",
        "device_profile": "chiller-controller",
        "service_name": "modbus-tcp-service",
        "labels": ["chiller", "hvac", "cooling"],
        "description": "Main Chiller Plant Controller",
        "admin_state": "ENABLED",
        "operating_state": "ENABLED",
        "properties": {
          "protocol": "modbus-tcp",
          "address": "192.168.1.100:502",
          "unit_id": 1
        }
      }
    ]
  }
}
```

**使用 mosquitto_pub 发布：**

```bash
mosquitto_pub -h 127.0.0.1 -p 1883 \
  -t "edgex/devices/report" \
  -m '{
    "header": {
      "message_id": "msg-dev-report-001",
      "timestamp": 1744680010000,
      "source": "edgex-node-001",
      "message_type": "device_report",
      "version": "1.0"
    },
    "body": {
      "node_id": "edgex-node-001",
      "devices": [
        {
          "device_id": "Room_FC_2014_19",
          "device_name": "Room FC 2014 HVAC Controller",
          "device_profile": "hvac-controller",
          "service_name": "bacnet-service",
          "labels": ["hvac", "room-control"],
          "description": "HVAC Controller for Room 2014",
          "admin_state": "ENABLED",
          "operating_state": "ENABLED",
          "properties": {"protocol": "bacnet-ip", "address": "192.168.1.50:47808"}
        },
        {
          "device_id": "Chiller_Plant_A",
          "device_name": "Chiller Plant A Controller",
          "device_profile": "chiller-controller",
          "service_name": "modbus-tcp-service",
          "labels": ["chiller", "hvac"],
          "description": "Main Chiller Plant Controller",
          "admin_state": "ENABLED",
          "operating_state": "ENABLED",
          "properties": {"protocol": "modbus-tcp", "address": "192.168.1.100:502", "unit_id": 1}
        }
      ]
    }
  }'
```

### 2.3 预期响应

#### WebSocket 推送 (device_synced)

```json
{
  "type": "device_synced",
  "timestamp": 1744680010100,
  "payload": {
    "node_id": "edgex-node-001",
    "count": 2
  }
}
```

#### API 返回设备列表

```bash
curl -s http://localhost:8000/api/nodes/edgex-node-001/devices | jq
```

```json
{
  "code": "0",
  "data": {
    "devices": [
      {
        "device_id": "Room_FC_2014_19",
        "device_name": "Room FC 2014 HVAC Controller",
        "device_profile": "hvac-controller",
        "service_name": "bacnet-service",
        "last_sync": 1744680010,
        ...
      },
      {
        "device_id": "Chiller_Plant_A",
        "device_name": "Chiller Plant A Controller",
        ...
      }
    ]
  }
}
```

### 2.4 验证要点

| 验证项 | 预期结果 | 验证方法 |
|--------|---------|---------|
| 设备存储 | 设备信息保存到 BoltDB，绑定到节点 | API 查询 |
| 设备数量 | 节点下有 2 个设备 | GET /api/nodes/{id}/devices |
| WebSocket 推送 | 前端收到 `device_synced` 事件 | 浏览器控制台 |
| UI 显示 | 设备列表显示 2 个设备 | 访问节点设备页 |

---

## 测试三：点位同步流程

### 3.1 时序图

```
┌─────────┐     ┌──────────────┐     ┌────────────────┐     ┌─────────────┐
│  EdgeX  │     │  MQTT Broker │     │    EdgeOS      │     │   Web UI    │
└────┬────┘     └──────┬───────┘     └───────┬────────┘     └──────┬──────┘
     │                 │                      │                     │
     │  1. Publish     │                      │                     │
     │  (point_report) │                      │                     │
     │────────────────>│  2. Forward          │                     │
     │                 │─────────────────────>│                     │
     │                 │                      │  3. UpsertPoint()   │
     │                 │                      │  ─────────────────>│ (BoltDB)
     │                 │                      │                     │
     │                 │                      │  4. HTTP GET points │
     │                 │                      │<────────────────────│
     │                 │                      │  5. Return points[] │
     │                 │                      │────────────────────>│
     │                 │                      │                     │
```

### 3.2 测试步骤

#### 步骤 1：模拟 EdgeX 节点上报点位

发布到 Topic: `edgex/points/report`

```json
{
  "header": {
    "message_id": "msg-point-report-001",
    "timestamp": 1744680020000,
    "source": "edgex-node-001",
    "message_type": "point_report",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "device_id": "Room_FC_2014_19",
    "points": [
      {
        "point_id": "Temperature.Indoor",
        "point_name": "室内温度",
        "resource_name": "Temperature",
        "value_type": "Float32",
        "access_mode": "R",
        "unit": "°C",
        "minimum": -40,
        "maximum": 120,
        "address": "AI-1",
        "data_type": "analog_input",
        "scale": 0.1,
        "offset": 0
      },
      {
        "point_id": "Temperature.Outdoor",
        "point_name": "室外温度",
        "resource_name": "Temperature",
        "value_type": "Float32",
        "access_mode": "R",
        "unit": "°C",
        "minimum": -40,
        "maximum": 120,
        "address": "AI-2",
        "data_type": "analog_input"
      },
      {
        "point_id": "SetPoint.Value",
        "point_name": "设定温度",
        "resource_name": "Setpoint",
        "value_type": "Float32",
        "access_mode": "RW",
        "unit": "°C",
        "minimum": 16,
        "maximum": 30,
        "address": "AV-1",
        "data_type": "analog_value"
      },
      {
        "point_id": "State.Heater",
        "point_name": "加热状态",
        "resource_name": "BinaryOutput",
        "value_type": "Boolean",
        "access_mode": "R",
        "unit": "",
        "address": "BO-1",
        "data_type": "binary_output"
      },
      {
        "point_id": "State.Chiller",
        "point_name": "制冷状态",
        "resource_name": "BinaryOutput",
        "value_type": "Boolean",
        "access_mode": "R",
        "unit": "",
        "address": "BO-2",
        "data_type": "binary_output"
      },
      {
        "point_id": "Temperature.Water",
        "point_name": "水温",
        "resource_name": "Temperature",
        "value_type": "Float32",
        "access_mode": "R",
        "unit": "°C",
        "address": "AI-3",
        "data_type": "analog_input"
      }
    ]
  }
}
```

**使用 mosquitto_pub 发布：**

```bash
mosquitto_pub -h 127.0.0.1 -p 1883 \
  -t "edgex/points/report" \
  -m '{
    "header": {
      "message_id": "msg-point-report-001",
      "timestamp": 1744680020000,
      "source": "edgex-node-001",
      "message_type": "point_report",
      "version": "1.0"
    },
    "body": {
      "node_id": "edgex-node-001",
      "device_id": "Room_FC_2014_19",
      "points": [
        {"point_id": "Temperature.Indoor", "point_name": "室内温度", "resource_name": "Temperature", "value_type": "Float32", "access_mode": "R", "unit": "°C", "address": "AI-1", "data_type": "analog_input"},
        {"point_id": "Temperature.Outdoor", "point_name": "室外温度", "resource_name": "Temperature", "value_type": "Float32", "access_mode": "R", "unit": "°C", "address": "AI-2", "data_type": "analog_input"},
        {"point_id": "SetPoint.Value", "point_name": "设定温度", "resource_name": "Setpoint", "value_type": "Float32", "access_mode": "RW", "unit": "°C", "address": "AV-1", "data_type": "analog_value"},
        {"point_id": "State.Heater", "point_name": "加热状态", "resource_name": "BinaryOutput", "value_type": "Boolean", "access_mode": "R", "address": "BO-1", "data_type": "binary_output"},
        {"point_id": "State.Chiller", "point_name": "制冷状态", "resource_name": "BinaryOutput", "value_type": "Boolean", "access_mode": "R", "address": "BO-2", "data_type": "binary_output"},
        {"point_id": "Temperature.Water", "point_name": "水温", "resource_name": "Temperature", "value_type": "Float32", "access_mode": "R", "unit": "°C", "address": "AI-3", "data_type": "analog_input"}
      ]
    }
  }'
```

### 3.3 预期响应

#### API 返回点位列表

```bash
curl -s http://localhost:8000/api/nodes/edgex-node-001/devices/Room_FC_2014_19/points | jq
```

```json
{
  "code": "0",
  "data": {
    "points": [
      {
        "point_id": "Temperature.Indoor",
        "point_name": "室内温度",
        "value_type": "Float32",
        "access_mode": "R",
        "unit": "°C",
        "last_sync": 1744680020,
        ...
      },
      {
        "point_id": "Temperature.Outdoor",
        ...
      }
    ],
    "snapshot": {
      "Temperature.Indoor": 22.5,
      "Temperature.Outdoor": 12.0,
      "SetPoint.Value": 19,
      "State.Heater": false,
      "State.Chiller": true,
      "Temperature.Water": 35.9
    }
  }
}
```

---

## 测试四：实时数据推送流程

### 4.1 时序图

```
┌─────────┐     ┌──────────────┐     ┌────────────────┐     ┌─────────────┐
│  EdgeX  │     │  MQTT Broker │     │    EdgeOS      │     │   Web UI    │
└────┬────┘     └──────┬───────┘     └───────┬────────┘     └──────┬──────┘
     │                 │                      │                     │
     │  (定时/循环)     │                      │                     │
     │  1. Publish     │                      │                     │
     │  (实时数据)      │                      │                     │
     │────────────────>│  2. Forward          │                     │
     │                 │─────────────────────>│                     │
     │                 │                      │  3. UpdatePointValue│
     │                 │                      │  ─────────────────>│ (BoltDB)
     │                 │                      │                     │
     │                 │                      │  4. WebSocket       │
     │                 │                      │  (data_update)      │
     │                 │                      │────────────────────>│
     │                 │                      │                     │
     │                 │                      │  5. UI 自动更新    │
     │                 │                      │                     │
```

### 4.2 测试步骤

#### 步骤 1：模拟 EdgeX 节点发送实时数据

发布到 Topic: `edgex/data/edgex-node-001/Room_FC_2014_19`

```json
{
  "header": {
    "message_id": "msg-data-001",
    "timestamp": 1744680030000,
    "source": "edgex-node-001",
    "message_type": "data",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "device_id": "Room_FC_2014_19",
    "timestamp": 1744680030000,
    "points": {
      "Temperature.Indoor": 18.9,
      "Temperature.Outdoor": 12.0,
      "SetPoint.Value": 19,
      "Setpoint.1": 19,
      "Setpoint.2": 19,
      "Setpoint.3": 19,
      "State.Heater": false,
      "State.Chiller": true,
      "Temperature.Water": 35.9
    },
    "quality": "Good"
  }
}
```

**使用 mosquitto_pub 发布：**

```bash
mosquitto_pub -h 127.0.0.1 -p 1883 \
  -t "edgex/data/edgex-node-001/Room_FC_2014_19" \
  -m '{
    "header": {
      "message_id": "msg-data-001",
      "timestamp": 1744680030000,
      "source": "edgex-node-001",
      "message_type": "data",
      "version": "1.0"
    },
    "body": {
      "node_id": "edgex-node-001",
      "device_id": "Room_FC_2014_19",
      "timestamp": 1744680030000,
      "points": {
        "Temperature.Indoor": 18.9,
        "Temperature.Outdoor": 12.0,
        "SetPoint.Value": 19,
        "Setpoint.1": 19,
        "Setpoint.2": 19,
        "Setpoint.3": 19,
        "State.Heater": false,
        "State.Chiller": true,
        "Temperature.Water": 35.9
      },
      "quality": "Good"
    }
  }'
```

### 4.3 预期响应

#### WebSocket 推送 (data_update)

```json
{
  "type": "data_update",
  "timestamp": 1744680030100,
  "payload": {
    "node_id": "edgex-node-001",
    "device_id": "Room_FC_2014_19",
    "points": {
      "Temperature.Indoor": 18.9,
      "Temperature.Outdoor": 12.0,
      "SetPoint.Value": 19,
      "State.Heater": false,
      "State.Chiller": true,
      "Temperature.Water": 35.9
    },
    "timestamp": 1744680030000,
    "quality": "Good",
    "is_full_snapshot": false
  }
}
```

---

## 测试四-B：实时数据增量更新验证流程

### 4B.1 验证目标

验证 EdgeOS 系统的实时数据增量更新机制，确保：
1. 点位数据能够正确增量更新（差量 Merge）
2. 数据能够持久化存储到 BoltDB
3. 前端能够实时接收并展示最新数据
4. 前后端数据一致性

### 4B.2 数据流架构

```
┌────────────────────────────────────────────────────────────────────────────────┐
│                           实时数据增量更新数据流                                  │
├────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  EdgeX 节点                EdgeOS Backend              BoltDB                    │
│  ─────────                ──────────────              ──────                    │
│     │                          │                         │                       │
│     │ 1. 发布实时数据           │                         │                       │
│     │ (edgex/data/+/+)        │                         │                       │
│     │────────────────────────>│                         │                       │
│     │                          │                         │                       │
│     │                          │ 2. 解析消息              │                       │
│     │                          │ (HandleRealtimeData)    │                       │
│     │                          │                         │                       │
│     │                          │ 3. 增量更新快照          │                       │
│     │                          │ (SaveSnapshot, full=false)                       │
│     │                          │────────────────────────>│                       │
│     │                          │                         │                       │
│     │                          │ 4. WebSocket 推送       │                       │
│     │                          │ (data_update)          │                       │
│     │                          │────────────────────────>│ (Web UI)              │
│     │                          │                         │                       │
│                                                                                 │
└────────────────────────────────────────────────────────────────────────────────┘
```

### 4B.3 验证步骤

#### 步骤 1：初始数据推送（全量快照）

先推送完整点位数据，确保物模型快照存在：

```bash
# 发布全量点位快照
mosquitto_pub -h 127.0.0.1 -p 1883 \
  -t "edgex/data/edgex-node-001/Room_FC_2014_19" \
  -m '{
    "header": {
      "message_id": "msg-init-001",
      "timestamp": 1744680030000,
      "source": "edgex-node-001",
      "message_type": "data",
      "version": "1.0"
    },
    "body": {
      "node_id": "edgex-node-001",
      "device_id": "Room_FC_2014_19",
      "timestamp": 1744680030000,
      "points": {
        "Temperature.Indoor": 22.5,
        "Temperature.Outdoor": 12.0,
        "SetPoint.Value": 19,
        "State.Heater": false,
        "State.Chiller": true,
        "Temperature.Water": 35.9
      },
      "quality": "Good"
    }
  }'
```

**验证方法：**

```bash
# 1. 检查 WebSocket 推送
# 浏览器控制台应收到:
# {"type":"data_update","timestamp":1744680030,"payload":{...}}

# 2. API 查询快照
curl -s http://localhost:8000/api/nodes/edgex-node-001/devices/Room_FC_2014_19/snapshot | jq

# 预期输出:
# {
#   "code": "0",
#   "data": {
#     "node_id": "edgex-node-001",
#     "device_id": "Room_FC_2014_19",
#     "points": {
#       "Temperature.Indoor": 22.5,
#       "Temperature.Outdoor": 12.0,
#       "SetPoint.Value": 19,
#       "State.Heater": false,
#       "State.Chiller": true,
#       "Temperature.Water": 35.9
#     },
#     "quality": "Good",
#     "timestamp": 1744680030
#   }
# }
```

#### 步骤 2：增量数据推送（差量更新）

仅推送部分点位变化，不发送全部点位：

```bash
# 模拟温度变化 - 只推送变化的点位
mosquitto_pub -h 127.0.0.1 -p 1883 \
  -t "edgex/data/edgex-node-001/Room_FC_2014_19" \
  -m '{
    "header": {
      "message_id": "msg-delta-001",
      "timestamp": 1744680040000,
      "source": "edgex-node-001",
      "message_type": "data",
      "version": "1.0"
    },
    "body": {
      "node_id": "edgex-node-001",
      "device_id": "Room_FC_2014_19",
      "timestamp": 1744680040000,
      "points": {
        "Temperature.Indoor": 23.1,
        "Temperature.Outdoor": 13.5
      },
      "quality": "Good"
    }
  }'
```

**预期行为：**
- 后端执行差量 Merge：将新值合并到现有快照
- 其他未发送的点位（SetPoint.Value, State.Heater 等）保持原值

**验证方法：**

```bash
# 1. API 查询快照（验证增量合并）
curl -s http://localhost:8000/api/nodes/edgex-node-001/devices/Room_FC_2014_19/snapshot | jq

# 预期输出（注意：只更新了变化的点位，其他点位保持不变）:
# {
#   "code": "0",
#   "data": {
#     "node_id": "edgex-node-001",
#     "device_id": "Room_FC_2014_19",
#     "points": {
#       "Temperature.Indoor": 23.1,      # 已更新（从 22.5 变为 23.1）
#       "Temperature.Outdoor": 13.5,    # 已更新（从 12.0 变为 13.5）
#       "SetPoint.Value": 19,           # 未变化（保持原值）
#       "State.Heater": false,           # 未变化（保持原值）
#       "State.Chiller": true,           # 未变化（保持原值）
#       "Temperature.Water": 35.9       # 未变化（保持原值）
#     },
#     "quality": "Good",
#     "timestamp": 1744680040
#   }
# }

# 2. 检查 WebSocket 推送内容
# {"type":"data_update","timestamp":1744680040,"payload":{
#   "node_id":"edgex-node-001",
#   "device_id":"Room_FC_2014_19",
#   "points":{"Temperature.Indoor":23.1,"Temperature.Outdoor":13.5},
#   "timestamp":1744680040,
#   "quality":"Good"
# }}
```

#### 步骤 3：多次增量更新测试

持续发送增量数据，观察累积效果：

```bash
# 循环发送增量更新
for i in {1..10}; do
  mosquitto_pub -h 127.0.0.1 -p 1883 \
    -t "edgex/data/edgex-node-001/Room_FC_2014_19" \
    -m "{
      \"header\": {\"message_id\": \"msg-delta-loop-$i\", \"timestamp\": $(date +%s000), \"source\": \"edgex-node-001\", \"message_type\": \"data\", \"version\": \"1.0\"},
      \"body\": {
        \"node_id\": \"edgex-node-001\",
        \"device_id\": \"Room_FC_2014_19\",
        \"timestamp\": $(date +%s000),
        \"points\": {
          \"Temperature.Indoor\": 2$i.\$((RANDOM % 10))
        },
        \"quality\": \"Good\"
      }
    }"
  sleep 2
done
```

**验证方法：**

```bash
# 最终快照应该是所有增量更新的累积结果
curl -s http://localhost:8000/api/nodes/edgex-node-001/devices/Room_FC_2014_19/snapshot | jq

# 验证：Temperature.Indoor 应该是最后一次更新的值
# 其他点位应该保持不变
```

### 4B.4 前后端数据一致性验证

#### 前端数据验证

在浏览器控制台或 Vue DevTools 中验证：

```javascript
// 1. 检查 Pinia store 数据
store = window.__pinia
rtStore = store._s.get('realtime')

// 2. 获取设备点位数据
pointsByDevice = rtStore.pointsByDevice
deviceKey = 'edgex-node-001/Room_FC_2014_19'
points = pointsByDevice[deviceKey]

// 3. 验证数据值
console.log('Temperature.Indoor:', points['Temperature.Indoor']?.current_value)
console.log('SetPoint.Value:', points['SetPoint.Value']?.current_value)
console.log('Quality:', points['Temperature.Indoor']?.quality)

// 4. 验证差量高亮状态
deltaKeys = rtStore.deltaKeys
isDelta = deltaKeys.has('edgex-node-001/Room_FC_2014_19/Temperature.Indoor')
console.log('Is Delta Highlighting:', isDelta)
```

#### 数据一致性检查脚本

```bash
#!/bin/bash
# verify_data_consistency.sh

NODE_ID="edgex-node-001"
DEVICE_ID="Room_FC_2014_19"
API_BASE="http://localhost:8000/api"

echo "=== 前后端数据一致性验证 ==="
echo ""

# 1. 获取后端快照
echo "[1/4] 获取后端快照..."
SNAPSHOT=$(curl -s "$API_BASE/nodes/$NODE_ID/devices/$DEVICE_ID/snapshot")
BACKEND_TEMP=$(echo $SNAPSHOT | jq -r '.data.points["Temperature.Indoor"]')
echo "  后端 Temperature.Indoor: $BACKEND_TEMP"

# 2. 获取点位列表（包含当前值）
echo ""
echo "[2/4] 获取点位列表..."
POINTS=$(curl -s "$API_BASE/nodes/$NODE_ID/devices/$DEVICE_ID/points")
echo $POINTS | jq -r '.data.points[] | "\(.point_id): \(.current_value)"'

# 3. 检查 WebSocket 事件日志（需要 EdgeOS 日志）
echo ""
echo "[3/4] 检查最近 WebSocket 推送..."
# 此步骤需要在 EdgeOS 日志中搜索
# grep "data_update" /var/log/edgeos/edgeos.log | tail -5

# 4. 对比验证
echo ""
echo "[4/4] 数据一致性检查..."
if [ "$BACKEND_TEMP" != "null" ] && [ "$BACKEND_TEMP" != "" ]; then
  echo "  ✓ 后端数据存在"
else
  echo "  ✗ 后端数据异常"
fi

echo ""
echo "=== 验证完成 ==="
```

### 4B.5 验证要点汇总

| 验证项 | 验证方法 | 预期结果 |
|--------|---------|---------|
| 全量快照保存 | API 查询 `/api/nodes/{id}/devices/{id}/snapshot` | 返回所有点位数据 |
| 增量数据合并 | 发送部分点位后再次查询快照 | 未发送的点位保持原值 |
| WebSocket 推送 | 浏览器控制台监听 | 收到 `data_update` 事件 |
| 前端状态更新 | Vue DevTools 查看 store | `pointsByDevice` 数据更新 |
| 差量高亮 | 观察 UI 变化 | 1500ms 内有高亮效果 |
| 数据持久化 | 重启 EdgeOS 后查询 | 快照数据依然存在 |

### 4B.6 完整测试脚本

```bash
#!/bin/bash
# test_realtime_data_incremental.sh

MQTT_HOST="127.0.0.1"
MQTT_PORT="1883"
NODE_ID="edgex-node-001"
DEVICE_ID="Room_FC_2014_19"
API_BASE="http://localhost:8000/api"

echo "=== 实时数据增量更新测试 ==="
echo ""

# 步骤 1: 发送全量快照
echo "[步骤 1] 发送全量快照..."
mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT \
  -t "edgex/data/$NODE_ID/$DEVICE_ID" \
  -m '{
    "header": {"message_id": "init-full", "timestamp": '$(date +%s000)', "source": "'$NODE_ID'", "message_type": "data", "version": "1.0"},
    "body": {
      "node_id": "'$NODE_ID'",
      "device_id": "'$DEVICE_ID'",
      "timestamp": '$(date +%s000)',
      "points": {
        "Temperature.Indoor": 22.5,
        "Temperature.Outdoor": 12.0,
        "SetPoint.Value": 19,
        "State.Heater": false,
        "State.Chiller": true,
        "Temperature.Water": 35.9
      },
      "quality": "Good"
    }
  }'
echo "  -> 全量快照已发送"
sleep 2

# 验证全量快照
echo ""
echo "[验证] 全量快照..."
curl -s "$API_BASE/nodes/$NODE_ID/devices/$DEVICE_ID/snapshot" | jq '.data.points'

# 步骤 2: 发送增量更新（只更新温度）
echo ""
echo "[步骤 2] 发送增量更新..."
mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT \
  -t "edgex/data/$NODE_ID/$DEVICE_ID" \
  -m '{
    "header": {"message_id": "delta-001", "timestamp": '$(date +%s000)', "source": "'$NODE_ID'", "message_type": "data", "version": "1.0"},
    "body": {
      "node_id": "'$NODE_ID'",
      "device_id": "'$DEVICE_ID'",
      "timestamp": '$(date +%s000)',
      "points": {
        "Temperature.Indoor": 23.8
      },
      "quality": "Good"
    }
  }'
echo "  -> 增量更新已发送（只更新 Temperature.Indoor）"
sleep 2

# 验证增量合并
echo ""
echo "[验证] 增量合并结果..."
echo "预期：Temperature.Indoor=23.8，其他点位保持原值"
curl -s "$API_BASE/nodes/$NODE_ID/devices/$DEVICE_ID/snapshot" | jq '.data.points'

# 步骤 3: 再次增量更新
echo ""
echo "[步骤 3] 再次发送增量更新..."
mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT \
  -t "edgex/data/$NODE_ID/$DEVICE_ID" \
  -m '{
    "header": {"message_id": "delta-002", "timestamp": '$(date +%s000)', "source": "'$NODE_ID'", "message_type": "data", "version": "1.0"},
    "body": {
      "node_id": "'$NODE_ID'",
      "device_id": "'$DEVICE_ID'",
      "timestamp": '$(date +%s000)',
      "points": {
        "Temperature.Outdoor": 15.2,
        "State.Chiller": false
      },
      "quality": "Good"
    }
  }'
echo "  -> 增量更新已发送（更新 2 个点位）"
sleep 2

# 最终验证
echo ""
echo "[最终验证] 快照数据..."
curl -s "$API_BASE/nodes/$NODE_ID/devices/$DEVICE_ID/snapshot" | jq '.data.points'

echo ""
echo "=== 测试完成 ==="
echo "请检查："
echo "  1. 浏览器控制台 WebSocket 消息"
echo "  2. 前端 UI 数据更新"
echo "  3. Vue DevTools store 状态"
```

---

## 测试五：心跳维持流程

### 5.1 时序图

```
┌─────────┐     ┌──────────────┐     ┌────────────────┐     ┌─────────────┐
│  EdgeX  │     │  MQTT Broker │     │    EdgeOS      │     │   Web UI    │
└────┬────┘     └──────┬───────┘     └───────┬────────┘     └──────┬──────┘
     │                 │                      │                     │
     │  (每30秒)        │                      │                     │
     │  1. Publish     │                      │                     │
     │  (heartbeat)    │                      │                     │
     │────────────────>│  2. Forward          │                     │
     │                 │─────────────────────>│                     │
     │                 │                      │  3. UpdateNodeStatus│
     │                 │                      │  ─────────────────>│ (BoltDB)
     │                 │                      │                     │
     │                 │                      │  4. WebSocket       │
     │                 │                      │  (node_status)     │
     │                 │                      │────────────────────>│
     │                 │                      │                     │
     │  (超时无心跳)    │                      │                     │
     │                 │                      │  5. Mark offline    │
     │                 │                      │  ─────────────────>│ (BoltDB)
     │                 │                      │                     │
     │                 │                      │  6. WebSocket       │
     │                 │                      │  (node_status)     │
     │                 │                      │────────────────────>│
     │                 │                      │                     │
```

### 5.2 测试步骤

#### 步骤 1：模拟 EdgeX 节点发送心跳

发布到 Topic: `edgex/nodes/edgex-node-001/heartbeat`

```json
{
  "header": {
    "message_id": "msg-hb-001",
    "timestamp": 1744680100000,
    "source": "edgex-node-001",
    "message_type": "heartbeat",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "status": "active",
    "uptime_seconds": 3600,
    "sequence": 100,
    "metrics": {
      "cpu_usage": 25.5,
      "memory_usage": 512,
      "disk_usage": 45.2,
      "active_devices": 2,
      "active_tasks": 5
    }
  }
}
```

**使用 mosquitto_pub 发布：**

```bash
mosquitto_pub -h 127.0.0.1 -p 1883 \
  -t "edgex/nodes/edgex-node-001/heartbeat" \
  -m '{
    "header": {
      "message_id": "msg-hb-001",
      "timestamp": 1744680100000,
      "source": "edgex-node-001",
      "message_type": "heartbeat",
      "version": "1.0"
    },
    "body": {
      "node_id": "edgex-node-001",
      "status": "active",
      "uptime_seconds": 3600,
      "sequence": 100,
      "metrics": {
        "cpu_usage": 25.5,
        "memory_usage": 512,
        "disk_usage": 45.2,
        "active_devices": 2,
        "active_tasks": 5
      }
    }
  }'
```

### 5.3 预期响应

#### WebSocket 推送 (node_status)

```json
{
  "type": "node_status",
  "timestamp": 1744680100100,
  "payload": {
    "node_id": "edgex-node-001",
    "status": "online",
    "last_seen": 1744680100
  }
}
```

---

## 测试六：告警推送流程

### 6.1 时序图

```
┌─────────┐     ┌──────────────┐     ┌────────────────┐     ┌─────────────┐
│  EdgeX  │     │  MQTT Broker │     │    EdgeOS      │     │   Web UI    │
└────┬────┘     └──────┬───────┘     └───────┬────────┘     └──────┬──────┘
     │                 │                      │                     │
     │  (异常事件)      │                      │                     │
     │  1. Publish     │                      │                     │
     │  (alert)        │                      │                     │
     │────────────────>│  2. Forward          │                     │
     │                 │─────────────────────>│                     │
     │                 │                      │  3. SaveAlert()     │
     │                 │                      │  ─────────────────>│ (BoltDB)
     │                 │                      │                     │
     │                 │                      │  4. WebSocket       │
     │                 │                      │  (alert)            │
     │                 │                      │────────────────────>│
     │                 │                      │                     │
     │                 │                      │  5. 告警通知弹窗    │
     │                 │                      │                     │
```

### 6.2 测试步骤

#### 步骤 1：模拟 EdgeX 节点发送告警

发布到 Topic: `edgex/events/alert`

```json
{
  "header": {
    "message_id": "msg-alert-001",
    "timestamp": 1744680200000,
    "source": "edgex-node-001",
    "message_type": "alert",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "device_id": "Room_FC_2014_19",
    "alert_id": "alert-001",
    "alert_type": "temperature_high",
    "severity": "warning",
    "message": "室内温度超过设定值，当前温度: 28.5°C, 设定温度: 24°C",
    "timestamp": 1744680200000,
    "details": {
      "current_value": 28.5,
      "threshold_value": 24,
      "point_id": "Temperature.Indoor",
      "duration_seconds": 300
    }
  }
}
```

**使用 mosquitto_pub 发布：**

```bash
mosquitto_pub -h 127.0.0.1 -p 1883 \
  -t "edgex/events/alert" \
  -m '{
    "header": {
      "message_id": "msg-alert-001",
      "timestamp": 1744680200000,
      "source": "edgex-node-001",
      "message_type": "alert",
      "version": "1.0"
    },
    "body": {
      "node_id": "edgex-node-001",
      "device_id": "Room_FC_2014_19",
      "alert_id": "alert-001",
      "alert_type": "temperature_high",
      "severity": "warning",
      "message": "室内温度超过设定值，当前温度: 28.5°C, 设定温度: 24°C",
      "timestamp": 1744680200000,
      "details": {
        "current_value": 28.5,
        "threshold_value": 24,
        "point_id": "Temperature.Indoor",
        "duration_seconds": 300
      }
    }
  }'
```

### 6.3 预期响应

#### WebSocket 推送 (alert)

```json
{
  "type": "alert",
  "timestamp": 1744680200100,
  "payload": {
    "id": "alert-001",
    "node_id": "edgex-node-001",
    "device_id": "Room_FC_2014_19",
    "alert_type": "temperature_high",
    "severity": "warning",
    "message": "室内温度超过设定值，当前温度: 28.5°C, 设定温度: 24°C",
    "timestamp": 1744680200,
    "details": {
      "current_value": 28.5,
      "threshold_value": 24,
      "point_id": "Temperature.Indoor",
      "duration_seconds": 300
    },
    "read": false
  }
}
```

### 6.4 不同告警类型测试

#### 设备离线告警

```json
{
  "header": {
    "message_id": "msg-alert-002",
    "timestamp": 1744680300000,
    "source": "edgex-node-001",
    "message_type": "alert",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "device_id": "Chiller_Plant_A",
    "alert_id": "alert-002",
    "alert_type": "device_offline",
    "severity": "critical",
    "message": "设备 Chiller_Plant_A 连接超时",
    "timestamp": 1744680300000,
    "details": {
      "last_seen": "2026-04-17T10:00:00Z",
      "retry_count": 3,
      "error": "Connection timeout"
    }
  }
}
```

#### 通信错误告警

```json
{
  "header": {
    "message_id": "msg-alert-003",
    "timestamp": 1744680400000,
    "source": "edgex-node-001",
    "message_type": "alert",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "alert_id": "alert-003",
    "alert_type": "communication_error",
    "severity": "error",
    "message": "BACnet 网络通信异常",
    "timestamp": 1744680400000,
    "details": {
      "protocol": "bacnet-ip",
      "error_code": "timeout",
      "retry_count": 5
    }
  }
}
```

---

## 测试七：控制命令下发流程

### 7.1 时序图

```
┌─────────┐     ┌──────────────┐     ┌────────────────┐     ┌─────────────┐
│  EdgeX  │     │  MQTT Broker │     │    EdgeOS      │     │   Web UI    │
└────┬────┘     └──────┬───────┘     └───────┬────────┘     └──────┬──────┘
     │                 │                      │                     │
     │                 │  1. Subscribe         │                     │
     │                 │<─────────────────────│                     │
     │                 │  (edgex/cmd/+/+/write)                   │
     │                 │                      │                     │
     │  (Web UI 点击)   │                      │                     │
     │                 │                      │  2. HTTP POST        │
     │                 │                      │  /commands          │
     │                 │                      │<────────────────────│
     │                 │                      │                     │
     │  3. Receive     │                      │                     │
     │  (write_cmd)    │                      │                     │
     │<────────────────│                      │                     │
     │                 │                      │                     │
     │  4. 执行写入操作  │                      │                     │
     │                 │                      │                     │
     │  5. Publish     │                      │                     │
     │  (cmd_response) │                      │                     │
     │────────────────>│  6. Forward          │                     │
     │                 │─────────────────────>│                     │
     │                 │                      │  7. WebSocket       │
     │                 │                      │  (command_response) │
     │                 │                      │────────────────────>│
     │                 │                      │                     │
```

### 7.2 测试步骤

#### 步骤 1：Web UI 下发控制命令

通过 Web UI 或 API 下发写入命令：

```bash
curl -X POST http://localhost:8000/api/nodes/edgex-node-001/devices/Room_FC_2014_19/commands \
  -H "Content-Type: application/json" \
  -d '{
    "points": {
      "SetPoint.Value": 22
    }
  }'
```

### 7.3 预期消息流

#### EdgeOS 发布写入命令

Topic: `edgex/cmd/edgex-node-001/Room_FC_2014_19/write`

```json
{
  "header": {
    "message_id": "msg-cmd-write-001",
    "timestamp": 1744680500000,
    "source": "edgeos",
    "destination": "edgex-node-001",
    "message_type": "write_command",
    "version": "1.0",
    "correlation_id": "req-write-001"
  },
  "body": {
    "request_id": "req-write-001",
    "node_id": "edgex-node-001",
    "device_id": "Room_FC_2014_19",
    "timestamp": 1744680500000,
    "points": {
      "SetPoint.Value": 22
    },
    "options": {
      "confirm": true,
      "timeout_seconds": 10
    }
  }
}
```

#### EdgeX 返回响应

Topic: `edgex/commands/response`

```json
{
  "header": {
    "message_id": "msg-cmd-resp-001",
    "timestamp": 1744680501000,
    "source": "edgex-node-001",
    "destination": "edgeos",
    "message_type": "command_response",
    "version": "1.0",
    "correlation_id": "req-write-001"
  },
  "body": {
    "request_id": "req-write-001",
    "node_id": "edgex-node-001",
    "device_id": "Room_FC_2014_19",
    "success": true,
    "timestamp": 1744680501000,
    "results": {
      "SetPoint.Value": {
        "success": true,
        "timestamp": 1744680501000
      }
    }
  }
}
```

#### WebSocket 推送 (command_response)

```json
{
  "type": "command_response",
  "timestamp": 1744680501100,
  "payload": {
    "request_id": "req-write-001",
    "node_id": "edgex-node-001",
    "device_id": "Room_FC_2014_19",
    "success": true,
    "error": null
  }
}
```

---

## 测试八：主动发现流程

### 8.1 时序图

```
┌─────────┐     ┌──────────────┐     ┌────────────────┐     ┌─────────────┐
│  EdgeX  │     │  MQTT Broker │     │    EdgeOS      │     │   Web UI    │
└────┬────┘     └──────┬───────┘     └───────┬────────┘     └──────┬──────┘
     │                 │                      │                     │
     │  (Web UI 点击)   │                      │                     │
     │  发现所有节点     │                      │                     │
     │                 │                      │  1. Publish          │
     │                 │                      │  (discover_request) │
     │                 │                      │────────────────────>│
     │                 │  2. Forward          │                     │
     │                 │<─────────────────────│                     │
     │                 │                      │                     │
     │  3. Receive     │                      │                     │
     │  (discover_cmd) │                      │                     │
     │<────────────────│                      │                     │
     │                 │                      │                     │
     │  4. 执行设备发现  │                      │                     │
     │  5. Publish     │                      │                     │
     │  (device_report)│                      │                     │
     │────────────────>│  6. Forward          │                     │
     │                 │─────────────────────>│                     │
     │                 │                      │  (回到测试二)        │
```

### 8.2 测试步骤

#### 步骤 1：触发节点发现

```bash
# 触发所有节点发现
curl -X POST http://localhost:8000/api/edgex/discover

# 或通过指定中间件触发
curl -X POST http://localhost:8000/api/edgex/discover/mqtt-001
```

### 8.3 预期消息流

#### EdgeOS 发布发现请求

Topic: `edgex/cmd/nodes/register`

```json
{
  "header": {
    "message_type": "discovery_request",
    "source": "edgeos",
    "timestamp": 1744680600000
  },
  "body": {}
}
```

---

## 综合测试脚本

### 完整流程测试脚本

```bash
#!/bin/bash
# test_edgex_communication.sh

MQTT_HOST="127.0.0.1"
MQTT_PORT="1883"
NODE_ID="edgex-node-001"
DEVICE_ID="Room_FC_2014_19"

echo "=== EdgeX 与 EdgeOS 通信测试 ==="
echo ""

# 测试 1: 节点注册
echo "[测试 1/6] 节点注册..."
mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT \
  -t "edgex/nodes/register" \
  -m '{
    "header": {"message_id": "test-001", "timestamp": '$(date +%s000)', "source": "'$NODE_ID'", "message_type": "node_register", "version": "1.0"},
    "body": {"node_id": "'$NODE_ID'", "node_name": "Test EdgeX Node", "model": "edge-gateway", "version": "1.0.0", "api_version": "v1", "capabilities": ["shadow-sync", "heartbeat"], "protocol": "edgeOS(MQTT)", "endpoint": {"host": "192.168.1.100", "port": 8082}}
  }'
echo "  -> 节点注册请求已发送"
sleep 2

# 测试 2: 设备上报
echo "[测试 2/6] 设备上报..."
mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT \
  -t "edgex/devices/report" \
  -m '{
    "header": {"message_id": "test-002", "timestamp": '$(date +%s000)', "source": "'$NODE_ID'", "message_type": "device_report", "version": "1.0"},
    "body": {"node_id": "'$NODE_ID'", "devices": [{"device_id": "'$DEVICE_ID'", "device_name": "Test Device", "device_profile": "test-profile", "service_name": "test-service", "admin_state": "ENABLED", "operating_state": "ENABLED"}]}
  }'
echo "  -> 设备上报请求已发送"
sleep 2

# 测试 3: 点位上报
echo "[测试 3/6] 点位上报..."
mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT \
  -t "edgex/points/report" \
  -m '{
    "header": {"message_id": "test-003", "timestamp": '$(date +%s000)', "source": "'$NODE_ID'", "message_type": "point_report", "version": "1.0"},
    "body": {"node_id": "'$NODE_ID'", "device_id": "'$DEVICE_ID'", "points": [{"point_id": "Temperature", "point_name": "温度", "value_type": "Float32", "access_mode": "R", "unit": "°C", "address": "AI-1"}]}
  }'
echo "  -> 点位上报请求已发送"
sleep 2

# 测试 4: 实时数据
echo "[测试 4/6] 实时数据..."
mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT \
  -t "edgex/data/$NODE_ID/$DEVICE_ID" \
  -m '{
    "header": {"message_id": "test-004", "timestamp": '$(date +%s000)', "source": "'$NODE_ID'", "message_type": "data", "version": "1.0"},
    "body": {"node_id": "'$NODE_ID'", "device_id": "'$DEVICE_ID'", "timestamp": '$(date +%s000)', "points": {"Temperature": 25.5}, "quality": "Good"}
  }'
echo "  -> 实时数据已发送"
sleep 2

# 测试 5: 心跳
echo "[测试 5/6] 心跳..."
mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT \
  -t "edgex/nodes/$NODE_ID/heartbeat" \
  -m '{
    "header": {"message_id": "test-005", "timestamp": '$(date +%s000)', "source": "'$NODE_ID'", "message_type": "heartbeat", "version": "1.0"},
    "body": {"node_id": "'$NODE_ID'", "status": "active", "uptime_seconds": 100, "sequence": 1}
  }'
echo "  -> 心跳已发送"
sleep 2

# 测试 6: 告警
echo "[测试 6/6] 告警..."
mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT \
  -t "edgex/events/alert" \
  -m '{
    "header": {"message_id": "test-006", "timestamp": '$(date +%s000)', "source": "'$NODE_ID'", "message_type": "alert", "version": "1.0"},
    "body": {"node_id": "'$NODE_ID'", "device_id": "'$DEVICE_ID'", "alert_id": "test-alert-001", "alert_type": "test_alert", "severity": "info", "message": "测试告警消息", "timestamp": '$(date +%s000)'}
  }'
echo "  -> 告警已发送"
sleep 2

echo ""
echo "=== 测试完成 ==="
echo "请检查:"
echo "  1. EdgeOS 日志确认消息处理"
echo "  2. Web UI 确认数据展示"
echo "  3. MQTT 监控工具确认消息流转"
```

---

## 故障排查

### 常见问题

| 问题 | 可能原因 | 解决方案 |
|------|---------|---------|
| 节点注册后不显示 | API 响应格式问题 | 检查 /api/nodes 返回值 |
| 设备列表为空 | 设备上报未发送 | 确认 MQTT Topic 正确 |
| 实时数据不更新 | WebSocket 未连接 | 检查浏览器控制台 |
| 告警未收到 | 告警 Topic 未订阅 | 检查 edgex/events/alert |

### 调试命令

```bash
# 1. 检查 MQTT 连接
mosquitto_sub -h 127.0.0.1 -p 1883 -t "edgex/#" -v

# 2. 检查 EdgeOS 日志
tail -f /var/log/edgeos/edgeos.log | grep -E "(HandleRegister|HandleDeviceReport|HandlePointReport)"

# 3. 检查 API 响应
curl -s http://localhost:8000/api/nodes | jq

# 4. 检查 WebSocket 连接 (浏览器控制台)
console.log('WebSocket connected:', ws.readyState === WebSocket.OPEN)
```

---

**文档版本**: v1.0  
**创建日期**: 2026-04-17  
**维护者**: edgeOS 团队
