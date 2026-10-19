<template>
  <div v-if="lead">
    <div class="page-header">
      <div>
        <span class="page-title">玩家 #{{ lead.id }}</span>
        <n-tag :type="statusType" size="small" style="margin-left: 8px">{{ statusLabel }}</n-tag>
        <n-tag v-if="lead.is_lost" type="error" size="small" style="margin-left: 8px">
           已流失 {{ lead.lost_reason ? `(${lostReasonLabel})` : '' }}
        </n-tag>
        <n-tag v-if="lead.reactivated_at" type="success" size="small" style="margin-left: 8px">
           已激活
        </n-tag>
      </div>
      <n-space>
        <n-button @click="$router.back()">返回</n-button>
        <n-button v-if="lead.status === 'claimed' && canMarkPaid && !lead.is_paid" :loading="markPaidLoading" type="primary" @click="handleMarkPaid">
           标记已付费
        </n-button>
        <n-button v-if="lead.status === 'claimed' && canMarkPaid && lead.is_paid" :loading="markPaidLoading" quaternary @click="handleUnmarkPaid">
          取消已付费标记
        </n-button>
        <n-button v-if="lead.status === 'claimed' && !lead.is_lost && isManager" :loading="markLostLoading" type="error" quaternary @click="handleMarkLost">
           手动标记流失
        </n-button>
        <n-button v-if="lead.is_lost && isManager" :loading="markActiveLoading" type="success" quaternary @click="handleMarkActive">
           手动激活
        </n-button>
        <n-button v-if="lead.status === 'claimed'" :loading="actionLoading" @click="handleArchive">归档</n-button>
      </n-space>
    </div>

    <!-- 付费状态横幅 -->
    <div v-if="lead.is_paid" class="paid-banner">
      <span class="paid-icon"></span>
      <span class="paid-text">
        <b>本线索已计入「已付费」</b>
        <small class="muted">
          {{ lead.paid_auto ? '订单系统自动识别' : '' }}
          {{ lead.paid_mark ? (lead.paid_auto ? ' + ' : '') + '人工标记' : '' }}
          <span v-if="lead.paid_marked_at"> · 标记于 {{ formatTime(lead.paid_marked_at) }}</span>
        </small>
      </span>
    </div>

    <!-- 流失提示 -->
    <n-alert v-if="lead.is_lost" type="error" :title="`该玩家已流失`" style="margin-bottom: 16px" :show-icon="false">
      <div>流失时间:{{ formatTime(lead.lost_at) }}</div>
      <div>流失原因:{{ lostReasonLabel }}</div>
      <div v-if="lead.reactivated_at">最近激活:{{ formatTime(lead.reactivated_at) }}</div>
    </n-alert>

    <n-grid :cols="2" :x-gap="16" :y-gap="16">
      <n-gi>
        <n-card title="基本信息">
          <n-descriptions :column="1" label-placement="left" label-width="120" bordered>
            <n-descriptions-item label="游戏">{{ lead.game?.code }} - {{ lead.game?.name }}</n-descriptions-item>
            <n-descriptions-item label="玩家 ID">{{ lead.player_id }}</n-descriptions-item>
            <n-descriptions-item label="玩家游戏名">{{ lead.player_game_name }}</n-descriptions-item>
            <n-descriptions-item label="玩家国家">{{ lead.player_country }}</n-descriptions-item>
            <n-descriptions-item label="区服">{{ lead.region }}</n-descriptions-item>
            <n-descriptions-item label="年龄">{{ lead.player_age }}</n-descriptions-item>
            <n-descriptions-item label="渠道">{{ lead.channel }}</n-descriptions-item>
            <n-descriptions-item label="社群">{{ lead.community || '-' }}</n-descriptions-item>
            <n-descriptions-item label="是否目标">
              <n-tag :type="lead.is_target === 'target' ? 'success' : 'default'" size="small">
                {{ lead.is_target === 'target' ? '目标' : '非目标' }}
              </n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="用户名">{{ lead.player_username }}</n-descriptions-item>
          </n-descriptions>
        </n-card>
      </n-gi>

      <n-gi>
        <n-card title=" Discord Token">
          <template #header-extra>
            <n-tag type="warning" size="small">公司自有账号</n-tag>
          </template>

          <div v-if="lead.token_code" class="token-box">
            <code>{{ lead.token_code }}</code>
          </div>
          <div v-else class="text-muted">（无 token）</div>

          <n-space style="margin-top: 12px" v-if="lead.token_code">
            <n-button type="primary" size="small" @click="copyToken">
               复制到剪贴板
            </n-button>
            <n-button size="small" @click="openDiscord">
               启动 Discord
            </n-button>
          </n-space>

          <p class="text-muted" style="margin-top: 12px; font-size: 12px">
            在 Discord 客户端按 Ctrl+V 粘贴 token 即可登录
          </p>
        </n-card>

        <n-card title="流转记录" style="margin-top: 16px">
          <n-timeline>
            <n-timeline-item type="info" :title="'提交'" :time="formatTime(lead.created_at)">
              {{ lead.creator?.real_name || lead.creator?.username }} 拉新提交
            </n-timeline-item>
            <n-timeline-item v-if="lead.claimed_at" type="success" title="认领" :time="formatTime(lead.claimed_at)">
              {{ lead.claimed_by_user?.real_name || lead.claimed_by_user?.username }} 接手维护
            </n-timeline-item>
            <n-timeline-item v-if="lead.released_at" :type="lead.status === 'archived' ? 'default' : 'warning'" :title="lead.status === 'archived' ? '归档' : '释放'" :time="formatTime(lead.released_at)">
              {{ lead.released_by_user?.real_name || '-' }} 操作
            </n-timeline-item>
            <n-timeline-item v-if="lead.is_lost && lead.lost_at" type="error" title="流失" :time="formatTime(lead.lost_at)">
              {{ lostReasonLabel }}
            </n-timeline-item>
            <n-timeline-item v-if="lead.reactivated_at" type="success" title="激活" :time="formatTime(lead.reactivated_at)">
              玩家重新上线
            </n-timeline-item>
          </n-timeline>
        </n-card>
      </n-gi>

      <n-gi v-if="lead.remark">
        <n-card title="备注">{{ lead.remark }}</n-card>
      </n-gi>
    </n-grid>

    <!-- P11 跟进表单(只有已认领的 lead 可以跟进) -->
    <n-card v-if="lead.status === 'claimed'" title="写跟进记录" style="margin-top: 16px">
      <n-form :model="followForm" :rules="followRules" ref="formRef" label-placement="left" label-width="80">
        <n-form-item label="类型" path="follow_type">
          <n-select v-model:value="followForm.follow_type" :options="followTypeOptions" style="width: 180px" />
        </n-form-item>
        <n-form-item label="跟进内容" path="content">
          <n-input
            v-model:value="followForm.content"
            type="textarea"
            placeholder="必填。记录与玩家的互动内容..."
            :autosize="{ minRows: 3, maxRows: 6 }"
          />
        </n-form-item>
        <n-space justify="end">
          <n-button @click="resetFollowForm">清空</n-button>
          <n-button type="primary" :loading="followSubmitting" @click="handleSubmitFollow">
             提交跟进
          </n-button>
        </n-space>
      </n-form>
    </n-card>

    <!-- P11 跟进历史 -->
    <n-card title="跟进历史 ({{ followRecords.length }})" style="margin-top: 16px">
      <n-empty v-if="followRecords.length === 0" description="暂无跟进记录" />
      <n-timeline v-else>
        <n-timeline-item
          v-for="rec in followRecords"
          :key="rec.id"
          :type="rec.follow_type === 'call' ? 'warning' : 'info'"
          :title="followTypeLabel(rec.follow_type)"
          :time="formatTime(rec.created_at)"
        >
          <div style="white-space: pre-wrap">{{ rec.content }}</div>
          <div style="font-size: 11px; color: #94a3b8; margin-top: 4px">
            跟进人 ID:{{ rec.follower_id || '系统' }}
          </div>
        </n-timeline-item>
      </n-timeline>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, reactive, watch } from 'vue'
