<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useUserStore } from '@/stores/user'
import { useSiteStore } from '@/stores/site'
import { useViewMode } from '@/composables/useViewMode'
import ManageHeader from './ManageHeader.vue'
import ManageSidebar from './ManageSidebar.vue'

const userStore = useUserStore()
const siteStore = useSiteStore()
const { mode, isMobile, isTablet } = useViewMode()

const drawerOpen = ref(false)
function closeDrawer(): void { drawerOpen.value = false }

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
    class="gf-manage min-h-screen flex flex-col bg-base text-primary"
    :data-mode="mode"
  >
    <ManageHeader
      :show-hamburger="isMobile"
      @toggle-drawer="drawerOpen = !drawerOpen"
    />
    <div class="flex-1 flex overflow-hidden relative">
      <ManageSidebar
        :variant="isMobile ? 'drawer' : isTablet ? 'icon-rail' : 'full'"
        :open="drawerOpen"
        @close="closeDrawer"
      />
      <main
        class="flex-1 overflow-x-auto overflow-y-auto p-[var(--gf-space-3)] md:p-[var(--gf-space-4)] lg:p-[var(--gf-space-6)]"
      >
        <slot />
      </main>
    </div>
  </div>
</template>
