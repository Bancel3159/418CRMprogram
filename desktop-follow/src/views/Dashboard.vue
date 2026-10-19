<template>
  <div>
    <div class="page-header">
      <span style="font-size: 18px; font-weight: 700">我的维护看板</span>
      <n-space align="center">
        <n-select v-model:value="days" :options="dayOptions" style="width: 110px" @update:value="loadOverview" />
        <n-button @click="loadOverview" :loading="loading">刷新</n-button>
      </n-space>
    </div>

    <!-- KPI 4 卡 -->
    <n-grid :cols="4" :x-gap="12" :y-gap="12" responsive="screen" item-responsive>
      <n-gi span="1 m:1 l:1">
        <div class="stat-card info">
          <div class="label">总认领</div>
          <div class="value">{{ kpi.total_claimed }}</div>
          <div class="sub">非流失的玩家</div>
        </div>
      </n-gi>
      <n-gi span="1 m:1 l:1">
        <div class="stat-card warning">
          <div class="label">待跟进</div>
          <div class="value">{{ kpi.active_players }}</div>
          <div class="sub">已认领 + 今天未跟进</div>
        </div>
      </n-gi>
      <n-gi span="1 m:1 l:1">
        <div class="stat-card success">
          <div class="label">已付费</div>
          <div class="value">{{ kpi.paid_players }}</div>
          <div class="sub">转化率 {{ (kpi.conversion_rate * 100).toFixed(1) }}%</div>
        </div>
      </n-gi>
      <n-gi span="1 m:1 l:1">
        <div class="stat-card danger">
          <div class="label">已流失</div>
          <div class="value">{{ kpi.lost_players }}</div>
          <div class="sub">已跟进过 + 归档</div>
        </div>
      </n-gi>
    </n-grid>

    <!-- 今日小卡 -->
    <n-card title="今日" style="margin-top: 16px">
      <n-grid :cols="3" :x-gap="12">
        <n-gi>
          <div class="mini-stat">
            <div class="mini-label">今日跟进次数</div>
            <div class="mini-value">{{ kpi.today_follow_up }}</div>
          </div>
        </n-gi>
        <n-gi>
          <div class="mini-stat">
            <div class="mini-label">近 {{ days }} 天新增认领</div>
            <div class="mini-value">{{ trendClaimedTotal }}</div>
          </div>
        </n-gi>
        <n-gi>
          <div class="mini-stat">
            <div class="mini-label">近 {{ days }} 天跟进总数</div>
            <div class="mini-value">{{ trendFollowUpTotal }}</div>
          </div>
        </n-gi>
      </n-grid>
    </n-card>

    <!-- 趋势 -->
    <n-card title="每日趋势" style="margin-top: 16px">
      <div v-if="trend.length === 0" class="text-muted">暂无数据</div>
      <div v-else style="display: flex; align-items: flex-end; height: 180px; gap: 6px; padding: 8px 0">
        <div
          v-for="(t, idx) in trend"
          :key="idx"
          style="flex: 1; display: flex; flex-direction: column; align-items: center; gap: 4px"
          :title="`${t.date}\n认领: ${t.claimed}\n跟进: ${t.follow_up}`"
        >
          <div style="display: flex; align-items: flex-end; height: 140px; gap: 2px">
            <div :style="{ width: '8px', height: barHeight(t.claimed) + 'px', background: '#3b82f6', borderRadius: '2px 2px 0 0' }" />
            <div :style="{ width: '8px', height: barHeight(t.follow_up) + 'px', background: '#10b981', borderRadius: '2px 2px 0 0' }" />
          </div>
          <div style="font-size: 10px; color: #6b7280">{{ shortDate(t.date) }}</div>
        </div>
      </div>
      <div style="display: flex; gap: 16px; margin-top: 8px; font-size: 12px; color: #6b7280">
        <span><span style="display: inline-block; width: 10px; height: 10px; background: #3b82f6; border-radius: 2px; margin-right: 4px" />认领</span>
        <span><span style="display: inline-block; width: 10px; height: 10px; background: #10b981; border-radius: 2px; margin-right: 4px" />跟进</span>
      </div>
    </n-card>

    <!-- 跳转到我的玩家 / 公开池 -->
    <n-card style="margin-top: 16px">
      <n-space>
        <n-button type="primary" @click="$router.push('/leads')">查看我的玩家</n-button>
        <n-button @click="$router.push('/pool')">公开池</n-button>
        <n-button @click="$router.push('/search')">查询玩家</n-button>
      </n-space>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NGrid, NGi, NCard, NSpace, NButton, NSelect, useMessage } from 'naive-ui'
import { statsApi } from '@/api/modules'

const message = useMessage()
const loading = ref(false)
const days = ref(7)
const kpi = ref<any>({
  total_claimed: 0, today_follow_up: 0, active_players: 0,
  paid_players: 0, lost_players: 0, conversion_rate: 0
})
const trend = ref<any[]>([])

const dayOptions = [
  { label: '近 7 天', value: 7 },
  { label: '近 14 天', value: 14 },
  { label: '近 30 天', value: 30 }
]

const trendClaimedTotal = computed(() => trend.value.reduce((s, t) => s + (t.claimed || 0), 0))
const trendFollowUpTotal = computed(() => trend.value.reduce((s, t) => s + (t.follow_up || 0), 0))

function shortDate(s: string) {
  // 2026-10-03 -> 10-03
  return s.substring(5)
}

function barHeight(v: number) {
  const max = Math.max(1, ...trend.value.map(t => Math.max(t.claimed || 0, t.follow_up || 0)))
  return Math.max(2, Math.round((v / max) * 130))
}

async function loadOverview() {
  loading.value = true
  try {
    const resp: any = await statsApi.followMine({ days: days.value })
    kpi.value = resp.data.kpi
    trend.value = resp.data.trend || []
  } catch (err: any) {
    message.error(err?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadOverview)
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.stat-card {
  background: white;
  border-radius: 10px;
  padding: 14px 16px;
  border-left: 4px solid #94a3b8;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}
.stat-card.info { border-left-color: #3b82f6; }
.stat-card.success { border-left-color: #10b981; }
.stat-card.warning { border-left-color: #f59e0b; }
.stat-card.danger { border-left-color: #ef4444; }
.stat-card .label { font-size: 13px; color: #6b7280; margin-bottom: 6px; }
.stat-card .value { font-size: 26px; font-weight: 700; color: #0f172a; line-height: 1.2; }
.stat-card .sub { font-size: 11px; color: #9ca3af; margin-top: 4px; }
.mini-stat { padding: 8px 12px; background: #f9fafb; border-radius: 8px; }
.mini-label { font-size: 12px; color: #6b7280; }
.mini-value { font-size: 20px; font-weight: 700; color: #0f172a; margin-top: 2px; }
.text-muted { color: #9ca3af; font-size: 13px; padding: 16px; text-align: center; }
</style>
