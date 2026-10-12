<template>
  <div>
    <!-- 筛选条 -->
    <div class="card-section" style="padding: 12px; margin-bottom: 12px">
      <n-space wrap>
        <n-input v-model:value="filterPlayerId" placeholder="玩家 ID(模糊)" clearable style="width: 200px" @keyup.enter="loadList" @clear="loadList" />
        <n-input v-model:value="filterCountry" placeholder="国家(模糊)" clearable style="width: 160px" @keyup.enter="loadList" @clear="loadList" />
        <n-select v-model:value="filterGameId" placeholder="游戏" clearable style="width: 200px" :options="gameOptions" @update:value="loadList" />
        <n-select v-model:value="filterIsTarget" placeholder="是否目标" clearable style="width: 120px" :options="targetOptions" @update:value="loadList" />
        <n-select v-model:value="filterHasOrders" placeholder="是否有订单" clearable style="width: 140px" :options="hasOrdersOptions" @update:value="loadList" />
        <n-button @click="loadList" :loading="loadingList">查询</n-button>
      </n-space>
    </div>

    <!-- 列表：列由默认视图配置 -->
    <n-data-table
      :columns="dynamicColumns"
      :data="players"
      :pagination="pagination"
      :bordered="false"
      :single-line="false"
      :loading="loadingList"
      :row-key="(row: any) => `${row.player_id}-${row.game_id}`"
      @update:page="(p: number) => { pagination.page = p; loadList() }"
      @update:page-size="(s: number) => { pagination.pageSize = s; pagination.page = 1; loadList() }"
      @row-click="openDrawer"
    />

    <!-- 详情抽屉 -->
    <PlayerDrawer
      v-model:show="drawerShow"
      :player-id="drawerPlayerId"
      :game-id="drawerGameId"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h, computed } from 'vue'
import { NDataTable, NButton, NSpace, NInput, NSelect, NTag, type DataTableColumns } from 'naive-ui'
import { playerApi, gameApi, listViewApi, type PlayerSummary, type ListViewField } from '@/api/modules'
import PlayerDrawer from './PlayerDrawer.vue'

const players = ref<PlayerSummary[]>([])
const loadingList = ref(false)

const filterPlayerId = ref('')
const filterCountry = ref('')
const filterGameId = ref<number | null>(null)
const filterIsTarget = ref<string | null>(null)
const filterHasOrders = ref<string | null>(null)

const gameOptions = ref<{ label: string; value: number }[]>([])

const targetOptions = [
  { label: '目标', value: 'target' },
  { label: '非目标', value: 'not_target' }
]
const hasOrdersOptions = [
  { label: '有订单', value: 'true' },
  { label: '无订单', value: 'false' }
]

const pagination = ref({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: true, pageSizes: [10, 20, 50, 100] })

const drawerShow = ref(false)
const drawerPlayerId = ref('')
const drawerGameId = ref(0)

function openDrawer(row: PlayerSummary) {
  drawerPlayerId.value = row.player_id
  drawerGameId.value = row.game_id
  drawerShow.value = true
}

// ===== 视图配置加载 + 列动态构建 =====
const viewFields = ref<ListViewField[]>([])

async function loadDefaultView() {
  try {
    const resp: any = await listViewApi.getDefault('player')
    viewFields.value = (resp.data?.fields || []) as ListViewField[]
  } catch (e) {
    // 失败时给个空数组 → 列表为空,提示用户去配置中心设置
    viewFields.value = []
  }
}

// 列动态构建
const dynamicColumns = computed<DataTableColumns<PlayerSummary>>(() => {
  return viewFields.value.map((f) => ({
    title: f.label,
    key: f.code,
    width: f.width || undefined,
    align: (f.align as any) || 'left',
    render: renderCellFor(f)
  }))
})

function renderCellFor(f: ListViewField) {
  const code = f.code
  const fmt = f.formatter || 'default'

  // extension.<key>
  if (code.startsWith('extension.')) {
    const key = code.substring('extension.'.length)
    return (row: PlayerSummary) => {
      const val = (row as any).extension?.[key]
      return val === undefined || val === null || val === '' ? '—' : String(val)
    }
  }

  return (row: PlayerSummary) => {
    const val = getRaw(row, code)
    if (val === undefined || val === null || val === '') return '—'
    return formatByFormatter(fmt, val, code, row)
  }
}

function getRaw(row: any, code: string): any {
  if (code in row) return row[code]
  // nested path 支持（虽然 player 不太需要）
  const parts = code.split('.')
  let cur: any = row
  for (const p of parts) {
    if (cur == null) return ''
    cur = cur[p]
  }
  return cur
}

function formatByFormatter(fmt: string, val: any, code: string, row: PlayerSummary): any {
  switch (fmt) {
    case 'datetime':
      return new Date(val).toLocaleString('zh-CN', { hour12: false })
    case 'date':
      return new Date(val).toLocaleDateString('zh-CN')
    case 'number':
      return typeof val === 'number' ? val.toLocaleString('zh-CN') : String(val)
    case 'money':
      return `¥${Number(val || 0).toFixed(2)}`
    case 'target_tag':
      return h(NTag, { size: 'small', type: val === 'target' ? 'success' : 'default' }, () => val === 'target' ? '目标' : '非目标')
    case 'paid_tag':
      return h(NTag, { size: 'small', type: val ? 'success' : 'default' }, () => val ? '已付' : '未付')
    case 'lead_status_tag':
      return h(NTag, { size: 'small' }, () => ({ pending: '待认领', claimed: '已认领', archived: '已流失' }[val as string] || String(val)))
    case 'user_list': {
      // creator_names / claimer_names - 当前 PlayerSummary 只有 ids,显示人数
      const ids = (row as any)[code] as number[] | undefined
      if (!ids || ids.length === 0) return '—'
      return `${ids.length} 人`
    }
    default:
      return String(val)
  }
}

// ===== 原有数据加载 =====
async function loadGames() {
  try {
    const r: any = await gameApi.list()
    const list = (r.data?.list || r.data || []) as any[]
    gameOptions.value = list.map((g: any) => ({ label: `${g.code} - ${g.name}`, value: g.id }))
  } catch (e) {}
}

async function loadList() {
  loadingList.value = true
  try {
    const params: any = {
      page: pagination.value.page,
      page_size: pagination.value.pageSize
    }
    if (filterPlayerId.value.trim()) params.player_id = filterPlayerId.value.trim()
    if (filterCountry.value.trim()) params.country = filterCountry.value.trim()
    if (filterGameId.value !== null) params.game_id = filterGameId.value
    if (filterIsTarget.value) params.is_target = filterIsTarget.value
    if (filterHasOrders.value) params.has_orders = filterHasOrders.value

    const r: any = await playerApi.list(params)
    players.value = r.data?.list || []
    pagination.value.itemCount = r.data?.total || 0
  } finally {
    loadingList.value = false
  }
}

onMounted(async () => {
  await loadDefaultView()
  loadGames()
  loadList()
})
</script>

<style scoped>
:deep(.num) {
  font-family: 'SF Mono', Consolas, monospace;
  font-size: 13px;
}
:deep(.game-tag) {
  background: #eff6ff;
  color: #1d4ed8;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}
</style>