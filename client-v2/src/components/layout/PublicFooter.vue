<script setup lang="ts">
import { useSiteStore } from '@/stores'
import { storeToRefs } from 'pinia'

const siteStore = useSiteStore()
const { basic } = storeToRefs(siteStore)
</script>

<template>
  <footer
    class="mt-[var(--gf-space-12)] border-t border-subtle bg-surface text-secondary text-sm"
  >
    <div
      class="container-page py-[var(--gf-space-8)] flex flex-col md:flex-row items-center justify-between gap-[var(--gf-space-3)]"
    >
      <span>{{ basic?.siteName || 'GoFilm' }} &copy; {{ new Date().getFullYear() }}</span>
      <span v-if="basic?.filing" class="text-muted">{{ basic.filing }}</span>
    </div>
  </footer>
</template>
