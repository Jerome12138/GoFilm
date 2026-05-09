<script setup lang="ts">
import { computed } from 'vue'
import type { FilmListItem } from '@/types/film'
import BaseImage from '@/components/base/BaseImage.vue'
import BaseTag from '@/components/base/BaseTag.vue'

interface Props {
  item: FilmListItem
  /** 是否显示卡片下方的标题（移动 / 平板 / TV 默认 true，桌面默认 false 由父级决定） */
  showTitleBelow?: boolean
  /** 评分（可选，1-10） */
  score?: number | string
  /** 是否懒加载图片 */
  lazy?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showTitleBelow: true,
  score: '',
  lazy: true
})

const linkTo = computed(() => ({
  path: '/filmDetail',
  query: { link: String(props.item.id ?? props.item.mid ?? '') }
}))

const cornerTags = computed<string[]>(() => {
  const tags: string[] = []
  if (props.item.year) tags.push(String(props.item.year))
  if (props.item.cName) tags.push(String(props.item.cName))
  if (props.item.area) tags.push(String(props.item.area))
  return tags.slice(0, 2)
})

const remarks = computed(() => props.item.remarks || '')
</script>

<template>
  <RouterLink
    :to="linkTo"
    class="gf-film-card block group"
    data-focusable="true"
    tabindex="0"
    :aria-label="item.name"
  >
    <div class="gf-film-card__poster relative overflow-hidden rounded-[var(--gf-radius-lg)] shadow-card">
      <BaseImage
        :src="item.picture"
        :alt="item.name"
        ratio="2/3"
        :eager="!lazy"
        fit="cover"
      />

      <!-- 角标：年份 / 分类 -->
      <div
        v-if="cornerTags.length"
        class="absolute top-[var(--gf-space-2)] left-[var(--gf-space-2)] flex flex-wrap gap-[var(--gf-space-1)] z-2"
      >
        <BaseTag
          v-for="(t, i) in cornerTags"
          :key="i"
          variant="default"
          size="xs"
        >
          {{ t }}
        </BaseTag>
      </div>

      <!-- 评分 -->
      <BaseTag
        v-if="score"
        variant="brand"
        size="xs"
        class="absolute top-[var(--gf-space-2)] right-[var(--gf-space-2)] z-2"
      >
        {{ score }}
      </BaseTag>

      <!-- remarks（更新到第几集等） -->
      <div
        v-if="remarks"
        class="absolute bottom-[var(--gf-space-2)] right-[var(--gf-space-2)] px-[6px] py-[2px] rounded-[var(--gf-radius-sm)] bg-[rgba(0,0,0,0.7)] text-white text-[var(--gf-fs-xs)] z-2"
      >
        {{ remarks }}
      </div>

      <!-- 蒙版 + hover/focus 内容浮层 -->
      <div class="gf-film-card__mask absolute inset-0 pointer-events-none" />
      <div class="gf-film-card__hover-info absolute inset-x-0 bottom-0 px-[var(--gf-space-3)] py-[var(--gf-space-3)] z-2">
        <h3
          class="text-[var(--gf-fs-md)] font-[var(--gf-fw-semibold)] text-primary line-clamp-2"
        >
          {{ item.name }}
        </h3>
      </div>
    </div>

    <!-- 卡片下方标题（移动/平板/TV 常驻） -->
    <h4
      v-if="showTitleBelow"
      class="gf-film-card__title-below mt-[var(--gf-space-2)] text-[var(--gf-fs-sm)] font-[var(--gf-fw-medium)] text-primary line-clamp-2 leading-[var(--gf-lh-snug)]"
    >
      {{ item.name }}
    </h4>
  </RouterLink>
</template>

<style scoped>
.gf-film-card {
  text-decoration: none;
  outline: none;
  border-radius: var(--gf-radius-lg);
  transition:
    transform var(--gf-dur-base) var(--gf-ease-spring),
    box-shadow var(--gf-dur-base) var(--gf-ease-standard);
}

.gf-film-card__poster {
  background-color: var(--gf-bg-elevated);
  transition:
    transform var(--gf-dur-base) var(--gf-ease-spring),
    box-shadow var(--gf-dur-base) var(--gf-ease-standard);
}

.gf-film-card__mask {
  background-image: var(--gf-mask-card-hover);
  opacity: 0;
  transition: opacity var(--gf-dur-base) var(--gf-ease-standard);
}

.gf-film-card__hover-info {
  opacity: 0;
  transform: translateY(8px);
  transition:
    opacity var(--gf-dur-base) var(--gf-ease-standard),
    transform var(--gf-dur-base) var(--gf-ease-standard);
}

/* 桌面 hover 显示标题浮层 */
@media (hover: hover) and (pointer: fine) {
  .gf-film-card:hover .gf-film-card__poster,
  .gf-film-card:focus-visible .gf-film-card__poster {
    transform: scale(1.06);
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
[data-mode='tv'] .gf-film-card:focus-visible .gf-film-card__poster {
  transform: scale(1.06);
  box-shadow: var(--gf-shadow-hover);
}
</style>
