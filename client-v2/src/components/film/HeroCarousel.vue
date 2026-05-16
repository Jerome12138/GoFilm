<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import type { FilmListItem } from '@/types/film'
import BaseImage from '@/components/base/BaseImage.vue'
import BaseTag from '@/components/base/BaseTag.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { useViewMode } from '@/composables/useViewMode'

interface Props {
  items: FilmListItem[]
  /** 自动切换间隔（ms），默认根据 mode：tv 6000 / 其他 4000 */
  interval?: number
  /** 是否显示左右箭头 */
  showArrows?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  interval: 0,
  showArrows: true
})

const router = useRouter()
const { isTV } = useViewMode()

const current = ref(0)
const total = computed(() => props.items.length)
const active = computed(() => props.items[current.value])

// 进度条重启 key: current 变化时 ++, 让 CSS animation 重新挂载
const progressTick = ref(0)
watch(current, () => {
  progressTick.value += 1
})

const effectiveInterval = computed(() => {
  if (props.interval && props.interval > 0) return props.interval
  return isTV.value ? 6000 : 4000
})

let timer: number | null = null
const paused = ref(false)

function go(idx: number): void {
  if (total.value === 0) return
  const next = (idx + total.value) % total.value
  current.value = next
}

function next(): void {
  go(current.value + 1)
}
function prev(): void {
  go(current.value - 1)
}

function startTimer(): void {
  stopTimer()
  if (total.value <= 1) return
  timer = window.setInterval(() => {
    if (!paused.value) next()
  }, effectiveInterval.value)
}
function stopTimer(): void {
  if (timer !== null) {
    window.clearInterval(timer)
    timer = null
  }
}

onMounted(startTimer)
onBeforeUnmount(stopTimer)

watch(effectiveInterval, () => startTimer())
watch(() => props.items.length, () => {
  current.value = 0
  startTimer()
})

// 触摸滑动
let touchStartX = 0
let touchDx = 0
function onTouchStart(e: TouchEvent): void {
  touchStartX = e.touches[0]?.clientX ?? 0
  touchDx = 0
  paused.value = true
}
function onTouchMove(e: TouchEvent): void {
  const x = e.touches[0]?.clientX ?? 0
  touchDx = x - touchStartX
}
function onTouchEnd(): void {
  if (Math.abs(touchDx) > 60) {
    if (touchDx < 0) next()
    else prev()
  }
  paused.value = false
}

// 详情跳转
function gotoDetail(item: FilmListItem | undefined): void {
  if (!item) return
  router.push({ path: '/filmDetail', query: { link: String(item.id ?? item.mid ?? '') } })
}

// TV 模式首屏自动聚焦"立即播放"按钮
const heroCtaEl = ref<HTMLElement | null>(null)
onMounted(() => {
  if (!isTV.value) return
  // 等待 BaseButton 渲染 + 数据就绪
  setTimeout(() => {
    const root = heroCtaEl.value
    if (!root) return
    const btn = root.querySelector<HTMLElement>('button[data-focusable="true"]')
    if (!btn) return
    const ae = document.activeElement
    if (!ae || ae === document.body || (ae as HTMLElement).tagName === 'BODY') {
      try {
        btn.focus()
      } catch {
        /* ignore */
      }
    }
  }, 250)
})

// 键盘导航：左右箭头切换
function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'ArrowLeft') {
    prev()
  } else if (e.key === 'ArrowRight') {
    next()
  }
}

const tags = computed<string[]>(() => {
  const it = active.value
  if (!it) return []
  const res: string[] = []
  if (it.year) res.push(String(it.year))
  if (it.cName) res.push(String(it.cName))
  if (it.area) res.push(String(it.area))
  return res
})
</script>

