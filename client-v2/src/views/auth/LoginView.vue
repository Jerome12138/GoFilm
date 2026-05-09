<script setup lang="ts">
/**
 * 登录占位 —— STORY-016 完善真实表单 + 调用 useUserStore.login
 */
import { useRoute } from 'vue-router'
const route = useRoute()
</script>

<template>
  <div class="flex flex-col gap-[var(--gf-space-4)]">
    <h1 class="text-[var(--gf-fs-2xl)] font-[var(--gf-fw-bold)] text-brand-gradient">
      登录
    </h1>
    <p class="text-secondary text-sm">LoginView 占位</p>
    <p v-if="route.query.redirect" class="text-muted text-sm">
      登录后将跳转至：{{ route.query.redirect }}
    </p>
  </div>
</template>
