<script setup lang="ts">
import { computed } from 'vue'

/**
 * 轻量 inline SVG 图标组件。
 *
 * 临时方案：unocss/preset-icons 0.62.4 + carbon/mdi 在 build 阶段会输出非法 CSS，
 * 故所有业务图标走本地 inline SVG 集合（仅常用 ~10 个，按需扩展）。
 *
 * 后续 Story 若升级 unocss 至兼容版本可重新启用 preset-icons。
 */

type IconName =
  | 'play'
  | 'info'
  | 'chevron-left'
  | 'chevron-right'
  | 'chevron-down'
  | 'chevron-up'
  | 'search'
  | 'close'
  | 'image'
  | 'menu'
  | 'plus'
  | 'minus'
  | 'history'
  | 'star'
  | 'heart'
  | 'share'
  | 'user'
  | 'home'
  | 'arrow-left'
  | 'film'
  | 'pause'
  | 'play-circle'
  | 'skip-next'
  | 'skip-prev'
  | 'volume-up'
  | 'volume-down'
  | 'autoplay'

interface Props {
  name: IconName
  /** CSS 尺寸值，默认 1em */
  size?: string
  label?: string
}

const props = withDefaults(defineProps<Props>(), {
  size: '1em',
  label: ''
})

const wrapperStyle = computed(() => ({
  width: props.size,
  height: props.size
}))

/** 24x24 viewBox 路径表 — 简化轮廓，足够列表 / 按钮使用 */
const PATHS: Record<IconName, string> = {
  'play': 'M8 5v14l11-7z',
  'info': 'M11 7h2v2h-2zM11 11h2v6h-2zM12 2C6.48 2 2 6.48 2 12s4.48 10 10 10s10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8s8 3.59 8 8s-3.59 8-8 8z',
  'chevron-left': 'M15.41 7.41L14 6l-6 6l6 6l1.41-1.41L10.83 12z',
  'chevron-right': 'M10 6L8.59 7.41L13.17 12l-4.58 4.59L10 18l6-6z',
  'search':
    'M15.5 14h-.79l-.28-.27a6.5 6.5 0 1 0-.7.7l.27.28v.79l5 4.99L20.49 19zm-6 0A4.5 4.5 0 1 1 14 9.5A4.5 4.5 0 0 1 9.5 14',
  'close': 'M19 6.41L17.59 5L12 10.59L6.41 5L5 6.41L10.59 12L5 17.59L6.41 19L12 13.41L17.59 19L19 17.59L13.41 12z',
  'image':
    'M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z',
  'menu': 'M3 18h18v-2H3zm0-5h18v-2H3zm0-7v2h18V6z',
  'plus': 'M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6z',
  'minus': 'M19 13H5v-2h14z',
  'chevron-down': 'M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6l-6-6z',
  'chevron-up': 'M7.41 15.41L12 10.83l4.59 4.58L18 14l-6-6l-6 6z',
  'history':
    'M13 3a9 9 0 0 0-9 9H1l3.89 3.89l.07.14L9 12H6c0-3.87 3.13-7 7-7s7 3.13 7 7s-3.13 7-7 7c-1.93 0-3.68-.79-4.94-2.06l-1.42 1.42A8.954 8.954 0 0 0 13 21a9 9 0 0 0 0-18m-1 5v5l4.28 2.54l.72-1.21l-3.5-2.08V8z',
  'star':
    'M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2L9.19 8.63L2 9.24l5.46 4.73L5.82 21z',
  'heart':
    'M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5C2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3C19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54z',
  'share':
    'M18 16.08c-.76 0-1.44.3-1.96.77L8.91 12.7c.05-.23.09-.46.09-.7s-.04-.47-.09-.7l7.05-4.11c.54.5 1.25.81 2.04.81c1.66 0 3-1.34 3-3s-1.34-3-3-3s-3 1.34-3 3c0 .24.04.47.09.7L8.04 9.81C7.5 9.31 6.79 9 6 9c-1.66 0-3 1.34-3 3s1.34 3 3 3c.79 0 1.5-.31 2.04-.81l7.12 4.16c-.05.21-.08.43-.08.65c0 1.61 1.31 2.92 2.92 2.92s2.92-1.31 2.92-2.92s-1.31-2.92-2.92-2.92',
  'user':
    'M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4s-4 1.79-4 4s1.79 4 4 4m0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4',
  'home': 'M10 20v-6h4v6h5v-8h3L12 3L2 12h3v8z',
  'arrow-left': 'M20 11H7.83l5.59-5.59L12 4l-8 8l8 8l1.41-1.41L7.83 13H20z',
  'film':
    'M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.11 0-1.99.89-1.99 2L2 18c0 1.11.89 2 2 2h16c1.11 0 2-.89 2-2V4z',
  'pause': 'M6 19h4V5H6v14m8-14v14h4V5h-4z',
  'play-circle':
    'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10s10-4.48 10-10S17.52 2 12 2m-2 14.5v-9l6 4.5l-6 4.5z',
  'skip-next': 'M6 18l8.5-6L6 6v12M16 6v12h2V6h-2z',
  'skip-prev': 'M6 6h2v12H6V6m3.5 6l8.5 6V6l-8.5 6z',
  'volume-up':
    'M3 9v6h4l5 5V4L7 9H3m13.5 3A4.5 4.5 0 0 0 14 7.97v8.05c1.48-.73 2.5-2.25 2.5-4.02M14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z',
  'volume-down':
    'M7 9v6h4l5 5V4l-5 5H7m9.5 3A4.5 4.5 0 0 0 14 7.97v8.05c1.48-.73 2.5-2.25 2.5-4.02z',
  'autoplay':
    'M12 20c4.41 0 8-3.59 8-8s-3.59-8-8-8s-8 3.59-8 8s3.59 8 8 8m0-18c5.52 0 10 4.48 10 10s-4.48 10-10 10S2 17.52 2 12S6.48 2 12 2m-1.86 5.5L16.5 12l-6.36 4.5V7.5z'
}

const path = computed(() => PATHS[props.name] ?? '')
</script>

<template>
  <span
    class="gf-icon"
    :style="wrapperStyle"
    role="img"
    :aria-label="label || undefined"
    :aria-hidden="label ? undefined : 'true'"
  >
    <svg
      viewBox="0 0 24 24"
      fill="currentColor"
      width="100%"
      height="100%"
      focusable="false"
    >
      <path :d="path" />
    </svg>
  </span>
</template>

<style scoped>
.gf-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  vertical-align: middle;
  line-height: 1;
}
.gf-icon svg {
  display: block;
}
</style>
