import { createRouter, createWebHistory } from 'vue-router'
import { installApi } from '@/api/install'

// 安装状态缓存：首次检查后缓存于内存，页面刷新（含服务重启后）重新检查
// Install status cache: checked once per page load, refreshed after a page reload
let installStatusChecked = false
let systemInstalled = true

async function ensureInstallStatus(): Promise<boolean> {
  if (installStatusChecked) return systemInstalled
  try {
    const st = await installApi.status()
    systemInstalled = st.is_installed
  } catch {
    systemInstalled = true // 状态检查失败时保守放行到登录页
  }
  installStatusChecked = true
  return systemInstalled
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/install',
      name: 'Install',
      component: () => import('@/views/Install.vue'),
      meta: { requiresAuth: false, title: '系统初始化安装' },
    },
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/LoginView.vue'),
      meta: { requiresAuth: false, title: '登录' },
    },
    {
      path: '/',
      component: () => import('@/components/layout/AppLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          redirect: '/dashboard',
        },
        {
          path: 'dashboard',
          name: 'Dashboard',
          component: () => import('@/views/DashboardView.vue'),
          meta: { title: '系统总览', sectionTitle: '采集运行' },
        },
        {
          path: 'middleware',
          name: 'Middleware',
          component: () => import('@/views/MiddlewareView.vue'),
          meta: { title: '消息总线', sectionTitle: '采集运行' },
        },
        {
          path: 'nodes',
          name: 'NodeList',
          component: () => import('@/views/NodeListView.vue'),
          meta: { title: '节点管理', sectionTitle: '采集运行' },
        },
        {
          path: 'topology',
          name: 'SpatialTopology',
          component: () => import('@/views/SpatialTreeView.vue'),
          meta: { title: '空间拓扑', sectionTitle: '采集运行' },
        },
        {
          path: 'nodes/:nodeId/devices',
          name: 'DeviceList',
          component: () => import('@/views/DeviceListView.vue'),
          meta: { title: '子设备列表', sectionTitle: '采集运行', hidden: true },
        },
        {
          path: 'nodes/:nodeId/devices/:deviceId/points',
          name: 'PointList',
          component: () => import('@/views/PointListView.vue'),
          meta: { title: '物模型点位', sectionTitle: '采集运行', hidden: true },
        },
        {
          path: 'control',
          name: 'Control',
          component: () => import('@/views/ControlView.vue'),
          meta: { title: '设备控制', sectionTitle: '采集运行' },
        },
        {
          path: 'alerts',
          name: 'Alerts',
          component: () => import('@/views/AlertListView.vue'),
          meta: { title: '告警管理', sectionTitle: '采集运行' },
        },
        {
          path: 'settings',
          name: 'Settings',
          component: () => import('@/views/SettingsView.vue'),
          meta: { title: '系统设置', sectionTitle: '采集运行' },
        },
        {
          path: 'ean',
          name: 'EanOverview',
          component: () => import('@/views/ean/EanOverviewView.vue'),
          meta: { title: 'EAN 协调中心', sectionTitle: 'EAN 协调', pageKey: 'ean-center', parentTitle: 'EAN 协调中心', parentPath: '/ean' },
        },
        {
          path: 'ean/agents',
          name: 'EanAgents',
          component: () => import('@/views/ean/EanAgentsView.vue'),
          meta: { title: 'Agent 管理', sectionTitle: 'EAN 协调', pageKey: 'ean-agents', parentTitle: 'EAN 协调中心', parentPath: '/ean' },
        },
        {
          path: 'ean/invoke',
          name: 'EanInvoke',
          component: () => import('@/views/ean/EanInvokeView.vue'),
          meta: { title: '能力调用', sectionTitle: 'EAN 协调', pageKey: 'ean-invoke', parentTitle: 'EAN 协调中心', parentPath: '/ean' },
        },
        {
          path: 'ean/events',
          name: 'EanEvents',
          component: () => import('@/views/ean/EanEventsView.vue'),
          meta: { title: '事件流', sectionTitle: 'EAN 协调', pageKey: 'ean-events', parentTitle: 'EAN 协调中心', parentPath: '/ean' },
        },
        {
          path: 'ean/debug',
          name: 'EanDebugHelp',
          component: () => import('@/views/ean/EanDebugHelpView.vue'),
          meta: { title: '联合调试帮助', sectionTitle: 'EAN 协调', pageKey: 'ean-debug', parentTitle: 'EAN 协调中心', parentPath: '/ean' },
        },
        {
          path: 'business-center',
          name: 'BusinessCenter',
          component: () => import('@/views/business/BusinessView.vue'),
          meta: { title: '业务中心总览', sectionTitle: '业务扩展', pageKey: 'business-center', parentTitle: '业务中心', parentPath: '/business-center' },
        },
        {
          path: 'business-center/energy-storage',
          name: 'EnergyStorage',
          component: () => import('@/views/business/EnergyStorageView.vue'),
          meta: { title: '储能管理', sectionTitle: '业务扩展', pageKey: 'energy-storage', parentTitle: '业务中心', parentPath: '/business-center' },
        },
        {
          path: 'business-center/power-bms',
          name: 'PowerBms',
          component: () => import('@/views/business/PowerBMSView.vue'),
          meta: { title: '电源BMS', sectionTitle: '业务扩展', pageKey: 'power-bms', parentTitle: '业务中心', parentPath: '/business-center' },
        },
        {
          path: 'business-center/charging',
          name: 'Charging',
          component: () => import('@/views/business/ChargingView.vue'),
          meta: { title: '充电管理', sectionTitle: '业务扩展', pageKey: 'charging', parentTitle: '业务中心', parentPath: '/business-center' },
        },
        {
          path: 'business-center/energy-monitoring',
          name: 'EnergyMonitoring',
          component: () => import('@/views/business/EnergyMonitorView.vue'),
          meta: { title: '能耗监测', sectionTitle: '业务扩展', pageKey: 'energy-monitoring', parentTitle: '业务中心', parentPath: '/business-center' },
        },
        {
          path: 'business-center/ledger',
          name: 'Ledger',
          component: () => import('@/views/business/AccountingView.vue'),
          meta: { title: '账务台账', sectionTitle: '业务扩展', pageKey: 'ledger', parentTitle: '业务中心', parentPath: '/business-center' },
        },
        {
          path: 'visual',
          name: 'VisualCenter',
          component: () => import('@/views/visual/VisualCenterView.vue'),
          meta: { title: '可视化中心总览', sectionTitle: '数据可视化', pageKey: 'visual-center', parentTitle: '可视化中心', parentPath: '/visual' },
        },
        {
          path: 'visual/storage-station',
          name: 'StorageStation',
          component: () => import('@/views/visual/StorageStationView.vue'),
          meta: { title: '储能电站', sectionTitle: '数据可视化', pageKey: 'visual-storage', parentTitle: '可视化中心', parentPath: '/visual' },
        },
        {
          path: 'visual/production-line',
          name: 'ProductionLine',
          component: () => import('@/views/visual/ProductionLineView.vue'),
          meta: { title: '产线展示', sectionTitle: '数据可视化', pageKey: 'visual-production-line', parentTitle: '可视化中心', parentPath: '/visual' },
        },
        {
          path: 'visual/industrial-screen',
          name: 'IndustrialScreen',
          component: () => import('@/views/visual/IndustrialScreenView.vue'),
          meta: { title: '工业大屏', sectionTitle: '数据可视化', pageKey: 'visual-industrial-screen', parentTitle: '可视化中心', parentPath: '/visual' },
        },
        {
          path: 'visual/instruments',
          name: 'Instruments',
          component: () => import('@/views/visual/InstrumentsView.vue'),
          meta: { title: '仪表监控', sectionTitle: '数据可视化', pageKey: 'visual-instruments', parentTitle: '可视化中心', parentPath: '/visual' },
        },
        {
          path: 'visual/data-center',
          name: 'DataCenter',
          component: () => import('@/views/visual/DataCenterView.vue'),
          meta: { title: '数据中心仿真', sectionTitle: '数据可视化', pageKey: 'visual-data-center', parentTitle: '可视化中心', parentPath: '/visual' },
        },
        {
          path: 'visual/power-distribution',
          name: 'PowerDistribution',
          component: () => import('@/views/visual/PowerDistributionView.vue'),
          meta: { title: '输配电', sectionTitle: '数据可视化', pageKey: 'visual-power', parentTitle: '可视化中心', parentPath: '/visual' },
        },
        {
          path: 'visual/port',
          name: 'Port',
          component: () => import('@/views/visual/PortView.vue'),
          meta: { title: '港口运输', sectionTitle: '数据可视化', pageKey: 'visual-port', parentTitle: '可视化中心', parentPath: '/visual' },
        },
        {
          path: 'group-control',
          name: 'GroupControl',
          component: () => import('@/views/group-control/GroupControlView.vue'),
          meta: { title: '群控管理总览', sectionTitle: '群控编排', pageKey: 'group-control', parentTitle: '群控管理', parentPath: '/group-control' },
        },
        {
          path: 'group-control/node-scheduling',
          name: 'NodeScheduling',
          component: () => import('@/views/group-control/NodeSchedulingView.vue'),
          meta: { title: '节点调度', sectionTitle: '群控编排', pageKey: 'node-scheduling', parentTitle: '群控管理', parentPath: '/group-control' },
        },
        {
          path: 'group-control/scenario-linkage',
          name: 'ScenarioLinkage',
          component: () => import('@/views/group-control/SceneLinkageView.vue'),
          meta: { title: '场景联动', sectionTitle: '群控编排', pageKey: 'scenario-linkage', parentTitle: '群控管理', parentPath: '/group-control' },
        },
        {
          path: 'group-control/function-execution',
          name: 'FunctionExecution',
          component: () => import('@/views/group-control/FunctionExecutionView.vue'),
          meta: { title: '函数执行', sectionTitle: '群控编排', pageKey: 'function-execution', parentTitle: '群控管理', parentPath: '/group-control' },
        },
        {
          path: 'group-control/script-orchestration',
          name: 'ScriptOrchestration',
          component: () => import('@/views/group-control/ScriptOrchestrationView.vue'),
          meta: { title: '脚本编排', sectionTitle: '群控编排', pageKey: 'script-orchestration', parentTitle: '群控管理', parentPath: '/group-control' },
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/dashboard',
    },
  ],
})

router.beforeEach(async (to, _from, next) => {
  const token = localStorage.getItem('token')
  const requiresAuth = to.meta.requiresAuth !== false

  // 安装引导页：已有配置则不得再次进入；未安装则放行
  if (to.path === '/install') {
    if (token) {
      next('/dashboard')
      return
    }
    const installed = await ensureInstallStatus()
    next(installed ? '/login' : true)
    return
  }

  if (requiresAuth && !token) {
    // 无令牌访问受保护页面：无数据库配置时必须进入安装引导，否则去登录页
    const installed = await ensureInstallStatus()
    next(installed ? '/login' : '/install')
  } else if (to.path === '/login' && token) {
    next('/dashboard')
  } else {
    next()
  }
})

export default router
