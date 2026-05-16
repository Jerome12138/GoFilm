<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { filmApi } from '@/api'
import HeroCarousel from '@/components/film/HeroCarousel.vue'
import FilmRow from '@/components/film/FilmRow.vue'
import FilmCard from '@/components/film/FilmCard.vue'
import BaseImage from '@/components/base/BaseImage.vue'
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

/** 热门榜单 top 10: 取所有 content 段的 hot 合并去重 */
const topRanking = computed<FilmListItem[]>(() => {
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
        if (merged.length >= 10) break
      }
    }
    if (merged.length >= 10) break
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

      <!-- 排行榜模块 (腾讯视频/Netflix Top 10 风格), 横向滚动, 每张卡片带大号排名数字 -->
      <section
        v-if="topRanking.length"
        class="gf-home__ranking container-page"
        aria-label="热门榜单"
      >
        <header class="gf-home__section-header">
          <h2 class="gf-home__section-title">
            <span class="gf-home__section-flame" aria-hidden="true">🔥</span>
            热门榜单
          </h2>
          <span class="gf-home__section-tip">本周播放最多</span>
        </header>
        <div class="gf-home__ranking-scroll">
          <RouterLink
            v-for="(item, idx) in topRanking"
            :key="String(item.id ?? item.mid ?? idx) + '-' + idx"
            :to="{ path: '/filmDetail', query: { link: String(item.id ?? item.mid ?? '') } }"
            class="gf-home__ranking-item"
            :aria-label="`第${idx + 1}名 ${item.name}`"
            data-focusable="true"
          >
            <span
              class="gf-home__ranking-rank"
              :class="idx < 3 ? 'gf-home__ranking-rank--top' : ''"
              aria-hidden="true"
            >
              {{ idx + 1 }}
            </span>
            <div class="gf-home__ranking-poster">
              <BaseImage :src="item.picture" :alt="item.name" ratio="3/4" fit="cover" />
            </div>
            <div class="gf-home__ranking-info">
              <h3 class="gf-home__ranking-name">{{ item.name }}</h3>
              <p v-if="item.remarks || item.cName" class="gf-home__ranking-meta">
                {{ item.remarks || item.cName }}
              </p>
            </div>
          </RouterLink>
        </div>
      </section>

      <!-- 主推荐 rows (横向滚动, 一行 6 卡) -->
      <div class="gf-home__rows container-page">
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

/* 主内容 rows 容器 (单列流式, 不再有 aside) */
.gf-home__rows {
  display: flex;
  flex-direction: column;
  gap: var(--gf-space-8);
  padding-block: var(--gf-space-6) var(--gf-space-8);
}

@media (min-width: 768px) {
  .gf-home__rows {
    gap: var(--gf-space-10);
  }
}

/* ========== 热门榜单模块 (Netflix Top 10 / 腾讯视频热播榜风格) ========== */
.gf-home__ranking {
  padding-block: var(--gf-space-6) var(--gf-space-4);
}

.gf-home__section-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: var(--gf-space-4);
  gap: var(--gf-space-3);
}

.gf-home__section-title {
  font-size: var(--gf-fs-xl);
  font-weight: var(--gf-fw-bold);
  color: var(--gf-text-primary);
  display: inline-flex;
  align-items: center;
  gap: var(--gf-space-2);
  margin: 0;
}

.gf-home__section-flame {
  font-size: 1.1em;
}

.gf-home__section-tip {
  font-size: var(--gf-fs-xs);
  color: var(--gf-text-muted);
}

.gf-home__ranking-scroll {
  display: flex;
  gap: var(--gf-space-3);
  overflow-x: auto;
  scroll-snap-type: x mandatory;
  scrollbar-width: thin;
  padding-block: var(--gf-space-2);
  margin-inline: calc(-1 * var(--gf-gutter-mobile));
  padding-inline: var(--gf-gutter-mobile);
}
@media (min-width: 768px) {
  .gf-home__ranking-scroll {
    gap: var(--gf-space-4);
    margin-inline: calc(-1 * var(--gf-gutter-tablet));
    padding-inline: var(--gf-gutter-tablet);
  }
}
@media (min-width: 1024px) {
  .gf-home__ranking-scroll {
    margin-inline: 0;
    padding-inline: 0;
  }
}

.gf-home__ranking-scroll::-webkit-scrollbar {
  height: 4px;
}
.gf-home__ranking-scroll::-webkit-scrollbar-thumb {
  background-color: rgba(255, 255, 255, 0.18);
  border-radius: 2px;
}

.gf-home__ranking-item {
  flex: 0 0 auto;
  display: grid;
  grid-template-columns: auto 84px 1fr;
  gap: var(--gf-space-3);
  align-items: center;
  width: 280px;
  padding: var(--gf-space-2);
  background-color: var(--gf-bg-surface);
  border: 1px solid var(--gf-border-subtle);
  border-radius: var(--gf-radius-lg);
  text-decoration: none;
  scroll-snap-align: start;
  transition:
    background-color var(--gf-dur-fast) var(--gf-ease-standard),
    transform var(--gf-dur-base) var(--gf-ease-spring);
  outline: none;
}
.gf-home__ranking-item:hover {
  background-color: var(--gf-bg-elevated);
  transform: translateY(-2px);
}
.gf-home__ranking-item:focus-visible {
  box-shadow: var(--gf-shadow-focus-ring);
}
@media (min-width: 768px) {
  .gf-home__ranking-item {
    width: 320px;
  }
}

.gf-home__ranking-rank {
  font-family: var(--gf-font-display);
  font-size: 48px;
  font-weight: 900;
  line-height: 1;
  color: var(--gf-text-muted);
  text-align: center;
  min-width: 48px;
  font-style: italic;
  letter-spacing: -0.04em;
}
.gf-home__ranking-rank--top {
  color: transparent;
  background-image: var(--gf-brand-gradient);
  background-clip: text;
  -webkit-background-clip: text;
}

.gf-home__ranking-poster {
  width: 84px;
  border-radius: var(--gf-radius-md);
  overflow: hidden;
  flex-shrink: 0;
}

.gf-home__ranking-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 4px;
}
.gf-home__ranking-name {
  font-size: var(--gf-fs-sm);
  font-weight: var(--gf-fw-semibold);
  color: var(--gf-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  margin: 0;
}
.gf-home__ranking-meta {
  font-size: var(--gf-fs-xs);
  color: var(--gf-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin: 0;
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
