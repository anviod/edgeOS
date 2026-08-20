import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { formatDateTime } from '@/lib/live-time'

// =====================================================
// 数据可视化实时仿真 Store
// 为储能电站 / 产线展示 / 数据中心 / 输配电 / 工业大屏 / 仪表监控 提供统一动效数据源
// =====================================================

export type MachineStatus = 'running' | 'standby' | 'warn' | 'fault'

export type StorageMode = 'charge' | 'discharge' | 'idle'

export interface MachineSim {
  id: string
  name: string
  col: number
  row: number
  status: MachineStatus
  oee: number
  rate: number
  temp: number
}

export interface AgvSim {
  id: string
  label: string
  x: number
  y: number
  targetX: number
  targetY: number
  load: boolean
  color: string
}

export interface VisEvent {
  title: string
  subtitle: string
  meta: string
  status: 'info' | 'warning' | 'error' | 'success'
}

export interface GaugeSim {
  key: string
  label: string
  value: number
  unit: string
  min: number
  max: number
  color: string
}

export type FeederStatus = 'normal' | 'warn' | 'fault'

export interface FeederSim {
  id: string
  name: string
  voltage: number
  current: number
  load: number
  status: FeederStatus
  color: string
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

const EVENTS: Omit<VisEvent, 'meta'>[] = [
  { title: '储能电站进入放电模式', subtitle: '削峰放电执行中，SOC 缓降', status: 'success' },
  { title: '2 号储能柜温度回升', subtitle: '风冷启动，温度回落至 33.1℃', status: 'warning' },
  { title: 'PCS 功率指令响应', subtitle: '2 号变流器跟随正常', status: 'info' },
  { title: '1 号线 3 号机 OEE 下滑', subtitle: '换型停机 4 分钟，自动恢复', status: 'warning' },
  { title: 'AGV-07 已到达卸货工位', subtitle: '转运任务完成，等待分配', status: 'success' },
  { title: '空压机温度回升', subtitle: '冷却回路自检通过，回落至 41.2℃', status: 'success' },
  { title: '2 号线输送带降速', subtitle: '前端物料积压，节拍自动放缓', status: 'warning' },
  { title: '仪表压力超限', subtitle: 'P-102 达 0.82MPa，接近上限', status: 'error' },
  { title: '质检工位通过率波动', subtitle: '近 10 分钟良率 98.4%', status: 'info' },
  { title: '第 3 号泊位船舶靠泊', subtitle: '完成系缆，已进入卸船作业', status: 'success' },
  { title: '岸桥 Q7 作业提速', subtitle: '卸船效率达 42 箱/时', status: 'info' },
  { title: '闸口集卡排队上升', subtitle: '当前排队 12 辆，预计等待 9 分钟', status: 'warning' },
  { title: '集装箱堆场翻倒作业', subtitle: 'Y18 贝位倒箱 6 个，效率正常', status: 'info' },
]

function nowMeta(tag: string) {
  return `${formatDateTime(new Date())} / ${tag}`
}

export const useVisualStore = defineStore('visual', () => {
  // ========== 储能电站 ==========
  const mode = ref<StorageMode>('discharge')
  const power = ref(41.8)          // MW，放电为正 / 充电为负
  const soc = ref(72)              // %
  const soh = ref(93)              // %
  const capacity = ref(126.8)      // MWh 可用容量
  const batteryTemp = ref(31.5)    // ℃
  const cycles = ref(1824)         // 循环次数
  const powerSeries = ref(buildSeries(41.8, 30, 3, 8, 68))
  const socSeries = ref(buildSeries(72, 30, 1.2, 50, 90))
  const energySeries = ref(buildSeries(126.8, 30, 2, 96, 150))

  function setMode(m: StorageMode) {
    mode.value = m
  }

  // ========== 产线展示 ==========
  const lineSpeed = ref(82)
  const okRate = ref(98.2)
  const produced = ref(12680)
  const wip = ref(36)
  const lineOee = ref(87.6)
  const lineSeries = ref(buildSeries(82, 30, 5, 62, 96))
  const okSeries = ref(buildSeries(98.2, 30, 0.4, 96, 99.8))

  const machines = ref<MachineSim[]>([
    { id: 'M1', name: '1 号线 1 号机', col: 0, row: 2, status: 'running', oee: 92.4, rate: 46, temp: 58.2 },
    { id: 'M2', name: '1 号线 2 号机', col: 1, row: 2, status: 'running', oee: 91.8, rate: 45, temp: 61.5 },
    { id: 'M3', name: '1 号线 3 号机', col: 2, row: 2, status: 'warn', oee: 84.6, rate: 41, temp: 74.9 },
    { id: 'M4', name: '2 号线 1 号机', col: 6, row: 4, status: 'running', oee: 93.1, rate: 47, temp: 56.8 },
    { id: 'M5', name: '2 号线 2 号机', col: 7, row: 4, status: 'standby', oee: 78.2, rate: 33, temp: 42.1 },
  ])

  const agvs = ref<AgvSim[]>([
    { id: 'AGV-01', label: 'AGV-01', x: 140, y: 210, targetX: 200, targetY: 240, load: true, color: '#0EA5E9' },
    { id: 'AGV-02', label: 'AGV-02', x: 320, y: 150, targetX: 360, targetY: 120, load: false, color: '#8B5CF6' },
  ])

  // ========== 工业大屏 ==========
  const powerLoad = ref(426.8)
  const totalEnergy = ref(5820)
  const activeDevices = ref(1286)
  const alarmCount = ref(7)
  const deviceOnline = ref(98.6)
  const loadSeries = ref(buildSeries(426, 36, 18, 360, 520))
  const energyTrend = ref(buildSeries(60, 36, 3, 42, 78))
  const qualitySeries = ref(buildSeries(97, 36, 1.2, 92, 99.8))
  const alarmSeries = ref([3, 5, 4, 6, 8, 7, 9, 7, 5, 6, 7, 7, 9, 8, 10, 7, 6, 8, 7, 9, 8, 7, 8, 7])

  // ========== 仪表监控 ==========
  const gauges = ref<GaugeSim[]>([
    { key: 'pressure', label: '管道压力', value: 0.62, unit: 'MPa', min: 0, max: 1, color: '#0EA5E9' },
    { key: 'temp', label: '炉膛温度', value: 458, unit: '℃', min: 0, max: 800, color: '#F59E0B' },
    { key: 'flow', label: '瞬时流量', value: 126.4, unit: 'm³/h', min: 0, max: 260, color: '#10B981' },
    { key: 'level', label: '储罐液位', value: 74, unit: '%', min: 0, max: 100, color: '#8B5CF6' },
    { key: 'current', label: '主回路电流', value: 386, unit: 'A', min: 0, max: 600, color: '#EC4899' },
    { key: 'voltage', label: '母线电压', value: 401.2, unit: 'V', min: 340, max: 460, color: '#38BDF8' },
    { key: 'vibration', label: '电机振动', value: 2.8, unit: 'mm/s', min: 0, max: 7.1, color: '#F59E0B' },
    { key: 'humidity', label: '车间湿度', value: 46, unit: '%RH', min: 0, max: 100, color: '#34D399' },
  ])
  const pressureSeries = ref(buildSeries(0.62, 30, 0.03, 0.45, 0.85))

  // ========== 数据中心 ==========
  const rackCount = ref(42)
  const gpuLoad = ref(64)
  const cpuLoad = ref(58)
  const pue = ref(1.28)
  const dcPower = ref(824.6)
  const dcTemp = ref(24.6)
  const dcCooling = ref(68)
  const networkThroughput = ref(186)
  const netSeries = ref(buildSeries(186, 30, 14, 120, 260))
  const pueSeries = ref(buildSeries(1.28, 30, 0.02, 1.15, 1.45))
  const dcPowerSeries = ref(buildSeries(824.6, 30, 22, 660, 980))

  // ========== 输配电 ==========
  const gridLoad = ref(1268)
  const gridFreq = ref(50.01)
  const busVoltage = ref(10.5)
  const powerFactor = ref(0.96)
  const feeders = ref<FeederSim[]>([
    { id: 'F1', name: '储能电站馈线', voltage: 10, current: 386, load: 62, status: 'normal', color: '#10B981' },
    { id: 'F2', name: '数据中心馈线', voltage: 10, current: 512, load: 81, status: 'normal', color: '#0EA5E9' },
    { id: 'F3', name: '产线馈线', voltage: 10, current: 428, load: 74, status: 'warn', color: '#F59E0B' },
    { id: 'F4', name: '充电站馈线', voltage: 10, current: 264, load: 51, status: 'normal', color: '#8B5CF6' },
  ])
  const gridLoadSeries = ref(buildSeries(1268, 30, 40, 980, 1500))
  const freqSeries = ref(buildSeries(50.01, 30, 0.01, 49.95, 50.05))

  // ========== 港口运输 ==========
  const teuToday = ref(12840)          // 今日吞吐 TEU
  const teuPerHour = ref(486)          // 吞吐 TEU/h
  const shipsInPort = ref(3)           // 在港船舶
  const berthOccupancy = ref(72)       // 泊位利用率 %
  const cranesWorking = ref(6)         // 作业岸桥
  const craneUtil = ref(84)            // 岸桥作业率 %
  const trucksIn = ref(0)              // 累计进港集卡
  const trucksQueued = ref(8)          // 闸口排队
  const truckFlow = ref(126)           // 集卡流量 辆/h
  const teuSeries = ref(buildSeries(486, 30, 30, 380, 620))
  const throughputSeries = ref(buildSeries(70, 30, 4, 54, 92))
  const berthSeries = ref(buildSeries(72, 30, 5, 55, 92))

  // ========== 事件流 ==========
  const events = ref<VisEvent[]>([
    { title: '储能电站放电指令生效', subtitle: 'PCS 输出 41.8MW，SOC 缓降', meta: nowMeta('储能电站'), status: 'success' },
    { title: '2 号储能柜温控正常', subtitle: '风冷回路运行，柜温 33.1℃', meta: nowMeta('储能电站'), status: 'info' },
    { title: '母线电压轻微波动', subtitle: '401.2V，区间内正常', meta: nowMeta('仪表监控'), status: 'info' },
  ])

  let timer: ReturnType<typeof setInterval> | null = null

  function pushEvent(tag: string) {
    const pool = EVENTS
    const item = pool[Math.floor(Math.random() * pool.length)]
    events.value.unshift({ ...item, meta: nowMeta(tag) })
    if (events.value.length > 7) events.value.pop()
  }

  function stepAgv() {
    const BOUNDS = { minX: 30, maxX: 430, minY: 90, maxY: 300 }
    agvs.value = agvs.value.map(agv => {
      const dx = agv.targetX - agv.x
      const dy = agv.targetY - agv.y
      const dist = Math.hypot(dx, dy)
      const step = 12
      if (dist <= step) {
        // 到达目标，随机换向（限定在场景范围内）
        const nx = clamp(agv.x + (Math.random() > 0.5 ? 1 : -1) * (90 + Math.random() * 70), BOUNDS.minX, BOUNDS.maxX)
        const ny = clamp(agv.y + (Math.random() > 0.5 ? 1 : -1) * (50 + Math.random() * 60), BOUNDS.minY, BOUNDS.maxY)
        return {
          ...agv,
          x: agv.targetX,
          y: agv.targetY,
          targetX: nx,
          targetY: ny,
          load: Math.random() > 0.5,
        }
      }
      return { ...agv, x: agv.x + (dx / dist) * step, y: agv.y + (dy / dist) * step }
    })
  }

  function tick() {
    // 储能电站
    const powerTarget = mode.value === 'discharge' ? 41.8 : mode.value === 'charge' ? -32 : 0
    power.value = clamp(power.value + (powerTarget - power.value) * 0.18 + drift(0, 2, -2.5, 2.5), -68, 68)
    soc.value = clamp(soc.value + (mode.value === 'charge' ? 0.35 : mode.value === 'discharge' ? -0.35 : drift(0, 0.15, -0.25, 0.25)), 15, 98)
    soh.value = drift(soh.value, 0.05, 82, 99)
    capacity.value = clamp(capacity.value + drift(0, 1.2, -1.4, 1.4), 96, 150)
    batteryTemp.value = drift(batteryTemp.value, 0.3, 26, 42)
    cycles.value = cycles.value + drift(0.01, 0.01, 0, 0.02)
    powerSeries.value = [...powerSeries.value.slice(1), power.value]
    socSeries.value = [...socSeries.value.slice(1), soc.value]
    energySeries.value = [...energySeries.value.slice(1), capacity.value]

    // 产线
    lineSpeed.value = Math.round(clamp(lineSpeed.value + drift(0, 2.4, -3, 3), 62, 96))
    okRate.value = clamp(okRate.value + drift(0, 0.18, -0.3, 0.3), 96, 99.8)
    produced.value += Math.round(drift(lineSpeed.value, 2, 40, 80))
    wip.value = jitterInt(wip.value, 3, 18, 58)
    lineOee.value = clamp(lineOee.value + drift(0, 0.5, -0.8, 0.8), 80, 96)
    machines.value = machines.value.map(m => ({
      ...m,
      oee: clamp(m.oee + drift(0, 0.8, -1.4, 1.4), 60, 99),
      rate: Math.round(clamp(m.rate + drift(0, 1.2, -2, 2), 24, 60)),
      temp: clamp(m.temp + drift(0, 1.2, -2, 2), 38, 88),
      status: m.status === 'warn' && m.temp < 70 ? 'running' : m.temp > 78 ? 'warn' : m.temp > 84 ? 'fault' : m.status,
    }))
    lineSeries.value = [...lineSeries.value.slice(1), lineSpeed.value]
    okSeries.value = [...okSeries.value.slice(1), okRate.value]
    stepAgv()

    // 大屏
    powerLoad.value = clamp(powerLoad.value + drift(0, 9, -16, 16), 320, 560)
    totalEnergy.value += drift(0.4, 0.2, 0.05, 0.9)
    activeDevices.value = jitterInt(activeDevices.value, 4, 1210, 1330)
    alarmCount.value = jitterInt(alarmCount.value, 1, 2, 12)
    deviceOnline.value = clamp(deviceOnline.value + drift(0, 0.1, -0.2, 0.2), 96.5, 99.9)
    loadSeries.value = [...loadSeries.value.slice(1), powerLoad.value]
    energyTrend.value = [...energyTrend.value.slice(1), clamp(energyTrend.value[energyTrend.value.length - 1] + drift(0, 3, -4, 4), 40, 80)]
    qualitySeries.value = [...qualitySeries.value.slice(1), deviceOnline.value]
    alarmSeries.value = [...alarmSeries.value.slice(1), alarmCount.value]

    // 仪表
    gauges.value = gauges.value.map(g => ({
      ...g,
      value: clamp(g.value + drift(0, (g.max - g.min) * 0.025, -(g.max - g.min) * 0.035, (g.max - g.min) * 0.035), g.min, g.max),
    }))
    const pressureGauge = gauges.value.find(g => g.key === 'pressure')
    if (pressureGauge) pressureSeries.value = [...pressureSeries.value.slice(1), pressureGauge.value]

    // 数据中心
    gpuLoad.value = jitterInt(gpuLoad.value, 5, 28, 92)
    cpuLoad.value = jitterInt(cpuLoad.value, 4, 26, 88)
    pue.value = clamp(pue.value + drift(0, 0.01, -0.02, 0.02), 1.12, 1.48)
    dcPower.value = clamp(dcPower.value + drift(0, 18, -30, 30), 640, 1000)
    dcTemp.value = drift(dcTemp.value, 0.3, 21, 29)
    dcCooling.value = jitterInt(dcCooling.value, 3, 42, 92)
    networkThroughput.value = jitterInt(networkThroughput.value, 16, 96, 280)
    netSeries.value = [...netSeries.value.slice(1), networkThroughput.value]
    pueSeries.value = [...pueSeries.value.slice(1), pue.value]
    dcPowerSeries.value = [...dcPowerSeries.value.slice(1), dcPower.value]

    // 输配电
    gridLoad.value = clamp(gridLoad.value + drift(0, 28, -46, 46), 940, 1560)
    gridFreq.value = clamp(gridFreq.value + drift(0, 0.008, -0.012, 0.012), 49.94, 50.06)
    busVoltage.value = clamp(busVoltage.value + drift(0, 0.03, -0.05, 0.05), 10.3, 10.7)
    powerFactor.value = clamp(powerFactor.value + drift(0, 0.004, -0.006, 0.006), 0.92, 0.99)
    feeders.value = feeders.value.map(f => ({
      ...f,
      current: Math.round(clamp(f.current + drift(0, 18, -26, 26), 120, 720)),
      load: clamp(f.load + drift(0, 2.4, -3.4, 3.4), 28, 96),
      status: f.load > 92 ? 'fault' : f.load > 80 ? 'warn' : f.status === 'fault' && f.load < 75 ? 'normal' : f.status,
    }))
    gridLoadSeries.value = [...gridLoadSeries.value.slice(1), gridLoad.value]
    freqSeries.value = [...freqSeries.value.slice(1), gridFreq.value]

    // 港口运输
    teuToday.value += Math.round(drift(8, 4, 2, 16))
    teuPerHour.value = jitterInt(teuPerHour.value, 24, 360, 640)
    shipsInPort.value = jitterInt(shipsInPort.value, 1, 2, 5)
    berthOccupancy.value = clamp(berthOccupancy.value + drift(0, 3, -4, 4), 55, 96)
    cranesWorking.value = jitterInt(cranesWorking.value, 1, 4, 8)
    craneUtil.value = clamp(craneUtil.value + drift(0, 1.6, -2.4, 2.4), 62, 97)
    trucksIn.value += jitterInt(1, 1, 0, 4)
    trucksQueued.value = jitterInt(trucksQueued.value, 1, 3, 18)
    truckFlow.value = jitterInt(truckFlow.value, 12, 80, 190)
    teuSeries.value = [...teuSeries.value.slice(1), teuPerHour.value]
    throughputSeries.value = [...throughputSeries.value.slice(1), clamp(throughputSeries.value[throughputSeries.value.length - 1] + drift(0, 2.4, -3.2, 3.2), 50, 96)]
    berthSeries.value = [...berthSeries.value.slice(1), berthOccupancy.value]

    if (Math.random() < 0.16) pushEvent('数据可视化')
  }

  function start() {
    if (timer) return
    timer = setInterval(tick, 1400)
  }

  function stop() {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  const avgOee = computed(() => {
    const sum = machines.value.reduce((a, m) => a + m.oee, 0)
    return Math.round((sum / machines.value.length) * 10) / 10
  })

  const warnCount = computed(() => machines.value.filter(m => m.status === 'warn' || m.status === 'fault').length)

  const busyGauges = computed(() => gauges.value.filter(g => g.value / g.max >= 0.85).length)

  return {
    // storage
    mode, power, soc, soh, capacity, batteryTemp, cycles,
    powerSeries, socSeries, energySeries, setMode,
    // production
    lineSpeed, okRate, produced, wip, lineOee, lineSeries, okSeries, machines, agvs, avgOee, warnCount,
    // screen
    powerLoad, totalEnergy, activeDevices, alarmCount, deviceOnline, loadSeries, energyTrend, qualitySeries, alarmSeries,
    // instruments
    gauges, busyGauges, pressureSeries,
    // data center
    rackCount, gpuLoad, cpuLoad, pue, dcPower, dcTemp, dcCooling, networkThroughput,
    netSeries, pueSeries, dcPowerSeries,
    // power distribution
    gridLoad, gridFreq, busVoltage, powerFactor, feeders, gridLoadSeries, freqSeries,
    // port
    teuToday, teuPerHour, shipsInPort, berthOccupancy, cranesWorking, craneUtil,
    trucksIn, trucksQueued, truckFlow, teuSeries, throughputSeries, berthSeries,
    // events
    events,
    start, stop,
  }
})
