<script setup lang="ts">
import PublicHeader from './PublicHeader.vue'
import PublicFooter from './PublicFooter.vue'
import MobileTabbar from './MobileTabbar.vue'
import BackToTop from './BackToTop.vue'
import RouteProgress from './RouteProgress.vue'
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
    <RouteProgress />
    <PublicHeader />
    <main class="gf-public-layout__main">
      <slot />
    </main>
    <PublicFooter />
    <!-- 移动端底部 tabbar (>= md 自身 hidden) -->
    <MobileTabbar />
    <!-- 滚动 > 600px 后浮出的回顶 FAB -->
    <BackToTop />
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
  /* 移动端 tabbar fixed 在底部, 留出 padding 不让最后一屏内容被压住 */
  padding-bottom: calc(var(--gf-tabbar-height, 56px) + env(safe-area-inset-bottom, 0));
}

@media (min-width: 768px) {
  .gf-public-layout__main {
    min-height: calc(100vh - 64px - 200px);
    padding-bottom: 0; /* md+ tabbar 隐藏, 不需要留位 */
  }
}
</style>

<style>
/* TV 安全区 + 居中容器 */
[data-mode='tv'] .gf-public-layout {
  /* layout 自身保持流式（背景需要全屏），仅 main 居中 */
}
[data-mode='tv'] .gf-public-layout__main {
  min-height: calc(100vh - 96px - 200px);
}
/* TV 模式下 .container-page 走 1600 居中（已由 theme.css 内 --gf-container-max-2xl 控制） */
[data-mode='tv'] .container-page {
  max-width: var(--gf-container-max-2xl);
  margin-inline: auto;
  padding-inline: var(--gf-tv-safe);
}
</style>
