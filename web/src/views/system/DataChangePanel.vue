<template>
  <!-- ========== 数据变更审计(有 before/after 的写操作) ========== -->
  <div>
    <!-- 统计卡:只看写操作 -->
    <n-grid :cols="4" :x-gap="12" :y-gap="12" style="margin-bottom: 16px">
      <n-gi>
        <n-card size="small">
          <div class="stat-label">数据变更总数</div>
          <div class="stat-value">{{ writeTotal }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <div class="stat-label">强制编辑(P13)</div>
          <div class="stat-value" style="color: #d03050">{{ forceEditCount }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <div class="stat-label">线索变更</div>
          <div class="stat-value" style="color: #2080f0">{{ leadCount }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <div class="stat-label">订单变更</div>
          <div class="stat-value" style="color: #18a058">{{ orderCount }}</div>
        </n-card>
      </n-gi>
    </n-grid>

    <n-alert type="info" :bordered="false" style="margin-bottom: 12px" size="small">
      本页只显示<strong>数据写操作</strong>(创建/更新/删除/导入),带完整的变更前(after)快照。
      登录登出等认证事件见「操作日志」tab。
    </n-alert>

    <!-- 过滤 -->
    <n-card size="small" style="margin-bottom: 12px">
      <n-space :wrap="false">
        <n-select v-model:value="filter.action" :options="actionOptions" placeholder="动作" clearable style="width: 160px" @update:value="loadList" />
        <n-select v-model:value="filter.target_type" :options="targetTypeOptions" placeholder="对象类型" clearable style="width: 150px" @update:value="loadList" />
        <n-input v-model:value="filter.username" placeholder="操作人" clearable style="width: 130px" @keyup.enter="loadList" @clear="loadList" />
        <n-input v-model:value="filter.target_id" placeholder="对象 ID" clearable style="width: 110px" @keyup.enter="loadList" @clear="loadList" />
        <n-date-picker v-model:value="dateRange" type="daterange" clearable style="width: 250px" @update:value="onDateChange" />
        <n-button @click="loadList" :loading="loading">查询</n-button>
        <n-button quaternary @click="resetFilter">重置</n-button>
      </n-space>
    </n-card>

    <n-data-table :columns="columns" :data="logs" :loading="loading" :bordered="false" :pagination="pagination" :row-key="(r: any) => r.id" />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, h, onMounted, watch } from 'vue'
import { NCard, NDataTable, NButton, NSpace, NTag, NSelect, NInput, NDatePicker, NGrid, NGi, NAlert, useMessage, useDialog, type DataTableColumns } from 'naive-ui'
import http from '@/api/client'

const props = defineProps<{ active: boolean }>()

const message = useMessage()
const dialog = useDialog()
const logs = ref<any[]>([])
const loading = ref(false)
const dateRange = ref<[number, number] | null>(null)

// 只看写操作
const WRITE_ACTIONS = ['create', 'update', 'delete', 'order_import', 'order_webhook', 'partner_pull', 'password_reset', 'order_bind']

const filter = reactive({
  action: null as string | null,
  target_type: null as string | null,
  username: '' as string,
  target_id: '' as string,
  start_date: '' as string,
  end_date: '' as string
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
  showSizePicker: true,
  pageSizes: [20, 50, 100],
  itemCount: 0,
  onChange: (p: number) => { pagination.page = p; loadList() },
  onUpdatePageSize: (s: number) => { pagination.pageSize = s; pagination.page = 1; loadList() }
})

const writeTotal = ref(0)
const forceEditCount = ref(0)
const leadCount = ref(0)
const orderCount = ref(0)

const actionOptions = [
  { label: '创建', value: 'create' },
  { label: '更新 / 强制编辑', value: 'update' },
  { label: '删除', value: 'delete' },
  { label: '人工绑定订单', value: 'order_bind' },
  { label: 'CSV/Excel 导入', value: 'order_import' },
  { label: 'Webhook 入库', value: 'order_webhook' },
  { label: 'API 拉取', value: 'partner_pull' },
  { label: '重置密码', value: 'password_reset' }
]
const targetTypeOptions = [
  { label: '用户', value: 'user' },
  { label: '部门', value: 'department' },
  { label: '游戏', value: 'game' },
  { label: '字典', value: 'dict' },
  { label: '线索', value: 'lead' },
  { label: '订单', value: 'order' },
  { label: '广播模板', value: 'broadcast_template' },
  { label: '上传模板', value: 'upload_template' },
  { label: '系统', value: 'system' }
]

function actionLabel(s: string) {
  return actionOptions.find(o => o.value === s)?.label || s
}

const columns: DataTableColumns<any> = [
  { title: '时间', key: 'created_at', width: 165, render: (r) => new Date(r.created_at).toLocaleString('zh-CN', { hour12: false }) },
  { title: '操作人', key: 'username', width: 100, render: (r) => r.username || h(NTag, { size: 'small', type: 'warning' }, () => '匿名') },
  {
    title: '动作', key: 'action', width: 150,
    render: (r) => {
      const isForce = r.extension?.force_edit === true
      return h(NTag, { size: 'small', type: isForce ? 'error' : 'info', bordered: isForce }, () => isForce ? '⚡ 强制编辑' : actionLabel(r.action))
    }
  },
  { title: '对象', key: 'target_label', width: 180, ellipsis: { tooltip: true } },
  {
    title: '变更字段', key: 'changed', width: 220, ellipsis: { tooltip: true },
    render: (r) => {
      const cf = r.extension?.changed_fields
      if (Array.isArray(cf) && cf.length > 0) {
        return h('span', { style: 'color:#d03050' }, `${cf.length} 个: ${cf.slice(0, 5).join(', ')}${cf.length > 5 ? '…' : ''}`)
      }
      return h('span', { style: 'color:#999' }, '—')
    }
  },
  {
    title: '详情', key: 'details', width: 80, fixed: 'right',
    render: (r) => h(NButton, { size: 'small', text: true, type: 'primary', onClick: () => showDiff(r) }, () => '对比')
  }
]

// 逐字段 diff 弹窗
function showDiff(row: any) {
  const before = row.before || {}
  const after = row.after || {}
  const keys = Array.from(new Set([...Object.keys(before), ...Object.keys(after)])).sort()

  if (keys.length === 0) {
    message.info('该记录没有 before/after 快照')
    return
  }

  const rows = keys.map((k) => {
    const b = before[k]
    const a = after[k]
    const changed = JSON.stringify(b) !== JSON.stringify(a)
    return {
      key: k,
      field: k,
      before: b === undefined || b === null || b === '' ? '—' : String(typeof b === 'object' ? JSON.stringify(b) : b),
      after: a === undefined || a === null || a === '' ? '—' : String(typeof a === 'object' ? JSON.stringify(a) : a),
      changed
    }
  })
  const changedCount = rows.filter(r => r.changed).length

  dialog.info({
    title: `${row.action} · ${row.target_label || row.target_type} #${row.target_id || '-'}`,
    content: `${changedCount} 个字段发生变化 / 共 ${rows.length} 个字段`,
    positiveText: '关闭',
    render: () => h('div', { style: 'max-height:60vh;overflow:auto' }, [
      h('table', { style: 'width:100%;border-collapse:collapse;font-size:12px' }, [
        h('thead', null, [h('tr', null, [
          h('th', { style: 'text-align:left;padding:4px;border-bottom:1px solid #e5e7eb' }, '字段'),
          h('th', { style: 'text-align:left;padding:4px;border-bottom:1px solid #e5e7eb' }, '变更前'),
          h('th', { style: 'text-align:left;padding:4px;border-bottom:1px solid #e5e7eb' }, '变更后')
        ])]),
        h('tbody', null, rows.map(r => h('tr', { style: r.changed ? 'background:#fff8f8' : '' }, [
          h('td', { style: 'padding:3px 4px;font-weight:600' }, r.field),
          h('td', { style: 'padding:3px 4px;color:#9ca3af' }, r.before),
          h('td', { style: 'padding:3px 4px;color:#059669;font-weight:600' }, r.after)
        ])))
      ])
    ])
  })
}

async function loadList() {
  loading.value = true
  try {
    const params: any = { page: pagination.page, page_size: pagination.pageSize }
    if (filter.action) params.action = filter.action
    if (filter.target_type) params.target_type = filter.target_type
    if (filter.username) params.username = filter.username
    if (filter.target_id) params.target_id = Number(filter.target_id)
    if (filter.start_date) params.start_date = filter.start_date
    if (filter.end_date) params.end_date = filter.end_date

    const resp: any = await http.get('/audit/logs', { params })
    let list = resp.data?.list || []

    // 未指定 action 时,前端只留写操作
    // (后端 action 是单值过滤,这里按 write 白名单再筛一次)
    if (!filter.action) {
      list = list.filter((r: any) => WRITE_ACTIONS.includes(r.action))
    }

    logs.value = list
    // 未指定 action 时 total 是全量数,会跟列表对不上 —— 用过滤后的本页数做兜底显示
    pagination.itemCount = filter.action ? (resp.data?.total || 0) : (list.length < pagination.pageSize ? list.length : resp.data?.total || 0)
  } catch {
    message.error('加载失败')
  } finally {
    loading.value = false
  }
}

async function loadStats() {
  try {
    const r: any = await http.get('/audit/write-stats')
    const d = r.data || {}
    writeTotal.value = d.total || 0
    forceEditCount.value = d.force_edit || 0
    leadCount.value = d.lead_count || 0
    orderCount.value = d.order_count || 0
  } catch {
    writeTotal.value = forceEditCount.value = leadCount.value = orderCount.value = 0
  }
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
  filter.action = null
  filter.target_type = null
  filter.username = ''
  filter.target_id = ''
  filter.start_date = ''
  filter.end_date = ''
  dateRange.value = null
  pagination.page = 1
  loadList()
}

watch(() => props.active, (v) => { if (v) { loadList(); loadStats() } })
onMounted(() => { loadList(); loadStats() })
</script>

<style scoped>
.stat-label { color: #999; font-size: 12px; margin-bottom: 4px; }
.stat-value { font-size: 22px; font-weight: 700; color: #333; }
</style>