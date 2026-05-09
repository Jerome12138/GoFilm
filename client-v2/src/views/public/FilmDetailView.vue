<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { filmApi } from '@/api'
import BaseImage from '@/components/base/BaseImage.vue'
import BaseTag from '@/components/base/BaseTag.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseEmpty from '@/components/base/BaseEmpty.vue'
import BaseSkeleton from '@/components/base/BaseSkeleton.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'
import EpisodeTabs from '@/components/film/EpisodeTabs.vue'
import RelatedList from '@/components/film/RelatedList.vue'
import type { FilmDetail, FilmDetailResp, FilmListItem } from '@/types/film'

/**
 * 影片详情 — STORY-009
 * - id 来自 route.query.link（旧站约定）
 * - 调 GET /api/filmDetail
 * - hero：模糊大图背景 + 海报 + 信息（标题 / 评分 / 类型 / 标签 / 导演 / 主演 / 上映 / 地区 / 剧情）
 * - 剧情可展开收起（>140 字截断）
 * - 立即播放：跳 /play?id=&source=&episode=  （source = list[0].id, episode = 0）
 * - EpisodeTabs：sources=detail.list；选集走 router.push
 * - RelatedList：传入 relate
 */

const route = useRoute()
const router = useRouter()

const linkId = computed(() => {
  const v = route.query.link
  if (typeof v === 'string') return v
  if (Array.isArray(v) && typeof v[0] === 'string') return v[0]
  return ''
})

const loading = ref(true)
const errored = ref(false)
const detail = ref<FilmDetail | null>(null)
const relate = ref<FilmListItem[]>([])

async function loadDetail(id: string): Promise<void> {
  loading.value = true
  errored.value = false
  detail.value = null
  relate.value = []
  if (!id) {
    errored.value = true
    loading.value = false
    return
  }
  try {
    const resp: FilmDetailResp = await filmApi.getFilmDetail(id)
    detail.value = resp?.detail ?? null
    relate.value = Array.isArray(resp?.relate) ? resp.relate : []
    if (!detail.value) {
      errored.value = true
    }
  } catch {
    errored.value = true
  } finally {
    loading.value = false
  }
}

onMounted(() => loadDetail(linkId.value))
watch(linkId, (next) => {
  if (next) loadDetail(next)
})

/** ============== 文案处理 ============== */

/** 清理 content 内 HTML 实体 / 全角空格 / br 等 */
function cleanContent(raw: string | undefined): string {
  if (!raw) return ''
  return raw.replace(/(&.*?;)|( )|(　　)|(\n)|(<[^>]+>)/g, '').trim()
}

/** 取前 N 个，逗号 / 空格 / 斜杠 / 顿号 切片 */
function takeNames(raw: string | undefined, max = 3): string[] {
  if (!raw) return []
  return raw
    .split(/[,，、\/\s]+/)
    .map((s) => s.trim())
    .filter((s) => s.length > 0)
    .slice(0, max)
}

/** Hero 背景图 */
const heroBg = computed(() => detail.value?.picture || '')

/** 标签 */
const tagChips = computed<string[]>(() => {
  const d = detail.value
  if (!d) return []
  const chips: string[] = []
  const desc = d.descriptor
  if (d.year) chips.push(String(d.year))
  if (desc?.cName) chips.push(desc.cName)
  if (d.area) chips.push(String(d.area))
  if (desc?.classTag) {
    for (const t of desc.classTag.split(/[,，、\/\s]+/)) {
      const v = t.trim()
      if (v) chips.push(v)
      if (chips.length >= 6) break
    }
  }
  return chips.slice(0, 6)
})

const directors = computed(() => takeNames(detail.value?.descriptor?.director, 3))
const actors = computed(() => takeNames(detail.value?.descriptor?.actor, 5))

/** 评分 */
const score = computed(() => {
  const s = detail.value?.descriptor?.dbScore
  if (s === undefined || s === null || s === '') return ''
  const n = Number(s)
  if (!Number.isFinite(n)) return String(s)
  return n.toFixed(1)
})

