import type { Component } from 'vue'
import {
  Activity,
  AlertTriangle,
  BarChart3,
  BatteryCharging,
  Blocks,
  Cable,
  Cpu,
  FileCode2,
  Fuel,
  LayoutDashboard,
  Radio,
  Receipt,
  Server,
  Settings,
  Sliders,
  Sparkles,
  Workflow,
} from 'lucide-vue-next'

export interface NavItem {
  key: string
  label: string
  path: string
  icon: Component
  group: string
  badge?: 'alerts'
  children?: NavItem[]
}

export interface NavSection {
  key: string
  label: string
  items: NavItem[]
}

export const navSections: NavSection[] = [
  {
    key: 'operations',
    label: '采集运行',
    items: [
      { key: 'dashboard', label: '系统总览', path: '/dashboard', icon: LayoutDashboard, group: 'operations' },
      { key: 'middleware', label: '消息总线', path: '/middleware', icon: Radio, group: 'operations' },
      { key: 'nodes', label: '节点管理', path: '/nodes', icon: Server, group: 'operations' },
      { key: 'control', label: '设备控制', path: '/control', icon: Sliders, group: 'operations' },
      { key: 'alerts', label: '告警管理', path: '/alerts', icon: AlertTriangle, group: 'operations', badge: 'alerts' },
      { key: 'settings', label: '系统设置', path: '/settings', icon: Settings, group: 'operations' },
    ],
  },
  {
    key: 'business',
    label: '业务扩展',
    items: [
      {
        key: 'business-center',
        label: '业务中心',
        path: '/business-center',
        icon: BarChart3,
        group: 'business',
        children: [
          { key: 'energy-storage', label: '储能管理', path: '/business-center/energy-storage', icon: BatteryCharging, group: 'business' },
          { key: 'power-bms', label: '电源BMS', path: '/business-center/power-bms', icon: Fuel, group: 'business' },
          { key: 'charging', label: '充电管理', path: '/business-center/charging', icon: Cable, group: 'business' },
          { key: 'energy-monitoring', label: '能耗监测', path: '/business-center/energy-monitoring', icon: Activity, group: 'business' },
          { key: 'ledger', label: '账务台账', path: '/business-center/ledger', icon: Receipt, group: 'business' },
        ],
      },
    ],
  },
  {
    key: 'group-control',
    label: '群控编排',
    items: [
      {
        key: 'group-control',
        label: '群控管理',
        path: '/group-control',
        icon: Workflow,
        group: 'group-control',
        children: [
          { key: 'node-scheduling', label: '节点调度', path: '/group-control/node-scheduling', icon: Blocks, group: 'group-control' },
          { key: 'scenario-linkage', label: '场景联动', path: '/group-control/scenario-linkage', icon: Sparkles, group: 'group-control' },
          { key: 'function-execution', label: '函数执行', path: '/group-control/function-execution', icon: Cpu, group: 'group-control' },
          { key: 'script-orchestration', label: '脚本编排', path: '/group-control/script-orchestration', icon: FileCode2, group: 'group-control' },
        ],
      },
    ],
  },
]

export function isPathActive(targetPath: string, currentPath: string): boolean {
  if (targetPath === '/dashboard') return currentPath === '/dashboard'
  return currentPath === targetPath || currentPath.startsWith(`${targetPath}/`)
}

export function flattenNavItems(items: NavItem[] = navSections.flatMap(section => section.items)): NavItem[] {
  return items.flatMap(item => [item, ...(item.children ? flattenNavItems(item.children) : [])])
}

export function findNavItemByPath(path: string): NavItem | undefined {
  return flattenNavItems().find(item => item.path === path)
}
