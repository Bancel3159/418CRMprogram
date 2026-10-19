<template>
  <div>
    <div class="page-header">
      <n-space>
        <n-radio-group v-model:value="period" @update:value="loadStats">
          <n-radio-button value="7">近 7 天</n-radio-button>
          <n-radio-button value="30">近 30 天</n-radio-button>
          <n-radio-button value="90">近 90 天</n-radio-button>
        </n-radio-group>
        <n-button @click="loadStats" :loading="loading">刷新</n-button>
      </n-space>
    </div>

    <!-- 总览 4 卡 -->
    <n-grid :cols="4" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <n-gi span="1 m:1 l:1">
        <div class="stat-card">
          <div class="label">拉新总数（{{ periodName }}）</div>
          <div class="value">{{ overview.lead_count || 0 }}</div>
          <div class="trend">今日 +{{ overview.today_count || 0 }} · 本周 +{{ overview.week_count || 0 }}</div>
        </div>
      </n-gi>
      <n-gi span="1 m:1 l:1">
        <div class="stat-card info">
          <div class="label">待认领（公开池）</div>
          <div class="value">{{ overview.pool_count || 0 }}</div>
          <div class="trend">B 部待处理</div>
        </div>
      </n-gi>
      <n-gi span="1 m:1 l:1">
        <div class="stat-card success">
          <div class="label">维护中</div>
          <div class="value">{{ overview.claimed_count || 0 }}</div>
          <div class="trend">B 部活跃玩家</div>
        </div>
      </n-gi>
      <n-gi span="1 m:1 l:1">
        <div class="stat-card warning">
          <div class="label">已流失 / 转化率</div>
          <div class="value">{{ overview.archived_count || 0 }} <small style="font-size: 14px; color: #6b7280">/ {{ overview.conversion_rate?.toFixed(1) || 0 }}%</small></div>
          <div class="trend">归档率 {{ overview.lead_count ? ((overview.archived_count / overview.lead_count) * 100).toFixed(1) : 0 }}%</div>
        </div>
      </n-gi>
    </n-grid>

    <!-- 转化漏斗 -->
    <n-card title=" AB 转化漏斗" style="margin-top: 16px">
      <n-grid :cols="4" :x-gap="16">
        <n-gi>
          <div style="text-align: center">
            <div style="font-size: 32px; font-weight: 700; color: #3b82f6">{{ funnel.created || 0 }}</div>
            <div style="color: #6b7280">拉新总数</div>
          </div>
        </n-gi>
        <n-gi>
          <div style="text-align: center">
            <div style="font-size: 32px; font-weight: 700; color: #10b981">{{ funnel.claimed || 0 }}</div>
            <div style="color: #6b7280">已认领 ({{ funnel.claim_rate?.toFixed(1) || 0 }}%)</div>
          </div>
        </n-gi>
        <n-gi>
          <div style="text-align: center">
            <div style="font-size: 32px; font-weight: 700; color: #f59e0b">{{ funnel.active || 0 }}</div>
            <div style="color: #6b7280">维护中 ({{ funnel.active_rate?.toFixed(1) || 0 }}%)</div>
          </div>
        </n-gi>
        <n-gi>
          <div style="text-align: center">
            <div style="font-size: 32px; font-weight: 700; color: #ef4444">{{ funnel.archived || 0 }}</div>
            <div style="color: #6b7280">已流失 ({{ funnel.loss_rate?.toFixed(1) || 0 }}%)</div>
          </div>
        </n-gi>
      </n-grid>
      <div v-if="funnel.avg_claim_duration_seconds > 0" style="margin-top: 16px; color: #6b7280; font-size: 13px">
         平均认领时长：<strong>{{ formatDuration(funnel.avg_claim_duration_seconds) }}</strong>（样本 {{ funnel.avg_claim_count }} 条）
      </div>
    </n-card>

    <!-- P7：营收 / ARPU / A&B 业绩 -->
    <n-card title="营收指标 (订单数据)" style="margin-top: 16px">
      <n-grid :cols="6" :x-gap="12" :y-gap="12">
        <n-gi span="1 m:1 l:1">
          <div class="mini-stat">
            <div class="mini-label">总营收</div>
            <div class="mini-value" style="color: #18a058">¥{{ formatMoney(orderStats.total_revenue) }}</div>
            <div class="mini-sub">已支付订单</div>
          </div>
        </n-gi>
        <n-gi span="1 m:1 l:1">
          <div class="mini-stat">
            <div class="mini-label">订单数</div>
            <div class="mini-value">{{ orderStats.total_orders || 0 }}</div>
            <div class="mini-sub">已支付</div>
          </div>
        </n-gi>
        <n-gi span="1 m:1 l:1">
          <div class="mini-stat">
            <div class="mini-label">付费玩家</div>
            <div class="mini-value">{{ orderStats.paid_players || 0 }}</div>
            <div class="mini-sub">去重</div>
          </div>
        </n-gi>
        <n-gi span="1 m:1 l:1">
          <div class="mini-stat">
            <div class="mini-label">ARPU</div>
            <div class="mini-value" style="color: #2080f0">¥{{ formatMoney(orderStats.arpu) }}</div>
            <div class="mini-sub">人均营收</div>
          </div>
        </n-gi>
        <n-gi span="1 m:1 l:1">
          <div class="mini-stat success-bg">
            <div class="mini-label">A 队业绩(首单)</div>
            <div class="mini-value" style="color: #18a058">¥{{ formatMoney(orderStats.a_team_revenue) }}</div>
            <div class="mini-sub">{{ orderStats.a_team_orders || 0 }} 笔</div>
          </div>
        </n-gi>
        <n-gi span="1 m:1 l:1">
          <div class="mini-stat info-bg">
            <div class="mini-label">B 队业绩(续单)</div>
            <div class="mini-value" style="color: #2080f0">¥{{ formatMoney(orderStats.b_team_revenue) }}</div>
            <div class="mini-sub">{{ orderStats.b_team_orders || 0 }} 笔</div>
          </div>
        </n-gi>
      </n-grid>
      <div v-if="orderStats.pending_orders > 0" style="margin-top: 12px; padding: 8px 12px; background: #fff7e6; border-radius: 4px; color: #d97706; font-size: 13px">
         当前有 <strong>{{ orderStats.pending_orders }}</strong>个订单待人工绑定 lead
        <router-link to="/orders?attribution_status=pending_attribution" style="margin-left: 12px; color: #d97706">立即处理 </router-link>
      </div>
    </n-card>

    <n-grid :cols="2" :x-gap="16" :y-gap="16" responsive="screen" item-responsive style="margin-top: 16px">
      <!-- 拉新趋势（按天） -->
      <n-gi span="1 m:1 l:1">
        <n-card title="拉新趋势">
          <div v-if="trend.length === 0" class="text-muted">暂无数据</div>
          <div v-else style="display: flex; align-items: flex-end; height: 180px; gap: 4px">
            <div
              v-for="t in trend"
              :key="t.date"
              :title="`${t.date}: ${t.count} 条`"
              style="flex: 1; display: flex; flex-direction: column; align-items: center"
            >
              <div :style="{
                width: '100%',
                background: 'linear-gradient(180deg, #36ad6a 0%, #18a058 100%)',
                height: `${Math.max(8, (t.count / maxTrendCount) * 160)}px`,
                borderRadius: '3px 3px 0 0',
                transition: 'height 0.3s'
              }"></div>
              <div style="font-size: 11px; color: #9ca3af; margin-top: 4px; writing-mode: vertical-rl; transform: rotate(180deg)">
                {{ t.date.slice(5) }}
              </div>
            </div>
          </div>
        </n-card>
      </n-gi>

      <!-- 时段分布 -->
      <n-gi span="1 m:1 l:1">
        <n-card title="拉新时段（按小时）">
          <div v-if="hourly.length === 0" class="text-muted">暂无数据</div>
          <div v-else style="display: flex; align-items: flex-end; height: 180px; gap: 2px">
            <div
              v-for="h in hourly"
              :key="h.hour"
              :title="`${h.hour}:00-${h.hour+1}:00 共 ${h.count} 条`"
              style="flex: 1; display: flex; flex-direction: column; align-items: center"
            >
              <div :style="{
                width: '100%',
                background: '#fbbf24',
                height: `${Math.max(2, (h.count / maxHourlyCount) * 160)}px`,
                borderRadius: '2px 2px 0 0'
              }"></div>
              <div v-if="h.hour % 4 === 0" style="font-size: 10px; color: #9ca3af; margin-top: 4px">{{ h.hour }}时</div>
            </div>
          </div>
        </n-card>
      </n-gi>
    </n-grid>

    <n-grid :cols="3" :x-gap="16" :y-gap="16" responsive="screen" item-responsive style="margin-top: 16px">
      <!-- 按游戏 -->
      <n-gi span="1 m:1 l:1">
        <n-card title="按游戏分布">
          <div v-if="byGame.length === 0" class="text-muted">暂无数据</div>
          <div v-else>
            <div v-for="g in byGame" :key="g.game_id" style="margin-bottom: 12px">
              <div style="display: flex; justify-content: space-between; margin-bottom: 4px">
                <span><strong>{{ g.code }}</strong> {{ g.name }}</span>
                <span>{{ g.count }} ({{ g.percent?.toFixed(1) }}%)</span>
              </div>
              <n-progress type="line" :percentage="g.percent" :show-indicator="false" :height="10" />
            </div>
          </div>
        </n-card>
      </n-gi>

      <!-- 按渠道 -->
      <n-gi span="1 m:1 l:1">
        <n-card title="按渠道分布">
          <div v-if="byChannel.length === 0" class="text-muted">暂无数据</div>
          <div v-else>
            <div v-for="c in byChannel" :key="c.channel" style="margin-bottom: 12px">
              <div style="display: flex; justify-content: space-between; margin-bottom: 4px">
                <span><strong>{{ c.channel }}</strong></span>
                <span>{{ c.count }} ({{ c.percent?.toFixed(1) }}%)</span>
              </div>
              <n-progress type="line" :percentage="c.percent" :show-indicator="false" :height="10" color="#36ad6a" />
            </div>
          </div>
        </n-card>
      </n-gi>

      <!-- 按国家（Top 10） -->
      <n-gi span="1 m:1 l:1">
        <n-card title="按国家（Top 10）">
          <div v-if="byCountry.length === 0" class="text-muted">暂无数据</div>
          <div v-else>
            <div v-for="c in byCountry" :key="c.country" style="margin-bottom: 12px">
              <div style="display: flex; justify-content: space-between; margin-bottom: 4px">
                <span>{{ c.country || '(未填)' }}</span>
                <span>{{ c.count }} ({{ c.percent?.toFixed(1) }}%)</span>
              </div>
              <n-progress type="line" :percentage="c.percent" :show-indicator="false" :height="10" color="#3b82f6" />
            </div>
          </div>
        </n-card>
      </n-gi>
    </n-grid>

    <!-- 业绩排行 -->
    <n-grid :cols="2" :x-gap="16" :y-gap="16" responsive="screen" item-responsive style="margin-top: 16px">
      <n-gi span="1 m:1 l:1">
        <n-card title=" A 部拉新 TOP 10">
          <div v-if="topStaff.length === 0" class="text-muted">暂无数据</div>
          <div v-else>
            <div v-for="(u, idx) in topStaff" :key="u.user_id" style="display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #f3f4f6">
              <div>
                <span style="display: inline-block; width: 24px; color: #6b7280">{{ idx + 1 }}.</span>
                <strong>{{ u.real_name || u.username }}</strong>
                <n-tag size="small" type="info" style="margin-left: 8px">{{ u.dept_name }}</n-tag>
              </div>
              <strong style="color: #18a058">{{ u.lead_count }}</strong>
            </div>
          </div>
        </n-card>
      </n-gi>

      <n-gi span="1 m:1 l:1">
        <n-card title=" B 部维护 TOP 10">
          <div v-if="topMaintainer.length === 0" class="text-muted">暂无数据</div>
          <div v-else>
            <div v-for="(u, idx) in topMaintainer" :key="u.user_id" style="display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #f3f4f6">
              <div>
                <span style="display: inline-block; width: 24px; color: #6b7280">{{ idx + 1 }}.</span>
                <strong>{{ u.real_name || u.username }}</strong>
                <n-tag size="small" type="warning" style="margin-left: 8px">{{ u.dept_name }}</n-tag>
              </div>
              <span>
                <strong style="color: #f59e0b">{{ u.active_count }}</strong>
                <span class="text-muted" style="margin-left: 4px">/ {{ u.total_claimed }}</span>
              </span>
            </div>
          </div>
        </n-card>
      </n-gi>
    </n-grid>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NGrid, NGi, NButton, NSpace, NRadioGroup, NRadioButton, NCard, NProgress, useMessage } from 'naive-ui'
