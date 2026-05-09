<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <div
    class="min-h-screen flex flex-col items-center justify-center gap-[var(--gf-space-6)] bg-base px-[var(--gf-space-4)]"
  >
    <p class="text-[var(--gf-fs-3xl)] font-[var(--gf-fw-black)] text-brand-gradient">
      404
    </p>
    <p class="text-secondary text-[var(--gf-fs-lg)]">页面不存在或已被移除</p>
    <button
      class="px-[var(--gf-space-6)] py-[var(--gf-space-3)] bg-brand text-primary rounded-pill transition shadow-brand-glow"
      data-focusable="true"
      @click="router.replace('/index')"
    >
      返回首页
    </button>
  </div>
</template>
