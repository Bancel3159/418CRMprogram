<template>
  <n-drawer
    :show="show"
    :width="720"
    placement="right"
    @update:show="onClose"
  >
    <n-drawer-content closable :title="title" :native-scrollbar="false">
      <template #header-extra>
        <n-tag type="warning" size="small" :bordered="false">P13 强制编辑 · 仅 admin</n-tag>
      </template>

      <n-alert type="warning" :bordered="false" style="margin-bottom: 16px">
        <template #default>
          <div style="font-size: 12px; line-height: 1.6">
            本操作<b>绕过常规业务校验</b>,可修改归属(游戏/拉新员/维护员)、状态机、付费标记、流失状态等。
            所有改动会写入<b>操作审计</b>(系统管理  操作审计),含 before/after 全量快照。
          </div>
        </template>
      </n-alert>

      <n-spin :show="loading">
        <n-form label-placement="top" :disabled="submitting">
          <!-- ========== 归属 / 流转 ========== -->
          <n-divider title-placement="left">
            <span style="font-size: 13px">归属与流转</span>
          </n-divider>

          <n-grid :cols="2" :x-gap="12" :y-gap="12">
            <n-gi>
              <n-form-item label="游戏 game_id">
                <n-select v-model:value="form.game_id" :options="gameOptions" filterable placeholder="选择游戏"
                  :consistent-menu-width="false" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="拉新员 creator_id">
                <n-select v-model:value="form.creator_id" :options="userOptions" filterable placeholder="选择拉新员"
                  :consistent-menu-width="false" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="维护员 claimed_by">
                <n-select v-model:value="form.claimed_by" :options="userOptions" filterable clearable placeholder="未认领"
                  :consistent-menu-width="false" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="状态 status">
                <n-select v-model:value="form.status" :options="statusOptions" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="渠道 channel">
                <n-input v-model:value="form.channel" placeholder="如 api_test / discord" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="释放人 released_by">
                <n-select v-model:value="form.released_by" :options="userOptions" filterable clearable placeholder="无"
                  :consistent-menu-width="false" />
              </n-form-item>
            </n-gi>
          </n-grid>

          <!-- ========== 13 核心业务字段 ========== -->
          <n-divider title-placement="left">
            <span style="font-size: 13px">核心业务字段</span>
          </n-divider>

          <n-grid :cols="2" :x-gap="12" :y-gap="12">
            <n-gi>
              <n-form-item label="玩家 ID player_id">
                <n-input v-model:value="form.player_id" placeholder="合作方唯一标识" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="玩家账号 player_username">
                <n-input v-model:value="form.player_username" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="游戏内昵称 player_game_name">
                <n-input v-model:value="form.player_game_name" clearable />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="区服 region">
                <n-input v-model:value="form.region" clearable />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="国家 player_country">
                <n-input v-model:value="form.player_country" clearable />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="年龄 player_age">
                <n-input-number v-model:value="form.player_age" :min="0" :max="150" style="width: 100%"
                  placeholder="0-150" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="是否目标 is_target">
                <n-select v-model:value="form.is_target" :options="targetOptions" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="社群 community">
                <n-input v-model:value="form.community" clearable />
              </n-form-item>
            </n-gi>
            <n-gi span="2">
              <n-form-item label="Discord token token_code">
                <n-input v-model:value="form.token_code" type="textarea" :rows="2" clearable
                  placeholder="敏感信息,仅 admin 可见" />
              </n-form-item>
            </n-gi>
            <n-gi span="2">
              <n-form-item label="备注 remark">
                <n-input v-model:value="form.remark" type="textarea" :rows="3" />
              </n-form-item>
            </n-gi>
          </n-grid>

          <!-- ========== 付费 / 流失状态 ========== -->
          <n-divider title-placement="left">
            <span style="font-size: 13px">付费与流失状态</span>
          </n-divider>

          <n-grid :cols="2" :x-gap="12" :y-gap="12">
            <n-gi>
              <n-form-item label="人工标记已付费 paid_mark">
                <n-switch v-model:value="form.paid_mark" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="是否流失 is_lost">
                <n-switch v-model:value="form.is_lost" />
              </n-form-item>
            </n-gi>
            <n-gi v-if="form.is_lost">
              <n-form-item label="流失原因 lost_reason">
                <n-input v-model:value="form.lost_reason" placeholder="如 manual / auto_paid_offline" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="玩家最后活跃时间">
                <n-date-picker v-model:value="form.last_player_active_at_ts" type="datetime" style="width: 100%" clearable />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="再次激活时间">
                <n-date-picker v-model:value="form.reactivated_at_ts" type="datetime" style="width: 100%" clearable />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="认领时间">
                <n-date-picker v-model:value="form.claimed_at_ts" type="datetime" style="width: 100%" clearable />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="释放时间">
                <n-date-picker v-model:value="form.released_at_ts" type="datetime" style="width: 100%" clearable />
              </n-form-item>
            </n-gi>
          </n-grid>

          <!-- ========== 扩展字段 ========== -->
          <n-divider title-placement="left">
            <span style="font-size: 13px">
              扩展字段 extension
              <n-button text type="primary" size="tiny" style="margin-left: 8px" @click="extDrawer = true">
                编辑 JSON
              </n-button>
            </span>
          </n-divider>
          <n-input
            :value="extPreview"
            type="textarea"
            :rows="3"
            readonly
            placeholder="(空)"
            style="font-family: Consolas, monospace; font-size: 12px"
          />

          <!-- ========== 变更预览 ========== -->
          <n-divider title-placement="left">
            <span style="font-size: 13px">本次将修改的字段 ({{ changedFields.length }})</span>
          </n-divider>
          <div v-if="changedFields.length === 0" style="color: #9ca3af; font-size: 13px; padding: 8px 0">
            尚未修改任何字段
          </div>
          <n-space v-else size="small" :max="20">
            <n-tag v-for="f in changedFields" :key="f" size="small" type="warning" :bordered="false">
              {{ fieldLabel(f) }}
            </n-tag>
          </n-space>
        </n-form>
      </n-spin>

      <template #footer>
        <n-space justify="space-between">
          <n-button :disabled="submitting" @click="resetForm">重置</n-button>
          <n-space>
            <n-button :disabled="submitting" @click="onClose">取消</n-button>
            <n-button
              type="primary"
              :loading="submitting"
              :disabled="changedFields.length === 0"
              @click="handleSubmit"
            >
              保存修改({{ changedFields.length }})
            </n-button>
          </n-space>
        </n-space>
      </template>
    </n-drawer-content>

    <!-- extension JSON 编辑子弹窗 -->
    <n-modal v-model:show="extDrawer" preset="card" title="编辑 extension JSON" style="width: 560px">
      <n-input
        v-model:value="extText"
        type="textarea"
        :rows="12"
        placeholder='{ "key": "value" }'
        style="font-family: Consolas, monospace; font-size: 12px"
      />
      <template #footer>
        <n-space justify="end">
          <n-button @click="extDrawer = false">取消</n-button>
          <n-button type="primary" @click="applyExt">应用</n-button>
        </n-space>
      </template>
    </n-modal>
  </n-drawer>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import {
  NDrawer, NDrawerContent, NForm, NFormItem, NGrid, NGi, NInput,
  NInputNumber, NSelect, NSwitch, NDatePicker, NButton, NSpace, NTag,
  NDivider, NAlert, NSpin, NModal, useMessage, useDialog
} from 'naive-ui'
import { leadApi, gameApi, type Game } from '@/api/modules'

