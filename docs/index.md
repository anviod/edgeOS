---
layout: landing
title: EdgeOS 文档中心
description: 面向 EdgeOS 工业大脑的系统设计、协议规范、后端实现与 UI 工程文档入口。
---

<section class="hero" id="overview">
  <div class="hero__grid">
    <div class="hero__copy">
      <span class="hero__eyebrow">EdgeOS Doc</span>
      <h2>边缘计算·核心中枢</h2>
      <p>
        汇总 EdgeOS 的系统架构、EdgeX 通信协议、后端实现路径、UI 规划与工业控制级样式规范，
        用于研发、联调与交付阶段的统一参考。
      </p>
      <div class="hero__actions">
        <a class="button-link button-link--primary" href="#documents">进入文档入口</a>
        <a class="button-link button-link--secondary" href="./EdgeOS-2026-P3-TODO.html">查看 P3 重点</a>
      </div>
    </div>

    <div class="hero__panel">
      <div class="status-panel">
        <div class="status-panel__header">
          <div class="status-panel__title">
            <span class="hero__eyebrow">System Snapshot</span>
            <strong>EdgeOS / EdgeX 协同运行</strong>
          </div>
          <span class="status-pill">RUNNING</span>
        </div>
        <p>面向工业现场的边缘控制与业务扩展平台，聚焦高可用、设备协同、通道健康和群控编排。</p>
      </div>

      <div class="metric-strip">
        <div class="metric-strip__item">
          <span class="metric-strip__label">架构模式</span>
          <span class="metric-strip__value">N+2</span>
        </div>
        <div class="metric-strip__item">
          <span class="metric-strip__label">核心专题</span>
          <span class="metric-strip__value">6+</span>
        </div>
        <div class="metric-strip__item">
          <span class="metric-strip__label">前端方向</span>
          <span class="metric-strip__value">UI / 监控</span>
        </div>
        <div class="metric-strip__item">
          <span class="metric-strip__label">交付阶段</span>
          <span class="metric-strip__value">P3</span>
        </div>
      </div>

      <div class="hero-card-board">
        <article class="hero-mini-card">
          <span class="hero-mini-card__eyebrow">Docs</span>
          <strong>协议、后端、UI、样式</strong>
          <p>文档入口按主题分组，首页负责快速分流。</p>
        </article>
        <article class="hero-mini-card">
          <span class="hero-mini-card__eyebrow">Runtime</span>
          <strong>Latency / Loss / Quality</strong>
          <p>突出工业运行态指标，而不是展示性大图。</p>
        </article>
        <article class="hero-mini-card">
          <span class="hero-mini-card__eyebrow">Workflow</span>
          <strong>P3 规划持续推进</strong>
          <p>从架构设计一路落到页面规划和联调记录。</p>
        </article>
      </div>
    </div>
  </div>
</section>

<section class="landing-section" id="capabilities">
  <div class="section-heading">
    <h2>核心能力</h2>
    <p>围绕高可用、设备协同、通道健康和群控编排，覆盖 EdgeOS 的核心能力与落地方向。</p>
  </div>

  <div class="feature-grid">
    <article class="capability-card">
      <h3>高可用边缘大脑</h3>
      <p>围绕 N+2 冗余、故障切换、节点协同和运行监测构建工业级边缘控制底座。</p>
      <ul>
        <li>主备与多节点协调</li>
        <li>故障检测与自动切换</li>
        <li>边缘运行态集中可视化</li>
      </ul>
    </article>

    <article class="capability-card">
      <h3>设备与通道协同</h3>
      <p>关注影子设备、上下行通信、协议接入和通道健康指标，支撑现场联调与长期运维。</p>
      <ul>
        <li>设备注册与同步</li>
        <li>MQTT / NATS 通信链路</li>
        <li>Latency / Loss / Quality 指标</li>
      </ul>
    </article>

    <article class="capability-card">
      <h3>业务扩展与群控</h3>
      <p>从采集接入扩展到业务中心和群控编排，覆盖储能、BMS、能耗监测与节点调度场景。</p>
      <ul>
        <li>业务页面规划</li>
        <li>群控调度与场景联动</li>
        <li>P3 阶段静态页面范围</li>
      </ul>
    </article>
  </div>
</section>

<section class="landing-section" id="documents">
  <div class="section-heading">
    <h2>文档入口</h2>
    <p>按主题进入架构、协议、实现、联调、UI 与样式规范文档。</p>
  </div>

  <div class="card-grid">
    <a class="doc-card" href="../readme.md">
      <h3>项目总览</h3>
      <p>快速了解 EdgeOS 的总体定位、仓库结构和关键模块，适合作为整个文档站的起点。</p>
      <span class="doc-card__meta">查看总览</span>
    </a>

    <a class="doc-card" href="./EdgeX通信协议规范(MQTT-NATS).html">
      <h3>通信协议规范</h3>
      <p>聚焦 EdgeX 与 EdgeOS 之间的 MQTT / NATS 设计、消息格式、时序与联调约束。</p>
      <span class="doc-card__meta">查看协议</span>
    </a>

    <a class="doc-card" href="./EdgeOS与EdgeX通信测试验证文档.html">
      <h3>通信测试验证</h3>
      <p>沉淀联调过程中的验证步骤、消息示例与测试链路，方便回归和对照检查。</p>
      <span class="doc-card__meta">查看验证文档</span>
    </a>

    <a class="doc-card" href="./EdgeOS_与EdgeX通信测试报告_20260417.html">
      <h3>通信测试报告</h3>
      <p>记录 2026-04-17 的联调报告、测试结论和现阶段问题，为后续迭代提供依据。</p>
      <span class="doc-card__meta">查看报告</span>
    </a>

    <a class="doc-card" href="./EdgeOS 后端实现指南.html">
      <h3>后端实现指南</h3>
      <p>梳理服务职责、实现路径与工程约束，帮助后端开发和前后端对接保持一致。</p>
      <span class="doc-card__meta">查看实现指南</span>
    </a>

    <a class="doc-card" href="./EdgeOS UI规划文档.html">
      <h3>UI 规划文档</h3>
      <p>从信息架构、页面拆分到模块化编排说明前端规划，是页面方案设计的主文档。</p>
      <span class="doc-card__meta">查看 UI 规划</span>
    </a>

    <a class="doc-card" href="./样式规范.html">
      <h3>工业级样式规范</h3>
      <p>定义面向工业控制场景的布局、状态、表格、危险操作与样式覆写规范。</p>
      <span class="doc-card__meta">查看样式规范</span>
    </a>

    <a class="doc-card" href="./EdgeOS-2026-P3-TODO.html">
      <h3>P3 任务重点</h3>
      <p>整理当前阶段的交付目标、页面扩展范围和优先事项，方便团队统一节奏。</p>
      <span class="doc-card__meta">查看 P3 TODO</span>
    </a>
  </div>
</section>

