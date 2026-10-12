<template>
  <div class="page">
    <!-- 顶部标题 + 重置按钮 -->
    <div class="page-head">
      <div>
        <div class="page-title">新注册</div>
        <div class="page-sub">填写注册信息 → 上传截图 → 推送给后端</div>
      </div>
      <n-button size="small" quaternary @click="resetForm">清空</n-button>
    </div>

    <n-form
      :model="form"
      :rules="rules"
      ref="formRef"
      label-placement="top"
      size="medium"
      :show-feedback="false"
    >
      <!-- 卡片 1:基础信息 -->
      <section class="card">
        <div class="card-title">基础信息</div>

        <n-form-item label="游戏" path="game_id">
          <n-select
            v-model:value="form.game_id"
            :options="gameOptions"
            placeholder="选择游戏"
          />
        </n-form-item>

        <n-form-item label="社交平台" path="channel">
          <n-select
            v-model:value="form.channel"
            :options="channelOptions"
            placeholder="请选择社交平台"
          />
        </n-form-item>

        <n-form-item label="社交平台用户名" path="player_username">
          <n-input
            v-model:value="form.player_username"
            placeholder="请输入玩家的社交平台用户名"
          />
        </n-form-item>

        <n-form-item label="国家">
          <n-input
            v-model:value="form.player_country"
            placeholder="如：美国"
          />
        </n-form-item>

        <n-form-item label="年龄">
          <n-input-number
            v-model:value="form.player_age"
            :min="0"
            :max="120"
            style="width: 100%"
          />
        </n-form-item>

        <n-form-item label="是否目标" path="is_target">
          <n-radio-group v-model:value="form.is_target" name="is_target">
            <n-radio value="target">目标国家</n-radio>
            <n-radio value="not_target">非目标国家</n-radio>
          </n-radio-group>
        </n-form-item>
      </section>

      <!-- 卡片 2:玩家信息 -->
      <section class="card">
        <div class="card-title">玩家信息</div>

        <n-form-item label="区服">
          <n-input
            v-model:value="form.region"
            placeholder="如：S019"
          />
        </n-form-item>

        <n-form-item label="玩家游戏名字">
          <n-input
            v-model:value="form.player_game_name"
            placeholder="请输入玩家的游戏昵称"
          />
        </n-form-item>

        <n-form-item label="玩家 ID" path="player_id">
          <n-input
            v-model:value="form.player_id"
            placeholder="请输入玩家ID"
          />
        </n-form-item>
      </section>

      <!-- 卡片 3:补充信息 -->
      <section class="card">
        <div class="card-title">补充信息</div>

        <n-form-item label="DC社群">
          <n-input
            v-model:value="form.community"
            placeholder="如：Top heroes"
          />
        </n-form-item>

        <n-form-item label="token 码">
          <n-input
            v-model:value="form.token_code"
            type="password"
            show-password-on="click"
            placeholder="非 discord 渠道可不填(选填)"
          />
        </n-form-item>

        <n-form-item label="备注">
          <n-input
            v-model:value="form.remark"
            type="textarea"
            :rows="2"
            placeholder="前端沟通细节、玩家信息等"
          />
        </n-form-item>
      </section>

      <!-- 卡片 4:上传截图 -->
      <section class="card">
        <div class="card-title">上传截图</div>
        <div class="upload-area" tabindex="0" ref="uploadAreaRef">
          <n-upload
            v-model:file-list="fileList"
            :action="uploadUrl"
            :headers="uploadHeaders"
            list-type="image-card"
            :max="9"
            accept="image/*"
            @finish="handleUploadFinish"
            @remove="handleUploadRemove"
          >
            鼠标聚焦后可直接粘贴
          </n-upload>
        </div>
      </section>

      <!-- 卡片 5:动态自定义字段(管理员后台配置后才显示) -->
      <section v-if="extensionFields.length > 0" class="card card-extension">
        <div class="card-title">
          <span>自定义字段</span>
          <n-tag size="tiny" :bordered="false" type="info">{{ extensionFields.length }} 项</n-tag>
        </div>
        <n-form-item
          v-for="f in extensionFields"
          :key="f.id"
          :label="f.label"
          :show-feedback="false"
          :required="f.required"
        >
          <n-input
            v-if="f.type === 'text'"
            v-model:value="extValues[f.code]"
            :placeholder="f.placeholder || `请输入${f.label}`"
          />
          <n-input
            v-else-if="f.type === 'textarea'"
            v-model:value="extValues[f.code]"
            type="textarea"
            :rows="2"
            :placeholder="f.placeholder || `请输入${f.label}`"
          />
          <n-input-number
            v-else-if="f.type === 'number'"
            v-model:value="extValues[f.code]"
            :placeholder="f.placeholder"
            style="width: 100%"
          />
          <n-date-picker
            v-else-if="f.type === 'date'"
            v-model:value="extValues[f.code]"
            type="date"
            style="width: 100%"
          />
          <n-select
            v-else-if="f.type === 'select'"
            v-model:value="extValues[f.code]"
            :options="getSelectOptions(f)"
            :placeholder="f.placeholder || `请选择${f.label}`"
          />
        </n-form-item>
      </section>
    </n-form>

    <!-- 浮动底部提交按钮 -->
    <div class="footer">
      <n-button
        type="primary"
        block
        size="large"
        :loading="submitting"
        class="submit-btn"
        @click="handleSubmit"
      >
        提交注册
      </n-button>
    </div>

    <!-- 提交流程进度条 -->
    <n-card v-if="steps.length > 0" class="steps-card">
      <n-steps :current="currentStep" :status="stepStatus" size="small">
        <n-step title="上传" />
        <n-step title="提交" />
        <n-step title="广播" />
      </n-steps>
      <div class="step-log">
        <div v-for="(log, idx) in steps" :key="idx">{{ log }}</div>
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onBeforeUnmount } from 'vue'
import {
  NCard, NForm, NFormItem, NInput, NInputNumber, NSelect,
  NUpload, NButton, NSteps, NStep, NRadioGroup, NRadio, NTag,
  NDatePicker, useMessage, type FormRules, type UploadFileInfo
} from 'naive-ui'
import { gameApi, dictApi, leadApi, uploadApi, extensionFieldApi } from '@/api/modules'

