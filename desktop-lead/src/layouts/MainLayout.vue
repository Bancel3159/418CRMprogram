<template>
  <n-layout class="root">
    <n-layout-header bordered class="header">
      <div class="brand">
        <div class="brand-mark">海外</div>
        <div class="brand-meta">
          <div class="brand-title">418前端</div>
        </div>
      </div>
      <n-dropdown
        v-if="displayName"
        :options="userMenuOptions"
        trigger="click"
        placement="bottom-end"
        @select="onUserMenuSelect"
      >
        <n-tag
          :bordered="false"
          size="small"
          class="user-tag"
          :title="`${displayName} - 点击展开菜单`"
          style="cursor: pointer"
        >
          {{ displayName }}
        </n-tag>
      </n-dropdown>
    </n-layout-header>
    <n-layout-content content-style="padding: 14px" :native-scrollbar="false" class="content">
      <router-view v-slot="{ Component }">
        <transition name="fade-slide" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </n-layout-content>
  </n-layout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { NLayout, NLayoutHeader, NLayoutContent, NTag, NDropdown } from 'naive-ui'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

// 只显示姓名,不显示用户名(姓名 > 用户名 > 空)
const displayName = computed(
  () => userStore.userInfo?.real_name || userStore.userInfo?.username || ''
)

// 点击姓名弹出的菜单:数据看板 / 快捷键设置 / 退出
const userMenuOptions = [
  {
    label: '数据看板',
    key: 'dashboard'
  },
  {
    label: '快捷键设置',
    key: 'settings'
  },
  { type: 'divider', key: 'd1' },
  {
    label: '退出',
    key: 'logout'
  }
]

function onUserMenuSelect(key: string) {
  if (key === 'dashboard') {
    router.push('/dashboard')
  } else if (key === 'settings') {
    router.push('/settings')
  } else if (key === 'logout') {
    userStore.logout()
    router.push('/login')
  }
}
</script>

<style scoped>
.root {
  height: 100vh;
  background: var(--c-ink-50, #f0f6ff);
}
.header {
  height: 52px;
  /* 右侧留 ~90px:36×2=72px 控件 + 18px 呼吸距离,让用户名 tag 推到控件左侧,避免水平挤在一起 */
  padding: 0 90px 0 18px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: linear-gradient(135deg, #ffffff, #f0f6ff);
  border-bottom: 1px solid rgba(37, 99, 235, 0.14);
  box-shadow: 0 1px 0 rgba(37, 99, 235, 0.04), 0 4px 14px rgba(37, 99, 235, 0.05);
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
}
.brand-mark {
  width: 32px;
  height: 32px;
  border-radius: var(--r-md, 10px);
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
  font-weight: 800;
  font-size: 11px;
  display: flex;
  align-items: center;
  justify-content: center;
  letter-spacing: 0.5px;
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.35);
}
.brand-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--c-blue-800, #1e3a8a);
  line-height: 1.2;
  letter-spacing: 0.3px;
}
.user-tag {
  font-size: 12px;
  font-weight: 500;
  background: #eff6ff !important;
  color: #1d4ed8 !important;
}
/* 下拉菜单项(图标 + 文字对齐) */
:deep(.n-dropdown-option) {
  font-size: 12px !important;
}
.content {
  background: #eff4fb;
}

/* 路由切换淡入 */
.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}
.fade-slide-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
.fade-slide-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
