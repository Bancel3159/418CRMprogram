<template>
  <n-popover
    v-model:show="showPopover"
    trigger="click"
    placement="bottom-end"
    :width="380"
    :show-arrow="false"
  >
    <template #trigger>
      <div class="alert-bell" @click="openPopover">
        <n-badge :value="unreadCount" :max="99" :show="unreadCount > 0" :offset="[-4, 4]">
          <span style="font-size: 20px"></span>
        </n-badge>
      </div>
    </template>
    <div class="alert-popover">
      <div class="alert-header">
        <strong>实时告警</strong>
        <n-space>
          <n-button size="tiny" quaternary @click="markAllRead">全部已读</n-button>
          <n-button size="tiny" quaternary @click="clearAll">清空</n-button>
        </n-space>
      </div>
      <div v-if="alerts.length === 0" class="alert-empty">
        <p>暂无告警</p>
        <p style="font-size: 12px; color: #999">新订单入库 / 待办超阈值时会自动推送</p>
      </div>
      <div v-else class="alert-list">
        <div
          v-for="a in alerts"
          :key="a.id"
          :class="['alert-item', `level-${a.level}`, !a.read && 'unread']"
          @click="onAlertClick(a)"
        >
          <div class="alert-icon">
            <span v-if="a.level === 'critical'"></span>
            <span v-else-if="a.level === 'warning'"></span>
            <span v-else></span>
          </div>
          <div class="alert-body">
            <div class="alert-title">{{ a.title }}</div>
            <div class="alert-message">{{ a.message }}</div>
            <div class="alert-time">{{ formatTime(a.created_at) }}</div>
          </div>
        </div>
      </div>
    </div>
  </n-popover>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { NPopover, NBadge, NButton, NSpace, useMessage } from 'naive-ui'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'

const message = useMessage()
const router = useRouter()
const userStore = useUserStore()

const showPopover = ref(false)
const alerts = ref<any[]>([])
const unreadCount = ref(0)

let ws: WebSocket | null = null
let reconnectTimer: number | null = null

const MAX_ALERTS = 50 // 内存中最多保留 50 条

function loadInitial() {
  const saved = localStorage.getItem('crm_alerts')
  if (saved) {
    try {
      alerts.value = JSON.parse(saved)
      unreadCount.value = alerts.value.filter((a: any) => !a.read).length
    } catch {}
  }
}

function saveAlerts() {
  localStorage.setItem('crm_alerts', JSON.stringify(alerts.value.slice(0, MAX_ALERTS)))
}

function connectWS() {
  if (!userStore.isAdmin) return
  // 使用 Vite 代理:开发环境直接连 5173 的 /ws 代理
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  // 直接连后端端口(开发时是 8080)
  const url = `${proto}//${window.location.hostname}:8080/ws?token=${userStore.token}`
  ws = new WebSocket(url)

  ws.onopen = () => {
    console.log('[alert] ws connected')
    // alert:admin 频道已在 admin 连接时自动订阅,无需手动 subscribe
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.channel === 'alert:admin' && msg.event === 'alert') {
        const alert = msg.data
        alerts.value.unshift({ ...alert, read: false })
        if (alerts.value.length > MAX_ALERTS) {
          alerts.value = alerts.value.slice(0, MAX_ALERTS)
        }
        unreadCount.value++
        saveAlerts()
        // 桌面右下角通知
        if (Notification.permission === 'granted') {
          try {
            new Notification(alert.title, { body: alert.message })
          } catch {}
        }
      }
    } catch {}
  }

  ws.onclose = () => {
    ws = null
    reconnectTimer = window.setTimeout(connectWS, 5000)
  }

  ws.onerror = () => {
    // onclose 会重连
  }
}

function sendSubscribe(channels: string[]) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return
  ws.send(JSON.stringify({ type: 'subscribe', channels }))
}

function openPopover() {
  showPopover.value = true
  // 不自动标已读,让用户点击或手动操作
}

function markAllRead() {
  alerts.value.forEach((a: any) => a.read = true)
  unreadCount.value = 0
  saveAlerts()
}

function clearAll() {
  alerts.value = []
  unreadCount.value = 0
  saveAlerts()
}

function onAlertClick(a: any) {
  a.read = true
  unreadCount.value = alerts.value.filter((x: any) => !x.read).length
  saveAlerts()
  if (a.link) {
    router.push(a.link)
    showPopover.value = false
  }
}

function formatTime(s: string) {
  if (!s) return ''
  const d = new Date(s)
  const now = Date.now()
  const diff = Math.floor((now - d.getTime()) / 1000)
  if (diff < 60) return `${diff}秒前`
  if (diff < 3600) return `${Math.floor(diff / 60)}分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)}小时前`
  return d.toLocaleString('zh-CN', { hour12: false })
}

onMounted(() => {
  loadInitial()
  if (userStore.isAdmin) {
    if ('Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission()
    }
    connectWS()
  }
})

onBeforeUnmount(() => {
  if (reconnectTimer) clearTimeout(reconnectTimer)
  if (ws) ws.close()
})
</script>

<style scoped>
.alert-bell {
  cursor: pointer;
  padding: 6px 10px;
  border-radius: 4px;
  position: relative;
  display: inline-block;
}
.alert-bell:hover {
  background: #f5f7fa;
}

.alert-popover {
  max-height: 480px;
  overflow-y: auto;
}

.alert-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 8px;
  border-bottom: 1px solid #f0f0f0;
}

.alert-empty {
  text-align: center;
  padding: 40px 20px;
  color: #999;
}
.alert-empty p {
  margin: 4px 0;
}

.alert-list {
  margin-top: 4px;
}
.alert-item {
  display: flex;
  padding: 10px;
  border-radius: 4px;
  margin-bottom: 4px;
  cursor: pointer;
  transition: background 0.2s;
  border-left: 3px solid transparent;
}
.alert-item:hover {
  background: #f5f7fa;
}
.alert-item.level-info { border-left-color: #2080f0; }
.alert-item.level-warning { border-left-color: #f0a020; background: #fff8e1; }
.alert-item.level-critical { border-left-color: #d03050; background: #ffe7e7; }
.alert-item.unread {
  font-weight: 600;
}

.alert-icon {
  margin-right: 10px;
  font-size: 20px;
  line-height: 1;
}

.alert-body {
  flex: 1;
  min-width: 0;
}
.alert-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 2px;
}
.alert-message {
  font-size: 12px;
  color: #555;
  word-break: break-all;
  margin-bottom: 4px;
}
.alert-time {
  font-size: 11px;
  color: #999;
}
</style>