import { useRoute } from 'vue-router'
import {
  NCard, NButton, NSpace, NGrid, NGi, NTag, NDescriptions, NDescriptionsItem,
  NTimeline, NTimelineItem, NInput, NSelect, NForm, NFormItem, NAlert, NEmpty,
  useMessage
} from 'naive-ui'
import { leadApi, followRecordApi, followApi } from '@/api/modules'
import { useUserStore } from '@/stores/user'
import { useWSStore } from '@/stores/ws'

const route = useRoute()
const message = useMessage()
const userStore = useUserStore()

const lead = ref<any>(null)
const actionLoading = ref(false)
const markPaidLoading = ref(false)
const markLostLoading = ref(false)
const markActiveLoading = ref(false)
const followSubmitting = ref(false)
const formRef = ref<any>(null)
const followRecords = ref<any[]>([])

// 跟进表单
const followForm = reactive({
  follow_type: 'text',
  content: ''
})

const followRules = {
  content: { required: true, message: '跟进内容不能为空', trigger: ['blur', 'input'] }
}

const followTypeOptions = [
  { label: '文字沟通', value: 'text' },
  { label: '电话', value: 'call' },
  { label: '在线', value: 'online' },
  { label: '微信', value: 'wechat' },
  { label: ' Discord', value: 'discord' },
  { label: '邮件', value: 'email' }
]

