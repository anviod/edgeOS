import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { formatDateTime } from '@/lib/live-time'

// =====================================================
// 群控编排实时仿真 Store
// 以固定节拍驱动节点调度 / 场景联动 / 函数执行 / 脚本编排
// 五大页面数据，为页面提供高保真动效数据
// =====================================================

function clamp(v: number, min: number, max: number) {
  return Math.max(min, Math.min(max, v))
}

function drift(value: number, amp: number, min: number, max: number) {
  const next = value + (Math.random() - 0.5) * 2 * amp
  return clamp(next, min, max)
}

function jitterInt(value: number, amp: number, min: number, max: number) {
  return Math.round(drift(value, amp, min, max))
}

function buildSeries(base: number, count: number, amp: number, min: number, max: number) {
  return Array.from({ length: count }, () => drift(base, amp, min, max))
}

export interface GcNodeSim {
  id: string
  name: string
  cpu: number
  memory: number
  queue: number
  policy: string
  status: 'stable' | 'watch' | 'standby'
}

export interface GcSceneSim {
  id: string
  name: string
  event: string
  condition: string
  action: string
  quality: number
  status: 'stable' | 'watch' | 'risk'
}

export interface GcFunctionSim {
  id: string
  name: string
  input: string
  output: string
  latency: number
  quality: number
  status: 'stable' | 'running' | 'watch' | 'retry'
}

export interface GcWorkflowSim {
  id: string
  name: string
  version: string
  approval: string
  dag: string
  quality: number
  status: 'stable' | 'approving' | 'pending' | 'risk'
}

