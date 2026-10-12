<template>
  <div class="page">
    <div class="page-head">
      <div>
        <div class="page-title">注册数据</div>
        <div class="page-sub" v-if="canPickScope">
        </div>
        <div class="page-sub" v-else>
        </div>
      </div>
      <n-space :size="6">
        <!-- 视角筛选(仅 admin / 主管 / 组长 可见) -->
        <n-radio-group
          v-if="canPickScope"
          :value="scope"
          @update:value="onScopeChange"
          size="small"
          name="scope"
        >
          <n-radio-button value="self">本人</n-radio-button>
          <n-radio-button value="all">团队</n-radio-button>
          <n-radio-button value="specific">员工</n-radio-button>
        </n-radio-group>
        <n-select
          v-if="scope === 'specific' && canPickScope"
          :value="specificUserId"
          @update:value="onSpecificChange"
          :options="memberOptions"
          size="small"
          placeholder="选择员工"
          style="width: 140px"
          filterable
        />
        <n-button size="small" quaternary @click="reload" :loading="loading">刷新</n-button>
        <n-button size="small" quaternary @click="goBack">返回</n-button>
      </n-space>
    </div>

    <!-- 视角标签:谁的数据 -->
    <div v-if="data.target" class="scope-banner" :class="`banner-${data.target.scope}`">
      <span class="scope-icon">
        {{ data.target.scope === 'self' ? '📌' : data.target.scope === 'all' ? '📊' : '👤' }}
      </span>
      <span class="scope-text">
        <b>当前查看:</b>
        <span class="scope-label">{{ data.target.scope_label }}</span>
        <span v-if="data.target.scope === 'all' && team_size > 0" class="scope-team-size">
          &nbsp;· 共 {{ team_size }} 人
        </span>
      </span>
      <span v-if="data.user" class="scope-user">
        {{ formatRoleLabel(data.user.role) }}
        <small v-if="data.user.dept_name">· {{ data.user.dept_name }}</small>
      </span>
    </div>

    <!-- 5 张数字卡 -->
    <div class="kpi-grid">
      <div class="kpi">
        <div class="kpi-label">今日</div>
        <div class="kpi-value">{{ numOrDash(data.today) }}</div>
        <div class="kpi-sub">注册数量</div>
      </div>
      <div class="kpi kpi-blue">
        <div class="kpi-label">昨日</div>
        <div class="kpi-value">{{ numOrDash(data.yesterday) }}</div>
        <div class="kpi-sub">注册数量</div>
      </div>
      <div class="kpi">
        <div class="kpi-label">最近 7 日</div>
        <div class="kpi-value">{{ numOrDash(data.last_7_days) }}</div>
        <div class="kpi-sub">注册数量</div>
      </div>
      <div class="kpi">
        <div class="kpi-label">最近 30 日</div>
        <div class="kpi-value">{{ numOrDash(data.last_30_days) }}</div>
        <div class="kpi-sub">注册数量</div>
      </div>
      <div class="kpi kpi-blue">
        <div class="kpi-label">总注册数量</div>
        <div class="kpi-value">{{ numOrDash(data.total) }}</div>
        <div class="kpi-sub">历史累计</div>
      </div>
    </div>

    <!-- 排名 + 漏斗 -->
    <div class="card-row">
      <div class="card card-rank">
        <div class="card-title">排名 · 最近 30 日前端注册</div>
        <div v-if="data.rank" class="rank-body">
          <div class="rank-position">
            <span class="rank-num">#{{ data.rank.position || '-' }}</span>
            <span class="rank-of">/ {{ data.rank.total_a_staff }} 人</span>
          </div>
          <div class="rank-meta">
            本期你的注册 <b>{{ data.rank.lead_count }}</b> 条 ·
            A 部合计 <b>{{ data.rank.total }}</b> 条
          </div>
          <!-- 排名进度条 -->
          <div v-if="data.rank.total_a_staff > 0" class="rank-bar">
            <div
              class="rank-bar-fill"
              :style="{ width: rankPercent + '%' }"
            ></div>
          </div>
          <div class="rank-hint">
            位列前 {{ rankTopPercent }}% 的前端
          </div>
        </div>
      </div>

      <!-- 漏斗:已对接 → 已付费 → 已流失 -->
      <div class="card card-status">
        <div class="card-title">注册转化</div>
        <div class="status-row">
          <div class="status-box s-engaged">
            <div class="status-value">{{ data.by_status.engaged }}</div>
            <div class="status-label">已对接</div>
          </div>
          <div class="status-arrow">→</div>
          <div class="status-box s-paid">
            <div class="status-value">{{ data.by_status.paid }}</div>
            <div class="status-label">已付费</div>
          </div>
          <div class="status-arrow">→</div>
          <div class="status-box s-archived">
            <div class="status-value">{{ data.by_status.archived }}</div>
            <div class="status-label">已流失</div>
          </div>
        </div>

        <!-- 转化率 + 付费来源拆分 -->
        <div class="conv">
          <span class="conv-label">付费率</span>
          <span class="conv-value">{{ conversionRate }}%</span>
        </div>
        <div class="paid-breakup" v-if="paidTotal > 0">
          <small class="muted">其中:</small>
          <small>
            <span class="dot dot-order"></span>订单系统识别
            <b>{{ data.paid_breakup.by_order + data.paid_breakup.both }}</b>
            条
          </small>
          <small>
            <span class="dot dot-mark"></span>人工标记
            <b>{{ data.paid_breakup.by_mark + data.paid_breakup.both }}</b>
            条
          </small>
        </div>
        <p class="lead-tip muted">
          <small>
            「已付费」= 已对接的注册中,订单系统存在该玩家ID的订单会自动标记;
            如发现某个注册漏判,联系管理员修正。
          </small>
        </p>
      </div>
    </div>

    <!-- 趋势图(SVG 自绘线 + 渐变区) -->
    <div class="card">
      <div class="card-title">
        <span>注册趋势 · 最近 {{ TREND_DAYS }} 日</span>
        <span class="card-sub">
          峰值 {{ trendMax }} 条 / 谷值 {{ trendMin }} 条
        </span>
      </div>
      <div class="chart-wrap" v-if="trendPoints.length > 0">
        <svg :viewBox="`0 0 ${svgW} ${svgH}`" preserveAspectRatio="none" class="trend-svg">
          <defs>
            <linearGradient id="trendFill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stop-color="#2563eb" stop-opacity="0.35" />
              <stop offset="100%" stop-color="#2563eb" stop-opacity="0" />
            </linearGradient>
          </defs>
          <line
            v-for="g in 4"
            :key="g"
            :x1="0" :x2="svgW"
            :y1="(g * svgH) / 5" :y2="(g * svgH) / 5"
            stroke="#e2e8f0" stroke-dasharray="2 4"
          />
          <path :d="areaPath" fill="url(#trendFill)" />
          <path :d="linePath" stroke="#2563eb" stroke-width="2" fill="none" />
          <g v-for="(p, i) in trendPoints" :key="`p-${i}`">
            <circle :cx="p.x" :cy="p.y" r="2.5" fill="#2563eb" />
            <title>{{ p.label }} · {{ p.value }} 条</title>
          </g>
        </svg>
      </div>
      <div v-else class="empty">暂无趋势数据,先去提交几条注册吧!</div>
    </div>

    <!-- 渠道 + 国家 分布 -->
    <div class="card-row">
      <div class="card">
        <div class="card-title">渠道分布 · 最近 30 日</div>
        <div v-if="data.by_channel.length > 0" class="bar-list">
          <div v-for="(c, i) in data.by_channel" :key="`ch-${i}-${c.channel}`" class="bar-row">
            <span class="bar-label">{{ c.channel }}</span>
            <div class="bar-track">
              <div class="bar-fill" :style="{ width: c.percent + '%' }"></div>
            </div>
            <span class="bar-num">{{ c.count }} <small>({{ c.percent.toFixed(1) }}%)</small></span>
          </div>
        </div>
        <div v-else class="empty">还没有渠道数据</div>
      </div>

      <div class="card">
        <div class="card-title">国家分布 · Top 10(最近 30 日)</div>
        <div v-if="data.by_country.length > 0" class="country-list">
          <div v-for="(c, i) in data.by_country" :key="`co-${i}-${c.country}`" class="country-row">
            <span class="country-rank">{{ i + 1 }}</span>
            <span class="country-name">{{ c.country }}</span>
            <span class="country-count">{{ c.count }}</span>
            <div class="country-bar">
              <div class="bar-fill" :style="{ width: c.percent + '%', background: '#10b981' }"></div>
            </div>
          </div>
        </div>
        <div v-else class="empty">还没有国家数据</div>
      </div>
    </div>

    <!-- (时段分布 已删除,见 todo.txt) -->
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { dashboardApi } from '@/api/modules'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

