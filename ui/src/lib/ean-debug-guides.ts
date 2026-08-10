/**
 * EAN 联合调试指南数据
 * 对照 docs/edgeos/EAN2.0-edgeCore-EdgeOS改造指南.md §2.3 / §6.3
 * 与 docs/MQTT_NATS_Implementation_Guide.md（V1 Topic 并存说明）
 */

export interface EanFlowStep {
  id: string
  title: string
  detail: string
  link?: { label: string; path: string }
  check?: string
}

export interface EanGuideExample {
  id: string
  title: string
  description: string
  topic: string
  payload: string
  expect: string
  fillInvoke?: {
    target: string
    capability: string
    arguments: string
    timeout_sec?: number
    tenant_id?: string
  }
}

export interface EanTroubleshootItem {
  symptom: string
  cause: string
  fix: string
}

/** 端到端联调流程 */
export const eanFlowSteps: EanFlowStep[] = [
  {
    id: 'enable',
    title: '启用 EAN',
    detail:
      '在系统配置中设置 ean.enabled=true（配置持久化于 data/config.db，无配置文件），配置 ean.mqtt 和/或 ean.nats，设置 planner_id（默认 edgeos-planner），重启 EdgeOS。',
    link: { label: '打开协调中心', path: '/ean' },
    check: 'GET /api/ean/health → status=ok, started=true',
  },
  {
    id: 'transport-mqtt',
    title: 'MQTT 传输连 Broker',
    detail:
      'ean.mqtt.enabled=true，broker 默认 tcp://127.0.0.1:18083（与 edgeCore 北向联调端口一致）。health.transport_details 中 mqtt.connected=true。',
    link: { label: '协调中心', path: '/ean' },
    check: 'transport_details 含 {name:mqtt, connected:true, endpoint:tcp://127.0.0.1:18083}',
  },
  {
    id: 'transport-nats',
    title: 'NATS 传输对称启用',
    detail:
      'ean.nats.enabled=true，url=nats://127.0.0.1:4222。Subject 使用 NATS 点分形式（$edgeos....，MQTT /→.，通配符 +→* / #→>）。与 MQTT 并行注册，registered_transports=2。',
    link: { label: '联调帮助', path: '/ean/debug' },
    check: 'transports 含 nats；transport_details 含 nats endpoint',
  },
  {
    id: 'discovery',
    title: 'Discovery 上线',
    detail:
      'edgeCore 北向 mqttBus/natsBus 发布 Agent / Capability 到 $edgeos/discovery/*；EdgeOS 建立索引。NATS 通道 Agent.metadata.northbound=edgeos_nats。',
    link: { label: 'Agent 管理', path: '/ean/agents' },
    check: 'Agent 列表出现 edgeCore-node-001 且 status=online',
  },
  {
    id: 'invoke',
    title: '发起 Invoke',
    detail:
      '向 $edgeos/invoke/{agent_id} 发 invoke_capability（MQTT+NATS 双发）；HTTP 可用 POST /api/ean/invoke。header.source 必须等于 planner_id。',
    link: { label: '能力调用', path: '/ean/invoke' },
    check: '返回 invoke_id，审计记录新增 pending/success，v1_fallback=0',
  },
  {
    id: 'reply',
    title: '接收 Reply',
    detail:
      'Reply 主题为 $edgeos/reply/{source}，source 即 EdgeOS planner_id。用 invoke_id / correlation_id 关联；双传输首个 Reply 生效。',
    link: { label: '能力调用结果区', path: '/ean/invoke' },
    check: 'response.status / result.success 可见',
  },
  {
    id: 'event',
    title: 'Event + previous_value',
    detail:
      '订阅 $edgeos/event/{agent_id} 或 broadcast；点位变化事件须含 value 与 previous_value。',
    link: { label: '事件流', path: '/ean/events' },
    check: '事件表 previous_value → value 高亮差分',
  },
  {
    id: 'heartbeat',
    title: '心跳超时 → offline',
    detail:
      'Agent 按 heartbeat_interval_sec 发 $edgeos/heartbeat/{id}；超时倍数默认 3。停止心跳后应标记 offline。',
    link: { label: 'Agent 离线筛选', path: '/ean/agents' },
    check: 'status=offline；审计可出现 heartbeat-timeout',
  },
]