export const useGroupControlStore = defineStore('groupControl', () => {
  // ========== 总览 ==========
  const onlineNodes = ref(26)
  const runningTasks = ref(83)
  const sceneSuccessRate = ref(97.4)
  const failedExecs = ref(2)
  const taskSeries = ref(buildSeries(80, 24, 6, 60, 96))
  const successSeries = ref(buildSeries(96, 24, 1.6, 90, 100))

  // ========== 节点调度 ==========
  const nodes = ref<GcNodeSim[]>([
    { id: 'ns1', name: 'node-sh-01', cpu: 72, memory: 64, queue: 6, policy: '负载均衡', status: 'stable' },
    { id: 'ns2', name: 'node-hz-02', cpu: 58, memory: 49, queue: 4, policy: '延迟优先', status: 'stable' },
    { id: 'ns3', name: 'node-gz-03', cpu: 81, memory: 77, queue: 8, policy: '失败重派', status: 'watch' },
    { id: 'ns4', name: 'node-xa-01', cpu: 43, memory: 39, queue: 0, policy: '冷备待命', status: 'standby' },
  ])
  const nodeCpuSeries = ref(buildSeries(62, 24, 4, 35, 90))

  // ========== 场景联动 ==========
  const triggerCount = ref(412)
  const triggerSeries = ref(buildSeries(80, 24, 8, 50, 96))
  const scenes = ref<GcSceneSim[]>([
    { id: 'sl1', name: '夜间谷充联动', event: '电价切谷', condition: 'SOC < 68%', action: '启动谷充', quality: 98, status: 'stable' },
    { id: 'sl2', name: '温差异常联动', event: '温差 > 5℃', condition: '持续 3 分钟', action: '派单 + 降载', quality: 95, status: 'stable' },
    { id: 'sl3', name: '停车场拥堵联动', event: '排队 > 5', condition: '连续 10 分钟', action: '增开枪位', quality: 89, status: 'watch' },
    { id: 'sl4', name: '整站停机保护', event: 'Loss > 3%', condition: '关键链路断连', action: '停机保护', quality: 92, status: 'risk' },
  ])

  // ========== 函数执行 ==========
  const execCount = ref(1824)
  const execSeries = ref(buildSeries(70, 24, 6, 40, 92))
  const functions = ref<GcFunctionSim[]>([
    { id: 'f1', name: 'func_energy_score', input: '站点功率流', output: '评分 96', latency: 182, quality: 97, status: 'stable' },
    { id: 'f2', name: 'func_charge_balance', input: '枪位状态', output: '负载分配', latency: 236, quality: 95, status: 'running' },
    { id: 'f3', name: 'func_bms_risk', input: '温压差矩阵', output: '风险等级', latency: 341, quality: 91, status: 'watch' },
    { id: 'f4', name: 'func_invoice_check', input: '账单批次', output: '异常列表', latency: 418, quality: 84, status: 'retry' },
  ])
  const latencySeries = ref(buildSeries(218, 24, 12, 120, 340))

  // ========== 脚本编排 ==========
  const workflows = ref<GcWorkflowSim[]>([
    { id: 'w1', name: 'wf-night-charge', version: 'v3.2.1', approval: '已通过', dag: '7 节点 / 2 分支', quality: 94, status: 'stable' },
    { id: 'w2', name: 'wf-bms-recover', version: 'v1.8.0', approval: '待复核', dag: '5 节点 / 1 回滚', quality: 90, status: 'approving' },
    { id: 'w3', name: 'wf-stop-cluster', version: 'v2.4.5', approval: '高风险', dag: '4 节点 / 1 确认词', quality: 88, status: 'pending' },
    { id: 'w4', name: 'wf-charge-rebalance', version: 'v2.0.3', approval: '已通过', dag: '6 节点 / 2 并行', quality: 92, status: 'stable' },
  ])

  // ========== 事件流（按域隔离）==========
  interface SimEvent {
    title: string
    subtitle: string
    meta: string
    status: string
  }
  const eventPools: Record<string, Omit<SimEvent, 'meta'>[]> = {
    scheduling: [
      { title: 'node-gz-03 负载偏高', subtitle: '自动迁移 2 个任务', status: 'warning' },
      { title: 'node-xa-01 热备上线', subtitle: '加入可调度资源池', status: 'active' },
      { title: '任务迁移完成', subtitle: 'worker-energy-19 已切换', status: 'resolved' },
    ],
    scene: [
      { title: '停车场联动失败 1 次', subtitle: '枪位控制回执超时', status: 'warning' },
      { title: '夜间谷充规则更新', subtitle: '新增 2 个站点范围', status: 'active' },
      { title: '温差联动闭环', subtitle: '巡检任务已派发', status: 'resolved' },
    ],
    function: [
      { title: 'func_invoice_check 超时', subtitle: 'I/O 请求超过 400ms', status: 'error' },
      { title: 'func_energy_score 预热成功', subtitle: '冷启动耗时下降', status: 'active' },
      { title: 'func_bms_risk 完成重试', subtitle: '输出结果已回写', status: 'resolved' },
    ],
    workflow: [
      { title: 'wf-stop-cluster 待审批', subtitle: '包含 L5 停止动作', status: 'warning' },
      { title: 'wf-bms-recover 自动重派', subtitle: '原节点执行失败后切换', status: 'error' },
      { title: 'wf-night-charge 完成', subtitle: '夜间谷充 DAG 执行完毕', status: 'resolved' },
    ],
    overview: [
      { title: '节点调度回收', subtitle: 'node-gz-03 释放高负载任务', status: 'warning' },
      { title: '脚本自动重派', subtitle: 'workflow-prod-17 切换至 node-sh-02', status: 'active' },
      { title: '场景联动恢复正常', subtitle: '停机联动链已恢复', status: 'resolved' },
    ],
  }
  const eventFeeds: Record<string, SimEvent[]> = Object.fromEntries(
    Object.entries(eventPools).map(([key, pool]) => [
      key,
      pool.map((item, i) => ({
        ...item,
        meta: `${formatDateTime(new Date(Date.now() - (i + 1) * 5 * 60000))} / ${key === 'overview' ? '群控指挥面' : key}`,
      })),
    ])
  )

  let timer: ReturnType<typeof setInterval> | null = null
  const currentTime = ref(formatDateTime(new Date()))

  function pushEvent(poolKey: string, tag: string) {
    const pool = eventPools[poolKey]
    const feed = eventFeeds[poolKey]
    if (!pool || !feed || feed.length >= 6) return
    const item = pool[Math.floor(Math.random() * pool.length)]
    feed.unshift({ ...item, meta: `${formatDateTime(new Date())} / ${tag}` })
    if (feed.length > 6) feed.pop()
  }

  function tick() {
    currentTime.value = formatDateTime(new Date())

    // 总览
    runningTasks.value = jitterInt(runningTasks.value, 4, 60, 110)
    sceneSuccessRate.value = drift(sceneSuccessRate.value, 0.3, 94, 99.5)
    failedExecs.value = jitterInt(failedExecs.value, 1, 0, 6)
    taskSeries.value = [...taskSeries.value.slice(1), runningTasks.value]
    successSeries.value = [...successSeries.value.slice(1), sceneSuccessRate.value]

    // 节点调度
    nodes.value = nodes.value.map(n => {
      const cpu = drift(n.cpu, 2.5, 25, 92)
      const memory = drift(n.memory, 1.8, 20, 88)
      const queue = jitterInt(n.queue, 1, 0, 14)
      const status = cpu > 78 ? 'watch' : n.policy === '冷备待命' ? 'standby' : 'stable'
      return { ...n, cpu, memory, queue, status }
    })
    nodeCpuSeries.value = [...nodeCpuSeries.value.slice(1), nodes.value.reduce((a, n) => a + n.cpu, 0) / nodes.value.length]

    // 场景联动
    triggerCount.value += jitterInt(2, 1, 0, 5)
    triggerSeries.value = [...triggerSeries.value.slice(1), triggerCount.value % 120]
    scenes.value = scenes.value.map(s => ({
      ...s,
      quality: Math.round(clamp(s.quality + drift(0, 0.5, -1, 1), 78, 100)),
    }))

    // 函数执行
    execCount.value += jitterInt(4, 2, 0, 10)
    execSeries.value = [...execSeries.value.slice(1), execCount.value % 200]
    functions.value = functions.value.map(f => ({
      ...f,
      latency: Math.round(clamp(f.latency + drift(0, 14, -20, 20), 80, 460)),
      quality: Math.round(clamp(f.quality + drift(0, 0.6, -1.2, 1.2), 78, 100)),
      status: f.latency > 390 ? 'retry' : f.latency > 300 ? 'watch' : f.latency > 200 ? 'running' : 'stable',
    }))
    latencySeries.value = [...latencySeries.value.slice(1), functions.value.reduce((a, f) => a + f.latency, 0) / functions.value.length]

    // 脚本编排
    workflows.value = workflows.value.map(w => ({
      ...w,
      quality: Math.round(clamp(w.quality + drift(0, 0.5, -1, 1), 78, 100)),
    }))

    // 事件流
    if (Math.random() < 0.16) pushEvent('scheduling', '调度器')
    if (Math.random() < 0.14) pushEvent('scene', '规则引擎')
    if (Math.random() < 0.14) pushEvent('function', '函数域')
    if (Math.random() < 0.13) pushEvent('workflow', '编排引擎')
    if (Math.random() < 0.14) pushEvent('overview', '群控指挥面')
  }

  function start() {
    if (timer) return
    timer = setInterval(tick, 1600)
  }

  function stop() {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  function domainEvents(key: string) {
    return eventFeeds[key] ?? []
  }

  const avgCpu = computed(() => Math.round(nodes.value.reduce((a, n) => a + n.cpu, 0) / nodes.value.length))
  const pendingQueue = computed(() => nodes.value.reduce((a, n) => a + n.queue, 0))
  const watchNodes = computed(() => nodes.value.filter(n => n.status === 'watch').length)
  const watchScenes = computed(() => scenes.value.filter(s => s.status !== 'stable').length)
  const avgLatency = computed(() => Math.round(functions.value.reduce((a, f) => a + f.latency, 0) / functions.value.length))
  const retryCount = computed(() => functions.value.filter(f => f.status === 'retry').length)
  const approvingWorkflows = computed(() => workflows.value.filter(w => w.status === 'approving').length)
  const riskWorkflows = computed(() => workflows.value.filter(w => w.status === 'risk' || w.approval === '高风险').length)

  return {
    onlineNodes, runningTasks, sceneSuccessRate, failedExecs, taskSeries, successSeries,
    nodes, nodeCpuSeries, avgCpu, pendingQueue, watchNodes,
    triggerCount, triggerSeries, scenes, watchScenes,
    execCount, execSeries, functions, avgLatency, retryCount, latencySeries,
    workflows, approvingWorkflows, riskWorkflows,
    domainEvents, currentTime,
    start, stop,
  }
})
