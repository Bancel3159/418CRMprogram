<template>
  <div>
    <div class="page-header">
      <n-text depth="3" style="font-size: 12px">
        订单列表由「订单核心 / 关联线索 / 元数据 / 数据回传原始字段」等聚合而成；从左侧字段池添加到右侧完成配置
      </n-text>
    </div>
    <ListViewEditor entity-type="order" />
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