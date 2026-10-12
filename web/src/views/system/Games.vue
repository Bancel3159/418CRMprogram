<template>
  <div>
    <div class="page-header">
      <n-button type="primary" @click="openCreate">新增游戏</n-button>
    </div>
    <n-data-table :columns="columns" :data="games" :loading="loading" :bordered="false" />

    <n-modal v-model:show="showForm" preset="card" :title="editing.id ? '编辑游戏' : '新增游戏'" style="width: 720px">
      <n-form :model="editing" label-placement="left" label-width="110">
        <n-form-item label="缩写"><n-input v-model:value="editing.code" :disabled="!!editing.id" placeholder="TWD" /></n-form-item>
        <n-form-item label="名称"><n-input v-model:value="editing.name" placeholder="The Walking Dead Survivors" /></n-form-item>
        <n-form-item label="所属公司"><n-input v-model:value="editing.company" /></n-form-item>
        <n-form-item label="状态">
          <n-select v-model:value="editing.status" :options="[{label:'启用',value:'enabled'},{label:'禁用',value:'disabled'}]" />
        </n-form-item>
        <n-form-item label="备注"><n-input v-model:value="editing.remark" type="textarea" /></n-form-item>

        <!-- 数据回传 -->
        <n-divider style="margin: 12px 0">
          <n-space size="small" align="center">
            <n-text strong>数据回传</n-text>
            <n-switch v-model:value="partnerEnabled" @update:value="onPartnerToggle" />
            <n-tag v-if="partnerEnabled" size="small" type="info">已启用 ({{ partnerTypeLabel }})</n-tag>
            <n-tag v-else size="small" type="default">未启用 · 仅手动导入订单</n-tag>
          </n-space>
        </n-divider>

        <template v-if="partnerEnabled">
          <n-form-item label="接入方式">
            <n-radio-group v-model:value="editing.partner_type">
              <n-radio value="api">API 主动拉取</n-radio>
              <n-radio value="webhook">Webhook 接收推送</n-radio>
            </n-radio-group>
          </n-form-item>

          <!-- Webhook 配置 -->
          <template v-if="editing.partner_type === 'webhook'">
            <n-form-item label="HMAC 密钥">
              <n-input v-model:value="editing.partner_config.hmac_secret" type="password" show-password-on="click" placeholder="用于 X-Signature 验签" />
            </n-form-item>
            <n-form-item label="回调 URL">
              <n-input :value="webhookUrl" readonly />
              <template #feedback>
                <n-text depth="3" style="font-size: 12px">
                  上游系统把订单 POST 到此 URL,使用 HMAC-SHA256(body, secret) 签名,header <code>X-Signature: sha256=&lt;hex&gt;</code>
                </n-text>
              </template>
            </n-form-item>
          </template>

          <!-- API 配置 -->
          <template v-if="editing.partner_type === 'api'">
            <n-form-item label="API 地址"><n-input v-model:value="editing.partner_config.base_url" placeholder="https://api.partner.com" /></n-form-item>
            <n-form-item label="认证方式">
              <n-select v-model:value="editing.partner_config.auth_type" :options="[
                { label: 'Bearer Token', value: 'bearer' },
                { label: 'Basic 用户名密码', value: 'basic' },
                { label: 'API Key (header)', value: 'apikey' }
              ]" />
            </n-form-item>
            <n-form-item v-if="editing.partner_config.auth_type === 'bearer'" label="Token">
              <n-input v-model:value="editing.partner_config.token" type="password" show-password-on="click" />
            </n-form-item>
            <n-form-item v-if="editing.partner_config.auth_type === 'basic'" label="用户名">
              <n-input v-model:value="editing.partner_config.username" />
            </n-form-item>
            <n-form-item v-if="editing.partner_config.auth_type === 'basic'" label="密码">
              <n-input v-model:value="editing.partner_config.password" type="password" show-password-on="click" />
            </n-form-item>
            <n-form-item v-if="editing.partner_config.auth_type === 'apikey'" label="Key">
              <n-input v-model:value="editing.partner_config.api_key" type="password" show-password-on="click" />
            </n-form-item>
            <n-form-item v-if="editing.partner_config.auth_type === 'apikey'" label="Header 名">
              <n-input v-model:value="editing.partner_config.header" placeholder="X-Api-Key" />
            </n-form-item>
            <n-form-item label="Cron 表达式">
              <n-input v-model:value="editing.partner_config.cron_expr" placeholder="0 */15 * * * * (空 = 仅手动)" />
            </n-form-item>
            <n-form-item label="字段映射 (JSON)">
              <n-input
                v-model:value="fieldMappingsJson"
                type="textarea"
                :rows="3"
                placeholder='{"order_no":"id","amount":"money","player_id":"uid"}'
              />
              <template #feedback>
                <n-text depth="3" style="font-size: 12px">
                  把上游字段名映射成标准字段。target_key 写标准名,value 写上游字段名
                </n-text>
              </template>
            </n-form-item>
          </template>
        </template>
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
import { ref, computed, onMounted, h, watch } from 'vue'
import { NDataTable, NButton, NTag, NModal, NForm, NFormItem, NInput, NSelect, NSpace, NDivider, NSwitch, NRadioGroup, NRadio, NText, useMessage, type DataTableColumns } from 'naive-ui'
import { gameApi, type Game } from '@/api/modules'

