<template>
  <div>
    <n-grid :cols="3" :x-gap="12" :y-gap="12" style="margin-bottom: 16px">
      <n-gi>
        <n-card size="small">
          <div class="stat-label">服务状态</div>
          <div class="stat-value" style="font-size: 15px">
            <n-tag v-if="health.status === 'ok'" type="success" size="small">● 运行中</n-tag>
            <n-tag v-else type="error" size="small">● 异常</n-tag>
          </div>
          <div class="stat-sub" v-if="health.timestamp">{{ new Date(health.timestamp).toLocaleString('zh-CN', { hour12: false }) }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <div class="stat-label">PostgreSQL</div>
          <div class="stat-value" style="font-size: 15px">
            <n-tag v-if="health.database === 'ok'" type="success" size="small">● 已连接</n-tag>
            <n-tag v-else type="error" size="small">● {{ health.database || '异常' }}</n-tag>
          </div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <div class="stat-label">Redis</div>
          <div class="stat-value" style="font-size: 15px">
            <n-tag v-if="health.redis === 'ok'" type="success" size="small">● 已连接</n-tag>
            <n-tag v-else type="error" size="small">● {{ health.redis || '异常' }}</n-tag>
          </div>
        </n-card>
      </n-gi>
    </n-grid>

    <n-grid :cols="2" :x-gap="16" :y-gap="12">
      <n-gi>
        <n-card title="📦 数据表行数统计" size="small">
          <div v-if="tableStats.length === 0" class="text-muted">加载中…</div>
          <div v-else>
            <div v-for="t in tableStats" :key="t.table" style="display: flex; justify-content: space-between; padding: 5px 0; border-bottom: 1px solid #f5f5f5; font-size: 13px">
              <span>{{ t.table }}</span>
              <strong>{{ Number(t.count).toLocaleString('zh-CN') }}</strong>
            </div>
          </div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card title="⚙️ 运行时配置(follow + realtime)" size="small">
          <n-spin :show="loadingCfg">
            <div style="font-size: 13px; line-height: 2">
              <div v-for="row in cfgRows" :key="row.key" style="display: flex; justify-content: space-between; padding: 3px 0; border-bottom: 1px solid #f5f5f5">
                <span style="color:#666">{{ row.key }}</span>
                <n-tag size="small" :type="row.type">{{ row.value }}</n-tag>
              </div>
            </div>
          </n-spin>
        </n-card>
      </n-gi>
    </n-grid>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { NCard, NGrid, NGi, NTag, NSpin } from 'naive-ui'
import http from '@/api/client'

const props = defineProps<{ active: boolean }>()

const health = ref<any>({})
const tableStats = ref<any[]>([])
const cfgRows = ref<any[]>([])
const loadingCfg = ref(false)

async function loadHealth() {
  try {
    const r: any = await http.get('/health')
    health.value = r.data || {}
  } catch {
    health.value = { status: 'unreachable' }
  }
}

async function loadTableStats() {
  try {
    const r: any = await http.get('/audit/stats')
    // 后端目前不提供表行数,先留空占位;如果后续加接口直接接
    tableStats.value = r.data?.tables || []
  } catch {
    tableStats.value = []
  }
}

async function loadConfig() {
  loadingCfg.value = true
  try {
    const [follow, realtime] = await Promise.all([
      http.get('/system/configs/follow'),
      http.get('/system/configs/realtime')
    ])
    const rows: any[] = []
    const add = (obj: any, prefix: string) => {
      for (const [k, v] of Object.entries(obj || {})) {
        const raw = String(v)
        rows.push({
          key: `${prefix}${k.replace(/^.*\./, '')}`,
          value: raw,
          type: raw === 'false' ? 'error' : raw === 'true' ? 'success' : 'default'
        })
      }
    }
    add((follow as any).data, 'follow.')
    add((realtime as any).data, 'realtime.')
    cfgRows.value = rows
  } catch {
    cfgRows.value = []
  } finally {
    loadingCfg.value = false
  }
}

// active 用 immediate —— 深链直达本页时 props.active 初始就是 true,
// 不加 immediate 会因为「没发生变化」而不触发,导致数据永远是空
watch(() => props.active, (v) => { if (v) { loadHealth(); loadTableStats(); loadConfig() } }, { immediate: true })
</script>

<style scoped>
.stat-label { color: #999; font-size: 12px; margin-bottom: 4px; }
.stat-value { font-weight: 700; color: #333; }
.stat-sub { font-size: 11px; color: #aaa; margin-top: 4px; }
.text-muted { color: #999; text-align: center; padding: 20px 0; }
</style>