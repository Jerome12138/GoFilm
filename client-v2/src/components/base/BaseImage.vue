<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

interface Props {
  src?: string
  alt?: string
  /** 宽高比，例如 '2/3' '16/9' */
  ratio?: string
  /** 跳过懒加载，直接立即加载（首屏 hero） */
  eager?: boolean
  /** 圆角类（透传），默认无 */
  rounded?: string
  /** object-fit */
  fit?: 'cover' | 'contain' | 'fill' | 'scale-down'
}

const props = withDefaults(defineProps<Props>(), {
  src: '',
  alt: '',
  ratio: '2/3',
  eager: false,
  rounded: '',
  fit: 'cover'
})

const wrapEl = ref<HTMLElement | null>(null)
const visible = ref(false)
const loaded = ref(false)
const errored = ref(false)
const currentSrc = ref<string>('')

let observer: IntersectionObserver | null = null

const aspectStyle = computed(() => {
  if (!props.ratio) {
    // 空 ratio 表示由父容器控制尺寸（hero 全屏背景等场景）
    return { width: '100%', height: '100%' }
  }
  return { aspectRatio: props.ratio }
})

function startLoad(): void {
  if (!props.src) {
    errored.value = true
    return
  }
  currentSrc.value = props.src
}

function onLoaded(): void {
  loaded.value = true
}
function onError(): void {
  errored.value = true
}

onMounted(() => {
  if (props.eager) {
    visible.value = true
    startLoad()
    return
  }
  if (typeof IntersectionObserver === 'undefined' || !wrapEl.value) {
    visible.value = true
    startLoad()
    return
  }
  // 大屏 / TV 视口预加载更远，移动端更近
  const dataMode =
    typeof document !== 'undefined' ? document.documentElement.getAttribute('data-mode') : ''
  const rootMargin =
    dataMode === 'tv' ? '600px 0px' : dataMode === 'desktop' ? '400px 0px' : '200px 0px'
  observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          visible.value = true
          startLoad()
          observer?.disconnect()
          observer = null
          break
        }
      }
    },
    { rootMargin }
  )
  observer.observe(wrapEl.value)
})

onBeforeUnmount(() => {
  observer?.disconnect()
  observer = null
})

watch(
  () => props.src,
  (next) => {
    loaded.value = false
    errored.value = false
    if (visible.value && next) {
      currentSrc.value = next
    }
  }
)
</script>

<template>
  <div
    ref="wrapEl"
    class="gf-base-image relative overflow-hidden bg-elevated"
    :class="rounded"
    :style="aspectStyle"
  >
    <!-- 骨架/占位 -->
    <div
      v-if="!loaded && !errored"
      class="absolute inset-0 gf-base-image__skeleton"
      aria-hidden="true"
    />
    <!-- 真实图片 -->
    <img
      v-if="visible && currentSrc && !errored"
      :src="currentSrc"
      :alt="alt"
      :loading="eager ? 'eager' : 'lazy'"
      decoding="async"
      class="gf-base-image__img absolute inset-0 w-full h-full"
      :class="[
        loaded ? 'opacity-100' : 'opacity-0',
        fit === 'cover' && 'object-cover',
        fit === 'contain' && 'object-contain',
        fit === 'fill' && 'object-fill',
        fit === 'scale-down' && 'object-scale-down'
      ]"
      @load="onLoaded"
      @error="onError"
    />
    <!-- 错误回退（无外部 404.png 时使用渐变） -->
    <div
      v-if="errored"
      class="absolute inset-0 flex items-center justify-center text-muted text-sm gf-base-image__fallback"
      role="img"
      :aria-label="alt || 'image failed to load'"
    >
      <BaseIcon name="image" size="32px" />
    </div>
  </div>
</template>

<style scoped>
.gf-base-image__img {
  transition: opacity var(--gf-dur-base) var(--gf-ease-standard);
}

.gf-base-image__skeleton {
  background: linear-gradient(
    90deg,
    rgba(255, 255, 255, 0.04) 0%,
    rgba(255, 255, 255, 0.08) 50%,
    rgba(255, 255, 255, 0.04) 100%
  );
  background-size: 200% 100%;
  animation: gf-shimmer 1.4s linear infinite;
}

.gf-base-image__fallback {
  background: linear-gradient(135deg, #1c1d22 0%, #2a2b32 50%, #1c1d22 100%);
}

@keyframes gf-shimmer {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}
</style>
