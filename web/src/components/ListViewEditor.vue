<template>
  <div class="list-view-editor">
    <!-- 头部：当前编辑的视图 + 保存 -->
    <div class="editor-header">
      <n-space align="center">
        <span class="hint">当前编辑</span>
        <n-tag :type="view?.is_system ? 'success' : 'info'" size="large">
          {{ view?.name || '加载中...' }}
        </n-tag>
        <n-text v-if="view?.is_system" depth="3" style="font-size: 12px">
          (系统默认视图，名称不可改)
        </n-text>
      </n-space>
      <n-space>
        <n-tag v-if="dirty" type="warning" size="small">有未保存改动</n-tag>
        <n-button @click="reset" :disabled="!dirty || submitting">放弃改动</n-button>
        <n-button type="primary" @click="save" :disabled="!dirty || submitting" :loading="submitting">
          保存
        </n-button>
      </n-space>
    </div>

    <n-grid :cols="2" :x-gap="16" responsive="screen">
      <!-- 左侧：已配置字段（可拖排序）-->
      <n-gi span="1 m:1">
        <n-card size="small" :bordered="false" title="已配置字段（点击  排序；点  编辑；点  移除）">
          <n-data-table
            :columns="configuredColumns"
            :data="configuredFields"
            :bordered="false"
            :single-line="false"
            size="small"
            :row-key="(r: any) => r.__key"
          />
          <n-empty v-if="configuredFields.length === 0" description="尚未添加任何字段" />
        </n-card>
      </n-gi>

      <!-- 右侧：可用字段池（按来源分组） -->
      <n-gi span="1 m:1">
        <n-card size="small" :bordered="false" title="可用字段池（双击或点击 + 加入）">
          <n-input v-model:value="filterKeyword" placeholder="搜索 code / label..." clearable style="margin-bottom: 12px" />
          <div v-for="group in groupedAvailable" :key="group.group_key" class="group-block">
            <div class="group-title">{{ group.name }} ({{ group.items.length }})</div>
            <n-space size="small" wrap>
              <n-tag
                v-for="item in group.items"
                :key="item.code"
                :type="isConfigured(item.code) ? 'success' : 'default'"
                :bordered="!isConfigured(item.code)"
                style="cursor: pointer"
                @click="addField(item)"
                @dblclick="addField(item)"
              >
                + {{ item.label }} <span style="opacity: 0.6; font-size: 11px">({{ item.code }})</span>
              </n-tag>
            </n-space>
          </div>
          <n-empty v-if="groupedAvailable.length === 0" description="该 entity 暂无可用字段" />
        </n-card>
      </n-gi>
    </n-grid>

    <!-- 编辑字段的 Popover -->
    <n-modal v-model:show="showEditModal" preset="card" title="编辑字段" style="width: 480px">
      <n-form v-if="editingField" :model="editingField" label-placement="left" label-width="80">
        <n-form-item label="显示名">
          <n-input v-model:value="editingField.label" />
        </n-form-item>
        <n-grid :cols="2" :x-gap="16">
          <n-gi>
            <n-form-item label="列宽 (px)">
              <n-input-number v-model:value="editingField.width" :min="40" :max="600" style="width: 100%" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="对齐">
              <n-select v-model:value="editingField.align" :options="alignOptions" />
            </n-form-item>
          </n-gi>
        </n-grid>
        <n-form-item label="格式化">
          <n-select v-model:value="editingField.formatter" :options="formatterOptionsFor(editingField.source)" />
        </n-form-item>
        <n-text depth="3" style="font-size: 12px">
          数据源：{{ sourceLabel(editingField.source) }} · code: {{ editingField.code }}
        </n-text>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showEditModal = false">取消</n-button>
          <n-button type="primary" @click="confirmEdit">确定</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { NButton, NSpace, NTag, NCard, NGrid, NGi, NInput, NSelect, NInputNumber, NDataTable, NForm, NFormItem, NModal, NEmpty, NText, useMessage, type DataTableColumns } from 'naive-ui'
import { listViewApi, type ListView, type ListViewField, type ListViewAvailableField, type ListViewEntityType } from '@/api/modules'

const props = defineProps<{
  entityType: ListViewEntityType
}>()

const emit = defineEmits<{
  (e: 'saved', view: ListView): void
}>()

const message = useMessage()
const view = ref<ListView | null>(null)
const originalFields = ref<ListViewField[]>([])
const configuredFields = ref<ListViewField[]>([])
const availableFields = ref<ListViewAvailableField[]>([])
const filterKeyword = ref('')
const submitting = ref(false)
const showEditModal = ref(false)
const editingField = ref<ListViewField | null>(null)

