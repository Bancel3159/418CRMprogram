<template>
  <div>
    <div class="page-header">
      <span style="font-size: 18px; font-weight: 700">📋 今日待办</span>
      <n-space align="center">
        <n-select v-model:value="statusFilter" :options="statusOptions" style="width: 110px" @update:value="loadList" />
        <n-button @click="loadList" :loading="loading">刷新</n-button>
      </n-space>
    </div>

    <!-- 顶部统计 -->
    <n-grid :cols="4" :x-gap="12" :y-gap="12" responsive="screen" item-responsive>
      <n-gi span="1 m:1 l:1">
        <div class="stat-card info">
          <div class="label">待处理</div>
          <div class="value">{{ counts.pending }}</div>
        </div>
      </n-gi>
      <n-gi span="1 m:1 l:1">
        <div class="stat-card danger">
          <div class="label">高优先级</div>
          <div class="value">{{ counts.high }}</div>
        </div>
      </n-gi>
      <n-gi span="1 m:1 l:1">
        <div class="stat-card warning">
          <div class="label">中优先级</div>
          <div class="value">{{ counts.medium }}</div>
        </div>
      </n-gi>
      <n-gi span="1 m:1 l:1">
        <div class="stat-card success">
          <div class="label">已完成(今日)</div>
          <div class="value">{{ counts.done }}</div>
        </div>
      </n-gi>
    </n-grid>

    <!-- 提醒列表 -->
    <n-empty v-if="!loading && reminders.length === 0" description="今日没有待办提醒" style="margin-top: 32px" />
    <div v-else style="margin-top: 16px">
      <n-card
        v-for="r in reminders"
        :key="r.id"
        class="reminder-card"
        :class="priorityClass(r.priority, r.status)"
        hoverable
        style="margin-bottom: 12px"
      >
        <div style="display: flex; align-items: center; gap: 12px">
          <!-- 类型图标 -->
          <div :class="['priority-icon', priorityBg(r.priority)]">
            {{ typeIcon(r.remind_type) }}
          </div>

          <!-- 主体 -->
          <div style="flex: 1">
            <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 4px">
              <n-tag :type="typeTagType(r.remind_type)" size="small">{{ typeLabel(r.remind_type) }}</n-tag>
              <n-tag v-if="r.priority === 2" type="error" size="small">🔥 高优先级</n-tag>
              <n-tag v-else-if="r.priority === 1" type="warning" size="small">⚡ 中优先级</n-tag>
              <n-tag v-if="r.status === 'done'" type="success" size="small">✓ 已完成</n-tag>
              <n-tag v-else-if="r.status === 'expired'" type="default" size="small">已过期</n-tag>
              <n-tag v-if="r.lead?.is_lost" type="error" size="small">⚠️ 已流失</n-tag>
              <span v-if="r.note" style="font-size: 12px; color: #6b7280">{{ r.note }}</span>
            </div>

            <div style="font-size: 15px; font-weight: 600">
              {{ r.lead?.player_game_name || r.lead?.player_username || '?' }}
              <span style="font-weight: 400; color: #6b7280; font-size: 13px">
                ({{ r.lead?.game_code || '-' }} · {{ r.lead?.player_country || '-' }}
                · ID: <code style="font-size: 12px">{{ r.lead?.player_id || '-' }}</code>)
              </span>
            </div>

            <div style="font-size: 12px; color: #9ca3af; margin-top: 4px">
              <span v-if="r.lead?.claimed_by_name">
                认领人:{{ r.lead.claimed_by_name }}
              </span>
              <span v-if="r.lead?.is_paid" style="color: #059669; margin-left: 8px">💰 已付费</span>
            </div>
          </div>

          <!-- 操作 -->
          <n-space>
            <n-button
              v-if="r.status === 'pending'"
              type="primary"
              size="small"
              @click="goFollow(r.lead_id)"
            >
              ✍️ 去跟进
            </n-button>
            <n-button v-else size="small" quaternary @click="goFollow(r.lead_id)">查看</n-button>
          </n-space>
        </div>
      </n-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NGrid, NGi, NCard, NSpace, NButton, NSelect, NTag, NEmpty, useMessage } from 'naive-ui'