// 当前用户的角色:用于判断要不要显示视角筛选
const myRole = computed<string>(() => userStore.userInfo?.role || '')
const canPickScope = computed(() =>
  ['admin', 'a_manager', 'a_lead'].includes(myRole.value)
)

// 提示:趋势图固定显示最近 30 天(滚动窗口)
// 后端的 today / last_30_days / total 等指标本身已是固定窗口,不受此参数影响
const TREND_DAYS = 30

// ----- 视角筛选 state -----
type ScopeKind = 'self' | 'all' | 'specific'
const scope = ref<ScopeKind>('self')
const specificUserId = ref<number | null>(null)
const teamMembers = ref<{ user_id: number; username: string; real_name: string; role: string; lead_count: number }[]>([])

const memberOptions = computed(() =>
  teamMembers.value.map(m => ({
    label: m.real_name || m.username,
    value: m.user_id
  }))
)

function formatRoleLabel(r: string): string {
  return {
    a_staff: '员工',
    a_lead: '组长',
    a_manager: '主管',
    admin: '管理员'
  }[r] || r
}

// 后端响应类型(避免 any → 让 TS 收紧)
interface TrendPoint { date: string; count: number }
interface ChannelPoint { channel: string; count: number; percent: number }
interface CountryPoint { country: string; count: number; percent: number }
interface StatusBlock {
  engaged: number
  paid: number
  archived: number
  pending: number
}
interface PaidBreakup { by_mark: number; by_order: number; both: number }
interface RankBlock {
  position: number
  total_a_staff: number
  lead_count: number
  total: number
}
interface TargetInfo {
  scope: ScopeKind
  scope_label: string
  creator_id?: number
  real_name?: string
  username?: string
}
interface UserBrief {
  id: number
  username: string
  real_name: string
  role: string
  dept_name?: string
}
interface OverviewResp {
  today: number
  yesterday: number
  last_7_days: number
  last_30_days: number
  total: number
  trend: TrendPoint[]
  by_channel: ChannelPoint[]
  by_country: CountryPoint[]
  by_status: StatusBlock
  paid_breakup: PaidBreakup
  conversion_rate: number
  rank: RankBlock | null
  target?: TargetInfo
  user?: UserBrief
  team_size?: number
}

