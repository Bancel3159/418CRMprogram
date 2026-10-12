<template>
  <div class="list-panel">
    <div class="filter-bar">
      <n-space wrap>
        <n-select v-model:value="filterStatus" placeholder="状态" clearable style="width: 140px" :options="statusOptions" @update:value="loadList" />
        <n-select v-model:value="filterGameId" placeholder="游戏" clearable filterable style="width: 200px" :options="gameOptions" @update:value="loadList" />
        <n-input v-model:value="filterKeyword" placeholder="玩家 ID / 用户名" clearable style="width: 200px" @keyup.enter="loadList" @clear="loadList" />
        <n-button @click="loadList" :loading="loading">查询</n-button>
        <n-button quaternary @click="resetFilter">重置</n-button>
      </n-space>
    </div>

    <n-data-table
      :columns="columns"
      :data="leads"
      :pagination="pagination"
      :bordered="false"
      :single-line="false"
      :loading="loading"
      @update:page="handlePageChange"
      @update:page-size="handlePageSizeChange"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useRouter } from 'vue-router'
import { NDataTable, NButton, NSpace, NSelect, NInput, NTag, type DataTableColumns, type PaginationProps } from 'naive-ui'
import { leadApi, gameApi, type Lead } from '@/api/modules'

const router = useRouter()

const leads = ref<Lead[]>([])
const loading = ref(false)
const filterStatus = ref<string | null>(null)
const filterKeyword = ref('')
const filterGameId = ref<number | null>(null)
const gameOptions = ref<{ label: string; value: number }[]>([])

const statusOptions = [
  { label: '待认领', value: 'pending' },
  { label: '已认领', value: 'claimed' },
  { label: '已流失', value: 'archived' }
]

const pagination = ref<PaginationProps>({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
  onChange: (page: number) => {
    pagination.value!.page = page
    loadList()
  },
  onUpdatePageSize: (pageSize: number) => {
    pagination.value!.pageSize = pageSize
    pagination.value!.page = 1
    loadList()
  }
})

const statusType = (s: string) => {
  if (s === 'pending') return 'info'
  if (s === 'claimed') return 'success'
  return 'default'
}
const statusLabel = (s: string) => ({ pending: '待认领', claimed: '已认领', archived: '已流失' }[s] || s)

const columns: DataTableColumns<Lead> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '游戏', key: 'game.code', width: 80, render: (row) => row.game?.code || '-' },
  { title: '玩家 ID', key: 'player_id', width: 140 },
  { title: '玩家名', key: 'player_game_name', width: 120 },
  { title: '渠道', key: 'channel', width: 100 },
  { title: '国家', key: 'player_country', width: 80 },
  { title: '区服', key: 'region', width: 80 },
  {
    title: '是否目标',
    key: 'is_target',
    width: 90,
    render: (row) => h(NTag, { type: row.is_target === 'target' ? 'success' : 'default', size: 'small' }, () => row.is_target === 'target' ? '目标' : '非目标')
  },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: (row) => h(NTag, { type: statusType(row.status), size: 'small' }, () => statusLabel(row.status))
  },
  { title: '提交时间', key: 'created_at', width: 170 },
  {
    title: '操作',
    key: 'actions',
    width: 80,
    fixed: 'right',
    render: (row) => h(NButton, { size: 'small', text: true, type: 'primary', onClick: () => router.push(`/leads/${row.id}`) }, () => '详情')
  }
]

async function loadGames() {
  try {
    const r: any = await gameApi.list()
    const list = (r.data?.list || r.data || []) as any[]
    gameOptions.value = list.map((g: any) => ({ label: `${g.code} - ${g.name}`, value: g.id }))
  } catch {}
}

async function loadList() {
  loading.value = true
  try {
    const params: any = {
      page: pagination.value.page,
      page_size: pagination.value.pageSize
    }
    if (filterStatus.value) params.status = filterStatus.value
    if (filterGameId.value !== null) params.game_id = filterGameId.value
    if (filterKeyword.value.trim()) {
      params.keyword = filterKeyword.value.trim()
    }
    const resp: any = await leadApi.list(params)
    leads.value = resp.data.list || []
    pagination.value.itemCount = resp.data.total || 0
  } catch (err) {
    // 错误由拦截器处理
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) {
  pagination.value!.page = page
  loadList()
}

function handlePageSizeChange(pageSize: number) {
  pagination.value!.pageSize = pageSize
  pagination.value!.page = 1
  loadList()
}

function resetFilter() {
  filterStatus.value = null
  filterGameId.value = null
  filterKeyword.value = ''
  pagination.value!.page = 1
  loadList()
}

onMounted(() => {
  loadGames()
  loadList()
})
</script>

<style scoped>
.list-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.filter-bar {
  padding: 4px 0 12px 0;
}
</style>