const dirty = computed(() => JSON.stringify(configuredFields.value) !== JSON.stringify(originalFields.value))

const alignOptions = [
  { label: '左对齐', value: 'left' },
  { label: '居中', value: 'center' },
  { label: '右对齐', value: 'right' }
]

// 不同 source 对应的 formatter 选项
const formatterMap: Record<string, { label: string; value: string }[]> = {
  default: [
    { label: '默认文本', value: 'default' },
    { label: '时间', value: 'datetime' },
    { label: '数字', value: 'number' },
    { label: '金额', value: 'money' }
  ],
  lead_basic: [
    { label: '默认文本', value: 'default' },
    { label: '时间', value: 'datetime' },
    { label: '数字', value: 'number' },
    { label: '是否目标', value: 'target_tag' },
    { label: '是否付费', value: 'paid_tag' },
    { label: '线索状态', value: 'lead_status_tag' }
  ],
  lead_aggregation: [
    { label: '数字', value: 'number' },
    { label: '用户列表', value: 'user_list' }
  ],
  lead_time: [
    { label: '时间', value: 'datetime' }
  ],
  follow_record: [
    { label: '默认文本', value: 'default' },
    { label: '时间', value: 'datetime' },
    { label: '数字', value: 'number' }
  ],
  order_summary: [
    { label: '数字', value: 'number' },
    { label: '金额', value: 'money' },
    { label: '时间', value: 'datetime' }
  ],
  order_core: [
    { label: '默认文本', value: 'default' },
    { label: '金额', value: 'money' },
    { label: '订单状态', value: 'order_status_tag' },
    { label: '来源', value: 'source_tag' },
    { label: '归属状态', value: 'attribution_tag' },
    { label: '是否首充', value: 'first_charge_tag' },
    { label: '数字', value: 'number' }
  ],
  order_time: [
    { label: '时间', value: 'datetime' }
  ],
  lead_joined: [
    { label: '默认文本', value: 'default' },
    { label: '是否目标', value: 'target_tag' }
  ],
  order_meta: [
    { label: '默认文本', value: 'default' }
  ],
  extension: [
    { label: '默认文本', value: 'default' },
    { label: '时间', value: 'date' },
    { label: '数字', value: 'number' }
  ],
  raw_payload: [
    { label: '默认文本', value: 'default' },
    { label: '时间', value: 'datetime' },
    { label: '数字', value: 'number' },
    { label: '金额', value: 'money' }
  ]
}

function formatterOptionsFor(source: string) {
  return formatterMap[source] || formatterMap.default
}

function sourceLabel(source: string) {
  const map: Record<string, string> = {
    lead_basic: '拉新端核心',
    lead_aggregation: '聚合',
    lead_time: '时间',
    follow_record: '维护端跟进',
    order_summary: '订单聚合',
    order_core: '订单核心',
    order_time: '时间',
    lead_joined: '关联线索',
    order_meta: '元数据',
    extension: '自定义扩展',
    raw_payload: '数据回传原始'
  }
  return map[source] || source
}

// 已配置列表的列
const configuredColumns = computed<DataTableColumns<ListViewField>>(() => [
  {
    title: '顺序',
    key: '__sort',
    width: 90,
    render: (_r, idx) => h(NSpace, { size: 4 }, () => [
      h(NButton, { size: 'tiny', disabled: idx === 0, onClick: () => move(idx, -1) }, () => ''),
      h(NButton, { size: 'tiny', disabled: idx === configuredFields.value.length - 1, onClick: () => move(idx, 1) }, () => '')
    ])
  },
  { title: '显示名', key: 'label', width: 120, ellipsis: { tooltip: true } },
  { title: 'code', key: 'code', width: 180, ellipsis: { tooltip: true } },
  { title: '数据源', key: 'source', width: 100, render: (r) => h(NTag, { size: 'small', type: 'info' }, () => sourceLabel(r.source)) },
  { title: '列宽', key: 'width', width: 70, render: (r) => `${r.width}` },
  {
    title: '操作',
    key: 'actions',
    width: 110,
    fixed: 'right',
    render: (r, idx) => h(NSpace, { size: 4 }, () => [
      h(NButton, { size: 'tiny', type: 'primary', onClick: () => openEdit(r, idx) }, () => ''),
      h(NButton, { size: 'tiny', type: 'error', onClick: () => remove(idx) }, () => '')
    ])
  }
])