/** 剧情展开 */
const SUMMARY_LIMIT = 140
const expanded = ref(false)
const cleanedContent = computed(() =>
  cleanContent(detail.value?.descriptor?.content)
)
const needsClamp = computed(() => cleanedContent.value.length > SUMMARY_LIMIT)
const displayContent = computed(() => {
  if (!needsClamp.value || expanded.value) return cleanedContent.value
  return cleanedContent.value.slice(0, SUMMARY_LIMIT) + '…'
})

/** ============== 跳转 ============== */

function gotoPlay(sourceId: string, episodeIndex: number): void {
  const d = detail.value
  if (!d) return
  router.push({
    path: '/play',
    query: {
      id: String(d.id),
      source: sourceId,
      episode: String(episodeIndex)
    }
  })
}

function playFirst(): void {
  const d = detail.value
  if (!d) return
  const firstSource = d.list?.[0]
  if (!firstSource || !firstSource.linkList?.length) return
  gotoPlay(firstSource.id, 0)
}

/** EpisodeTabs 选中：走 router.push 去 /play */
function onEpisodeSelect(payload: {
  sourceId: string
  episodeIndex: number
}): void {
  gotoPlay(payload.sourceId, payload.episodeIndex)
}

/** 当前激活播放源（用户切 source tab 时只是视觉） */
const activeSourceId = ref('')
function onChangeSource(id: string): void {
  activeSourceId.value = id
}

watch(detail, (d) => {
  activeSourceId.value = d?.list?.[0]?.id ?? ''
}, { immediate: true })
</script>

<template>
  <div class="gf-detail">
    <!-- 加载骨架 -->
    <template v-if="loading">
      <div class="gf-detail__hero gf-detail__hero--skeleton">
        <div class="gf-detail__hero-inner container-page">
          <BaseSkeleton
            shape="rect"
            width="100%"
            ratio="2/3"
            class="gf-detail__poster-skel"
          />
          <div class="flex flex-col gap-[var(--gf-space-3)] flex-1 min-w-0">
            <BaseSkeleton shape="text" width="60%" height="40px" />
            <BaseSkeleton shape="text" width="40%" />
            <BaseSkeleton shape="text" :count="4" />
          </div>
        </div>
      </div>
    </template>

    <!-- 错误 -->
    <template v-else-if="errored || !detail">
      <div class="container-page py-[var(--gf-space-12)]">
        <BaseEmpty
          title="影片不存在或加载失败"
          :description="linkId ? `id=${linkId} 未找到对应内容` : '请从首页或搜索进入此页面'"
        >
          <template #action>
            <BaseButton variant="primary" size="md" @click="router.push('/index')">
              返回首页
            </BaseButton>
          </template>
        </BaseEmpty>
      </div>
    </template>

    <!-- 正常 -->
    <template v-else>
      <!-- Hero -->
      <section class="gf-detail__hero">
        <div
          class="gf-detail__hero-bg"
          :style="heroBg ? { backgroundImage: `url('${heroBg}')` } : undefined"
        />
        <div class="gf-detail__hero-mask" />

        <div class="gf-detail__hero-inner container-page">
          <!-- 海报 -->
          <div class="gf-detail__poster">
            <BaseImage
              :src="detail.picture"
              :alt="detail.name"
              ratio="2/3"
              eager
              fit="cover"
            />
          </div>

          <!-- 信息 -->
          <div class="gf-detail__info">
            <h1 class="gf-detail__title">{{ detail.name }}</h1>

            <div v-if="score || tagChips.length" class="gf-detail__chips">
              <span v-if="score" class="gf-detail__score">
                <BaseIcon name="star" size="0.9em" class="gf-detail__score-icon" />
                {{ score }}
              </span>
              <BaseTag
                v-for="(t, i) in tagChips"
                :key="i"
                :variant="i === 0 ? 'purple' : 'default'"
                size="md"
              >
                {{ t }}
              </BaseTag>
            </div>

            <dl v-if="directors.length || actors.length || detail.descriptor?.releaseDate || detail.descriptor?.area" class="gf-detail__meta">
              <div v-if="directors.length" class="gf-detail__meta-row">
                <dt>导演</dt>
                <dd>{{ directors.join(' · ') }}</dd>
              </div>
              <div v-if="actors.length" class="gf-detail__meta-row">
                <dt>主演</dt>
                <dd>{{ actors.join(' · ') }}</dd>
              </div>
              <div v-if="detail.descriptor?.releaseDate" class="gf-detail__meta-row">
                <dt>上映</dt>
                <dd>{{ detail.descriptor.releaseDate }}</dd>
              </div>
              <div v-if="detail.area || detail.descriptor?.area" class="gf-detail__meta-row">
                <dt>地区</dt>
                <dd>{{ detail.area || detail.descriptor?.area }}</dd>
              </div>
              <div v-if="detail.descriptor?.remarks || detail.remarks" class="gf-detail__meta-row">
                <dt>状态</dt>
                <dd>{{ detail.descriptor?.remarks || detail.remarks }}</dd>
              </div>
            </dl>

            <p v-if="cleanedContent" class="gf-detail__summary">
              {{ displayContent }}
              <button
                v-if="needsClamp"
                type="button"
                class="gf-detail__expand"
                @click="expanded = !expanded"
              >
                {{ expanded ? '收起' : '展开' }}
              </button>
            </p>

            <div class="gf-detail__cta">
              <BaseButton
                variant="primary"
                size="lg"
                :disabled="!detail.list?.[0]?.linkList?.length"
                @click="playFirst"
              >
                <template #icon>
                  <BaseIcon name="play" size="1.1em" />
                </template>
                立即播放
              </BaseButton>
              <BaseButton variant="outline" size="lg">
                <template #icon>
                  <BaseIcon name="heart" size="1.1em" />
                </template>
                收藏
              </BaseButton>
              <BaseButton variant="ghost" size="lg">
                <template #icon>
                  <BaseIcon name="share" size="1.1em" />
                </template>
                分享
              </BaseButton>
            </div>
          </div>
        </div>
      </section>

      <!-- 集数 -->
      <section v-if="detail.list?.length" class="gf-detail__episodes container-page">
        <h2 class="gf-detail__section-title">播放源 / 集数选择</h2>
        <EpisodeTabs
          :sources="detail.list"
          :current-source-id="activeSourceId"
          @select="onEpisodeSelect"
          @change-source="onChangeSource"
        />
      </section>

      <!-- 相关推荐 -->
      <section v-if="relate.length" class="gf-detail__relate container-page">
        <RelatedList :items="relate" title="相关推荐" />
      </section>
    </template>
  </div>
