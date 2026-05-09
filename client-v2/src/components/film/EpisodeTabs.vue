<script setup lang="ts">
import { computed } from 'vue'
import type { PlaySource } from '@/types/film'

interface Props {
  sources: PlaySource[]
  currentSourceId?: string
  /** 当前集的 link */
  currentEpisode?: string
  /** 已观看集合（cookie 历史记录），按 link */
  watchedLinks?: string[]
}

const props = withDefaults(defineProps<Props>(), {
  currentSourceId: '',
  currentEpisode: '',
  watchedLinks: () => []
})

const emit = defineEmits<{
  (e: 'select', payload: { sourceId: string; episodeIndex: number; link: string }): void
  (e: 'change-source', sourceId: string): void
}>()

const activeSourceId = computed(() => {
  if (props.currentSourceId) return props.currentSourceId
  return props.sources[0]?.id ?? ''
})

const activeSource = computed(() =>
  props.sources.find((s) => s.id === activeSourceId.value) ??
  props.sources[0]
)

function selectSource(id: string): void {
  if (id === activeSourceId.value) return
  emit('change-source', id)
}

function selectEpisode(idx: number): void {
  const src = activeSource.value
  if (!src) return
  const ep = src.linkList[idx]
  if (!ep) return
  emit('select', { sourceId: src.id, episodeIndex: idx, link: ep.link })
}
</script>

<template>
  <section class="gf-episodes flex flex-col gap-[var(--gf-space-4)]">
    <!-- 播放源 Tab -->
    <div
      v-if="sources.length > 1"
      class="gf-source-tabs flex items-center gap-[var(--gf-space-6)] border-b border-default overflow-x-auto"
    >
      <button
        v-for="s in sources"
        :key="s.id"
        class="gf-source-tab"
        :class="s.id === activeSourceId ? 'gf-source-tab--active' : ''"
        data-focusable="true"
        tabindex="0"
        :aria-selected="s.id === activeSourceId"
        @click="selectSource(s.id)"
      >
        {{ s.name }}
      </button>
    </div>

    <!-- 集数网格 -->
    <div v-if="activeSource && activeSource.linkList.length" class="gf-episode-grid">
      <button
        v-for="(ep, idx) in activeSource.linkList"
        :key="ep.link + '-' + idx"
        class="gf-episode-chip"
        :class="[
          ep.link === currentEpisode ? 'gf-episode-chip--active' : '',
          watchedLinks.includes(ep.link) ? 'gf-episode-chip--watched' : ''
        ]"
        data-focusable="true"
        tabindex="0"
        :aria-current="ep.link === currentEpisode ? 'true' : undefined"
        @click="selectEpisode(idx)"
      >
        <span class="gf-episode-chip__label">{{ ep.episode }}</span>
        <span
          v-if="watchedLinks.includes(ep.link) && ep.link !== currentEpisode"
          class="gf-episode-chip__dot"
          aria-hidden="true"
        />
      </button>
    </div>
  </section>
</template>

<style scoped>
.gf-source-tab {
  position: relative;
  background: transparent;
  border: none;
  height: 48px;
  padding: 0 var(--gf-space-1);
  color: var(--gf-text-secondary);
  font-size: var(--gf-fs-md);
  font-weight: var(--gf-fw-medium);
  cursor: pointer;
  white-space: nowrap;
  min-height: 44px;
  transition: color var(--gf-dur-fast) var(--gf-ease-standard);
}

.gf-source-tab:hover {
  color: var(--gf-text-primary);
}

.gf-source-tab--active {
  color: var(--gf-text-primary);
  font-weight: var(--gf-fw-semibold);
}

.gf-source-tab--active::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: -1px;
  height: 2px;
  background-image: var(--gf-brand-gradient);
  border-radius: 2px;
}

.gf-episode-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: var(--gf-space-2);
}
@media (min-width: 480px) {
  .gf-episode-grid {
    grid-template-columns: repeat(5, minmax(0, 1fr));
  }
}
@media (min-width: 768px) {
  .gf-episode-grid {
    grid-template-columns: repeat(6, minmax(0, 1fr));
    gap: var(--gf-space-3);
  }
}
@media (min-width: 1024px) {
  .gf-episode-grid {
    grid-template-columns: repeat(8, minmax(0, 1fr));
  }
}
@media (min-width: 1440px) {
  .gf-episode-grid {
    grid-template-columns: repeat(10, minmax(0, 1fr));
  }
}

.gf-episode-chip {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 48px;
  padding: 0 var(--gf-space-2);
  border-radius: var(--gf-radius-md);
  background-color: var(--gf-bg-elevated);
  color: var(--gf-text-secondary);
  font-size: var(--gf-fs-sm);
  font-weight: var(--gf-fw-semibold);
  border: none;
  cursor: pointer;
  min-height: 44px;
  transition:
    background-color var(--gf-dur-fast) var(--gf-ease-standard),
    color var(--gf-dur-fast) var(--gf-ease-standard),
    transform var(--gf-dur-fast) var(--gf-ease-standard);
  overflow: hidden;
}

@media (min-width: 768px) {
  .gf-episode-chip {
    height: 52px;
  }
}
@media (min-width: 1024px) {
  .gf-episode-chip {
    height: 56px;
  }
}

.gf-episode-chip:hover {
  background-color: rgba(255, 255, 255, 0.08);
  color: var(--gf-text-primary);
}

.gf-episode-chip--active {
  background-image: var(--gf-brand-gradient);
  color: #fff;
  box-shadow: var(--gf-shadow-purple-glow);
}

.gf-episode-chip--watched .gf-episode-chip__dot {
  position: absolute;
  top: 6px;
  left: 6px;
  width: 6px;
  height: 6px;
  border-radius: 9999px;
  background-color: var(--gf-success);
}

.gf-episode-chip__label {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>

<style>
[data-mode='tv'] .gf-episode-grid {
  grid-template-columns: repeat(10, minmax(0, 1fr));
  gap: var(--gf-space-4);
}
[data-mode='tv'] .gf-episode-chip {
  height: 64px;
  font-size: var(--gf-fs-base);
}
[data-mode='tv'] .gf-episode-chip:focus,
[data-mode='tv'] .gf-episode-chip:focus-visible {
  outline: none;
  box-shadow: 0 0 0 4px var(--gf-brand-cyan);
  background-color: rgba(255, 255, 255, 0.12);
  color: var(--gf-text-primary);
}
[data-mode='tv'] .gf-source-tab {
  height: 64px;
  font-size: var(--gf-fs-lg);
  padding: 0 var(--gf-space-3);
}
[data-mode='tv'] .gf-source-tab:focus,
[data-mode='tv'] .gf-source-tab:focus-visible {
  outline: none;
  box-shadow: 0 0 0 4px var(--gf-brand-cyan);
  border-radius: var(--gf-radius-sm);
  color: var(--gf-text-primary);
}
</style>