const message = useMessage()

const gameOptions = ref<{label: string, value: number}[]>([])
const channelOptions = ref<{label: string, value: string}[]>([])
const extensionFields = ref<any[]>([])
const extValues = reactive<Record<string, any>>({})

const formRef = ref<any>(null)
const submitting = ref(false)
const fileList = ref<UploadFileInfo[]>([])
const uploadedPaths = ref<string[]>([])

const form = reactive({
  game_id: null as number | null,
  channel: '',
  player_username: '',
  player_game_name: '',
  player_id: '',
  region: '',
  player_country: '',
  player_age: null as number | null,
  is_target: 'target',
  community: '',
  token_code: '',
  remark: ''
})

const rules: FormRules = {
  game_id: { required: true, type: 'number', message: '请选择游戏', trigger: 'change' },
  channel: { required: true, message: '请选择社交平台', trigger: 'change' },
  player_username: { required: true, message: '请输入玩家用户名', trigger: 'blur' },
  player_id: { required: true, message: '请输入玩家 ID', trigger: 'blur' },
  is_target: { required: true, message: '请选择是否目标', trigger: 'change' }
  // token_code 不再必填(选填,前端提示"非 discord 渠道可不填")
}

const uploadUrl = '/api/v1/upload'
const uploadHeaders = () => ({
  Authorization: `Bearer ${localStorage.getItem('access_token') || ''}`
})

const steps = ref<string[]>([])
const currentStep = ref(0)
const stepStatus = ref<'process' | 'finish' | 'error'>('process')

async function loadOptions() {
  const games: any = await gameApi.list()
  gameOptions.value = (games.data.list || []).map((g: any) => ({ label: `${g.code} - ${g.name}`, value: g.id }))

  const resp: any = await dictApi.list({ category: 'channel' })
  channelOptions.value = (resp.data || []).map((d: any) => ({ label: d.label, value: d.code }))

  try {
    const extResp: any = await extensionFieldApi.list()
    extensionFields.value = extResp.data || []
  } catch {
    extensionFields.value = []
  }
}