const data = ref<OverviewResp>({
  today: 0, yesterday: 0, last_7_days: 0, last_30_days: 0, total: 0,
  trend: [], by_channel: [], by_country: [],
  by_status: { engaged: 0, paid: 0, archived: 0, pending: 0 },
  paid_breakup: { by_mark: 0, by_order: 0, both: 0 },
  conversion_rate: 0,
  rank: null,
  target: { scope: 'self', scope_label: '本人' },
  team_size: 0
})

// 暴露 team_size 给模板
const team_size = computed(() => data.value.team_size || 0)

const loading = ref(false)

async function loadMembers() {
  if (!canPickScope.value) return
  try {
    const r: any = await dashboardApi.teamMembers()
    teamMembers.value = Array.isArray(r.data) ? r.data : []
    // 默认选第一个非本人的成员作为「某员工」初始值
    if (!specificUserId.value && teamMembers.value.length > 0) {
      const me = userStore.userInfo?.id
      const firstOther = teamMembers.value.find(m => m.user_id !== me)
      if (firstOther) specificUserId.value = firstOther.user_id
    }
  } catch (err: any) {
    console.warn('[Dashboard] loadMembers failed:', err)
  }
}

async function reload() {
  loading.value = true
  try {
    let r: any
    if (scope.value === 'self' || !canPickScope.value) {
      r = await dashboardApi.mine()
    } else if (scope.value === 'all') {
      r = await dashboardApi.team({ scope: 'all' })
    } else {
      // specific
      if (!specificUserId.value) {
        loading.value = false
        return
      }
      r = await dashboardApi.team({ creator_id: specificUserId.value })
    }

    data.value = {
      today: Number(r.today) || 0,
      yesterday: Number(r.yesterday) || 0,
      last_7_days: Number(r.last_7_days) || 0,
      last_30_days: Number(r.last_30_days) || 0,
      total: Number(r.total) || 0,
      trend: Array.isArray(r.trend) ? r.trend : [],
      by_channel: Array.isArray(r.by_channel) ? r.by_channel : [],
      by_country: Array.isArray(r.by_country) ? r.by_country : [],
      by_status: r.by_status || { engaged: 0, paid: 0, archived: 0, pending: 0 },
      paid_breakup: r.paid_breakup || { by_mark: 0, by_order: 0, both: 0 },
      conversion_rate: Number(r.conversion_rate) || 0,
      rank: r.rank || null,
      target: r.target || { scope: scope.value, scope_label: scopeLabel() },
      user: r.user || undefined,
      team_size: Number(r.team_size) || 0
    }
  } catch (err: any) {
    console.warn('[Dashboard] reload failed:', err)
  } finally {
    loading.value = false
  }
}

