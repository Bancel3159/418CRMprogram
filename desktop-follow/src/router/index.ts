import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

// follow 端白名单(其他路径一律跳回 dashboard)
// 需求 1:follow 端不开放玩家列表 / 订单数据这种统一的数据视图
const ALLOWED_PATHS = new Set(['login', 'dashboard', 'todo', 'pool', 'leads', 'lead-detail', 'search'])

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: () => import('@/views/Login.vue'), meta: { public: true } },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      redirect: '/todo', // P11 默认进今日待办(用户最常用的工作起点)
      children: [
        { path: 'dashboard', name: 'dashboard', component: () => import('@/views/Dashboard.vue'), meta: { title: '看板' } },
        { path: 'todo', name: 'todo', component: () => import('@/views/TodayTodo.vue'), meta: { title: '今日待办' } },
        { path: 'pool', name: 'pool', component: () => import('@/views/Pool.vue'), meta: { title: '公开池' } },
        { path: 'leads', name: 'leads', component: () => import('@/views/Leads.vue'), meta: { title: '我的玩家' } },
        { path: 'leads/:id', name: 'lead-detail', component: () => import('@/views/LeadDetail.vue'), meta: { title: '玩家详情' } },
        { path: 'search', name: 'search', component: () => import('@/views/Search.vue'), meta: { title: '查询' } }
      ]
    }
  ]
})

router.beforeEach(async (to) => {
  const userStore = useUserStore()

  // 公开页:直接放行
  if (to.meta.public) return true

  // 未登录:跳登录
  if (!userStore.isLoggedIn) return { name: 'login', query: { redirect: to.fullPath } }

  // 已登录:确保 userInfo + 接单状态都拉过
  if (!userStore.userInfo) await userStore.fetchMe()
  if (userStore.acceptOrders === null || (userStore.token && localStorage.getItem('accept_orders') === null)) {
    userStore.fetchSettings()
  }

  // 白名单校验(防手动敲 URL 跳到 follow 端不允许的页面,如 /players /orders 等)
  if (!ALLOWED_PATHS.has(to.name as string)) {
    return { name: 'dashboard', replace: true }
  }

  return true
})

export default router
