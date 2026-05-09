<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import type { FilmListItem } from '@/types/film'
import FilmCard from './FilmCard.vue'

interface Props {
  title?: string
  /** "更多" 链接 router-link to */
  moreLink?: { path: string; query?: Record<string, string | number> } | string
  items: FilmListItem[]
  /** key 字段名，默认 id */
  itemKey?: keyof FilmListItem
}

const props = withDefaults(defineProps<Props>(), {
  title: '',
  moreLink: '',
  itemKey: 'id'
})

const scrollEl = ref<HTMLElement | null>(null)
const canScrollLeft = ref(false)
const canScrollRight = ref(false)

function updateArrows(): void {
  const el = scrollEl.value
  if (!el) return
  const left = el.scrollLeft
  const max = el.scrollWidth - el.clientWidth
  canScrollLeft.value = left > 4
  canScrollRight.value = left < max - 4
}

function scrollByDir(dir: 1 | -1): void {
  const el = scrollEl.value
  if (!el) return
  const delta = el.clientWidth * 0.8 * dir
  el.scrollBy({ left: delta, behavior: 'smooth' })
}

let resizeObserver: ResizeObserver | null = null

/** TV / 桌面键盘焦点：把获得焦点的子项滚到视口居中（仅在 row 内部） */
function onFocusIn(e: FocusEvent): void {
  const target = e.target as HTMLElement | null
  if (!target) return
  // 只在 row 内部 scroll 容器内的目标才接管
  const scroll = scrollEl.value
  if (!scroll || !scroll.contains(target)) return
  // 只对 focusable 子项生效，避免每个内部按钮都触发
  const focusable = target.closest<HTMLElement>('[data-focusable="true"]')
  if (!focusable) return
  // 行内居中滚动（不影响竖向）
  try {
    focusable.scrollIntoView({ inline: 'center', block: 'nearest', behavior: 'smooth' })
  } catch {
    /* ignore */
  }
}

onMounted(() => {
  nextTick(updateArrows)
  scrollEl.value?.addEventListener('scroll', updateArrows, { passive: true })
  scrollEl.value?.addEventListener('focusin', onFocusIn)
  if (typeof ResizeObserver !== 'undefined' && scrollEl.value) {
    resizeObserver = new ResizeObserver(updateArrows)
    resizeObserver.observe(scrollEl.value)
  }
})
onBeforeUnmount(() => {
  scrollEl.value?.removeEventListener('scroll', updateArrows)
  scrollEl.value?.removeEventListener('focusin', onFocusIn)
  resizeObserver?.disconnect()
  resizeObserver = null
})

const moreTo = computed(() => {
  if (!props.moreLink) return null
  return typeof props.moreLink === 'string'
    ? { path: props.moreLink }
    : props.moreLink
})

function getItemKey(item: FilmListItem, idx: number): string | number {
  const k = props.itemKey
  const v = item[k as keyof FilmListItem]
  if (typeof v === 'string' || typeof v === 'number') return v
  return idx
}
</script>

<template>
  <section class="gf-film-row">
    <header
      v-if="title || moreTo"
      class="container-page flex items-end justify-between gap-[var(--gf-space-4)] mb-[var(--gf-space-4)]"
    >
      <h2
        v-if="title"
        class="gf-film-row__title text-[var(--gf-fs-xl)] font-[var(--gf-fw-bold)] text-primary"
      >
        {{ title }}
      </h2>
      <RouterLink
        v-if="moreTo"
        :to="moreTo"
        class="text-link text-[var(--gf-fs-sm)] inline-flex items-center gap-[var(--gf-space-1)] shrink-0"
        data-focusable="true"
        tabindex="0"
      >
        更多
        <BaseIcon name="chevron-right" size="16px" />
      </RouterLink>
    </header>

    <div class="gf-film-row__viewport relative group">
      <!-- 左右遮罩（桌面） -->
      <div class="gf-film-row__mask-left absolute inset-y-0 left-0 pointer-events-none hidden lg:block" />
      <div class="gf-film-row__mask-right absolute inset-y-0 right-0 pointer-events-none hidden lg:block" />

      <!-- 横向滚动容器 -->
      <div
        ref="scrollEl"
        class="gf-film-row__scroll flex gap-[var(--gf-space-3)] md:gap-[var(--gf-space-4)] overflow-x-auto scroll-smooth"
      >
        <!-- 左侧缩进（与页面 gutter 对齐） -->
        <div class="gf-film-row__edge shrink-0" aria-hidden="true" />
        <div
          v-for="(item, idx) in items"
          :key="getItemKey(item, idx)"
          class="gf-film-row__item shrink-0"
        >
          <slot name="item" :item="item" :index="idx">
            <FilmCard :item="item" :show-title-below="true" />
          </slot>
        </div>
        <div class="gf-film-row__edge shrink-0" aria-hidden="true" />
      </div>

      <!-- 左箭头 -->
      <button
        v-show="canScrollLeft"
        class="gf-film-row__arrow gf-film-row__arrow--left"
        data-focusable="true"
        tabindex="0"
        aria-label="scroll left"
        @click="scrollByDir(-1)"
      >
        <BaseIcon name="chevron-left" size="24px" />
      </button>
      <button
        v-show="canScrollRight"
        class="gf-film-row__arrow gf-film-row__arrow--right"
        data-focusable="true"
        tabindex="0"
        aria-label="scroll right"
        @click="scrollByDir(1)"
      >
        <BaseIcon name="chevron-right" size="24px" />
      </button>
    </div>
  </section>
