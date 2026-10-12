<template>
  <div>
    <div class="page-header">
      <div>
        <span class="page-title">公开池</span>
        <n-tag type="warning" size="large" style="margin-left: 12px">
          {{ total }} 条待认领
        </n-tag>
      </div>
      <n-space>
        <n-tag :type="wsStore.connected ? 'success' : 'error'" size="small">
          {{ wsStore.connected ? '🟢 实时推送已连接' : '🔴 离线' }}
        </n-tag>
        <n-button @click="loadList" :loading="loading">刷新</n-button>
      </n-space>
    </div>

    <!-- 实时推送提醒 -->
    <n-alert
      v-if="latestBroadcast"
      type="info"
      closable
      style="margin-bottom: 16px"
      @close="latestBroadcast = null"
    >
      <strong>🆕 新线索：</strong>{{ latestBroadcast }}
    </n-alert>

    <!-- 线索列表 -->
    <n-card v-for="lead in leads" :key="lead.id" style="margin-bottom: 12px" hoverable>
      <div style="display: flex; justify-content: space-between; align-items: start; gap: 16px">
        <div style="flex: 1">
          <n-space align="center" style="margin-bottom: 8px">
            <n-tag type="info" size="small">#{{ lead.id }}</n-tag>
            <n-tag size="small">{{ lead.game?.code }} - {{ lead.game?.name }}</n-tag>
            <n-tag :type="lead.is_target === 'target' ? 'success' : 'default'" size="small">
              {{ lead.is_target === 'target' ? '目标' : '非目标' }}
            </n-tag>
            <span style="color: #6b7280; font-size: 13px">{{ formatTime(lead.created_at) }}</span>
          </n-space>

          <div style="font-size: 16px; font-weight: 600; margin-bottom: 4px">
            {{ lead.player_game_name }}
            <span style="color: #6b7280; font-size: 13px; font-weight: 400">
              ({{ lead.player_country }} · {{ lead.region }} · {{ lead.player_age }}岁)
            </span>
          </div>

          <div style="color: #6b7280; font-size: 13px; margin-bottom: 4px">
            玩家 ID：<code>{{ lead.player_id }}</code>
            · 渠道：{{ lead.channel }}
            · 社群：{{ lead.community || '-' }}
          </div>

          <div style="color: #6b7280; font-size: 13px">
            拉新员工：{{ lead.creator?.real_name || lead.creator?.username }}
          </div>
        </div>

        <n-space vertical>
          <n-button type="primary" size="large" :loading="claimingId === lead.id" @click="handleClaim(lead)">
            🎯 抢单认领
          </n-button>
          <n-button size="small" text @click="viewDetail(lead.id)">查看详情</n-button>
        </n-space>
      </div>
    </n-card>

    <n-empty v-if="leads.length === 0 && !loading" description="公开池暂时没有线索" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NButton, NSpace, NTag, NAlert, NEmpty, useMessage, useNotification } from 'naive-ui'
import { leadApi } from '@/api/modules'
import { useWSStore } from '@/stores/ws'

const router = useRouter()
const message = useMessage()
const notification = useNotification()
const wsStore = useWSStore()

const leads = ref<any[]>([])
const total = ref(0)
const loading = ref(false)
const claimingId = ref<number | null>(null)
const latestBroadcast = ref<string | null>(null)

function formatTime(t: string) {
  return new Date(t).toLocaleTimeString('zh-CN')
}

function viewDetail(id: number) {
  router.push(`/leads/${id}`)
}

async function loadList() {
  loading.value = true
  try {
    const resp: any = await leadApi.pool({ page: 1, page_size: 100 })
    leads.value = resp.data.list || []
    total.value = resp.data.total || 0
  } catch {} finally {
    loading.value = false
  }
}

// 抢单（核心：自动复制 token）
async function handleClaim(lead: any) {
  claimingId.value = lead.id
  try {
    const resp: any = await leadApi.claim(lead.id)
    const token = resp.data.token_to_copy
    // 自动复制到剪贴板
    if (token) {
      try {
        await navigator.clipboard.writeText(token)
        notification.success({
          title: '🎯 认领成功',
          content: `Token 已自动复制到剪贴板！\n长度：${token.length} 字符`,
          duration: 5000
        })
      } catch {
        notification.warning({
          title: '⚠️ 自动复制失败',
          content: `Token: ${token}\n请手动复制`,
          duration: 8000
        })
      }
    } else {
      message.success('认领成功')
    }
    loadList()
  } catch (err: any) {
    message.error(err?.message || '认领失败')
  } finally {
    claimingId.value = null
  }
}

// 监听 WebSocket 消息
function handleWSMessage(msg: any) {
  if (!msg) return
  if (msg.channel === 'pool:broadcast' && msg.event === 'lead.created') {
    const d = msg.data
    const summary = `${d.game_code || ''} 新玩家：${d.player_country || ''}${d.player_age || ''}岁 加入${d.region || ''}区（${d.is_target || ''}）`
    latestBroadcast.value = summary
    notification.info({
      title: '🆕 新线索进入公开池',
      content: summary + (d.rendered_message ? `\n${d.rendered_message}` : ''),
      duration: 6000
    })
    loadList() // 重新拉取
  } else if (msg.channel === 'pool:remove') {
    loadList()
  }
}

watch(() => wsStore.lastMessage, (msg) => handleWSMessage(msg), { deep: true })

onMounted(() => {
  loadList()
})
</script>