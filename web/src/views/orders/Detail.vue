<template>
  <div>
    <div class="page-header">
      <div class="page-title">订单详情 #{{ order.id }}</div>
      <n-button @click="$router.back()">返回</n-button>
    </div>

    <n-spin :show="loading">
      <n-grid :cols="2" :x-gap="16" :y-gap="16">
        <!-- 左：订单信息 -->
        <n-gi>
          <n-card title="订单信息">
            <n-descriptions :column="2" bordered size="small">
              <n-descriptions-item label="订单号">{{ order.order_no }}</n-descriptions-item>
              <n-descriptions-item label="玩家 ID">{{ order.player_id }}</n-descriptions-item>
              <n-descriptions-item label="游戏">{{ order.game?.code }} - {{ order.game?.name }}</n-descriptions-item>
              <n-descriptions-item label="金额">¥{{ order.amount }} {{ order.currency }}</n-descriptions-item>
              <n-descriptions-item label="状态">{{ statusLabel(order.status) }}</n-descriptions-item>
              <n-descriptions-item label="来源">{{ sourceLabel(order.source) }}</n-descriptions-item>
              <n-descriptions-item label="下单时间">{{ formatDate(order.order_time) }}</n-descriptions-item>
              <n-descriptions-item label="支付时间">{{ order.paid_at ? formatDate(order.paid_at) : '-' }}</n-descriptions-item>
              <n-descriptions-item label="合作方">{{ order.partner?.name || '-' }}</n-descriptions-item>
              <n-descriptions-item label="首单">{{ order.is_first_charge ? '是' : '否' }}</n-descriptions-item>
              <n-descriptions-item label="批次号">{{ order.import_batch || '-' }}</n-descriptions-item>
              <n-descriptions-item label="创建时间">{{ formatDate(order.created_at) }}</n-descriptions-item>
              <n-descriptions-item label="备注" :span="2">{{ order.remark || '-' }}</n-descriptions-item>
            </n-descriptions>
          </n-card>

          <n-card title="归属信息" style="margin-top: 16px">
            <n-descriptions :column="2" bordered size="small">
              <n-descriptions-item label="归属状态">
                <n-tag :type="attributionType(order.attribution_status)" size="small">
                  {{ attributionLabel(order.attribution_status) }}
                </n-tag>
              </n-descriptions-item>
              <n-descriptions-item label="匹配 lead 数">{{ order.matched_lead_count }}</n-descriptions-item>
              <n-descriptions-item label="归属时间">{{ order.attributed_at ? formatDate(order.attributed_at) : '-' }}</n-descriptions-item>
              <n-descriptions-item label="归属操作人">{{ order.attributor?.real_name || '-' }}</n-descriptions-item>
              <n-descriptions-item label="关联 Lead" :span="2">
                <span v-if="order.lead">
                  <n-tag>#{{ order.lead.id }}</n-tag>
                  {{ order.lead.player_username }} ({{ order.lead.player_game_name }})
                  - 状态: {{ leadStatusLabel(order.lead.status) }}
                </span>
                <span v-else style="color: #f0a020">未绑定 lead</span>
              </n-descriptions-item>
            </n-descriptions>
          </n-card>

          <!-- 人工绑定 -->
          <n-card v-if="order.attribution_status === 'pending_attribution'" title="人工绑定 Lead" style="margin-top: 16px; border: 2px solid #f0a020">
            <p style="color: #f0a020">该订单未自动归属,请从下方候选列表中选择一个 Lead 完成绑定,或手动指定 lead ID。</p>
            <n-space vertical>
              <n-input v-model:value="manualLeadId" placeholder="候选 lead ID,或手动输入" type="number" />
              <n-button type="primary" :loading="binding" @click="doBind">绑定</n-button>
            </n-space>

            <n-divider>候选 Lead ({{ candidates.length }})</n-divider>
            <n-data-table
              v-if="candidates.length > 0"
              :columns="candidateColumns"
              :data="candidates"
              :bordered="false"
              size="small"
              :pagination="false"
            />
            <n-empty v-else description="无候选 lead" />
          </n-card>
        </n-gi>

        <!-- 右：原始数据 -->
        <n-gi>
          <n-card title="原始数据 (raw_payload)">
            <pre style="background: #f5f7fa; padding: 12px; border-radius: 4px; overflow-x: auto; font-size: 12px;">{{ JSON.stringify(order.raw_payload, null, 2) }}</pre>
          </n-card>
        </n-gi>
      </n-grid>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NCard, NDescriptions, NDescriptionsItem, NTag, NButton, NSpace, NSpin, NInput, NDivider, NDataTable, NEmpty, useMessage, type DataTableColumns } from 'naive-ui'
import { orderApi, type Order, type Lead } from '@/api/modules'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const loading = ref(false)
const binding = ref(false)
const order = ref<Order>({} as Order)
const candidates = ref<Lead[]>([])
const manualLeadId = ref<number | null>(null)

const candidateColumns: DataTableColumns<Lead> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '玩家 ID', key: 'player_id', width: 130 },
  { title: '用户名', key: 'player_username', ellipsis: { tooltip: true } },
  { title: '状态', key: 'status', width: 80, render: (r) => leadStatusLabel(r.status) },
  { title: '渠道', key: 'channel', width: 100 },
  { title: '创建', key: 'created_at', width: 140, render: (r) => formatDate(r.created_at) },
  {
    title: '操作', key: 'actions', width: 80,
    render: (r) => h(NButton, { size: 'small', type: 'primary', text: true, onClick: () => { manualLeadId.value = r.id } }, () => '选用')
  }
]

async function loadDetail() {
  loading.value = true
  try {
    const id = Number(route.params.id)
    const resp: any = await orderApi.get(id)
    order.value = resp.data?.order || {}
    candidates.value = resp.data?.candidates || []
    manualLeadId.value = null
  } catch {
    message.error('加载失败')
  } finally {
    loading.value = false
  }
}

async function doBind() {
  if (!manualLeadId.value) {
    message.warning('请选择 lead')
    return
  }
  binding.value = true
  try {
    await orderApi.bind(order.value.id, manualLeadId.value)
    message.success('绑定成功')
    loadDetail()
  } catch (e: any) {
    message.error(e?.message || '绑定失败')
  } finally {
    binding.value = false
  }
}

function statusLabel(s: string) {
  return ({ paid: '已支付', refunded: '已退款', pending: '待支付', cancelled: '已取消' } as any)[s] || s
}
function sourceLabel(s: string) {
  return ({ manual: '手动上传', api: 'API 拉取', webhook: 'Webhook' } as any)[s] || s
}
function attributionLabel(s: string) {
  return ({ auto_attributed: '自动归属', manual_attributed: '人工绑定', pending_attribution: '待绑定' } as any)[s] || s
}
function attributionType(s: string) {
  return ({ auto_attributed: 'success', manual_attributed: 'success', pending_attribution: 'warning' } as any)[s] || 'default'
}
function leadStatusLabel(s: string) {
  return h(NTag, { size: 'small', type: s === 'claimed' ? 'success' : s === 'archived' ? 'default' : 'warning' as any }, () => ({ pending: '公开池', claimed: '已认领', archived: '已归档' } as any)[s] || s)
}
function formatDate(s: string) {
  if (!s) return '-'
  return new Date(s).toLocaleString('zh-CN', { hour12: false })
}

onMounted(loadDetail)
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-title { font-size: 18px; font-weight: 600; }
</style>