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
        <n-tab-pane name="games" tab="游戏列表">
          <GamesPanel />
        </n-tab-pane>
        <n-tab-pane name="upload" tab="上传模板">
          <UploadTemplatesPanel />
        </n-tab-pane>
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NCard, NTabs, NTabPane, NText } from 'naive-ui'
import GamesPanel from './Games.vue'
import UploadTemplatesPanel from './UploadTemplates.vue'

const route = useRoute()
const router = useRouter()

const TAB_MAP: Record<string, string> = {
  games: 'games',
  upload: 'upload'
}

const activeTab = ref<string>((route.query.tab as string) || 'games')

function onTabChange(val: string) {
  router.replace({ query: { ...route.query, tab: val === 'games' ? undefined : val } })
}

onMounted(() => {
  const q = route.query.tab as string
  if (q && TAB_MAP[q]) activeTab.value = q
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