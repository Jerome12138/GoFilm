<script setup lang="ts">
import ManageHeader from './ManageHeader.vue'
import ManageSidebar from './ManageSidebar.vue'
</script>

<template>
  <div class="min-h-screen flex flex-col bg-base text-primary" data-mode="desktop">
    <ManageHeader />
    <div class="flex-1 flex">
      <ManageSidebar />
      <main class="flex-1 p-[var(--gf-space-6)] overflow-x-auto">
        <slot />
      </main>
    </div>
  </div>
</template>
