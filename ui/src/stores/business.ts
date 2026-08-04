import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { formatDateTime } from '@/lib/live-time'

// =====================================================
// 业务中心实时仿真 Store
// 以固定节拍驱动五大业务域数据变化，为页面提供高保真动效数据
// =====================================================

export type StorageMode = 'idle' | 'charging' | 'discharging'

export interface StorageSiteSim {
  id: string
  name: string
  soc: number
  soh: number
  power: number
  strategy: string
  status: 'running' | 'watch' | 'warn'
}

export interface BmsClusterSim {
  id: string
  name: string
  temp: number
  voltage: number
  balance: string
  life: number
  status: 'healthy' | 'watch' | 'risk'
  risk: number
}

export interface ChargingLaneSim {
  id: string
  name: string
  total: number
  occupied: number
  slots: boolean[]
  tone: 'hot' | 'calm' | 'risk'
  note?: string
}

export interface LedgerEntrySim {
  id: string
  no: string
  amount: number
  invoice: string
  settlement: string
  quality: number
  status: 'normal' | 'watch' | 'error'
}

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

function seedSlots(total: number, occupied: number) {
  const slots = Array.from({ length: total }, (_, i) => i < occupied)
  for (let i = slots.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[slots[i], slots[j]] = [slots[j], slots[i]]
  }
  return slots
}

