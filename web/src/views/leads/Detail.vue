<template>
  <div v-if="lead">
    <div class="page-header">
      <div class="page-title">
        线索详情 #{{ lead.id }}
        <n-tag :type="statusType" size="small" style="margin-left: 8px">{{ statusLabel }}</n-tag>
      </div>
      <n-space>
        <n-button @click="$router.back()">返回</n-button>
        <n-button
          v-if="lead.status === 'pending' && canClaim"
          type="primary"
          :loading="actionLoading"
          @click="handleClaim"
        >
          抢单认领
        </n-button>
        <n-button
          v-if="lead.status === 'claimed' && canRelease"
          :loading="actionLoading"
          @click="handleRelease"
        >
          释放回公开池
        </n-button>
        <n-button
          v-if="lead.status !== 'archived' && canArchive"
          :loading="actionLoading"
          @click="handleArchive"
        >
          归档
        </n-button>
      </n-space>
    </div>

    <n-grid :cols="2" :x-gap="16" :y-gap="16">
      <n-gi>
        <div class="card-section">
          <h3 style="margin: 0 0 16px">基本信息</h3>
          <n-descriptions :column="2" label-placement="left" label-width="100" bordered>
            <n-descriptions-item label="游戏">{{ lead.game?.name }} ({{ lead.game?.code }})</n-descriptions-item>
            <n-descriptions-item label="拉新员工">{{ lead.creator?.real_name || lead.creator?.username }}</n-descriptions-item>
            <n-descriptions-item label="渠道">{{ lead.channel }}</n-descriptions-item>
            <n-descriptions-item label="是否目标">
              <n-tag :type="lead.is_target === 'target' ? 'success' : 'default'" size="small">
                {{ lead.is_target === 'target' ? '目标' : '非目标' }}
              </n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="玩家 ID">{{ lead.player_id }}</n-descriptions-item>
            <n-descriptions-item label="玩家游戏名">{{ lead.player_game_name }}</n-descriptions-item>
            <n-descriptions-item label="玩家国家">{{ lead.player_country }}</n-descriptions-item>
            <n-descriptions-item label="区服">{{ lead.region }}</n-descriptions-item>
            <n-descriptions-item label="玩家年龄">{{ lead.player_age }}</n-descriptions-item>
            <n-descriptions-item label="社群">{{ lead.community }}</n-descriptions-item>
            <n-descriptions-item label="玩家用户名">{{ lead.player_username }}</n-descriptions-item>
            <n-descriptions-item label="提交时间">{{ formatTime(lead.created_at) }}</n-descriptions-item>
          </n-descriptions>
        </div>
      </n-gi>

      <n-gi>
        <div class="card-section">
          <h3 style="margin: 0 0 16px">
            token 码
            <n-tag type="warning" size="small" style="margin-left: 8px">仅限Discord渠道填写</n-tag>
          </h3>
          <div class="token-box">
            <code>{{ lead.token_code || '(空)' }}</code>
            <n-button
              v-if="lead.token_code"
              size="small"
              type="primary"
              style="margin-left: 8px"
              @click="copyToken"
            >
              复制
            </n-button>
          </div>
          <p class="text-muted" style="margin-top: 8px; font-size: 12px">
            认领时已自动复制到剪贴板
          </p>
        </div>
      </n-gi>

      <n-gi>
        <div class="card-section">
          <h3 style="margin: 0 0 16px">流转记录</h3>
          <n-timeline>
            <n-timeline-item type="info" :title="'提交时间'" :time="formatTime(lead.created_at)">
              {{ lead.creator?.real_name }} 提交了线索
            </n-timeline-item>
            <n-timeline-item v-if="lead.claimed_at" type="success" title="认领时间" :time="formatTime(lead.claimed_at)">
              {{ lead.claimed_by_user?.real_name }} 认领了线索
            </n-timeline-item>
            <n-timeline-item v-if="lead.released_at" type="warning" title="释放/归档时间" :time="formatTime(lead.released_at)">
              状态变为 {{ statusLabel }}
            </n-timeline-item>
          </n-timeline>
        </div>
      </n-gi>

      <n-gi>
        <div class="card-section">
          <h3 style="margin: 0 0 16px">备注</h3>
          <div>{{ lead.remark || '(无)' }}</div>
        </div>

        <div class="card-section">
          <h3 style="margin: 0 0 16px">自定义字段</h3>
          <pre v-if="hasExtension" style="background: #f5f7fa; padding: 12px; border-radius: 4px; overflow: auto">{{ JSON.stringify(lead.extension, null, 2) }}</pre>
          <div v-else class="text-muted">(无)</div>
        </div>
      </n-gi>

      <n-gi v-if="lead.screenshots && lead.screenshots.length > 0" :cols="2">
        <div class="card-section">
          <h3 style="margin: 0 0 16px">截图 ({{ lead.screenshots.length }})</h3>
          <n-space>
            <n-image
              v-for="(s, idx) in lead.screenshots"
              :key="idx"
              :src="getFileURL(s)"
              width="160"
              height="160"
              object-fit="cover"
              preview-disabled
            />
          </n-space>
        </div>
      </n-gi>
    </n-grid>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NGrid, NGi, NButton, NSpace, NTag, NDescriptions, NDescriptionsItem, NTimeline, NTimelineItem, NImage, useMessage } from 'naive-ui'
