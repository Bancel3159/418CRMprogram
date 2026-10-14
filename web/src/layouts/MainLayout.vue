<template>
  <n-layout has-sider style="height: 100vh">
    <n-layout-sider
      bordered
      :width="220"
      :native-scrollbar="false"
      collapse-mode="width"
      :collapsed-width="64"
      :collapsed="collapsed"
      show-trigger
      @collapse="collapsed = true"
      @expand="collapsed = false"
    >
      <div class="logo">
        <span v-if="!collapsed">418 CRM</span>
        <span v-else>418</span>
      </div>
      <n-menu
        :value="activeKey"
        :collapsed="collapsed"
        :collapsed-width="64"
        :collapsed-icon-size="22"
        :options="menuOptions"
        :indent="18"
        @update:value="handleMenuClick"
      />
    </n-layout-sider>
    <n-layout>
      <n-layout-header bordered class="header">
        <div class="header-left">
          <span style="font-size: 16px; font-weight: 600">{{ currentTitle }}</span>
        </div>
        <div class="header-right">
          <AlertBell v-if="userStore.isAdmin" />
          <n-dropdown :options="userMenuOptions" trigger="click" @select="handleUserMenu">
            <div class="user-info">
              <n-avatar round size="small" style="background: #18a058">
                {{ userStore.userInfo?.real_name?.charAt(0) || '?' }}
              </n-avatar>
              <span style="margin-left: 8px">
                {{ userStore.userInfo?.real_name || userStore.userInfo?.username }}
              </span>
              <n-tag size="small" style="margin-left: 8px" type="success">
                {{ userStore.roleLabel }}
              </n-tag>
            </div>
          </n-dropdown>
        </div>
      </n-layout-header>
      <n-layout-content content-style="padding: 16px;" :native-scrollbar="false">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { ref, computed, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NLayout, NLayoutSider, NLayoutHeader, NLayoutContent, NMenu, NDropdown, NAvatar, NTag, type MenuOption, type DropdownOption } from 'naive-ui'
import { useUserStore } from '@/stores/user'
import AlertBell from '@/components/AlertBell.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const collapsed = ref(false)
const activeKey = computed(() => route.path)
const currentTitle = computed(() => (route.meta.title as string) || '418 CRM')

// 菜单
const menuOptions = computed<MenuOption[]>(() => {
  const baseMenus: MenuOption[] = [
    { label: '数据看板', key: '/dashboard', icon: () => h('span', '📊') },
    { label: '线索管理', key: '/leads', icon: () => h('span', '📝') },
    { label: '玩家列表', key: '/players', icon: () => h('span', '👥') },
    { label: '订单数据', key: '/orders', icon: () => h('span', '💰') }
  ]

  const systemMenus: MenuOption[] = [
    {
      label: '系统管理',
      key: 'system',
      icon: () => h('span', '⚙️'),
      children: [
        { label: '用户管理', key: '/system/users' },
        { label: '部门管理', key: '/system/departments' },
        {
          label: '游戏管理',
          key: '/system/games'
        },
        { label: '配置中心', key: '/system/config' },
        { label: '系统日志', key: '/system/audit' }
      ]
    }
  ]

  if (userStore.isAdmin) {
    baseMenus.push(...systemMenus)
  }

  return baseMenus
})

function handleMenuClick(key: string) {
  router.push(key)
}

// 用户下拉菜单
const userMenuOptions: DropdownOption[] = [
  { label: '退出登录', key: 'logout' }
]

async function handleUserMenu(key: string) {
  if (key === 'logout') {
    try { await userStore.logout() } catch {}
    router.push('/login')
  }
}
</script>

<style scoped>
.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: 700;
  color: #18a058;
  border-bottom: 1px solid #f0f0f0;
}

.header {
  height: 60px;
  padding: 0 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: white;
}

.header-right {
  display: flex;
  align-items: center;
}

.user-info {
  display: flex;
  align-items: center;
  cursor: pointer;
  padding: 6px 12px;
  border-radius: 4px;
}

.user-info:hover {
  background: #f5f7fa;
}
</style>