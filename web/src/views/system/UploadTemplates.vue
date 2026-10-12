<template>
  <div>
    <div class="page-header">
      <n-space>
        <n-tag type="info" size="small">按游戏隔离</n-tag>
        <n-button type="primary" @click="openCreate">新增模板</n-button>
      </n-space>
    </div>

    <n-card size="small" style="margin-bottom: 12px">
      <n-space>
        <n-select v-model:value="filterGameId" :options="gameOptions" placeholder="按游戏筛选" clearable style="width: 200px" @update:value="loadList" />
        <n-button quaternary @click="resetFilter">重置</n-button>
      </n-space>
    </n-card>

    <n-data-table :columns="columns" :data="templates" :loading="loading" :bordered="false" />

    <n-modal v-model:show="showForm" preset="card" :title="editing.id ? '编辑模板' : '新增模板'" style="width: 800px">
      <n-form :model="editing" label-placement="left" label-width="120">
        <n-grid :cols="2" :x-gap="16">
          <n-gi>
            <n-form-item label="模板名" required><n-input v-model:value="editing.name" /></n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="所属游戏" required>
              <n-select v-model:value="editing.game_id" :options="gameOptions" placeholder="选择游戏" :disabled="!!editing.id" />
            </n-form-item>
          </n-gi>
        </n-grid>

        <n-divider>字段映射(系统字段 → Excel 列名)</n-divider>
        <n-alert type="info" :show-icon="false" style="margin-bottom: 12px">
          必填字段: order_no / player_id / amount / order_time。可选: currency / status / paid_at / remark。
        </n-alert>
        <n-grid :cols="2" :x-gap="16" :y-gap="8">
          <n-gi v-for="sysField in systemFields" :key="sysField.key">
            <n-form-item :label="`${sysField.label}${sysField.required ? ' *' : ''}`">
              <n-input v-model:value="editing.column_mappings[sysField.key]" :placeholder="sysField.placeholder" />
            </n-form-item>
          </n-gi>
        </n-grid>

        <n-divider>状态映射(原始值 → 系统状态)</n-divider>
        <n-grid :cols="2" :x-gap="16" :y-gap="8">
          <n-gi>
            <n-form-item label="已支付 → paid">
              <n-input v-model:value="editing.status_mapping['已支付']" placeholder="如 已支付" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="已退款 → refunded">
              <n-input v-model:value="editing.status_mapping['已退款']" placeholder="如 已退款" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="待支付 → pending">
              <n-input v-model:value="editing.status_mapping['待支付']" placeholder="如 待支付" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="已取消 → cancelled">
              <n-input v-model:value="editing.status_mapping['已取消']" placeholder="如 已取消" />
            </n-form-item>
          </n-gi>
        </n-grid>

        <n-grid :cols="2" :x-gap="16">
          <n-gi>
            <n-form-item label="时间格式">
              <n-input v-model:value="editing.time_format" placeholder="2006-01-02 15:04:05" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="默认币种">
              <n-input v-model:value="editing.currency" placeholder="CNY" />
            </n-form-item>
          </n-gi>
        </n-grid>

        <n-form-item label="备注"><n-input v-model:value="editing.remark" type="textarea" :rows="2" /></n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showForm = false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="handleSubmit">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, h, onMounted } from 'vue'
import { NCard, NDataTable, NButton, NSpace, NTag, NModal, NForm, NFormItem, NInput, NSelect, NGrid, NGi, NDivider, NAlert, useMessage, type DataTableColumns } from 'naive-ui'
import { uploadTemplateApi, gameApi, type UploadTemplate, type Game } from '@/api/modules'

const message = useMessage()
const templates = ref<UploadTemplate[]>([])
const loading = ref(false)
const showForm = ref(false)
const submitting = ref(false)
const filterGameId = ref<number | null>(null)
const gameOptions = ref<{ label: string; value: number }[]>([])

