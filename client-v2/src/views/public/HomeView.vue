<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { filmApi } from '@/api'
import HeroCarousel from '@/components/film/HeroCarousel.vue'
import FilmRow from '@/components/film/FilmRow.vue'
import FilmCard from '@/components/film/FilmCard.vue'
import BaseSkeleton from '@/components/base/BaseSkeleton.vue'
import BaseEmpty from '@/components/base/BaseEmpty.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import type { FilmListItem, IndexPageData } from '@/types/film'

/**
 * 首页 — STORY-008
 * - 调 GET /api/index 拿 { banner, content[] }
 * - HeroCarousel：banner 优先，回退 content[0].movies.slice(0, 5)
 * - 按 content[i] 渲染若干 FilmRow（title=nav.name，items=movies）
 * - PC 大屏右侧栏显示 hot 前 12 条（取 content[0].hot，兜底合并）
 * - 加载中：HeroCarousel 区骨架 + 多行骨架
 * - 失败：整页 BaseEmpty
 */

interface IndexState {
  loading: boolean
  errored: boolean
  data: IndexPageData | null
}

const state = ref<IndexState>({
  loading: true,
  errored: false,
  data: null
})

const heroItems = computed<FilmListItem[]>(() => {
  const data = state.value.data
  if (!data) return []
  if (data.banner && data.banner.length > 0) return data.banner
  const first = data.content?.[0]?.movies ?? []
  return first.slice(0, 5)
})

/** 旁栏热播：取所有 content 段的 hot 合并去重，最多 12 条 */
const hotSidebar = computed<FilmListItem[]>(() => {
  const data = state.value.data
  if (!data) return []
  const merged: FilmListItem[] = []
  const seen = new Set<string>()
  for (const block of data.content || []) {
    for (const item of block.hot || []) {
      const key = String(item.id ?? item.mid ?? item.name)
      if (!seen.has(key)) {
        seen.add(key)
        merged.push(item)
        if (merged.length >= 12) break
      }
    }
    if (merged.length >= 12) break
  }
  return merged
})

const rows = computed(() => {
  const data = state.value.data
  if (!data) return []
  return (data.content || [])
    .filter((b) => b && b.movies && b.movies.length > 0)
    .map((b) => ({
      pid: b.nav?.id ?? 0,
      title: b.nav?.name ?? '推荐',
      items: b.movies
    }))
})

/** 分类入口 chip: 由 content 数组派生 (电影/剧集/综艺/动漫/纪录片) */
const categoryChips = computed<Array<{ id: number; name: string }>>(() => {
  const data = state.value.data
  if (!data) return []
  return (data.content || [])
    .filter((b) => b.nav?.id && b.nav?.name)
    .map((b) => ({ id: b.nav!.id, name: b.nav!.name }))
})

/** 猜你喜欢: 合并所有 movies 去重, 取 24 条作为瀑布流 */
const recommendGrid = computed<FilmListItem[]>(() => {
  const data = state.value.data
  if (!data) return []
  const merged: FilmListItem[] = []
  const seen = new Set<string>()
  for (const block of data.content || []) {
    for (const item of block.movies || []) {
      const key = String(item.id ?? item.mid ?? item.name)
      if (!seen.has(key)) {
        seen.add(key)
        merged.push(item)
      }
    }
  }
  // 简单随机化, 避免每次相同顺序
  return merged.slice().sort(() => Math.random() - 0.5).slice(0, 24)
})

async function loadIndex(): Promise<void> {
  state.value.loading = true
  state.value.errored = false
  try {
    const data = await filmApi.getIndex()
    state.value = { loading: false, errored: false, data }
  } catch {
    state.value = { loading: false, errored: true, data: null }
  }
}

onMounted(() => {
  loadIndex()
})
</script>

