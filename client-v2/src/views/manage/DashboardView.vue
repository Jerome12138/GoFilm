<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { manageApi } from '@/api'
import type { DashboardStat } from '@/types/manage'
import BaseSkeleton from '@/components/base/BaseSkeleton.vue'
import BaseEmpty from '@/components/base/BaseEmpty.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'

const data = ref<DashboardStat | null>(null)
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  try {
    data.value = await manageApi.system.dashboard()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
})

const cards = computed(() => {
  if (!data.value) return []
  const d = data.value
  return [
    { label: '影片总数', value: d.filmCount ?? 0, icon: 'film', tint: 'from-[#9b49e7] to-[#4ad1e5]' },
    { label: '采集源', value: d.collectCount ?? 0, icon: 'magic', tint: 'from-[#E50914] to-[#ff6b6b]' },
    { label: '定时任务', value: d.cronCount ?? 0, icon: 'clock', tint: 'from-[#22c55e] to-[#4ad1e5]' }
  ]
})

/* ------------------------------------------------------------------ */
/* 近 7 天新增影片 — 纯 SVG 折线图 (mock)                              */
/* ------------------------------------------------------------------ */

interface ChartPoint {
  date: string // MM-DD
  label: string // 今 / 周一 etc.
  value: number
  x: number
  y: number
}

const CHART_RAW: number[] = [12, 18, 9, 25, 30, 22, 35]
const CHART_W = 600
const CHART_H = 240
const CHART_PAD = { top: 24, right: 16, bottom: 32, left: 36 }

const chartData = computed<ChartPoint[]>(() => {
  const today = new Date()
  const innerW = CHART_W - CHART_PAD.left - CHART_PAD.right
  const innerH = CHART_H - CHART_PAD.top - CHART_PAD.bottom
  const max = Math.max(...CHART_RAW, 10)
  const stepX = innerW / (CHART_RAW.length - 1)
  const weekDays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  return CHART_RAW.map<ChartPoint>((value, i) => {
    const d = new Date(today)
    d.setDate(today.getDate() - (CHART_RAW.length - 1 - i))
    const mm = String(d.getMonth() + 1).padStart(2, '0')
    const dd = String(d.getDate()).padStart(2, '0')
    const isToday = i === CHART_RAW.length - 1
    return {
      date: `${mm}-${dd}`,
      label: isToday ? '今天' : (weekDays[d.getDay()] ?? ''),
      value,
      x: CHART_PAD.left + stepX * i,
      y: CHART_PAD.top + innerH - (value / max) * innerH
    }
  })
})

const chartMax = computed(() => Math.max(...CHART_RAW, 10))
const chartTotal = computed(() => CHART_RAW.reduce((s, v) => s + v, 0))
const chartAvg = computed(() => Math.round(chartTotal.value / CHART_RAW.length))

/** 折线 path: M x0,y0 L x1,y1 ... */
const linePath = computed(() => {
  return chartData.value
    .map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x.toFixed(1)},${p.y.toFixed(1)}`)
    .join(' ')
})

/** Area path: 折线下方闭合，用于渐变填充 */
const areaPath = computed(() => {
  const pts = chartData.value
  if (pts.length === 0) return ''
  const bottomY = CHART_H - CHART_PAD.bottom
  const first = pts[0]
  const last = pts[pts.length - 1]
  if (!first || !last) return ''
  const line = pts.map((p) => `L${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ')
  return `M${first.x.toFixed(1)},${bottomY} ${line} L${last.x.toFixed(1)},${bottomY} Z`
})

/** y 轴 5 等分网格 */
const gridLines = computed(() => {
  const innerH = CHART_H - CHART_PAD.top - CHART_PAD.bottom
  const count = 4
  return Array.from({ length: count + 1 }, (_, i) => {
    const y = CHART_PAD.top + (innerH * i) / count
    const value = Math.round(chartMax.value - (chartMax.value * i) / count)
    return { y, value }
  })
})

const hoveredIdx = ref<number | null>(null)

/* ------------------------------------------------------------------ */
/* 最近活动流 (mock)                                                   */
/* ------------------------------------------------------------------ */

interface Activity {
  id: number
  type: 'film' | 'collect' | 'login' | 'settings' | 'star' | 'refresh'
  text: string
  time: string
  iconColor: string
}

