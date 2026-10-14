<template>
  <div>
    <div class="page-header">
      <n-space align="center">
        <n-text depth="3" style="font-size: 12px">
          仅 admin 可见 · 日志异步记录,不影响业务主流程
        </n-text>
        <n-button size="small" @click="reloadAll" :loading="loading">刷新</n-button>
      </n-space>
    </div>

    <n-card size="small" :bordered="false">
      <n-tabs v-model:value="activeTab" type="line" animated @update:value="onTabChange">
        <n-tab-pane name="operation" tab="操作日志">
          <OperationLogPanel :active="activeTab === 'operation'" />
        </n-tab-pane>
        <n-tab-pane name="data" tab="数据变更审计">
          <DataChangePanel :active="activeTab === 'data'" />
        </n-tab-pane>
        <n-tab-pane name="sysinfo" tab="系统运行信息">
          <SysInfoPanel :active="activeTab === 'sysinfo'" />
        </n-tab-pane>
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NCard, NTabs, NTabPane, NText, NButton, NSpace, useMessage } from 'naive-ui'
import OperationLogPanel from './OperationLogPanel.vue'
import DataChangePanel from './DataChangePanel.vue'
import SysInfoPanel from './SysInfoPanel.vue'

const route = useRoute()
const router = useRouter()
const message = useMessage()

const activeTab = ref<string>((route.query.tab as string) || 'operation')
const loading = ref(false)

function onTabChange(v: string) {
  activeTab.value = v
  router.replace({ query: { ...route.query, tab: v } })
}

function reloadAll() {
  loading.value = true
  // 切到别的 tab 时 v-if 会重新 mount,trigger 各 panel 的 onMounted;
  // 当前 tab 已经在 active watcher 里处理,这里只需要切一次以强制刷新
  const cur = activeTab.value
  activeTab.value = ''
  setTimeout(() => { activeTab.value = cur }, 50)
  setTimeout(() => { loading.value = false }, 800)
  message.success('已刷新')
}

onMounted(() => {
  const t = route.query.tab as string
  if (t) activeTab.value = t
})

watch(() => route.query.tab, (v) => {
  if (v && v !== activeTab.value) activeTab.value = v as string
})
</script>

<style scoped>
.page-header { display: flex; justify-content: flex-end; align-items: center; margin-bottom: 12px; }
</style>