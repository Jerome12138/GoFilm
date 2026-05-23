<script setup lang="ts">
/**
 * 播放页 PlayView
 *
 * 路由：/play?id=&source=&episode=&currentTime=
 *
 * 数据：filmApi.getPlayInfo({ id, playFrom, episode })
 *   响应：{ detail, current, currentPlayFrom, currentEpisode, relate }
 *
 * 本视图职责：
 *  1. 加载并渲染播放器（usePlayer + 原生 <video>）
 *  2. 标题行：影片名 / 当前集 / 标签 / 自动播放开关 / 下一集按钮
 *  3. EpisodeTabs 切换播放源 & 集数（不重建 player，仅 player.src(...)）
 *  4. RelatedList 相关推荐
 *  5. 键盘 / D-pad 快捷键（空格 暂停 / 左右 ±10s / 上下 音量 / Esc 返回）
 *  6. 历史记录写入（卸载 / beforeunload / 切换 episode 都写一次）
 *  7. 错误处理：API 失败 → BaseEmpty + 返回首页；视频源 error → on-page banner + 自动尝试下一个 source
 */

import {
  computed,
  nextTick,
  onMounted,
  ref,
  watch
} from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { filmApi } from '@/api'
import type { PlayInfo, PlaySource } from '@/types/film'
import { usePlayer } from '@/composables/usePlayer'
import { useFilmHistory, buildPlayLink } from '@/composables/useFilmHistory'
import { useHistoryStore } from '@/stores/history'
import { useViewMode } from '@/composables/useViewMode'
import { useNetworkHint } from '@/composables/useNetworkHint'
import { useLocalLikes } from '@/composables/useLocalLikes'
import { useFavoriteStore } from '@/stores/favorite'
import { storeToRefs } from 'pinia'
import { normalizeDpadKey } from '@/utils/dpad'
import EpisodeTabs from '@/components/film/EpisodeTabs.vue'
import RelatedList from '@/components/film/RelatedList.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseEmpty from '@/components/base/BaseEmpty.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'
import BaseTag from '@/components/base/BaseTag.vue'
import posterFallback from '@/assets/play.svg'

const route = useRoute()
const router = useRouter()
const { isTV } = useViewMode()
const historyStore = useHistoryStore()
// 弱网感知: 决定 player 初始化参数 + 错误重试策略
const { isSlow: isSlowNetwork } = useNetworkHint()
const favoriteStore = useFavoriteStore()
const { map: favoriteMap } = storeToRefs(favoriteStore)

/* ============ bilibili 三连操作条 ============ */

/** 点赞 — 走 useLocalLikes composable (后端无接口, localStorage 持久化) */
const likes = useLocalLikes()
const liked = computed(() => {
  const id = detail.value?.id
  return id !== undefined ? likes.isLiked(id).value : false
})
const likeText = computed(() => (liked.value ? '已点赞' : '点赞'))
function toggleLike(): void {
  const id = detail.value?.id
  if (id === undefined) return
  likes.toggle(id)
}

/** 收藏: 走真后端 (favoriteStore) */
const favorited = computed(() => {
  const id = detail.value?.id
  return id !== undefined && !!favoriteMap.value[String(id)]
})
function toggleFavorite(): void {
  const d = detail.value
  if (!d) return
  void favoriteStore.toggle({
    id: String(d.id),
    name: d.name,
    picture: d.picture,
    remarks: d.remarks,
    pid: d.pid,
    cid: d.cid
  })
}

/** 分享: 复制当前 URL 到剪贴板, 短暂展示"已复制"反馈 */
const shareLabel = ref<string>('分享')
async function handleShare(): Promise<void> {
  const url = typeof window !== 'undefined' ? window.location.href : ''
  if (!url) return
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(url)
    } else {
      // fallback: 极老浏览器 / 非 secure context
      const t = document.createElement('textarea')
      t.value = url
      document.body.appendChild(t)
      t.select()
      document.execCommand('copy')
      document.body.removeChild(t)
    }
    shareLabel.value = '已复制'
    window.setTimeout(() => {
      shareLabel.value = '分享'
    }, 1800)
  } catch {
    shareLabel.value = '复制失败'
    window.setTimeout(() => {
      shareLabel.value = '分享'
    }, 1800)
  }
}

