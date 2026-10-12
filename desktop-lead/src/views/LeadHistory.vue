<template>
  <div>
    <div class="page-header">
      <div class="page-title">我提交的线索</div>
      <n-button @click="loadList">刷新</n-button>
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
import { NDataTable, NButton, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import axios from '@/api/modules'

const message = useMessage()
const leads = ref<any[]>([])
const loading = ref(false)
const pagination = ref({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: true, pageSizes: [10, 20, 50] })

const statusType = (s: string) => ({ pending: 'info', claimed: 'success', archived: 'default' }[s] || 'default')
const statusLabel = (s: string) => ({ pending: '待认领', claimed: '已认领', archived: '已流失' }[s] || s)

const columns: DataTableColumns<any> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '游戏', key: 'game.code', width: 100, render: (row) => row.game?.code || '-' },
  { title: '玩家 ID', key: 'player_id', width: 140 },
  { title: '渠道', key: 'channel', width: 100 },
  { title: '国家', key: 'player_country', width: 80 },
  {
    title: '状态', key: 'status', width: 100,
    render: (row) => h(NTag, { type: statusType(row.status), size: 'small' }, () => statusLabel(row.status))
  },
  { title: '提交时间', key: 'created_at', width: 170 }
]

async function loadList() {
  loading.value = true
  try {
    const resp: any = await axios.get('/leads', { params: { page: pagination.value.page, page_size: pagination.value.pageSize } })
    leads.value = resp.data.list || []
    pagination.value.itemCount = resp.data.total || 0
  } catch (err: any) {
    message.error(err?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadList)
</script>