function scopeLabel(): string {
  if (scope.value === 'self') return '本人'
  if (scope.value === 'all') return '团队'
  if (specificUserId.value) {
    const m = teamMembers.value.find(x => x.user_id === specificUserId.value)
    if (m) return `${m.real_name || m.username}(${m.username})`
  }
  return '员工'
}

function onScopeChange(v: ScopeKind) {
  scope.value = v
  reload()
}

function onSpecificChange(v: number) {
  specificUserId.value = v
  reload()
}

function goBack() {
  router.push('/leads/new')
}

function numOrDash(v: number | null | undefined) {
  if (v === null || v === undefined) return '-'
  return String(v)
}

onMounted(async () => {
  if (!userStore.userInfo) {
    try { await userStore.fetchMe() } catch {}
  }
  if (canPickScope.value) await loadMembers()
  reload()
})

// ===== 计算属性 =====

interface Pt { x: number; y: number; value: number; label: string }

const trendPoints = computed<Pt[]>(() => {
  if (!data.value.trend || data.value.trend.length === 0) return []
  const list: TrendPoint[] = data.value.trend
  const n = list.length
  const max = Math.max(1, ...list.map((x) => Number(x.count) || 0))
  const min = 0
  const padX = 4
  const padY = 12
  const usableW = svgW - padX * 2
  const usableH = svgH - padY * 2

  return list.map((it, i) => {
    const cnt = Number(it.count) || 0
    const x = padX + (i / Math.max(1, n - 1)) * usableW
    const y = padY + (1 - (cnt - min) / Math.max(1, max - min)) * usableH
    return { x, y, value: cnt, label: it.date }
  })
})

const svgW = 600
const svgH = 160

const linePath = computed(() =>
  trendPoints.value
    .map((p: Pt, i: number) =>
      `${i === 0 ? 'M' : 'L'} ${p.x.toFixed(2)} ${p.y.toFixed(2)}`
    )
    .join(' ')
)

const areaPath = computed(() => {
  const pts: Pt[] = trendPoints.value
  if (pts.length === 0) return ''
  const first = pts[0]
  const last = pts[pts.length - 1]
  const body = pts
    .map((p: Pt, i: number) =>
      `${i === 0 ? 'M' : 'L'} ${p.x.toFixed(2)} ${p.y.toFixed(2)}`
    )
    .join(' ')
  return `${body} L ${last.x.toFixed(2)} ${svgH} L ${first.x.toFixed(2)} ${svgH} Z`
})

const trendMax = computed(() => {
  const list = data.value.trend || []
  return Math.max(0, ...list.map((x) => Number(x.count) || 0))
})

const trendMin = computed(() => {
  const list = data.value.trend || []
  const nonempty = list
    .filter((x) => Number(x.count) > 0)
    .map((x) => Number(x.count))
  if (nonempty.length === 0) return 0
  return Math.min(...nonempty)
})

