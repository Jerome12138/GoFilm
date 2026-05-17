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
import RelatedList from '@/components/film/RelatedList.vue'
import type { FilmDetail, FilmDetailResp, FilmListItem } from '@/types/film'
import { useHistoryStore } from '@/stores/history'
import { useFavoriteStore } from '@/stores/favorite'
import { storeToRefs } from 'pinia'

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
const historyStore = useHistoryStore()
const favoriteStore = useFavoriteStore()
const { map: favoriteMap } = storeToRefs(favoriteStore)

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

/** Hero 背景图（CSS escaping，防御后端字段污染） */
const heroBg = computed(() => detail.value?.picture || '')
const heroBgStyle = computed(() => {
  const url = heroBg.value
  if (!url) return undefined
  // 用 JSON.stringify 转义引号 / 反斜杠等 CSS 注入字符
  return { backgroundImage: `url(${JSON.stringify(url)})` }
})

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

const directors = computed(() => takeNames(detail.value?.descriptor?.director, 6))
const actors = computed(() => takeNames(detail.value?.descriptor?.actor, 12))

/** 演职人员卡: 名字 → 渐变首字头像 (基于 hash 选 8 种渐变之一) */
const PERSON_GRADIENTS = [
  'linear-gradient(135deg, #9b49e7 0%, #4ad1e5 100%)',
  'linear-gradient(135deg, #f59e0b 0%, #e50914 100%)',
  'linear-gradient(135deg, #22c55e 0%, #4ad1e5 100%)',
  'linear-gradient(135deg, #3b82f6 0%, #9b49e7 100%)',
  'linear-gradient(135deg, #ef4444 0%, #f59e0b 100%)',
  'linear-gradient(135deg, #06b6d4 0%, #6366f1 100%)',
  'linear-gradient(135deg, #ec4899 0%, #f59e0b 100%)',
  'linear-gradient(135deg, #14b8a6 0%, #4ad1e5 100%)'
]
function personGradient(name: string): string {
  let h = 0
  for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) & 0xffffffff
  return PERSON_GRADIENTS[Math.abs(h) % PERSON_GRADIENTS.length] ?? PERSON_GRADIENTS[0]!
}
function personInitial(name: string): string {
  return name ? name.charAt(0) : '?'
}

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

/** "继续观看": 用户在本机看过该片时显示, 跳到上次中断的源/集 */
const resumeRecord = computed(() => {
  const d = detail.value
  if (!d) return null
  const rec = historyStore.get(String(d.id))
  if (!rec || !rec.source) return null
  // 校验记录里的 source/episode 在当前 detail 仍然有效
  const src = d.list?.find((s) => s.id === rec.source)
  if (!src) return null
  const idx = Math.max(0, rec.episodeIndex ?? 0)
  if (!src.linkList[idx]) return null
  return {
    source: rec.source,
    episodeIndex: idx,
    episodeName: src.linkList[idx]?.episode || String(idx + 1),
    currentTime: rec.currentTime ?? 0
  }
})
function resumeWatching(): void {
  const r = resumeRecord.value
  if (!r) return
  const d = detail.value
  if (!d) return
  router.push({
    path: '/play',
    query: {
      id: String(d.id),
      source: r.source,
      episode: String(r.episodeIndex),
      currentTime: r.currentTime > 0 ? String(r.currentTime) : undefined
    }
  })
}

/** ============== 收藏 ============== */
const isFavorited = computed(() => {
  const d = detail.value
  if (!d) return false
  return !!favoriteMap.value[String(d.id)]
})

