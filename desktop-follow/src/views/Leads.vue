<template>
  <div>
    <div class="page-header">
      <div class="page-title">我的玩家</div>
      <n-space>
        <n-select v-model:value="filterStatus" placeholder="状态" clearable style="width: 140px" :options="statusOptions" @update:value="loadList" />
        <n-button @click="loadList" :loading="loading">刷新</n-button>
      </n-space>
    </div>

    <n-data-table
      :columns="columns"
      :data="leads"
      :loading="loading"
      :pagination="pagination"
      :bordered="false"
      @update:page="(p: number) => { pagination.page = p; loadList() }"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useRouter } from 'vue-router'
import { NDataTable, NButton, NSpace, NSelect, NTag, type DataTableColumns } from 'naive-ui'
import { leadApi } from '@/api/modules'

const router = useRouter()
const leads = ref<any[]>([])
const loading = ref(false)
const filterStatus = ref<string | null>(null)
const pagination = ref({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: true, pageSizes: [10, 20, 50] })

const statusOptions = [
  { label: '已认领', value: 'claimed' },
  { label: '已流失', value: 'archived' }
]

const statusType = (s: string) => ({ pending: 'info', claimed: 'success', archived: 'default' }[s] || 'default')
const statusLabel = (s: string) => ({ pending: '待认领', claimed: '已认领', archived: '已流失' }[s] || s)

const columns: DataTableColumns<any> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '游戏', key: 'game.code', width: 100, render: (row) => row.game?.code || '-' },
  { title: '玩家 ID', key: 'player_id', width: 140 },
  { title: '玩家名', key: 'player_game_name' },
  { title: '区服', key: 'region', width: 80 },
  {
    title: '状态', key: 'status', width: 100,
    render: (row) => h(NTag, { type: statusType(row.status), size: 'small' }, () => statusLabel(row.status))
  },
  { title: '认领时间', key: 'claimed_at', width: 170, render: (row) => row.claimed_at ? new Date(row.claimed_at).toLocaleString('zh-CN') : '-' },
  {
    title: '操作', key: 'actions', width: 100,
    render: (row) => h(NButton, { size: 'small', type: 'primary', text: true, onClick: () => router.push(`/leads/${row.id}`) }, () => '详情')
  }
]

async function loadList() {
  loading.value = true
  try {
    const params: any = { page: pagination.value.page, page_size: pagination.value.pageSize }
    if (filterStatus.value) params.status = filterStatus.value
    const resp: any = await leadApi.list(params)
    leads.value = resp.data.list || []
    pagination.value.itemCount = resp.data.total || 0
  } catch {} finally {
    loading.value = false
  }
}

onMounted(loadList)
</script>