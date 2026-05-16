<script setup lang="ts">
import { computed } from 'vue'
import type { FilmListItem } from '@/types/film'
import BaseImage from '@/components/base/BaseImage.vue'

interface Props {
  item: FilmListItem
  /** 是否显示卡片下方的标题（移动 / 平板 / TV 默认 true，桌面默认 false 由父级决定） */
  showTitleBelow?: boolean
  /** 评分（可选，1-10） */
  score?: number | string
  /** 是否懒加载图片 */
  lazy?: boolean
  /** 封面比例, 默认 3:4 (影视行业标准); 历史调用方可改 "2/3" 等 */
  ratio?: string
}

const props = withDefaults(defineProps<Props>(), {
  showTitleBelow: true,
  score: '',
  lazy: true,
  ratio: '3/4'
})

const cardRatio = computed(() => props.ratio)

/** hover 浮层副信息: 年份 · 地区 · 分类, 缺字段则跳过 */
const metaText = computed(() => {
  const parts: string[] = []
  if (props.item.year) parts.push(String(props.item.year))
  if (props.item.area) parts.push(String(props.item.area))
  if (props.item.cName) parts.push(String(props.item.cName))
  return parts.join(' · ')
})

/** 标题下方常驻副信息: 年份 · 分类 (省略地区, 控制长度避免 2 行); 评分另算 */
const subTextBelow = computed(() => {
  const parts: string[] = []
  if (props.item.year) parts.push(String(props.item.year))
  if (props.item.cName) parts.push(String(props.item.cName))
  return parts.join(' · ')
})

const linkTo = computed(() => ({
  path: '/filmDetail',
  query: { link: String(props.item.id ?? props.item.mid ?? '') }
}))

/** 角标 remarks：更新到第几集这种关键信息（其它如年份/分类太冗，移到 hover 浮层与详情页） */
const remarks = computed(() => props.item.remarks || '')

/**
 * 评分显示策略：
 *  1. 父组件显式传 score 优先
 *  2. 否则尝试 item.dbScore / item.score（后端列表接口通常不返回，但 mock / 部分聚合接口会带）
 *  3. 评分需要 ≥ 1 才显示，过滤掉 0 / NaN / "暂无"
 */
const scoreText = computed(() => {
  const raw =
    props.score !== '' && props.score !== undefined && props.score !== null
      ? props.score
      : props.item.dbScore ?? props.item.score ?? ''
  if (raw === '' || raw === undefined || raw === null) return ''
  const n = Number(raw)
  if (!Number.isFinite(n) || n < 1) return ''
  // 1-10 区间保留 1 位小数（已是整数则不加）
  return n % 1 === 0 ? n.toFixed(0) : n.toFixed(1)
})
</script>