const props = defineProps<{
  show: boolean
  leadId: number | null
}>()
const emit = defineEmits<{ (e: 'update:show', v: boolean): void; (e: 'saved'): void }>()

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const submitting = ref(false)
const extDrawer = ref(false)
const extText = ref('{}')

const gameOptions = ref<any[]>([])
const userOptions = ref<any[]>([])

const statusOptions = [
  { label: '待认领 pending', value: 'pending' },
  { label: '已认领 claimed', value: 'claimed' },
  { label: '已流失 archived', value: 'archived' }
]
const targetOptions = [
  { label: '目标 target', value: 'target' },
  { label: '非目标 not_target', value: 'not_target' }
]

// 表单模型
const form = reactive<any>({
  game_id: null, creator_id: null, claimed_by: null, released_by: null,
  channel: '', status: 'pending',
  player_id: '', player_username: '', player_game_name: '',
  region: '', player_country: '', player_age: null,
  is_target: 'target', community: '', token_code: '', remark: '',
  paid_mark: false, is_lost: false, lost_reason: '',
  extension: null as Record<string, any> | null,
  // 原始快照(用于 diff)
  _orig: null as any,
  // 日期用 timestamp(naive-ui date-picker 格式)
  last_player_active_at_ts: null as number | null,
  reactivated_at_ts: null as number | null,
  claimed_at_ts: null as number | null,
  released_at_ts: null as number | null
})