function getSelectOptions(field: any) {
  const opts = field.options
  if (!opts) return []
  // 后端已将字典选项格式化为 [{label, value}, ...]
  if (Array.isArray(opts)) {
    return opts.map((o: any) => ({ label: o.label || o.value, value: o.value }))
  }
  // 兼容旧的 options JSON 写法
  return Object.entries(opts).map(([k, v]) => ({ label: String(v), value: k }))
}

function handleUploadFinish({ file, event }: any) {
  const resp = JSON.parse(event.target.responseText)
  if (resp.code === 0) {
    file.url = resp.data.files[0].url
    file.path = resp.data.files[0].path
    uploadedPaths.value.push(resp.data.files[0].path)
    message.success(`截图 ${file.name} 上传成功`)
  } else {
    message.error('上传失败：' + resp.message)
  }
}

function handleUploadRemove({ file }: any) {
  if (file.path) {
    uploadedPaths.value = uploadedPaths.value.filter(p => p !== file.path)
  }
}

async function handleSubmit() {
  if (!formRef.value) return
  try { await formRef.value.validate() } catch { return }

  submitting.value = true
  steps.value = []
  currentStep.value = 0
  stepStatus.value = 'process'

  try {
    if (fileList.value.length > 0 && uploadedPaths.value.length === 0) {
      steps.value.push(`[1] 上传 ${fileList.value.length} 张截图...`)
      for (const f of fileList.value) {
        if (!f.file) continue
        await uploadApi.upload(f.file as File)
      }
      steps.value.push(`[1] ✓ 上传完成`)
      currentStep.value = 1
    }

    const extension: Record<string, any> = {}
    for (const f of extensionFields.value) {
      const v = extValues[f.code]
      if (v !== undefined && v !== null && v !== '') {
        extension[f.code] = v
      }
    }

    steps.value.push('[2] 提交注册...')
    const payload = {
      ...form,
      game_id: Number(form.game_id),
      player_age: form.player_age ? Number(form.player_age) : 0,
      screenshots: uploadedPaths.value,
      extension
    }
    const resp: any = await leadApi.create(payload)
    steps.value.push(`[2] ✓ 已创建(ID: ${resp.data.lead.id})`)
    currentStep.value = 2

    if (resp.data.rendered_message) {
      steps.value.push(`[3] 广播:${resp.data.rendered_message}`)
    }
    steps.value.push('[3] ✓ 已推送给后端部门')
    stepStatus.value = 'finish'
    currentStep.value = 3

    message.success('注册提交成功!')
    setTimeout(() => resetForm(), 1500)
  } catch (err: any) {
    stepStatus.value = 'error'
    steps.value.push(`[X] ✗ ${err?.message || '提交失败'}`)
    message.error(err?.message || '提交失败')
  } finally {
    submitting.value = false
  }
}

function resetForm() {
  formRef.value?.restoreValidation()
  Object.assign(form, {
    game_id: null, channel: '', player_username: '', player_game_name: '',
    player_id: '', region: '', player_country: '', player_age: null,
    is_target: 'target', community: '', token_code: '', remark: ''
  })
  fileList.value = []
  uploadedPaths.value = []
  for (const k of Object.keys(extValues)) {
    delete extValues[k]
  }
  steps.value = []
  currentStep.value = 0
}

// --------- 粘贴图片支持 ---------
const uploadAreaRef = ref<HTMLElement | null>(null)

