import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export interface WSMessage {
  type: string
  channel: string
  event: string
  data: any
  time: string
}

// ============ P12 Tauri 环境检测 ============
// Tauri 2 在 webview 里挂 __TAURI_INTERNALS__,在普通浏览器里没有
function isTauri(): boolean {
  return typeof window !== 'undefined' && !!(window as any).__TAURI_INTERNALS__
}

// Tauri 系统通知触发:任务栏闪 + 系统 toast
// 仅 order.paid / order.high_value / player.react 走(已在 pushRealtimeNotification 里分流)
let flashStopTimer: number | null = null
async function fireTauriSystemNotification(n: RealtimeNotification) {
  if (!isTauri()) return
  const tauri = (window as any).__TAURI_INTERNALS__
  const invoke = (cmd: string, args: any) => tauri.invoke(cmd, args)

  try {
    // 1. 任务栏闪(直到窗口获焦;FlashWindowEx uCount=0 + FLASHW_TIMERNOFG)
    await invoke('flash_window', { on: true })
    // 10 秒后强制停止(防止窗口不在前台时永久闪)
    if (flashStopTimer) clearTimeout(flashStopTimer)
    flashStopTimer = window.setTimeout(() => {
      invoke('flash_window', { on: false }).catch(() => {})
    }, 10000)

    // 2. 系统 toast(Win10/11 Action Center)
    await invoke('plugin:notification|notify', {
      options: {
        title: n.title,
        body: n.body,
      }
    })
  } catch (err) {
    console.warn('[tauri notify] failed:', err)
  }
}

// ============ P12 实时通知项(铃铛面板用) ============
export interface RealtimeNotification {
  id: string                  // 唯一 key(由 time + event + order_id 拼)
  event: string               // order.paid / order.high_value / player.react
  channel: string             // follow:realtime
  title: string               // 简略标题(供下拉面板列表)
  body: string                // 详细描述
  link?: string               // 可选跳转(详情 / 订单 / 玩家)
  amount?: number
  high_value?: boolean
  lead_id?: number
  order_no?: string
  player_id?: string
  claimed_by?: number
  claimed_by_name?: string
  game_code?: string
  read: boolean              // 已读标记
  received_at: string         // RFC3339
  source?: string             // 仅 player.react(api/webhook/manual)
  was_lost?: boolean          // 仅 player.react
}

const STORAGE_KEY = 'realtime_notifications_v1'
const MAX_NOTIFICATIONS = 20

