<template>
  <div>
    <div class="page-header">
      <n-button @click="$router.back()">← 返回</n-button>
    </div>

    <n-card>
      <n-alert type="info" style="margin-bottom: 16px" :show-icon="false">
        <p><strong>使用流程：</strong></p>
        <ol style="margin: 8px 0; padding-left: 24px">
          <li>先到「<router-link to="/system/upload-templates">上传模板管理</router-link>」按游戏配置字段映射模板</li>
          <li>选择游戏 + 模板 + 上传文件(CSV / Excel)</li>
          <li>点击「预览」可先查看解析结果(不入库)</li>
          <li>点击「导入」直接入库,自动执行归属(命中唯一 lead → 自动绑定,否则待人工)</li>
        </ol>
      </n-alert>

      <n-form label-placement="left" label-width="100">
        <n-form-item label="游戏" required>
          <n-select v-model:value="gameId" :options="gameOptions" placeholder="选择游戏" filterable @update:value="onGameChange" />
        </n-form-item>
        <n-form-item label="上传模板" required>
          <n-select v-model:value="templateId" :options="templateOptions" placeholder="选择该游戏的模板" :disabled="!gameId" />
        </n-form-item>
        <n-form-item label="文件" required>
          <n-upload
            :default-upload="false"
            :max="1"
            accept=".csv,.xlsx,.xls"
            @change="handleFileChange"
            v-model:file-list="fileList"
          >
            <n-button>选择 CSV / Excel 文件</n-button>
          </n-upload>
          <div style="margin-top: 4px; color: #999; font-size: 12px">支持 .csv / .xlsx / .xls</div>
        </n-form-item>
        <n-form-item>
          <n-space>
            <n-button :disabled="!canPreview" :loading="previewing" @click="doPreview">📋 仅预览</n-button>
            <n-button type="primary" :disabled="!canImport" :loading="importing" @click="doImport">📤 导入入库</n-button>
          </n-space>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- 预览结果 -->
    <n-card v-if="previewResult" title="预览结果" style="margin-top: 16px">
      <n-alert :type="previewResult.errors?.length ? 'warning' : 'success'" style="margin-bottom: 12px">
        共解析 <strong>{{ previewResult.total_rows }}</strong> 行有效数据
        <span v-if="previewResult.errors?.length">,{{ previewResult.errors.length }} 行错误</span>
      </n-alert>

      <n-data-table
        v-if="previewResult.preview?.length"
        :columns="previewColumns"
        :data="previewResult.preview"
        :bordered="false"
        size="small"
        :pagination="false"
      />

      <n-divider v-if="previewResult.errors?.length">解析错误</n-divider>
      <n-ul v-if="previewResult.errors?.length">
        <n-li v-for="(err, i) in previewResult.errors" :key="i">
          第 {{ err.row_number }} 行 - {{ err.field }}: {{ err.message }}
        </n-li>
      </n-ul>
    </n-card>

    <!-- 导入结果 -->
    <n-card v-if="importResult" title="导入结果" style="margin-top: 16px">
      <n-alert :type="importResult.summary.failed > 0 ? 'warning' : 'success'">
        <p>批次号: <code>{{ importResult.batch }}</code></p>
        <p>总行数: <strong>{{ importResult.summary.total }}</strong></p>
        <p>新增: {{ importResult.summary.inserted }} | 跳过(已存在): {{ importResult.summary.skipped }} | 失败: {{ importResult.summary.failed }}</p>
        <p>自动归属: {{ importResult.summary.auto_attributed }} | 待人工绑定: {{ importResult.summary.pending_attribution }}</p>
      </n-alert>

      <n-divider v-if="importResult.row_errors?.length">错误明细</n-divider>
      <n-ul v-if="importResult.row_errors?.length">
        <n-li v-for="(err, i) in importResult.row_errors" :key="i">
          第 {{ err.row_number }} 行 - {{ err.field }}: {{ err.message }}
        </n-li>
      </n-ul>

      <n-button style="margin-top: 12px" @click="$router.push('/orders')">返回订单列表</n-button>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { NCard, NForm, NFormItem, NSelect, NUpload, NButton, NSpace, NAlert, NDivider, NDataTable, NUl, NLi, useMessage, type DataTableColumns, type UploadFileInfo } from 'naive-ui'
import { orderApi, gameApi, uploadTemplateApi, type Game, type UploadTemplate } from '@/api/modules'

const message = useMessage()
const gameId = ref<number | null>(null)
const templateId = ref<number | null>(null)
const gameOptions = ref<{ label: string; value: number }[]>([])
const templateOptions = ref<{ label: string; value: number }[]>([])
const templates = ref<UploadTemplate[]>([])
const fileList = ref<UploadFileInfo[]>([])
const previewing = ref(false)
const importing = ref(false)
const previewResult = ref<any>(null)
const importResult = ref<any>(null)

const currentFile = computed<File | null>(() => {
  const f = fileList.value[0]
  if (!f || !f.file) return null
  return f.file
})

const canPreview = computed(() => gameId.value && templateId.value && currentFile.value)
const canImport = computed(() => canPreview.value)

const previewColumns: DataTableColumns<any> = [
  { title: '订单号', key: 'OrderNo' },
  { title: '玩家 ID', key: 'PlayerID' },
  { title: '金额', key: 'Amount' },
  { title: '状态', key: 'Status' },
  { title: '下单时间', key: 'OrderTime', render: (r) => new Date(r.OrderTime).toLocaleString('zh-CN') }
]

async function loadGames() {
  const resp: any = await gameApi.list()
  gameOptions.value = (resp.data || []).map((g: Game) => ({ label: `${g.code} - ${g.name}`, value: g.id }))
}

async function onGameChange() {
  templateId.value = null
  templateOptions.value = []
  if (!gameId.value) return
  const resp: any = await uploadTemplateApi.list({ game_id: gameId.value })
  templates.value = resp.data || []
  templateOptions.value = templates.value.map((t) => ({ label: t.name, value: t.id }))
}

function handleFileChange(_options: { fileList: UploadFileInfo[] }) {
  // 文件已在 fileList 中,无需额外处理
}

async function doPreview() {
  if (!canPreview.value) return
  previewing.value = true
  try {
    const resp: any = await orderApi.importPreview(gameId.value!, templateId.value!, currentFile.value!)
    previewResult.value = resp.data
    message.success(`解析完成,${resp.data?.total_rows || 0} 行`)
  } catch (e: any) {
    message.error(e?.message || '预览失败')
  } finally {
    previewing.value = false
  }
}

async function doImport() {
  if (!canImport.value) return
  if (!confirm('确认导入?已存在的订单号将被跳过,新订单将自动归属。')) return
  importing.value = true
  try {
    const resp: any = await orderApi.import(gameId.value!, templateId.value!, currentFile.value!)
    importResult.value = resp.data
    if (resp.data?.summary?.inserted > 0) {
      message.success(`成功导入 ${resp.data.summary.inserted} 单`)
    } else {
      message.warning('未导入任何订单')
    }
  } catch (e: any) {
    message.error(e?.message || '导入失败')
  } finally {
    importing.value = false
  }
}

onMounted(loadGames)
</script>

<style scoped>
.page-header { display: flex; justify-content: flex-end; align-items: center; margin-bottom: 16px; }
</style>