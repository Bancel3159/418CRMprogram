<template>
  <div class="pool-panel">
    <div class="filter-bar">
      <n-space wrap>
        <n-input v-model:value="filterKeyword" placeholder="玩家 ID" clearable style="width: 200px" @keyup.enter="loadList" @clear="loadList" />
        <n-button @click="loadList" :loading="loading">刷新</n-button>
      </n-space>
    </div>

    <n-data-table
      :columns="columns"
      :data="leads"
      :pagination="pagination"
      :loading="loading"
      :bordered="false"
      @update:page="(p: number) => { pagination.page = p; loadList() }"
      @update:page-size="(s: number) => { pagination.pageSize = s; pagination.page = 1; loadList() }"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useRouter } from 'vue-router'
import { NDataTable, NButton, NSpace, NInput, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { leadApi, type Lead } from '@/api/modules'

const router = useRouter()
const message = useMessage()

const leads = ref<Lead[]>([])
const total = ref(0)
const loading = ref(false)
const filterKeyword = ref('')

const pagination = ref({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: true, pageSizes: [10, 20, 50] })

const columns: DataTableColumns<Lead> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '游戏', key: 'game.code', width: 80, render: (row: Lead) => row.game?.code || '-' },
  { title: '玩家 ID', key: 'player_id', width: 140 },
  { title: '玩家名', key: 'player_game_name', width: 120 },
  { title: '渠道', key: 'channel', width: 100 },
  { title: '国家', key: 'player_country', width: 80 },
  { title: '区服', key: 'region', width: 80 },
  { title: '年龄', key: 'player_age', width: 60 },
  {
    title: '是否目标',
    key: 'is_target',
    width: 90,
    render: (row: Lead) => h(NTag, { type: row.is_target === 'target' ? 'success' : 'default', size: 'small' }, () => row.is_target === 'target' ? '目标' : '非目标')
  },
  { title: '提交时间', key: 'created_at', width: 170 },
  { title: '拉新员工', key: 'creator.real_name', width: 100, render: (row: Lead) => row.creator?.real_name || row.creator?.username },
  {
    title: '操作',
    key: 'actions',
    width: 120,
    fixed: 'right',
    render: (row: Lead) => h('div', { style: 'display: flex; gap: 8px' }, [
      h(NButton, { size: 'small', type: 'primary', onClick: () => handleClaim(row) }, () => '抢单'),
      h(NButton, { size: 'small', text: true, onClick: () => router.push(`/leads/${row.id}`) }, () => '详情')
    ])
  }
]

async function loadList() {
  loading.value = true
  try {
    const resp: any = await leadApi.pool({
      page: pagination.value.page,
      page_size: pagination.value.pageSize
    })
    leads.value = resp.data.list || []
    total.value = resp.data.total || 0
    pagination.value.itemCount = total.value
  } catch {} finally {
    loading.value = false
  }
}

async function handleClaim(row: Lead) {
  try {
    const resp: any = await leadApi.claim(row.id)
    if (resp.data.token_to_copy) {
      try { await navigator.clipboard.writeText(resp.data.token_to_copy) } catch {}
    }
    message?.success?.('认领成功，token 已复制')
    loadList()
  } catch {}
}

onMounted(loadList)
</script>

<style scoped>
.pool-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.filter-bar {
  padding: 4px 0 12px 0;
}
</style>