</template>

<style scoped>
.gf-detail {
  width: 100%;
  display: flex;
  flex-direction: column;
}

/* ============= Hero ============= */
.gf-detail__hero {
  position: relative;
  width: 100%;
  isolation: isolate;
  overflow: hidden;
}

.gf-detail__hero-bg {
  position: absolute;
  inset: -40px;
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  filter: blur(40px) brightness(0.4);
  transform: scale(1.1);
  z-index: 0;
}

.gf-detail__hero-mask {
  position: absolute;
  inset: 0;
  background-image: linear-gradient(
    180deg,
    rgba(11, 11, 15, 0.5) 0%,
    rgba(11, 11, 15, 0.78) 75%,
    rgba(11, 11, 15, 1) 100%
  );
  z-index: 0;
}

.gf-detail__hero-inner {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  gap: var(--gf-space-6);
  padding-top: var(--gf-space-8);
  padding-bottom: var(--gf-space-12);
  align-items: center;
  text-align: center;
}

@media (min-width: 768px) {
  .gf-detail__hero-inner {
    flex-direction: row;
    align-items: flex-start;
    text-align: left;
    gap: var(--gf-space-8);
    padding-top: var(--gf-space-12);
    padding-bottom: var(--gf-space-12);
  }
}

.gf-detail__poster {
  width: 200px;
  flex-shrink: 0;
  border-radius: var(--gf-radius-lg);
  overflow: hidden;
  box-shadow: var(--gf-shadow-xl);
}

.gf-detail__poster-skel {
  width: 200px;
  flex-shrink: 0;
}

