<script setup lang="ts">
import { ref } from 'vue'
import HeroCarousel from '@/components/film/HeroCarousel.vue'
import FilmRow from '@/components/film/FilmRow.vue'
import type { FilmListItem } from '@/types/film'

/**
 * STORY-006/007 视觉验证页：使用 mock 数据，仅用于本地组件展示。
 * 后续 STORY-008 接入 /api/index 后会被替换为真实数据 + 骨架屏。
 */

interface MockBanner extends FilmListItem {}

const heroItems = ref<MockBanner[]>([
  {
    id: 'mock-hero-1',
    name: '星际边境：黎明之战',
    cName: '科幻',
    year: '2025',
    area: '美国',
    remarks: '银河尽头的人类前哨陷落，最后的舰队踏上反击之路。',
    picture:
      'https://images.unsplash.com/photo-1518562180175-34a163b1a9a6?auto=format&fit=crop&w=1920&q=80'
  },
  {
    id: 'mock-hero-2',
    name: '雾都迷案',
    cName: '悬疑',
    year: '2024',
    area: '英国',
    remarks: '一桩发生在维多利亚时代的连环失踪案，揭开雾都最阴暗的秘密。',
    picture:
      'https://images.unsplash.com/photo-1542204165-65bf26472b9b?auto=format&fit=crop&w=1920&q=80'
  },
  {
    id: 'mock-hero-3',
    name: '风暴之巅',
    cName: '动作',
    year: '2025',
    area: '中国',
    remarks: '一名退役登山员被卷入跨国阴谋，必须在 24 小时内登顶世界之巅。',
    picture:
      'https://images.unsplash.com/photo-1464822759023-fed622ff2c3b?auto=format&fit=crop&w=1920&q=80'
  }
])

function makeRow(prefix: string, count = 12): FilmListItem[] {
  const pics = [
    'https://images.unsplash.com/photo-1485846234645-a62644f84728?auto=format&fit=crop&w=600&q=80',
    'https://images.unsplash.com/photo-1440404653325-ab127d49abc1?auto=format&fit=crop&w=600&q=80',
    'https://images.unsplash.com/photo-1478720568477-152d9b164e26?auto=format&fit=crop&w=600&q=80',
    'https://images.unsplash.com/photo-1542204165-65bf26472b9b?auto=format&fit=crop&w=600&q=80',
    'https://images.unsplash.com/photo-1518930259200-3e8b1c1c1f3d?auto=format&fit=crop&w=600&q=80',
    'https://images.unsplash.com/photo-1497032628192-86f99bcd76bc?auto=format&fit=crop&w=600&q=80'
  ]
  const cats = ['剧情', '动作', '悬疑', '科幻', '爱情', '惊悚']
  const areas = ['中国', '美国', '日本', '韩国', '英国', '法国']
  return Array.from({ length: count }).map<FilmListItem>((_, i) => ({
    id: `${prefix}-${i}`,
    name: `${prefix}示例片名 ${i + 1}`,
    cName: cats[i % cats.length] ?? '剧情',
    year: String(2020 + (i % 6)),
    area: areas[i % areas.length] ?? '中国',
    remarks: i % 3 === 0 ? `更新至第 ${10 + i} 集` : '',
    picture: pics[i % pics.length] ?? ''
  }))
}

const movieRow = ref<FilmListItem[]>(makeRow('电影', 14))
const tvRow = ref<FilmListItem[]>(makeRow('剧集', 14))
const animeRow = ref<FilmListItem[]>(makeRow('动漫', 14))
</script>

<template>
  <div class="gf-home flex flex-col">
    <HeroCarousel :items="heroItems" />

    <div class="gf-home__rows flex flex-col gap-[var(--gf-space-8)] py-[var(--gf-space-8)] md:gap-[var(--gf-space-12)] md:py-[var(--gf-space-12)]">
      <FilmRow
        title="热门电影"
        :more-link="{ path: '/filmClassify', query: { Pid: 1 } }"
        :items="movieRow"
      />
      <FilmRow
        title="热门剧集"
        :more-link="{ path: '/filmClassify', query: { Pid: 2 } }"
        :items="tvRow"
      />
      <FilmRow
        title="动漫精选"
        :more-link="{ path: '/filmClassify', query: { Pid: 3 } }"
        :items="animeRow"
      />
    </div>

    <div class="container-page pb-[var(--gf-space-12)] text-muted text-sm">
      <span>* 当前为本地 mock 视觉验证，真实数据将在 STORY-008 接入 /api/index。</span>
    </div>
  </div>
</template>
