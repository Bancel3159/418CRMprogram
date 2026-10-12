<template>
  <n-config-provider :theme-overrides="themeOverrides" :locale="zhCN" :date-locale="dateZhCN">
    <n-message-provider>
      <n-dialog-provider>
        <!-- 窗口控制(无原生标题栏,自定义拖拽条 + 最小化/关闭按钮) -->
        <WindowControls />
        <router-view />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import { zhCN, dateZhCN, NConfigProvider, NMessageProvider, NDialogProvider } from 'naive-ui'
import type { GlobalThemeOverrides } from 'naive-ui'
import WindowControls from '@/layouts/WindowControls.vue'

// 整体蓝色主题(从绿色 #18a058 调整为蓝色)
const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#2563eb',
    primaryColorHover: '#3b82f6',
    primaryColorPressed: '#1d4ed8',
    primaryColorSuppl: '#3b82f6',
    successColor: '#2563eb',
    infoColor: '#3b82f6',
    warningColor: '#f59e0b'
  }
}

const HOTKEY_STORAGE_KEY = 'lead_hotkey_combo'
const DEFAULT_COMBO = 'Ctrl+`' // 默认 Ctrl + 反引号(`)

/**
 * 挂载时把当前用户绑定的快捷键组合应用到 Rust 侧,完成"前端决定,Rust 注册"的链路。
 * - 首次启动:localStorage 没值 → 用默认 Ctrl+`
 * - 用户改过:从 localStorage 读出上次保存的 combo 字符串,重新注册
 * - 用户选「disabled」:Rust 注销旧快捷键并跳过注册
 * 注:此操作在浏览器 dev 模式下 invoke 会失败 —— 静默忽略即可(Vite 用,无 Rust)。
 */
onMounted(async () => {
  try {
    let combo = localStorage.getItem(HOTKEY_STORAGE_KEY)
    if (!combo) {
      combo = DEFAULT_COMBO
      localStorage.setItem(HOTKEY_STORAGE_KEY, combo)
    }
    await invoke('set_window_shortcut', { combo })
  } catch (e) {
    // 在 Vite 仅浏览器调试时会失败(没有 invoke bridge),静默即可
    // eslint-disable-next-line no-console
    console.warn('[App] set_window_shortcut skipped:', e)
  }
})
</script>

<style>
/* 全局 reset(细节看 styles/global.css 里的设计 token) */
html, body, #app {
  margin: 0;
  padding: 0;
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, "PingFang SC", "Microsoft YaHei", sans-serif;
  -webkit-font-smoothing: antialiased;
}
body {
  background: transparent;
}
#app {
  padding-top: 32px;
  background: var(--c-ink-50);
  /* 整窗圆角 —— 与 OS 层 DWMWCP_ROUND 对齐,修复"圆角 + 矩形边角并存"的视觉割裂 */
  border-radius: 14px;
  /* 把内部内容裁剪到圆角范围内(包含 fixed 子元素,需 transform 让 #app 成为它们的 containing block) */
  overflow: hidden;
  /* 让 #app 成为 fixed 子元素的 containing block,从而 overflow:hidden 能裁剪 WindowControls */
  transform: translateZ(0);
  -webkit-transform: translateZ(0);
  /* OS 阴影已去掉,改用 CSS box-shadow 制造视觉深度(被 #app 圆角自然裁剪) */
  box-shadow: 0 8px 28px rgba(37, 99, 235, 0.18), 0 2px 8px rgba(15, 23, 42, 0.06);
}
</style>