/** ---------- 数据状态 ---------- */
const loading = ref(true)
const loadError = ref<string>('')
const detail = ref<PlayInfo['detail'] | null>(null)
const relate = ref<PlayInfo['relate']>([])
/** 当前选中的播放源 ID（与 detail.list[i].id 对应） */
const currentSourceId = ref<string>('')
/** 当前集索引 */
const currentEpisodeIndex = ref<number>(0)
/** 是否自动播放下一集 */
const autoPlayNext = ref<boolean>(true)
/** 视频源错误提示文本 */
const videoErrorMsg = ref<string>('')

/** ---------- 派生 ---------- */
const currentSource = computed<PlaySource | null>(() => {
  if (!detail.value) return null
  return (
    detail.value.list.find((s) => s.id === currentSourceId.value) ??
    detail.value.list[0] ??
    null
  )
})

const currentEpisode = computed(() => {
  const src = currentSource.value
  if (!src) return null
  return src.linkList[currentEpisodeIndex.value] ?? src.linkList[0] ?? null
})

const currentSrc = ref<string>('')
const currentSrcType = ref<string>('')

/**
 * 广告过滤开关 (localStorage 持久化).
 * 开启后, 若当前 src 是 m3u8, 重写为 `${API}/m3u8/proxy?src=<encoded>`,
 * 由后端拉源后剔除疑似广告 segment 再回吐, video.js 透明消费.
 * 非 m3u8 (mp4 / flv 等) 不重写, 走原始 URL.
 */
const AD_FILTER_LS_KEY = 'gf-ad-filter'
const adFilter = ref<boolean>(
  (() => {
    try {
      return localStorage.getItem(AD_FILTER_LS_KEY) === '1'
    } catch {
      return false
    }
  })()
)
watch(adFilter, (v) => {
  try {
    localStorage.setItem(AD_FILTER_LS_KEY, v ? '1' : '0')
  } catch {
    /* 隐私模式忽略 */
  }
})

