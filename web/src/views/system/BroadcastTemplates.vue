<template>
  <div>
    <div class="page-header">
      <n-space>
        <n-tag type="info" size="small">占位符：{游戏} {国家} {年龄} {区服} {token码} 等</n-tag>
        <n-button @click="openPreview">预览渲染</n-button>
        <n-button type="primary" @click="openCreate">新增模板</n-button>
      </n-space>
    </div>
    <n-data-table :columns="columns" :data="templates" :loading="loading" :bordered="false" />

    <n-modal v-model:show="showForm" preset="card" :title="editing.id ? '编辑模板' : '新增模板'" style="width: 700px">
      <n-form :model="editing" label-placement="left" label-width="100">
        <n-form-item label="名称"><n-input v-model:value="editing.name" /></n-form-item>
        <n-form-item label="模板内容">
          <n-input v-model:value="editing.template" type="textarea" :rows="4" placeholder="{游戏} 新玩家！{国家}{年龄}岁，加入{区服}区..." />
        </n-form-item>
        <n-grid :cols="3" :x-gap="16">
          <n-gi>
            <n-form-item label="权重"><n-input-number v-model:value="editing.weight" :min="1" /></n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="启用">
              <n-switch v-model:value="editing.enabled" />
            </n-form-item>
          </n-gi>
        </n-grid>
        <n-form-item label="备注"><n-input v-model:value="editing.remark" type="textarea" /></n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showForm = false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="handleSubmit">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 预览渲染 -->
    <n-modal v-model:show="showPreviewModal" preset="card" title="预览渲染（随机抽一条模板）" style="width: 700px">
      <n-form :model="previewData" label-placement="left" label-width="100">
        <n-grid :cols="3" :x-gap="16">
          <n-gi><n-form-item label="游戏"><n-input v-model:value="previewData.game" /></n-form-item></n-gi>
          <n-gi><n-form-item label="国家"><n-input v-model:value="previewData.country" /></n-form-item></n-gi>
          <n-gi><n-form-item label="年龄"><n-input v-model:value="previewData.age" /></n-form-item></n-gi>
          <n-gi><n-form-item label="区服"><n-input v-model:value="previewData.region" /></n-form-item></n-gi>
          <n-gi><n-form-item label="渠道"><n-input v-model:value="previewData.channel" /></n-form-item></n-gi>
          <n-gi><n-form-item label="社群"><n-input v-model:value="previewData.community" /></n-form-item></n-gi>
          <n-gi><n-form-item label="是否目标"><n-input v-model:value="previewData.is_target" /></n-form-item></n-gi>
          <n-gi><n-form-item label="玩家名"><n-input v-model:value="previewData.player_game_name" /></n-form-item></n-gi>
          <n-gi><n-form-item label="用户名"><n-input v-model:value="previewData.player_username" /></n-form-item></n-gi>
        </n-grid>
        <n-form-item label="token 码"><n-input v-model:value="previewData.token_code" /></n-form-item>
      </n-form>
      <div style="padding: 12px; background: #f5f7fa; border-radius: 4px; margin-top: 12px">
        <strong>渲染结果：</strong>
        <p>{{ previewResult }}</p>
      </div>
      <template #footer>
        <n-space justify="end">
          <n-button type="primary" @click="doPreview">重新渲染</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { NDataTable, NButton, NTag, NModal, NForm, NFormItem, NInput, NSelect, NSpace, NInputNumber, NSwitch, NGrid, NGi, useMessage, type DataTableColumns } from 'naive-ui'
import { broadcastApi, type BroadcastTemplate } from '@/api/modules'

const message = useMessage()
const templates = ref<BroadcastTemplate[]>([])
const loading = ref(false)
const showForm = ref(false)
const showPreviewModal = ref(false)
const submitting = ref(false)
const editing = ref<any>({ id: null, name: '', template: '', enabled: true, weight: 1, remark: '' })
const previewResult = ref('')
const previewData = ref({
  template_id: 0, game: 'TWD', country: '美国', age: '25', region: 'S1',
  channel: 'whatsapp', community: 'twd_global', is_target: '目标',
  player_game_name: 'ZombieMike', player_username: '+1-555-0123',
  token_code: 'MTIzNDU2.XYZ123.ABC456'
})

const columns: DataTableColumns<BroadcastTemplate> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '名称', key: 'name' },
  { title: '模板', key: 'template', ellipsis: { tooltip: true } },
  { title: '权重', key: 'weight', width: 80 },
  {
    title: '启用', key: 'enabled', width: 90,
    render: (row) => h(NTag, { type: row.enabled ? 'success' : 'default', size: 'small' }, () => row.enabled ? '启用' : '禁用')
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
    const resp: any = await broadcastApi.list()
    templates.value = resp.data || []
  } catch {} finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = { id: null, name: '', template: '', enabled: true, weight: 1, remark: '' }
  showForm.value = true
}

function openEdit(row: BroadcastTemplate) {
  editing.value = { ...row }
  showForm.value = true
}

async function handleSubmit() {
  submitting.value = true
  try {
    if (editing.value.id) {
      await broadcastApi.update(editing.value.id, editing.value)
      message.success('更新成功')
    } else {
      await broadcastApi.create(editing.value)
      message.success('创建成功')
    }
    showForm.value = false
    loadList()
  } catch {} finally {
    submitting.value = false
  }
}

async function handleDelete(row: BroadcastTemplate) {
  if (!confirm(`确认删除「${row.name}」？`)) return
  try {
    await broadcastApi.delete(row.id)
    message.success('已删除')
    loadList()
  } catch {}
}

function openPreview() {
  previewResult.value = ''
  showPreviewModal.value = true
  doPreview()
}

async function doPreview() {
  try {
    const resp: any = await broadcastApi.render(previewData.value)
    previewResult.value = resp.data.rendered
  } catch (err) {
    previewResult.value = '（渲染失败）'
  }
}

// 暴露给父组件(Pool.vue 嵌入式使用)
defineExpose({ loadList, templates })

onMounted(loadList)
</script>