const followTypeLabel = (t: string) => followTypeOptions.find(o => o.value === t)?.label || ` ${t}`

// 当前用户是否能手动标记付费 / 流失 / 激活(主管及以上)
const canMarkPaid = computed(() => {
  const role = userStore.userInfo?.role
  return ['admin', 'a_manager', 'a_lead', 'b_manager', 'b_lead'].includes(role)
})
const isManager = computed(() => {
  const role = userStore.userInfo?.role
  return ['admin', 'a_manager', 'b_manager'].includes(role)
})

const statusType = computed(() => {
  if (!lead.value) return 'default'
  return ({ pending: 'info', claimed: 'success', archived: 'default' }[lead.value.status] || 'default')
})
const statusLabel = computed(() => {
  if (!lead.value) return ''
  return ({ pending: '待认领', claimed: '已对接', archived: '已流失' }[lead.value.status] || lead.value.status)
})

const lostReasonLabel = computed(() => {
  const r = lead.value?.lost_reason
  if (!r) return ''
  return ({
    auto_new_register_timeout: '新注册跟进超时',
    auto_paid_offline: '付费玩家离线超期',
    manual: '手动标记'
  }[r] || r)
})

function formatTime(t: string | null) {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN')
}

async function copyToken() {
  if (!lead.value?.token_code) return
  try {
    await navigator.clipboard.writeText(lead.value.token_code)
    message.success(' Token 已复制到剪贴板')
  } catch {
    message.error('复制失败，请手动选中复制')
  }
}

function openDiscord() {
  window.open('https://discord.com/login', '_blank')
  message.info('请在 Discord 中粘贴 token 登录')
}

async function handleArchive() {
  if (!confirm('确认归档？归档后该玩家不再出现在活跃列表。')) return
  actionLoading.value = true
  try {
    await leadApi.archive(lead.value.id)
    message.success('已归档')
    loadDetail()
  } catch (err: any) {
    message.error(err?.message || '操作失败')
  } finally {
    actionLoading.value = false
  }
}