function handleToggleFavorite(): void {
  const d = detail.value
  if (!d) return
  void favoriteStore.toggle({
    id: String(d.id),
    name: d.name,
    picture: d.picture,
    remarks: d.remarks ?? d.descriptor?.remarks,
    pid: d.pid,
    cid: d.cid
  })
}

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
            ratio="3/4"
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
          :style="heroBgStyle"
        />
        <div class="gf-detail__hero-mask" />

        <div class="gf-detail__hero-inner container-page">
          <!-- 海报 -->
          <div class="gf-detail__poster">
            <BaseImage
              :src="detail.picture"
              :alt="detail.name"
              ratio="3/4"
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

            <!-- hero meta: 只保留上映/地区/状态等紧凑字段, 演职人员下沉到独立 section -->
            <dl v-if="detail.descriptor?.releaseDate || detail.area || detail.descriptor?.area || detail.descriptor?.remarks || detail.remarks" class="gf-detail__meta">
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
              <!-- 续播优先, 有进度时主按钮变"继续观看" -->
              <BaseButton
                v-if="resumeRecord"
                variant="primary"
                size="lg"
                @click="resumeWatching"
              >
                <template #icon>
                  <BaseIcon name="play" size="1.1em" />
                </template>
                继续观看 · {{ resumeRecord.episodeName }}
              </BaseButton>
              <BaseButton
                :variant="resumeRecord ? 'outline' : 'primary'"
                size="lg"
                :disabled="!detail.list?.[0]?.linkList?.length"
                @click="playFirst"
              >
                <template #icon>
                  <BaseIcon name="play" size="1.1em" />
                </template>
                {{ resumeRecord ? '从头播放' : '立即播放' }}
              </BaseButton>
              <BaseButton
                :variant="isFavorited ? 'primary' : 'outline'"
                size="lg"
                @click="handleToggleFavorite"
              >
                <template #icon>
                  <BaseIcon name="heart" size="1.1em" />
                </template>
                {{ isFavorited ? '已收藏' : '收藏' }}
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

      <!-- 演职人员 (横滚卡片: 渐变首字头像 + 名字), bilibili/腾讯视频风格 -->
      <section
        v-if="directors.length || actors.length"
        class="gf-detail__cast container-page"
        aria-label="演职人员"
      >
        <h2 class="gf-detail__section-title">演职人员</h2>
        <div class="gf-detail__cast-scroll">
          <div
            v-for="(name, i) in directors"
            :key="'d-' + i"
            class="gf-detail__person"
          >
            <span
              class="gf-detail__person-avatar"
              :style="{ backgroundImage: personGradient(name) }"
              aria-hidden="true"
            >
              {{ personInitial(name) }}
            </span>
            <span class="gf-detail__person-name" :title="name">{{ name }}</span>
            <span class="gf-detail__person-role">导演</span>
          </div>
          <div
            v-for="(name, i) in actors"
            :key="'a-' + i"
            class="gf-detail__person"
          >
            <span
              class="gf-detail__person-avatar"
              :style="{ backgroundImage: personGradient(name) }"
              aria-hidden="true"
            >
              {{ personInitial(name) }}
            </span>
            <span class="gf-detail__person-name" :title="name">{{ name }}</span>
            <span class="gf-detail__person-role">主演</span>
          </div>
        </div>
      </section>

      <!-- 相关推荐 (选集职责已移交播放页, 详情页只做"看不看"决策) -->
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

/* ============= Cast (演职人员) ============= */
.gf-detail__cast {
  padding-block: var(--gf-space-8);
}
.gf-detail__cast-scroll {
  display: flex;
  gap: var(--gf-space-4);
  overflow-x: auto;
  scrollbar-width: thin;
  padding-block: var(--gf-space-2);
  margin-inline: calc(-1 * var(--gf-gutter-mobile));
  padding-inline: var(--gf-gutter-mobile);
}
@media (min-width: 768px) {
  .gf-detail__cast-scroll {
    margin-inline: calc(-1 * var(--gf-gutter-tablet));
    padding-inline: var(--gf-gutter-tablet);
    gap: var(--gf-space-5);
  }
}
@media (min-width: 1024px) {
  .gf-detail__cast-scroll {
    margin-inline: 0;
    padding-inline: 0;
  }
}
.gf-detail__cast-scroll::-webkit-scrollbar { height: 4px; }
.gf-detail__cast-scroll::-webkit-scrollbar-thumb {
  background-color: rgba(255, 255, 255, 0.18);
  border-radius: 2px;
}

.gf-detail__person {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  width: 72px;
  text-align: center;
}
@media (min-width: 768px) {
  .gf-detail__person {
    width: 84px;
  }
}

.gf-detail__person-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 9999px;
  font-size: 22px;
  font-weight: var(--gf-fw-bold);
  color: #fff;
  background-size: cover;
  background-position: center;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
}
@media (min-width: 768px) {
  .gf-detail__person-avatar {
    width: 68px;
    height: 68px;
    font-size: 26px;
  }
}

.gf-detail__person-name {
  font-size: var(--gf-fs-xs);
  color: var(--gf-text-primary);
  font-weight: var(--gf-fw-medium);
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-top: 6px;
}
.gf-detail__person-role {
  font-size: 10px;
  color: var(--gf-text-muted);
  line-height: 1;
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
