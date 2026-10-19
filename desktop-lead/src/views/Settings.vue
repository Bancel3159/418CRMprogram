<template>
  <div class="page">
    <div class="page-head">
      <div>
        <div class="page-title">快捷键设置</div>
        <div class="page-sub">用于切换客户端主窗口的显隐;设置后立即生效</div>
      </div>
      <n-button size="small" quaternary @click="goBack">返回</n-button>
    </div>

    <!-- 当前生效快捷键 -->
    <div class="card current">
      <div class="card-title">当前生效快捷键</div>
      <div class="current-row">
        <span class="kbd">{{ formatCombo(currentCombo) }}</span>
        <n-tag :type="isEnabled ? 'success' : 'warning'" size="small" :bordered="false">
          {{ isEnabled ? '已启用' : '已禁用' }}
        </n-tag>
      </div>
    </div>

    <!-- 设置新快捷键 -->
    <div class="card">
      <div class="card-title">设置新快捷键</div>
      <p class="hint">点击下方按钮,然后按下你想要的组合键(如 Ctrl + `、Ctrl + Shift + K 等);必须包含至少 1 个修饰键</p>

      <div
        ref="recorderRef"
        class="recorder"
        :class="{ active: isRecording }"
        tabindex="0"
        @keydown.prevent.stop="onKeyDown"
        @keyup.prevent.stop
        @blur="cancelRecording"
      >
        <span v-if="isRecording" class="recorder-text">
          <span class="recorder-pulse">●</span>
          请按键组合(按 Esc 取消)
        </span>
        <span v-else class="recorder-text">
          {{ capturedCombo ? formatCombo(capturedCombo) : '点击下方按钮开始设置' }}
        </span>
      </div>

      <n-space style="margin-top: 12px">
        <n-button type="primary" @click="startRecording" :disabled="isRecording">
          {{ capturedCombo ? '重新设置' : '开始设置' }}
        </n-button>
        <n-button quaternary @click="disableHotkey" :disabled="!isEnabled">禁用快捷键</n-button>
        <n-button quaternary @click="resetDefault" :disabled="currentCombo === DEFAULT_COMBO">恢复默认</n-button>
      </n-space>

      <n-space justify="end" style="margin-top: 16px">
        <n-button type="primary" @click="onSave" :loading="saving" :disabled="!isDirty || !capturedCombo">
          保存并应用
        </n-button>
      </n-space>
    </div>

    <!-- 使用说明 -->
    <div class="card tip">
      <div class="card-title">使用说明</div>
      <ul>
        <li>支持任意组合键(Ctrl / Shift / Alt / Cmd 之一及以上) + 任意主键。</li>
        <li>主键支持字母、数字、功能键(F1-F12)、方向键、Backquote(<code>`</code>)、Space、Tab、Esc 等。</li>
        <li>保存后立即生效,无需重启。</li>
        <li>如与 QQ、微信、VSCode 等冲突,可点「禁用快捷键」放弃快捷键,改用任务栏图标或开始菜单唤出。</li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { invoke } from '@tauri-apps/api/core'
import { useMessage, NButton, NTag, NSpace } from 'naive-ui'

const router = useRouter()
const message = useMessage()

const STORAGE_KEY = 'lead_hotkey_combo'
const DEFAULT_COMBO = 'Ctrl+`'

// 当前生效(从 localStorage 读)
const currentCombo = ref<string>(DEFAULT_COMBO)
// 设置中的组合
const capturedCombo = ref<string>('')
// 是否正在设置
const isRecording = ref(false)
// 保存中
const saving = ref(false)
const recorderRef = ref<HTMLElement | null>(null)

// 是否变更
const isDirty = computed(() => capturedCombo.value && capturedCombo.value !== currentCombo.value)
const isEnabled = computed(() => currentCombo.value !== 'disabled')

// ============ 键映射 ============
// KeyboardEvent.code  combo token (与 global-hotkey crate 的 parse_key 对齐)
const CODE_TO_TOKEN: Record<string, string> = {
  Backquote: '`',
  Backslash: '\\',
  BracketLeft: '[',
  BracketRight: ']',
  Comma: ',',
  Period: '.',
  Slash: '/',
  Semicolon: ';',
  Quote: "'",
  Minus: '-',
  Equal: '=',
  Space: 'Space',
  Tab: 'Tab',
  Enter: 'Enter',
  NumpadEnter: 'Enter',
  Escape: 'Escape',
  Backspace: 'Backspace',
  Delete: 'Delete',
  Home: 'Home',
  End: 'End',
  PageUp: 'PageUp',
  PageDown: 'PageDown',
  Insert: 'Insert',
  ArrowUp: 'Up',
  ArrowDown: 'Down',
  ArrowLeft: 'Left',
  ArrowRight: 'Right',
  ContextMenu: 'ContextMenu',
  CapsLock: 'CapsLock',
  NumLock: 'NumLock',
  ScrollLock: 'ScrollLock',
  PrintScreen: 'PrintScreen',
  Pause: 'Pause'
}

// 字母 / 数字 / 功能键:动态生成
function codeToToken(code: string): string | null {
  if (CODE_TO_TOKEN[code]) return CODE_TO_TOKEN[code]
  // KeyA - KeyZ
  const letterMatch = code.match(/^Key([A-Z])$/)
  if (letterMatch) return letterMatch[1]
  // Digit0 - Digit9
  const digitMatch = code.match(/^Digit([0-9])$/)
  if (digitMatch) return digitMatch[1]
  // Numpad0 - Numpad9
  const numpadMatch = code.match(/^Numpad([0-9])$/)
  if (numpadMatch) return `Numpad${numpadMatch[1]}`
  // F1 - F24
  const fMatch = code.match(/^F(\d{1,2})$/)
  if (fMatch) return `F${fMatch[1]}`
  return null
}