import axios from '@/api/client'

const message = useMessage()
const period = ref('30')
const loading = ref(false)

const overview = ref<any>({})
const funnel = ref<any>({})
const trend = ref<any[]>([])
const hourly = ref<any[]>([])
const byGame = ref<any[]>([])
const byChannel = ref<any[]>([])
const byCountry = ref<any[]>([])
const topStaff = ref<any[]>([])
const topMaintainer = ref<any[]>([])
const orderStats = ref<any>({ total_revenue: 0, total_orders: 0, paid_players: 0, arpu: 0, pending_orders: 0, a_team_revenue: 0, a_team_orders: 0, b_team_revenue: 0, b_team_orders: 0 })

const periodName = computed(() => `近 ${period.value} 天`)

const maxTrendCount = computed(() => Math.max(1, ...trend.value.map(t => t.count)))
const maxHourlyCount = computed(() => Math.max(1, ...hourly.value.map(h => h.count)))

function formatMoney(v: number) {
  if (!v) return '0.00'
  return Number(v).toFixed(2)
}

function formatDuration(seconds: number): string {
  if (!seconds) return '-'
  if (seconds < 60) return `${seconds.toFixed(1)} 秒`
  if (seconds < 3600) return `${(seconds / 60).toFixed(1)} 分钟`
  if (seconds < 86400) return `${(seconds / 3600).toFixed(1)} 小时`
  return `${(seconds / 86400).toFixed(1)} 天`
}

