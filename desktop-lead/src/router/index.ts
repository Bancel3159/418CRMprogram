import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: () => import('@/views/Login.vue'), meta: { public: true } },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      redirect: '/leads/new',
      children: [
        { path: 'leads/new', name: 'lead-new', component: () => import('@/views/LeadNew.vue') },
        { path: 'dashboard', name: 'dashboard', component: () => import('@/views/Dashboard.vue') },
        { path: 'settings', name: 'settings', component: () => import('@/views/Settings.vue') }
      ]
    }
  ]
})

router.beforeEach(async (to) => {
  const userStore = useUserStore()
  if (to.meta.public) return true
  if (!userStore.isLoggedIn) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (!userStore.userInfo) await userStore.fetchMe()
  return true
})

export default router