const activities: Activity[] = [
  {
    id: 1,
    type: 'film',
    text: '管理员 <b>john</b> 新增影片《沙丘 第二部》',
    time: '3 分钟前',
    iconColor: '#9b49e7'
  },
  {
    id: 2,
    type: 'collect',
    text: '采集源 <b>#3 苹果资源</b> 成功采集 <b>12</b> 部影片',
    time: '17 分钟前',
    iconColor: '#4ad1e5'
  },
  {
    id: 3,
    type: 'login',
    text: '用户 <b>alice@gofilm</b> 从 IP 39.144.x.x 登录',
    time: '1 小时前',
    iconColor: '#22c55e'
  },
  {
    id: 4,
    type: 'refresh',
    text: '定时任务 <b>每日采集</b> 执行完毕，新增 <b>48</b> 条记录',
    time: '今天 10:23',
    iconColor: '#f59e0b'
  },
  {
    id: 5,
    type: 'settings',
    text: '系统配置 <b>站点 SEO</b> 由 <b>root</b> 更新',
    time: '今天 09:11',
    iconColor: '#E50914'
  },
  {
    id: 6,
    type: 'star',
    text: '影片《奥本海默》评分破 <b>9.0</b>，进入精选推荐',
    time: '昨天 22:48',
    iconColor: '#fbbf24'
  }
]

const activityIcon = (type: Activity['type']): string => {
  const map: Record<Activity['type'], string> = {
    film: 'film',
    collect: 'magic',
    login: 'user',
    settings: 'settings',
    star: 'star',
    refresh: 'refresh'
  }
  return map[type]
}
</script>