/** 可复制指导用例 */
export const eanGuideExamples: EanGuideExample[] = [
  {
    id: 'discovery-agent',
    title: 'Discovery · Agent 上线',
    description: 'edgeCore → EdgeOS：注册 Agent 描述符',
    topic: '$edgeos/discovery/agent',
    payload: JSON.stringify(
      {
        header: {
          message_id: 'msg-agent-1',
          timestamp: Date.now(),
          source: 'edgeCore-node-001',
          message_type: 'discovery_agent',
          version: '2.0',
        },
        body: {
          agent: {
            id: 'edgeCore-node-001',
            kind: 'edgeCore',
            version: '2.0',
            status: 'online',
            transport: ['mqtt', 'nats'],
            heartbeat_interval_sec: 10,
            metadata: { hostname: 'edge-lab', northbound: 'edgeos_nats' },
          },
        },
      },
      null,
      2,
    ),
    expect: 'Agent 管理页出现 edgeCore-node-001，状态 online',
  },
  {
    id: 'discovery-capability',
    title: 'Discovery · Capability',
    description: '按 agent_id 聚合能力列表',
    topic: '$edgeos/discovery/capability',
    payload: JSON.stringify(
      {
        header: {
          message_id: 'msg-cap-1',
          timestamp: Date.now(),
          source: 'edgeCore-node-001',
          message_type: 'discovery_capability',
          version: '2.0',
        },
        body: {
          capabilities: [
            {
              id: 'system.diagnostics',
              agent_id: 'edgeCore-node-001',
              description: '系统诊断',
              category: 'system',
              timeout_sec: 30,
              permission: 'read',
            },
          ],
        },
      },
      null,
      2,
    ),
    expect: 'Agent 详情 → 能力列表含 system.diagnostics',
  },
  {
    id: 'discovery-nats-note',
    title: 'NATS · Subject 点分约定',
    description: 'NATS 使用点分 Subject（$edgeos....）；勿用斜杠',
    topic: '$edgeos.discovery.capability',
    payload: JSON.stringify(
      {
        note: 'NATS subject = MQTT topic 映射（/→.，通配符 +→* / #→>）',
        publish: 'nats pub \'$edgeos.discovery.agent\' \'<json>\'',
        edgeos_config: {
          'ean.nats.enabled': true,
          'ean.nats.url': 'nats://127.0.0.1:4222',
          'ean.nats.client_name': 'edgeos-ean',
        },
        edgeCore_channel: {
          name: 'EAN-NATS',
          enable: true,
          ean_enabled: true,
          url: 'nats://127.0.0.1:4222',
          node_id: 'edgeCore-node-001',
        },
      },
      null,
      2,
    ),
    expect: 'health.transport_details 含 nats；Agent.transport 可含 nats；northbound=edgeos_nats',
  },
  {
    id: 'invoke-diagnostics',
    title: 'Invoke · system.diagnostics',
    description: 'EdgeOS Planner 发起调用；也可在 UI「能力调用」页用示例填充',
    topic: '$edgeos/invoke/edgeCore-node-001',
    payload: JSON.stringify(
      {
        header: {
          message_id: 'msg-1',
          timestamp: 0,
          source: 'edgeos-planner',
          message_type: 'invoke_capability',
          version: '2.0',
          correlation_id: 'corr-1',
        },
        body: {
          invoke_id: 'inv-1',
          target: 'edgeCore-node-001',
          capability: 'system.diagnostics',
          arguments: {},
        },
      },
      null,
      2,
    ),
    expect: 'Reply 到 $edgeos/reply/edgeos-planner，status=completed/success，invoke_id 对齐',
    fillInvoke: {
      target: 'edgeCore-node-001',
      capability: 'system.diagnostics',
      arguments: '{}',
      timeout_sec: 30,
      tenant_id: 'default',
    },
  },
  {
    id: 'event-previous',
    title: 'Event · previous_value',
    description: '点位变化须带 previous_value',
    topic: '$edgeos/event/edgeCore-node-001',
    payload: JSON.stringify(
      {
        header: {
          message_id: 'msg-evt-1',
          timestamp: Date.now(),
          source: 'edgeCore-node-001',
          message_type: 'point_change',
          version: '2.0',
        },
        body: {
          event_type: 'point.change',
          agent_id: 'edgeCore-node-001',
          device_id: 'device-001',
          point_id: 'temp',
          value: 26.5,
          previous_value: 25.0,
          timestamp: Date.now(),
        },
      },
      null,
      2,
    ),
    expect: '事件流页 previous_value=25 → value=26.5 高亮差分',
  },
  {
    id: 'heartbeat',
    title: 'Heartbeat',
    description: '周期性心跳；停止后超时标记 offline',
    topic: '$edgeos/heartbeat/edgeCore-node-001',
    payload: JSON.stringify(
      {
        header: {
          message_id: 'msg-hb-1',
          timestamp: Date.now(),
          source: 'edgeCore-node-001',
          message_type: 'heartbeat',
          version: '2.0',
        },
        body: {
          agent_id: 'edgeCore-node-001',
          status: 'online',
          timestamp: Date.now(),
          sequence: 1,
        },
      },
      null,
      2,
    ),
    expect: 'tracked_agents 增加；停发后约 3×interval 变 offline',
  },
]

