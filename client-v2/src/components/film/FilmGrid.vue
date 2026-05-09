<script setup lang="ts">
import type { FilmListItem } from '@/types/film'
import FilmCard from './FilmCard.vue'

interface Props {
  items: FilmListItem[]
  /** 间距（CSS 值），默认按断点 */
  gap?: string
  /** 每个 item 的 key 字段，默认 id */
  itemKey?: keyof FilmListItem
}

const props = withDefaults(defineProps<Props>(), {
  gap: '',
  itemKey: 'id'
})

function getItemKey(item: FilmListItem, idx: number): string | number {
  const v = item[props.itemKey as keyof FilmListItem]
  if (typeof v === 'string' || typeof v === 'number') return v
  return idx
}
</script>

<template>
  <div
    class="gf-film-grid"
    :style="gap ? { gap } : undefined"
  >
    <div
      v-for="(item, idx) in items"
      :key="getItemKey(item, idx)"
      class="gf-film-grid__cell"
    >
      <slot name="item" :item="item" :index="idx">
        <FilmCard :item="item" :show-title-below="true" />
      </slot>
    </div>
  </div>
</template>

<style scoped>
.gf-film-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--gf-space-3);
}
@media (min-width: 480px) {
  .gf-film-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
@media (min-width: 768px) {
  .gf-film-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: var(--gf-space-4);
  }
}
@media (min-width: 1024px) {
  .gf-film-grid {
    grid-template-columns: repeat(5, minmax(0, 1fr));
  }
}
@media (min-width: 1440px) {
  .gf-film-grid {
    grid-template-columns: repeat(6, minmax(0, 1fr));
  }
}
@media (min-width: 1920px) {
  .gf-film-grid {
    grid-template-columns: repeat(8, minmax(0, 1fr));
    gap: var(--gf-space-6);
  }
}

</style>

<style>
[data-mode='tv'] .gf-film-grid {
  grid-template-columns: repeat(8, minmax(0, 1fr));
}
</style>