// 判断是否是修饰键本身(单独按下 Ctrl/Shift 等不应作为快捷键)
const MODIFIER_CODES = new Set(['ControlLeft', 'ControlRight', 'ShiftLeft', 'ShiftRight', 'AltLeft', 'AltRight', 'MetaLeft', 'MetaRight'])

function onKeyDown(ev: KeyboardEvent) {
  if (!isRecording.value) return

  // Esc 取消
  if (ev.code === 'Escape') {
    cancelRecording()
    return
  }

  // 单独的修饰键不构成组合,忽略
  if (MODIFIER_CODES.has(ev.code)) return

  const token = codeToToken(ev.code)
  if (!token) return

  const mods: string[] = []
  if (ev.ctrlKey) mods.push('Ctrl')
  if (ev.altKey) mods.push('Alt')
  if (ev.shiftKey) mods.push('Shift')
  if (ev.metaKey) mods.push('Super')

  // 必须有修饰键
  if (mods.length === 0) {
    message.warning('请加上修饰键(Ctrl / Alt / Shift)')
    return
  }

  const combo = [...mods, token].join('+')
  capturedCombo.value = combo
  isRecording.value = false
  // 设置完成,释放焦点
  recorderRef.value?.blur()
  message.info(`已捕获:${combo} ——点击保存生效`)
}

// 格式化为人类可读: Ctrl + `
function formatCombo(combo: string): string {
  if (!combo || combo === 'disabled') return '未设置'
  return combo.split('+').map(s => s.trim()).join(' + ')
}

// ============ 设置控制 ============
async function startRecording() {
  capturedCombo.value = ''
  isRecording.value = true
  await nextTick()
  recorderRef.value?.focus()
}

function cancelRecording() {
  isRecording.value = false
}

// ============ 保存 / 禁用 / 重置 ============
async function onSave() {
  if (!capturedCombo.value) return
  saving.value = true
  try {
    await applyCombo(capturedCombo.value)
    currentCombo.value = capturedCombo.value
    message.success(`已应用:${formatCombo(capturedCombo.value)}`)
  } catch (err: any) {
    message.error('应用失败:' + (err?.message || String(err)))
  } finally {
    saving.value = false
  }
}

async function disableHotkey() {
  saving.value = true
  try {
    await applyCombo('disabled')
    currentCombo.value = 'disabled'
    capturedCombo.value = ''
    message.success('已禁用快捷键')
  } catch (err: any) {
    message.error('禁用失败:' + (err?.message || String(err)))
  } finally {
    saving.value = false
  }
}

async function resetDefault() {
  capturedCombo.value = DEFAULT_COMBO
  await onSave()
}

async function applyCombo(combo: string) {
  await invoke('set_window_shortcut', { combo })
  localStorage.setItem(STORAGE_KEY, combo)
}

function load() {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v) currentCombo.value = v
  } catch {}
}

function goBack() {
  router.push('/leads/new')
}

onMounted(() => {
  load()
})

onBeforeUnmount(() => {
  // 确保设置状态被清理
  isRecording.value = false
})
</script>

<style scoped>
.page {
  padding: 12px;
  max-width: 540px;
  margin: 0 auto;
}
.page-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  margin-bottom: 12px;
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

.card {
  background: #ffffff;
  border-radius: var(--r-lg, 14px);
  padding: 14px 14px 12px;
  margin-bottom: 12px;
  box-shadow: 0 1px 0 rgba(37,99,235,0.04), 0 2px 8px rgba(37,99,235,0.06);
  transition: box-shadow 200ms ease, transform 200ms ease;
}
.card:hover {
  box-shadow: 0 4px 14px rgba(37,99,235,0.12);
  transform: translateY(-1px);
}
.card.current {
  border-left: 3px solid #2563eb;
}
.card.tip {
  border-left: 3px solid #94a3b8;
}
.card-title {
  font-size: 13px;
  font-weight: 700;
  color: #1f2937;
  margin-bottom: 10px;
}

.current-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.hint {
  margin: 0 0 10px 0;
  font-size: 12px;
  color: #64748b;
  line-height: 1.6;
}

.kbd {
  display: inline-block;
  font-family: "SF Mono", Consolas, monospace;
  font-size: 12px;
  background: var(--c-ink-100, #f1f5fb);
  border: 1px solid var(--c-ink-300, #cbd5e1);
  border-bottom-width: 2px;
  border-radius: var(--r-xs, 6px);
  padding: 4px 10px;
  color: var(--c-blue-800, #1e3a8a);
  letter-spacing: 1px;
  box-shadow: 0 1px 0 rgba(0,0,0,0.04);
}

/* 设置快捷键器 */
.recorder {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 64px;
  border: 2px dashed var(--c-ink-300, #cbd5e1);
  border-radius: var(--r-md, 10px);
  background: var(--c-ink-50, #f8fafc);
  transition: all 180ms ease;
  outline: none;
}
.recorder.active {
  border-color: var(--c-blue-600, #2563eb);
  background: linear-gradient(180deg, #eff6ff 0%, #dbeafe 100%);
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.12);
}
.recorder-text {
  font-size: 14px;
  font-weight: 600;
  color: var(--c-ink-700, #334155);
  letter-spacing: 1px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.recorder.active .recorder-text {
  color: var(--c-blue-700, #1d4ed8);
}
.recorder-pulse {
  display: inline-block;
  color: #ef4444;
  animation: pulse 1.2s ease-in-out infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

ul {
  margin: 6px 0;
  padding-left: 18px;
  font-size: 12px;
  color: #475569;
  line-height: 1.7;
}
ul code {
  background: #f1f5f9;
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 11px;
}
</style>