// 按 group_name 分组的可用字段
const groupedAvailable = computed(() => {
  const kw = filterKeyword.value.trim().toLowerCase()
  const filtered = availableFields.value.filter((f) =>
    !kw || f.code.toLowerCase().includes(kw) || f.label.toLowerCase().includes(kw)
  )
  const groups: Record<string, { group_key: string; name: string; items: ListViewAvailableField[] }> = {}
  for (const f of filtered) {
    if (!groups[f.group_key]) {
      groups[f.group_key] = { group_key: f.group_key, name: f.group_name, items: [] }
    }
    groups[f.group_key].items.push(f)
  }
  return Object.values(groups)
})

function isConfigured(code: string) {
  return configuredFields.value.some((f) => f.code === code)
}

function addField(item: ListViewAvailableField) {
  if (isConfigured(item.code)) return
  configuredFields.value.push({
    code: item.code,
    label: item.label,
    source: item.source,
    width: defaultWidthFor(item.code),
    align: defaultAlignFor(item.code),
    formatter: item.formatter
  })
}

function defaultWidthFor(code: string) {
  if (code.includes('amount') || code === 'total_order_amount') return 110
  if (code.includes('time') || code.includes('at')) return 160
  if (code === 'player_id' || code.includes('username')) return 140
  if (code === 'order_no') return 200
  if (code === 'remark' || code.includes('content')) return 200
  return 100
}

function defaultAlignFor(code: string) {
  if (code.includes('amount') || code.includes('count')) return 'right'
  if (code === 'is_target' || code === 'is_paid' || code.includes('status') || code.includes('first_charge') || code === 'game_code') return 'center'
  return 'left'
}

function move(idx: number, delta: number) {
  const newIdx = idx + delta
  if (newIdx < 0 || newIdx >= configuredFields.value.length) return
  const tmp = configuredFields.value[idx]
  configuredFields.value[idx] = configuredFields.value[newIdx]
  configuredFields.value[newIdx] = tmp
}

function remove(idx: number) {
  configuredFields.value.splice(idx, 1)
}

function openEdit(field: ListViewField, idx: number) {
  editingField.value = { ...field, __key: `${idx}` } as any
  showEditModal.value = true
}

function confirmEdit() {
  if (!editingField.value) return
  const idx = configuredFields.value.findIndex((f) => f.code === editingField.value!.code)
  if (idx >= 0) {
    configuredFields.value[idx] = { ...editingField.value }
  }
  showEditModal.value = false
  editingField.value = null
}

async function loadDefault() {
  try {
    const resp: any = await listViewApi.getDefault(props.entityType)
    view.value = resp.data
    const fields = (resp.data?.fields || []) as ListViewField[]
    configuredFields.value = fields.map((f) => ({ ...f }))
    originalFields.value = JSON.parse(JSON.stringify(fields))
  } catch (e: any) {
    message.error('加载默认视图失败：' + (e?.message || ''))
  }
}

async function loadAvailable() {
  try {
    const resp: any = await listViewApi.availableFields(props.entityType)
    availableFields.value = resp.data || []
  } catch (e: any) {
    message.error('加载字段池失败：' + (e?.message || ''))
  }
}

async function save() {
  if (!view.value) return
  if (configuredFields.value.length === 0) {
    message.warning('至少保留一个字段')
    return
  }
  submitting.value = true
  try {
    const payload = {
      fields: configuredFields.value.map((f) => ({
        code: f.code,
        label: f.label,
        source: f.source,
        width: f.width,
        align: f.align || 'left',
        formatter: f.formatter || 'default'
      }))
    }
    const resp: any = await listViewApi.update(view.value.id, payload)
    view.value = resp.data
    originalFields.value = JSON.parse(JSON.stringify(resp.data.fields || []))
    message.success('已保存')
    emit('saved', view.value)
  } catch (e: any) {
    message.error('保存失败：' + (e?.message || ''))
  } finally {
    submitting.value = false
  }
}

function reset() {
  configuredFields.value = JSON.parse(JSON.stringify(originalFields.value))
}

onMounted(() => {
  loadDefault()
  loadAvailable()
})
</script>

<style scoped>
.list-view-editor {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;
}
.hint {
  font-size: 12px;
  color: #999;
  margin-right: 4px;
}
.group-block {
  margin-bottom: 16px;
}
.group-title {
  font-size: 12px;
  color: #666;
  margin-bottom: 6px;
  font-weight: 600;
}
</style>