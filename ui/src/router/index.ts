import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/LoginView.vue'),
      meta: { requiresAuth: false },
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
          meta: { title: '总览', icon: 'LayoutDashboard' },
        },
        {
          path: 'middleware',
          name: 'Middleware',
          component: () => import('@/views/MiddlewareView.vue'),
          meta: { title: '中间件连接', icon: 'Network' },
        },
        {
          path: 'nodes',
          name: 'NodeList',
          component: () => import('@/views/NodeListView.vue'),
          meta: { title: '节点管理', icon: 'Server' },
        },
        {
          path: 'nodes/:nodeId/devices',
          name: 'DeviceList',
          component: () => import('@/views/DeviceListView.vue'),
          meta: { title: '设备列表', hidden: true },
        },
        {
          path: 'nodes/:nodeId/devices/:deviceId/points',
          name: 'PointList',
          component: () => import('@/views/PointListView.vue'),
          meta: { title: '物模型点位', hidden: true },
        },
        {
          path: 'control',
          name: 'Control',
          component: () => import('@/views/ControlView.vue'),
          meta: { title: '设备控制', icon: 'Sliders' },
        },
        {
          path: 'alerts',
          name: 'Alerts',
          component: () => import('@/views/AlertListView.vue'),
          meta: { title: '告警管理', icon: 'Bell' },
        },
        {
          path: 'settings',
          name: 'Settings',
          component: () => import('@/views/SettingsView.vue'),
          meta: { title: '系统设置', icon: 'Settings' },
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