import { leadApi, type Lead } from '@/api/modules'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const userStore = useUserStore()

const lead = ref<Lead | null>(null)
const actionLoading = ref(false)

const statusType = computed(() => {
  if (!lead.value) return 'default'
  if (lead.value.status === 'pending') return 'info'
  if (lead.value.status === 'claimed') return 'success'
  return 'default'
})
const statusLabel = computed(() => {
  if (!lead.value) return ''
  return ({ pending: '待认领', claimed: '已认领', archived: '已流失' }[lead.value.status] || lead.value.status)
})

const hasExtension = computed(() => {
  if (!lead.value) return false
  return lead.value.extension && Object.keys(lead.value.extension).length > 0
})

const canClaim = computed(() => {
  const role = userStore.userInfo?.role
  return ['admin', 'b_staff', 'b_lead', 'b_manager'].includes(role || '')
})
const canRelease = computed(() => {
  return userStore.isManager
})
const canArchive = computed(() => {
  if (!lead.value || !userStore.userInfo) return false
  const uid = userStore.userInfo.id
  return lead.value.creator_id === uid || lead.value.claimed_by === uid || userStore.isManager
})

function formatTime(t: string | null) {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN')
}

function getFileURL(path: string) {
  if (!path) return ''
  if (path.startsWith('http')) return path
  if (path.startsWith('/uploads')) return path
  return '/uploads/' + path
}

async function copyToken() {
  if (!lead.value?.token_code) return
  try {
    await navigator.clipboard.writeText(lead.value.token_code)
    message.success('Token 已复制到剪贴板')
  } catch {
    message.error('复制失败，请手动复制')
  }
}

async function loadDetail() {
  try {
    const resp: any = await leadApi.get(Number(route.params.id))
    lead.value = resp.data
  } catch {}
}

async function handleClaim() {
  if (!lead.value) return
  actionLoading.value = true
  try {
    const resp: any = await leadApi.claim(lead.value.id)
    message.success('认领成功')
    // 自动复制 token
    if (resp.data.token_to_copy) {
      try {
        await navigator.clipboard.writeText(resp.data.token_to_copy)
        message.success('Token 已自动复制')
      } catch {}
    }
    loadDetail()
  } catch {} finally {
    actionLoading.value = false
  }
}

async function handleRelease() {
  if (!lead.value) return
  actionLoading.value = true
  try {
    await leadApi.release(lead.value.id)
    message.success('已释放回公开池')
    loadDetail()
  } catch {} finally {
    actionLoading.value = false
  }
}

async function handleArchive() {
  if (!lead.value) return
  actionLoading.value = true
  try {
    await leadApi.archive(lead.value.id)
    message.success('已归档')
    loadDetail()
  } catch {} finally {
    actionLoading.value = false
  }
}

onMounted(loadDetail)
</script>

<style scoped>
.token-box {
  display: flex;
  align-items: center;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
  font-family: 'FiraCode', monospace;
  word-break: break-all;
}

.token-box code {
  flex: 1;
  font-size: 13px;
  color: #1f2937;
}
</style>