function dateRange(p: string) {
  const end = new Date()
  const start = new Date(end.getTime() - parseInt(p) * 24 * 3600 * 1000)
  return {
    start: start.toISOString().slice(0, 10),
    end: end.toISOString().slice(0, 10)
  }
}

async function loadStats() {
  loading.value = true
  const params = dateRange(period.value)
  const qs = `?start=${params.start}&end=${params.end}`
  try {
    const [o, f, t, h, g, c, cy, ts, tm, ords] = await Promise.all([
      axios.get(`/stats/overview${qs}`),
      axios.get(`/stats/funnel${qs}`),
      axios.get(`/stats/trend${qs}`),
      axios.get(`/stats/hourly${qs}`),
      axios.get(`/stats/by-game${qs}`),
      axios.get(`/stats/by-channel${qs}`),
      axios.get(`/stats/by-country${qs}`),
      axios.get(`/stats/top-staff?role=a_staff&limit=10${qs.includes('start') ? '&' + qs.slice(1) : ''}`),
      axios.get(`/stats/top-maintainer?limit=10`),
      axios.get(`/orders/stats${qs}`)
    ])
    overview.value = (o as any).data
    funnel.value = (f as any).data
    trend.value = (t as any).data || []
    hourly.value = (h as any).data || []
    byGame.value = (g as any).data || []
    byChannel.value = (c as any).data || []
    byCountry.value = (cy as any).data || []
    topStaff.value = (ts as any).data || []
    topMaintainer.value = (tm as any).data || []
    orderStats.value = (ords as any).data?.overview || orderStats.value
  } catch (err: any) {
    message.error(err?.message || '加载看板失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadStats)
</script>

<style scoped>
.stat-card .value small {
  font-weight: 400;
  margin-left: 4px;
}

.mini-stat {
  padding: 12px;
  background: #fafbfc;
  border-radius: 6px;
  text-align: center;
  height: 100%;
}
.mini-stat.success-bg { background: #f6ffed; }
.mini-stat.info-bg { background: #f0f7ff; }
.mini-label { font-size: 12px; color: #6b7280; margin-bottom: 4px; }
.mini-value { font-size: 20px; font-weight: 700; color: #333; }
.mini-sub { font-size: 11px; color: #9ca3af; margin-top: 2px; }
</style>