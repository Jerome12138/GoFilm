<script setup lang="ts">
import { useUserStore, useUIStore } from '@/stores'
import { storeToRefs } from 'pinia'
import { useRouter } from 'vue-router'

const userStore = useUserStore()
const uiStore = useUIStore()
const router = useRouter()
const { info } = storeToRefs(userStore)

async function handleLogout(): Promise<void> {
  await userStore.logout()
  await router.replace('/login')
}
</script>

<template>
  <header
    class="flex-between bg-surface border-b border-subtle px-[var(--gf-space-6)] h-[56px] sticky top-0 z-[var(--gf-z-header)]"
  >
    <div class="flex items-center gap-[var(--gf-space-3)]">
      <button
        class="text-secondary hover:text-primary"
        type="button"
        @click="uiStore.toggleSidebar()"
      >
        <span aria-hidden="true">≡</span>
        <span class="sr-only">切换侧栏</span>
      </button>
      <span class="text-lg font-semibold">GoFilm 管理后台</span>
    </div>
    <div class="flex items-center gap-[var(--gf-space-3)]">
      <span class="text-secondary text-sm">
        {{ info?.nickname || info?.username || '管理员' }}
      </span>
      <button
        class="text-secondary hover:text-primary text-sm"
        type="button"
        @click="handleLogout"
      >
        退出
      </button>
    </div>
  </header>
</template>
