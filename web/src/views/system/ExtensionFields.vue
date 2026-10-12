<template>
  <div>
    <div class="page-header">
      <n-space>
        <n-tag type="info" size="small">管理员配置：前端需要新增字段</n-tag>
        <n-button type="primary" @click="openCreate">新增选项</n-button>
      </n-space>
    </div>

    <n-card size="small" style="margin-bottom: 12px">
      <p style="margin: 0 0 4px 0"><strong>说明</strong></p>
      <p style="margin: 0; color: #666; font-size: 12px; line-height: 1.6">
        • <strong>code</strong>:字段英文 key,提交后存入 lead.extension JSON 的这个字段,不可重复、不可修改<br>
        • <strong>type</strong>:字段类型(text 单行 / textarea 多行 / number 数字 / select 下拉 / date 日期)<br>
        • <strong>enabled</strong>:禁用后拉新端不再显示,但历史数据保留<br>
        • <strong>select 类型</strong>:可以填"候选项" JSON,或选择"引用字典分类"(优先);引用字典后选项与字典同步,免维护
      </p>
    </n-card>

    <n-data-table :columns="columns" :data="fields" :loading="loading" :bordered="false" />

    <n-modal v-model:show="showForm" preset="card" :title="editing.id ? '编辑字段' : '新增字段'" style="width: 600px">
      <n-form :model="editing" label-placement="left" label-width="100">
        <n-grid :cols="2" :x-gap="16">
          <n-gi>
            <n-form-item label="code" required>
              <n-input v-model:value="editing.code" :disabled="!!editing.id" placeholder="英文/拼音,如 wechat_id" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="显示名" required>
              <n-input v-model:value="editing.label" placeholder="如 微信号" />
            </n-form-item>
          </n-gi>
        </n-grid>
        <n-grid :cols="2" :x-gap="16">
          <n-gi>
            <n-form-item label="类型" required>
              <n-select v-model:value="editing.type" :options="typeOptions" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="排序">
              <n-input-number v-model:value="editing.sort" :min="0" style="width: 100%" />
            </n-form-item>
          </n-gi>
        </n-grid>
        <n-grid :cols="2" :x-gap="16">
          <n-gi>
            <n-form-item label="必填">
              <n-switch v-model:value="editing.required" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="启用">
              <n-switch v-model:value="editing.enabled" />
            </n-form-item>
          </n-gi>
        </n-grid>
        <n-form-item v-if="editing.type === 'select'" label="候选项来源">
          <n-space vertical size="small" style="width: 100%">
            <n-radio-group v-model:value="editing.optionsSource" size="small">
              <n-radio value="static">自定义 JSON</n-radio>
              <n-radio value="dict">引用字典分类</n-radio>
            </n-radio-group>
            <n-input
              v-if="editing.optionsSource === 'static'"
              v-model:value="editing.optionsJson"
              type="textarea"
              :rows="4"
              placeholder='[{"label":"微信","value":"wechat"},{"label":"QQ","value":"qq"}]'
            />
            <n-select
              v-else
              v-model:value="editing.dictCategory"
              :options="dictCategoryOptions"
              placeholder="选择字典分类(选项自动同步)"
            />
            <n-text v-if="editing.optionsSource === 'dict'" depth="3" style="font-size: 12px">
              引用字典后,选项从「字典管理」实时拉取;在此填写的候选项会被忽略
            </n-text>
          </n-space>
        </n-form-item>
        <n-form-item label="占位提示">
          <n-input v-model:value="editing.placeholder" placeholder="输入提示文字" />
        </n-form-item>
        <n-form-item label="管理员备注">
          <n-input v-model:value="editing.remark" type="textarea" :rows="2" />
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
import { ref, reactive, h, onMounted, watch } from 'vue'
import { NCard, NDataTable, NButton, NSpace, NTag, NModal, NForm, NFormItem, NInput, NSelect, NSwitch, NInputNumber, NGrid, NGi, NRadioGroup, NRadio, useMessage, type DataTableColumns } from 'naive-ui'
import http from '@/api/client'

const message = useMessage()
const fields = ref<any[]>([])
const loading = ref(false)
const showForm = ref(false)
const submitting = ref(false)

const typeOptions = [
  { label: '单行文本 text', value: 'text' },
  { label: '多行文本 textarea', value: 'textarea' },
  { label: '数字 number', value: 'number' },
  { label: '下拉选择 select', value: 'select' },
  { label: '日期 date', value: 'date' }
]

// 字典分类选项(目前只支持 channel)
const dictCategoryOptions = [
  { label: '来源渠道 (channel)', value: 'channel' }
]

const editing = ref<any>(newField())

