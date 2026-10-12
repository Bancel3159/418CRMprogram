<template>
  <div class="broadcast-panel">
    <!-- 复用 system/BroadcastTemplates 主体（admin 限定） -->
    <BroadcastTemplates />
  </div>
</template>

<script setup lang="ts">
import BroadcastTemplates from '@/views/system/BroadcastTemplates.vue'
</script>

<style scoped>
.broadcast-panel {
  padding-top: 4px;
}
</style>