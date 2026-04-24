import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
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

router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('token')
  const requiresAuth = to.meta.requiresAuth !== false

  if (requiresAuth && !token) {
    next('/login')
  } else if (to.path === '/login' && token) {
    next('/dashboard')
  } else {
    next()
  }
})

export default router
