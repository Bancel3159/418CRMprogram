<template>
  <div>
    <div class="page-header">
      <span style="font-size: 18px; font-weight: 700">查询</span>
    </div>

    <n-card size="small" :bordered="false" style="margin-bottom: 16px">
      <n-space :wrap="false" align="center">
        <n-input
          v-model:value="keyword"
          placeholder="玩家 ID / 玩家用户名 / 玩家游戏名"
          clearable
          style="width: 320px"
          @keyup.enter="doSearch"
        />
        <n-button type="primary" :loading="loading" @click="doSearch">查询</n-button>
        <n-tag :bordered="false" type="info" size="small">
          范围:自己可见的线索(基于 RBAC)
        </n-tag>
      </n-space>
    </n-card>

    <n-data-table
      :columns="columns"
      :data="rows"
      :loading="loading"
      :bordered="false"
      :pagination="pagination"
      @update:page="(p: number) => { pagination.page = p; doSearch() }"
    />

    <n-empty v-if="!loading && rows.length === 0 && searched" description="未找到匹配的线索" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NInput, NButton, NSpace, NTag, NEmpty, NDataTable, useMessage, type DataTableColumns } from 'naive-ui'
import { leadApi } from '@/api/modules'

const router = useRouter()
const message = useMessage()

const keyword = ref('')
const rows = ref<any[]>([])
const loading = ref(false)
const searched = ref(false)

const pagination = ref({ page: 1, pageSize: 20, itemCount: 0 })

const statusType = (s: string) => ({ pending: 'info', claimed: 'success', archived: 'default' }[s] || 'default')
const statusLabel = (s: string) => ({ pending: '待认领', claimed: '已认领', archived: '已流失' }[s] || s)

const columns: DataTableColumns<any> = [
  { title: 'ID', key: 'id', width: 60 },
  {
    title: '游戏', key: 'game', width: 110,
    render: (row) => row.game ? `${row.game.code} - ${row.game.name}` : '-'
  },
  { title: '玩家 ID', key: 'player_id', width: 140 },
  { title: '玩家用户名', key: 'player_username', width: 140 },
  { title: '玩家名', key: 'player_game_name', width: 120 },
  { title: '区服', key: 'region', width: 80 },
  {
    title: '状态', key: 'status', width: 100,
    render: (row) => h(NTag, { type: statusType(row.status), size: 'small' }, () => statusLabel(row.status))
  },
  {
    title: '认领人', key: 'claimed_by_user', width: 110,
    render: (row) => row.claimed_by_user?.real_name || row.claimed_by_user?.username || '—'
  },
  {
    title: '操作', key: 'actions', width: 80,
    render: (row) => h(NButton, {
      size: 'small', text: true, type: 'primary',
      onClick: () => router.push(`/leads/${row.id}`)
    }, () => '详情')
  }
]

async function doSearch() {
  loading.value = true
  searched.value = true
  try {
    const params: any = {
      page: pagination.value.page,
      page_size: pagination.value.pageSize
    }
    if (keyword.value.trim()) params.keyword = keyword.value.trim()
    const resp: any = await leadApi.list(params)
    rows.value = resp.data.list || []
    pagination.value.itemCount = resp.data.total || 0
  } catch (err: any) {
    message.error(err?.message || '查询失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  // 首次进入空查询,展示自己可见的线索(默认 RBAC 范围)
  doSearch()
})
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
}
</style>