export const useWSStore = defineStore('ws', () => {
  const connected = ref(false)
  const connecting = ref(false)
  const lastMessage = ref<WSMessage | null>(null)
  const messageLog = ref<WSMessage[]>([])
  // P12 通知
  const notifications = ref<RealtimeNotification[]>([])
  const unreadCount = ref(0)

  let ws: WebSocket | null = null
  let reconnectTimer: number | null = null

  // ============ 持久化(刷新后保留最近未读) ============
  function loadPersisted() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      if (raw) {
        const arr = JSON.parse(raw) as RealtimeNotification[]
        if (Array.isArray(arr) && arr.length > 0) {
          notifications.value = arr
          // 重新统计未读(刷新前都视为未读)
          unreadCount.value = arr.filter((n) => !n.read).length
        }
      }
    } catch {
      // 损坏的本地数据忽略
    }
  }

  function persist() {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(notifications.value))
    } catch {
      // 容量满 / 隐私模式 → 静默
    }
  }

  function pushRealtimeNotification(msg: WSMessage) {
    if (msg.channel !== 'follow:realtime') return

    const data = msg.data || {}
    let n: RealtimeNotification | null = null

    if (msg.event === 'order.high_value') {
      n = {
        id: `${msg.time}_${msg.event}_${data.order_id}`,
        event: msg.event,
        channel: msg.channel,
        title: `💰 高额订单 ¥${data.amount}`,
        body: `${data.game_code || ''} 玩家 ${data.player_id} 充值 ¥${data.amount}(阈值 ¥${data.high_value_threshold})` +
              (data.claimed_by_name ? ` · 维护员 ${data.claimed_by_name}` : ' · 暂无人认领'),
        link: data.lead_id ? `/leads/${data.lead_id}` : undefined,
        amount: data.amount,
        high_value: true,
        lead_id: data.lead_id,
        order_no: data.order_no,
        player_id: data.player_id,
        claimed_by: data.claimed_by,
        claimed_by_name: data.claimed_by_name,
        game_code: data.game_code,
        read: false,
        received_at: msg.time,
      }
    } else if (msg.event === 'order.paid') {
      n = {
        id: `${msg.time}_${msg.event}_${data.order_id}`,
        event: msg.event,
        channel: msg.channel,
        title: `💳 玩家充值 ¥${data.amount}`,
        body: `${data.game_code || ''} 玩家 ${data.player_id} 支付 ¥${data.amount}` +
              (data.claimed_by_name ? ` · 维护员 ${data.claimed_by_name}` : ' · 暂无人认领'),
        link: data.lead_id ? `/leads/${data.lead_id}` : undefined,
        amount: data.amount,
        high_value: false,
        lead_id: data.lead_id,
        order_no: data.order_no,
        player_id: data.player_id,
        claimed_by: data.claimed_by,
        claimed_by_name: data.claimed_by_name,
        game_code: data.game_code,
        read: false,
        received_at: msg.time,
      }
    } else if (msg.event === 'player.react') {
      n = {
        id: `${msg.time}_${msg.event}_${data.player_id}_${data.game_id}`,
        event: msg.event,
        channel: msg.channel,
        title: data.was_lost ? `🌱 玩家重新激活` : `📍 玩家活跃更新`,
        body: `${data.game_code || ''} 玩家 ${data.player_id} ${data.was_lost ? '再次上线,已从流失状态恢复' : '最近活跃'}` +
              (data.claimed_by_name ? ` · 维护员 ${data.claimed_by_name}` : ''),
        link: data.lead_id ? `/leads/${data.lead_id}` : undefined,
        lead_id: data.lead_id,
        player_id: data.player_id,
        game_code: data.game_code,
        claimed_by: data.claimed_by,
        claimed_by_name: data.claimed_by_name,
        source: data.source,
        was_lost: data.was_lost,
        read: false,
        received_at: msg.time,
      }
    }

    if (!n) return
    // 截断到 MAX_NOTIFICATIONS
    if (notifications.value.length >= MAX_NOTIFICATIONS) {
      notifications.value = notifications.value.slice(0, MAX_NOTIFICATIONS - 1)
    }
    notifications.value.unshift(n)
    unreadCount.value += 1
    persist()

    // P12 Tauri 系统通知(任务栏闪 + 系统 toast)— 仅 Tauri 环境生效
    fireTauriSystemNotification(n)
  }

  function markAllRead() {
    if (unreadCount.value === 0) return
    notifications.value = notifications.value.map((n) => ({ ...n, read: true }))
    unreadCount.value = 0
    persist()
  }

  function clearAll() {
    notifications.value = []
    unreadCount.value = 0
    persist()
  }

  function connect(token: string) {
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
      return
    }

    connecting.value = true
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = location.host
    const url = `${proto}//${host}/ws?token=${encodeURIComponent(token)}`

    ws = new WebSocket(url)

    ws.onopen = () => {
      connected.value = true
      connecting.value = false
      console.log('[WS] connected')
    }

    ws.onmessage = (e) => {
      try {
        const msg: WSMessage = JSON.parse(e.data)
        lastMessage.value = msg
        messageLog.value.unshift(msg)
        if (messageLog.value.length > 50) messageLog.value.pop()

        // P12 实时通知队列(独立事件)
        if (msg.channel === 'follow:realtime' && msg.event) {
          pushRealtimeNotification(msg)
        }
      } catch (err) {
        console.error('[WS] parse error', err)
      }
    }

    ws.onerror = (e) => {
      console.error('[WS] error', e)
    }

    ws.onclose = () => {
      connected.value = false
      connecting.value = false
      ws = null
      if (reconnectTimer) clearTimeout(reconnectTimer)
      reconnectTimer = window.setTimeout(() => {
        const t = localStorage.getItem('access_token')
        if (t) connect(t)
      }, 5000)
    }
  }

  function disconnect() {
    if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null }
    if (ws) { ws.close(); ws = null }
    connected.value = false
  }

  // 监听通知变化,自动持久化
  watch(notifications, persist, { deep: true })

  // 启动时先从 localStorage 加载
  loadPersisted()

  return {
    connected, connecting, lastMessage, messageLog,
    notifications, unreadCount,
    connect, disconnect,
    markAllRead, clearAll,
    pushRealtimeNotification,
  }
})