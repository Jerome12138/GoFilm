<script setup lang="ts">
import { onMounted } from 'vue'
import { useUserStore } from '@/stores/user'
import { useSiteStore } from '@/stores/site'
import ManageHeader from './ManageHeader.vue'
import ManageSidebar from './ManageSidebar.vue'

const userStore = useUserStore()
const siteStore = useSiteStore()

onMounted(async () => {
  await siteStore.ensureLoaded()
  if (userStore.isLoggedIn && !userStore.info) {
    try {
      await userStore.fetchInfo()
    } catch {
      // 拦截器会处理 401
    }
  }
})
</script>

<template>
  <div
    class="min-h-screen flex flex-col bg-base text-primary"
    data-mode="desktop"
  >
    <ManageHeader />
    <div class="flex-1 flex overflow-hidden">
      <ManageSidebar />
      <main
        class="flex-1 p-[var(--gf-space-6)] overflow-x-auto overflow-y-auto"
      >
        <slot />
      </main>
    </div>
  </div>
</template>