const systemFields = [
  { key: 'order_no', label: '订单号', required: true, placeholder: '如 订单号' },
  { key: 'player_id', label: '玩家ID', required: true, placeholder: '如 玩家ID' },
  { key: 'amount', label: '金额', required: true, placeholder: '如 充值金额' },
  { key: 'order_time', label: '下单时间', required: true, placeholder: '如 下单时间' },
  { key: 'currency', label: '币种', required: false, placeholder: '如 币种(可选)' },
  { key: 'status', label: '状态', required: false, placeholder: '如 支付状态(可选)' },
  { key: 'paid_at', label: '支付时间', required: false, placeholder: '如 支付时间(可选)' },
  { key: 'remark', label: '备注', required: false, placeholder: '如 备注(可选)' }
]

const editing = ref<any>(newTemplate())

function newTemplate() {
  return {
    id: null,
    name: '',
    game_id: null,
    column_mappings: {} as Record<string, string>,
    time_format: '2006-01-02 15:04:05',
    status_mapping: {} as Record<string, string>,
    currency: 'CNY',
    remark: ''
  }
}

const columns: DataTableColumns<UploadTemplate> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '名称', key: 'name' },
  { title: '游戏', key: 'game', width: 120, render: (r) => r.game ? `${r.game.code} - ${r.game.name}` : '-' },
  { title: '币种', key: 'currency', width: 80 },
  {
    title: '字段映射', key: 'column_mappings', width: 240, ellipsis: { tooltip: true },
    render: (r) => {
      const keys = Object.keys(r.column_mappings || {})
      const mapped = keys.filter(k => r.column_mappings[k]).length
      return h(NTag, { type: mapped >= 4 ? 'success' : 'warning', size: 'small' }, () => `${mapped}/4 必填`)
    }
  },
  { title: '备注', key: 'remark', ellipsis: { tooltip: true } },
  {
    title: '操作', key: 'actions', width: 160,
    render: (r) => h(NSpace, {}, () => [
      h(NButton, { size: 'small', text: true, type: 'primary', onClick: () => openEdit(r) }, () => '编辑'),
      h(NButton, { size: 'small', text: true, type: 'error', onClick: () => handleDelete(r) }, () => '删除')
    ])
  }
]

async function loadList() {
  loading.value = true
  try {
    const params: any = filterGameId.value ? { game_id: filterGameId.value } : {}
    const resp: any = await uploadTemplateApi.list(params)
    templates.value = resp.data || []
  } catch {} finally {
    loading.value = false
  }
}

async function loadGames() {
  const resp: any = await gameApi.list()
  gameOptions.value = (resp.data || []).map((g: Game) => ({ label: `${g.code} - ${g.name}`, value: g.id }))
}

function resetFilter() {
  filterGameId.value = null
  loadList()
}

function openCreate() {
  editing.value = newTemplate()
  showForm.value = true
}

function openEdit(row: UploadTemplate) {
  editing.value = {
    id: row.id,
    name: row.name,
    game_id: row.game_id,
    column_mappings: { ...row.column_mappings },
    time_format: row.time_format || '2006-01-02 15:04:05',
    status_mapping: { ...(row.status_mapping || {}) },
    currency: row.currency || 'CNY',
    remark: row.remark || ''
  }
  showForm.value = true
}

async function handleSubmit() {
  if (!editing.value.name || !editing.value.game_id) {
    message.warning('请填写名称并选择游戏')
    return
  }
  const required = ['order_no', 'player_id', 'amount', 'order_time']
  for (const f of required) {
    if (!editing.value.column_mappings[f]) {
      message.warning(`字段映射缺少必填项: ${f}`)
      return
    }
  }
  submitting.value = true
  try {
    if (editing.value.id) {
      await uploadTemplateApi.update(editing.value.id, editing.value)
      message.success('更新成功')
    } else {
      await uploadTemplateApi.create(editing.value)
      message.success('创建成功')
    }
    showForm.value = false
    loadList()
  } catch (e: any) {
    message.error(e?.message || '保存失败')
  } finally {
    submitting.value = false
  }
}

async function handleDelete(row: UploadTemplate) {
  if (!confirm(`确认删除「${row.name}」?`)) return
  try {
    await uploadTemplateApi.delete(row.id)
    message.success('已删除')
    loadList()
  } catch {}
}

onMounted(() => {
  loadGames()
  loadList()
})
</script>

<style scoped>
.page-header { display: flex; justify-content: flex-end; align-items: center; margin-bottom: 16px; }
</style>