<template>
  <section class="flex flex-col gap-[var(--gf-space-6)]">
    <header>
      <h1 class="text-[var(--gf-fs-2xl)] font-[var(--gf-fw-bold)] text-brand-gradient">
        管理中心
      </h1>
      <p class="text-secondary text-sm mt-[var(--gf-space-1)]">
        概览 GoFilm 的核心运营指标
      </p>
    </header>

    <!-- 统计卡片 -->
    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-[var(--gf-space-4)]">
      <BaseSkeleton v-for="i in 3" :key="i" shape="rect" height="128px" />
    </div>

    <BaseEmpty v-else-if="error" :description="error" />

    <div
      v-else
      class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-[var(--gf-space-4)]"
    >
      <div
        v-for="card in cards"
        :key="card.label"
        class="card p-[var(--gf-space-5)] bg-gradient-to-br relative overflow-hidden"
        :class="card.tint"
      >
        <div class="absolute -right-4 -bottom-4 opacity-20 text-white">
          <BaseIcon :name="card.icon" size="120px" />
        </div>
        <div class="relative">
          <div class="text-white/80 text-sm">{{ card.label }}</div>
          <div class="text-white text-[var(--gf-fs-3xl)] font-[var(--gf-fw-black)] mt-[var(--gf-space-2)]">
            {{ card.value }}
          </div>
        </div>
      </div>
    </div>

    <!-- 图表 + 活动 -->
    <div class="grid grid-cols-1 lg:grid-cols-5 gap-[var(--gf-space-4)]">
      <!-- 折线图 -->
      <div class="lg:col-span-3 bg-surface rounded-[var(--gf-radius-lg)] shadow-card p-[var(--gf-space-5)] flex flex-col">
        <div class="flex items-start justify-between gap-[var(--gf-space-4)] flex-wrap">
          <div>
            <h2 class="text-[var(--gf-fs-lg)] font-[var(--gf-fw-semibold)] text-primary">
              近 7 天新增影片
            </h2>
            <p class="text-muted text-xs mt-[var(--gf-space-1)]">
              数据每小时同步一次
            </p>
          </div>
          <div class="flex gap-[var(--gf-space-5)]">
            <div>
              <div class="text-muted text-xs">7 日总计</div>
              <div class="text-primary text-[var(--gf-fs-xl)] font-[var(--gf-fw-bold)] mt-[2px]">
                {{ chartTotal }}
              </div>
            </div>
            <div>
              <div class="text-muted text-xs">日均</div>
              <div class="text-primary text-[var(--gf-fs-xl)] font-[var(--gf-fw-bold)] mt-[2px]">
                {{ chartAvg }}
              </div>
            </div>
          </div>
        </div>

        <div class="mt-[var(--gf-space-4)] w-full" style="height: 240px">
          <svg
            :viewBox="`0 0 ${CHART_W} ${CHART_H}`"
            width="100%"
            height="100%"
            preserveAspectRatio="none"
            class="block overflow-visible"
            @mouseleave="hoveredIdx = null"
          >
            <defs>
              <linearGradient id="dashAreaGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stop-color="#9b49e7" stop-opacity="0.45" />
                <stop offset="100%" stop-color="#4ad1e5" stop-opacity="0.02" />
              </linearGradient>
              <linearGradient id="dashLineGrad" x1="0" y1="0" x2="1" y2="0">
                <stop offset="0%" stop-color="#9b49e7" />
                <stop offset="100%" stop-color="#4ad1e5" />
              </linearGradient>
            </defs>

            <!-- 网格 -->
            <g>
              <line
                v-for="g in gridLines"
                :key="`g-${g.y}`"
                :x1="CHART_PAD.left"
                :x2="CHART_W - CHART_PAD.right"
                :y1="g.y"
                :y2="g.y"
                stroke="var(--gf-border-subtle)"
                stroke-dasharray="3 4"
                stroke-width="1"
              />
              <text
                v-for="g in gridLines"
                :key="`t-${g.y}`"
                :x="CHART_PAD.left - 8"
                :y="g.y + 4"
                text-anchor="end"
                font-size="11"
                fill="var(--gf-text-muted)"
              >
                {{ g.value }}
              </text>
            </g>

            <!-- area -->
            <path :d="areaPath" fill="url(#dashAreaGrad)" />
            <!-- line -->
            <path
              :d="linePath"
              fill="none"
              stroke="url(#dashLineGrad)"
              stroke-width="2.5"
              stroke-linejoin="round"
              stroke-linecap="round"
            />

            <!-- 数据点 + x 轴 label + hover 热区 -->
            <g v-for="(p, i) in chartData" :key="`p-${i}`">
              <circle
                :cx="p.x"
                :cy="p.y"
                r="4"
                fill="var(--gf-bg-surface)"
                stroke="#9b49e7"
                stroke-width="2"
              />
              <circle
                v-if="hoveredIdx === i"
                :cx="p.x"
                :cy="p.y"
                r="7"
                fill="#9b49e7"
                fill-opacity="0.25"
              />
              <text
                :x="p.x"
                :y="CHART_H - 12"
                text-anchor="middle"
                font-size="11"
                fill="var(--gf-text-muted)"
              >
                {{ p.label }}
              </text>
              <!-- hover label -->
              <g v-if="hoveredIdx === i" :transform="`translate(${p.x}, ${p.y - 14})`">
                <rect
                  x="-22"
                  y="-22"
                  width="44"
                  height="20"
                  rx="6"
                  fill="var(--gf-bg-elevated)"
                  stroke="var(--gf-border-subtle)"
                />
                <text
                  x="0"
                  y="-8"
                  text-anchor="middle"
                  font-size="11"
                  font-weight="600"
                  fill="var(--gf-text-primary)"
                >
                  {{ p.value }}
                </text>
              </g>
              <!-- 透明热区 -->
              <rect
                :x="p.x - 20"
                :y="CHART_PAD.top"
                width="40"
                :height="CHART_H - CHART_PAD.top - CHART_PAD.bottom"
                fill="transparent"
                @mouseenter="hoveredIdx = i"
              />
            </g>
          </svg>
        </div>
      </div>

      <!-- 活动流 -->
      <div class="lg:col-span-2 bg-surface rounded-[var(--gf-radius-lg)] shadow-card p-[var(--gf-space-5)] flex flex-col">
        <div class="flex items-center justify-between">
          <h2 class="text-[var(--gf-fs-lg)] font-[var(--gf-fw-semibold)] text-primary">
            最近活动
          </h2>
          <span class="text-muted text-xs">实时更新</span>
        </div>

        <ul class="mt-[var(--gf-space-4)] flex flex-col gap-[var(--gf-space-3)]">
          <li
            v-for="a in activities"
            :key="a.id"
            class="flex items-start gap-[var(--gf-space-3)] p-[var(--gf-space-3)] rounded-[var(--gf-radius-md,8px)] hover:bg-[var(--gf-bg-elevated)] transition"
          >
            <span
              class="flex-shrink-0 w-9 h-9 rounded-full flex items-center justify-center text-white"
              :style="{ background: a.iconColor }"
            >
              <BaseIcon :name="activityIcon(a.type)" size="18px" />
            </span>
            <div class="flex-1 min-w-0">
              <div class="text-primary text-sm leading-relaxed" v-html="a.text" />
              <div class="text-muted text-xs mt-[2px]">{{ a.time }}</div>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </section>
</template>

<style scoped>
:deep(b) {
  color: var(--gf-text-primary);
  font-weight: var(--gf-fw-semibold);
}
</style>
