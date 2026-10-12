<template>
  <n-drawer
    :show="show"
    :width="720"
    placement="right"
    :on-update:show="(v: boolean) => emit('update:show', v)"
  >
    <n-drawer-content :title="drawerTitle" closable>
      <div v-if="loading" style="padding: 24px; text-align: center;">
        <n-spin />
      </div>
      <div v-else-if="!detail" style="padding: 24px; color: #94a3b8">未加载到玩家数据</div>
      <div v-else>
        <!-- 头部信息 -->
        <div class="player-header">
          <div class="ph-row">
            <span class="ph-label">玩家 ID</span>
            <span class="ph-value mono">{{ detail.summary.player_id }}</span>
          </div>
          <div class="ph-row">
            <span class="ph-label">游戏</span>
            <span class="ph-value">{{ detail.summary.game_code }} - {{ detail.summary.game_name }}</span>
          </div>
          <div class="ph-row">
            <span class="ph-label">玩家名</span>
            <span class="ph-value">{{ detail.summary.latest_username || '—' }}</span>
            <span class="ph-label" style="margin-left: 24px">国家</span>
            <span class="ph-value">{{ detail.summary.latest_country || '—' }}</span>
            <span class="ph-label" style="margin-left: 24px">目标</span>
            <n-tag size="small" :type="detail.summary.latest_is_target === 'target' ? 'success' : 'default'">
              {{ detail.summary.latest_is_target === 'target' ? '是' : '否' }}
            </n-tag>
          </div>
          <div class="kpi-strip">
            <div class="kpi-cell">
              <div class="kpi-num">{{ detail.summary.lead_count }}</div>
              <div class="kpi-cap">线索</div>
            </div>
            <div class="kpi-cell">
              <div class="kpi-num">{{ detail.summary.order_count }}</div>
              <div class="kpi-cap">订单</div>
            </div>
            <div class="kpi-cell">
              <div class="kpi-num" style="color: #10b981">{{ detail.summary.paid_order_count }}</div>
              <div class="kpi-cap">已付订单</div>
            </div>
            <div class="kpi-cell">
              <div class="kpi-num" style="color: #2563eb">{{ Number(detail.summary.total_order_amount).toFixed(2) }}</div>
              <div class="kpi-cap">总金额</div>
            </div>
          </div>
          <div class="ph-row small">
            <span class="ph-label">首次 lead</span>
            <span class="ph-value mono">{{ fmtDateTime(detail.summary.first_lead_at) }}</span>
            <span class="ph-label" style="margin-left: 24px">最后 lead</span>
            <span class="ph-value mono">{{ fmtDateTime(detail.summary.last_lead_at) }}</span>
            <span class="ph-label" style="margin-left: 24px">最后跟进</span>
            <span class="ph-value mono">{{ fmtDateTime(detail.summary.last_follow_at) }}</span>
          </div>
        </div>

        <!-- Tabs -->
        <n-tabs v-model:value="activeTab" type="line" animated style="margin-top: 16px">
          <!-- 时间线 -->
          <n-tab-pane name="timeline" tab="时间线">
            <div class="timeline">
              <div v-for="item in timeline" :key="item.key" class="timeline-item" :class="`type-${item.type}`">
                <div class="ts-dot" :class="`type-${item.type}`"></div>
                <div class="ts-content">
                  <div class="ts-header">
                    <n-tag size="tiny" :type="item.tagType">{{ item.tag }}</n-tag>
                    <span class="ts-time">{{ fmtDateTime(item.time) }}</span>
                  </div>
                  <div class="ts-body">
                    <div v-if="item.type === 'lead'">
                      <div><b>玩家名:</b>{{ item.data.player_username || '—' }} · <b>渠道:</b>{{ item.data.channel }}</div>
                      <div v-if="item.data.remark"><b>备注:</b>{{ item.data.remark }}</div>
                      <div v-if="item.data.is_target === 'target'" style="color: #10b981">✓ 目标玩家</div>
                    </div>
                    <div v-else-if="item.type === 'follow'">
                      <n-tag size="tiny">{{ item.data.follow_type }}</n-tag>
                      <span style="margin-left: 8px"><b>{{ item.data.follower?.real_name || item.data.follower?.username || '系统' }}:</b></span>
                      <span style="margin-left: 4px">{{ item.data.content }}</span>
                    </div>
                    <div v-else-if="item.type === 'order'">
                      <div>
                        <b>订单:</b>{{ item.data.order_no }}
                        · <b>金额:</b>
                        <span :style="`color: ${item.data.status === 'paid' ? '#10b981' : '#64748b'}; font-weight: 600`">
                          {{ Number(item.data.amount).toFixed(2) }} {{ item.data.currency }}
                        </span>
                        <n-tag size="tiny" :type="orderTagType(item.data.status)" style="margin-left: 6px">
                          {{ orderStatusLabel(item.data.status) }}
                        </n-tag>
                      </div>
                      <div v-if="item.data.remark"><b>备注:</b>{{ item.data.remark }}</div>
                    </div>
                  </div>
                </div>
              </div>
              <div v-if="!timeline.length" style="padding: 24px; text-align: center; color: #94a3b8">
                暂无时间线数据
              </div>
            </div>
          </n-tab-pane>

          <!-- 线索 -->
          <n-tab-pane name="leads" tab="线索 ({{ detail.leads.length }})">
            <div v-for="l in detail.leads" :key="l.id" class="lead-card">
              <div class="lead-row">
                <n-tag size="small" :type="leadStatusType(l.status)">{{ leadStatusLabel(l.status) }}</n-tag>
                <span class="lead-id">#{{ l.id }}</span>
                <span style="margin-left: auto; color: #94a3b8; font-size: 12px">{{ fmtDateTime(l.created_at) }}</span>
              </div>
              <div style="margin-top: 6px">
                <b>{{ l.player_username || '—' }}</b> · 渠道 {{ l.channel }} · 区服 {{ l.region || '—' }} · 国家 {{ l.player_country || '—' }}
              </div>
              <div v-if="l.remark" style="margin-top: 4px; color: #475569">{{ l.remark }}</div>
              <div v-if="l.creator" style="margin-top: 6px; font-size: 12px; color: #64748b">
                创建者: {{ l.creator.real_name || l.creator.username }}
                <span v-if="l.claimed_by_user"> · 认领人: {{ l.claimed_by_user.real_name || l.claimed_by_user.username }}</span>
              </div>
            </div>
          </n-tab-pane>

          <!-- 订单 -->
          <n-tab-pane name="orders" tab="订单 ({{ detail.orders.length }})">
            <div v-for="o in detail.orders" :key="o.id" class="order-card">
              <div class="order-row">
                <span class="mono">#{{ o.id }} · {{ o.order_no }}</span>
                <n-tag size="small" :type="orderTagType(o.status)">{{ orderStatusLabel(o.status) }}</n-tag>
                <span class="amount" :style="`color: ${o.status === 'paid' ? '#10b981' : '#64748b'}`">
                  {{ Number(o.amount).toFixed(2) }} {{ o.currency }}
                </span>
                <span style="margin-left: auto; color: #94a3b8; font-size: 12px">{{ fmtDateTime(o.order_time) }}</span>
              </div>
              <div v-if="o.remark" style="margin-top: 4px; color: #475569">{{ o.remark }}</div>
              <div style="margin-top: 6px; font-size: 12px; color: #64748b">
                来源: {{ o.source }}
                <span v-if="o.attribution_status"> · 归属: {{ o.attribution_status }}</span>
              </div>
            </div>
            <div v-if="!detail.orders.length" style="padding: 24px; text-align: center; color: #94a3b8">暂无订单</div>
          </n-tab-pane>

          <!-- 跟进 -->
          <n-tab-pane name="follows" tab="跟进 ({{ detail.follow_records.length }})">
            <div v-for="f in detail.follow_records" :key="f.id" class="follow-card">
              <div class="follow-row">
                <n-tag size="small">{{ f.follow_type }}</n-tag>
                <span class="follow-author">{{ f.follower?.real_name || f.follower?.username || '系统' }}</span>
                <span style="margin-left: auto; color: #94a3b8; font-size: 12px">{{ fmtDateTime(f.created_at) }}</span>
              </div>
              <div style="margin-top: 6px; white-space: pre-wrap">{{ f.content }}</div>
              <div style="margin-top: 4px; font-size: 12px; color: #94a3b8">线索 #{{ f.lead_id }}</div>
            </div>
            <div v-if="!detail.follow_records.length" style="padding: 24px; text-align: center; color: #94a3b8">暂无跟进记录</div>
          </n-tab-pane>
        </n-tabs>
      </div>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { NDrawer, NDrawerContent, NTabs, NTabPane, NTag, NSpin, useMessage } from 'naive-ui'