<template>
  <section
    class="gf-hero relative w-full overflow-hidden"
    aria-roledescription="carousel"
    @mouseenter="paused = true"
    @mouseleave="paused = false"
    @touchstart="onTouchStart"
    @touchmove="onTouchMove"
    @touchend="onTouchEnd"
    @keydown="onKeydown"
    tabindex="0"
  >
    <!-- 背景图层 -->
    <div class="gf-hero__layers absolute inset-0">
      <div
        v-for="(it, i) in items"
        :key="(it.id ?? i) + '-' + i"
        class="gf-hero__slide absolute inset-0"
        :class="i === current ? 'opacity-100' : 'opacity-0 pointer-events-none'"
        :aria-hidden="i !== current"
      >
        <BaseImage
          :src="it.picture"
          :alt="it.name"
          ratio=""
          :eager="i === 0"
          fit="cover"
          class="gf-hero__image"
        />
      </div>
    </div>

    <!-- 蒙版 -->
    <div class="gf-hero__mask-bottom absolute inset-0 pointer-events-none" />
    <div class="gf-hero__mask-left absolute inset-0 pointer-events-none hidden md:block" />

    <!-- 左下信息层 -->
    <div
      v-if="active"
      class="gf-hero__content absolute inset-x-0 bottom-0 container-page"
    >
      <div class="gf-hero__info">
        <div
          v-if="tags.length"
          class="flex flex-wrap gap-[var(--gf-space-2)] mb-[var(--gf-space-3)]"
        >
          <BaseTag
            v-for="(t, i) in tags"
            :key="i"
            variant="purple"
            size="md"
          >
            {{ t }}
          </BaseTag>
        </div>
        <h2 class="gf-hero__title text-primary">
          {{ active.name }}
        </h2>
        <p
          v-if="active.remarks"
          class="gf-hero__desc text-secondary mt-[var(--gf-space-3)] line-clamp-2"
        >
          {{ active.remarks }}
        </p>
        <div ref="heroCtaEl" class="gf-hero__cta flex flex-wrap gap-[var(--gf-space-3)] mt-[var(--gf-space-5)]">
          <BaseButton
            variant="primary"
            size="lg"
            @click="gotoDetail(active)"
          >
            <template #icon>
              <BaseIcon name="play" size="1.1em" />
            </template>
            立即播放
          </BaseButton>
          <BaseButton
            variant="outline"
            size="lg"
            @click="gotoDetail(active)"
          >
            <template #icon>
              <BaseIcon name="info" size="1.1em" />
            </template>
            详情
          </BaseButton>
        </div>
      </div>
    </div>

    <!-- 左右箭头 -->
    <template v-if="showArrows && total > 1">
      <button
        class="gf-hero__arrow gf-hero__arrow--left"
        data-focusable="true"
        tabindex="0"
        aria-label="prev slide"
        @click="prev"
      >
        <BaseIcon name="chevron-left" size="24px" />
      </button>
      <button
        class="gf-hero__arrow gf-hero__arrow--right"
        data-focusable="true"
        tabindex="0"
        aria-label="next slide"
        @click="next"
      >
        <BaseIcon name="chevron-right" size="24px" />
      </button>
    </template>

    <!-- 指示器 -->
    <!-- 指示条 (bilibili 风格底部横条; 当前条带 4s 自动推进进度填充) -->
    <div
      v-if="total > 1"
      class="gf-hero__bars absolute bottom-[var(--gf-space-4)] left-1/2 -translate-x-1/2 flex items-center gap-[var(--gf-space-2)]"
    >
      <button
        v-for="(_, i) in items"
        :key="i"
        class="gf-hero__bar"
        :class="i === current ? 'gf-hero__bar--active' : ''"
        :aria-label="`go to slide ${i + 1}`"
        :aria-current="i === current ? 'true' : 'false'"
        data-focusable="true"
        tabindex="0"
        @click="go(i)"
      >
        <span
          v-if="i === current"
          :key="progressTick"
          class="gf-hero__bar-progress"
          :style="{ animationDuration: effectiveInterval + 'ms', animationPlayState: paused ? 'paused' : 'running' }"
        />
      </button>
    </div>
  </section>
</template>

<style scoped>
/**
 * Hero 容器尺寸策略 —— 各档屏幕都用 aspect-ratio 主导 + 安全区兜底，
 * 避免单纯 vh 在窄竖屏 / 超宽屏 / 横屏小高度下变形：
 *
 *  ┌──────────────────────────────────────────────────────────────────┐
 *  │ 视口            纵横比         min-height   max-height          │
 *  │ < 480 (mobile)  4 / 5         320px        66vh                 │
 *  │ ≥ 480           16 / 10       360px        62vh                 │
 *  │ ≥ 768 (tablet)  16 / 9        420px        70vh                 │
 *  │ ≥ 1024 (PC)     21 / 9        480px        720px                │
 *  │ ≥ 1600 (大屏)   21 / 9        clamp(560,55vh,820)               │
 *  └──────────────────────────────────────────────────────────────────┘
 */
.gf-hero {
  width: 100%;
  /* 手机竖屏：用 16/10 而不是 4/5，避免大图占满半屏 */
  aspect-ratio: 16 / 10;
  min-height: 220px;
  max-height: 50vh;
  background-color: var(--gf-bg-base);
  outline: none;
}

@media (min-width: 480px) {
  .gf-hero {
    aspect-ratio: 16 / 9;
    min-height: 260px;
    max-height: 55vh;
  }
}

@media (min-width: 768px) {
  .gf-hero {
    aspect-ratio: 16 / 9;
    min-height: 320px;
    max-height: 50vh;
  }
}

@media (min-width: 1024px) {
  .gf-hero {
    aspect-ratio: 21 / 9;
    min-height: 360px;
    max-height: 520px;
  }
}

@media (min-width: 1600px) {
  .gf-hero {
    aspect-ratio: 21 / 9;
    min-height: 420px;
    max-height: clamp(420px, 45vh, 600px);
  }
}

