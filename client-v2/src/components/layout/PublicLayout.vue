<script setup lang="ts">
import PublicHeader from './PublicHeader.vue'
import PublicFooter from './PublicFooter.vue'
import { useViewMode } from '@/composables/useViewMode'

/**
 * 用户端布局壳
 * mobile / desktop / tv 三档共享同一壳
 * 字号 / 间距 / 焦点环差异由 theme.css 内 [data-mode] 段处理
 */
const { mode } = useViewMode()
</script>

<template>
  <div
    class="gf-public-layout"
    :data-mode="mode"
  >
    <PublicHeader />
    <main class="gf-public-layout__main">
      <slot />
    </main>
    <PublicFooter />
  </div>
</template>

<style scoped>
.gf-public-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background-color: var(--gf-bg-base);
  color: var(--gf-text-primary);
}

.gf-public-layout__main {
  flex: 1 0 auto;
  display: flex;
  flex-direction: column;
  min-height: calc(100vh - 56px - 220px);
}

@media (min-width: 768px) {
  .gf-public-layout__main {
    min-height: calc(100vh - 64px - 200px);
  }
}
</style>

<style>
/* TV 安全区：仅作用在 main 内的页面区，header/footer 自行处理 */
[data-mode='tv'] .gf-public-layout__main {
  min-height: calc(100vh - 96px - 200px);
}
</style>