<template>
  <div class="gf-home flex flex-col">
    <!-- 加载骨架 -->
    <template v-if="state.loading">
      <div class="gf-home__hero-skeleton">
        <BaseSkeleton shape="rect" width="100%" height="45vh" />
      </div>
      <div class="container-page py-[var(--gf-space-8)] flex flex-col gap-[var(--gf-space-8)]">
        <div v-for="i in 3" :key="i" class="flex flex-col gap-[var(--gf-space-3)]">
          <BaseSkeleton shape="text" width="160px" height="24px" />
          <div class="gf-home__row-skeleton">
            <BaseSkeleton
              v-for="j in 7"
              :key="j"
              shape="rect"
              ratio="3/4"
              width="100%"
            />
          </div>
        </div>
      </div>
    </template>

    <!-- 错误态 -->
    <template v-else-if="state.errored">
      <div class="container-page py-[var(--gf-space-12)]">
        <BaseEmpty
          title="加载失败"
          description="无法获取首页数据，请稍后重试或检查网络。"
        >
          <template #action>
            <BaseButton variant="primary" size="md" @click="loadIndex">
              重新加载
            </BaseButton>
          </template>
        </BaseEmpty>
      </div>
    </template>

    <!-- 正常 -->
    <template v-else-if="state.data">
      <HeroCarousel v-if="heroItems.length" :items="heroItems" />

      <!-- 分类入口 chip 行 (hero 下方, bilibili 风格快速跳转) -->
      <nav
        v-if="categoryChips.length"
        class="gf-home__cat-chips container-page"
        aria-label="分类入口"
      >
        <RouterLink
          v-for="c in categoryChips"
          :key="c.id"
          :to="{ path: '/filmClassify', query: { Pid: c.id } }"
          class="gf-home__cat-chip"
          data-focusable="true"
          tabindex="0"
        >
          {{ c.name }}
        </RouterLink>
      </nav>

      <!-- 桌面（lg+）双列：左 rows / 右 hot -->
      <div class="gf-home__main">
        <div class="gf-home__rows">
          <FilmRow
            v-for="row in rows"
            :key="row.pid + '-' + row.title"
            :title="row.title"
            :more-link="{ path: '/filmClassify', query: { Pid: row.pid } }"
            :items="row.items"
          />
          <BaseEmpty
            v-if="!rows.length"
            title="暂无内容"
            description="后端尚未返回分类影片列表。"
          />
        </div>

        <aside v-if="hotSidebar.length" class="gf-home__aside">
          <h2 class="gf-home__aside-title">热播榜</h2>
          <ol class="gf-home__hot-list">
            <li
              v-for="(item, idx) in hotSidebar"
              :key="String(item.id ?? item.mid ?? idx) + '-' + idx"
              class="gf-home__hot-item"
            >
              <span
                class="gf-home__hot-rank"
                :class="idx < 3 ? 'gf-home__hot-rank--top' : ''"
              >
                {{ idx + 1 }}
              </span>
              <FilmCard
                :item="item"
                :show-title-below="false"
                class="gf-home__hot-card"
              />
              <div class="gf-home__hot-meta">
                <span class="gf-home__hot-name">{{ item.name }}</span>
                <span v-if="item.remarks" class="gf-home__hot-remarks">
                  {{ item.remarks }}
                </span>
              </div>
            </li>
          </ol>
        </aside>
      </div>

      <!-- 猜你喜欢瀑布流 (bilibili 风格底部推荐) -->
      <section
        v-if="recommendGrid.length"
        class="gf-home__recommend container-page"
        aria-label="猜你喜欢"
      >
        <header class="gf-home__recommend-header">
          <h2 class="gf-home__recommend-title">猜你喜欢</h2>
          <span class="gf-home__recommend-tip">基于浏览数据混合推荐</span>
        </header>
        <div class="gf-home__recommend-grid">
          <FilmCard
            v-for="(item, idx) in recommendGrid"
            :key="String(item.id ?? item.mid ?? idx) + '-' + idx"
            :item="item"
            :show-title-below="true"
          />
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.gf-home {
  width: 100%;
}

.gf-home__hero-skeleton {
  width: 100%;
}

.gf-home__row-skeleton {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--gf-space-3);
}

/* 分类入口 chip 行 (hero 下方) */
.gf-home__cat-chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--gf-space-2);
  padding-block: var(--gf-space-5) var(--gf-space-3);
}

.gf-home__cat-chip {
  display: inline-flex;
  align-items: center;
  height: var(--gf-chip-height);
  padding: 0 var(--gf-chip-padding-x);
  border-radius: var(--gf-chip-radius);
  background-color: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: var(--gf-text-secondary);
  font-size: var(--gf-fs-sm);
  font-weight: var(--gf-fw-medium);
  text-decoration: none;
  transition:
    background-color var(--gf-dur-fast) var(--gf-ease-standard),
    color var(--gf-dur-fast) var(--gf-ease-standard),
    border-color var(--gf-dur-fast) var(--gf-ease-standard);
  outline: none;
}

.gf-home__cat-chip:hover,
.gf-home__cat-chip:focus-visible {
  background-color: rgba(255, 255, 255, 0.12);
  border-color: rgba(255, 255, 255, 0.2);
  color: var(--gf-text-primary);
}

.gf-home__cat-chip.router-link-active {
  background-image: var(--gf-brand-gradient);
  border-color: transparent;
  color: #fff;
}

