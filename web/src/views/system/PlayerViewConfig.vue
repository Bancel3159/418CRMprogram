<template>
  <div>
    <div class="page-header">
      <n-text depth="3" style="font-size: 12px">
        玩家列表由「拉新端核心 / 维护端跟进 / 订单聚合 / 自定义扩展」等字段聚合而成；从左侧字段池添加到右侧完成配置
      </n-text>
    </div>
    <ListViewEditor entity-type="player" />
  </div>
</template>

<script setup lang="ts">
import { NText } from 'naive-ui'
import ListViewEditor from '@/components/ListViewEditor.vue'
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: flex-end;
  align-items: baseline;
  margin-bottom: 12px;
}
</style>