import { playerApi, type PlayerDetailResponse } from '@/api/modules'

const props = defineProps<{
  show: boolean
  playerId: string
  gameId: number
}>()
const emit = defineEmits<{ (e: 'update:show', v: boolean): void }>()

const detail = ref<PlayerDetailResponse | null>(null)
const loading = ref(false)
const activeTab = ref('timeline')

const message = useMessage()

const drawerTitle = computed(() => {
  if (!detail.value) return '玩家详情'
  return `玩家 ${detail.value.summary.latest_username || detail.value.summary.player_id}`
})

const timeline = computed(() => {
  if (!detail.value) return []
  const items: Array<{ key: string; type: string; tag: string; tagType: 'success' | 'info' | 'warning' | 'error' | 'default'; time: string; data: any }> = []
  for (const l of detail.value.leads) {
    items.push({ key: `lead-${l.id}`, type: 'lead', tag: '新线索', tagType: 'info', time: l.created_at, data: l })
  }
  for (const f of detail.value.follow_records) {
    items.push({ key: `follow-${f.id}`, type: 'follow', tag: '跟进', tagType: 'warning', time: f.created_at, data: f })
  }
  for (const o of detail.value.orders) {
    items.push({ key: `order-${o.id}`, type: 'order', tag: '订单', tagType: o.status === 'paid' ? 'success' : 'default', time: o.order_time, data: o })
  }
  return items.sort((a, b) => (b.time || '').localeCompare(a.time || ''))
})