/* 猜你喜欢瀑布流 */
.gf-home__recommend {
  padding-block: var(--gf-space-8) var(--gf-space-16);
}

.gf-home__recommend-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: var(--gf-space-5);
  gap: var(--gf-space-3);
}

.gf-home__recommend-title {
  font-size: var(--gf-fs-xl);
  font-weight: var(--gf-fw-bold);
  color: var(--gf-text-primary);
}

.gf-home__recommend-tip {
  font-size: var(--gf-fs-xs);
  color: var(--gf-text-muted);
}

.gf-home__recommend-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--gf-card-gap);
}

@media (min-width: 480px) {
  .gf-home__recommend-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (min-width: 768px) {
  .gf-home__recommend-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (min-width: 1024px) {
  .gf-home__recommend-grid {
    grid-template-columns: repeat(5, minmax(0, 1fr));
  }
}

@media (min-width: 1440px) {
  .gf-home__recommend-grid {
    grid-template-columns: repeat(6, minmax(0, 1fr));
  }
}

@media (min-width: 768px) {
  .gf-home__row-skeleton {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (min-width: 1024px) {
  .gf-home__row-skeleton {
    grid-template-columns: repeat(7, minmax(0, 1fr));
  }
}

/* 主内容布局 */
.gf-home__main {
  display: flex;
  flex-direction: column;
  gap: var(--gf-space-8);
  padding-block: var(--gf-space-8) var(--gf-space-12);
}

@media (min-width: 1024px) {
  .gf-home__main {
    flex-direction: row;
    align-items: flex-start;
    padding-inline: var(--gf-gutter-desktop);
    max-width: var(--gf-container-max);
    margin-inline: auto;
    width: 100%;
    gap: var(--gf-space-8);
  }
}

@media (min-width: 1920px) {
  .gf-home__main {
    max-width: var(--gf-container-max-2xl);
  }
}

.gf-home__rows {
  display: flex;
  flex-direction: column;
  gap: var(--gf-space-8);
  min-width: 0;
  flex: 1;
}

@media (min-width: 768px) {
  .gf-home__rows {
    gap: var(--gf-space-12);
  }
}

/* row 与 hot sidebar 共存时，FilmRow 内部 container-page 会双重 padding；
   在 lg+ 下覆盖 row 的 container 让其与 sidebar 平铺 */
@media (min-width: 1024px) {
  .gf-home__rows :deep(.gf-film-row > header.container-page) {
    padding-inline: 0;
    margin-inline: 0;
  }
  .gf-home__rows :deep(.gf-film-row__edge) {
    width: 0;
  }
}

/* 旁栏 */
.gf-home__aside {
  display: none;
  width: 320px;
  flex-shrink: 0;
  background-color: var(--gf-bg-surface);
  border: 1px solid var(--gf-border-subtle);
  border-radius: var(--gf-radius-lg);
  padding: var(--gf-space-5);
}

@media (min-width: 1024px) {
  .gf-home__aside {
    display: block;
  }
}

.gf-home__aside-title {
  font-size: var(--gf-fs-lg);
  font-weight: var(--gf-fw-bold);
  color: var(--gf-text-primary);
  margin: 0 0 var(--gf-space-4);
}

.gf-home__hot-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--gf-space-3);
}

.gf-home__hot-item {
  display: grid;
  grid-template-columns: 32px 60px 1fr;
  align-items: center;
  gap: var(--gf-space-3);
}

.gf-home__hot-rank {
  font-family: var(--gf-font-display);
  font-size: var(--gf-fs-lg);
  font-weight: var(--gf-fw-black);
  color: var(--gf-text-muted);
  text-align: center;
  line-height: 1;
}

.gf-home__hot-rank--top {
  color: transparent;
  background-image: var(--gf-brand-gradient);
  background-clip: text;
  -webkit-background-clip: text;
}

.gf-home__hot-card {
  width: 60px;
}

.gf-home__hot-meta {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 2px;
}

.gf-home__hot-name {
  font-size: var(--gf-fs-sm);
  font-weight: var(--gf-fw-medium);
  color: var(--gf-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gf-home__hot-remarks {
  font-size: var(--gf-fs-xs);
  color: var(--gf-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>

<style>
/* TV 模式下不显示侧栏（屏幕宽度足够，但旁栏会破坏 10-foot UI 节奏） */
[data-mode='tv'] .gf-home__aside {
  display: none;
}
[data-mode='tv'] .gf-home__main {
  padding-inline: var(--gf-tv-safe);
}
</style>
