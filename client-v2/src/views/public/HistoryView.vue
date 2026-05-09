<script setup lang="ts">
import BasePagePlaceholder from '@/components/base/BasePagePlaceholder.vue'
import { useHistoryStore } from '@/stores'
import { storeToRefs } from 'pinia'

const historyStore = useHistoryStore()
const { list } = storeToRefs(historyStore)
</script>

<template>
  <BasePagePlaceholder
    title="观看历史"
    description="HistoryView 占位 — 已读取 cookie filmHistory"
  >
    <div class="text-secondary">当前历史条数：{{ list.length }}</div>
  </BasePagePlaceholder>
</template>
