<template>
  <div>
    <div class="page-header">
      <n-button type="primary" @click="openCreate(null)">新增部门</n-button>
    </div>
    <n-data-table :columns="columns" :data="departments" :loading="loading" :bordered="false" :children-key="'children'" />

    <n-modal v-model:show="showForm" preset="card" :title="editing.id ? '编辑部门' : '新增部门'" style="width: 500px">
      <n-form :model="editing" label-placement="left" label-width="100">
        <n-form-item label="名称"><n-input v-model:value="editing.name" /></n-form-item>
        <n-form-item label="类型">
          <n-select v-model:value="editing.type" :options="[
            { label: '公司', value: 'company' },
            { label: '前端', value: 'dept_a' },
            { label: '后端', value: 'dept_b' },
            { label: '组', value: 'group' }
          ]" />
        </n-form-item>
        <n-form-item label="上级">
          <n-select v-model:value="editing.parent_id" :options="parentOptions" clearable />
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
import { ref, onMounted, computed, h } from 'vue'
import { NDataTable, NButton, NTag, NModal, NForm, NFormItem, NInput, NSelect, NSpace, useMessage, type DataTableColumns } from 'naive-ui'
import { deptApi, type Department } from '@/api/modules'

const message = useMessage()
const departments = ref<Department[]>([])
const loading = ref(false)
const showForm = ref(false)
const submitting = ref(false)
const editing = ref<any>({ id: null, name: '', type: 'group', parent_id: null })

const parentOptions = computed(() => {
  const opts = departments.value
    .filter(d => d.id !== editing.value.id && d.type !== 'group')
    .map(d => ({ label: d.name, value: d.id }))
  return [{ label: '（顶级）', value: null }, ...opts]
})

const typeLabel = (t: string) => ({ company: '公司', dept_a: '前端', dept_b: '后端', group: '组' }[t] || t)
const typeColor = (t: string) => ({ company: 'success', dept_a: 'info', dept_b: 'warning', group: 'default' }[t] || 'default')

const columns: DataTableColumns<Department> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '名称', key: 'name' },
  {
    title: '类型', key: 'type', width: 100,
    render: (row) => h(NTag, { type: typeColor(row.type), size: 'small' }, () => typeLabel(row.type))
  },
  { title: '上级 ID', key: 'parent_id', width: 80 },
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
    const resp: any = await deptApi.list()
    departments.value = resp.data || []
  } catch {} finally {
    loading.value = false
  }
}

function openCreate(parentId: number | null) {
  editing.value = { id: null, name: '', type: 'group', parent_id: parentId }
  showForm.value = true
}

function openEdit(row: Department) {
  editing.value = { ...row }
  showForm.value = true
}

async function handleSubmit() {
  submitting.value = true
  try {
    if (editing.value.id) {
      await deptApi.update(editing.value.id, {
        name: editing.value.name, type: editing.value.type, parent_id: editing.value.parent_id
      })
      message.success('更新成功')
    } else {
      await deptApi.create({
        name: editing.value.name, type: editing.value.type, parent_id: editing.value.parent_id
      })
      message.success('创建成功')
    }
    showForm.value = false
    loadList()
  } catch {} finally {
    submitting.value = false
  }
}

async function handleDelete(row: Department) {
  if (!confirm(`确认删除「${row.name}」？`)) return
  try {
    await deptApi.delete(row.id)
    message.success('已删除')
    loadList()
  } catch {}
}

onMounted(loadList)
</script>