const conversionRate = computed(() => {
  // 后端已经算好,但保留 fallback:本地再算一次
  const cb = data.value.conversion_rate
  if (typeof cb === 'number' && !isNaN(cb)) return cb.toFixed(1)
  const s = data.value.by_status
  if (!s || !s.engaged) return '0.0'
  return ((Number(s.paid) / Number(s.engaged)) * 100).toFixed(1)
})

const paidTotal = computed(() => {
  const pb = data.value.paid_breakup
  if (!pb) return 0
  return (pb.by_mark || 0) + (pb.by_order || 0) + (pb.both || 0)
})

const rankPercent = computed(() => {
  const r = data.value.rank
  if (!r || !r.total_a_staff) return 0
  if (!r.position) return 0
  return Math.round((1 - (r.position - 1) / r.total_a_staff) * 100)
})

const rankTopPercent = computed(() => {
  const r = data.value.rank
  if (!r || !r.total_a_staff || !r.position) return 0
  return Math.max(1, Math.round((r.position / r.total_a_staff) * 100))
})
</script>

<style scoped>
.page {
  padding: 12px 12px 80px;
  max-width: 720px;
  margin: 0 auto;
}

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
  color: #1e3a8a;
  letter-spacing: 0.3px;
}
.page-sub {
  font-size: 11px;
  color: #64748b;
  margin-top: 4px;
}