const title = computed(() => {
  const pid = form.player_id || '—'
  return `强制编辑 · ${pid}`
})

const extPreview = computed(() => {
  if (!form.extension || Object.keys(form.extension).length === 0) return ''
  return JSON.stringify(form.extension, null, 2)
})

// 字段中文名映射(用于变更预览 tag)
const FIELD_LABELS: Record<string, string> = {
  game_id: '游戏', creator_id: '拉新员', channel: '渠道', player_username: '玩家账号',
  player_game_name: '游戏内昵称', player_id: '玩家ID', region: '区服',
  player_country: '国家', player_age: '年龄', is_target: '是否目标', community: '社群',
  token_code: 'Discord Token', status: '状态', claimed_by: '维护员', claimed_at: '认领时间',
  released_by: '释放人', released_at: '释放时间', extension: '扩展字段', remark: '备注',
  paid_mark: '付费标记', is_lost: '是否流失',
  lost_reason: '流失原因', reactivated_at: '再次激活时间',
  last_player_active_at: '最后活跃时间'
}
function fieldLabel(f: string) {
  return FIELD_LABELS[f] || f
}

// 后端自动维护的派生字段 —— 不进 diff 列表(表单没有对应输入控件)
//   lost_at      : is_lost 变 true 时后端自动写 now
//   paid_marked_at: paid_mark 变更时后端自动写 now / 置 NULL
const DERIVED_FIELDS = new Set(['lost_at', 'paid_marked_at'])

// ===== diff:对比当前表单 vs 原始快照 =====
const FIELD_KEYS = Object.keys(FIELD_LABELS)

const changedFields = computed(() => {
  const orig = form._orig
  if (!orig) return []
  const out: string[] = []
  for (const k of FIELD_KEYS) {
    if (DERIVED_FIELDS.has(k)) continue
    let a = orig[k]
    let b: any
    if (k.endsWith('_at')) {
      // 时间字段:表单存 timestamp,DB 存 ISO 字符串
      a = orig[k] ? new Date(orig[k]).getTime() : null
      b = form[k + '_ts'] ?? null
    } else {
      b = form[k]
    }
    // 归一化比较
    const na = a === null || a === undefined ? '' : String(a)
    const nb = b === null || b === undefined ? '' : String(b)
    if (na !== nb) out.push(k)
  }
  return out
})

// ===== 加载 =====
async function loadOptions() {
  try {
    const r: any = await gameApi.list()
    const list = (r.data?.list || r.data || []) as Game[]
    gameOptions.value = list.map((g) => ({ label: `${g.code} - ${g.name}`, value: g.id }))
  } catch {}
  try {
    const r: any = await (await import('@/api/modules')).userApi.options()
    const list = (r.data || []) as any[]
    userOptions.value = list.map((u) => ({ label: `${u.username}${u.real_name ? ` (${u.real_name})` : ''}`, value: u.id }))
  } catch {}
}