@media (min-width: 768px) {
  .gf-detail__poster,
  .gf-detail__poster-skel {
    width: 220px;
  }
}

@media (min-width: 1024px) {
  .gf-detail__poster,
  .gf-detail__poster-skel {
    width: 280px;
  }
}

.gf-detail__info {
  display: flex;
  flex-direction: column;
  gap: var(--gf-space-4);
  flex: 1;
  min-width: 0;
  align-items: center;
}

@media (min-width: 768px) {
  .gf-detail__info {
    align-items: flex-start;
  }
}

.gf-detail__title {
  font-family: var(--gf-font-display);
  font-size: var(--gf-fs-2xl);
  font-weight: var(--gf-fw-black);
  letter-spacing: var(--gf-tracking-tight);
  line-height: var(--gf-lh-tight);
  color: var(--gf-text-primary);
  margin: 0;
}

@media (min-width: 768px) {
  .gf-detail__title {
    font-size: var(--gf-fs-3xl);
  }
}

.gf-detail__chips {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--gf-space-2);
}

.gf-detail__score {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  padding: 0 10px;
  border-radius: var(--gf-radius-sm);
  background-color: rgba(245, 158, 11, 0.16);
  color: var(--gf-warning);
  font-weight: var(--gf-fw-bold);
  font-size: var(--gf-fs-sm);
}

.gf-detail__score-icon {
  margin-bottom: 1px;
}

.gf-detail__meta {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: var(--gf-text-secondary);
}

.gf-detail__meta-row {
  display: flex;
  gap: var(--gf-space-2);
  font-size: var(--gf-fs-sm);
  line-height: var(--gf-lh-snug);
  flex-wrap: wrap;
  justify-content: center;
}

@media (min-width: 768px) {
  .gf-detail__meta-row {
    justify-content: flex-start;
  }
}

.gf-detail__meta-row dt {
  flex: 0 0 auto;
  color: var(--gf-text-muted);
  min-width: 36px;
}

.gf-detail__meta-row dd {
  margin: 0;
  color: var(--gf-text-secondary);
  flex: 1;
  min-width: 0;
}

.gf-detail__summary {
  margin: 0;
  color: var(--gf-text-secondary);
  font-size: var(--gf-fs-sm);
  line-height: var(--gf-lh-relaxed);
  max-width: 720px;
}

.gf-detail__expand {
  display: inline;
  background: transparent;
  border: none;
  color: var(--gf-text-link);
  font-size: var(--gf-fs-sm);
  cursor: pointer;
  padding: 0 4px;
}

.gf-detail__expand:hover,
.gf-detail__expand:focus-visible {
  color: var(--gf-text-link-hover);
  outline: none;
}

.gf-detail__cta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--gf-space-3);
  justify-content: center;
}

@media (min-width: 768px) {
  .gf-detail__cta {
    justify-content: flex-start;
  }
}

/* ============= Episodes / Relate ============= */
.gf-detail__episodes,
.gf-detail__relate {
  padding-block: var(--gf-space-8);
}

.gf-detail__section-title {
  font-size: var(--gf-fs-xl);
  font-weight: var(--gf-fw-bold);
  color: var(--gf-text-primary);
  margin: 0 0 var(--gf-space-4);
}

/* skeleton 容器 */
.gf-detail__hero--skeleton .gf-detail__hero-inner {
  z-index: 0;
}
</style>

<style>
/* TV 模式：海报放大、字号放大、安全区 */
[data-mode='tv'] .gf-detail__hero-inner {
  padding-inline: var(--gf-tv-safe);
  flex-direction: row;
  align-items: flex-start;
  text-align: left;
}
[data-mode='tv'] .gf-detail__poster {
  width: 360px;
}
[data-mode='tv'] .gf-detail__title {
  font-size: var(--gf-fs-3xl);
}
[data-mode='tv'] .gf-detail__summary,
[data-mode='tv'] .gf-detail__meta-row {
  font-size: var(--gf-fs-md);
}
[data-mode='tv'] .gf-detail__episodes,
[data-mode='tv'] .gf-detail__relate {
  padding-inline: var(--gf-tv-safe);
}
</style>
