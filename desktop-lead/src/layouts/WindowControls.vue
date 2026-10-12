<template>
  <!-- 顶部 28px 可拖拽条 - 视觉圆角 + 蓝色调,与 #app 顶部圆角对齐 -->
  <div class="drag-region" data-tauri-drag-region>
    <div class="drag-handle">
      <span class="brand-dot"></span>
    </div>
  </div>

  <!-- 右上角窗口控制:最小化 / 关闭 -->
  <div class="window-controls">
    <button class="ctrl-btn" title="最小化" @click="onMinimize">
      <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
        <line x1="2" y1="6" x2="10" y2="6" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
      </svg>
    </button>
    <button class="ctrl-btn close-btn" title="关闭 (Ctrl+` 唤回)" @click="onClose">
      <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
        <line x1="2" y1="2" x2="10" y2="10" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
        <line x1="10" y1="2" x2="2" y2="10" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
      </svg>
    </button>
  </div>

  <!-- 自定义 resize 边缘 + 右下角抓手
       (decorations:false 模式下 OS 不画 resize hit 区,要靠 CSS 兜底) -->
  <div class="resize-edge resize-top"    @mousedown="(e) => startResize(e, 'North')" />
  <div class="resize-edge resize-bottom" @mousedown="(e) => startResize(e, 'South')" />
  <div class="resize-edge resize-left"   @mousedown="(e) => startResize(e, 'West')" />
  <div class="resize-edge resize-right"  @mousedown="(e) => startResize(e, 'East')" />
  <div class="resize-grip" title="拖动调整窗口大小" @mousedown="(e) => startResize(e, 'SouthEast')">
    <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
      <path d="M9 4 L4 9 M9 7 L7 9" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
    </svg>
  </div>
</template>

<script setup lang="ts">
import { getCurrentWindow } from '@tauri-apps/api/window'

const appWindow = getCurrentWindow()

async function onMinimize() {
  try { await appWindow.minimize() } catch (e) { console.warn(e) }
}
async function onClose() {
  try { await appWindow.close() } catch (e) { console.warn(e) }
}

/**
 * 启动原生 resize(invoke Tauri 的 start_resize Rust 端 API)
 * direction: 'North' | 'South' | 'East' | 'West' | 'NorthEast' | 'NorthWest' | 'SouthEast' | 'SouthWest'
 */
async function startResize(e: MouseEvent, direction: any) {
  e.preventDefault()
  try {
    // @ts-ignore  startResize 在 Tauri 2 的 Window 类上以类型存在
    await appWindow.startResize(direction)
  } catch (err) {
    console.warn('[WindowControls] startResize failed:', err)
  }
}
</script>

<style scoped>
/* 顶部拖拽条:14px 圆角顶部 + 蓝色调,中间放品牌标识 */
.drag-region {
  -webkit-app-region: drag;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 28px;
  z-index: 9998;
  background: linear-gradient(180deg, rgba(37, 99, 235, 0.12), rgba(37, 99, 235, 0));
  border-radius: 14px 14px 0 0; /* 与 #app 顶部圆角一致,实现"窗口栏圆角"效果 */
  pointer-events: auto;
  cursor: move;
  display: flex;
  align-items: center;
  justify-content: center;
  /* 与 #app 同步:14px 圆角,这样窗口顶角呈现完美的圆弧(不会露出方角) */
  overflow: hidden;
}
.drag-handle {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: #1d4ed8;
  font-weight: 600;
  letter-spacing: 0.5px;
  -webkit-app-region: drag;
  pointer-events: none;
}
.brand-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #2563eb;
  box-shadow: 0 0 6px rgba(37, 99, 235, 0.6);
}

/* 右上角窗口控件 */
.window-controls {
  position: fixed;
  top: 0;
  right: 0;
  height: 28px;
  display: flex;
  z-index: 9999;
  -webkit-app-region: no-drag;
}

.ctrl-btn {
  width: 36px;
  height: 28px;
  border: none;
  background: transparent;
  color: #475569;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: background-color 0.12s, color 0.12s;
}
.ctrl-btn:hover {
  background: rgba(37, 99, 235, 0.15);
  color: #1d4ed8;
}
.ctrl-btn.close-btn {
  /* 关闭按钮在窗口右上角,圆角与 #app 顶角弧度一致,避免方角戳出圆角窗 */
  border-radius: 0 14px 0 0;
}
.ctrl-btn.close-btn:hover {
  background: #ef4444;
  color: #ffffff;
}

/* ===== resize 区域 ===== */
/* 四条边缘各 6px 透明 hit 区,鼠标变对应 cursor */
.resize-edge {
  position: fixed;
  -webkit-app-region: no-drag;
  z-index: 9997;
}
.resize-top    { top: 0;    left: 6px;   right: 6px;  height: 6px; cursor: ns-resize; }
.resize-bottom { bottom: 0; left: 6px;   right: 22px; height: 6px; cursor: ns-resize; }
.resize-left   { left: 0;   top: 6px;    bottom: 6px; width: 6px;  cursor: ew-resize; }
.resize-right  { right: 0;  top: 6px;    bottom: 22px;width: 6px;  cursor: ew-resize; }

/* 右下角抓手(可见)+ 三角 grips */
.resize-grip {
  position: fixed;
  right: 4px;
  bottom: 4px;
  width: 18px;
  height: 18px;
  z-index: 9998;
  cursor: nwse-resize;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(37, 99, 235, 0.5);
  -webkit-app-region: no-drag;
}
.resize-grip:hover {
  color: rgba(37, 99, 235, 0.9);
}
</style>
