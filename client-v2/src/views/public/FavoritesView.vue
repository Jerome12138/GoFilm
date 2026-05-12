<script setup lang="ts">
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useFavoriteStore, useUserStore } from '@/stores'
import BaseImage from '@/components/base/BaseImage.vue'
import BaseEmpty from '@/components/base/BaseEmpty.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'
import BaseTag from '@/components/base/BaseTag.vue'

const favoriteStore = useFavoriteStore()
const userStore = useUserStore()
const { list, remoteMode, remoteLoading } = storeToRefs(favoriteStore)
const { isLoggedIn } = storeToRefs(userStore)

const items = computed(() => list.value)
const sourceLabel = computed(() =>
  remoteMode.value ? '云端收藏 · 跨设备同步' : '本地收藏 · 仅当前浏览器'
)

function detailLink(id: string): { path: string; query: { link: string } } {
  return { path: '/filmDetail', query: { link: id } }
}

function handleRemove(id: string, e: Event): void {
  e.preventDefault()
  e.stopPropagation()
  void favoriteStore.remove(id)
}
</script>

<template>
  <section class="px-[var(--gf-space-4)] md:px-[var(--gf-space-6)] py-[var(--gf-space-6)]">
    <header class="flex items-center justify-between mb-[var(--gf-space-5)] flex-wrap gap-[var(--gf-space-3)]">
      <div>
        <h1 class="text-[var(--gf-fs-2xl)] font-[var(--gf-fw-bold)]">我的收藏</h1>
        <p class="text-sm text-muted mt-[var(--gf-space-1)] flex items-center gap-[var(--gf-space-2)] flex-wrap">
          <BaseTag :variant="remoteMode ? 'purple' : 'default'" size="xs">
            {{ remoteMode ? '云端' : '本地' }}
          </BaseTag>
          <span>{{ sourceLabel }}</span>
          <span>·</span>
          <span>共 {{ items.length }} 部</span>
          <span v-if="remoteLoading" class="text-link">同步中…</span>
        </p>
      </div>
      <RouterLink
        v-if="!isLoggedIn"
        to="/login"
        class="gf-link-btn"
      >
        登录以云端同步
      </RouterLink>
    </header>

    <BaseEmpty
      v-if="!items.length"
      title="还没有收藏内容"
      description="在影片详情页点击「收藏」即可加入这里"
    />

    <div
      v-else
      class="grid gap-[var(--gf-space-4)] grid-cols-[repeat(auto-fill,minmax(150px,1fr))] md:grid-cols-[repeat(auto-fill,minmax(180px,1fr))]"
    >
      <RouterLink
        v-for="record in items"
        :key="record.id"
        :to="detailLink(record.id)"
        class="gf-fav-card group block"
        data-focusable="true"
        tabindex="0"
        :aria-label="record.name"
      >
        <div class="relative overflow-hidden rounded-[var(--gf-radius-lg)] shadow-card aspect-[2/3] bg-elevated">
          <BaseImage
            :src="record.picture || ''"
            :alt="record.name"
            ratio="2/3"
            fit="cover"
          />

          <button
            type="button"
            class="absolute top-[var(--gf-space-2)] right-[var(--gf-space-2)] z-2 w-[24px] h-[24px] rounded-full bg-[rgba(0,0,0,0.6)] hover:bg-[rgba(0,0,0,0.85)] flex-center text-white transition-colors"
            :aria-label="`取消收藏 ${record.name}`"
            @click="handleRemove(record.id, $event)"
          >
            <BaseIcon name="close" size="14px" />
          </button>

          <div class="absolute inset-0 bg-[linear-gradient(180deg,transparent_50%,rgba(0,0,0,0.85)_100%)] opacity-0 group-hover:opacity-100 group-focus-visible:opacity-100 transition-opacity" />
        </div>

        <div class="mt-[var(--gf-space-2)]">
          <h3 class="text-[var(--gf-fs-sm)] font-[var(--gf-fw-medium)] text-primary line-clamp-1">
            {{ record.name }}
          </h3>
          <p v-if="record.remarks" class="text-[var(--gf-fs-xs)] text-muted mt-[2px] line-clamp-1">
            {{ record.remarks }}
          </p>
        </div>
      </RouterLink>
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
.gf-fav-card {
  text-decoration: none;
  outline: none;
  transition: transform var(--gf-dur-base) var(--gf-ease-spring);
}
.gf-fav-card:focus-visible > div:first-child {
  box-shadow: var(--gf-shadow-focus-ring), var(--gf-shadow-hover);
}
@media (hover: hover) and (pointer: fine) {
  .gf-fav-card:hover > div:first-child {
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
</style>
