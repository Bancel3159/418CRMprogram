<template>
  <n-popover
    v-model:show="showPanel"
    trigger="manual"
    placement="bottom-end"
    :width="380"
    :show-arrow="true"
    style="padding: 0"
  >
    <template #trigger>
      <n-badge
        :value="wsStore.unreadCount"
        :max="99"
        :show="wsStore.unreadCount > 0"
        :offset="[-2, 2]"
      >
        <n-button
          circle
          quaternary
          size="small"
          aria-label="实时通知"
          @click="togglePanel"
        >
          <template #icon>
            <span style="font-size: 18px">🔔</span>
          </template>
        </n-button>
      </n-badge>
    </template>

    <!-- 面板内容 -->
    <div class="notif-panel">
      <div class="notif-header">
        <span class="notif-title">📬 实时通知</span>
        <n-space size="small">
          <n-button v-if="wsStore.unreadCount > 0" size="tiny" @click="markAllRead">
            全部已读
          </n-button>
          <n-button v-if="wsStore.notifications.length > 0" size="tiny" type="error" text @click="clearAll">
            清空
          </n-button>
        </n-space>
      </div>

      <div class="notif-body">
        <div v-if="wsStore.notifications.length === 0" class="notif-empty">
          <div style="font-size: 32px; opacity: 0.4">🔕</div>
          <div style="color: #9ca3af; font-size: 13px; margin-top: 6px">暂无通知</div>
        </div>

        <div
          v-for="n in wsStore.notifications"
          :key="n.id"
          :class="['notif-item', { unread: !n.read, 'high-value': n.high_value }]"
          @click="onItemClick(n)"
        >
          <div class="notif-item-title">
            <span :class="['notif-dot', { active: !n.read }]" />
            {{ n.title }}
          </div>
          <div class="notif-item-body">{{ n.body }}</div>
          <div class="notif-item-meta">
            <span>{{ formatTime(n.received_at) }}</span>
            <span v-if="n.source" class="notif-source">via {{ n.source }}</span>
          </div>
        </div>
      </div>
    </div>
  </n-popover>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { NPopover, NBadge, NButton, NSpace, useNotification, useMessage } from 'naive-ui'
import { useWSStore, type RealtimeNotification } from '@/stores/ws'

const wsStore = useWSStore()
const router = useRouter()
const notify = useNotification()
const message = useMessage()
const showPanel = ref(false)

function togglePanel() {
  showPanel.value = !showPanel.value
}

function markAllRead() {
  wsStore.markAllRead()
  message.success('已全部标为已读')
}

function clearAll() {
  wsStore.clearAll()
}

function onItemClick(n: RealtimeNotification) {
  showPanel.value = false
  if (n.link) {
    router.push(n.link)
  }
}

function formatTime(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  const now = new Date()
  const diffSec = (now.getTime() - d.getTime()) / 1000
  if (diffSec < 60) return '刚刚'
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)} 分钟前`
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)} 小时前`
  // 同年省略年
  const sameYear = d.getFullYear() === now.getFullYear()
  const opts: Intl.DateTimeFormatOptions = sameYear
      ? { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }
      : { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }
  return new Intl.DateTimeFormat('zh-CN', opts).format(d)
}
</script>

<style scoped>
.notif-panel {
  width: 380px;
  max-height: 480px;
  display: flex;
  flex-direction: column;
}
.notif-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px;
  border-bottom: 1px solid #e5e7eb;
  background: #f9fafb;
}
.notif-title {
  font-weight: 600;
  font-size: 14px;
}
.notif-body {
  flex: 1;
  overflow-y: auto;
  max-height: 420px;
}
.notif-empty {
  padding: 36px 16px;
  text-align: center;
}
.notif-item {
  padding: 10px 14px;
  border-bottom: 1px solid #f3f4f6;
  cursor: pointer;
  transition: background 0.12s;
}
.notif-item:hover {
  background: #f3f4f6;
}
.notif-item.unread {
  background: #eff6ff;
}
.notif-item.unread:hover {
  background: #dbeafe;
}
.notif-item.high-value {
  border-left: 3px solid #f59e0b;
}
.notif-item-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  font-size: 13px;
  color: #1f2937;
  margin-bottom: 4px;
}
.notif-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #d1d5db;
  flex-shrink: 0;
}
.notif-dot.active {
  background: #2563eb;
}
.notif-item-body {
  font-size: 12px;
  color: #4b5563;
  line-height: 1.5;
  padding-left: 12px;
}
.notif-item-meta {
  font-size: 11px;
  color: #9ca3af;
  padding-left: 12px;
  margin-top: 4px;
  display: flex;
  gap: 8px;
}
.notif-source {
  color: #6b7280;
}
</style>