<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { PlaySource } from '@/types/film'

interface Props {
  sources: PlaySource[]
  currentSourceId?: string
  /** 当前集的 link */
  currentEpisode?: string
  /** 已观看集合（cookie 历史记录），按 link */
  watchedLinks?: string[]
  /** 单个分段大小, 超过则启用分段切换. 0 = 不分段 */
  pageSize?: number
}

const props = withDefaults(defineProps<Props>(), {
  currentSourceId: '',
  currentEpisode: '',
  watchedLinks: () => [],
  pageSize: 30
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

/** ===== 分段切换 (>30 集时显示) ===== */
const totalCount = computed(() => activeSource.value?.linkList.length ?? 0)
const needsSegments = computed(() => props.pageSize > 0 && totalCount.value > props.pageSize)
const segments = computed<Array<{ start: number; end: number; label: string }>>(() => {
  if (!needsSegments.value) return []
  const size = props.pageSize
  const out: Array<{ start: number; end: number; label: string }> = []
  for (let i = 0; i < totalCount.value; i += size) {
    const end = Math.min(i + size, totalCount.value) - 1
    out.push({ start: i, end, label: `${i + 1}-${end + 1}` })
  }
  return out
})
const segmentIndex = ref(0)

/** 切源时, 段索引重置, 但优先定位到包含当前播放集的段 */
watch([activeSourceId, () => props.currentEpisode], () => {
  if (!needsSegments.value) {
    segmentIndex.value = 0
    return
  }
  const src = activeSource.value
  if (!src) return
  const curIdx = src.linkList.findIndex((e) => e.link === props.currentEpisode)
  if (curIdx >= 0) {
    segmentIndex.value = Math.floor(curIdx / props.pageSize)
  } else {
    segmentIndex.value = 0
  }
}, { immediate: true })

/** 当前段范围内的 episodes (含原索引) */
const visibleEpisodes = computed(() => {
  const src = activeSource.value
  if (!src) return []
  if (!needsSegments.value) {
    return src.linkList.map((ep, idx) => ({ ep, idx }))
  }
  const seg = segments.value[segmentIndex.value]
  if (!seg) return []
  return src.linkList.slice(seg.start, seg.end + 1).map((ep, i) => ({
    ep,
    idx: seg.start + i
  }))
})

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

function selectSegment(i: number): void {
  segmentIndex.value = i
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

    <!-- 分段切换 (集数 > pageSize 时显示) -->
    <div
      v-if="needsSegments"
      class="gf-episode-segments flex flex-wrap gap-[var(--gf-space-2)]"
      role="tablist"
      aria-label="集数分段"
    >
      <button
        v-for="(seg, i) in segments"
        :key="i"
        class="gf-episode-seg"
        :class="i === segmentIndex ? 'gf-episode-seg--active' : ''"
        data-focusable="true"
        tabindex="0"
        :aria-selected="i === segmentIndex"
        role="tab"
        @click="selectSegment(i)"
      >
        {{ seg.label }}
      </button>
    </div>

    <!-- 集数网格 -->
    <div v-if="visibleEpisodes.length" class="gf-episode-grid">
      <button
        v-for="{ ep, idx } in visibleEpisodes"
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

/* 分段 chip (1-30 / 31-60 ...) */
.gf-episode-seg {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: var(--gf-chip-height, 32px);
  padding: 0 var(--gf-chip-padding-x, 14px);
  border-radius: var(--gf-chip-radius, 9999px);
  background-color: var(--gf-bg-elevated);
  color: var(--gf-text-secondary);
  font-size: var(--gf-fs-sm);
  font-weight: var(--gf-fw-medium);
  border: 1px solid transparent;
  cursor: pointer;
  transition:
    background-color var(--gf-dur-fast) var(--gf-ease-standard),
    color var(--gf-dur-fast) var(--gf-ease-standard);
}
.gf-episode-seg:hover {
  background-color: rgba(255, 255, 255, 0.08);
  color: var(--gf-text-primary);
}
.gf-episode-seg--active {
  background-image: var(--gf-brand-gradient);
  color: #fff;
  border-color: transparent;
  box-shadow: var(--gf-shadow-purple-glow);
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