function newField() {
  return {
    id: null,
    code: '',
    label: '',
    type: 'text',
    required: false,
    enabled: true,
    sort: 0,
    optionsJson: '',
    dictCategory: '',
    optionsSource: 'static' as 'static' | 'dict',
    placeholder: '',
    remark: ''
  }
}

const columns: DataTableColumns<any> = [
  { title: '排序', key: 'sort', width: 60 },
  { title: 'code', key: 'code', width: 120 },
  { title: '显示名', key: 'label', width: 140 },
  {
    title: '类型', key: 'type', width: 100,
    render: (r) => h(NTag, { size: 'small', type: 'info' }, () => r.type)
  },
  {
    title: '选项来源', key: 'dict_category', width: 140,
    render: (r) => {
      if (r.dict_category) return h(NTag, { size: 'small', type: 'success' }, () => '字典: ' + (dictCategoryOptions.find(o => o.value === r.dict_category)?.label.split(' ')[0] || r.dict_category))
      if (r.options && Object.keys(r.options).length > 0) return h(NTag, { size: 'small', type: 'info' }, () => '自定义')
      return '-'
    }
  },
  {
    title: '必填', key: 'required', width: 70,
    render: (r) => r.required ? h(NTag, { size: 'small', type: 'warning' }, () => '必填') : '-'
  },
  {
    title: '启用', key: 'enabled', width: 70,
    render: (r) => h(NTag, { size: 'small', type: r.enabled ? 'success' : 'default' }, () => r.enabled ? '✓' : '✗')
  },
  { title: '占位提示', key: 'placeholder', ellipsis: { tooltip: true } },
  {
    title: '操作', key: 'actions', width: 160, fixed: 'right',
    render: (r) => h(NSpace, {}, () => [
      h(NButton, { size: 'small', text: true, type: 'primary', onClick: () => openEdit(r) }, () => '编辑'),
      h(NButton, { size: 'small', text: true, type: 'error', onClick: () => handleDelete(r) }, () => '删除')
    ])
  }
]

async function loadList() {
  loading.value = true
  try {
    const resp: any = await http.get('/lead-extension-fields/admin')
    fields.value = resp.data || []
  } catch {
    message.error('加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = newField()
  showForm.value = true
}

watch(() => editing.value.type, (newType) => {
  if (newType !== 'select') {
    editing.value.optionsSource = 'static'
    editing.value.dictCategory = ''
  }
})

function openEdit(row: any) {
  const hasDict = !!row.dict_category
  editing.value = {
    id: row.id,
    code: row.code,
    label: row.label,
    type: row.type,
    required: !!row.required,
    enabled: !!row.enabled,
    sort: row.sort || 0,
    optionsJson: row.options && Object.keys(row.options).length > 0 ? JSON.stringify(row.options, null, 2) : '',
    dictCategory: row.dict_category || '',
    optionsSource: hasDict ? 'dict' : 'static',
    placeholder: row.placeholder || '',
    remark: row.remark || ''
  }
  showForm.value = true
}

async function handleSubmit() {
  if (!editing.value.code || !editing.value.label || !editing.value.type) {
    message.warning('code / 显示名 / 类型 必填')
    return
  }

  // 校验 optionsJson
  let opts = {}
  if (editing.value.type === 'select' && editing.value.optionsJson.trim()) {
    try {
      opts = JSON.parse(editing.value.optionsJson)
      if (!Array.isArray(opts)) {
        message.warning('候选项必须是 JSON 数组')
        return
      }
    } catch {
      message.warning('候选项 JSON 格式错误')
      return
    }
  }

  const payload = {
    code: editing.value.code,
    label: editing.value.label,
    type: editing.value.type,
    required: editing.value.required,
    enabled: editing.value.enabled,
    sort: editing.value.sort,
    options: editing.value.optionsSource === 'static' ? opts : {},
    dict_category: editing.value.optionsSource === 'dict' ? editing.value.dictCategory : '',
    placeholder: editing.value.placeholder,
    remark: editing.value.remark
  }

  submitting.value = true
  try {
    if (editing.value.id) {
      await http.put(`/lead-extension-fields/${editing.value.id}`, payload)
      message.success('更新成功')
    } else {
      await http.post('/lead-extension-fields', payload)
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

async function handleDelete(row: any) {
  if (!confirm(`确认删除字段「${row.label}」(${row.code})?删除后历史数据保留,但拉新端不再显示。`)) return
  try {
    await http.delete(`/lead-extension-fields/${row.id}`)
    message.success('已删除')
    loadList()
  } catch (e: any) {
    message.error(e?.message || '删除失败')
  }
}

onMounted(loadList)
</script>

<style scoped>
.page-header { display: flex; justify-content: flex-end; align-items: center; margin-bottom: 16px; }
</style>