async function loadLead() {
  if (!props.leadId) return
  loading.value = true
  try {
    const r: any = await leadApi.get(props.leadId)
    const d = r.data
    if (!d) return
    form.game_id = d.game_id
    form.creator_id = d.creator_id
    form.claimed_by = d.claimed_by ?? null
    form.released_by = d.released_by ?? null
    form.channel = d.channel || ''
    form.status = d.status || 'pending'
    form.player_id = d.player_id || ''
    form.player_username = d.player_username || ''
    form.player_game_name = d.player_game_name || ''
    form.region = d.region || ''
    form.player_country = d.player_country || ''
    form.player_age = d.player_age ?? null
    form.is_target = d.is_target || 'target'
    form.community = d.community || ''
    form.token_code = d.token_code || ''
    form.remark = d.remark || ''
    form.paid_mark = !!d.paid_mark
    form.is_lost = !!d.is_lost
    form.lost_reason = d.lost_reason || ''
    form.extension = d.extension && Object.keys(d.extension).length > 0 ? { ...d.extension } : null
    form.last_player_active_at_ts = d.last_player_active_at ? new Date(d.last_player_active_at).getTime() : null
    form.reactivated_at_ts = d.reactivated_at ? new Date(d.reactivated_at).getTime() : null
    form.claimed_at_ts = d.claimed_at ? new Date(d.claimed_at).getTime() : null
    form.released_at_ts = d.released_at ? new Date(d.released_at).getTime() : null
    form._orig = JSON.parse(JSON.stringify(d))
    extText.value = form.extension ? JSON.stringify(form.extension, null, 2) : '{}'
  } catch (e: any) {
    message.error('加载失败: ' + (e?.message || ''))
  } finally {
    loading.value = false
  }
}

watch(() => [props.show, props.leadId], ([s]) => {
  if (s && props.leadId) {
    loadOptions()
    loadLead()
  }
})

// ===== 提交 =====
function buildPayload(): any {
  const p: any = {}
  const has = changedFields.value
  for (const k of has) {
    if (k.endsWith('_at')) {
      // 时间字段:表单 timestamp  ISO
      const ts = form[k + '_ts']
      p[k] = ts ? new Date(ts).toISOString() : null
    } else {
      p[k] = form[k]
    }
  }
  // is_lost 变 true 时带上 lost_reason
  if (has.includes('is_lost') && form.is_lost) {
    p.lost_reason = form.lost_reason || 'manual_force_edit'
  }
  return p
}

function handleSubmit() {
  const payload = buildPayload()
  const keys = Object.keys(payload)
  if (keys.length === 0) {
    message.warning('没有需要保存的变更')
    return
  }

  dialog.warning({
    title: '确认强制编辑',
    content: `即将修改 lead #${props.leadId} 的 ${keys.length} 个字段:\n\n${keys.map(fieldLabel).join('、')}\n\n此操作会写入审计日志,确认继续?`,
    positiveText: '确认修改',
    negativeText: '取消',
    onPositiveClick: async () => {
      submitting.value = true
      try {
        const r: any = await leadApi.forceEdit(props.leadId!, payload)
        const changed = r.data?.changed_fields || []
        message.success(`已修改 ${changed.length} 个字段`)
        emit('saved')
        emit('update:show', false)
      } catch (e: any) {
        message.error('保存失败: ' + (e?.message || e?.response?.data?.message || ''))
      } finally {
        submitting.value = false
      }
    }
  })
}

function applyExt() {
  const t = (extText.value || '').trim()
  if (!t) {
    form.extension = null
    extDrawer.value = false
    return
  }
  try {
    const parsed = JSON.parse(t)
    if (typeof parsed !== 'object' || Array.isArray(parsed)) {
      message.error('extension 必须是 JSON 对象')
      return
    }
    form.extension = parsed
    extDrawer.value = false
  } catch {
    message.error('JSON 格式错误')
  }
}

function resetForm() {
  if (!form._orig) return
  loadLead()
}

function onClose() {
  emit('update:show', false)
}
</script>