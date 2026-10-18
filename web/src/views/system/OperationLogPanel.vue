<template>
  <div>
    <!-- 统计卡 -->
    <n-grid :cols="4" :x-gap="12" :y-gap="12" style="margin-bottom: 16px">
      <n-gi>
        <n-card size="small">
          <div class="stat-label">近 7 天总操作</div>
          <div class="stat-value">{{ stats.total || 0 }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <div class="stat-label">登录失败</div>
          <div class="stat-value" style="color: #d03050">{{ stats.failed_login || 0 }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <div class="stat-label">业务动作(claim/release/archive/bind)</div>
          <div class="stat-value" style="color: #2080f0">{{ stats.business_actions || 0 }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <div class="stat-label">数据写入(CUD)</div>
          <div class="stat-value" style="color: #18a058">{{ stats.write_actions || 0 }}</div>
        </n-card>
      </n-gi>
    </n-grid>

    <!-- 动作分布 TOP + 用户 TOP -->
    <n-grid :cols="2" :x-gap="16" :y-gap="12" style="margin-bottom: 16px">
      <n-gi>
        <n-card title="📊 动作分布(近 7 天)" size="small">
          <div v-if="byAction.length === 0" class="text-muted">暂无数据</div>
          <div v-else>
            <div v-for="row in byAction" :key="row.action" style="margin-bottom: 8px">
              <div style="display: flex; justify-content: space-between; margin-bottom: 2px; font-size: 13px">
                <span>{{ actionLabel(row.action) }}</span>
                <strong>{{ row.count }}</strong>
              </div>
              <n-progress type="line" :percentage="(row.count / byAction[0].count) * 100" :show-indicator="false" :height="6" color="#2080f0" />
            </div>
          </div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card title="👤 活跃用户(近 7 天 TOP 10)" size="small">
          <div v-if="byUser.length === 0" class="text-muted">暂无数据</div>
          <div v-else>
            <div v-for="(row, idx) in byUser" :key="row.username" style="display: flex; justify-content: space-between; padding: 6px 0; border-bottom: 1px solid #f5f5f5">
              <span>
                <span style="display: inline-block; width: 20px; color: #999">{{ idx + 1 }}.</span>
                <strong>{{ row.username }}</strong>
              </span>
              <strong style="color: #18a058">{{ row.count }}</strong>
            </div>
          </div>
        </n-card>
      </n-gi>
    </n-grid>

    <!-- 过滤 -->
    <n-card size="small" style="margin-bottom: 12px">
      <n-space :wrap="false">
        <n-select v-model:value="filter.action" :options="actionOptions" placeholder="动作" clearable style="width: 160px" @update:value="loadList" />
        <n-select v-model:value="filter.target_type" :options="targetTypeOptions" placeholder="对象类型" clearable style="width: 140px" @update:value="loadList" />
        <n-select v-model:value="filter.status" :options="statusOptions" placeholder="状态" clearable style="width: 120px" @update:value="loadList" />
        <n-input v-model:value="filter.username" placeholder="用户名" clearable style="width: 140px" @keyup.enter="loadList" @clear="loadList" />
        <n-date-picker v-model:value="dateRange" type="daterange" clearable style="width: 260px" @update:value="onDateChange" />
        <n-button @click="loadList">查询</n-button>
        <n-button quaternary @click="resetFilter">重置</n-button>
      </n-space>
    </n-card>

    <!-- 表格 -->
    <n-data-table :columns="columns" :data="logs" :loading="loading" :bordered="false" :pagination="pagination" :row-key="(r: any) => r.id" />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, h, watch } from 'vue'
import { NCard, NDataTable, NButton, NSpace, NTag, NSelect, NInput, NDatePicker, NGrid, NGi, NProgress, useMessage, type DataTableColumns } from 'naive-ui'
import http from '@/api/client'

const props = defineProps<{ active: boolean }>()

const message = useMessage()
const logs = ref<any[]>([])
const loading = ref(false)
const dateRange = ref<[number, number] | null>(null)

const filter = reactive({
  action: null as string | null,
  target_type: null as string | null,
  status: null as string | null,
  username: '' as string,
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

const stats = ref<any>({})
const byAction = ref<any[]>([])
const byUser = ref<any[]>([])

const actionOptions = [
  { label: '登录', value: 'login' },
  { label: '登录失败', value: 'login_failed' },
  { label: '登出', value: 'logout' },
  { label: '认领 lead', value: 'lead_claim' },
  { label: '释放 lead', value: 'lead_release' },
  { label: '归档 lead', value: 'lead_archive' },
  { label: '人工绑定订单', value: 'order_bind' },
  { label: 'CSV/Excel 导入', value: 'order_import' },
  { label: 'Webhook 入库', value: 'order_webhook' },
  { label: 'API 拉取', value: 'partner_pull' },
  { label: '重置密码', value: 'password_reset' },
  { label: '创建', value: 'create' },
  { label: '更新', value: 'update' },
  { label: '删除', value: 'delete' }
]
const targetTypeOptions = [
  { label: '用户', value: 'user' },
  { label: '部门', value: 'department' },
  { label: '游戏', value: 'game' },
  { label: '字典', value: 'dict' },
  { label: '广播模板', value: 'broadcast_template' },
  { label: '线索', value: 'lead' },
  { label: '订单', value: 'order' },
  { label: '合作方', value: 'partner' },
  { label: '上传模板', value: 'upload_template' },
  { label: '系统', value: 'system' }
]
const statusOptions = [
  { label: '成功', value: 'success' },
  { label: '失败', value: 'failure' }
]

function actionLabel(s: string) {
  return actionOptions.find(o => o.value === s)?.label || s
}

const columns: DataTableColumns<any> = [
  { title: '时间', key: 'created_at', width: 160, render: (r) => new Date(r.created_at).toLocaleString('zh-CN', { hour12: false }) },
  { title: '操作人', key: 'username', width: 110, render: (r) => r.username || h(NTag, { size: 'small', type: 'warning' }, () => '匿名') },
  {
    title: '动作', key: 'action', width: 140,
    render: (r) => h(NTag, { size: 'small', type: actionTagType(r.action) }, () => actionLabel(r.action))
  },
  { title: '对象', key: 'target_label', width: 200, ellipsis: { tooltip: true } },
  { title: 'IP', key: 'ip', width: 120 },
  {
    title: '状态', key: 'status', width: 80,
    render: (r) => h(NTag, { size: 'small', type: r.status === 'success' ? 'success' : 'error' }, () => r.status === 'success' ? '✓' : '✗')
  },
  {
    title: '详情', key: 'details', width: 80,
    render: (r) => h(NButton, { size: 'small', text: true, type: 'primary', onClick: () => showDetails(r) }, () => '查看')
  }
]

function actionTagType(a: string): any {
  if (a === 'login_failed') return 'error'
  if (a.includes('import') || a.includes('webhook') || a.includes('pull')) return 'info'
  if (a.startsWith('lead_') || a === 'order_bind') return 'success'
  return 'default'
}

async function loadList() {
  loading.value = true
  try {
    const params: any = { page: pagination.page, page_size: pagination.pageSize }
    if (filter.action) params.action = filter.action
    if (filter.target_type) params.target_type = filter.target_type
    if (filter.status) params.status = filter.status
    if (filter.username) params.username = filter.username
    if (filter.start_date) params.start_date = filter.start_date
    if (filter.end_date) params.end_date = filter.end_date

    const resp: any = await http.get('/audit/logs', { params })
    logs.value = resp.data?.list || []
    pagination.itemCount = resp.data?.total || 0
  } catch {
    message.error('加载失败')
  } finally {
    loading.value = false
  }
}

async function loadStats() {
  try {
    const resp: any = await http.get('/audit/stats')
    byAction.value = resp.data?.by_action || []
    byUser.value = resp.data?.by_user || []

    const allLogs = byAction.value
    stats.value = {
      total: allLogs.reduce((s: number, r: any) => s + Number(r.count), 0),
      failed_login: allLogs.find((r: any) => r.action === 'login_failed')?.count || 0,
      business_actions: allLogs.filter((r: any) =>
        ['lead_claim', 'lead_release', 'lead_archive', 'order_bind'].includes(r.action)
      ).reduce((s: number, r: any) => s + Number(r.count), 0),
      write_actions: allLogs.filter((r: any) =>
        ['create', 'update', 'delete', 'order_import', 'order_webhook', 'partner_pull', 'password_reset'].includes(r.action)
      ).reduce((s: number, r: any) => s + Number(r.count), 0)
    }
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
  filter.action = null
  filter.target_type = null
  filter.status = null
  filter.username = ''
  filter.start_date = ''
  filter.end_date = ''
  dateRange.value = null
  pagination.page = 1
  loadList()
}

function showDetails(row: any) {
  const ext = row.extension || {}
  const before = row.before || {}
  const after = row.after || {}
  const lines: string[] = []
  if (row.error_message) lines.push(`错误: ${row.error_message}`)
  if (Object.keys(before).length > 0) lines.push(`变更前: ${JSON.stringify(before)}`)
  if (Object.keys(after).length > 0) lines.push(`变更后: ${JSON.stringify(after)}`)
  if (Object.keys(ext).length > 0) lines.push(`扩展: ${JSON.stringify(ext)}`)
  if (lines.length === 0) lines.push('无附加信息')
  message.info(lines.join('\n'), { duration: 6000 })
}

// active 用 immediate —— 深链直达本页时 props.active 初始就是 true,
// 不加 immediate 会因为「没发生变化」而不触发,导致数据永远是空
watch(() => props.active, (v) => { if (v) { loadList(); loadStats() } }, { immediate: true })
</script>

<style scoped>
.stat-label { color: #999; font-size: 12px; margin-bottom: 4px; }
.stat-value { font-size: 22px; font-weight: 700; color: #333; }
.text-muted { color: #999; text-align: center; padding: 20px 0; }
</style>