watch(() => [props.show, props.playerId, props.gameId], async () => {
  if (props.show && props.playerId) {
    loading.value = true
    try {
      const r: any = await playerApi.detail(props.playerId, props.gameId)
      detail.value = r.data || null
    } catch (e: any) {
      message.error('加载玩家详情失败:' + (e?.message || e))
      detail.value = null
    } finally {
      loading.value = false
    }
  }
}, { immediate: true })

function fmtDateTime(s: string | null) {
  if (!s) return '—'
  return new Date(s).toLocaleString('zh-CN', { hour12: false })
}

const leadStatusType = (s: string) => ({ pending: 'info', claimed: 'success', archived: 'default' }[s] || 'default') as any
const leadStatusLabel = (s: string) => ({ pending: '待认领', claimed: '已认领', archived: '已流失' }[s] || s)
const orderTagType = (s: string) => ({ paid: 'success', refunded: 'warning', pending: 'info', cancelled: 'default' }[s] || 'default') as any
const orderStatusLabel = (s: string) => ({ paid: '已支付', refunded: '已退款', pending: '待支付', cancelled: '已取消' }[s] || s)
</script>

<style scoped>
.player-header {
  background: linear-gradient(135deg, #eff6ff, #f8fafc);
  border-radius: 10px;
  padding: 14px 16px;
}
.ph-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 13px;
}
.ph-row.small { font-size: 12px; margin-bottom: 0; color: #475569; }
.ph-label { color: #94a3b8; min-width: 64px; }
.ph-value { color: #0f172a; }
.mono { font-family: 'SF Mono', Consolas, monospace; font-size: 12px; }

.kpi-strip {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
  margin: 12px 0;
}
.kpi-cell {
  background: #fff;
  border-radius: 8px;
  padding: 10px 8px;
  text-align: center;
  box-shadow: var(--shadow-xs);
}
.kpi-num {
  font-size: 20px;
  font-weight: 700;
  color: #0f172a;
}
.kpi-cap {
  font-size: 11px;
  color: #94a3b8;
  margin-top: 2px;
}

/* 时间线 */
.timeline { padding: 8px 0; }
.timeline-item {
  position: relative;
  padding: 0 0 18px 24px;
  border-left: 2px solid #e2e8f0;
  margin-left: 8px;
}
.timeline-item:last-child { border-left-color: transparent; padding-bottom: 0; }
.ts-dot {
  position: absolute;
  left: -7px;
  top: 0;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #cbd5e1;
  border: 2px solid #fff;
  box-shadow: 0 0 0 1px #cbd5e1;
}
.ts-dot.type-lead { background: #3b82f6; box-shadow: 0 0 0 1px #3b82f6; }
.ts-dot.type-follow { background: #f59e0b; box-shadow: 0 0 0 1px #f59e0b; }
.ts-dot.type-order { background: #10b981; box-shadow: 0 0 0 1px #10b981; }
.ts-content { font-size: 13px; line-height: 1.6; }
.ts-header { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.ts-time { color: #94a3b8; font-size: 12px; margin-left: auto; }

/* 卡片 */
.lead-card, .order-card, .follow-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px 12px;
  margin-bottom: 8px;
  font-size: 13px;
}
.lead-row, .order-row, .follow-row {
  display: flex; align-items: center; gap: 8px;
}
.lead-id, .amount { font-family: 'SF Mono', Consolas, monospace; font-size: 12px; }
.amount { font-weight: 700; }
</style>
