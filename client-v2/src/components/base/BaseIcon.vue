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
  | 'search'
  | 'close'
  | 'image'
  | 'menu'
  | 'plus'
  | 'minus'

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
  'minus': 'M19 13H5v-2h14z'
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
