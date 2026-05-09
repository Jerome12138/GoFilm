<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { manageApi } from '@/api'
import type { DashboardStat } from '@/types/manage'
import BaseSkeleton from '@/components/base/BaseSkeleton.vue'
import BaseEmpty from '@/components/base/BaseEmpty.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'

const data = ref<DashboardStat | null>(null)
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  try {
    data.value = await manageApi.system.dashboard()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
})

const cards = computed(() => {
  if (!data.value) return []
  const d = data.value
  return [
    { label: '影片总数', value: d.filmCount ?? 0, icon: 'film', tint: 'from-[#9b49e7] to-[#4ad1e5]' },
    { label: '采集源', value: d.collectCount ?? 0, icon: 'magic', tint: 'from-[#E50914] to-[#ff6b6b]' },
    { label: '定时任务', value: d.cronCount ?? 0, icon: 'clock', tint: 'from-[#22c55e] to-[#4ad1e5]' },
    {
      label: '磁盘使用',
      value: d.diskUsage
        ? `${Math.round((d.diskUsage.used / d.diskUsage.total) * 100)}%`
        : '—',
      icon: 'folder',
      tint: 'from-[#f59e0b] to-[#E50914]'
    }
  ]
})
</script>

<template>
  <section class="flex flex-col gap-[var(--gf-space-6)]">
    <header>
      <h1 class="text-[var(--gf-fs-2xl)] font-[var(--gf-fw-bold)] text-brand-gradient">
        管理中心
      </h1>
      <p class="text-secondary text-sm mt-[var(--gf-space-1)]">
        概览 GoFilm 的核心运营指标
      </p>
    </header>

    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-[var(--gf-space-4)]">
      <BaseSkeleton v-for="i in 4" :key="i" shape="rect" height="128px" />
    </div>

    <BaseEmpty v-else-if="error" :description="error" />

    <div
      v-else
      class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-[var(--gf-space-4)]"
    >
      <div
        v-for="card in cards"
        :key="card.label"
        class="card p-[var(--gf-space-5)] bg-gradient-to-br relative overflow-hidden"
        :class="card.tint"
      >
        <div class="absolute -right-4 -bottom-4 opacity-20 text-white">
          <BaseIcon :name="card.icon" size="120px" />
        </div>
        <div class="relative">
          <div class="text-white/80 text-sm">{{ card.label }}</div>
          <div class="text-white text-[var(--gf-fs-3xl)] font-[var(--gf-fw-black)] mt-[var(--gf-space-2)]">
            {{ card.value }}
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