<template>
  <RouterLink
    :to="linkTo"
    class="gf-film-card block group"
    data-focusable="true"
    tabindex="0"
    :aria-label="item.name"
  >
    <div class="gf-film-card__poster relative overflow-hidden shadow-card">
      <BaseImage
        :src="item.picture"
        :alt="item.name"
        :ratio="cardRatio"
        :eager="!lazy"
        fit="cover"
      />

      <!-- remarks 角标 ("更新至 N 集" / "HD" / "BD" / "独播"), 右上 -->
      <!-- 评分已下沉到卡片下方副信息行, 不在封面再重复出现 -->
      <span
        v-if="remarks"
        class="gf-film-card__remark absolute top-[6px] right-[6px] z-2"
      >
        {{ remarks }}
      </span>

      <!-- 蒙版 + hover 浮层 (PC: hover 上滑显示副信息 + 播放图标; 触屏: 不显示) -->
      <div class="gf-film-card__mask absolute inset-0 pointer-events-none" />
      <div class="gf-film-card__hover-info absolute inset-x-0 bottom-0 px-[var(--gf-space-3)] pb-[var(--gf-space-3)] pt-[var(--gf-space-5)] z-2">
        <h3
          class="text-[var(--gf-fs-md)] font-[var(--gf-fw-semibold)] text-primary line-clamp-2 leading-[var(--gf-lh-snug)]"
        >
          {{ item.name }}
        </h3>
        <div
          v-if="metaText"
          class="gf-film-card__meta mt-[var(--gf-space-1)] text-[var(--gf-fs-xs)] text-secondary truncate"
        >
          {{ metaText }}
        </div>
      </div>

      <!-- PC hover 播放图标 (中央) -->
      <div class="gf-film-card__play absolute inset-0 flex items-center justify-center pointer-events-none z-2" aria-hidden="true">
        <span class="gf-film-card__play-btn">
          <svg viewBox="0 0 24 24" fill="currentColor" width="22" height="22"><path d="M8 5v14l11-7z"/></svg>
        </span>
      </div>
    </div>

    <!-- 卡片下方信息区: 标题 + 副信息 (年份·分类·⭐评分), 常驻可见 (bilibili/腾讯视频风格) -->
    <div v-if="showTitleBelow" class="gf-film-card__below">
      <h4 class="gf-film-card__title-below">
        {{ item.name }}
      </h4>
      <div v-if="subTextBelow" class="gf-film-card__sub-below">
        <span v-if="scoreText" class="gf-film-card__sub-score">
          <svg viewBox="0 0 24 24" fill="currentColor" width="11" height="11" aria-hidden="true">
            <path d="M12 .587l3.668 7.568L24 9.75l-6 5.852L19.336 24 12 19.897 4.664 24 6 15.602 0 9.75l8.332-1.595z"/>
          </svg>
          {{ scoreText }}
        </span>
        <span v-if="subTextBelow" class="gf-film-card__sub-meta">{{ subTextBelow }}</span>
      </div>
    </div>
  </RouterLink>
</template>

<style scoped>
.gf-film-card {
  text-decoration: none;
  outline: none;
  border-radius: var(--gf-card-radius);
  transition:
    transform var(--gf-dur-base) var(--gf-ease-spring),
    box-shadow var(--gf-dur-base) var(--gf-ease-standard);
}

.gf-film-card__poster {
  background-color: var(--gf-bg-elevated);
  border-radius: var(--gf-card-radius);
  transition:
    transform var(--gf-dur-base) var(--gf-ease-spring),
    box-shadow var(--gf-dur-base) var(--gf-ease-standard);
}

.gf-film-card__mask {
  background-image: linear-gradient(
    to top,
    var(--gf-hover-overlay) 0%,
    rgba(0, 0, 0, 0.35) 45%,
    rgba(0, 0, 0, 0) 70%
  );
  opacity: 0;
  transition: opacity var(--gf-dur-base) var(--gf-ease-standard);
}

.gf-film-card__hover-info {
  opacity: 0;
  transform: translateY(12px);
  transition:
    opacity var(--gf-dur-base) var(--gf-ease-standard),
    transform var(--gf-dur-base) var(--gf-ease-standard);
}

.gf-film-card__meta {
  color: rgba(255, 255, 255, 0.78);
}

/* 中央播放图标 (hover 才显示) */
.gf-film-card__play {
  opacity: 0;
  transform: scale(0.85);
  transition:
    opacity var(--gf-dur-base) var(--gf-ease-standard),
    transform var(--gf-dur-base) var(--gf-ease-spring);
}
.gf-film-card__play-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 9999px;
  background-image: var(--gf-brand-gradient);
  color: #fff;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.45);
}

/* 桌面 hover: 卡片轻微缩放 + 蒙版/简介浮层上滑 + 中央播放按钮浮出 */
@media (hover: hover) and (pointer: fine) {
  .gf-film-card:hover .gf-film-card__poster,
  .gf-film-card:focus-visible .gf-film-card__poster {
    transform: scale(1.04);
    box-shadow: var(--gf-shadow-hover);
  }
  .gf-film-card:hover .gf-film-card__mask,
  .gf-film-card:focus-visible .gf-film-card__mask {
    opacity: 1;
  }
  .gf-film-card:hover .gf-film-card__hover-info,
  .gf-film-card:focus-visible .gf-film-card__hover-info {
    opacity: 1;
    transform: translateY(0);
  }
  .gf-film-card:hover .gf-film-card__play,
  .gf-film-card:focus-visible .gf-film-card__play {
    opacity: 1;
    transform: scale(1);
  }
  .gf-film-card:hover .gf-film-card__title-below,
  .gf-film-card:focus-visible .gf-film-card__title-below {
    color: var(--gf-text-primary);
  }
}

