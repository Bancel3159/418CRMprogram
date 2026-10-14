import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/Login.vue'),
      meta: { public: true }
    },
    // 已删除/合并菜单的旧路径跳转
    { path: '/system/dicts', redirect: '/system/config' },
    { path: '/system/extension-fields', redirect: '/system/config?tab=extension' },
    { path: '/system/broadcast-templates', redirect: '/leads?tab=broadcast' },
    // /pool 已合并到 /leads，保留旧路径重定向
    { path: '/pool', redirect: '/leads?tab=pool' },
    { path: '/system/upload-templates', redirect: '/system/games?tab=upload' },
    { path: '/system/partners', redirect: '/system/games?tab=partners' },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      redirect: '/dashboard',
      children: [
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('@/views/Dashboard.vue'),
          meta: { title: '数据看板' }
        },
        {
          path: 'leads',
          name: 'leads',
          component: () => import('@/views/leads/LeadsCenter.vue'),
          meta: { title: '线索管理' }
        },
        {
          path: 'leads/:id',
          name: 'lead-detail',
          component: () => import('@/views/leads/Detail.vue'),
          meta: { title: '线索详情' }
        },
        {
          path: 'orders',
          name: 'orders',
          component: () => import('@/views/orders/List.vue'),
          meta: { title: '订单数据' }
        },
        {
          path: 'players',
          name: 'players',
          component: () => import('@/views/players/Players.vue'),
          meta: { title: '玩家列表' }
        },
        {
          path: 'orders/import',
          name: 'order-import',
          component: () => import('@/views/orders/Import.vue'),
          meta: { title: '导入订单', adminOnly: true }
        },
        {
          path: 'orders/:id',
          name: 'order-detail',
          component: () => import('@/views/orders/Detail.vue'),
          meta: { title: '订单详情' }
        },
        {
          path: 'system/users',
          name: 'users',
          component: () => import('@/views/system/Users.vue'),
          meta: { title: '用户管理', adminOnly: true }
        },
        {
          path: 'system/departments',
          name: 'departments',
          component: () => import('@/views/system/Departments.vue'),
          meta: { title: '部门管理', adminOnly: true }
        },
        {
          path: 'system/games',
          name: 'games',
          component: () => import('@/views/system/GamesCenter.vue'),
          meta: { title: '游戏管理', adminOnly: true }
        },
        {
          path: 'system/config',
          name: 'config-center',
          component: () => import('@/views/system/ConfigCenter.vue'),
          meta: { title: '配置中心', adminOnly: true }
        },
        {
          path: 'system/audit',
          name: 'audit',
          component: () => import('@/views/system/SystemLogCenter.vue'),
          meta: { title: '系统日志', adminOnly: true }
        }
      ]
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/dashboard'
    }
  ]
})

// 路由守卫
router.beforeEach(async (to, from, next) => {
  const userStore = useUserStore()

  if (to.meta.public) {
    next()
    return
  }

  if (!userStore.isLoggedIn) {
    next({ name: 'login', query: { redirect: to.fullPath } })
    return
  }

  // 首次进入加载用户信息
  if (!userStore.userInfo) {
    await userStore.fetchMe()
  }

  // admin 专属页面
  if (to.meta.adminOnly && !userStore.isAdmin) {
    next({ name: 'dashboard' })
    return
  }

  next()
})

export default router