function handlePaste(ev: ClipboardEvent) {
  if (!ev.clipboardData) return
  if (fileList.value.length >= 9) {
    message.warning('最多 9 张截图')
    return
  }
  const items = Array.from(ev.clipboardData.items || [])
  const imageItems = items.filter(it => it.kind === 'file' && it.type.startsWith('image/'))
  if (imageItems.length === 0) return

  ev.preventDefault()
  for (const item of imageItems) {
    const blob = item.getAsFile()
    if (!blob) continue
    const fileName = (blob as any).name || `pasted-${Date.now()}.png`
    const file = new File([blob], fileName, { type: blob.type })
    fileList.value.push({
      id: `paste-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
      name: fileName,
      status: 'pending',
      file: file as any,
      url: ''
    } as UploadFileInfo)
    uploadApi.upload(file).then((resp: any) => {
      const url = resp.data.files[0].url
      const path = resp.data.files[0].path
      const last = fileList.value[fileList.value.length - 1]
      if (last) {
        last.url = url
        last.status = 'finished'
      }
      uploadedPaths.value.push(path)
      message.success(`粘贴图片 ${fileName} 上传成功`)
    }).catch((err: any) => {
      message.error('粘贴上传失败:' + (err?.message || ''))
      fileList.value.pop()
    })
  }
}

onMounted(() => {
  loadOptions()
  window.addEventListener('paste', handlePaste)
})

onBeforeUnmount(() => {
  window.removeEventListener('paste', handlePaste)
})
</script>

<style scoped>
.page {
  /* 留底部空间给浮动提交按钮 */
  padding: 0 0 80px 0;
  /* 自适应:窗口被拉宽时,内容保持 480 宽居中,留白扩两侧 */
  max-width: 540px;
  margin: 0 auto;
  width: 100%;
  box-sizing: border-box;
  padding-left: 12px;
  padding-right: 12px;
}

/* 顶部标题栏 */
.page-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  margin: 0 0 12px 0;
  padding: 0 4px;
}
.page-title {
  font-size: 17px;
  font-weight: 700;
  color: #1f2937;
  letter-spacing: 0.3px;
}
.page-sub {
  font-size: 11px;
  color: #9ca3af;
  margin-top: 2px;
}

/* 卡片化分节 */
.card {
  background: #ffffff;
  border-radius: var(--r-lg, 14px);
  padding: 14px 14px 12px;
  margin: 0 0 12px 0;
  box-shadow: 0 1px 0 rgba(37, 99, 235, 0.04), 0 2px 8px rgba(37, 99, 235, 0.06);
  transition: box-shadow 200ms ease, transform 200ms ease;
}
.card:hover {
  box-shadow: 0 4px 14px rgba(37, 99, 235, 0.12);
  transform: translateY(-1px);
}
.card-title {
  font-size: 13px;
  font-weight: 700;
  color: #1f2937;
  margin-bottom: 10px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.card-warning {
  border-left: 3px solid #f59e0b;
}
.card-extension {
  border-left: 3px solid #FFFFFF; /* 自定义字段卡片蓝色标识 */
}

/* 顶部 label 模式:label 在输入框上方 */
:deep(.n-form-item .n-form-item-label) {
  font-size: 12px;
  color: #6b7280;
  font-weight: 600;
  padding: 0 0 4px 0 !important;
  height: auto !important;
  line-height: 1.4 !important;
}
:deep(.n-form-item) {
  margin-bottom: 10px;
}

/* 两列行(区服/国家) */
.row-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

/* 截图区 */
.upload-area {
  outline: none;
}
.upload-area:focus-within {
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.12); /* 蓝色聚焦环 */
  border-radius: 8px;
}
:deep(.n-upload .n-upload-dragger) {
  border-radius: 10px;
  padding: 14px 8px;
}

/* 单选 chips 化 */
:deep(.n-radio) {
  margin-right: 18px;
}

/* 浮动底部按钮 */
.footer {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 10px 14px calc(10px + env(safe-area-inset-bottom));
  background: linear-gradient(180deg, rgba(245, 247, 246, 0) 0%, #f5f7f6 35%);
}
.submit-btn {
  border-radius: var(--r-md, 10px);
  font-weight: 600;
  letter-spacing: 1px;
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.25); /* 蓝色阴影 */
}

/* 步骤日志 */
.steps-card {
  margin-top: 16px;
  border-radius: var(--r-lg, 14px);
  box-shadow: var(--shadow-sm, 0 1px 0 rgba(37,99,235,0.04), 0 2px 8px rgba(37,99,235,0.06));
}
.step-log {
  margin-top: 12px;
  font-size: 12px;
  color: #6b7280;
  line-height: 1.6;
}
</style>
