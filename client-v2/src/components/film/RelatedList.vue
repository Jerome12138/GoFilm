<script setup lang="ts">
import type { FilmListItem } from '@/types/film'
import FilmCard from './FilmCard.vue'

interface Props {
  items: FilmListItem[]
  title?: string
}

withDefaults(defineProps<Props>(), {
  title: '相关推荐'
})
</script>

<template>
  <section
    v-if="items.length"
    class="gf-related-list flex flex-col gap-[var(--gf-space-4)]"
  >
    <h2
      v-if="title"
      class="text-[var(--gf-fs-xl)] font-[var(--gf-fw-bold)] text-primary"
    >
      {{ title }}
    </h2>
    <div class="gf-related-list__grid">
      <FilmCard
        v-for="(item, idx) in items"
        :key="(item.id ?? idx) + '-' + idx"
        :item="item"
        :show-title-below="true"
      />
    </div>
  </section>
</template>

<style scoped>
.gf-related-list__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--gf-space-3);
}
@media (min-width: 768px) {
  .gf-related-list__grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: var(--gf-space-4);
  }
}
@media (min-width: 1024px) {
  .gf-related-list__grid {
    grid-template-columns: repeat(6, minmax(0, 1fr));
  }
}
@media (min-width: 1440px) {
  .gf-related-list__grid {
    grid-template-columns: repeat(6, minmax(0, 1fr));
  }
}
@media (min-width: 1920px) {
  .gf-related-list__grid {
    grid-template-columns: repeat(8, minmax(0, 1fr));
  }
}

</style>

<style>
[data-mode='tv'] .gf-related-list__grid {
  grid-template-columns: repeat(8, minmax(0, 1fr));
}
</style>
