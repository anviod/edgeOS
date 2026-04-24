---
layout: default
title: EdgeOS 后端实现指南
description: EdgeOS 后端服务职责、接口协作与实现路径说明。
---

# EdgeOS 后端实现指南

> 本文档聚焦于 **EdgeOS 通过 UI 添加消息总线（MQTT/NATS），再通过中间件监听指定主题**，按以下四个核心功能顺序完整说明后端实现：
> 1. EdgeX 节点注册
> 2. EdgeX 子设备列表同步
> 3. EdgeX 子设备点位同步(即物模型点位上报 可用将实时数据点位理解成物模型)
> 4. EdgeX 子设备双向控制

---

## 目录

1. [架构总览](#1-架构总览)
2. [消息总线管理](#2-消息总线管理)
3. [EdgeX 节点注册](#3-edgex-节点注册)
4. [EdgeX 子设备列表同步](#4-edgex-子设备列表同步)
5. [EdgeX 子设备点位同步(即物模型点位上报 可用将实时数据点位理解成物模型)](#5-edgex-子设备点位同步(即物模型点位上报 可用将实时数据点位理解成物模型))
6. [EdgeX 子设备双向控制](#6-edgex-子设备双向控制)
7. [心跳与状态管理](#7-心跳与状态管理)
8. [告警与事件处理](#8-告警与事件处理)
9. [错误处理与重试策略](#9-错误处理与重试策略)
10. [安全与认证](#10-安全与认证)
11. [测试与排查](#11-测试与排查)
12. [项目结构参考](#12-项目结构参考)

---

## 1. 架构总览

```
┌────────────────────────────────────────────────────────────────┐
│                        EdgeOS 后端                             │
│                                                                │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │                   消息总线层                           │  │
│  │   ┌──────────────────┐  ┌──────────────────┐           │  │
│  │   │  MQTT Client     │  │  NATS Client     │           │  │
│  │   │  (paho.mqtt)     │  │  (nats.go)       │           │  │
│  │   └────────┬─────────┘  └────────┬─────────┘           │  │
│  └────────────┼───────────────────┬─┘                      │  │
│               │                   │                        │  │
│  ┌────────────▼───────────────────▼────────────────────┐   │  │
│  │              消息路由器 (MessageRouter)              │   │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────┐  │   │  │
│  │  │注册Handler│ │设备Handler│ │点位Handler│ │控制Hdl│  │   │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └───────┘  │   │  │
│  └──────────────────────────┬──────────────────────────┘   │  │
│                             │                              │  │
│  ┌──────────────────────────▼──────────────────────────┐   │  │
│  │                    服务层 (Services)                 │   │  │
│  │  NodeService / DeviceService / PointService / ...   │   │  │
│  └──────────────────────────┬──────────────────────────┘   │  │
│                             │                              │  │
│  ┌──────────────────────────▼──────────────────────────┐   │  │
│  │                    存储层 (Storage)                  │   │  │
│  └──────────────────────────────────────────────────────┘   │  │
└────────────────────────────────────────────────────────────────┘
         ▲                                    ▲
         │  消息中间件 (MQTT/NATS)             │
         ▼                                    ▼
┌──────────────────────────────────────────────────────────────┐
│                       EdgeX 边缘采集网关                      │
│  发布：节点注册 / 设备上报 / 点位上报 / 实时数据 / 心跳       │
│  订阅：设备发现命令 / 写入命令 / 任务控制 / 配置更新          │
└──────────────────────────────────────────────────────────────┘
```

**核心数据流向：**

```
EdgeX → 发布 → 消息中间件 → EdgeOS 订阅 → 路由 → 处理器 → 服务层 → 存储/推送
EdgeOS → 发布控制命令 → 消息中间件 → EdgeX 订阅 → 执行 → 响应
```

---

## 2. 消息总线管理

### 2.1 功能描述

EdgeOS 通过 **UI 界面** 动态添加 MQTT/NATS 消息总线配置，后端提供 REST API 保存连接信息并创建客户端实例，连接成功后立即订阅四个功能所需的全部 Topic/Subject。

### 2.2 连接配置数据结构

```go
// internal/models/middleware.go

type MiddlewareType string

const (
    MiddlewareMQTT MiddlewareType = "edgeOS(MQTT)"
    MiddlewareNATS MiddlewareType = "edgeOS(NATS)"
)

// MiddlewareConfig 前端提交的连接配置
type MiddlewareConfig struct {
    ID          string         `json:"id"`           // 唯一ID（后端生成）
    Name        string         `json:"name"`         // 连接名称，如 "生产环境MQTT"
    Type        MiddlewareType `json:"type"`         // edgeOS(MQTT) 或 edgeOS(NATS)
    Description string         `json:"description"`  // 备注
    MQTT        *MQTTConfig    `json:"mqtt,omitempty"`
    NATS        *NATSConfig    `json:"nats,omitempty"`
    Status      ConnStatus     `json:"status"`       // connected / disconnected / error
    CreatedAt   int64          `json:"created_at"`
    UpdatedAt   int64          `json:"updated_at"`
}

type MQTTConfig struct {
    Broker             string `json:"broker"`               // tcp://127.0.0.1:1883
    ClientID           string `json:"client_id"`
    Username           string `json:"username"`
    Password           string `json:"password"`
    QoS                byte   `json:"qos"`                  // 0/1/2
    CleanSession       bool   `json:"clean_session"`
    KeepAlive          int    `json:"keep_alive"`           // 秒
    ConnectTimeout     int    `json:"connect_timeout"`      // 秒
    AutoReconnect      bool   `json:"auto_reconnect"`
    MaxReconnectInterval int  `json:"max_reconnect_interval"` // 秒
    TLSEnabled         bool   `json:"tls_enabled"`
    CACert             string `json:"ca_cert,omitempty"`
    ClientCert         string `json:"client_cert,omitempty"`
    ClientKey          string `json:"client_key,omitempty"`
}

type NATSConfig struct {
    URL                string `json:"url"`                  // nats://127.0.0.1:4222
    ClientName         string `json:"client_name"`
    Username           string `json:"username"`
    Password           string `json:"password"`
    Token              string `json:"token,omitempty"`
    ConnectTimeout     int    `json:"connect_timeout"`
    ReconnectWait      int    `json:"reconnect_wait"`
    MaxReconnects      int    `json:"max_reconnects"`
    JetStreamEnabled   bool   `json:"jetstream_enabled"`
    TLSEnabled         bool   `json:"tls_enabled"`
}

type ConnStatus string

const (
    ConnStatusConnected    ConnStatus = "connected"
    ConnStatusDisconnected ConnStatus = "disconnected"
    ConnStatusError        ConnStatus = "error"
    ConnStatusConnecting   ConnStatus = "connecting"
)
```

### 2.3 REST API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/middlewares` | 添加消息总线 |
| `GET` | `/api/v1/middlewares` | 获取所有连接列表 (包含 连接/断开 状态)|
| `GET` | `/api/v1/middlewares/:id` | 获取单个连接详情 |
| `PUT` | `/api/v1/middlewares/:id` | 更新连接配置 |
| `DELETE` | `/api/v1/middlewares/:id` | 删除连接 |
| `POST` | `/api/v1/middlewares/:id/connect` | 手动触发连接 |
| `POST` | `/api/v1/middlewares/:id/disconnect` | 断开连接 |
| `GET` | `/api/v1/middlewares/:id/status` | 查询连接状态 |
| `POST` | `/api/v1/middlewares/:id/test` | 测试连接（不持久化） |

### 2.4 连接管理器实现

```go
// internal/middleware/manager.go

package middleware

import (
    "fmt"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/sirupsen/logrus"
)

// ConnectionManager 管理所有消息总线
type ConnectionManager struct {
    mu          sync.RWMutex
    connections map[string]*Connection  // id -> Connection
    storage     MiddlewareStorage
    router      MessageRouter
    log         *logrus.Logger
}

// Connection 单个消息总线实例
type Connection struct {
    Config  *MiddlewareConfig
    Client  MiddlewareClient  // 接口，MQTT或NATS实现
    subs    []Subscription    // 已订阅列表
}

// MiddlewareClient 统一客户端接口
type MiddlewareClient interface {
    Connect() error
    Disconnect()
    Subscribe(topic string, handler MessageHandler) error
    Publish(topic string, msgType string, body interface{}) error
    IsConnected() bool
    Status() ConnStatus
}

// MessageHandler 消息处理函数
type MessageHandler func(topic string, msg *Message) error

func NewConnectionManager(storage MiddlewareStorage, router MessageRouter, log *logrus.Logger) *ConnectionManager {
    return &ConnectionManager{
        connections: make(map[string]*Connection),
        storage:     storage,
        router:      router,
        log:         log,
    }
}

// AddConnection 添加并连接中间件
func (m *ConnectionManager) AddConnection(cfg *MiddlewareConfig) (*MiddlewareConfig, error) {
    cfg.ID = uuid.NewString()
    cfg.Status = ConnStatusConnecting
    cfg.CreatedAt = time.Now().UnixMilli()
    cfg.UpdatedAt = cfg.CreatedAt

    var client MiddlewareClient
    var err error

    switch cfg.Type {
    case MiddlewareMQTT:
        client, err = NewMQTTClient(cfg.MQTT, m.log)
    case MiddlewareNATS:
        client, err = NewNATSClient(cfg.NATS, m.log)
    default:
        return nil, fmt.Errorf("unsupported middleware type: %s", cfg.Type)
    }
    if err != nil {
        cfg.Status = ConnStatusError
        return cfg, err
    }

    if err = client.Connect(); err != nil {
        cfg.Status = ConnStatusError
        _ = m.storage.Save(cfg)
        return cfg, fmt.Errorf("connect failed: %w", err)
    }

    cfg.Status = ConnStatusConnected
    conn := &Connection{Config: cfg, Client: client}

    // 连接成功后订阅全部功能主题
    if err = m.subscribeAll(conn); err != nil {
        m.log.Warnf("partial subscribe error: %v", err)
    }

    m.mu.Lock()
    m.connections[cfg.ID] = conn
    m.mu.Unlock()

    _ = m.storage.Save(cfg)
    return cfg, nil
}

// subscribeAll 订阅四个核心功能所需的所有主题
func (m *ConnectionManager) subscribeAll(conn *Connection) error {
    topics := buildTopicList(conn.Config.Type)
    for _, t := range topics {
        if err := conn.Client.Subscribe(t, m.router.Route); err != nil {
            return fmt.Errorf("subscribe %s failed: %w", t, err)
        }
        conn.subs = append(conn.subs, Subscription{Topic: t})
        m.log.Infof("[%s] subscribed: %s", conn.Config.Name, t)
    }
    return nil
}

// buildTopicList 根据协议类型返回需要订阅的主题列表
func buildTopicList(t MiddlewareType) []string {
    if t == MiddlewareMQTT {
        return []string{
            // 1. 节点注册（Stage 1：被动）
            "edgex/nodes/register",
            "edgex/nodes/unregister",
            // 1b. 节点发现（Stage 2：主动触发 EdgeX 重新注册）
            "edgex/cmd/nodes/register",
            // 2. 设备同步
            "edgex/devices/report",
            // 3. 点位同步(即物模型点位上报 可用将实时数据点位理解成物模型)
            "edgex/points/report",
            // 4. 实时数据（双向控制读侧）
            "edgex/data/#",
            // 响应
            "edgex/responses/#",
            // 心跳与状态
            "edgex/nodes/+/heartbeat",
            "edgex/nodes/+/status",
            "edgex/nodes/+/online",
            "edgex/nodes/+/offline",
            // 告警
            "edgex/events/alert",
            "edgex/events/error",
            "edgex/events/info",
        }
    }
    // NATS
    return []string{
        "edgex.nodes.register",
        "edgex.nodes.unregister",
        "edgex.cmd.nodes.register",
        "edgex.devices.report",
        "edgex.points.report",
        "edgex.data.>",
        "edgex.res.>",
        "edgex.nodes.heartbeat.>",
        "edgex.nodes.status.>",
        "edgex.events.alert",
        "edgex.events.error",
        "edgex.events.info",
    }
}
```

### 2.5 MQTT 客户端实现

```go
// internal/middleware/mqtt_client.go

package middleware

import (
    "encoding/json"
    "fmt"
    "time"

    mqtt "github.com/eclipse/paho.mqtt.golang"
    "github.com/sirupsen/logrus"
)

type mqttClient struct {
    cfg    *MQTTConfig
    client mqtt.Client
    log    *logrus.Logger
}

func NewMQTTClient(cfg *MQTTConfig, log *logrus.Logger) (MiddlewareClient, error) {
    opts := mqtt.NewClientOptions()
    opts.AddBroker(cfg.Broker)
    opts.SetClientID(cfg.ClientID)
    opts.SetUsername(cfg.Username)
    opts.SetPassword(cfg.Password)
    opts.SetCleanSession(cfg.CleanSession)
    opts.SetKeepAlive(time.Duration(cfg.KeepAlive) * time.Second)
    opts.SetConnectTimeout(time.Duration(cfg.ConnectTimeout) * time.Second)
    opts.SetAutoReconnect(cfg.AutoReconnect)

    opts.SetOnConnectHandler(func(c mqtt.Client) {
        log.Info("[MQTT] connected")
    })
    opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
        log.Warnf("[MQTT] connection lost: %v", err)
    })

    return &mqttClient{cfg: cfg, client: mqtt.NewClient(opts), log: log}, nil
}

func (c *mqttClient) Connect() error {
    if token := c.client.Connect(); token.Wait() && token.Error() != nil {
        return token.Error()
    }
    return nil
}

func (c *mqttClient) Disconnect() { c.client.Disconnect(250) }

func (c *mqttClient) IsConnected() bool { return c.client.IsConnected() }

func (c *mqttClient) Status() ConnStatus {
    if c.client.IsConnected() {
        return ConnStatusConnected
    }
    return ConnStatusDisconnected
}

func (c *mqttClient) Subscribe(topic string, handler MessageHandler) error {
    token := c.client.Subscribe(topic, c.cfg.QoS, func(_ mqtt.Client, msg mqtt.Message) {
        var m Message
        if err := json.Unmarshal(msg.Payload(), &m); err != nil {
            c.log.Errorf("[MQTT] unmarshal error on %s: %v", msg.Topic(), err)
            return
        }
        if err := handler(msg.Topic(), &m); err != nil {
            c.log.Errorf("[MQTT] handler error on %s: %v", msg.Topic(), err)
        }
    })
    token.Wait()
    return token.Error()
}

func (c *mqttClient) Publish(topic string, msgType string, body interface{}) error {
    msg := buildMessage(c.cfg.ClientID, msgType, body)
    payload, err := json.Marshal(msg)
    if err != nil {
        return err
    }
    token := c.client.Publish(topic, c.cfg.QoS, false, payload)
    token.Wait()
    return token.Error()
}
```

### 2.6 NATS 客户端实现

```go
// internal/middleware/nats_client.go

package middleware

import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/nats-io/nats.go"
    "github.com/sirupsen/logrus"
)

type natsClient struct {
    cfg *NATSConfig
    nc  *nats.Conn
    js  nats.JetStreamContext
    log *logrus.Logger
}

func NewNATSClient(cfg *NATSConfig, log *logrus.Logger) (MiddlewareClient, error) {
    return &natsClient{cfg: cfg, log: log}, nil
}

func (c *natsClient) Connect() error {
    opts := []nats.Option{
        nats.Name(c.cfg.ClientName),
        nats.UserInfo(c.cfg.Username, c.cfg.Password),
        nats.ReconnectWait(time.Duration(c.cfg.ReconnectWait) * time.Second),
        nats.MaxReconnects(c.cfg.MaxReconnects),
        nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
            c.log.Warnf("[NATS] disconnected: %v", err)
        }),
        nats.ReconnectHandler(func(_ *nats.Conn) {
            c.log.Info("[NATS] reconnected")
        }),
    }
    if c.cfg.Token != "" {
        opts = append(opts, nats.Token(c.cfg.Token))
    }

    nc, err := nats.Connect(c.cfg.URL, opts...)
    if err != nil {
        return err
    }
    c.nc = nc

    if c.cfg.JetStreamEnabled {
        if c.js, err = nc.JetStream(); err != nil {
            c.log.Warnf("[NATS] jetstream unavailable: %v", err)
        }
    }
    return nil
}

func (c *natsClient) Disconnect() {
    if c.nc != nil {
        c.nc.Drain()
    }
}

func (c *natsClient) IsConnected() bool { return c.nc != nil && c.nc.IsConnected() }

func (c *natsClient) Status() ConnStatus {
    if c.IsConnected() {
        return ConnStatusConnected
    }
    return ConnStatusDisconnected
}

func (c *natsClient) Subscribe(subject string, handler MessageHandler) error {
    _, err := c.nc.Subscribe(subject, func(msg *nats.Msg) {
        var m Message
        if err := json.Unmarshal(msg.Data, &m); err != nil {
            c.log.Errorf("[NATS] unmarshal error on %s: %v", msg.Subject, err)
            return
        }
        if err := handler(msg.Subject, &m); err != nil {
            c.log.Errorf("[NATS] handler error on %s: %v", msg.Subject, err)
        }
    })
    return err
}

func (c *natsClient) Publish(subject string, msgType string, body interface{}) error {
    msg := buildMessage(c.cfg.ClientName, msgType, body)
    payload, err := json.Marshal(msg)
    if err != nil {
        return err
    }
    return c.nc.Publish(subject, payload)
}

// Request 发起请求/响应（NATS 特有）
func (c *natsClient) Request(subject string, msgType string, body interface{}, timeout time.Duration) (*Message, error) {
    msg := buildMessage(c.cfg.ClientName, msgType, body)
    payload, err := json.Marshal(msg)
    if err != nil {
        return nil, err
    }
    resp, err := c.nc.Request(subject, payload, timeout)
    if err != nil {
        return nil, err
    }
    var m Message
    if err = json.Unmarshal(resp.Data, &m); err != nil {
        return nil, err
    }
    return &m, nil
}
```

---

## 3. EdgeX 节点注册

### 3.1 两阶段注册流程概述

EdgeX 节点注册分为两个阶段：

- **Stage 1（被动注册）**：EdgeX 节点主动发布 `edgex/nodes/register`，EdgeOS 接收并处理注册请求。这是节点初始化时的标准注册流程。
- **Stage 2（主动发现）**：EdgeOS 主动发布 `edgex/cmd/nodes/register`，触发已注册或待注册的 EdgeX 节点重新上报注册信息。用于运维管理场景——例如节点重启后需重新建立映射、拓扑变更后需刷新节点列表、或 EdgeOS 重启后需主动探测可用节点。

```
Stage 1: 被动注册（EdgeX → EdgeOS）
EdgeX 节点 ──(edgex/nodes/register)──→ MQTT Broker ──→ EdgeOS（处理注册）

Stage 2: 主动发现（EdgeOS → EdgeX → EdgeOS）
EdgeOS ──(edgex/cmd/nodes/register)──→ MQTT Broker ──→ EdgeX 节点
                                                     │
                                                     ▼
                                          EdgeX 节点重新发布:
                                          edgex/nodes/register
                                                     │
                                                     ▼
                                          MQTT Broker ──→ EdgeOS（处理注册）
```

### 3.2 Stage 1：被动注册

```
EdgeX                    消息中间件               EdgeOS
  │                          │                      │
  │── node_register ────────►│                      │
  │   Topic: edgex/nodes/register                   │
  │                          │──────────────────────►│
  │                          │      处理注册消息      │
  │                          │   1. 验证消息格式      │
  │                          │   2. 检查节点是否已存在 │
  │                          │   3. 持久化节点信息    │
  │                          │   4. 返回注册响应      │
  │◄── node_register_resp ───│◄──────────────────────│
  │   包含 access_token       │                      │
```

### 3.2 订阅主题

| 协议 | 主题 | QoS | 说明 |
|------|------|-----|------|
| MQTT | `edgex/nodes/register` | 1 | **Stage 1: 节点被动注册** |
| MQTT | `edgex/nodes/unregister` | 1 | 节点注销 |
| MQTT | `edgex/cmd/nodes/register` | 1 | **Stage 2: EdgeOS 主动触发节点重新注册** |
| NATS | `edgex.nodes.register` | - | **Stage 1: 节点被动注册** |
| NATS | `edgex.nodes.unregister` | - | 节点注销 |
| NATS | `edgex.cmd.nodes.register` | - | **Stage 2: EdgeOS 主动触发节点重新注册** |

### 3.3 消息结构

**接收（EdgeX → EdgeOS）：**
```json
{
  "header": {
    "message_id": "msg-node-reg-001",
    "timestamp": 1744680000000,
    "source": "edgex-node-001",
    "destination": "edgeos-queen",
    "message_type": "node_register",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "node_name": "EdgeX Gateway Node",
    "model": "edge-gateway",
    "version": "1.0.0",
    "api_version": "v1",
    "capabilities": ["shadow-sync", "heartbeat", "device-control", "task-execution"],
    "protocol": "edgeOS(MQTT)",
    "endpoint": { "host": "127.0.0.1", "port": 8082 },
    "metadata": { "os": "linux", "arch": "amd64", "hostname": "edgex-node-001.local" }
  }
}
```

**回复（EdgeOS → EdgeX）：**
```json
{
  "header": {
    "message_id": "msg-node-reg-resp-001",
    "timestamp": 1744680000500,
    "source": "edgeos-queen",
    "destination": "edgex-node-001",
    "message_type": "node_register_response",
    "version": "1.0",
    "correlation_id": "msg-node-reg-001"
  },
  "body": {
    "success": true,
    "node_id": "edgex-node-001",
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600,
    "message": "Node registered successfully"
  }
}
```

### 3.4 处理器实现

```go
// internal/handlers/node_handler.go

package handlers

import (
    "fmt"
    "time"

    "github.com/sirupsen/logrus"
)

type NodeHandler struct {
    nodeService NodeService
    publisher   Publisher
    log         *logrus.Logger
}

func (h *NodeHandler) HandleRegister(topic string, msg *Message) error {
    var body NodeRegisterBody
    if err := decodeBody(msg.Body, &body); err != nil {
        return fmt.Errorf("decode node_register body: %w", err)
    }

    // 1. 幂等检查：已注册则直接返回成功
    existing, _ := h.nodeService.GetByID(body.NodeID)
    if existing != nil {
        h.log.Infof("[NodeReg] node %s already registered, update info", body.NodeID)
        existing.Status = "online"
        existing.UpdatedAt = time.Now().UnixMilli()
        _ = h.nodeService.Update(existing)
        return h.replyRegisterSuccess(msg.Header, body.NodeID)
    }

    // 2. 持久化节点
    node := &Node{
        ID:           body.NodeID,
        Name:         body.NodeName,
        Model:        body.Model,
        Version:      body.Version,
        Capabilities: body.Capabilities,
        Protocol:     body.Protocol,
        Endpoint:     body.Endpoint,
        Metadata:     body.Metadata,
        Status:       "online",
        RegisteredAt: time.Now().UnixMilli(),
        UpdatedAt:    time.Now().UnixMilli(),
    }
    if err := h.nodeService.Save(node); err != nil {
        return fmt.Errorf("save node: %w", err)
    }
    h.log.Infof("[NodeReg] registered node: %s (%s)", body.NodeID, body.NodeName)

    // 3. 回复注册响应
    return h.replyRegisterSuccess(msg.Header, body.NodeID)
}

func (h *NodeHandler) replyRegisterSuccess(reqHeader MessageHeader, nodeID string) error {
    respBody := map[string]interface{}{
        "success":      true,
        "node_id":      nodeID,
        "access_token": generateToken(nodeID),
        "expires_in":   3600,
        "message":      "Node registered successfully",
    }
    respHeader := MessageHeader{
        MessageID:     generateUUID(),
        Timestamp:     time.Now().UnixMilli(),
        Source:        "edgeos-queen",
        Destination:   reqHeader.Source,
        MessageType:   "node_register_response",
        Version:       "1.0",
        CorrelationID: reqHeader.MessageID,
    }
    topic := fmt.Sprintf("edgex/responses/%s/%s", reqHeader.Source, reqHeader.MessageID)
    return h.publisher.Publish(topic, respHeader, respBody)
}

func (h *NodeHandler) HandleUnregister(topic string, msg *Message) error {
    var body NodeUnregisterBody
    if err := decodeBody(msg.Body, &body); err != nil {
        return fmt.Errorf("decode node_unregister body: %w", err)
    }
    return h.nodeService.MarkOffline(body.NodeID)
}
```

### 3.5 节点模型

```go
// internal/models/node.go

type Node struct {
    ID           string            `json:"id"`
    Name         string            `json:"name"`
    Model        string            `json:"model"`
    Version      string            `json:"version"`
    APIVersion   string            `json:"api_version"`
    Capabilities []string          `json:"capabilities"`
    Protocol     string            `json:"protocol"`
    Endpoint     NodeEndpoint      `json:"endpoint"`
    Metadata     map[string]string `json:"metadata"`
    Status       string            `json:"status"`  // online / offline / error
    RegisteredAt int64             `json:"registered_at"`
    UpdatedAt    int64             `json:"updated_at"`
    LastHeartbeat int64            `json:"last_heartbeat"`
}

type NodeEndpoint struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}
```

### 3.7 Stage 2：主动节点发现

EdgeOS 支持主动向中间件发布节点发现请求，触发 EdgeX 节点重新注册。

**API 端点：**

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/edgex/discover` | 向第一个已连接中间件发布发现请求（广播模式） |
| `POST` | `/api/edgex/discover/:middlewareId` | 向指定中间件实例发布发现请求 |

**请求示例：**
```bash
# 触发所有已连接中间件的节点发现
curl -X POST http://localhost:8000/api/edgex/discover \
  -H "Authorization: Bearer <token>"

# 向指定中间件触发发现
curl -X POST http://localhost:8000/api/edgex/discover/mqtt-1 \
  -H "Authorization: Bearer <token>"
```

**实现原理（`messaging.Manager`）：**

```go
// PublishNodeDiscovery 主动向第一个已连接中间件发布节点发现请求
func (m *Manager) PublishNodeDiscovery() error {
    msg := map[string]interface{}{
        "header": map[string]interface{}{
            "message_type": "discovery_request",
            "source":       "edgeos",
            "timestamp":    time.Now().UnixMilli(),
        },
        "body": map[string]interface{}{},
    }
    payload, _ := json.Marshal(msg)
    return m.publishToFirstClient("edgex/cmd/nodes/register", payload)
}

// PublishNodeDiscoveryTo 向指定中间件发布节点发现请求
func (m *Manager) PublishNodeDiscoveryTo(middlewareID string) error {
    m.mu.RLock()
    defer m.mu.RUnlock()
    entry, ok := m.clients[middlewareID]
    if !ok || entry.client == nil || !entry.client.IsConnected() {
        return fmt.Errorf("middleware %s not connected", middlewareID)
    }
    msg := map[string]interface{}{
        "header": map[string]interface{}{
            "message_type": "discovery_request",
            "source":       "edgeos",
            "timestamp":    time.Now().UnixMilli(),
        },
        "body": map[string]interface{}{},
    }
    payload, _ := json.Marshal(msg)
    return entry.publishFn("edgex/cmd/nodes/register", payload)
}
```

**NATS 等效实现：**

```go
// NATS 客户端 PublishNodeDiscoveryTo
func (c *natsClient) PublishDiscoveryRequest() error {
    msg := map[string]interface{}{
        "header": map[string]interface{}{
            "message_type": "discovery_request",
            "source":       "edgeos",
            "timestamp":    time.Now().UnixMilli(),
        },
        "body": map[string]interface{}{},
    }
    payload, _ := json.Marshal(msg)
    return c.nc.Publish("edgex.cmd.nodes.register", payload)
}
```

**节点发现请求消息格式：**

```json
{
  "header": {
    "message_type": "discovery_request",
    "source": "edgeos",
    "timestamp": 1744680000000
  },
  "body": {}
}
```

EdgeX 节点收到该消息后，应立即重新发布 `edgex/nodes/register` 消息，从而触发完整的 Stage 1 注册流程（验证 + 持久化 + 响应）。

---

## 4. EdgeX 子设备列表同步

### 4.1 功能流程

```
EdgeX                    消息中间件               EdgeOS
  │                          │                      │
  │── device_report ────────►│                      │
  │   Topic: edgex/devices/report                   │
  │                          │──────────────────────►│
  │                          │   1. 校验来源节点是否  │
  │                          │      已注册            │
  │                          │   2. 遍历设备列表      │
  │                          │   3. 新增或更新设备    │
  │                          │   4. 标记消失的设备    │
  │                          │                      │
  │                          │   (可选) EdgeOS主动    │
  │◄─── list query ──────────│◄─ 发 list 查询命令    │
  │── device list response ─►│──────────────────────►│
```

EdgeOS 也可主动发起设备同步：向 `edgex/devices/{node_id}/list` 发布查询命令，EdgeX 以 `edgex/devices/report` 响应。

### 4.2 订阅 / 发布主题

| 方向 | 协议 | 主题 | 说明 |
|------|------|------|------|
| 订阅（收） | MQTT | `edgex/devices/report` | 接收设备列表上报 |
| 订阅（收） | NATS | `edgex.devices.report` | 接收设备列表上报 |
| 发布（发） | MQTT | `edgex/devices/{node_id}/list` | 主动查询设备列表 |
| 发布（发） | NATS | `edgex.devices.{node_id}.list` | 主动查询设备列表 |
| 发布（发） | MQTT | `edgex/cmd/{node_id}/discover` | 触发设备发现 |

### 4.3 消息结构

**接收（EdgeX → EdgeOS）：**
```json
{
  "header": { "message_type": "device_report", "source": "edgex-node-001" },
  "body": {
    "node_id": "edgex-node-001",
    "devices": [
      {
        "device_id": "device-001",
        "device_name": "Modbus TCP Device",
        "device_profile": "modbus-tcp-device",
        "service_name": "modbus-tcp-service",
        "labels": ["sensor", "modbus"],
        "description": "Test Modbus TCP device",
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

### 4.4 处理器实现

```go
// internal/handlers/device_handler.go

package handlers

type DeviceHandler struct {
    nodeService   NodeService
    deviceService DeviceService
    log           *logrus.Logger
}

func (h *DeviceHandler) HandleDeviceReport(topic string, msg *Message) error {
    var body DeviceReportBody
    if err := decodeBody(msg.Body, &body); err != nil {
        return fmt.Errorf("decode device_report: %w", err)
    }

    // 校验节点是否已注册
    node, err := h.nodeService.GetByID(body.NodeID)
    if err != nil || node == nil {
        return fmt.Errorf("node %s not registered", body.NodeID)
    }

    reportedIDs := make(map[string]struct{})
    for _, d := range body.Devices {
        reportedIDs[d.DeviceID] = struct{}{}

        existing, _ := h.deviceService.GetByID(body.NodeID, d.DeviceID)
        device := &Device{
            NodeID:         body.NodeID,
            DeviceID:       d.DeviceID,
            DeviceName:     d.DeviceName,
            DeviceProfile:  d.DeviceProfile,
            ServiceName:    d.ServiceName,
            Labels:         d.Labels,
            Description:    d.Description,
            AdminState:     d.AdminState,
            OperatingState: d.OperatingState,
            Properties:     d.Properties,
            UpdatedAt:      time.Now().UnixMilli(),
        }
        if existing == nil {
            device.CreatedAt = device.UpdatedAt
            if err = h.deviceService.Save(device); err != nil {
                h.log.Errorf("[DeviceSync] save device %s: %v", d.DeviceID, err)
                continue
            }
            h.log.Infof("[DeviceSync] new device: %s / %s", body.NodeID, d.DeviceID)
        } else {
            if err = h.deviceService.Update(device); err != nil {
                h.log.Errorf("[DeviceSync] update device %s: %v", d.DeviceID, err)
            }
        }
    }

    // 标记不在上报列表中的设备为离线
    allDevices, _ := h.deviceService.ListByNode(body.NodeID)
    for _, d := range allDevices {
        if _, ok := reportedIDs[d.DeviceID]; !ok {
            _ = h.deviceService.MarkOffline(body.NodeID, d.DeviceID)
        }
    }
    return nil
}

// TriggerDiscover 主动触发设备发现
func (h *DeviceHandler) TriggerDiscover(nodeID string, protocol string, publisher Publisher) error {
    body := map[string]interface{}{
        "protocol": protocol,
        "options": map[string]interface{}{
            "auto_register":    true,
            "sync_immediately": true,
        },
    }
    topic := fmt.Sprintf("edgex/cmd/%s/discover", nodeID)
    return publisher.Publish(topic, "discover_command", body)
}
```

### 4.5 设备模型

```go
// internal/models/device.go

type Device struct {
    NodeID         string                 `json:"node_id"`
    DeviceID       string                 `json:"device_id"`
    DeviceName     string                 `json:"device_name"`
    DeviceProfile  string                 `json:"device_profile"`
    ServiceName    string                 `json:"service_name"`
    Labels         []string               `json:"labels"`
    Description    string                 `json:"description"`
    AdminState     string                 `json:"admin_state"`
    OperatingState string                 `json:"operating_state"`
    Properties     map[string]interface{} `json:"properties"`
    Status         string                 `json:"status"`  // online / offline
    CreatedAt      int64                  `json:"created_at"`
    UpdatedAt      int64                  `json:"updated_at"`
}
```

---

## 5. EdgeX 子设备点位同步（物模型）

### 5.1 设计原则：物模型驱动的两阶段数据流

EdgeX 的点位数据通过**实时数据消息**（`edgex/data/{node_id}/{device_id}`）传递，遵循以下两阶段规则：

| 阶段 | 触发时机 | 消息内容 | EdgeOS 处理方式 |
|------|---------|---------|---------------|
| **全量上报**（物模型初始化） | 设备连接后首次上报，或 EdgeOS 主动触发同步 | `points` 包含该设备**全部**点位及其当前值 | 以 `point_id` 为 key 全量写入点位缓存，建立物模型快照 |
| **差量上报**（增量实时数据） | 后续周期性或变化触发 | `points` 仅包含**自上次上报以来发生变化**的点位 | Merge 到全量缓存，仅更新出现的 key，不删除未出现的点位 |

> **关键原则**：两种消息结构完全相同，区分方式由 EdgeOS 业务逻辑判断——设备无缓存时首条视为全量，有缓存后每条均视为差量 Merge。`edgex/points/report` 是独立的**点位元数据上报**（含类型/单位/地址等配置信息），与实时数据上报相互独立、可选。

### 5.2 功能流程

```
EdgeX                    消息中间件               EdgeOS
  │                          │                      │
  │  ─(可选) point_report ──►│                      │
  │   Topic: edgex/points/report                    │
  │                          │──────────────────────►│
  │                          │  保存点位元数据定义    │
  │                          │  (类型/单位/地址/范围) │
  │                          │                      │
  │  ── data [首次·全量] ────►│                      │
  │   Topic: edgex/data/{node}/{dev}                │
  │   body.points = {所有点位:当前值}                 │
  │                          │──────────────────────►│
  │                          │  1. 设备无缓存         │
  │                          │  2. 全量写入点位缓存   │
  │                          │  3. 建立物模型快照     │
  │                          │  4. 广播 data_update  │
  │                          │     (is_full_snapshot=true)
  │                          │                      │
  │  ── data [后续·差量] ────►│                      │
  │   body.points = {仅变化点位}                     │
  │                          │──────────────────────►│
  │                          │  1. 设备缓存已存在     │
  │                          │  2. Merge 到全量缓存  │
  │                          │     (只更新出现的key)  │
  │                          │  3. 广播 data_update  │
  │                          │     (is_full_snapshot=false)
```

EdgeOS 可主动触发全量同步：向 `edgex/cmd/{node_id}/sync` 发布请求，EdgeX 将重新触发全量数据上报。

### 5.3 订阅 / 发布主题

| 方向 | 协议 | 主题 | 说明 |
|------|------|------|------|
| 订阅（收） | MQTT | `edgex/points/report` | 接收点位元数据定义（可选） |
| 订阅（收） | MQTT | `edgex/data/#` | 接收实时数据（全量或差量） |
| 订阅（收） | NATS | `edgex.points.report` | 接收点位元数据定义（可选） |
| 订阅（收） | NATS | `edgex.data.>` | 接收实时数据（全量或差量） |
| 发布（发） | MQTT | `edgex/cmd/{node_id}/sync` | 主动触发 EdgeX 全量上报 |
| 发布（发） | MQTT | `edgex/points/{node_id}/{device_id}/sync` | 请求指定设备点位元数据同步 |

### 5.4 消息结构

**点位元数据上报 `edgex/points/report`（可选，含配置信息）：**

```json
{
  "header": { "message_type": "point_report", "source": "edgex-node-001" },
  "body": {
    "node_id": "edgex-node-001",
    "device_id": "Room_FC_2014_19",
    "points": [
      {
        "point_id": "SetPoint.Value",
        "point_name": "设定温度",
        "resource_name": "SetPoint.Value",
        "value_type": "Float32",
        "access_mode": "RW",
        "unit": "°C",
        "minimum": 16,
        "maximum": 30,
        "address": "40001",
        "data_type": "holding_register",
        "scale": 1.0,
        "offset": 0
      }
    ]
  }
}
```

**首次全量实时数据上报（物模型初始化）：**

```json
{
  "header": {
    "message_id": "msg-c9028d5e3e3018e6324ad61c91927561",
    "timestamp": 1776312621808,
    "source": "edgex-node-001",
    "message_type": "data",
    "version": "1.0"
  },
  "body": {
    "device_id": "Room_FC_2014_19",
    "node_id": "edgex-node-001",
    "points": {
      "SetPoint.Value": 19,
      "Setpoint.1": 19,
      "Setpoint.2": 19,
      "Setpoint.3": 19,
      "State.Chiller": 0,
      "State.Heater": 1,
      "Temperature.Indoor": 18.8,
      "Temperature.Outdoor": 12,
      "Temperature.Water": 39.7
    },
    "quality": "Good",
    "timestamp": 1776312621573
  }
}
```

> 首次上报包含设备**所有点位**，EdgeOS 以此建立物模型全量快照。`points` 的 key 即为 `point_id`，value 为当前采集值。

**后续差量实时数据上报（仅变化点位）：**

```json
{
  "header": {
    "message_id": "msg-c9028d5e3e3018e6324ad61c91927561",
    "timestamp": 1776312623000,
    "source": "edgex-node-001",
    "message_type": "data",
    "version": "1.0"
  },
  "body": {
    "device_id": "Room_FC_2014_19",
    "node_id": "edgex-node-001",
    "points": {
      "SetPoint.Value": 20,
      "Setpoint.1": 20,
      "Setpoint.2": 20
    },
    "quality": "Good",
    "timestamp": 1776312623000
  }
}
```

> 差量上报只包含**发生变化**的点位（`Setpoint.3`、`State.*`、`Temperature.*` 未变化故不出现）。EdgeOS 收到后仅更新这三个点位的值，其余点位保持上次快照中的值不变。

### 5.5 处理器实现

```go
// internal/handlers/point_handler.go

package handlers

import (
    "fmt"
    "strings"
    "time"

    "github.com/sirupsen/logrus"
)

type PointHandler struct {
    nodeService   NodeService
    deviceService DeviceService
    pointService  PointService
    broadcast     BroadcastFunc  // 向前端 WebSocket 推送
    log           *logrus.Logger
}

// HandlePointReport 处理点位元数据上报（edgex/points/report，可选）
// 保存点位的配置定义：类型、单位、地址、范围等，不覆盖 CurrentValue/Quality/LastUpdated。
func (h *PointHandler) HandlePointReport(topic string, msg *Message) error {
    var body PointReportBody
    if err := decodeBody(msg.Body, &body); err != nil {
        return fmt.Errorf("decode point_report: %w", err)
    }

    dev, err := h.deviceService.GetByID(body.NodeID, body.DeviceID)
    if err != nil || dev == nil {
        return fmt.Errorf("device %s/%s not found", body.NodeID, body.DeviceID)
    }

    now := time.Now().UnixMilli()
    for _, p := range body.Points {
        point := &Point{
            NodeID:       body.NodeID,
            DeviceID:     body.DeviceID,
            PointID:      p.PointID,
            PointName:    p.PointName,
            ResourceName: p.ResourceName,
            ValueType:    p.ValueType,
            AccessMode:   p.AccessMode,
            Unit:         p.Unit,
            Minimum:      p.Minimum,
            Maximum:      p.Maximum,
            Address:      p.Address,
            DataType:     p.DataType,
            Scale:        p.Scale,
            Offset:       p.Offset,
            UpdatedAt:    now,
        }
        existing, _ := h.pointService.GetByID(body.NodeID, body.DeviceID, p.PointID)
        if existing == nil {
            point.CreatedAt = now
            _ = h.pointService.Save(point)
        } else {
            // 只更新元数据字段，不覆盖 CurrentValue / Quality / LastUpdated
            _ = h.pointService.UpdateMeta(point)
        }
    }
    h.log.Infof("[PointMeta] synced %d point definitions for %s/%s",
        len(body.Points), body.NodeID, body.DeviceID)
    return nil
}

// HandleRealtimeData 处理实时数据上报（edgex/data/{node_id}/{device_id}）
//
// 两阶段 Merge 策略：
//   - 首次上报（设备无缓存）：body.points 为物模型全量数据，全量写入缓存，
//     同时为未经 point_report 注册的点位自动创建占位记录（point_id 作为 point_name）。
//   - 后续上报（设备已有缓存）：body.points 仅含变化点位，执行 Merge，
//     未出现在本次 payload 中的点位保持原值不变。
func (h *PointHandler) HandleRealtimeData(topic string, msg *Message) error {
    var body DataBody
    if err := decodeBody(msg.Body, &body); err != nil {
        return fmt.Errorf("decode data: %w", err)
    }

    // 兜底：从 topic 解析 nodeId / deviceId
    nodeID, deviceID := body.NodeID, body.DeviceID
    if nodeID == "" || deviceID == "" {
        parts := strings.Split(topic, "/")
        if len(parts) >= 4 {
            nodeID, deviceID = parts[2], parts[3]
        }
    }

    isFirstReport := !h.pointService.HasCache(nodeID, deviceID)

    now := time.Now().UnixMilli()
    for pointID, value := range body.Points {
        existing, _ := h.pointService.GetByID(nodeID, deviceID, pointID)
        if existing == nil {
            // 自动创建点位占位（元数据待 point_report 补全）
            placeholder := &Point{
                NodeID:       nodeID,
                DeviceID:     deviceID,
                PointID:      pointID,
                PointName:    pointID, // 暂用 point_id 作为名称
                CurrentValue: value,
                Quality:      body.Quality,
                LastUpdated:  body.Timestamp,
                CreatedAt:    now,
                UpdatedAt:    now,
            }
            _ = h.pointService.Save(placeholder)
        } else {
            // Merge：只更新当前值，保留元数据不变
            _ = h.pointService.UpdateValue(nodeID, deviceID, pointID, value, body.Quality, body.Timestamp)
        }
    }

    if isFirstReport {
        h.log.Infof("[PointSync] full snapshot: %d points for %s/%s",
            len(body.Points), nodeID, deviceID)
    } else {
        h.log.Debugf("[PointSync] delta merge: %d changed points for %s/%s",
            len(body.Points), nodeID, deviceID)
    }

    // 广播实时事件到前端
    // payload 只含本次上报的点位（差量），前端 store 同步执行 merge
    h.broadcast(RealtimeDataEvent{
        NodeID:         nodeID,
        DeviceID:       deviceID,
        Points:         body.Points,
        Timestamp:      body.Timestamp,
        Quality:        body.Quality,
        IsFullSnapshot: isFirstReport,
    })
    return nil
}
```

### 5.6 PointService 接口

```go
// internal/services/point_service.go

type PointService interface {
    // HasCache 检查设备是否已有点位缓存（用于判断全量 vs 差量 Merge）
    HasCache(nodeID, deviceID string) bool

    GetByID(nodeID, deviceID, pointID string) (*Point, error)
    Save(p *Point) error

    // UpdateMeta 仅更新点位元数据字段，不覆盖 CurrentValue/Quality/LastUpdated
    UpdateMeta(p *Point) error

    // UpdateValue 差量 Merge 的最小操作单元：更新当前值+质量+时间戳
    UpdateValue(nodeID, deviceID, pointID string, value interface{}, quality string, ts int64) error

    ListByDevice(nodeID, deviceID string) ([]*Point, error)
}
```

### 5.7 点位模型

```go
// internal/models/point.go

type Point struct {
    NodeID       string      `json:"node_id"`
    DeviceID     string      `json:"device_id"`
    PointID      string      `json:"point_id"`
    PointName    string      `json:"point_name"`         // 来自 point_report，缺省时等于 point_id
    ResourceName string      `json:"resource_name"`
    ValueType    string      `json:"value_type"`         // Float32 / Int16 / Bool / ...（元数据）
    AccessMode   string      `json:"access_mode"`        // R / W / RW（元数据，缺省时为空）
    Unit         string      `json:"unit"`
    Minimum      interface{} `json:"minimum,omitempty"`
    Maximum      interface{} `json:"maximum,omitempty"`
    Address      string      `json:"address"`
    DataType     string      `json:"data_type"`
    Scale        float64     `json:"scale"`
    Offset       float64     `json:"offset"`
    CurrentValue interface{} `json:"current_value"`      // 最新采集值（由实时数据驱动）
    Quality      string      `json:"quality"`            // Good / Bad / Uncertain
    LastUpdated  int64       `json:"last_updated"`       // 最新值对应的设备时间戳
    CreatedAt    int64       `json:"created_at"`
    UpdatedAt    int64       `json:"updated_at"`         // 元数据最后修改时间
}

// RealtimeDataEvent 向前端广播的实时数据事件
type RealtimeDataEvent struct {
    NodeID         string                 `json:"node_id"`
    DeviceID       string                 `json:"device_id"`
    Points         map[string]interface{} `json:"points"`           // 本次上报的点位（全量或差量）
    Timestamp      int64                  `json:"timestamp"`
    Quality        string                 `json:"quality"`
    IsFullSnapshot bool                   `json:"is_full_snapshot"` // true=全量，前端可用于初始化展示
}
```

### 5.8 全量 vs 差量 Merge 示意

```
设备: Room_FC_2014_19  物模型缓存状态

T1 收到全量上报（IsFullSnapshot=true）:
  payload.points = 9 个点位
  缓存写入全部 9 个:
    SetPoint.Value=19  Setpoint.1=19  Setpoint.2=19  Setpoint.3=19
    State.Chiller=0    State.Heater=1
    Temperature.Indoor=18.8  Temperature.Outdoor=12  Temperature.Water=39.7

T2 收到差量上报（IsFullSnapshot=false）:
  payload.points = { SetPoint.Value:20, Setpoint.1:20, Setpoint.2:20 }
  Merge 结果（仅更新 3 个，其余 6 个保持 T1 值）:
    SetPoint.Value=20 ← 已更新   Setpoint.1=20 ← 已更新   Setpoint.2=20 ← 已更新
    Setpoint.3=19    ← T1 值     State.Chiller=0 ← T1 值   State.Heater=1 ← T1 值
    Temperature.Indoor=18.8 ← T1 值  Temperature.Outdoor=12 ← T1 值  Temperature.Water=39.7 ← T1 值
```

---

## 6. EdgeX 子设备双向控制

### 6.1 功能流程

```
EdgeOS (控制发起)        消息中间件               EdgeX (执行)
    │                        │                       │
    │─── write_command ──────►│                      │
    │  edgex/cmd/{node}/{dev}/write                  │
    │                        │──────────────────────►│
    │                        │  EdgeX 写入物理设备   │
    │                        │◄──────────────────────│
    │◄─── 命令响应 ───────────│  edgex/responses/... │
    │                        │                       │
    │                        │                       │
    │  (实时状态反馈路径)      │                       │
    │◄── data ───────────────│◄──────────────────────│
    │  edgex/data/{node}/{dev} 实时数据包含控制结果   │
```

**双向指的是：**
- **下行（EdgeOS → EdgeX）**：写入命令、任务控制、配置更新、设备发现
- **上行（EdgeX → EdgeOS）**：命令执行响应、实时数据（含写后读验证）、告警

### 6.2 下行控制主题

| 协议 | 主题 | 消息类型 | 说明 |
|------|------|---------|------|
| MQTT | `edgex/cmd/{node_id}/{device_id}/write` | `write_command` | 写入设备点位 |
| MQTT | `edgex/cmd/{node_id}/discover` | `discover_command` | 触发设备发现 |
| MQTT | `edgex/cmd/{node_id}/task/create` | `task_create` | 创建采集任务 |
| MQTT | `edgex/cmd/{node_id}/task/{task_id}/pause` | `task_control` | 暂停任务 |
| MQTT | `edgex/cmd/{node_id}/task/{task_id}/resume` | `task_control` | 恢复任务 |
| MQTT | `edgex/cmd/{node_id}/task/{task_id}/stop` | `task_control` | 停止任务 |
| MQTT | `edgex/cmd/{node_id}/config/update` | `config_update` | 更新节点配置 |
| NATS | `edgex.cmd.{node_id}.{device_id}.write` | `write_command` | 写入设备点位 |

### 6.3 上行响应主题

| 协议 | 主题 | 说明 |
|------|------|------|
| MQTT | `edgex/responses/{node_id}/{request_id}` | 命令执行响应 |
| MQTT | `edgex/responses/{node_id}/error/{request_id}` | 错误响应 |
| NATS | `edgex.res.{node_id}.{request_id}` | 命令执行响应 |

### 6.4 写入命令消息

```json
{
  "header": {
    "message_id": "msg-cmd-write-001",
    "timestamp": 1744680000000,
    "source": "edgeos-queen",
    "destination": "edgex-node-001",
    "message_type": "write_command",
    "version": "1.0",
    "correlation_id": "req-write-001"
  },
  "body": {
    "request_id": "req-write-001",
    "device_id": "device-001",
    "timestamp": 1744680000000,
    "points": {
      "Switch": true,
      "Setpoint": 80.5
    },
    "options": {
      "confirm": true,
      "timeout_seconds": 10
    }
  }
}
```

### 6.5 控制服务实现

```go
// internal/services/control_service.go

package services

import (
    "fmt"
    "sync"
    "time"
)

type ControlService struct {
    publisher   Publisher
    pendingReqs sync.Map  // requestID -> chan *CommandResponse
    log         *logrus.Logger
}

// WritePoints 向 EdgeX 写入设备点位（异步等待响应）
func (s *ControlService) WritePoints(nodeID, deviceID string, points map[string]interface{}, timeout time.Duration) (*CommandResponse, error) {
    reqID := generateUUID()
    respCh := make(chan *CommandResponse, 1)
    s.pendingReqs.Store(reqID, respCh)
    defer s.pendingReqs.Delete(reqID)

    body := map[string]interface{}{
        "request_id": reqID,
        "device_id":  deviceID,
        "timestamp":  time.Now().UnixMilli(),
        "points":     points,
        "options": map[string]interface{}{
            "confirm":         true,
            "timeout_seconds": int(timeout.Seconds()),
        },
    }
    topic := fmt.Sprintf("edgex/cmd/%s/%s/write", nodeID, deviceID)
    if err := s.publisher.Publish(topic, "write_command", body); err != nil {
        return nil, fmt.Errorf("publish write_command: %w", err)
    }

    select {
    case resp := <-respCh:
        return resp, nil
    case <-time.After(timeout):
        return nil, fmt.Errorf("write command timeout after %v", timeout)
    }
}

// HandleCommandResponse 处理来自 EdgeX 的命令响应
func (s *ControlService) HandleCommandResponse(topic string, msg *Message) error {
    var body CommandResponseBody
    if err := decodeBody(msg.Body, &body); err != nil {
        return err
    }
    if ch, ok := s.pendingReqs.Load(body.RequestID); ok {
        ch.(chan *CommandResponse) <- &CommandResponse{
            RequestID: body.RequestID,
            Success:   body.Success,
            Message:   body.Message,
            Data:      body.Data,
        }
    }
    return nil
}

// CreateTask 创建采集任务
func (s *ControlService) CreateTask(nodeID string, task *TaskCreateRequest) error {
    body := map[string]interface{}{
        "task_id":   task.TaskID,
        "task_name": task.TaskName,
        "device_id": task.DeviceID,
        "schedule":  task.Schedule,
        "points":    task.Points,
        "options":   task.Options,
    }
    topic := fmt.Sprintf("edgex/cmd/%s/task/create", nodeID)
    return s.publisher.Publish(topic, "task_create", body)
}

// ControlTask 任务控制（pause/resume/stop）
func (s *ControlService) ControlTask(nodeID, taskID, action string) error {
    body := map[string]interface{}{
        "task_id": taskID,
        "action":  action,
    }
    topic := fmt.Sprintf("edgex/cmd/%s/task/%s/%s", nodeID, taskID, action)
    return s.publisher.Publish(topic, "task_control", body)
}
```

### 6.6 控制 REST API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/nodes/:nodeId/devices/:deviceId/write` | 写入点位值 |
| `POST` | `/api/v1/nodes/:nodeId/discover` | 触发设备发现 |
| `POST` | `/api/v1/nodes/:nodeId/tasks` | 创建采集任务 |
| `PUT` | `/api/v1/nodes/:nodeId/tasks/:taskId/pause` | 暂停任务 |
| `PUT` | `/api/v1/nodes/:nodeId/tasks/:taskId/resume` | 恢复任务 |
| `DELETE` | `/api/v1/nodes/:nodeId/tasks/:taskId` | 停止并删除任务 |
| `POST` | `/api/v1/nodes/:nodeId/config` | 更新节点配置 |
| `POST` | `/api/edgex/discover` | **Stage 2: 触发全量节点发现** |
| `POST` | `/api/edgex/discover/:middlewareId` | **Stage 2: 向指定中间件触发节点发现** |

---

## 7. 心跳与状态管理

### 7.1 订阅主题

| 协议 | 主题 | QoS | 说明 |
|------|------|-----|------|
| MQTT | `edgex/nodes/+/heartbeat` | 0 | 节点心跳 |
| MQTT | `edgex/nodes/+/status` | 1 | 节点状态变更 |
| MQTT | `edgex/nodes/+/online` | 2 | 节点上线 |
| MQTT | `edgex/nodes/+/offline` | 2 | 节点离线 |
| NATS | `edgex.nodes.heartbeat.>` | - | 节点心跳 |
| NATS | `edgex.nodes.status.>` | - | 节点状态 |

### 7.2 心跳超时自动离线

```go
// internal/services/heartbeat_service.go

const HeartbeatTimeout = 3 * time.Minute

type HeartbeatService struct {
    nodeService NodeService
    lastSeen    sync.Map  // nodeID -> int64(UnixMilli)
    log         *logrus.Logger
}

func (s *HeartbeatService) HandleHeartbeat(_ string, msg *Message) error {
    var body HeartbeatBody
    if err := decodeBody(msg.Body, &body); err != nil {
        return err
    }
    s.lastSeen.Store(body.NodeID, time.Now().UnixMilli())
    return s.nodeService.UpdateHeartbeat(body.NodeID, body.Status, body.Metrics)
}

// StartWatchdog 定时检查心跳超时
func (s *HeartbeatService) StartWatchdog(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.checkTimeouts()
        }
    }
}

func (s *HeartbeatService) checkTimeouts() {
    now := time.Now().UnixMilli()
    s.lastSeen.Range(func(key, value interface{}) bool {
        nodeID := key.(string)
        lastTS := value.(int64)
        if now-lastTS > HeartbeatTimeout.Milliseconds() {
            s.log.Warnf("[Heartbeat] node %s timeout, marking offline", nodeID)
            _ = s.nodeService.MarkOffline(nodeID)
        }
        return true
    })
}
```

---

## 8. 告警与事件处理

### 8.1 订阅主题

| 协议 | 主题 | QoS | 说明 |
|------|------|-----|------|
| MQTT | `edgex/events/alert` | 2 | 告警 |
| MQTT | `edgex/events/error` | 1 | 错误 |
| MQTT | `edgex/events/info` | 0 | 信息 |
| NATS | `edgex.events.alert` | - | 告警 |

### 8.2 处理器

```go
// internal/handlers/event_handler.go

func (h *EventHandler) HandleAlert(_ string, msg *Message) error {
    var body AlertBody
    if err := decodeBody(msg.Body, &body); err != nil {
        return err
    }
    alert := &Alert{
        AlertID:   body.AlertID,
        NodeID:    body.NodeID,
        DeviceID:  body.DeviceID,
        Type:      body.AlertType,
        Severity:  body.Severity,
        Message:   body.Message,
        Timestamp: body.Timestamp,
        Details:   body.Details,
    }
    _ = h.alertService.Save(alert)
    // 高优先级告警实时推送前端
    if body.Severity == "critical" || body.Severity == "error" {
        h.broadcast(AlertEvent{Alert: alert})
    }
    return nil
}
```

---

## 9. 错误处理与重试策略

### 9.1 消息处理错误分类

| 错误类型 | 处理方式 |
|---------|---------|
| JSON 解析失败 | 记录日志，丢弃消息，不重试 |
| 节点未注册 | 记录日志，丢弃消息，等待注册 |
| 存储失败（可恢复） | 指数退避重试，最多3次 |
| 网络/连接异常 | MQTT 自动重连 / NATS 自动重连 |
| 命令超时 | 返回超时错误，前端提示重试 |

### 9.2 指数退避实现

```go
// internal/utils/retry.go

type RetryConfig struct {
    MaxAttempts     int
    InitialInterval time.Duration
    MaxInterval     time.Duration
    Multiplier      float64
}

func RetryWithBackoff(cfg RetryConfig, fn func() error) error {
    interval := cfg.InitialInterval
    for i := 0; i < cfg.MaxAttempts; i++ {
        if err := fn(); err == nil {
            return nil
        } else if i == cfg.MaxAttempts-1 {
            return err
        }
        time.Sleep(interval)
        interval = time.Duration(float64(interval) * cfg.Multiplier)
        if interval > cfg.MaxInterval {
            interval = cfg.MaxInterval
        }
    }
    return nil
}
```

### 9.3 错误码参考

| 错误码 | 说明 | 处理建议 |
|--------|------|---------|
| `E001` | 消息格式错误 | 检查 JSON 格式与字段 |
| `E002` | 消息类型不支持 | 检查 message_type 字段 |
| `E003` | 节点未注册 | 等待节点先完成注册流程 |
| `E004` | 设备不存在 | 等待设备列表同步完成 |
| `E005` | 认证失败 | 检查 access_token / 凭证 |
| `E006` | 权限不足 | 检查 ACL 配置 |
| `E007` | 超时 | 重试或增加超时时间 |
| `E008` | 重复消息 | 根据 message_id 去重 |

---

## 10. 安全与认证

### 10.1 MQTT 安全配置

```yaml
mqtt:
  tls_enabled: true
  ca_cert: "/etc/edgeos/certs/ca.crt"
  client_cert: "/etc/edgeos/certs/client.crt"
  client_key: "/etc/edgeos/certs/client.key"
  # ACL: edgex/# 仅 edgeos 用户可订阅
```

### 10.2 NATS 安全配置

```yaml
nats:
  tls_enabled: true
  token: "your-auth-token"
  # 或使用用户名密码
  username: "edgeos"
  password: "secure-password"
```

### 10.3 节点 access_token

节点注册成功后 EdgeOS 返回 `access_token`，后续 EdgeX 发送的消息头中应携带此 token，EdgeOS 在处理消息时校验有效性：

```go
func (h *BaseHandler) ValidateToken(msg *Message) error {
    token := msg.Header.Token
    if token == "" {
        return errors.New("missing access_token")
    }
    return h.authService.ValidateNodeToken(msg.Header.Source, token)
}
```

---

## 11. 测试与排查

### 11.1 本地测试环境启动

```bash
# 启动 MQTT Broker (EMQX)
docker run -d --name emqx -p 1883:1883 -p 18083:18083 emqx/emqx:latest

# 启动 NATS Server（含 JetStream）
docker run -d --name nats -p 4222:4222 -p 8222:8222 nats -js
```

### 11.2 模拟 EdgeX 消息

```bash
# 1. 模拟节点注册
mosquitto_pub -h 127.0.0.1 -p 1883 -t "edgex/nodes/register" -m '{
  "header":{"message_id":"test-reg-001","timestamp":1744680000000,"source":"edgex-test-001","message_type":"node_register","version":"1.0"},
  "body":{"node_id":"edgex-test-001","node_name":"测试网关","model":"edge-gateway","version":"1.0.0","api_version":"v1","capabilities":["shadow-sync","heartbeat"],"protocol":"edgeOS(MQTT)"}
}'

# 2. 模拟设备上报
mosquitto_pub -h 127.0.0.1 -p 1883 -t "edgex/devices/report" -m '{
  "header":{"message_id":"test-dev-001","source":"edgex-test-001","message_type":"device_report","version":"1.0","timestamp":1744680000000},
  "body":{"node_id":"edgex-test-001","devices":[{"device_id":"dev-001","device_name":"Modbus设备","device_profile":"modbus-tcp","admin_state":"ENABLED","operating_state":"ENABLED"}]}
}'

# 3. 模拟点位上报
mosquitto_pub -h 127.0.0.1 -p 1883 -t "edgex/points/report" -m '{
  "header":{"message_id":"test-pt-001","source":"edgex-test-001","message_type":"point_report","version":"1.0","timestamp":1744680000000},
  "body":{"node_id":"edgex-test-001","device_id":"dev-001","points":[{"point_id":"Temperature","point_name":"温度","value_type":"Float32","access_mode":"R","unit":"°C"}]}
}'

# 4. 模拟实时数据
mosquitto_pub -h 127.0.0.1 -p 1883 -t "edgex/data/edgex-test-001/dev-001" -m '{
  "header":{"message_id":"test-data-001","source":"edgex-test-001","message_type":"data","version":"1.0","timestamp":1744680000000},
  "body":{"node_id":"edgex-test-001","device_id":"dev-001","timestamp":1744680000000,"points":{"Temperature":25.5,"Humidity":65.2},"quality":"good"}
}'

# 5. 订阅监听 EdgeOS 下发的控制命令（验证双向控制）
mosquitto_sub -h 127.0.0.1 -p 1883 -t "edgex/cmd/#" -v
```

### 11.3 NATS 测试

```bash
# 订阅所有 edgex 消息
nats sub "edgex.>"

# 发布节点注册
nats pub "edgex.nodes.register" '{"header":{"message_id":"test-001","timestamp":1744680000000,"source":"edgex-test","message_type":"node_register","version":"1.0"},"body":{"node_id":"edgex-test","node_name":"Test Node","protocol":"edgeOS(NATS)"}}'
```

### 11.4 常见排查

| 问题 | 排查步骤 |
|------|---------|
| 连接失败 | 检查 broker/url 地址与端口，检查用户名密码，`telnet host port` |
| 消息未接收 | 检查订阅 Topic 是否正确，检查 QoS，确认消息发布成功 |
| 注册无响应 | 检查响应 Topic 订阅，检查 correlation_id 匹配 |
| 点位不更新 | 确认 `edgex/data/#` 已订阅，检查 data 消息格式 |
| 控制命令无反应 | 检查下行 Topic 拼写，检查 EdgeX 是否在线，查看响应 Topic 日志 |

---

## 12. 项目结构参考

```
internal/
├── middleware/            # 消息总线层
│   ├── manager.go         # 连接管理器
│   ├── mqtt_client.go     # MQTT 客户端
│   ├── nats_client.go     # NATS 客户端
│   ├── message.go         # 消息通用结构
│   └── router.go          # 消息路由器
├── handlers/              # 消息处理器
│   ├── node_handler.go    # 节点注册处理
│   ├── device_handler.go  # 设备同步处理
│   ├── point_handler.go   # 点位同步(即物模型点位上报 可用将实时数据点位理解成物模型)+实时数据
│   ├── control_handler.go # 控制命令响应处理
│   └── event_handler.go   # 告警事件处理
├── services/              # 业务逻辑层
│   ├── node_service.go
│   ├── device_service.go
│   ├── point_service.go
│   ├── control_service.go # 下行控制命令构建与发送
│   └── heartbeat_service.go
├── models/                # 数据模型
│   ├── middleware.go      # 中间件配置模型
│   ├── node.go
│   ├── device.go
│   └── point.go
└── api/                   # REST API 路由
    ├── middleware_api.go  # 中间件 CRUD 接口
    ├── node_api.go
    ├── device_api.go
    ├── point_api.go
    └── control_api.go     # 控制命令接口

config/
├── config.yaml            # 主配置（含中间件默认值）
└── config.dev.yaml

cmd/
└── main.go                # 启动入口，加载配置，初始化连接管理器
```

---

**文档版本**: v2.1
**最后更新**: 2026-04-17  
**维护者**: edgeOS 团队