/** 常见失败排查 */
export const eanTroubleshoot: EanTroubleshootItem[] = [
  {
    symptom: 'GET /api/ean/agents 返回 503「EAN Bus 未启用」',
    cause: 'ean.enabled=false 或未配置可用传输',
    fix: '打开 ean.enabled，启用 mqtt/nats 后重启；health 应返回 status=ok',
  },
  {
    symptom: 'NATS 收不到 $edgeos.... 消息',
    cause: '在 NATS 上仍用斜杠 subject（如 $edgeos.discovery.agent 写成 $edgeos/discovery/agent），或 ean.nats.enabled=false',
    fix: 'EAN 2.0 NATS 须用点分 subject：$edgeos.discovery.agent（MQTT /→.）；配置启用 nats://127.0.0.1:4222',
  },
  {
    symptom: 'health.transports 只有 mqtt 没有 nats',
    cause: 'NATS 未启用、4222 不可达，或 MaxReconnects 耗尽后连接关闭',
    fix: '确认 nats-server 监听 4222；ean.nats.enabled=true；查看 transport_details[].connected',
  },
  {
    symptom: 'Invoke 超时 / 无 Reply',
    cause: 'reply topic 与 header.source / planner_id 不一致',
    fix: 'Reply 订阅 $edgeos/reply/{planner_id}；信封 header.source 须等于 planner_id',
  },
  {
    symptom: 'write/admin/AI 能力被拒绝',
    cause: '无租户策略时敏感能力默认拒绝',
    fix: 'POST /api/ean/governance/policies 配置 allow_cap / allow_target',
  },
  {
    symptom: 'Agent 一直 offline 或索引为空',
    cause: 'Discovery 信封未按 {header,body} 发布，或 transport 未连上',
    fix: '对照本页 Discovery 示例 JSON；检查 health.transports',
  },
  {
    symptom: '与 V1 数据混乱 / 双写',
    cause: '同一业务同时走 edgeCore/* 命令与 EAN Invoke',
    fix: 'Topic 隔离：V1=edgeCore/*，EAN=$edgeos/*；新功能只走 EAN',
  },
]

/** HTTP API 速查 */
export const eanHttpApiQuickRef = [
  { method: 'GET', path: '/api/ean/health', note: '未启用也返回 200，status=disabled' },
  { method: 'GET', path: '/api/ean/agents', note: '列表 + total' },
  { method: 'GET', path: '/api/ean/agents/:id/capabilities', note: '能力列表' },
  { method: 'POST', path: '/api/ean/invoke', note: '{ target, capability, arguments, timeout_sec, tenant_id }' },
  { method: 'GET', path: '/api/ean/events/recent?n=100', note: '含 previous_value' },
  { method: 'GET', path: '/api/ean/audit?limit=100', note: '内存审计' },
  { method: 'POST', path: '/api/ean/governance/policies', note: '租户策略' },
]

/** 构建 Invoke 页深链 */
export function buildInvokeFillPath(fill: NonNullable<EanGuideExample['fillInvoke']>): string {
  const q = new URLSearchParams()
  q.set('target', fill.target)
  q.set('capability', fill.capability)
  q.set('arguments', fill.arguments)
  if (fill.timeout_sec != null) q.set('timeout_sec', String(fill.timeout_sec))
  if (fill.tenant_id) q.set('tenant_id', fill.tenant_id)
  q.set('fill', '1')
  return `/ean/invoke?${q.toString()}`
}
