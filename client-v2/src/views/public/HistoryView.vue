<script setup lang="ts">
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useHistoryStore, useUserStore } from '@/stores'
import BaseImage from '@/components/base/BaseImage.vue'
import BaseEmpty from '@/components/base/BaseEmpty.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'
import BaseTag from '@/components/base/BaseTag.vue'
import { confirm } from '@/composables/useConfirm'
import {
  groupByTimeBucket,
  progressPercent
} from '@/composables/useTimeBucket'

const historyStore = useHistoryStore()
const userStore = useUserStore()
const { list, remoteMode, remoteLoading } = storeToRefs(historyStore)
const { isLoggedIn } = storeToRefs(userStore)

const items = computed(() => list.value)

/** 按时间分桶 (今天/本周/本月/更早) */
const groups = computed(() => groupByTimeBucket(items.value))
const sourceLabel = computed(() =>
  remoteMode.value ? '云端历史 · 跨设备同步' : '本地历史 · 仅当前浏览器'
)

function formatTime(ts: number): string {
  if (!ts) return ''
  const d = new Date(ts)
  const now = Date.now()
  const diff = now - ts
  const day = 24 * 60 * 60 * 1000
  if (diff < day) {
    return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
  }
  if (diff < 7 * day) {
    return `${Math.floor(diff / day)} 天前`
  }
  return `${d.getMonth() + 1}/${d.getDate()}`
}

