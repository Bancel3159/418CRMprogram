<template>
  <div>
    <div class="page-header">
      <n-button type="primary" @click="openCreate">新增字典</n-button>
    </div>
    <n-card size="small" style="margin-bottom: 12px">
      <p style="margin: 0 0 4px 0"><strong>说明</strong></p>
      <p style="margin: 0; color: #666; font-size: 12px; line-height: 1.6">
        • 这里维护的是<strong>来源渠道</strong>下拉选项(前端唯一来源)<br>
        • 增删选项会影响<strong>前端</strong>渠道下拉;若是<strong>自定义字段</strong>类型为 select,可选"引用字典"复用本表
      </p>
    </n-card>
    <div class="card-section" style="padding: 12px 16px; margin-bottom: 16px">
      <n-space>
        <n-select v-model:value="filterCategory" placeholder="按分类过滤" clearable style="width: 200px" :options="categoryOptions" @update:value="loadList" />
      </n-space>
    </div>
    <n-data-table :columns="columns" :data="dicts" :loading="loading" :bordered="false" />

    <n-modal v-model:show="showForm" preset="card" :title="editing.id ? '编辑字典' : '新增字典'" style="width: 500px">
      <n-form :model="editing" label-placement="left" label-width="100">
        <n-form-item label="分类"><n-select v-model:value="editing.category" :options="categoryOptions" /></n-form-item>
        <n-form-item label="代码"><n-input v-model:value="editing.code" :disabled="!!editing.id" /></n-form-item>
        <n-form-item label="显示名"><n-input v-model:value="editing.label" /></n-form-item>
        <n-form-item label="排序"><n-input-number v-model:value="editing.sort" :min="0" /></n-form-item>
        <n-form-item label="状态">
          <n-select v-model:value="editing.status" :options="[{label:'启用',value:'enabled'},{label:'禁用',value:'disabled'}]" />
        </n-form-item>
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
import { ref, onMounted, h } from 'vue'
import { NDataTable, NButton, NTag, NModal, NForm, NFormItem, NInput, NSelect, NSpace, NInputNumber, useMessage, type DataTableColumns } from 'naive-ui'
import { dictApi, type Dict } from '@/api/modules'

const message = useMessage()
const dicts = ref<Dict[]>([])
const loading = ref(false)
const showForm = ref(false)
const submitting = ref(false)
const filterCategory = ref<string | null>(null)
const editing = ref<any>({ id: null, category: 'channel', code: '', label: '', sort: 0, status: 'enabled' })

const categoryOptions = [
  { label: '来源渠道 (channel)', value: 'channel' }
]

const columns: DataTableColumns<Dict> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '分类', key: 'category', width: 120, render: (row) => h(NTag, { size: 'small' }, () => row.category) },
  { title: '代码', key: 'code', width: 150 },
  { title: '显示名', key: 'label' },
  { title: '排序', key: 'sort', width: 80 },
  {
    title: '状态', key: 'status', width: 90,
    render: (row) => h(NTag, { type: row.status === 'enabled' ? 'success' : 'default', size: 'small' }, () => row.status === 'enabled' ? '启用' : '禁用')
  },
  {
    title: '操作', key: 'actions', width: 160,
    render: (row) => h(NSpace, {}, () => [
      h(NButton, { size: 'small', text: true, type: 'primary', onClick: () => openEdit(row) }, () => '编辑'),
      h(NButton, { size: 'small', text: true, type: 'error', onClick: () => handleDelete(row) }, () => '删除')
    ])
  }
]

async function loadList() {
  loading.value = true
  try {
    const params: any = {}
    if (filterCategory.value) params.category = filterCategory.value
    const resp: any = await dictApi.list(params)
    dicts.value = resp.data || []
  } catch {} finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = { id: null, category: 'channel', code: '', label: '', sort: 0, status: 'enabled' }
  showForm.value = true
}

function openEdit(row: Dict) {
  editing.value = { ...row }
  showForm.value = true
}

async function handleSubmit() {
  submitting.value = true
  try {
    if (editing.value.id) {
      await dictApi.update(editing.value.id, {
        label: editing.value.label, sort: editing.value.sort, status: editing.value.status
      })
      message.success('更新成功')
    } else {
      await dictApi.create({
        category: editing.value.category, code: editing.value.code,
        label: editing.value.label, sort: editing.value.sort, status: editing.value.status
      })
      message.success('创建成功')
    }
    showForm.value = false
    loadList()
  } catch {} finally {
    submitting.value = false
  }
}

async function handleDelete(row: Dict) {
  if (!confirm(`确认删除「${row.label}」？`)) return
  try {
    await dictApi.delete(row.id)
    message.success('已删除')
    loadList()
  } catch {}
}

onMounted(loadList)
</script>