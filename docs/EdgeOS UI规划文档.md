---
layout: default
title: EdgeOS UI 规划文档
description: EdgeOS 前端信息架构、页面规划与模块化编排说明。
---

# EdgeOS UI 规划文档

> 本文档按照 EdgeOS 四个核心功能的顺序规划前端界面：
> 1. **消息总线管理**（UI 添加 MQTT/NATS 连接）
> 2. **EdgeX 节点注册**（注册状态监控）
> 3. **EdgeX 子设备列表同步**（设备列表管理）
> 4. **EdgeX 子设备点位同步**（点位查看与实时数据）
> 5. **EdgeX 子设备双向控制**（命令下发与响应追踪）

---

## 目录

1. [设计规范](#1-设计规范)
2. [路由结构](#2-路由结构)
3. [功能一：消息总线管理](#3-功能一消息总线管理)
4. [功能二：EdgeX 节点注册](#4-功能二edgex-节点注册)
5. [功能三：子设备列表同步](#5-功能三子设备列表同步)
6. [功能四：子设备点位同步](#6-功能四子设备点位同步)
7. [功能五：子设备双向控制](#7-功能五子设备双向控制)
8. [实时通信方案](#8-实时通信方案)
9. [全局状态管理](#9-全局状态管理)
10. [类型定义](#10-类型定义)
11. [API 服务层](#11-api-服务层)
12. [组件复用规范](#12-组件复用规范)

---

## 1. 设计规范

### 1.1 视觉风格

**工业仪表盘风格**：信息密度高、状态可视、实时流、告警高亮

| 项目 | 规格 |
|------|------|
| 背景色 | `#0B1220`（主）/ `#111827`（卡片） |
| 主色 | `#1D4ED8` / `#0EA5E9` |
| 文字 | `#E5E7EB`（主）/ `#9CA3AF`（辅） |
| 成功色 | `#10B981` |
| 警告色 | `#F59E0B` |
| 错误色 | `#EF4444` |
| 强调色 | `#6366F1` |
| 字体 | `system-ui` |
| 标题 | `24px / 600` |
| 正文 | `14px / 400` |

### 1.2 状态色标准

| 状态 | 颜色 | 说明 |
|------|------|------|
| `connected` / `online` | `#10B981` | 正常在线 |
| `connecting` | `#F59E0B` | 连接中 |
| `disconnected` / `offline` | `#9CA3AF` | 离线 |
| `error` | `#EF4444` | 异常 |
| `critical` 告警 | `#EF4444` | 红色闪烁 |
| `warning` 告警 | `#F59E0B` | 橙色 |
| `info` | `#6366F1` | 蓝紫色 |

### 1.3 布局规则

- 顶部导航栏固定高度 `64px`，左侧 Sidebar 宽度 `240px`（折叠 `64px`）
- 主内容区 `padding: 24px`，使用 CSS Grid / Flexbox 混合布局
- 每屏卡片数量控制在 4-8 个，避免过度分散
- 实时数据刷新不超过 1s 间隔
- 告警角标在 Sidebar 菜单项右上角实时展示数量

---

## 2. 路由结构

```
/
├── /                           → 重定向到 /dashboard
├── /dashboard                  → 总览仪表盘
├── /middlewares                → 消息总线列表
│   ├── /middlewares/add        → 添加连接（表单页或弹窗）
│   └── /middlewares/:id        → 连接详情与订阅主题状态
├── /nodes                      → EdgeX 节点列表
│   └── /nodes/:nodeId          → 节点详情（设备列表入口）
├── /nodes/:nodeId/devices      → 子设备列表
│   └── /nodes/:nodeId/devices/:deviceId → 设备详情（点位列表入口）
├── /nodes/:nodeId/devices/:deviceId/points → 点位列表与实时数据
├── /control                    → 双向控制面板
│   └── /control/:nodeId/:deviceId → 指定设备控制界面
├── /realtime                   → 全局实时数据流
├── /alerts                     → 告警与事件列表
└── /settings                   → 系统设置
```

### 2.1 导航菜单结构

```
◎ 总览
─ 消息总线     [+]  ← 核心入口，支持 MQTT/NATS 添加
─ EdgeX 节点          ← 注册状态
─ 子设备管理          ← 设备列表同步
─ 点位监控            ← 点位同步 + 实时数据
─ 设备控制       [!]  ← 双向控制（告警角标）
─ 事件告警       [N]  ← 数字角标
─ 系统设置
```

---

## 3. 功能一：消息总线管理

### 3.1 页面：消息总线列表（`/middlewares`）

**职责：** 展示所有已配置的 MQTT/NATS 连接，支持新增、编辑、删除、连接/断开操作。

```
┌─────────────────────────────────────────────────────────────┐
│  消息总线管理                          [+ 添加连接]        │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ 生产 MQTT     │  │ 测试 NATS    │  │ 备用 MQTT    │      │
│  │ edgeOS(MQTT) │  │ edgeOS(NATS) │  │ edgeOS(MQTT) │      │
│  │ ● 已连接      │  │ ● 已连接    │  │ ○ 断开       │      │
│  │ 127.0.0.1    │  │ 127.0.0.1   │  │ 10.0.0.1     │      │
│  │ :1883        │  │ :4222       │  │ :1883        │      │
│  │ 订阅: 12个主题│  │ 订阅: 10个  │  │ 订阅: 0个    │      │
│  │ [详情][断开]  │  │ [详情][断开] │  │ [连接][删除] │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

**组件：** `MiddlewareCard.vue`

```vue
<!-- src/components/middleware/MiddlewareCard.vue -->
<template>
  <div class="bg-gray-800 rounded-xl p-5 border border-gray-700 hover:border-blue-500
              transition-all cursor-pointer" @click="$emit('detail', config.id)">
    <div class="flex items-center justify-between mb-3">
      <div class="flex items-center gap-2">
        <Server class="w-5 h-5 text-blue-400" />
        <span class="font-medium text-gray-100">{{ config.name }}</span>
      </div>
      <StatusBadge :status="config.status" />
    </div>
    <div class="text-sm text-gray-400 mb-1">{{ config.type }}</div>
    <div class="text-sm text-gray-300 mb-3">{{ brokerAddress }}</div>
    <div class="text-xs text-gray-500 mb-4">
      已订阅 {{ subscriptionCount }} 个主题
    </div>
    <div class="flex gap-2">
      <button v-if="config.status !== 'connected'" @click.stop="$emit('connect', config.id)"
        class="flex-1 py-1.5 text-xs bg-blue-600 hover:bg-blue-500 rounded-lg text-white transition-colors">
        连接
      </button>
      <button v-else @click.stop="$emit('disconnect', config.id)"
        class="flex-1 py-1.5 text-xs bg-gray-700 hover:bg-gray-600 rounded-lg text-gray-300 transition-colors">
        断开
      </button>
      <button @click.stop="$emit('delete', config.id)"
        class="px-3 py-1.5 text-xs bg-red-900/40 hover:bg-red-900/60 rounded-lg text-red-400 transition-colors">
        删除
      </button>
    </div>
  </div>
</template>
```

### 3.2 添加连接表单

**职责：** 用户通过表单添加 MQTT 或 NATS 连接配置，填写完成后点击「测试连接」验证，再「保存并连接」。

```
┌────────────────────────────────────────────────────┐
│  添加消息总线                              [×]    │
├────────────────────────────────────────────────────┤
│  连接名称   [___________________________]           │
│  协议类型   ○ edgeOS(MQTT)  ○ edgeOS(NATS)         │
│                                                    │
│  ── MQTT 配置 ─────────────────────────────────── │
│  Broker 地址  [tcp://127.0.0.1     ] 端口 [1883]  │
│  Client ID    [edgeos-queen-001    ]              │
│  用户名       [__________________  ]              │
│  密码         [••••••••••••••••••  ]              │
│  QoS 级别     [1 ▼]  Keep Alive [60] 秒          │
│  □ 自动重连   □ 清理会话   □ 启用 TLS             │
│                                                    │
│  ──────────────────────────────────────────────── │
│  [测试连接]              [取消]  [保存并连接]       │
└────────────────────────────────────────────────────┘
```

**NATS 配置表单字段：**

| 字段 | 输入类型 | 默认值 |
|------|---------|-------|
| URL | text | `nats://127.0.0.1:4222` |
| Client Name | text | `edgeos-queen` |
| 用户名 | text | - |
| 密码 | password | - |
| Token | password | - |
| Reconnect Wait | number | `2` 秒 |
| Max Reconnects | number | `10` |
| 启用 JetStream | checkbox | true |
| 启用 TLS | checkbox | false |

**组件：** `AddMiddlewareModal.vue`

```vue
<!-- src/components/middleware/AddMiddlewareModal.vue -->
<template>
  <div class="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
    <div class="bg-gray-900 rounded-2xl w-[560px] border border-gray-700 shadow-2xl">
      <div class="flex items-center justify-between p-6 border-b border-gray-700">
        <h2 class="text-xl font-semibold text-gray-100">添加消息总线</h2>
        <button @click="$emit('close')" class="text-gray-400 hover:text-gray-200">
          <X class="w-5 h-5" />
        </button>
      </div>
      <div class="p-6 space-y-4 max-h-[70vh] overflow-y-auto">
        <!-- 基础信息 -->
        <FormField label="连接名称" required>
          <input v-model="form.name" type="text" class="form-input" placeholder="如：生产环境MQTT" />
        </FormField>
        <!-- 协议类型 -->
        <FormField label="协议类型" required>
          <div class="flex gap-4">
            <label v-for="t in middlewareTypes" :key="t.value"
              class="flex items-center gap-2 cursor-pointer">
              <input type="radio" v-model="form.type" :value="t.value"
                class="accent-blue-500" />
              <span class="text-gray-300 text-sm">{{ t.label }}</span>
            </label>
          </div>
        </FormField>
        <!-- 动态表单 -->
        <MQTTConfigForm v-if="form.type === 'edgeOS(MQTT)'" v-model="form.mqtt" />
        <NATSConfigForm v-else v-model="form.nats" />
      </div>
      <div class="flex justify-between items-center p-6 border-t border-gray-700">
        <button @click="testConnection" :disabled="testing"
          class="flex items-center gap-2 px-4 py-2 text-sm bg-gray-700 hover:bg-gray-600
                 rounded-lg text-gray-200 transition-colors">
          <Zap class="w-4 h-4" />
          {{ testing ? '测试中...' : '测试连接' }}
        </button>
        <div class="flex gap-3">
          <button @click="$emit('close')" class="btn-secondary">取消</button>
          <button @click="submit" :disabled="submitting" class="btn-primary">
            保存并连接
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
```

### 3.3 连接详情页（`/middlewares/:id`）

展示该连接的详细信息及已订阅的主题状态：

```
┌────────────────────────────────────────────────────────────┐
│  生产 MQTT · 已连接 ●                   [断开] [编辑]       │
├─────────────────────────┬──────────────────────────────────┤
│ 连接信息                │  已订阅主题列表                   │
│ 地址: tcp://127.0.0.1   │  edgex/nodes/register      ● 活跃 │
│ 端口: 1883              │  edgex/nodes/unregister    ● 活跃 │
│ ClientID: edgeos-001    │  edgex/devices/report      ● 活跃 │
│ QoS: 1                 │  edgex/points/report       ● 活跃 │
│                         │  edgex/data/#              ● 活跃 │
│ 统计                    │  edgex/nodes/+/heartbeat   ● 活跃 │
│ 接收消息: 1,243         │  edgex/nodes/+/status      ● 活跃 │
│ 发送消息: 87            │  edgex/events/alert        ● 活跃 │
│ 重连次数: 0             │  edgex/responses/#         ● 活跃 │
└─────────────────────────┴──────────────────────────────────┘
```

---

## 4. 功能二：EdgeX 节点注册

### 4.1 页面：节点列表（`/nodes`）

**职责：** 展示所有通过消息中间件注册的 EdgeX 节点，包含在线状态、能力、上次心跳时间。

```
┌──────────────────────────────────────────────────────────────────┐
│  EdgeX 节点  (共 3 个 · 在线 2 · 离线 1)      [刷新] [触发发现]  │
├──────────────────────────────────────────────────────────────────┤
│  节点ID            名称            协议           状态  最后心跳  │
│  ─────────────────────────────────────────────────────────────── │
│  edgex-node-001   EdgeX Gateway   edgeOS(MQTT)  ● 在线  3s前    │
│  edgex-node-002   EdgeX Gateway2  edgeOS(NATS)  ● 在线  8s前    │
│  edgex-node-003   老版网关         edgeOS(MQTT)  ○ 离线  5m前    │
│                                                                  │
│  点击行 → 查看节点详情 / 子设备列表                               │
└──────────────────────────────────────────────────────────────────┘
```

**节点注册事件实时通知：** 当新节点通过 MQTT/NATS 发送 `node_register` 消息时，页面顶部弹出 Toast 通知：

```
┌──────────────────────────────────────┐
│ ● 新节点已注册: edgex-node-004       │
│   EdgeX Gateway Node · edgeOS(MQTT)  │
│                            [查看]    │
└──────────────────────────────────────┘
```

### 4.2 页面：节点详情（`/nodes/:nodeId`）

```
┌────────────────────────────────────────────────────────────┐
│  ← 节点列表  /  edgex-node-001                             │
├────────────────┬───────────────────────────────────────────┤
│  基本信息       │  资源监控                                  │
│  ID: edgex-001 │  CPU: ████░░░░ 25%                        │
│  名称: Gateway  │  内存: ██████░░ 512MB                    │
│  协议: MQTT     │  磁盘: ███░░░░░ 45%                       │
│  版本: 1.0.0   │  活跃设备: 10 / 活跃任务: 5               │
│  能力:          │                                           │
│  shadow-sync   │  心跳序列: #100  在线时长: 1h              │
│  heartbeat      │                                           │
│  device-control │  ─────────────────────────────────────── │
│  task-execution │  子设备 (10)    [同步设备列表]  [发现设备] │
├────────────────┴───────────────────────────────────────────┤
│  子设备列表快览（最多展示5个，[查看全部]跳转设备列表）         │
└────────────────────────────────────────────────────────────┘
```

### 4.3 节点状态流转

```
          发送 node_register
              ↓
         [注册中] → 注册成功 → [在线]
                              ↓ 心跳超时 (3分钟无心跳)
                           [超时] → 自动标记 [离线]
                              ↓ 发送 node_unregister
                           [已注销]
```

**组件：** `NodeStatusBadge.vue` — 展示带动画脉冲的在线状态点。

---

## 5. 功能三：子设备列表同步

### 5.1 页面：子设备列表（`/nodes/:nodeId/devices`）

**职责：** 展示指定节点下所有已同步的子设备，支持主动触发设备发现与手动同步。

```
┌─────────────────────────────────────────────────────────────────┐
│  ← edgex-node-001  /  子设备列表 (共 10 台)                      │
│                                [触发设备发现] [请求同步] [刷新]   │
├───────┬──────────────┬───────────┬──────┬───────────┬──────────┤
│ 设备ID │ 设备名称      │ 配置文件  │ 状态  │ 最后更新   │  操作  │
├───────┼──────────────┼───────────┼──────┼───────────┼──────────┤
│dev-001│ Modbus TCP   │modbus-tcp │● 在线 │ 1s前      │[点位][控制]│
│dev-002│ OPC-UA设备   │opc-ua     │● 在线 │ 2s前      │[点位][控制]│
│dev-003│ BACnet设备   │bacnet     │○ 离线 │ 5m前      │[点位][控制]│
└───────┴──────────────┴───────────┴──────┴───────────┴──────────┘
```

**触发设备发现**弹窗：

```
┌───────────────────────────────────┐
│  触发设备发现                 [×]  │
│                                   │
│  目标节点: edgex-node-001         │
│  协议类型: [modbus-tcp      ▼]    │
│  网络范围: [192.168.1.0/24  ]     │
│  超时时间: [30] 秒                │
│  □ 自动注册发现的设备              │
│  □ 立即同步                       │
│                                   │
│                [取消] [发起发现]   │
└───────────────────────────────────┘
```

### 5.2 设备同步实时反馈

收到 `edgex/devices/report` 消息时，设备列表自动刷新，新增设备闪烁高亮 2 秒：

```vue
<!-- src/views/DeviceListView.vue 关键逻辑 -->
<script setup lang="ts">
import { useDeviceStore } from '@/stores/device'
import { useRealtimeStore } from '@/stores/realtime'

const deviceStore = useDeviceStore()
const realtime = useRealtimeStore()
const newDeviceIds = ref<Set<string>>(new Set())

// 监听实时事件：设备同步
realtime.on('device_synced', (device: Device) => {
  deviceStore.upsertDevice(device)
  newDeviceIds.value.add(device.device_id)
  setTimeout(() => newDeviceIds.value.delete(device.device_id), 2000)
})
</script>
```

---

## 6. 功能四：子设备点位同步（物模型）

### 6.1 数据加载策略：物模型驱动

点位列表页的数据来源分为两种，前端需同时处理：

| 来源 | 触发时机 | 内容 | 前端处理 |
|------|---------|------|---------|
| **REST API** | 页面首次加载 | 从后端获取当前存储的全量点位（含 `current_value`） | 初始化点位列表展示 |
| **WebSocket `data_update` 事件** | 后续实时推送 | `is_full_snapshot=true` 全量 / `false` 差量 | Merge 到 Pinia store，组件响应式刷新 |

**差量 Merge 原则**：收到 `data_update` 事件后，**只更新 payload 中出现的 point_id**，未出现的点位保留上一次的值，不清空也不置为空。

### 6.2 页面：点位列表（`/nodes/:nodeId/devices/:deviceId/points`）

**职责：** 展示设备物模型（全量点位 + 实时值），区分只读（R）和可写（W/RW）点位，实时更新差量变化。

```
┌──────────────────────────────────────────────────────────────────────────┐
│  ← Room_FC_2014_19 (风机盘管)  /  物模型点位 (共 9 个)                    │
│                                         [触发全量同步] [创建采集任务]      │
├──────────────────┬──────────┬──────┬───────┬──────────────┬───────┬──────┤
│ 点位ID            │ 点位名称  │ 类型  │ 访问  │ 当前值        │ 单位  │ 操作  │
├──────────────────┼──────────┼──────┼───────┼──────────────┼───────┼──────┤
│ SetPoint.Value   │ 设定温度  │Float │ RW    │ 20 ●         │ °C    │[写入]│
│ Setpoint.1       │ 设定点1   │Float │ RW    │ 20 ●         │ °C    │[写入]│
│ Setpoint.2       │ 设定点2   │Float │ RW    │ 20 ●         │ °C    │[写入]│
│ Setpoint.3       │ 设定点3   │Float │ RW    │ 19           │ °C    │[写入]│
│ State.Chiller    │ 制冷状态  │Int   │ R     │ 0            │ -     │[图表]│
│ State.Heater     │ 制热状态  │Int   │ R     │ 1            │ -     │[图表]│
│ Temperature.Indoor│室内温度  │Float │ R     │ 18.8         │ °C    │[图表]│
│ Temperature.Outdoor│室外温度 │Float │ R     │ 12           │ °C    │[图表]│
│ Temperature.Water│水管温度   │Float │ R     │ 39.7         │ °C    │[图表]│
└──────────────────┴──────────┴──────┴───────┴──────────────┴───────┴──────┘

 ● 绿点 = 本轮差量更新的点位（闪烁 1.5s）  无点 = 自上次全量后未变化
 数据质量 Good/Uncertain/Bad 用行底色区分
```

> **「触发全量同步」**按钮：向后端发送 POST，后端向 EdgeX 发布 `edgex/cmd/{node_id}/sync`，触发 EdgeX 重新全量上报物模型数据。

### 6.3 点位实时卡片视图（切换）

用户可在表格视图与卡片视图之间切换：

```
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│  SetPoint.Value  │  │  Temperature.Indoor│  │  State.Heater   │
│  设定温度        │  │  室内温度          │  │  制热状态         │
│                  │  │                  │  │                  │
│   20  °C  ●      │  │   18.8  °C       │  │   1 (开启)        │
│  ████░░░░ 20/30  │  │  ████░░░░ 18.8   │  │                  │
│  范围: 16~30     │  │  R · Float32     │  │  R · Int         │
│  RW · Float32    │  │  3s前更新        │  │  未变化            │
│  [立即写入]       │  │  [图表]           │  │  [图表]           │
└──────────────────┘  └──────────────────┘  └──────────────────┘
```

### 6.4 实时差量更新的前端 Store

```typescript
// src/stores/realtime.ts

import { defineStore } from 'pinia'
import type { DataUpdatePayload } from '@/types/realtime'

export const useRealtimeStore = defineStore('realtime', {
  state: () => ({
    // key = `${nodeId}/${deviceId}` → 全量物模型缓存（初始化后持续 Merge）
    deviceSnapshot: {} as Record<string, Record<string, {
      value: unknown
      ts: number
      quality: string
      changed: boolean  // 标记本轮差量中是否刚更新（用于 UI 高亮）
    }>>,
    alertCount: 0,
  }),
  actions: {
    // 处理 data_update WebSocket 事件
    updateData(payload: DataUpdatePayload) {
      const key = `${payload.node_id}/${payload.device_id}`

      if (payload.is_full_snapshot) {
        // 全量：重建整个设备快照
        const snap: typeof this.deviceSnapshot[string] = {}
        for (const [pointId, value] of Object.entries(payload.points)) {
          snap[pointId] = { value, ts: payload.timestamp, quality: payload.quality, changed: true }
        }
        this.deviceSnapshot[key] = snap
      } else {
        // 差量：仅 Merge 出现的点位，不删除未出现的
        if (!this.deviceSnapshot[key]) this.deviceSnapshot[key] = {}
        // 先将上一轮的 changed 标记清除
        for (const p of Object.values(this.deviceSnapshot[key])) p.changed = false
        for (const [pointId, value] of Object.entries(payload.points)) {
          this.deviceSnapshot[key][pointId] = {
            value,
            ts: payload.timestamp,
            quality: payload.quality,
            changed: true,  // 本轮差量更新，UI 高亮 1.5s
          }
        }
      }
    },

    getPointValue(nodeId: string, deviceId: string, pointId: string) {
      return this.deviceSnapshot[`${nodeId}/${deviceId}`]?.[pointId]
    },

    getDeviceSnapshot(nodeId: string, deviceId: string) {
      return this.deviceSnapshot[`${nodeId}/${deviceId}`] ?? {}
    },

    incrementAlertCount() { this.alertCount++ },
    clearAlertCount() { this.alertCount = 0 },
  },
})
```

### 6.5 点位行差量高亮逻辑

```vue
<!-- src/components/point/PointTable.vue 关键片段 -->
<script setup lang="ts">
import { useRealtimeStore } from '@/stores/realtime'

const realtimeStore = useRealtimeStore()

// 订阅 data_update 事件（由 useWebSocket composable 注入）
realtime.on('data_update', (payload: DataUpdatePayload) => {
  realtimeStore.updateData(payload)
})

// 点位行是否本轮刚更新（差量高亮）
const isChanged = (pointId: string) =>
  realtimeStore.getPointValue(props.nodeId, props.deviceId, pointId)?.changed ?? false
</script>

<template>
  <tr v-for="point in points" :key="point.point_id"
    :class="isChanged(point.point_id)
      ? 'bg-blue-900/30 transition-colors duration-1500'
      : 'bg-transparent'">
    <td>{{ point.point_id }}</td>
    <td>{{ point.point_name }}</td>
    <td>{{ point.value_type }}</td>
    <td>{{ point.access_mode }}</td>
    <td class="font-mono">
      {{ realtimeStore.getPointValue(props.nodeId, props.deviceId, point.point_id)?.value ?? point.current_value }}
      <span v-if="isChanged(point.point_id)"
        class="inline-block w-2 h-2 rounded-full bg-green-400 animate-pulse ml-1" />
    </td>
    <td>{{ point.unit }}</td>
    <td>
      <button v-if="point.access_mode?.includes('W')" @click="openWriteModal(point)">写入</button>
      <button v-else @click="openChart(point)">图表</button>
    </td>
  </tr>
</template>
```

### 6.6 点位历史图表

```
┌──────────────────────────────────────────────────┐
│  Temperature.Indoor 历史曲线  最近 1 小时    [×]  │
│                                                  │
│  20 ─────────────────────────────────            │
│  19 ────╮    ╭────────────╮                      │
│  18 ────╯╮  ╭╯            ╰──────────────        │
│  17 ─────╰──╯                                    │
│       T1    T2    T3    T4    T5                  │
│  当前: 18.8°C  最高: 19.5°C  最低: 17.2°C         │
└──────────────────────────────────────────────────┘
```

### 6.7 创建采集任务

```
┌───────────────────────────────────────────────────┐
│  创建采集任务                                 [×]  │
│                                                   │
│  任务名称: [Room_FC_2014_19 采集             ]     │
│  目标设备: Room_FC_2014_19                        │
│  采集点位: ☑ SetPoint.Value  ☑ Temperature.Indoor │
│            □ State.Chiller   □ Temperature.Water  │
│  采集方式: ● 定时间隔  ○ 变化触发                  │
│  间隔时间: [5] 秒                                  │
│  批量大小: [10]  最大重试: [3]                     │
│                                                   │
│                          [取消] [创建并启动]        │
└───────────────────────────────────────────────────┘
```

---

## 7. 功能五：子设备双向控制

### 7.1 页面：控制面板（`/control` 或 `/control/:nodeId/:deviceId`）

**职责：** 提供统一的命令下发界面，支持写入点位值、任务控制，并实时展示命令执行结果。

```
┌─────────────────────────────────────────────────────────────────────┐
│  设备控制面板                                                         │
├────────────────────────┬────────────────────────────────────────────┤
│ 选择目标                │  控制操作                                   │
│                        │                                            │
│ 节点: [edgex-node-001▼]│  ── 写入点位 ───────────────────────────── │
│ 设备: [dev-001       ▼]│  点位名称: [Switch              ▼]         │
│                        │  写入值:   [true                ]           │
│ 设备状态: ● 在线        │                        [写入并等待响应]       │
│                        │                                            │
│ 可写点位 (3/12):        │  ── 任务控制 ──────────────────────────── │
│ ● Switch   RW  Bool    │  任务 task-001 [Temperature Coll...]       │
│ ● Setpoint W   Float   │  状态: ● 运行中                            │
│ ● Relay    W   Bool    │  [暂停]  [恢复]  [停止]                    │
│                        │                                            │
│ [←] [→] 分页           │  ── 配置更新 ───────────────────────────── │
│                        │                    [更新节点配置]           │
└────────────────────────┴────────────────────────────────────────────┘
```

### 7.2 命令执行状态追踪

每次下发控制命令后，在页面右侧展示「命令执行记录」面板：

```
┌─────────────────────────────────────────────────┐
│  命令执行记录                            [清空]  │
│                                                 │
│  14:05:32  写入命令  Switch=true                 │
│           → 发送中... ⟳                         │
│                                                 │
│  14:05:28  写入命令  Setpoint=80.5              │
│           ✓ 执行成功  延迟: 145ms               │
│                                                 │
│  14:05:10  写入命令  Switch=false               │
│           ✗ 执行失败: Connection timeout        │
│             [重试]                              │
│                                                 │
│  14:04:55  任务控制  task-001 暂停              │
│           ✓ 执行成功  延迟: 89ms                │
└─────────────────────────────────────────────────┘
```

**组件：** `CommandLogPanel.vue`

```vue
<!-- src/components/control/CommandLogPanel.vue -->
<template>
  <div class="bg-gray-900 rounded-xl border border-gray-700 h-full flex flex-col">
    <div class="flex items-center justify-between p-4 border-b border-gray-700">
      <span class="font-medium text-gray-200">命令执行记录</span>
      <button @click="commandStore.clearLogs()" class="text-xs text-gray-500 hover:text-gray-300">
        清空
      </button>
    </div>
    <div class="flex-1 overflow-y-auto p-3 space-y-3">
      <div v-for="log in commandStore.logs" :key="log.id"
        class="p-3 rounded-lg border text-sm"
        :class="{
          'border-yellow-700/50 bg-yellow-900/20': log.status === 'pending',
          'border-green-700/50 bg-green-900/20': log.status === 'success',
          'border-red-700/50 bg-red-900/20': log.status === 'error',
        }">
        <div class="flex items-center justify-between mb-1">
          <span class="text-gray-400 text-xs">{{ formatTime(log.timestamp) }}</span>
          <span class="text-xs font-medium"
            :class="{
              'text-yellow-400': log.status === 'pending',
              'text-green-400': log.status === 'success',
              'text-red-400': log.status === 'error',
            }">
            {{ statusLabel[log.status] }}
          </span>
        </div>
        <div class="text-gray-200">{{ log.command }} → {{ log.target }}</div>
        <div v-if="log.status === 'success'" class="text-xs text-gray-500 mt-1">
          延迟: {{ log.latencyMs }}ms
        </div>
        <div v-if="log.status === 'error'" class="text-xs text-red-400 mt-1">
          {{ log.errorMessage }}
          <button @click="retryCommand(log)" class="ml-2 underline hover:no-underline">重试</button>
        </div>
      </div>
    </div>
  </div>
</template>
```

### 7.3 写入点位弹窗（快速控制）

在点位列表页点击「写入」按钮触发：

```
┌──────────────────────────────────┐
│  写入点位                   [×]  │
│                                  │
│  设备: dev-001 (Modbus TCP)      │
│  点位: Switch (Bool · RW)        │
│                                  │
│  写入值:  ●——○  false → true     │
│                                  │
│  □ 等待执行确认 (超时 10 秒)      │
│                                  │
│              [取消] [确认写入]    │
└──────────────────────────────────┘
```

对于数值型点位（Float/Int），展示数字输入框并显示范围限制：

```
┌──────────────────────────────────┐
│  写入点位                   [×]  │
│                                  │
│  设备: dev-001                   │
│  点位: Setpoint (Float32 · W)    │
│  范围: 0 ~ 100  单位: °C         │
│                                  │
│  写入值: [80.5               ]   │
│                                  │
│              [取消] [确认写入]    │
└──────────────────────────────────┘
```

---

## 8. 实时通信方案

### 8.1 架构

```
EdgeOS 后端 (Go)
     │
     │  WebSocket / SSE  (HTTP /api/v1/ws  或  /api/v1/sse)
     ↓
EdgeOS 前端 (Vue)
     │
     ↓ 事件分发
  useRealtimeStore (Pinia) → 各页面组件响应式更新
```

### 8.2 事件类型定义

```typescript
// src/types/realtime.ts

export type RealtimeEventType =
  | 'node_registered'       // 新节点注册
  | 'node_offline'          // 节点离线
  | 'node_heartbeat'        // 心跳更新
  | 'device_synced'         // 设备列表同步
  | 'point_synced'          // 点位元数据同步
  | 'data_update'           // 实时数据更新
  | 'command_response'      // 命令执行响应
  | 'alert'                 // 告警事件
  | 'middleware_status'     // 消息总线状态变化

export interface RealtimeEvent<T = unknown> {
  type: RealtimeEventType
  timestamp: number
  payload: T
}

export interface DataUpdatePayload {
  node_id: string
  device_id: string
  points: Record<string, number | boolean | string>  // 全量或差量点位
  timestamp: number
  quality: 'Good' | 'Bad' | 'Uncertain'
  is_full_snapshot: boolean  // true=物模型首次全量上报，false=差量 Merge
}

export interface CommandResponsePayload {
  request_id: string
  node_id: string
  device_id: string
  success: boolean
  message: string
  latency_ms: number
}

export interface AlertPayload {
  alert_id: string
  node_id: string
  device_id: string
  alert_type: string
  severity: 'info' | 'warning' | 'error' | 'critical'
  message: string
  timestamp: number
}
```

### 8.3 WebSocket Composable

```typescript
// src/composables/useWebSocket.ts

import { ref, onMounted, onUnmounted } from 'vue'
import type { RealtimeEvent } from '@/types/realtime'

export function useWebSocket() {
  const ws = ref<WebSocket | null>(null)
  const connected = ref(false)
  const handlers = new Map<string, Set<(payload: unknown) => void>>()

  function connect() {
    ws.value = new WebSocket(`ws://${location.host}/api/v1/ws`)
    ws.value.onopen = () => { connected.value = true }
    ws.value.onclose = () => {
      connected.value = false
      setTimeout(connect, 3000) // 自动重连
    }
    ws.value.onmessage = (event) => {
      const evt: RealtimeEvent = JSON.parse(event.data)
      const set = handlers.get(evt.type)
      if (set) set.forEach(fn => fn(evt.payload))
    }
  }

  function on(type: string, fn: (payload: unknown) => void) {
    if (!handlers.has(type)) handlers.set(type, new Set())
    handlers.get(type)!.add(fn)
    return () => handlers.get(type)?.delete(fn)
  }

  onMounted(connect)
  onUnmounted(() => ws.value?.close())

  return { connected, on }
}
```

---

## 9. 全局状态管理

### 9.1 中间件 Store

```typescript
// src/stores/middleware.ts

import { defineStore } from 'pinia'
import type { MiddlewareConfig } from '@/types/middleware'
import * as middlewareApi from '@/api/middleware'

export const useMiddlewareStore = defineStore('middleware', {
  state: () => ({
    list: [] as MiddlewareConfig[],
    loading: false,
  }),
  getters: {
    connectedCount: (s) => s.list.filter(m => m.status === 'connected').length,
    getById: (s) => (id: string) => s.list.find(m => m.id === id),
  },
  actions: {
    async fetchAll() {
      this.loading = true
      this.list = await middlewareApi.getAll()
      this.loading = false
    },
    async addMiddleware(cfg: Omit<MiddlewareConfig, 'id' | 'status'>) {
      const result = await middlewareApi.create(cfg)
      this.list.push(result)
      return result
    },
    async connect(id: string) {
      const cfg = await middlewareApi.connect(id)
      this.upsert(cfg)
    },
    async disconnect(id: string) {
      const cfg = await middlewareApi.disconnect(id)
      this.upsert(cfg)
    },
    async remove(id: string) {
      await middlewareApi.remove(id)
      this.list = this.list.filter(m => m.id !== id)
    },
    upsert(cfg: MiddlewareConfig) {
      const idx = this.list.findIndex(m => m.id === cfg.id)
      if (idx >= 0) this.list[idx] = cfg
      else this.list.push(cfg)
    },
    updateStatus(id: string, status: MiddlewareConfig['status']) {
      const m = this.list.find(m => m.id === id)
      if (m) m.status = status
    },
  },
})
```

### 9.2 节点 Store

```typescript
// src/stores/node.ts

import { defineStore } from 'pinia'
import type { Node } from '@/types/node'
import * as nodeApi from '@/api/node'

export const useNodeStore = defineStore('node', {
  state: () => ({
    list: [] as Node[],
    loading: false,
  }),
  getters: {
    onlineNodes: (s) => s.list.filter(n => n.status === 'online'),
    offlineNodes: (s) => s.list.filter(n => n.status === 'offline'),
  },
  actions: {
    async fetchAll() {
      this.loading = true
      this.list = await nodeApi.getAll()
      this.loading = false
    },
    upsertNode(node: Node) {
      const idx = this.list.findIndex(n => n.id === node.id)
      if (idx >= 0) this.list[idx] = { ...this.list[idx], ...node }
      else this.list.push(node)
    },
    markOffline(nodeId: string) {
      const node = this.list.find(n => n.id === nodeId)
      if (node) node.status = 'offline'
    },
    updateHeartbeat(nodeId: string, metrics: Node['metrics']) {
      const node = this.list.find(n => n.id === nodeId)
      if (node) {
        node.last_heartbeat = Date.now()
        node.metrics = metrics
        node.status = 'online'
      }
    },
  },
})
```

### 9.3 点位实时数据 Store

> 完整实现见 **§ 6.4 实时差量更新的前端 Store**，核心策略如下：
> - `is_full_snapshot=true`：重建整个设备的物模型快照（替换）
> - `is_full_snapshot=false`：仅 Merge 本次 payload 中出现的 point_id，未出现的点位保留上次值
> - 每个点位条目附带 `changed: boolean` 标记，供 UI 差量高亮动画使用（1.5s 后清除）

```typescript
// src/stores/realtime.ts  ← 完整代码见 §6.4
// 关键接口摘要：

realtimeStore.updateData(payload: DataUpdatePayload)  // 全量或差量 Merge
realtimeStore.getPointValue(nodeId, deviceId, pointId) // 获取点位最新值+changed标记
realtimeStore.getDeviceSnapshot(nodeId, deviceId)      // 获取设备物模型完整快照
```

### 9.4 命令 Store

```typescript
// src/stores/command.ts

import { defineStore } from 'pinia'
import type { CommandLog } from '@/types/command'
import * as controlApi from '@/api/control'

export const useCommandStore = defineStore('command', {
  state: () => ({
    logs: [] as CommandLog[],
  }),
  actions: {
    async writePoints(nodeId: string, deviceId: string, points: Record<string, unknown>) {
      const log: CommandLog = {
        id: crypto.randomUUID(),
        type: 'write',
        nodeId, deviceId,
        command: `写入: ${JSON.stringify(points)}`,
        target: `${nodeId}/${deviceId}`,
        status: 'pending',
        timestamp: Date.now(),
      }
      this.logs.unshift(log)

      const result = await controlApi.writePoints(nodeId, deviceId, points)
      const idx = this.logs.findIndex(l => l.id === log.id)
      if (idx >= 0) {
        this.logs[idx] = {
          ...this.logs[idx],
          status: result.success ? 'success' : 'error',
          latencyMs: result.latency_ms,
          errorMessage: result.message,
        }
      }
      return result
    },
    clearLogs() { this.logs = [] },
  },
})
```

---

## 10. 类型定义

```typescript
// src/types/middleware.ts
export interface MiddlewareConfig {
  id: string
  name: string
  type: 'edgeOS(MQTT)' | 'edgeOS(NATS)'
  description?: string
  mqtt?: MQTTConfig
  nats?: NATSConfig
  status: 'connected' | 'disconnected' | 'connecting' | 'error'
  created_at: number
  updated_at: number
}

export interface MQTTConfig {
  broker: string
  client_id: string
  username: string
  password: string
  qos: 0 | 1 | 2
  clean_session: boolean
  keep_alive: number
  connect_timeout: number
  auto_reconnect: boolean
  tls_enabled: boolean
}

export interface NATSConfig {
  url: string
  client_name: string
  username: string
  password: string
  token?: string
  reconnect_wait: number
  max_reconnects: number
  jetstream_enabled: boolean
  tls_enabled: boolean
}

// src/types/node.ts
export interface Node {
  id: string
  name: string
  model: string
  version: string
  capabilities: string[]
  protocol: string
  endpoint: { host: string; port: number }
  metadata: Record<string, string>
  status: 'online' | 'offline' | 'error'
  metrics?: NodeMetrics
  registered_at: number
  last_heartbeat: number
}

export interface NodeMetrics {
  cpu_usage: number
  memory_usage: number
  disk_usage: number
  active_devices: number
  active_tasks: number
}

// src/types/device.ts
export interface Device {
  node_id: string
  device_id: string
  device_name: string
  device_profile: string
  service_name: string
  labels: string[]
  description: string
  admin_state: 'ENABLED' | 'DISABLED'
  operating_state: 'ENABLED' | 'DISABLED'
  properties: Record<string, unknown>
  status: 'online' | 'offline'
  created_at: number
  updated_at: number
}

// src/types/point.ts
export interface Point {
  node_id: string
  device_id: string
  point_id: string
  point_name: string
  resource_name: string
  value_type: string
  access_mode: 'R' | 'W' | 'RW'
  unit: string
  minimum?: number
  maximum?: number
  address: string
  data_type: string
  scale: number
  offset: number
  current_value?: unknown
  quality?: 'good' | 'bad' | 'uncertain'
  last_updated?: number
}

// src/types/command.ts
export interface CommandLog {
  id: string
  type: 'write' | 'task_control' | 'discover' | 'config'
  nodeId: string
  deviceId: string
  command: string
  target: string
  status: 'pending' | 'success' | 'error'
  timestamp: number
  latencyMs?: number
  errorMessage?: string
}
```

---

## 11. API 服务层

```typescript
// src/api/middleware.ts
import { http } from './http'
import type { MiddlewareConfig } from '@/types/middleware'

const BASE = '/api/v1/middlewares'
export const getAll = () => http.get<MiddlewareConfig[]>(BASE)
export const getById = (id: string) => http.get<MiddlewareConfig>(`${BASE}/${id}`)
export const create = (data: Partial<MiddlewareConfig>) => http.post<MiddlewareConfig>(BASE, data)
export const update = (id: string, data: Partial<MiddlewareConfig>) => http.put<MiddlewareConfig>(`${BASE}/${id}`, data)
export const remove = (id: string) => http.delete(`${BASE}/${id}`)
export const connect = (id: string) => http.post<MiddlewareConfig>(`${BASE}/${id}/connect`)
export const disconnect = (id: string) => http.post<MiddlewareConfig>(`${BASE}/${id}/disconnect`)
export const testConnection = (data: Partial<MiddlewareConfig>) => http.post<{ success: boolean; message: string }>(`${BASE}/test`, data)

// src/api/node.ts
const NODE_BASE = '/api/v1/nodes'
export const getAll = () => http.get<Node[]>(NODE_BASE)
export const getById = (id: string) => http.get<Node>(`${NODE_BASE}/${id}`)
export const triggerDiscover = (nodeId: string, opts: object) => http.post(`${NODE_BASE}/${nodeId}/discover`, opts)

// src/api/device.ts
export const list = (nodeId: string) => http.get<Device[]>(`/api/v1/nodes/${nodeId}/devices`)
export const getDetail = (nodeId: string, deviceId: string) => http.get<Device>(`/api/v1/nodes/${nodeId}/devices/${deviceId}`)
export const triggerSync = (nodeId: string) => http.post(`/api/v1/nodes/${nodeId}/devices/sync`)

// src/api/point.ts
export const list = (nodeId: string, deviceId: string) => http.get<Point[]>(`/api/v1/nodes/${nodeId}/devices/${deviceId}/points`)
export const syncPoints = (nodeId: string, deviceId: string) => http.post(`/api/v1/nodes/${nodeId}/devices/${deviceId}/points/sync`)
export const createTask = (nodeId: string, data: object) => http.post(`/api/v1/nodes/${nodeId}/tasks`, data)

// src/api/control.ts
export const writePoints = (nodeId: string, deviceId: string, points: Record<string, unknown>) =>
  http.post<{ success: boolean; latency_ms: number; message: string }>(
    `/api/v1/nodes/${nodeId}/devices/${deviceId}/write`, { points }
  )
export const controlTask = (nodeId: string, taskId: string, action: 'pause' | 'resume' | 'stop') =>
  http.put(`/api/v1/nodes/${nodeId}/tasks/${taskId}/${action}`)
```

---

## 12. 组件复用规范

### 12.1 通用组件

| 组件 | 路径 | 说明 |
|------|------|------|
| `StatusBadge` | `components/common/StatusBadge.vue` | 带颜色点的状态标签 |
| `PulsingDot` | `components/common/PulsingDot.vue` | 动态脉冲在线指示 |
| `FormField` | `components/common/FormField.vue` | 带 label + 验证的表单行 |
| `DataQualityDot` | `components/common/DataQualityDot.vue` | 数据质量指示点 |
| `CommandResultToast` | `components/common/CommandResultToast.vue` | 命令执行结果通知 |
| `ConfirmModal` | `components/common/ConfirmModal.vue` | 二次确认弹窗 |

### 12.2 业务组件

| 组件 | 路径 | 说明 |
|------|------|------|
| `MiddlewareCard` | `components/middleware/MiddlewareCard.vue` | 连接卡片 |
| `AddMiddlewareModal` | `components/middleware/AddMiddlewareModal.vue` | 添加连接弹窗 |
| `MQTTConfigForm` | `components/middleware/MQTTConfigForm.vue` | MQTT 配置表单 |
| `NATSConfigForm` | `components/middleware/NATSConfigForm.vue` | NATS 配置表单 |
| `NodeCard` | `components/node/NodeCard.vue` | 节点卡片 |
| `NodeMetricsBar` | `components/node/NodeMetricsBar.vue` | 节点资源监控条 |
| `DeviceTable` | `components/device/DeviceTable.vue` | 设备列表表格 |
| `PointTable` | `components/point/PointTable.vue` | 点位列表表格 |
| `PointCard` | `components/point/PointCard.vue` | 点位实时数据卡片 |
| `WritePointModal` | `components/control/WritePointModal.vue` | 写入点位弹窗 |
| `CommandLogPanel` | `components/control/CommandLogPanel.vue` | 命令执行记录 |
| `TaskControlPanel` | `components/control/TaskControlPanel.vue` | 采集任务控制 |
| `AlertBanner` | `components/alert/AlertBanner.vue` | 告警横幅 |
| `RealtimeDataChart` | `components/chart/RealtimeDataChart.vue` | 实时/历史折线图 |

### 12.3 视图页面

| 路径 | 视图文件 | 说明 |
|------|---------|------|
| `/dashboard` | `views/DashboardView.vue` | 总览仪表盘 |
| `/middlewares` | `views/MiddlewareListView.vue` | 消息总线列表 |
| `/middlewares/:id` | `views/MiddlewareDetailView.vue` | 连接详情 |
| `/nodes` | `views/NodeListView.vue` | 节点列表 |
| `/nodes/:nodeId` | `views/NodeDetailView.vue` | 节点详情 |
| `/nodes/:nodeId/devices` | `views/DeviceListView.vue` | 子设备列表 |
| `/nodes/:nodeId/devices/:deviceId/points` | `views/PointListView.vue` | 点位列表 |
| `/control` | `views/ControlView.vue` | 双向控制面板 |
| `/alerts` | `views/AlertListView.vue` | 告警列表 |
| `/settings` | `views/SettingsView.vue` | 系统设置 |

---

**文档版本**: v2.0  
**最后更新**: 2026-04-16  
**维护者**: edgeOS 团队