/* 移动端按下反馈 */
@media (hover: none) {
  .gf-film-card:active .gf-film-card__poster {
    transform: scale(0.97);
  }
}

/* 焦点态强化 */
.gf-film-card:focus-visible {
  outline: none;
}
.gf-film-card:focus-visible .gf-film-card__poster {
  box-shadow: var(--gf-shadow-focus-ring), var(--gf-shadow-hover);
}

.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* 标题下方区域: 双行结构 (bilibili / 腾讯视频风格) */
.gf-film-card__below {
  margin-top: var(--gf-space-2);
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gf-film-card__title-below {
  /* 默认两行截断 */
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  min-height: calc(var(--gf-fs-sm) * var(--gf-lh-snug, 1.3) * 2);
}
.gf-film-card__sub-below {
  display: flex;
  align-items: center;
  gap: var(--gf-space-2);
  font-size: var(--gf-fs-xs);
  color: var(--gf-text-muted);
  line-height: 1.4;
}
.gf-film-card__sub-score {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: var(--gf-warning);
  font-weight: var(--gf-fw-semibold);
  flex-shrink: 0;
}
.gf-film-card__sub-meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

/* 角标：评分 / remarks（紧凑版，不再用 BaseTag，避免在小封面上视觉过重） */
.gf-film-card__remark,
.gf-film-card__score {
  display: inline-flex;
  align-items: center;
  height: 18px;
  padding: 0 6px;
  border-radius: var(--gf-radius-sm);
  font-size: 11px;
  font-weight: var(--gf-fw-semibold);
  letter-spacing: 0.02em;
  line-height: 1;
  pointer-events: none;
  white-space: nowrap;
}
.gf-film-card__remark {
  background-color: rgba(0, 0, 0, 0.7);
  color: #fff;
  backdrop-filter: blur(4px);
}
.gf-film-card__score {
  height: 20px;
  padding: 0 7px;
  background-image: var(--gf-brand-gradient);
  color: #fff;
  font-weight: var(--gf-fw-bold);
}

/* 中等以上屏幕（封面更大）允许稍微抬高字号 */
@media (min-width: 1024px) {
  .gf-film-card__remark,
  .gf-film-card__score {
    height: 20px;
    font-size: 12px;
    padding: 0 7px;
  }
}
</style>

<style>
/* TV 模式：默认显示标题浮层（不依赖 hover） */
[data-mode='tv'] .gf-film-card__mask {
  opacity: 0.65;
}
[data-mode='tv'] .gf-film-card__hover-info {
  opacity: 1;
  transform: translateY(0);
}
/* TV 焦点态：加强 scale + 阴影 + ring（focus-visible 与 focus 双触发） */
[data-mode='tv'] .gf-film-card:focus,
[data-mode='tv'] .gf-film-card:focus-visible {
  outline: none;
}
[data-mode='tv'] .gf-film-card:focus .gf-film-card__poster,
[data-mode='tv'] .gf-film-card:focus-visible .gf-film-card__poster {
  transform: scale(1.06);
  box-shadow:
    0 0 0 4px var(--gf-brand-cyan),
    0 24px 60px rgba(0, 0, 0, 0.7);
}
[data-mode='tv'] .gf-film-card:focus .gf-film-card__title-below,
[data-mode='tv'] .gf-film-card:focus-visible .gf-film-card__title-below {
  color: var(--gf-text-primary);
}
/* TV 卡片标题字号（不靠 hover 显示） */
[data-mode='tv'] .gf-film-card__title-below {
  font-size: var(--gf-fs-base);
}
[data-mode='tv'] .gf-film-card__remark,
[data-mode='tv'] .gf-film-card__score {
  height: 26px;
  padding: 0 10px;
  font-size: 14px;
}
[data-mode='tv'] .gf-film-card__hover-info h3 {
  font-size: var(--gf-fs-md);
}
</style>
