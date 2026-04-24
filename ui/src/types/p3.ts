export type P3Tone = 'sky' | 'emerald' | 'amber' | 'red' | 'violet' | 'slate'

export type P3RiskLevel = 'L1' | 'L2' | 'L3' | 'L4' | 'L5'

export interface P3StatusStrip {
  system: string
  devices: number
  alarms: number
  latency: number
  loss: number
  quality: number
}

export interface P3KpiItem {
  label: string
  value: string | number
  unit?: string
  delta?: number
  note: string
  tone: P3Tone
}

export interface P3HealthMetric {
  label: string
  value: string | number
  unit?: string
  hint: string
  trend?: number
  status: 'running' | 'standby' | 'fault' | 'unknown'
}

export interface P3TrendCardData {
  title: string
  summary: string
  latency: number
  loss: number
  quality: number
  points: number[]
  status: 'running' | 'standby' | 'fault' | 'unknown'
}

export interface P3TableColumn {
  key: string
  label: string
  align?: 'left' | 'center' | 'right'
}

export interface P3TableRow {
  id: string
  status?: string
  accent?: P3Tone
  [key: string]: string | number | undefined
}

export interface P3TableData {
  title: string
  description: string
  columns: P3TableColumn[]
  rows: P3TableRow[]
}

export interface P3SideEvent {
  title: string
  subtitle: string
  meta: string
  status: string
}

export interface P3FlowNode {
  title: string
  subtitle: string
  status: 'running' | 'standby' | 'fault' | 'unknown'
}

export interface P3TimelineItem {
  title: string
  detail: string
  timestamp: string
  result?: string
}

export interface P3ActionPreset {
  label: string
  description: string
  level: P3RiskLevel
  tone: P3Tone
  confirmText?: string
}

export interface P3AuditRecord {
  user: string
  action: string
  target: string
  timestamp: string
  result: string
}

export interface P3PageData {
  key: string
  title: string
  subtitle: string
  section: string
  module: string
  statusStrip: P3StatusStrip
  kpis: P3KpiItem[]
  metrics: P3HealthMetric[]
  trends: P3TrendCardData[]
  mainTable: P3TableData
  sidePanelTitle: string
  sideEvents: P3SideEvent[]
  flowTitle: string
  flowNodes: P3FlowNode[]
  timelineTitle: string
  timeline: P3TimelineItem[]
  actionTitle: string
  actions: P3ActionPreset[]
  auditRecords: P3AuditRecord[]
}
