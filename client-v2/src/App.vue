<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useViewMode } from '@/composables/useViewMode'
import { useSiteStore, useNavStore } from '@/stores'

// 布局壳静态导入（首屏必须）
import PublicLayout from '@/components/layout/PublicLayout.vue'
import ManageLayout from '@/components/layout/ManageLayout.vue'
import AuthLayout from '@/components/layout/AuthLayout.vue'
import MinimalLayout from '@/components/layout/MinimalLayout.vue'

const route = useRoute()

// 启动 view-mode 检测，写入 <html data-mode>
useViewMode()

const layoutMap = {
  public: PublicLayout,
  manage: ManageLayout,
  auth: AuthLayout,
  minimal: MinimalLayout
} as const

type LayoutKey = keyof typeof layoutMap

const currentLayout = computed(() => {
  const key = (route.meta.layout as LayoutKey | undefined) ?? 'public'
  return layoutMap[key] ?? PublicLayout
})

// Toast 容器懒加载（拦截器需要）
const ToastContainer = defineAsyncComponent(
  () => import('@/components/base/BaseToastContainer.vue')
)

const siteStore = useSiteStore()
const navStore = useNavStore()

onMounted(() => {
  // 站点信息 / 顶级导航预热（并行，失败静默，不阻塞页面渲染）
  Promise.all([siteStore.ensureLoaded(), navStore.ensureLoaded()]).catch(() => {
    // 拦截器已统一 toast，这里仅吞错避免冒泡
  })
})
</script>

<template>
  <component :is="currentLayout">
    <RouterView v-slot="{ Component, route: r }">
      <Transition name="page" mode="out-in">
        <component :is="Component" :key="r.fullPath" />
      </Transition>
    </RouterView>
  </component>
  <ToastContainer />
</template>

<style>
.page-enter-active,
.page-leave-active {
  transition:
    opacity var(--gf-dur-page) var(--gf-ease-out),
    transform var(--gf-dur-page) var(--gf-ease-out);
}
.page-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
.page-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