/* ====== Scope banner ====== */
.scope-banner {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-radius: var(--r-md, 10px);
  background: linear-gradient(135deg, #ffffff, #f0f6ff);
  border: 1px solid rgba(37, 99, 235, 0.18);
  margin: 0 0 12px 0;
  font-size: 12px;
  color: #475569;
  box-shadow: 0 1px 0 rgba(37, 99, 235, 0.05);
}
.scope-banner.banner-all {
  background: linear-gradient(135deg, #f0f6ff, #dbeafe);
  border-color: rgba(37, 99, 235, 0.32);
}
.scope-banner.banner-specific {
  background: linear-gradient(135deg, #ecfeff, #cffafe);
  border-color: rgba(6, 182, 212, 0.32);
}
.scope-icon { font-size: 16px; }
.scope-text { display: flex; align-items: baseline; gap: 4px; flex-wrap: wrap; }
.scope-text b { color: #1e3a8a; font-weight: 700; }
.scope-label {
  font-weight: 700;
  color: #2563eb;
  font-size: 13px;
}
.banner-all .scope-label { color: #1d4ed8; }
.banner-specific .scope-label { color: #0e7490; }
.scope-team-size { color: #64748b; }
.scope-user {
  margin-left: auto;
  padding: 3px 10px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.7);
  color: #1e40af;
  font-weight: 600;
  font-size: 11px;
}
.scope-user small { color: #64748b; font-weight: 400; margin-left: 4px; }

/* ====== KPI ====== */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 8px;
  margin: 0 0 12px 0;
}
.kpi {
  background: #ffffff;
  border-radius: var(--r-md, 12px);
  padding: 10px 8px;
  text-align: center;
  box-shadow: 0 1px 0 rgba(37,99,235,0.04), 0 2px 6px rgba(37,99,235,0.06);
  border-left: 3px solid #cbd5e1;
  transition: box-shadow 200ms ease, transform 200ms ease;
}
.kpi:hover {
  box-shadow: 0 4px 12px rgba(37,99,235,0.12);
  transform: translateY(-1px);
}
.kpi.kpi-blue {
  border-left-color: #2563eb;
}
.kpi-label {
  font-size: 11px;
  color: #64748b;
  font-weight: 600;
}
.kpi-value {
  font-size: 22px;
  font-weight: 800;
  color: #0f172a;
  margin: 4px 0 2px;
  letter-spacing: 0.5px;
}
.kpi-sub {
  font-size: 10px;
  color: #94a3b8;
}

/* ====== Cards ====== */
.card {
  background: #ffffff;
  border-radius: var(--r-lg, 14px);
  padding: 12px 14px 10px;
  margin: 0 0 12px 0;
  box-shadow: 0 1px 0 rgba(37,99,235,0.04), 0 2px 8px rgba(37,99,235,0.06);
  transition: box-shadow 200ms ease, transform 200ms ease;
}
.card:hover {
  box-shadow: 0 4px 14px rgba(37,99,235,0.12);
  transform: translateY(-1px);
}
.card-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin: 0 0 12px 0;
}
.card-rank {
  border-left: 3px solid #2563eb;
}
.card-status {
  border-left: 3px solid #f59e0b;
}
.card-title {
  font-size: 13px;
  font-weight: 700;
  color: #1e3a8a;
  margin-bottom: 8px;
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.card-sub {
  font-size: 11px;
  color: #64748b;
  font-weight: 400;
}

/* ====== Rank ====== */
.rank-body { padding: 4px 0; }
.rank-position { display: flex; align-items: baseline; gap: 4px; }
.rank-num {
  font-size: 30px;
  font-weight: 800;
  color: #2563eb;
  letter-spacing: 1px;
}
.rank-of {
  font-size: 12px;
  color: #64748b;
}
.rank-meta {
  font-size: 11px;
  color: #475569;
  margin: 4px 0 6px;
}
.rank-bar {
  height: 6px;
  border-radius: 3px;
  background: #e2e8f0;
  overflow: hidden;
}
.rank-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #2563eb, #38bdf8);
  transition: width 0.4s ease;
}
.rank-hint {
  font-size: 10.5px;
  color: #64748b;
  margin-top: 4px;
}

/* ====== Funnel 漏斗 ====== */
.status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 4px;
}
.status-box {
  flex: 1;
  text-align: center;
  padding: 8px 4px;
  border-radius: 8px;
  background: #f1f5fb;
}
.s-engaged  { background: #dbeafe; }   /* 已对接:蓝 */
.s-paid     { background: #d1fae5; }   /* 已付费:绿 */
.s-archived { background: #fee2e2; }   /* 已流失:红 */
.status-value {
  font-size: 18px;
  font-weight: 800;
  color: #0f172a;
}
.status-label {
  font-size: 10px;
  color: #64748b;
  margin-top: 2px;
}
.status-arrow {
  font-size: 12px;
  color: #94a3b8;
}

.conv {
  margin-top: 10px;
  display: flex;
  align-items: baseline;
  gap: 6px;
}
.conv-label {
  font-size: 11px;
  color: #64748b;
}
.conv-value {
  font-size: 18px;
  font-weight: 800;
  color: #2563eb;
  letter-spacing: 0.5px;
}

.paid-breakup {
  margin-top: 6px;
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 11px;
  color: #475569;
}
.paid-breakup b {
  color: #0f172a;
  font-weight: 700;
}
.dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 4px;
  vertical-align: 0;
}
.dot-order { background: #2563eb; }
.dot-mark  { background: #10b981; }
.muted { color: #94a3b8; }
.lead-tip {
  margin: 6px 0 0;
  line-height: 1.5;
}

/* ====== Trend chart ====== */
.chart-wrap {
  width: 100%;
  height: 160px;
  background: #f8fafc;
  border-radius: 8px;
  padding: 4px 6px;
}
.trend-svg {
  width: 100%;
  height: 100%;
  display: block;
}

/* ====== Bar list ====== */
.bar-list, .country-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.bar-row {
  display: grid;
  grid-template-columns: 70px 1fr 60px;
  gap: 6px;
  align-items: center;
  font-size: 11px;
  color: #475569;
}
.bar-label, .country-name {
  font-weight: 600;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.bar-track, .country-bar {
  height: 12px;
  border-radius: 6px;
  background: #e2e8f0;
  overflow: hidden;
}
.bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #2563eb, #60a5fa);
  border-radius: 6px;
  transition: width 0.4s ease;
}
.bar-num {
  font-size: 11px;
  color: #475569;
  font-weight: 500;
}
.bar-num small {
  color: #94a3b8;
  font-size: 10px;
}

/* ====== Country list ====== */
.country-row {
  display: grid;
  grid-template-columns: 16px 80px 36px 1fr;
  gap: 6px;
  align-items: center;
  font-size: 11px;
}
.country-rank {
  color: #94a3b8;
  font-weight: 700;
  text-align: right;
}
.country-count {
  font-weight: 700;
  color: #0f172a;
  text-align: right;
}

.empty {
  padding: 20px;
  text-align: center;
  color: #94a3b8;
  font-size: 12px;
}
</style>