async function handleMarkPaid() {
  if (!lead.value) return
  if (!confirm('确认将该线索标记为「已付费」？')) return
  markPaidLoading.value = true
  try {
    const resp: any = await leadApi.markPaid(lead.value.id)
    lead.value = resp.data
    message.success('已标记为已付费')
  } catch (err: any) {
    message.error(err?.message || '标记失败')
  } finally {
    markPaidLoading.value = false
  }
}

async function handleUnmarkPaid() {
  if (!lead.value) return
  if (!confirm('确认取消「已付费」标记？')) return
  markPaidLoading.value = true
  try {
    const resp: any = await leadApi.unmarkPaid(lead.value.id)
    lead.value = resp.data
    message.success('已取消标记')
  } finally {
    markPaidLoading.value = false
  }
}

async function handleMarkLost() {
  if (!confirm('确认手动标记该玩家为「流失」？')) return
  markLostLoading.value = true
  try {
    await followApi.markLost(lead.value.id)
    message.success('已标记流失')
    loadDetail()
  } catch (err: any) {
    message.error(err?.message || '标记失败')
  } finally {
    markLostLoading.value = false
  }
}

async function handleMarkActive() {
  if (!confirm('确认手动激活该玩家?(流失状态会清除)')) return
  markActiveLoading.value = true
  try {
    await followApi.markActive(lead.value.id)
    message.success('已激活')
    loadDetail()
  } catch (err: any) {
    message.error(err?.message || '激活失败')
  } finally {
    markActiveLoading.value = false
  }
}

async function handleSubmitFollow() {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch { return }

  followSubmitting.value = true
  try {
    const resp: any = await followRecordApi.create(lead.value.id, {
      follow_type: followForm.follow_type,
      content: followForm.content
    })
    message.success('跟进已记录(今日待办已完成)')
    // 重置表单
    followForm.content = ''
    // 刷新跟进记录
    await loadFollowRecords()
    // 刷新 lead 信息(last_follow_at 等)
    await loadDetail()
  } catch (err: any) {
    message.error(err?.message || '提交失败')
  } finally {
    followSubmitting.value = false
  }
}

function resetFollowForm() {
  followForm.follow_type = 'text'
  followForm.content = ''
}

async function loadDetail() {
  try {
    const resp: any = await leadApi.get(Number(route.params.id))
    lead.value = resp.data
  } catch {}
}

async function loadFollowRecords() {
  try {
    const resp: any = await followRecordApi.list(Number(route.params.id))
    followRecords.value = resp.data?.list || []
  } catch {
    followRecords.value = []
  }
}

onMounted(async () => {
  if (!userStore.userInfo) {
    try { await userStore.fetchMe() } catch {}
  }
  await loadDetail()
  await loadFollowRecords()
})

// P12 实时事件:当前 lead 收到 order.* / player.react  自动刷新详情
const wsStore = useWSStore()
const currentLeadId = computed(() => Number(route.params.id))

watch(
  () => wsStore.lastMessage,
  (msg) => {
    if (!msg || msg.channel !== 'follow:realtime') return
    const data = msg.data || {}
    const targetLeadId = data.lead_id
    if (!targetLeadId || targetLeadId !== currentLeadId.value) return
    // 收到当前 lead 的实时事件,刷新详情(不弹 toast — MainLayout 已经弹过)
    loadDetail()
  }
)
</script>

<style scoped>
.token-box {
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
  word-break: break-all;
  font-family: 'FiraCode', monospace;
  font-size: 13px;
}

/* 已付费横幅 */
.paid-banner {
  margin: 12px 0;
  padding: 10px 14px;
  border-radius: 10px;
  background: linear-gradient(135deg, #d1fae5, #a7f3d0);
  border: 1px solid #6ee7b7;
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: #064e3b;
}
.paid-icon { font-size: 22px; }
.paid-text { display: flex; flex-direction: column; gap: 2px; }
.paid-text b { color: #065f46; font-weight: 700; }
.muted { color: rgba(6, 78, 59, 0.7); font-size: 11px; }
</style>
