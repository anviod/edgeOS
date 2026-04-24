---
layout: default
title: EdgeOS 文档中心
description: EdgeOS 边缘大脑系统 GitHub Pages 文档首页
---

# EdgeOS 文档中心

EdgeOS 是面向工业边缘场景的“边缘大脑系统”，用于配合 EdgeX 边缘采集网关，构建具备 **N+2 冗余架构、群控调度、影子设备协同、业务运营扩展** 的工业级平台。

---

## 核心能力

### 1. 高可用边缘大脑

- N+2 冗余架构
- 主母皇 / 备用母皇双机热备
- 多边缘采集网关统一协调
- 故障检测与自动切换

<div align="center">
  <img src="./img/edge_brain.svg" width="100%" alt="EdgeOS 总体架构" />
</div>

### 2. 影子设备与双向通信

- 影子设备自动发现
- 跨边缘节点的设备注册与同步
- 上行遥测、事件、告警统一接入
- 下行控制、配置、任务分发能力

<div align="center">
  <img src="./img/edge_01.svg" width="90%" alt="影子设备自动发现" />
</div>

<div align="center">
  <img src="./img/edge_02.svg" width="90%" alt="双向通信机制" />
</div>

### 3. 群控与节能优化

- 群控算法调度
- 节点任务分配与负载均衡
- 业务策略执行
- 能耗优化与节能调度

<div align="center">
  <img src="./img/edge_brain_n2.svg" width="100%" alt="N+2 冗余架构" />
</div>

---

## 前端能力现状

当前前端已完成从“采集运行层”向“业务中心 + 群控管理”两大域的扩展，支持 GitHub Pages 文档展示与 UI 规划沉淀。

### 业务中心

- 储能管理
- 电源BMS
- 充电管理
- 能耗监测
- 账务台账

<div align="center">
  <img src="./img/储能管理.png" width="85%" alt="储能管理页面" />
</div>

<div align="center">
  <img src="./img/电源BMS.png" width="85%" alt="电源BMS页面" />
</div>

<div align="center">
  <img src="./img/能耗监测.png" width="85%" alt="能耗监测页面" />
</div>

<div align="center">
  <img src="./img/账务台账.png" width="85%" alt="账务台账页面" />
</div>

### 群控管理

- 节点调度
- 场景联动
- 函数执行
- 脚本编排

<div align="center">
  <img src="./img/节点调度.png" width="85%" alt="节点调度页面" />
</div>

<div align="center">
  <img src="./img/场景联动.png" width="85%" alt="场景联动页面" />
</div>

---

## 文档导航

### 系统与协议

- [项目总说明](../readme.md)
- [EdgeOS 与 EdgeX 通信测试验证文档](./EdgeOS与EdgeX通信测试验证文档.md)
- [EdgeOS 与 EdgeX 通信测试报告](./EdgeOS_与EdgeX通信测试报告_20260417.md)
- [EdgeX 通信协议规范 (MQTT-NATS)](./EdgeX通信协议规范(MQTT-NATS).md)
- [EdgeOS 后端实现指南](./EdgeOS%20后端实现指南.md)

### UI 与前端规划

- [EdgeOS UI 规划文档](./EdgeOS%20UI规划文档.md)
- [EdgeOS 工业大脑 UI 开发实操指南](./样式规范.md)
- [EdgeOS 2026 P3 TODO](./EdgeOS-2026-P3-TODO.md)

---

## P3 扩展重点

P3 阶段重点是把前端从“采集接入平台”提升为“业务运营平台 + 群控编排平台”：

- 导航升级为一级分组 + 二级菜单
- 新增 11 个高仿真静态页面
- 所有页面统一展示 `Latency / Loss / Quality`
- 页面从统一模板升级为模块化、差异化独立编排
- 规划文档中已补齐技术实现路径与依赖关系

---

## 建议阅读顺序

1. [项目总说明](../readme.md)
2. [EdgeX 通信协议规范 (MQTT-NATS)](./EdgeX通信协议规范(MQTT-NATS).md)
3. [EdgeOS 后端实现指南](./EdgeOS%20后端实现指南.md)
4. [EdgeOS UI 规划文档](./EdgeOS%20UI规划文档.md)
5. [EdgeOS 工业大脑 UI 开发实操指南](./样式规范.md)
6. [EdgeOS 2026 P3 TODO](./EdgeOS-2026-P3-TODO.md)