import { remindersApi } from '@/api/modules'

const router = useRouter()
const message = useMessage()

const statusFilter = ref('pending') // pending / done / all
const statusOptions = [
  { label: '待处理', value: 'pending' },
  { label: '已完成', value: 'done' },
  { label: '全部', value: 'all' }
]

const reminders = ref<any[]>([])
const loading = ref(false)

// 顶部统计(独立拉一次 done 状态)
const counts = ref({ pending: 0, medium: 0, high: 0, done: 0 })

async function loadList() {
  loading.value = true
  try {
    const resp: any = await remindersApi.today({ status: statusFilter.value, page_size: 100 })
    reminders.value = resp.data.list || []
  } catch (err: any) {
    message.error(err?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadCounts() {
  try {
    const [pendingR, doneR] = await Promise.all([
      remindersApi.today({ status: 'pending', page_size: 100 }),
      remindersApi.today({ status: 'done', page_size: 100 })
    ])
    const pendingList = pendingR.data.list || []
    const doneList = doneR.data.list || []
    counts.value = {
      pending: pendingList.length,
      high: pendingList.filter((r: any) => r.priority === 2).length,
      medium: pendingList.filter((r: any) => r.priority === 1).length,
      done: doneList.length
    }
  } catch {}
}

function goFollow(leadId: number) {
  router.push(`/leads/${leadId}`)
}

const TYPE_LABELS: Record<string, { label: string; icon: string; tagType: string }> = {
  follow_new_register: { label: '新注册跟进', icon: '🆕', tagType: 'info' },
  follow_paid_daily: { label: '付费每日跟进', icon: '💎', tagType: 'success' },
  follow_paid_offline: { label: '付费离线', icon: '⏰', tagType: 'warning' },
  lost_check: { label: '流失检查', icon: '⚠️', tagType: 'error' }
}

function typeLabel(t: string) {
  return TYPE_LABELS[t]?.label || t
}
function typeIcon(t: string) {
  return TYPE_LABELS[t]?.icon || '📌'
}
function typeTagType(t: string) {
  return TYPE_LABELS[t]?.tagType || 'default'
}
function priorityBg(p: number) {
  if (p === 2) return 'priority-high'
  if (p === 1) return 'priority-medium'
  return 'priority-normal'
}
function priorityClass(p: number, status: string) {
  if (status === 'done') return 'done'
  if (p === 2) return 'priority-high-card'
  if (p === 1) return 'priority-medium-card'
  return ''
}

onMounted(async () => {
  await Promise.all([loadList(), loadCounts()])
})
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.stat-card {
  background: white;
  border-radius: 10px;
  padding: 14px 16px;
  border-left: 4px solid #94a3b8;
}
.stat-card.info { border-left-color: #3b82f6; }
.stat-card.success { border-left-color: #10b981; }
.stat-card.warning { border-left-color: #f59e0b; }
.stat-card.danger { border-left-color: #ef4444; }
.stat-card .label { font-size: 13px; color: #6b7280; margin-bottom: 6px; }
.stat-card .value { font-size: 26px; font-weight: 700; color: #0f172a; }

.reminder-card {
  transition: all 0.2s;
}
.reminder-card.done { opacity: 0.5; }
.reminder-card.priority-high-card {
  border-left: 4px solid #ef4444;
  background: linear-gradient(to right, #fef2f2 0%, white 10%);
}
.reminder-card.priority-medium-card {
  border-left: 4px solid #f59e0b;
  background: linear-gradient(to right, #fffbeb 0%, white 10%);
}
.priority-icon {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  flex-shrink: 0;
}
.priority-normal { background: #f3f4f6; }
.priority-medium { background: #fef3c7; }
.priority-high { background: #fee2e2; }
</style>
