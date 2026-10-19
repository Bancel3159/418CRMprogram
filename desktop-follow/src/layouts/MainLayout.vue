<template>
  <n-layout has-sider style="height: 100vh">
    <!-- 侧栏菜单 -->
    <n-layout-sider
      bordered
      :collapsed-width="64"
      :width="200"
      show-trigger
      collapse-mode="width"
      :native-scrollbar="false"
    >
      <div class="logo">
        <span style="font-size: 18px; font-weight: 700; color: #f59e0b">维护端</span>
      </div>
      <n-menu
        :value="currentMenuKey"
        :options="menuOptions"
        :collapsed-width="64"
        :collapsed-icon-size="20"
        @update:value="handleMenuSelect"
      />
    </n-layout-sider>

    <n-layout>
      <!-- 顶部 header -->
      <n-layout-header bordered class="header">
        <span style="font-size: 16px; font-weight: 600">{{ currentTitle }}</span>
        <n-space align="center">
          <!-- 接单开关(P11) -->
          <n-tooltip placement="bottom">
            <template #trigger>
              <n-space align="center" size="small" :wrap-item="false">
                <span style="font-size: 13px; color: #4b5563">
                  {{ userStore.acceptOrders ? '接单中' : '休息中' }}
                </span>
                <n-switch
                  :value="userStore.acceptOrders"
                  :loading="switchLoading"
                  size="small"
                  @update:value="onToggleAccept"
                />
              </n-space>
            </template>
            关闭后,公开池新线索不再推送到此设备(本端继续可访问)
          </n-tooltip>

          <n-divider vertical />

          <span>
            <span :class="['ws-indicator', wsStore.connected ? 'connected' : 'disconnected']" />
            {{ wsStore.connected ? '已连接' : '未连接' }}
          </span>

          <!-- P12 实时通知铃铛 -->
          <NotificationBell />

          <n-tag type="info" size="small">
            {{ userStore.userInfo?.real_name || userStore.userInfo?.username }}
          </n-tag>
          <n-tag type="warning" size="small">{{ userStore.userInfo?.role }}</n-tag>

          <n-button size="small" @click="logout">登出</n-button>
        </n-space>
      </n-layout-header>

      <!-- 内容 -->
      <n-layout-content content-style="padding: 16px" :native-scrollbar="false">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NLayout, NLayoutSider, NLayoutHeader, NLayoutContent,
  NSpace, NTag, NButton, NMenu, NSwitch, NDivider, NTooltip,
  useMessage, useNotification, type MenuOption
} from 'naive-ui'
import { useUserStore } from '@/stores/user'
import { useWSStore } from '@/stores/ws'
import NotificationBell from '@/components/NotificationBell.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const wsStore = useWSStore()
const message = useMessage()
const notify = useNotification()

const switchLoading = ref(false)

const currentTitle = computed(() => (route.meta.title as string) || '维护')
const currentMenuKey = computed(() => (route.name as string) || '')

const menuOptions: MenuOption[] = [
  { label: '今日待办', key: 'todo' },
  { label: '看板', key: 'dashboard' },
  { label: '公开池', key: 'pool' },
  { label: '我的玩家', key: 'leads' },
  { label: '查询', key: 'search' }
]

function handleMenuSelect(key: string) {
  router.push({ name: key })
}

// 切换接单开关
async function onToggleAccept(val: boolean) {
  switchLoading.value = true
  try {
    await userStore.setAcceptOrders(val)
    message.success(val ? '已开启接单 — 公开池将推送新线索' : '已停止接单 — 休息时间不会被打扰')
  } catch (err: any) {
    message.error(err?.message || '切换失败')
  } finally {
    switchLoading.value = false
  }
}

// 登录态变  WS 跟着变
watch(
  () => userStore.token,
  (token) => {
    if (token) {
      wsStore.connect(token)
      userStore.fetchSettings()
    } else {
      wsStore.disconnect()
    }
  },
  { immediate: true }
)

onMounted(() => {
  if (userStore.token) {
    wsStore.connect(userStore.token)
    userStore.fetchSettings()
  }
})

// P12 实时事件  高额 / 重新激活弹窗
watch(
  () => wsStore.lastMessage,
  (msg) => {
    if (!msg || msg.channel !== 'follow:realtime') return
    const data = msg.data || {}
    if (msg.event === 'order.high_value') {
      // 高额订单:弹 n-notification + 跳详情可点
      notify.create({
        type: 'warning',
        title: `高额订单 ¥${data.amount}`,
        content: `${data.game_code || ''} 玩家 ${data.player_id} 充值 ¥${data.amount}` +
                 (data.claimed_by_name ? ` · 维护员 ${data.claimed_by_name}` : ' · 暂无人认领'),
        duration: 8000,
        closable: true,
        onClick: () => {
          if (data.lead_id) router.push(`/leads/${data.lead_id}`)
        }
      })
    } else if (msg.event === 'player.react' && data.was_lost) {
      // 玩家重新激活(从流失恢复)
      notify.create({
        type: 'success',
        title: '玩家重新激活',
        content: `${data.game_code || ''} 玩家 ${data.player_id} 再次上线,已从流失状态恢复` +
                 (data.claimed_by_name ? ` · 维护员 ${data.claimed_by_name}` : ''),
        duration: 6000,
        onClick: () => {
          if (data.lead_id) router.push(`/leads/${data.lead_id}`)
        }
      })
    }
  }
)

function logout() {
  wsStore.disconnect()
  userStore.logout()
  router.push('/login')
}
</script>

<style scoped>
.logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 16px;
  border-bottom: 1px solid #e5e7eb;
}
.header {
  height: 56px;
  padding: 0 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: white;
}
.ws-indicator {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 6px;
}
.ws-indicator.connected { background: #10b981; }
.ws-indicator.disconnected { background: #ef4444; }
</style>