function formatProgress(seconds?: number): string {
  if (!seconds || seconds < 1) return ''
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${s.toString().padStart(2, '0')}`
}

async function handleClear(): Promise<void> {
  if (!items.value.length) return
  const ok = await confirm({
    title: '确认清空全部观看历史？',
    desc: '此操作不可恢复',
    okText: '清空',
    danger: true
  })
  if (ok) {
    historyStore.clear()
  }
}

function handleRemove(id: string, e: Event): void {
  e.preventDefault()
  e.stopPropagation()
  historyStore.remove(id)
}
</script>

<template>
  <section class="px-[var(--gf-space-4)] md:px-[var(--gf-space-6)] py-[var(--gf-space-6)]">
    <header class="flex items-center justify-between mb-[var(--gf-space-5)] flex-wrap gap-[var(--gf-space-3)]">
      <div>
        <h1 class="text-[var(--gf-fs-2xl)] font-[var(--gf-fw-bold)]">观看历史</h1>
        <p class="text-sm text-muted mt-[var(--gf-space-1)] flex items-center gap-[var(--gf-space-2)] flex-wrap">
          <BaseTag :variant="remoteMode ? 'purple' : 'default'" size="xs">
            {{ remoteMode ? '云端' : '本地' }}
          </BaseTag>
          <span>{{ sourceLabel }}</span>
          <span>·</span>
          <span>共 {{ items.length }} 条</span>
          <span v-if="remoteLoading" class="text-link">同步中…</span>
        </p>
      </div>
      <div class="flex items-center gap-[var(--gf-space-2)]">
        <RouterLink
          v-if="!isLoggedIn"
          to="/login"
          class="gf-link-btn"
        >
          登录以云端同步
        </RouterLink>
        <BaseButton
          v-if="items.length"
          variant="ghost"
          size="sm"
          @click="handleClear"
        >
          <BaseIcon name="trash" size="16px" />
          清空
        </BaseButton>
      </div>
    </header>

    <BaseEmpty
      v-if="!items.length"
      title="还没有观看记录"
      description="去首页找一部喜欢的影片开始观看吧"
    />

    <div v-else class="flex flex-col gap-[var(--gf-space-8)]">
      <section
        v-for="group in groups"
        :key="group.bucket"
        class="flex flex-col gap-[var(--gf-space-4)]"
      >
        <h2 class="gf-history-group__title flex items-baseline gap-[var(--gf-space-2)]">
          <span class="text-[var(--gf-fs-lg)] font-[var(--gf-fw-bold)] text-primary">
            {{ group.label }}
          </span>
          <span class="text-[var(--gf-fs-xs)] text-muted">
            {{ group.items.length }} 条
          </span>
        </h2>
        <div
          class="grid gap-[var(--gf-space-4)] grid-cols-[repeat(auto-fill,minmax(150px,1fr))] md:grid-cols-[repeat(auto-fill,minmax(180px,1fr))]"
        >
          <RouterLink
            v-for="record in group.items"
            :key="record.id"
            :to="record.link"
            class="gf-history-card group block"
            data-focusable="true"
            tabindex="0"
            :aria-label="record.name"
          >
            <div class="relative overflow-hidden rounded-[var(--gf-radius-lg)] shadow-card aspect-[3/4] bg-elevated">
              <BaseImage
                :src="record.picture || ''"
                :alt="record.name"
                ratio="3/4"
                fit="cover"
              />

              <BaseTag
                v-if="record.episode"
                variant="brand"
                size="xs"
                class="absolute top-[var(--gf-space-2)] left-[var(--gf-space-2)] z-2"
              >
                {{ record.episode }}
              </BaseTag>

              <button
                type="button"
                class="absolute top-[var(--gf-space-2)] right-[var(--gf-space-2)] z-2 w-[24px] h-[24px] rounded-full bg-[rgba(0,0,0,0.6)] hover:bg-[rgba(0,0,0,0.85)] flex-center text-white transition-colors"
                :aria-label="`从历史中移除 ${record.name}`"
                @click="handleRemove(record.id, $event)"
              >
                <BaseIcon name="close" size="14px" />
              </button>

              <div
                v-if="formatProgress(record.currentTime)"
                class="absolute bottom-[8px] right-[var(--gf-space-2)] px-[6px] py-[2px] rounded-[var(--gf-radius-sm)] bg-[rgba(0,0,0,0.7)] text-white text-[var(--gf-fs-xs)] z-2"
              >
                {{ formatProgress(record.currentTime) }}
              </div>

              <!-- 进度条 (基于 currentTime / duration), 卡片底部 4px 横条 -->
              <div
                v-if="progressPercent(record.currentTime, record.duration) > 0"
                class="gf-history-progress"
                :aria-label="`已观看 ${progressPercent(record.currentTime, record.duration)}%`"
              >
                <span
                  class="gf-history-progress__fill"
                  :style="{ width: progressPercent(record.currentTime, record.duration) + '%' }"
                />
              </div>

              <div class="absolute inset-0 bg-[linear-gradient(180deg,transparent_50%,rgba(0,0,0,0.85)_100%)] opacity-0 group-hover:opacity-100 group-focus-visible:opacity-100 transition-opacity" />

              <div class="absolute inset-x-0 bottom-0 px-[var(--gf-space-3)] pb-[var(--gf-space-3)] pt-[var(--gf-space-5)] z-2 opacity-0 group-hover:opacity-100 group-focus-visible:opacity-100 transition-opacity">
                <div class="text-white text-[var(--gf-fs-sm)] font-[var(--gf-fw-semibold)] flex items-center gap-[var(--gf-space-1)]">
                  <BaseIcon name="play" size="14px" />
                  继续观看
                </div>
              </div>
            </div>

            <div class="mt-[var(--gf-space-2)]">
              <h3 class="text-[var(--gf-fs-sm)] font-[var(--gf-fw-medium)] text-primary line-clamp-1">
                {{ record.name }}
              </h3>
              <p class="text-[var(--gf-fs-xs)] text-muted mt-[2px]">
                {{ formatTime(record.timeStamp) }}
              </p>
            </div>
          </RouterLink>
        </div>
      </section>
    </div>
  </section>
</template>

<style scoped>
.gf-link-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: var(--gf-radius-sm);
  background-color: rgba(155, 73, 231, 0.16);
  color: var(--gf-text-link);
  font-size: var(--gf-fs-sm);
  text-decoration: none;
  transition: background-color var(--gf-dur-fast) var(--gf-ease-standard);
}
.gf-link-btn:hover,
.gf-link-btn:focus-visible {
  background-color: rgba(155, 73, 231, 0.28);
  outline: none;
}

.gf-history-card {
  text-decoration: none;
  outline: none;
  transition: transform var(--gf-dur-base) var(--gf-ease-spring);
}
.gf-history-card:focus-visible {
  outline: none;
}
.gf-history-card:focus-visible > div:first-child {
  box-shadow: var(--gf-shadow-focus-ring), var(--gf-shadow-hover);
}
@media (hover: hover) and (pointer: fine) {
  .gf-history-card:hover > div:first-child {
    transform: scale(1.04);
    box-shadow: var(--gf-shadow-hover);
  }
}
.line-clamp-1 {
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* 进度条: 卡片底部 4px 横条 */
.gf-history-progress {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 3px;
  background-color: var(--gf-progress-bg);
  z-index: 2;
  overflow: hidden;
}
.gf-history-progress__fill {
  display: block;
  height: 100%;
  background-image: var(--gf-progress-fg);
  border-top-right-radius: 2px;
  border-bottom-right-radius: 2px;
  transition: width var(--gf-dur-base) var(--gf-ease-standard);
}
</style>