/* 横屏小高度设备（手机横屏 / 平板横屏低分辨率）：限制 max-height 防 hero 过高顶走列表 */
@media (orientation: landscape) and (max-height: 600px) {
  .gf-hero {
    max-height: 88vh;
    min-height: 280px;
  }
}

.gf-hero__image,
.gf-hero__image :deep(img) {
  width: 100%;
  height: 100%;
  border-radius: 0;
}

.gf-hero__slide {
  transition: opacity var(--gf-dur-slow) var(--gf-ease-out);
}

.gf-hero__mask-bottom {
  background-image: var(--gf-mask-hero-bottom);
}
.gf-hero__mask-left {
  background-image: var(--gf-mask-hero-left);
}

.gf-hero__content {
  padding-top: var(--gf-space-8);
  padding-bottom: var(--gf-space-12);
  z-index: 2;
}
@media (min-width: 1024px) {
  .gf-hero__content {
    padding-bottom: 80px;
  }
}

.gf-hero__info {
  max-width: min(640px, 100%);
}

@media (min-width: 768px) {
  .gf-hero__info {
    max-width: min(640px, 60%);
  }
}

.gf-hero__title {
  font-size: var(--gf-fs-hero);
  font-weight: var(--gf-fw-black);
  line-height: var(--gf-lh-tight);
  letter-spacing: var(--gf-tracking-tight);
}

.gf-hero__desc {
  font-size: var(--gf-fs-md);
  line-height: var(--gf-lh-relaxed);
}

.gf-hero__arrow {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 48px;
  height: 64px;
  display: none;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: var(--gf-radius-md);
  background-color: rgba(0, 0, 0, 0.55);
  color: var(--gf-text-primary);
  cursor: pointer;
  z-index: 3;
  transition:
    background-color var(--gf-dur-fast) var(--gf-ease-standard),
    opacity var(--gf-dur-fast) var(--gf-ease-standard);
}
.gf-hero__arrow:hover {
  background-color: rgba(0, 0, 0, 0.8);
}
.gf-hero__arrow--left {
  left: var(--gf-space-4);
}
.gf-hero__arrow--right {
  right: var(--gf-space-4);
}

@media (min-width: 768px) {
  .gf-hero__arrow {
    display: inline-flex;
  }
}

/* 指示条 (横条 + 当前条进度填充) */
.gf-hero__bars {
  z-index: 3;
}

.gf-hero__bar {
  position: relative;
  width: 36px;
  height: 3px;
  border-radius: 2px;
  background-color: rgba(255, 255, 255, 0.3);
  border: none;
  padding: 0;
  cursor: pointer;
  overflow: hidden;
  transition: width var(--gf-dur-base) var(--gf-ease-standard);
}

.gf-hero__bar--active {
  width: 56px;
}

.gf-hero__bar:hover {
  background-color: rgba(255, 255, 255, 0.45);
}

.gf-hero__bar:focus-visible {
  outline: none;
  box-shadow: 0 0 0 2px rgba(74, 209, 229, 0.8);
}

.gf-hero__bar-progress {
  position: absolute;
  inset: 0;
  background-image: var(--gf-brand-gradient);
  transform: scaleX(0);
  transform-origin: left center;
  animation-name: gf-hero-progress;
  animation-timing-function: linear;
  animation-fill-mode: forwards;
  animation-iteration-count: 1;
}

@keyframes gf-hero-progress {
  from { transform: scaleX(0); }
  to { transform: scaleX(1); }
}

.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>

<style>
/* TV 默认显示箭头（不依赖 hover），加大尺寸 + 安全区缩进 */
[data-mode='tv'] .gf-hero__arrow {
  display: inline-flex;
  width: 64px;
  height: 80px;
}
[data-mode='tv'] .gf-hero__arrow--left {
  left: var(--gf-tv-safe);
}
[data-mode='tv'] .gf-hero__arrow--right {
  right: var(--gf-tv-safe);
}
[data-mode='tv'] .gf-hero__arrow:focus,
[data-mode='tv'] .gf-hero__arrow:focus-visible {
  outline: none;
  background-color: rgba(0, 0, 0, 0.85);
  box-shadow: 0 0 0 4px var(--gf-brand-cyan);
}
/* TV 信息区放大 —— 大屏 21/9 + 安全的 max-height 区间，含 4K */
[data-mode='tv'] .gf-hero {
  aspect-ratio: 21 / 9;
  min-height: 600px;
  max-height: clamp(720px, 65vh, 1080px);
}
[data-mode='tv'] .gf-hero__content {
  padding-inline: var(--gf-tv-safe);
  padding-bottom: var(--gf-space-12);
}
[data-mode='tv'] .gf-hero__info {
  max-width: min(900px, 60%);
}
[data-mode='tv'] .gf-hero__desc {
  font-size: var(--gf-fs-lg);
}
[data-mode='tv'] .gf-hero__bar {
  width: 48px;
  height: 4px;
}
[data-mode='tv'] .gf-hero__bar--active {
  width: 72px;
}
</style>
