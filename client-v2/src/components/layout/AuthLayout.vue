<script setup lang="ts">
/**
 * 鉴权（登录）布局
 * 居中容器 + 紫色光晕背景
 */
</script>

<template>
  <div
    class="min-h-screen flex-center bg-base text-primary p-[var(--gf-space-6)]"
  >
    <div
      class="w-full max-w-[420px] bg-surface rounded-[var(--gf-radius-2xl)] shadow-card-xl p-[var(--gf-space-8)]"
    >
      <slot />
    </div>
  </div>
</template>