</template>

<style scoped>
.gf-film-row {
  /* row 之间间距由父级或 grid 控制 */
}

.gf-film-row__scroll {
  scroll-snap-type: x mandatory;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}
.gf-film-row__scroll::-webkit-scrollbar {
  display: none;
}

.gf-film-row__edge {
  /* 与页面 gutter 对齐 */
  width: var(--gf-gutter-mobile);
}
@media (min-width: 768px) {
  .gf-film-row__edge {
    width: var(--gf-gutter-tablet);
  }
}
@media (min-width: 1024px) {
  .gf-film-row__edge {
    width: var(--gf-gutter-desktop);
  }
}

.gf-film-row__item {
  scroll-snap-align: start;
  /* 默认列宽：移动 ~2.2，平板 4.5，桌面 6，>=1440 7，>=1920 8 */
  width: calc((100vw - 32px) / 2.2);
}
@media (min-width: 480px) {
  .gf-film-row__item {
    width: calc((100vw - 32px) / 3.2);
  }
}
@media (min-width: 768px) {
  .gf-film-row__item {
    width: calc((100vw - 48px) / 4.5);
  }
}
@media (min-width: 1024px) {
  .gf-film-row__item {
    width: calc((100vw - 80px) / 6);
  }
}
@media (min-width: 1440px) {
  .gf-film-row__item {
    width: calc(min(100vw - 80px, 1280px) / 7);
  }
}
@media (min-width: 1920px) {
  .gf-film-row__item {
    width: calc(min(100vw - 80px, 1600px) / 8);
  }
}

.gf-film-row__mask-left {
  width: 64px;
  background-image: var(--gf-mask-row-left);
  z-index: var(--gf-z-row);
}
.gf-film-row__mask-right {
  width: 64px;
  background-image: var(--gf-mask-row-right);
  z-index: var(--gf-z-row);
}

.gf-film-row__arrow {
  position: absolute;
  top: 0;
  height: 100%;
  width: 48px;
  display: none;
  align-items: center;
  justify-content: center;
  border: none;
  background-color: rgba(0, 0, 0, 0.55);
  color: var(--gf-text-primary);
  cursor: pointer;
  z-index: calc(var(--gf-z-row) + 1);
  opacity: 0;
  transition:
    opacity var(--gf-dur-fast) var(--gf-ease-standard),
    background-color var(--gf-dur-fast) var(--gf-ease-standard);
}
.gf-film-row__arrow:hover {
  background-color: rgba(0, 0, 0, 0.8);
}
.gf-film-row__arrow--left {
  left: 0;
  border-top-right-radius: var(--gf-radius-md);
  border-bottom-right-radius: var(--gf-radius-md);
}
.gf-film-row__arrow--right {
  right: 0;
  border-top-left-radius: var(--gf-radius-md);
  border-bottom-left-radius: var(--gf-radius-md);
}

@media (hover: hover) and (min-width: 1024px) {
  .gf-film-row__arrow {
    display: inline-flex;
  }
  .gf-film-row__viewport:hover .gf-film-row__arrow,
  .gf-film-row__viewport:focus-within .gf-film-row__arrow {
    opacity: 1;
  }
}

</style>

<style>
/* TV 默认显示箭头（不依赖 hover），加大尺寸 */
[data-mode='tv'] .gf-film-row__arrow {
  display: inline-flex;
  opacity: 1;
  width: 64px;
}
[data-mode='tv'] .gf-film-row__arrow:focus,
[data-mode='tv'] .gf-film-row__arrow:focus-visible {
  background-color: rgba(0, 0, 0, 0.85);
  outline: none;
  box-shadow: 0 0 0 4px var(--gf-brand-cyan);
}

/* TV 卡片间距 +50%（基础 16px → 24px；large 断点 →） */
[data-mode='tv'] .gf-film-row__scroll {
  gap: 24px;
}
@media (min-width: 768px) {
  [data-mode='tv'] .gf-film-row__scroll {
    gap: var(--gf-space-6);
  }
}

/* TV 列宽：1920 视口默认 8 列 */
[data-mode='tv'] .gf-film-row__item {
  width: calc(min(100vw - 96px, 1600px) / 6);
}

/* TV title 字号 */
[data-mode='tv'] .gf-film-row__title {
  font-size: var(--gf-fs-2xl);
}

/* TV header 安全区缩进 */
[data-mode='tv'] .gf-film-row > header.container-page {
  padding-inline: var(--gf-tv-safe);
}
[data-mode='tv'] .gf-film-row__edge {
  width: var(--gf-tv-safe);
}
</style>