export const useBusinessStore = defineStore('business', () => {
  // ========== 储能管理 ==========
  const storageMode = ref<StorageMode>('idle')
  const avgSoc = ref(68)
  const avgSoh = ref(93)
  const totalPower = ref(41.8)
  const socSeries = ref(buildSeries(68, 24, 1.6, 50, 88))
  const powerSeries = ref(buildSeries(41.8, 24, 2.4, 30, 56))

  const sites = ref<StorageSiteSim[]>([
    { id: 'es1', name: '华东一站', soc: 72, soh: 95, power: 5.8, strategy: '峰前预充', status: 'running' },
    { id: 'es2', name: '华南二站', soc: 64, soh: 92, power: 4.9, strategy: '滚动削峰', status: 'watch' },
    { id: 'es3', name: '西北储能簇', soc: 59, soh: 88, power: 3.6, strategy: '备用待命', status: 'warn' },
    { id: 'es4', name: '长三角站群', soc: 71, soh: 94, power: 7.2, strategy: '套利执行', status: 'running' },
  ])

  // ========== 电源BMS ==========
  const clusters = ref<BmsClusterSim[]>([
    { id: 'b1', name: 'Cluster-A12', temp: 2.1, voltage: 0.08, balance: '均衡完成', life: 95, status: 'healthy', risk: 0.22 },
    { id: 'b2', name: 'Cluster-C03', temp: 4.8, voltage: 0.17, balance: '均衡中', life: 91, status: 'watch', risk: 0.48 },
    { id: 'b3', name: 'Cluster-N07', temp: 6.2, voltage: 0.24, balance: '中止', life: 83, status: 'risk', risk: 0.86 },
    { id: 'b4', name: 'Cluster-S09', temp: 3.5, voltage: 0.13, balance: '均衡完成', life: 93, status: 'healthy', risk: 0.3 },
  ])
  const tempSeries = ref(buildSeries(4, 24, 0.25, 2.5, 5.5))

  // ========== 充电管理 ==========
  const activeSessions = ref(41)
  const queueVehicles = ref(6)
  const lanes = ref<ChargingLaneSim[]>([
    { id: 'a', name: 'A 区快充站', total: 20, occupied: 18, slots: seedSlots(20, 18), tone: 'hot' },
    { id: 'b', name: 'B 区综合站', total: 16, occupied: 11, slots: seedSlots(16, 11), tone: 'calm' },
    { id: 'c', name: '园区北站', total: 12, occupied: 7, slots: seedSlots(12, 7), tone: 'calm' },
    { id: 'd', name: '物流南站', total: 8, occupied: 5, slots: seedSlots(8, 5), tone: 'risk', note: '支付回执延迟，补单队列等待人工确认。' },
  ])
  const sessionSeries = ref(buildSeries(41, 24, 2.4, 32, 50))

  // ========== 能耗监测 ==========
  const totalEnergy = ref(126.8)
  const energyLevel = ref(72)
  const energySeries = ref(buildSeries(72, 24, 2.6, 55, 90))
  const loops = ref([
    { id: 'e1', name: '华东一站 / 主回路', energy: 38.2, quality: 98, latency: 17, loss: 0.1, status: '稳定' },
    { id: 'e2', name: '华南二站 / 辅回路', energy: 24.7, quality: 93, latency: 22, loss: 0.2, status: '观察' },
    { id: 'e3', name: '西北储能簇 / 旧支路', energy: 12.9, quality: 88, latency: 34, loss: 0.6, status: '异常波动' },
    { id: 'e4', name: '园区北站 / 生产线', energy: 19.1, quality: 96, latency: 19, loss: 0.1, status: '稳定' },
  ])

  // ========== 账务台账 ==========
  const pendingAmount = ref(86.2)
  const settleRate = ref(93)
  const ledgerEntries = ref<LedgerEntrySim[]>([
    { id: 'l1', no: 'LED-20260424-001', amount: 128000, invoice: '已开票', settlement: '已结算', quality: 97, status: 'normal' },
    { id: 'l2', no: 'LED-20260424-014', amount: 92400, invoice: '待开票', settlement: '对账中', quality: 92, status: 'watch' },
    { id: 'l3', no: 'LED-20260424-018', amount: 31200, invoice: '开票失败', settlement: '冻结', quality: 84, status: 'error' },
    { id: 'l4', no: 'LED-20260424-023', amount: 76500, invoice: '已开票', settlement: '已结算', quality: 95, status: 'normal' },
  ])

  // ========== 业务中心总览 ==========
  const todayRevenue = ref(186.4)
  const revenueSeries = ref(buildSeries(150, 24, 4, 130, 200))

  // ========== 事件流 ==========
  interface SimEvent {
    title: string
    subtitle: string
    meta: string
    status: string
  }
  const eventPools: Record<string, Omit<SimEvent, 'meta'>[]> = {
    storage: [
      { title: '华东一站 SOC 回升', subtitle: '谷充策略执行中', status: 'resolved' },
      { title: '西北储能簇 SOH 下降', subtitle: '建议 72 小时内复检', status: 'error' },
      { title: '华南二站调峰偏差', subtitle: '实际出力低于计划 6%', status: 'warning' },
      { title: '长三角站群功率回正', subtitle: '夜间谷充恢复稳定', status: 'resolved' },
    ],
    bms: [
      { title: 'Cluster-N07 温差过限', subtitle: '温差 6.2℃ 超阈值', status: 'error' },
      { title: 'Cluster-C03 均衡耗时长', subtitle: '连续 3 次均衡超时', status: 'warning' },
      { title: 'Cluster-A12 压差回落', subtitle: '压差已回到安全带', status: 'resolved' },
    ],
    charging: [
      { title: 'A 区排队积压', subtitle: '预计等待 18 分钟', status: 'warning' },
      { title: '物流南站异常订单', subtitle: '回执流水未完成', status: 'error' },
      { title: 'B 区预约成功', subtitle: '夜间预约车位释放', status: 'resolved' },
    ],
    energy: [
      { title: '旧支路波动过大', subtitle: '频繁超出预估带宽', status: 'error' },
      { title: '华南辅回路峰时偏高', subtitle: '建议开启削峰策略', status: 'warning' },
      { title: '主回路恢复正常', subtitle: '波动回归基线', status: 'resolved' },
    ],
    ledger: [
      { title: 'LED-018 开票失败', subtitle: '税号校验未通过', status: 'error' },
      { title: '批次对账延迟', subtitle: '南区账单导入延后 12 分钟', status: 'warning' },
      { title: '自动结算完成', subtitle: '主批次 214 笔已归档', status: 'resolved' },
    ],
    overview: [
      { title: '华南二站收益偏差', subtitle: '计划收益偏差 +12.4%', status: 'warning' },
      { title: '账务对账失败', subtitle: '发票号 INV-20260424-18', status: 'error' },
      { title: '充电队列突增', subtitle: '午高峰预计延迟 18 分钟', status: 'active' },
      { title: '站点巡检完成', subtitle: 'BMS 温差异常任务闭环', status: 'resolved' },
    ],
  }

  // 按业务域隔离的事件流，保证每个页面只展示本域强关联信息
  const eventFeeds: Record<string, SimEvent[]> = Object.fromEntries(
    Object.entries(eventPools).map(([key, pool]) => [
      key,
      pool.map((item, i) => ({
        ...item,
        meta: `${formatDateTime(new Date(Date.now() - (i + 1) * 5 * 60000))} / ${key === 'overview' ? '经营驾驶舱' : key}`,
      })),
    ])
  )

  let timer: ReturnType<typeof setInterval> | null = null

  function nowMeta(tag: string) {
    return `${formatDateTime(new Date())} / ${tag}`
  }

  function pushEvent(poolKey: string, tag: string) {
    const pool = eventPools[poolKey]
    const feed = eventFeeds[poolKey]
    if (!pool || !feed || feed.length >= 6) return
    const item = pool[Math.floor(Math.random() * pool.length)]
    feed.unshift({ ...item, meta: nowMeta(tag) })
    if (feed.length > 6) feed.pop()
  }

  const currentTime = ref(formatDateTime(new Date()))

  function tick() {
    currentTime.value = formatDateTime(new Date())
    storageLatency.value = Math.round(clamp(storageLatency.value + drift(0, 1.2, -2, 2), 14, 30))
    storageQuality.value = Math.round(clamp(storageQuality.value + drift(0, 0.6, -1.2, 1), 95, 100))
    storageLoss.value = Math.round(clamp(storageLoss.value + drift(0, 0.02, -0.04, 0.04), 0, 0.5) * 10) / 10
    // 储能
    const socStep = storageMode.value === 'charging' ? 0.7 : storageMode.value === 'discharging' ? -0.7 : drift(0, 0.25, -0.4, 0.4)
    avgSoc.value = clamp(avgSoc.value + socStep, 15, 98)
    avgSoh.value = drift(avgSoh.value, 0.05, 80, 99)
    totalPower.value = drift(totalPower.value, 1.6, 12, 68)
    socSeries.value = [...socSeries.value.slice(1), avgSoc.value]
    powerSeries.value = [...powerSeries.value.slice(1), totalPower.value]
    sites.value = sites.value.map(s => ({
      ...s,
      soc: clamp(s.soc + (storageMode.value === 'charging' ? 0.3 : storageMode.value === 'discharging' ? -0.3 : drift(0, 0.3, -0.5, 0.5)), 20, 96),
      soh: clamp(s.soh + drift(0, 0.05, -0.1, 0.1), 80, 99),
      power: clamp(s.power + drift(0, 0.5, -0.6, 0.6), 1.5, 10),
    }))

    // BMS（行业规范：温差≤3℃正常 / 3~5℃关注 / >5℃告警；压差≤0.1V正常 / 0.1~0.2V关注 / >0.2V告警）
    clusters.value = clusters.value.map(c => {
      const temp = drift(c.temp, 0.2, 1.2, 7.5)
      const voltage = drift(c.voltage, 0.012, 0.03, 0.38)
      const tempRisk = temp > 5 ? 2 : temp > 3 ? 1 : 0
      const voltRisk = voltage > 0.2 ? 2 : voltage > 0.1 ? 1 : 0
      const status = tempRisk + voltRisk >= 3 ? 'risk' : tempRisk + voltRisk >= 1 ? 'watch' : 'healthy'
      const risk = clamp(temp / 7.5 + voltage / 0.38, 0.15, 0.95)
      return { ...c, temp, voltage, status, risk, balance: temp > 3 ? '均衡中' : '均衡完成' }
    })
    tempSeries.value = [...tempSeries.value.slice(1), clusters.value.reduce((a, c) => a + c.temp, 0) / clusters.value.length]

    // 充电
    activeSessions.value = jitterInt(activeSessions.value, 3, 24, 58)
    queueVehicles.value = jitterInt(queueVehicles.value, 2, 0, 18)
    sessionSeries.value = [...sessionSeries.value.slice(1), activeSessions.value]
    lanes.value = lanes.value.map(lane => {
      if (lane.tone === 'risk' || Math.random() > 0.55) return lane
      const idx = Math.floor(Math.random() * lane.total)
      const slots = [...lane.slots]
      slots[idx] = !slots[idx]
      const occupied = slots.filter(Boolean).length
      return { ...lane, slots, occupied }
    })

    // 能耗
    totalEnergy.value += drift(0.4, 0.2, 0.05, 0.9)
    // 平滑负荷曲线：围绕 72 基线小幅波动，贴合真实峰谷负荷，不做突变
    energyLevel.value = clamp(energyLevel.value + drift(0, 1.4, -1.8, 1.8), 46, 94)
    energySeries.value = [...energySeries.value.slice(1), energyLevel.value]
    loops.value = loops.value.map(l => ({
      ...l,
      energy: clamp(l.energy + drift(0, 0.4, -0.3, 0.5), 5, 60),
      quality: Math.round(clamp(l.quality + drift(0, 0.8, -1.2, 1.2), 78, 100)),
      latency: Math.round(clamp(l.latency + drift(0, 1.4, -2, 2), 12, 42)),
    }))

    // 账务
    pendingAmount.value = drift(pendingAmount.value, 1.2, 40, 120)
    settleRate.value = drift(settleRate.value, 0.4, 82, 99)
    ledgerEntries.value = ledgerEntries.value.map(e => ({
      ...e,
      amount: Math.round(clamp(e.amount + drift(0, 400, -600, 600), 20000, 160000)),
      quality: Math.round(clamp(e.quality + drift(0, 0.7, -1, 1), 78, 100)),
    }))

    // 总览：今日收益稳步累计（每拍小幅增长，体现 "慢慢累计增加"）
    todayRevenue.value = clamp(todayRevenue.value + drift(0.35, 0.2, 0.05, 0.8), 150, 230)
    revenueSeries.value = [...revenueSeries.value.slice(1), todayRevenue.value]

    // 事件流（低概率触发新事件，各域独立）
    if (Math.random() < 0.16) pushEvent('storage', '调峰策略')
    if (Math.random() < 0.12) pushEvent('bms', 'BMS 诊断')
    if (Math.random() < 0.14) pushEvent('charging', '充电调度')
    if (Math.random() < 0.12) pushEvent('energy', '能流分析')
    if (Math.random() < 0.12) pushEvent('ledger', '台账链路')
    if (Math.random() < 0.14) pushEvent('overview', '经营驾驶舱')
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

  function setStorageMode(mode: StorageMode) {
    storageMode.value = mode
  }

  const avgTemp = computed(() => {
    const sum = clusters.value.reduce((a, c) => a + c.temp, 0)
    return sum / clusters.value.length
  })

  const riskCount = computed(() => clusters.value.filter(c => c.status === 'risk').length)

  const balancingCount = computed(() => clusters.value.filter(c => c.balance === '均衡中').length)

  function domainEvents(key: string) {
    return eventFeeds[key] ?? []
  }

  const busyRate = computed(() => {
    const total = lanes.value.reduce((a, l) => a + l.total, 0)
    const occupied = lanes.value.reduce((a, l) => a + l.occupied, 0)
    return Math.round((occupied / total) * 100)
  })

  const avgLoopQuality = computed(() => Math.round(loops.value.reduce((a, l) => a + l.quality, 0) / loops.value.length))

  const avgLoopLatency = computed(() => Math.round(loops.value.reduce((a, l) => a + l.latency, 0) / loops.value.length))

  const storageLatency = ref(19)
  const storageQuality = ref(99)
  const storageLoss = ref(0.1)

  return {
    storageMode, avgSoc, avgSoh, totalPower, socSeries, powerSeries, sites,
    clusters, tempSeries, riskCount, avgTemp, balancingCount,
    activeSessions, queueVehicles, lanes, sessionSeries, busyRate,
    totalEnergy, energySeries, loops, avgLoopQuality, avgLoopLatency,
    pendingAmount, settleRate, ledgerEntries,
    todayRevenue, revenueSeries,
    domainEvents, currentTime,
    storageLatency, storageQuality, storageLoss,
    start, stop, setStorageMode,
  }
})