const API_BASE = (import.meta.env.VITE_API_BASE as string | undefined) || '/api'
const reM3u8 = /\.m3u8(\?|#|$)/i

/** 实际喂给 player 的 src; adFilter 开启且原 src 是 m3u8 时改走代理. */
const effectiveSrc = computed<string>(() => {
  const s = currentSrc.value
  if (!s || !adFilter.value) return s
  if (!reM3u8.test(s)) return s
  return `${API_BASE}/m3u8/proxy?src=${encodeURIComponent(s)}`
})

/**
 * adFilter 切换会让 effectiveSrc 变化, 进而触发 video.js 重新 load → 进度归零.
 * 这里在切换前抓一下当前 currentTime, 切完后 (next loadedmetadata) 跳回去, 体感无中断.
 */
watch(adFilter, () => {
  if (!playerReady.value || !player.value) return
  const resume = playerCurrentTime.value
  if (resume <= 0) return
  const off = onPlayerEvent('loadedmetadata', () => {
    const p = player.value
    if (!p) return
    try {
      p.currentTime(resume)
      void playerPlay()
    } catch {
      /* ignore */
    }
    off()
  })
})

const hasNext = computed(() => {
  const src = currentSource.value
  if (!src) return false
  return currentEpisodeIndex.value < src.linkList.length - 1
})

const hasPrev = computed(() => currentEpisodeIndex.value > 0)

const tagList = computed<string[]>(() => {
  if (!detail.value) return []
  const d = detail.value.descriptor
  const tags: string[] = []
  if (d.cName) tags.push(d.cName)
  if (d.classTag) {
    for (const t of String(d.classTag).split(',')) {
      const v = t.trim()
      if (v) tags.push(v)
    }
  }
  if (d.year) tags.push(String(d.year))
  if (d.area) tags.push(String(d.area))
  return tags.slice(0, 6)
})

const filmId = computed(() => String(route.query.id ?? ''))

/** 已观看链接：基于 history store 推算（同 source 中 0..episodeIndex 全部视为已观看） */
const watchedLinks = computed<string[]>(() => {
  if (!detail.value) return []
  const id = String(detail.value.id)
  const rec = historyStore.get(id)
  if (!rec || !rec.source) return []
  const src = detail.value.list.find((s) => s.id === rec.source)
  if (!src) return []
  const upto = Math.max(0, rec.episodeIndex ?? 0)
  return src.linkList.slice(0, upto + 1).map((e) => e.link)
})

/** ---------- 播放器 ---------- */
const videoEl = ref<HTMLVideoElement | null>(null)
const {
  init: initPlayer,
  dispose: disposePlayer,
  play: playerPlay,
  pause: playerPause,
  seekBy: playerSeekBy,
  setVolume: playerSetVolume,
  on: onPlayerEvent,
  player,
  paused,
  currentTime: playerCurrentTime,
  ready: playerReady
} = usePlayer({
  src: effectiveSrc,
  type: currentSrcType,
  poster: ref<string | undefined>(posterFallback),
  autoplay: false,
  volume: 0.6,
  playbackRates: [0.5, 1.0, 1.25, 1.5, 2.0],
  // 弱网友好默认: preload metadata 而非 auto, 起播更快; 弱网时启用 VHS 低带宽预设
  preload: 'metadata',
  lowBandwidth: isSlowNetwork.value
})

/** ---------- 历史记录 ---------- */
const { flush: flushHistory } = useFilmHistory({
  collect: () => {
    if (!detail.value || !currentEpisode.value) {
      return null
    }
    const link = buildPlayLink({
      id: detail.value.id,
      source: currentSourceId.value,
      episodeIndex: currentEpisodeIndex.value,
      currentTime: playerCurrentTime.value
    })
    return {
      id: String(detail.value.id),
      name: detail.value.name,
      link,
      episode: currentEpisode.value.episode,
      picture: detail.value.picture,
      source: currentSourceId.value,
      episodeIndex: currentEpisodeIndex.value,
      currentTime: Math.floor(playerCurrentTime.value),
      pid: detail.value.pid,
      cid: detail.value.cid
    }
  }
})

/** ---------- 数据加载 ---------- */
async function loadPlayInfo(): Promise<void> {
  const id = String(route.query.id ?? '')
  const playFrom = String(route.query.source ?? '')
  const episode = String(route.query.episode ?? '0')

  if (!id) {
    loadError.value = '缺少影片 ID'
    loading.value = false
    return
  }

  loading.value = true
  loadError.value = ''
  videoErrorMsg.value = ''

  try {
    const data: PlayInfo = await filmApi.getPlayInfo({
      id,
      playFrom,
      episode
    })
    if (!data || !data.detail) {
      loadError.value = '播放信息为空'
      loading.value = false
      return
    }
    detail.value = data.detail
    relate.value = data.relate ?? []
    currentSourceId.value = data.currentPlayFrom || data.detail.list[0]?.id || ''
    currentEpisodeIndex.value = Number(data.currentEpisode) || 0
    // 续播进度优先级：URL query.currentTime > history store（同 source 同 episode 才匹配）
    let resumeAt = Number(route.query.currentTime) || 0
    if (!resumeAt) {
      const rec = historyStore.get(id)
      if (
        rec &&
        rec.source === currentSourceId.value &&
        rec.episodeIndex === currentEpisodeIndex.value &&
        (rec.currentTime ?? 0) > 0
      ) {
        resumeAt = rec.currentTime ?? 0
      }
    }
    applyCurrentEpisodeToPlayer(resumeAt)
    loading.value = false
  } catch {
    // http 拦截器已 toast，这里只设页面态
    loadError.value = '播放信息加载失败'
    loading.value = false
  }
}

/** 把当前 episode 的 link 写入播放器 src ref（player composable 会响应式切换） */
function applyCurrentEpisodeToPlayer(resumeAt = 0): void {
  const ep = currentEpisode.value
  if (!ep) return
  currentSrc.value = ep.link
  currentSrcType.value = ''
  // 等待 player ready + src 切完再 seek + 自动播放
  if (resumeAt > 0) {
    const off = onPlayerEvent('loadedmetadata', () => {
      const p = player.value
      if (!p) return
      try {
        p.currentTime(resumeAt)
      } catch {
        // ignore
      }
      off()
    })
  }
}

/** ---------- 集数 / 源切换 ---------- */
function changeSource(sourceId: string): void {
  if (!detail.value) return
  const next = detail.value.list.find((s) => s.id === sourceId)
  if (!next) return
  // 切换源时重置集数为 0
  selectEpisode({ sourceId, episodeIndex: 0 })
}

function selectEpisode(payload: { sourceId: string; episodeIndex: number }): void {
  if (!detail.value) return
  const src = detail.value.list.find((s) => s.id === payload.sourceId)
  if (!src) return
  const ep = src.linkList[payload.episodeIndex]
  if (!ep) return

  // 用户切换源/集, 视为新的尝试, 重置错误重试计数
  resetRetry()
  // 切换前先把当前进度写历史
  flushHistory()

  currentSourceId.value = payload.sourceId
  currentEpisodeIndex.value = payload.episodeIndex
  applyCurrentEpisodeToPlayer(0)

  // 同步 router query（不刷新页面，仅替换 URL，保证刷新后能恢复）
  void router.replace({
    path: '/play',
    query: {
      id: String(detail.value.id),
      source: payload.sourceId,
      episode: String(payload.episodeIndex)
    }
  })

  // 切换后尝试自动播放（用户已交互）
  void nextTick(() => {
    void playerPlay()
  })
}

function playNext(): void {
  if (!hasNext.value) return
  selectEpisode({
    sourceId: currentSourceId.value,
    episodeIndex: currentEpisodeIndex.value + 1
  })
}

function playPrev(): void {
  if (!hasPrev.value) return
  selectEpisode({
    sourceId: currentSourceId.value,
    episodeIndex: currentEpisodeIndex.value - 1
  })
}

/** ---------- 键盘 / D-pad ---------- */
function handleKeydown(e: KeyboardEvent): void {
  // 输入框聚焦时不拦截
  const target = e.target as HTMLElement | null
  const tag = target?.tagName?.toLowerCase()
  if (tag === 'input' || tag === 'textarea' || target?.isContentEditable) {
    return
  }
  const key = normalizeDpadKey(e)
  switch (key) {
    case ' ':
    case 'Spacebar':
    case 'Enter': {
      // OK / 空格：暂停 / 播放（仅当焦点不在按钮上）
      if (tag === 'button' || tag === 'a') {
        return
      }
      e.preventDefault()
      if (paused.value) {
        void playerPlay()
      } else {
        playerPause()
      }
      break
    }
    case 'ArrowLeft':
      e.preventDefault()
      playerSeekBy(-10)
      break
    case 'ArrowRight':
      e.preventDefault()
      playerSeekBy(10)
      break
    case 'ArrowUp': {
      e.preventDefault()
      const cur = player.value?.volume() ?? 0.6
      playerSetVolume(Math.min(1, cur + 0.05))
      break
    }
    case 'ArrowDown': {
      e.preventDefault()
      const cur = player.value?.volume() ?? 0.6
      playerSetVolume(Math.max(0, cur - 0.05))
      break
    }
    case 'Escape':
      // 返回详情页
      e.preventDefault()
      goBackToDetail()
      break
    default:
      break
  }
}

function goBackToDetail(): void {
  const id = filmId.value
  if (id) {
    void router.push({ path: '/filmDetail', query: { link: id } })
    return
  }
  void router.push('/index')
}

/** ---------- 视频源错误处理 ----------
 * 旧实现见到 error 立刻换源, 但弱网下临时超时也会触发 error,
 * 直接换源用户体验是"莫名其妙跳到下一个源". 现在改:
 *   1. 同 (source, episode) 先原地重试 2 次, 退避 1.5s / 4s
 *   2. 仍失败再换下一个源 (老逻辑)
 *   3. canplay 触发时重置计数, 避免历史错误累积
 */
const MAX_SAME_SOURCE_RETRIES = 2
const RETRY_DELAYS_MS = [1500, 4000]
const retryCount = ref<number>(0)
let retryTimer: number | undefined

function resetRetry(): void {
  retryCount.value = 0
  if (retryTimer !== undefined) {
    window.clearTimeout(retryTimer)
    retryTimer = undefined
  }
}

function handleVideoError(): void {
  if (!detail.value) return
  const p = player.value
  if (!p) return

  if (retryCount.value < MAX_SAME_SOURCE_RETRIES) {
    const delay = RETRY_DELAYS_MS[retryCount.value] ?? 4000
    retryCount.value += 1
    videoErrorMsg.value = `加载失败, ${Math.round(delay / 1000)} 秒后第 ${retryCount.value} 次重试…`
    if (retryTimer !== undefined) {
      window.clearTimeout(retryTimer)
    }
    retryTimer = window.setTimeout(() => {
      retryTimer = undefined
      const next = effectiveSrc.value
      if (!next || !player.value) return
      const cur = currentTimePersisted.value // 记忆当前时间, 重试后跳回去
      try {
        player.value.src({ src: next, type: currentSrcType.value || guessMime(next) })
        if (cur > 0) {
          const off = onPlayerEvent('loadedmetadata', () => {
            try {
              player.value?.currentTime(cur)
            } catch {
              /* ignore */
            }
            off()
          })
        }
        void playerPlay()
      } catch {
        // src 调用本身失败极少见; 直接落到换源
        fallbackSwitchSource()
      }
    }, delay)
    return
  }

  // 同源重试上限, 换下一个源
  resetRetry()
  fallbackSwitchSource()
}

function fallbackSwitchSource(): void {
  if (!detail.value) return
  videoErrorMsg.value = '当前播放源不可用，正在尝试切换…'
  const sources = detail.value.list
  const curIdx = sources.findIndex((s) => s.id === currentSourceId.value)
  if (curIdx < 0 || sources.length <= 1) return
  const nextIdx = (curIdx + 1) % sources.length
  if (nextIdx === curIdx) return
  const targetSource = sources[nextIdx]
  if (!targetSource) return
  const targetEpisodeIdx = Math.min(
    currentEpisodeIndex.value,
    Math.max(0, targetSource.linkList.length - 1)
  )
  selectEpisode({ sourceId: targetSource.id, episodeIndex: targetEpisodeIdx })
}

/** 记忆错误发生时的播放进度, 重试后用. */
const currentTimePersisted = computed(() => {
  try {
    return player.value?.currentTime() ?? 0
  } catch {
    return 0
  }
})

function guessMime(url: string): string {
  if (/\.m3u8(\?|#|$)/i.test(url)) return 'application/x-mpegURL'
  if (/\.mp4(\?|#|$)/i.test(url)) return 'video/mp4'
  if (/\.webm(\?|#|$)/i.test(url)) return 'video/webm'
  return ''
}

/** ---------- 生命周期 ---------- */
onMounted(async () => {
  await loadPlayInfo()
  // 等 DOM 渲染完成后再 init player
  await nextTick()
  if (videoEl.value) {
    initPlayer(videoEl.value)
    onPlayerEvent('ended', () => {
      if (autoPlayNext.value && hasNext.value) {
        playNext()
      }
    })
    onPlayerEvent('error', () => {
      handleVideoError()
    })
    onPlayerEvent('canplay', () => {
      videoErrorMsg.value = ''
      resetRetry()
    })
  }
  if (typeof window !== 'undefined') {
    window.addEventListener('keydown', handleKeydown)
  }
})

onBeforeRouteLeave(() => {
  flushHistory()
  // 卸载播放器（onScopeDispose 也会兜底，但提前 dispose 可避免短暂的画面残留）
  disposePlayer()
  if (typeof window !== 'undefined') {
    window.removeEventListener('keydown', handleKeydown)
  }
})

/** 监听 query 变化（仅 id / source / episode），仅在 path 仍为 /play 时响应；
 *  selectEpisode 内部会调用 router.replace 主动同步 query —— 此 watch 在那种情况下
 *  仅做一次幂等检查，发现 store state 与 query 已一致则跳过。 */
watch(
  () => [route.query.id, route.query.source, route.query.episode],
  ([qId, qSource, qEpisode]) => {
    if (route.path !== '/play') return
    if (!detail.value) return
    if (String(qId ?? '') !== String(detail.value.id)) {
      // 影片切换 → 重新拉取
      void loadPlayInfo()
      return
    }
    const wantSource = String(qSource ?? '')
    const wantEpisode = Number(qEpisode ?? 0)
    if (
      wantSource &&
      (wantSource !== currentSourceId.value || wantEpisode !== currentEpisodeIndex.value)
    ) {
      const src = detail.value.list.find((s) => s.id === wantSource)
      if (!src) return
      const ep = src.linkList[wantEpisode]
      if (!ep) return
      currentSourceId.value = wantSource
      currentEpisodeIndex.value = wantEpisode
      videoErrorMsg.value = ''
      applyCurrentEpisodeToPlayer(Number(route.query.currentTime) || 0)
    }
  }
)

/** TV 模式下首次进入页面把焦点放到播放器（playerReady 后） */
watch(playerReady, (v) => {
  if (v && isTV.value) {
    nextTick(() => {
      videoEl.value?.focus()
    })
  }
})
</script>

<template>
  <div class="gf-play-view container-page py-[var(--gf-space-6)]">
    <!-- 顶部返回 / 面包屑 -->
    <div class="gf-play-view__breadcrumb mb-[var(--gf-space-4)]">
      <BaseButton
        variant="ghost"
        size="sm"
        @click="goBackToDetail"
      >
        <template #icon>
          <BaseIcon name="arrow-left" size="18px" />
        </template>
        返回详情
      </BaseButton>
    </div>

    <!-- 错误：API 失败 -->
    <BaseEmpty
      v-if="loadError && !loading"
      :title="loadError"
      description="影片暂时无法播放，请稍后重试"
    >
      <template #action>
        <BaseButton variant="gradient" size="lg" @click="router.push('/index')">
          返回首页
        </BaseButton>
      </template>
    </BaseEmpty>

    <!-- 主内容 -->
    <template v-if="!loadError">
      <!-- 主栅格: lg+ 左视频/简介 + 右选集; 小屏单栏堆叠 -->
      <div class="gf-play-grid">
        <section class="gf-play-grid__main flex flex-col gap-[var(--gf-space-5)]">
          <!-- 播放器容器 -->
          <div class="gf-player-wrap" :data-loading="loading ? '1' : '0'">
            <video
              ref="videoEl"
              class="video-js vjs-default-skin gf-player"
              playsinline
              tabindex="0"
            />
            <div v-if="loading" class="gf-player-loading">
              <span class="gf-player-loading__dot" />
              <span class="gf-player-loading__dot" />
              <span class="gf-player-loading__dot" />
            </div>
            <div v-if="videoErrorMsg" class="gf-player-error" role="alert">
              {{ videoErrorMsg }}
            </div>
          </div>

          <!-- 当前播放信息 + 控件 -->
          <header
            v-if="detail"
            class="gf-play-info flex flex-col md:flex-row md:items-center md:justify-between gap-[var(--gf-space-3)]"
          >
        <div class="flex flex-col gap-[var(--gf-space-2)] min-w-0">
          <h1 class="gf-play-info__title text-[var(--gf-fs-xl)] font-[var(--gf-fw-bold)] text-primary leading-[var(--gf-lh-snug)]">
            {{ detail.name }}
            <span v-if="currentEpisode" class="gf-play-info__episode ml-[var(--gf-space-2)] text-secondary text-[var(--gf-fs-md)]">
              · {{ currentEpisode.episode }}
            </span>
          </h1>
          <div class="flex flex-wrap items-center gap-[var(--gf-space-2)]">
            <BaseTag
              v-for="t in tagList"
              :key="t"
              variant="default"
              size="sm"
            >
              {{ t }}
            </BaseTag>
            <!-- 详情已不再展示选集等冗余, 想看完整剧情/演员/导演 → 跳详情页 -->
            <RouterLink
              :to="{ path: '/filmDetail', query: { link: String(detail.id) } }"
              class="gf-play-info__detail-link"
            >
              查看完整介绍 ›
            </RouterLink>
          </div>
        </div>

        <div class="flex items-center gap-[var(--gf-space-2)]">
          <BaseButton
            variant="outline"
            size="md"
            :class="adFilter ? 'gf-toggle--on' : ''"
            :aria-pressed="adFilter"
            :title="adFilter ? '已开启: m3u8 走服务端代理过滤广告' : '点开后服务端代理 m3u8 并剔除疑似广告片段'"
            @click="adFilter = !adFilter"
          >
            <template #icon>
              <BaseIcon name="magic" size="18px" />
            </template>
            过滤广告
          </BaseButton>
          <BaseButton
            variant="outline"
            size="md"
            :class="autoPlayNext ? 'gf-toggle--on' : ''"
            :aria-pressed="autoPlayNext"
            @click="autoPlayNext = !autoPlayNext"
          >
            <template #icon>
              <BaseIcon name="autoplay" size="18px" />
            </template>
            自动连播
          </BaseButton>
          <BaseButton
            variant="gradient"
            size="md"
            :disabled="!hasNext"
            @click="playNext"
          >
            <template #icon>
              <BaseIcon name="skip-next" size="18px" />
            </template>
            下一集
          </BaseButton>
        </div>
      </header>

      <!-- bilibili 风格三连操作条: 点赞 (本地) / 收藏 (真接口) / 分享 (clipboard) -->
      <div v-if="detail" class="gf-play-actions flex items-center gap-[var(--gf-space-6)]">
        <button
          type="button"
          class="gf-play-action"
          :class="liked ? 'gf-play-action--on' : ''"
          :aria-pressed="liked"
          @click="toggleLike"
        >
          <BaseIcon name="heart" size="22px" />
          <span class="gf-play-action__label">{{ likeText }}</span>
        </button>
        <button
          type="button"
          class="gf-play-action"
          :class="favorited ? 'gf-play-action--on' : ''"
          :aria-pressed="favorited"
          @click="toggleFavorite"
        >
          <BaseIcon name="star" size="22px" />
          <span class="gf-play-action__label">{{ favorited ? '已收藏' : '收藏' }}</span>
        </button>
        <button
          type="button"
          class="gf-play-action"
          @click="handleShare"
        >
          <BaseIcon name="share" size="22px" />
          <span class="gf-play-action__label">{{ shareLabel }}</span>
        </button>
      </div>

          <!-- 选集 (视频下方主栏内, bilibili 风格: 用户看完本集向下扫即可继续) -->
          <EpisodeTabs
            v-if="detail"
            :sources="detail.list"
            :current-source-id="currentSourceId"
            :current-episode="currentEpisode?.link ?? ''"
            :watched-links="watchedLinks"
            @change-source="changeSource"
            @select="selectEpisode"
          />

          <!-- 剧情简介 -->
          <section
            v-if="detail?.descriptor.content"
            class="gf-play-synopsis flex flex-col gap-[var(--gf-space-2)]"
          >
            <h2 class="text-[var(--gf-fs-lg)] font-[var(--gf-fw-semibold)] text-primary">剧情简介</h2>
            <p class="text-[var(--gf-fs-sm)] text-secondary leading-[var(--gf-lh-relaxed)]">
              {{ detail.descriptor.content }}
            </p>
          </section>
        </section>

        <!-- 右侧相关推荐 sticky (bilibili 风格), 不抢占主区视频 + 选集的视线 -->
        <aside v-if="relate.length" class="gf-play-grid__aside">
          <RelatedList :items="relate" title="相关推荐" />
        </aside>
      </div>
    </template>
  </div>
</template>

<style scoped>
.gf-play-view {
  min-height: 60vh;
}

/* 播放器容器：16:9 自适应 */
.gf-player-wrap {
  position: relative;
  width: 100%;
  background-color: #000;
  border-radius: var(--gf-radius-lg);
  overflow: hidden;
  aspect-ratio: 16 / 9;
  box-shadow: var(--gf-shadow-xl);
}

.gf-player {
  position: absolute;
  inset: 0;
  width: 100% !important;
  height: 100% !important;
  outline: none;
}

.gf-player:focus,
.gf-player:focus-visible {
  outline: none;
}

/* 桌面端最大宽度（>= 1280 居中） */
@media (min-width: 1280px) {
  .gf-player-wrap {
    max-width: 1280px;
    margin: 0 auto;
  }
}

/* 加载占位：3 个跳动圆点 */
.gf-player-loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--gf-space-2);
  pointer-events: none;
}
.gf-player-loading__dot {
  width: 10px;
  height: 10px;
  border-radius: 9999px;
  background-color: rgba(255, 255, 255, 0.5);
  animation: gf-play-dot 1s ease-in-out infinite;
}
.gf-player-loading__dot:nth-child(2) {
  animation-delay: 0.15s;
}
.gf-player-loading__dot:nth-child(3) {
  animation-delay: 0.3s;
}
@keyframes gf-play-dot {
  0%, 80%, 100% {
    opacity: 0.3;
    transform: translateY(0);
  }
  40% {
    opacity: 1;
    transform: translateY(-4px);
  }
}

.gf-player-error {
  position: absolute;
  left: var(--gf-space-3);
  bottom: var(--gf-space-3);
  padding: var(--gf-space-2) var(--gf-space-3);
  background-color: rgba(0, 0, 0, 0.65);
  color: #fff;
  font-size: var(--gf-fs-sm);
  border-radius: var(--gf-radius-md);
  z-index: 6;
  pointer-events: none;
}

/* 当前播放信息块 */
.gf-play-info__title :deep(a) {
  color: inherit;
  text-decoration: none;
}

/* 自动连播开关激活态 */
.gf-toggle--on {
  color: var(--gf-brand-primary) !important;
  border-color: var(--gf-brand-primary) !important;
}

/* 主体栅格：移动 / 平板 单列；桌面 1024+ 双列 */
/* 标题旁"查看完整介绍"链接 */
.gf-play-info__detail-link {
  display: inline-flex;
  align-items: center;
  color: var(--gf-text-link);
  font-size: var(--gf-fs-xs);
  text-decoration: none;
  padding: 2px 8px;
  border-radius: var(--gf-radius-sm);
  transition: color var(--gf-dur-fast) var(--gf-ease-standard);
}
.gf-play-info__detail-link:hover,
.gf-play-info__detail-link:focus-visible {
  color: var(--gf-text-link-hover);
  background-color: rgba(74, 209, 229, 0.08);
  outline: none;
}

/* bilibili 三连操作条 */
.gf-play-actions {
  margin-top: var(--gf-space-5);
}
.gf-play-action {
  display: inline-flex;
  align-items: center;
  gap: var(--gf-space-2);
  background: transparent;
  border: none;
  padding: var(--gf-space-2) var(--gf-space-1);
  color: var(--gf-text-secondary);
  font-size: var(--gf-fs-sm);
  font-weight: var(--gf-fw-medium);
  cursor: pointer;
  transition: color var(--gf-dur-fast) var(--gf-ease-standard);
  outline: none;
  border-radius: var(--gf-radius-sm);
}
.gf-play-action:hover {
  color: var(--gf-text-primary);
}
.gf-play-action--on {
  color: var(--gf-brand-cyan);
}
.gf-play-action__label {
  font-size: var(--gf-fs-xs);
}
.gf-play-action:focus-visible {
  box-shadow: var(--gf-shadow-focus-ring);
}

.gf-play-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: var(--gf-space-6);
}

/* 大屏: 左视频+简介, 右选集. 选集 sticky 跟随滚动 */
@media (min-width: 1024px) {
  .gf-play-grid {
    grid-template-columns: minmax(0, 2.4fr) minmax(280px, 1fr);
    align-items: start;
  }
  .gf-play-grid__aside {
    position: sticky;
    top: var(--gf-space-6);
    max-height: calc(100vh - var(--gf-space-6) * 2);
    overflow-y: auto;
    padding-right: var(--gf-space-1); /* 留滚动条空间, 防内容被挤 */
  }
}

.gf-play-grid__main {
  min-width: 0;
}
.gf-play-grid__aside {
  min-width: 0;
}

.gf-play-synopsis {
  background-color: var(--gf-bg-surface);
  border-radius: var(--gf-radius-md);
  padding: var(--gf-space-4);
}

/* video.js 控件按钮去除白边 */
:deep(video) {
  outline: none !important;
}
:deep(.vjs-tech) {
  border-radius: var(--gf-radius-lg);
}
:deep(.vjs-control-bar) {
  background-color: rgba(0, 0, 0, 0.55);
  font-size: 14px;
}
:deep(.vjs-big-play-button) {
  height: 2em;
  width: 2em;
  line-height: 2em;
  border-radius: 50%;
  border: none;
  background-color: rgba(0, 0, 0, 0.6);
  top: 50%;
  left: 50%;
  margin-top: -1em;
  margin-left: -1em;
}
:deep(.vjs-play-progress) {
  background-color: var(--gf-brand-primary);
}
:deep(.vjs-load-progress div) {
  background-color: rgba(255, 255, 255, 0.45);
}
:deep(.vjs-slider) {
  background-color: rgba(255, 255, 255, 0.18);
}

/* 移动端：标题块换行 + 按钮组靠右 */
@media (max-width: 767px) {
  .gf-play-info {
    align-items: flex-start;
  }
}
</style>

<style>
/* TV 模式覆盖 */
[data-mode='tv'] .gf-player-wrap {
  border-radius: 16px;
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.7);
}
[data-mode='tv'] .gf-play-view {
  padding-block: var(--gf-tv-safe);
  padding-inline: var(--gf-tv-safe);
}
[data-mode='tv'] .gf-play-view .video-js .vjs-control-bar {
  font-size: 18px;
  height: 4em;
}
[data-mode='tv'] .gf-play-view .vjs-big-play-button {
  height: 3em;
  width: 3em;
  line-height: 3em;
  margin-top: -1.5em;
  margin-left: -1.5em;
}
</style>