const message = useMessage()
const games = ref<Game[]>([])
const loading = ref(false)
const showForm = ref(false)
const submitting = ref(false)
const editing = ref<any>({
  id: null, code: '', name: '', company: '', status: 'enabled', remark: '',
  partner_type: '' as '' | 'api' | 'webhook',
  partner_config: {} as any
})
const partnerEnabled = ref(false)

const partnerTypeLabel = computed(() => editing.value.partner_type === 'api' ? 'API' : editing.value.partner_type === 'webhook' ? 'Webhook' : '')

const fieldMappingsJson = ref('')

const webhookUrl = computed(() => {
  if (!editing.value.code || editing.value.partner_type !== 'webhook') return ''
  return `${window.location.origin}/api/v1/orders/webhook/${editing.value.code}`
})

const columns: DataTableColumns<Game> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '缩写', key: 'code', width: 100 },
  { title: '名称', key: 'name' },
  { title: '公司', key: 'company', width: 130 },
  {
    title: '数据回传', key: 'partner_type', width: 130,
    render: (row) => {
      if (row.partner_enabled) {
        const typeLabel = row.partner_type === 'api' ? 'API 拉取' : row.partner_type === 'webhook' ? 'Webhook 推送' : '已启用'
        return h(NTag, { type: 'success', size: 'small' }, () => typeLabel)
      }
      return h('span', { style: 'color: #94a3b8' }, '—')
    }
  },
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
    const resp: any = await gameApi.list({ page: 1, page_size: 100 })
    games.value = resp.data.list || []
  } catch {} finally {
    loading.value = false
  }
}

function newPartnerConfig(type: '' | 'api' | 'webhook', code: string) {
  if (type === 'webhook') return { hmac_secret: '' }
  if (type === 'api') return { base_url: '', auth_type: 'bearer', token: '', username: '', password: '', api_key: '', header: 'X-Api-Key', cron_expr: '', field_mappings: {} }
  return {}
}

function openCreate() {
  editing.value = {
    id: null, code: '', name: '', company: '', status: 'enabled', remark: '',
    partner_type: '', partner_config: {}
  }
  partnerEnabled.value = false
  fieldMappingsJson.value = ''
  showForm.value = true
}

function openEdit(row: Game) {
  const pt = (row.partner_type as any) || ''
  editing.value = {
    id: row.id,
    code: row.code,
    name: row.name,
    company: row.company,
    status: row.status,
    remark: row.remark,
    partner_type: pt,
    partner_config: row.partner_config ? { ...row.partner_config } : {}
  }
  partnerEnabled.value = !!(row.partner_enabled)
  if (row.partner_config && (row.partner_config as any).field_mappings) {
    fieldMappingsJson.value = JSON.stringify((row.partner_config as any).field_mappings, null, 2)
  } else {
    fieldMappingsJson.value = ''
  }
  showForm.value = true
}

function onPartnerToggle(v: boolean) {
  if (v && !editing.value.partner_type) {
    editing.value.partner_type = 'webhook'
    editing.value.partner_config = newPartnerConfig('webhook', editing.value.code)
  } else if (!v) {
    editing.value.partner_type = ''
    editing.value.partner_config = {}
  }
}

// 切换类型时重置 config
watch(() => editing.value.partner_type, (newType, oldType) => {
  if (newType && newType !== oldType && partnerEnabled.value) {
    editing.value.partner_config = newPartnerConfig(newType, editing.value.code)
  }
})

async function handleSubmit() {
  // 构造 partner_config(序列化 field_mappings)
  let partnerConfig: any = null
  if (partnerEnabled.value && editing.value.partner_type) {
    partnerConfig = { ...editing.value.partner_config }
    if (editing.value.partner_type === 'api') {
      try {
        partnerConfig.field_mappings = fieldMappingsJson.value.trim() ? JSON.parse(fieldMappingsJson.value) : {}
      } catch {
        message.error('字段映射 JSON 格式错误')
        return
      }
    }
  }

  submitting.value = true
  try {
    if (editing.value.id) {
      await gameApi.update(editing.value.id, {
        name: editing.value.name,
        company: editing.value.company,
        status: editing.value.status,
        remark: editing.value.remark,
        partner_type: partnerEnabled.value ? editing.value.partner_type : '',
        partner_config: partnerConfig
      })
      message.success('更新成功')
    } else {
      await gameApi.create({
        code: editing.value.code,
        name: editing.value.name,
        company: editing.value.company,
        status: editing.value.status,
        remark: editing.value.remark,
        partner_type: partnerEnabled.value ? editing.value.partner_type : '',
        partner_config: partnerConfig || {}
      })
      message.success('创建成功')
    }
    showForm.value = false
    loadList()
  } catch (err: any) {
    message.error(err?.message || '保存失败')
  } finally {
    submitting.value = false
  }
}

async function handleDelete(row: Game) {
  if (!confirm(`确认删除「${row.name}」？`)) return
  try {
    await gameApi.delete(row.id)
    message.success('已删除')
    loadList()
  } catch {}
}

onMounted(loadList)
</script>