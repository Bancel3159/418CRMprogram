<template>
  <div>
    <div class="page-header">
      <n-text v-if="activeTab === 'list'" depth="3" style="font-size: 12px">
      </n-text>
      <n-text v-else-if="activeTab === 'pool'" depth="3" style="font-size: 12px">
      </n-text>
    </div>

    <n-card size="small" :bordered="false">
      <n-tabs
        v-model:value="activeTab"
        type="line"
        animated
        @update:value="onTabChange"
      >
        <n-tab-pane name="list" tab="线索列表">
          <ListPanel />
        </n-tab-pane>
        <n-tab-pane name="pool" tab="线索池">
          <PoolPanel />
        </n-tab-pane>
        <n-tab-pane v-if="userStore.isAdmin" name="broadcast" tab="广播模板">
          <BroadcastPanel />
        </n-tab-pane>
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NCard, NTabs, NTabPane, NText } from 'naive-ui'
import { useUserStore } from '@/stores/user'
import ListPanel from './ListPanel.vue'
import PoolPanel from './PoolPanel.vue'
import BroadcastPanel from './BroadcastPanel.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const validTabs = ['list', 'pool', 'broadcast'] as const
type TabName = (typeof validTabs)[number]

function pickInitialTab(): TabName {
  const q = route.query.tab as string | undefined
  if (q === 'pool' || q === 'broadcast') return q
  return 'list'
}

const activeTab = ref<TabName>(pickInitialTab())

function onTabChange(val: TabName) {
  // broadcast 仅 admin 可见；URL 同步时也守住
  const next = { ...route.query, tab: val === 'list' ? undefined : val }
  router.replace({ query: next })
}

watch(() => route.query.tab, (val) => {
  if (!val) {
    activeTab.value = 'list'
    return
  }
  if (val === 'pool' || val === 'broadcast') {
    if (val === 'broadcast' && !userStore.isAdmin) {
      activeTab.value = 'list'
      return
    }
    activeTab.value = val
  }
})

onMounted(() => {
  // 兼容旧 /pool?tab=broadcast 直接访问 admin 才能看到
  if (route.query.tab === 'broadcast' && !userStore.isAdmin) {
    router.replace({ path: '/leads' })
  }
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