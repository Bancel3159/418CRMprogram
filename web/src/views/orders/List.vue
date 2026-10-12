<template>
  <div>
    <div class="page-header">
      <n-space>
        <n-button v-if="userStore.isAdmin" @click="goImport">📤 手动导入</n-button>
        <n-button @click="loadList" :loading="loading">刷新</n-button>
      </n-space>
    </div>

    <!-- 顶部统计卡 -->
    <n-grid :cols="5" :x-gap="12" :y-gap="12" style="margin-bottom: 16px">
      <n-gi>
        <n-card size="small">
          <div class="stat-label">总营收</div>
          <div class="stat-value" style="color: #18a058">¥{{ formatMoney(stats.total_revenue) }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <div class="stat-label">订单数(已支付)</div>
          <div class="stat-value">{{ stats.total_orders }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <div class="stat-label">付费玩家</div>
          <div class="stat-value">{{ stats.paid_players }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <div class="stat-label">ARPU</div>
          <div class="stat-value">¥{{ formatMoney(stats.arpu) }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <div class="stat-label">待人工绑定</div>
          <div class="stat-value" style="color: #f0a020">{{ stats.pending_orders }}</div>
        </n-card>
      </n-gi>
    </n-grid>

    <!-- 过滤栏 -->
    <n-card size="small" style="margin-bottom: 12px">
      <n-space :wrap="false">
        <n-select v-model:value="filter.attribution_status" :options="attributionOptions" placeholder="归属状态" clearable style="width: 140px" @update:value="loadList" />
        <n-select v-model:value="filter.source" :options="sourceOptions" placeholder="来源" clearable style="width: 120px" @update:value="loadList" />
        <n-select v-model:value="filter.status" :options="statusOptions" placeholder="订单状态" clearable style="width: 120px" @update:value="loadList" />
        <n-select v-model:value="filter.game_id" :options="gameOptions" placeholder="游戏" clearable filterable style="width: 160px" @update:value="loadList" />
        <n-input v-model:value="filter.order_no" placeholder="订单号" clearable style="width: 180px" @keyup.enter="loadList" @clear="loadList" />
        <n-input v-model:value="filter.player_id" placeholder="玩家 ID" clearable style="width: 140px" @keyup.enter="loadList" @clear="loadList" />
        <n-date-picker v-model:value="dateRange" type="daterange" clearable style="width: 260px" @update:value="onDateChange" />
        <n-button @click="loadList">查询</n-button>
        <n-button quaternary @click="resetFilter">重置</n-button>
      </n-space>
    </n-card>

    <!-- 表格：列由默认视图配置 -->
    <n-data-table
      :columns="dynamicColumns"
      :data="orders"
      :loading="loading"
      :bordered="false"
      :pagination="pagination"
      :row-key="(row: any) => row.id"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, h, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NSpace, NTag, NSelect, NInput, NDatePicker, NCard, NGrid, NGi, NDataTable, useMessage, type DataTableColumns } from 'naive-ui'
import { orderApi, gameApi, listViewApi, type Order, type OrderStats, type ListViewField } from '@/api/modules'
import { useUserStore } from '@/stores/user'

const message = useMessage()
const router = useRouter()
const userStore = useUserStore()

const orders = ref<Order[]>([])
const stats = ref<OrderStats>({ total_revenue: 0, total_orders: 0, paid_players: 0, arpu: 0, pending_orders: 0, a_team_revenue: 0, a_team_orders: 0, b_team_revenue: 0, b_team_orders: 0 })
const loading = ref(false)
const total = ref(0)

const filter = reactive({
  attribution_status: 'pending_attribution' as string | null,
  source: null as string | null,
  status: null as string | null,
  game_id: null as number | null,
  order_no: '' as string,
  player_id: '' as string,
  start_date: '' as string,
  end_date: '' as string
})

const dateRange = ref<[number, number] | null>(null)
const gameOptions = ref<{ label: string; value: number }[]>([])

const pagination = reactive({
  page: 1,
  pageSize: 20,
  showSizePicker: true,
  pageSizes: [20, 50, 100],
  itemCount: 0,
  onChange: (p: number) => { pagination.page = p; loadList() },
  onUpdatePageSize: (s: number) => { pagination.pageSize = s; pagination.page = 1; loadList() }
})

const attributionOptions = [
  { label: '待人工绑定', value: 'pending_attribution' },
  { label: '自动归属', value: 'auto_attributed' },
  { label: '人工绑定', value: 'manual_attributed' }
]
const sourceOptions = [
  { label: '手动上传', value: 'manual' },
  { label: 'API 拉取', value: 'api' },
  { label: 'Webhook', value: 'webhook' }
]
const statusOptions = [
  { label: '已支付', value: 'paid' },
  { label: '已退款', value: 'refunded' },
  { label: '待支付', value: 'pending' },
  { label: '已取消', value: 'cancelled' }
]

// ===== 视图配置 + 动态列 =====
const viewFields = ref<ListViewField[]>([])

async function loadDefaultView() {
  try {
    const resp: any = await listViewApi.getDefault('order')
    viewFields.value = (resp.data?.fields || []) as ListViewField[]
  } catch (e) {
    viewFields.value = []
  }
}

const dynamicColumns = computed<DataTableColumns<Order>>(() => {
  // 操作列固定加在最后
  const cols: DataTableColumns<Order> = viewFields.value.map((f) => ({
    title: f.label,
    key: f.code,
    width: f.width || undefined,
    align: (f.align as any) || 'left',
    ellipsis: { tooltip: true },
    render: renderCellFor(f)
  }))
  cols.push({
    title: '操作',
    key: 'actions',
    width: 90,
    render: (r: Order) => h(NButton, { size: 'small', text: true, type: 'primary', onClick: () => viewDetail(r) }, () => '详情')
  })
  return cols
})

function renderCellFor(f: ListViewField) {
  const code = f.code
  const fmt = f.formatter || 'default'

  // raw_payload.<key>
  if (code.startsWith('raw_payload.')) {
    const key = code.substring('raw_payload.'.length)
    return (row: Order) => {
      const val = (row as any).raw_payload?.[key]
      if (val === undefined || val === null || val === '') return '—'
      return formatByFormatter(fmt, val, code)
    }
  }

  // lead_joined.* - 通过 row.lead.* 取
  if (code.startsWith('lead_')) {
    const leadField = code.substring('lead_'.length)
    const leadAccessor: Record<string, string> = {
      player_username: 'player_username',
      player_game_name: 'player_game_name',
      player_country: 'player_country',
      region: 'region',
      is_target: 'is_target',
      creator: '__creator_name',
      claimer: '__claimer_name'
    }
    return (row: Order) => {
      const lead = (row as any).lead
      if (!lead) return '—'
      let val: any
      if (leadAccessor[leadField] === '__creator_name') val = lead.creator?.real_name || lead.creator?.username
      else if (leadAccessor[leadField] === '__claimer_name') val = lead.claimed_by_user?.real_name || lead.claimed_by_user?.username
      else val = lead[leadAccessor[leadField]]
      if (val === undefined || val === null || val === '') return '—'
      return formatByFormatter(fmt, val, code)
    }
  }

  return (row: Order) => {
    const val = getRaw(row, code)
    if (val === undefined || val === null || val === '') return '—'
    return formatByFormatter(fmt, val, code)
  }
}

function getRaw(row: any, code: string): any {
  if (code in row) return row[code]
  const parts = code.split('.')
  let cur: any = row
  for (const p of parts) {
    if (cur == null) return ''
    cur = cur[p]
  }
  return cur
}

function formatByFormatter(fmt: string, val: any, _code: string): any {
  switch (fmt) {
    case 'datetime':
      return new Date(val).toLocaleString('zh-CN', { hour12: false })
    case 'date':
      return new Date(val).toLocaleDateString('zh-CN')
    case 'number':
      return typeof val === 'number' ? val.toLocaleString('zh-CN') : String(val)
    case 'money':
      return `¥${Number(val || 0).toFixed(2)}`
    case 'order_status_tag': {
      const map: Record<string, { t: string; l: string }> = {
        paid: { t: 'success', l: '已支付' },
        refunded: { t: 'default', l: '已退款' },
        pending: { t: 'warning', l: '待支付' },
        cancelled: { t: 'error', l: '已取消' }
      }
      const m = map[val as string] || { t: 'default', l: String(val) }
      return h(NTag, { type: m.t as any, size: 'small' }, () => m.l)
    }
    case 'source_tag': {
      const map: Record<string, { t: string; l: string }> = {
        manual: { t: 'info', l: '手动' },
        api: { t: 'info', l: 'API' },
        webhook: { t: 'info', l: 'Webhook' }
      }
      const m = map[val as string] || { t: 'default', l: String(val) }
      return h(NTag, { type: m.t as any, size: 'small' }, () => m.l)
    }
    case 'attribution_tag': {
      const map: Record<string, { t: string; l: string }> = {
        auto_attributed: { t: 'success', l: '自动' },
        manual_attributed: { t: 'success', l: '人工' },
        pending_attribution: { t: 'warning', l: '待绑定' }
      }
      const m = map[val as string] || { t: 'default', l: String(val) }
      return h(NTag, { type: m.t as any, size: 'small' }, () => m.l)
    }
    case 'first_charge_tag':
      return h(NTag, { size: 'small', type: val ? 'success' : 'default' }, () => val ? '首单' : '否')
    case 'target_tag':
      return h(NTag, { size: 'small', type: val === 'target' ? 'success' : 'default' }, () => val === 'target' ? '目标' : '非目标')
    default:
      return String(val)
  }
}

// ===== 数据加载 =====
async function loadList() {
  loading.value = true
  try {
    const params: any = {
      page: pagination.page,
      page_size: pagination.pageSize
    }
    if (filter.attribution_status) params.attribution_status = filter.attribution_status
    if (filter.source) params.source = filter.source
    if (filter.status) params.status = filter.status
    if (filter.game_id) params.game_id = filter.game_id
    if (filter.order_no) params.order_no = filter.order_no
    if (filter.player_id) params.player_id = filter.player_id
    if (filter.start_date) params.start_date = filter.start_date
    if (filter.end_date) params.end_date = filter.end_date

    const resp: any = await orderApi.list(params)
    orders.value = resp.data?.list || []
    total.value = resp.data?.total || 0
    pagination.itemCount = total.value
  } catch (e) {
    message.error('加载失败')
  } finally {
    loading.value = false
  }
}

async function loadStats() {
  try {
    const resp: any = await orderApi.stats()
    stats.value = resp.data?.overview || stats.value
  } catch {}
}

async function loadGames() {
  try {
    const resp: any = await gameApi.list()
    gameOptions.value = (resp.data || []).map((g: any) => ({ label: `${g.code} - ${g.name}`, value: g.id }))
  } catch {}
}

function onDateChange(v: [number, number] | null) {
  if (v) {
    const fmt = (ts: number) => new Date(ts).toISOString().slice(0, 10)
    filter.start_date = fmt(v[0])
    filter.end_date = fmt(v[1])
  } else {
    filter.start_date = ''
    filter.end_date = ''
  }
  loadList()
}

function resetFilter() {
  filter.attribution_status = null
  filter.source = null
  filter.status = null
  filter.game_id = null
  filter.order_no = ''
  filter.player_id = ''
  filter.start_date = ''
  filter.end_date = ''
  dateRange.value = null
  pagination.page = 1
  loadList()
}

function viewDetail(row: Order) {
  router.push(`/orders/${row.id}`)
}

function goImport() {
  router.push('/orders/import')
}

function formatMoney(v: number) {
  return (v || 0).toFixed(2)
}

onMounted(async () => {
  await loadDefaultView()
  loadGames()
  loadList()
  loadStats()
})
</script>

<style scoped>
.page-header { display: flex; justify-content: flex-end; align-items: center; margin-bottom: 16px; }
.stat-label { color: #999; font-size: 12px; margin-bottom: 4px; }
.stat-value { font-size: 22px; font-weight: 700; color: #333; }
</style>