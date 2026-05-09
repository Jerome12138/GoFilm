<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useSiteStore, useNavStore } from '@/stores'
import { storeToRefs } from 'pinia'

/**
 * 公开端 Header —— 占位实现
 * 后续 STORY-007/008 由专门组件完善（搜索框、滚动变背景、TV 焦点）
 */

const siteStore = useSiteStore()
const navStore = useNavStore()
const { basic } = storeToRefs(siteStore)
const { list: navList } = storeToRefs(navStore)

const failed = ref(false)

onMounted(() => {
  // 容错：拉取失败时不阻塞页面
  Promise.all([siteStore.ensureLoaded(), navStore.ensureLoaded()]).catch(() => {
    failed.value = true
  })
})
</script>

<template>
  <header
    class="sticky top-0 z-[var(--gf-z-header)] flex-between bg-header-scrolled px-[var(--gf-space-6)] h-[64px] border-b border-subtle"
  >
    <RouterLink to="/index" class="flex items-center gap-[var(--gf-space-3)]">
      <span class="text-xl font-bold text-brand-gradient">
        {{ basic?.siteName || 'GoFilm' }}
      </span>
    </RouterLink>

    <nav class="hidden md:flex items-center gap-[var(--gf-space-6)]">
      <RouterLink
        v-for="nav in navList.slice(0, 6)"
        :key="nav.id"
        :to="{ path: '/filmClassify', query: { Pid: nav.id } }"
        class="text-secondary hover:text-primary transition-colors"
      >
        {{ nav.name }}
      </RouterLink>
    </nav>

    <div class="flex items-center gap-[var(--gf-space-3)]">
      <RouterLink to="/search" class="text-secondary hover:text-primary">
        搜索
      </RouterLink>
      <RouterLink to="/history" class="text-secondary hover:text-primary">
        历史
      </RouterLink>
    </div>
  </header>
</template>
