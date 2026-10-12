<template>
  <div>
    <div class="page-header">
      <n-text depth="3" style="font-size: 12px">
      </n-text>
    </div>

    <n-card size="small" :bordered="false" style="margin-bottom: 12px">
      <n-tabs
        v-model:value="activeTab"
        type="line"
        animated
        @update:value="onTabChange"
      >
        <n-tab-pane name="dict" tab="字典管理">
          <DictsPanel />
        </n-tab-pane>
        <n-tab-pane name="extension" tab="注册自定义字段">
          <ExtensionFieldsPanel />
        </n-tab-pane>
        <n-tab-pane name="player-view" tab="玩家列表字段配置">
          <PlayerViewConfig />
        </n-tab-pane>
        <n-tab-pane name="order-view" tab="订单列表字段配置">
          <OrderViewConfig />
        </n-tab-pane>
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NCard, NTabs, NTabPane, NText } from 'naive-ui'
import DictsPanel from './Dicts.vue'
import ExtensionFieldsPanel from './ExtensionFields.vue'
import PlayerViewConfig from './PlayerViewConfig.vue'
import OrderViewConfig from './OrderViewConfig.vue'

const route = useRoute()
const router = useRouter()

const validTabs = ['dict', 'extension', 'player-view', 'order-view'] as const
type TabName = (typeof validTabs)[number]

function pickInitial(): TabName {
  const q = route.query.tab as string | undefined
  if (q === 'extension') return 'extension'
  if (q === 'player-view') return 'player-view'
  if (q === 'order-view') return 'order-view'
  return 'dict'
}

const activeTab = ref<TabName>(pickInitial())

function onTabChange(val: TabName) {
  router.replace({ query: { ...route.query, tab: val === 'dict' ? undefined : val } })
}

onMounted(() => {
  if (route.query.tab) activeTab.value = route.query.tab as TabName
})
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: flex-end;
  align-items: baseline;
  margin-bottom: 16px;
}
</style>