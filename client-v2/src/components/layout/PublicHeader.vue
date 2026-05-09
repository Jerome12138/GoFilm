<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useSiteStore, useNavStore, useHistoryStore } from '@/stores'
import BaseIcon from '@/components/base/BaseIcon.vue'

/**
 * 公开端 Header
 *
 * - 透明 → 滚动后实色（监听 window scroll）
 * - 左：站名（gradient）
 * - 中：搜索框（pill）
 * - 右：导航 + 历史浮层 + 移动端汉堡
 * - 移动端：搜索收成图标 + 抽屉式导航
 * - TV：高度 96px，导航字号放大
 *
 * 数据：useSiteStore / useNavStore / useHistoryStore（不在此处 ensureLoaded，由 App.vue 预热）
 */

const route = useRoute()
const router = useRouter()

const siteStore = useSiteStore()
const navStore = useNavStore()
const historyStore = useHistoryStore()

const { basic } = storeToRefs(siteStore)
const { list: navList } = storeToRefs(navStore)
const { list: historyList } = storeToRefs(historyStore)

/** 滚动 → 切实色背景 */
const scrolled = ref(false)
function onScroll(): void {
  scrolled.value = (window.scrollY ?? 0) > 12
}
onMounted(() => {
  onScroll()
  window.addEventListener('scroll', onScroll, { passive: true })
})
onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll)
})

/** 站名 */
const siteName = computed(() => basic.value?.siteName || 'GoFilm')

/** 顶部 6 项导航 */
const topNav = computed(() => navList.value.slice(0, 6))

/** 搜索 */
const keyword = ref<string>(typeof route.query.search === 'string' ? route.query.search : '')
function submitSearch(): void {
  const k = keyword.value.trim()
  if (!k) return
  router.push({ path: '/search', query: { search: k } })
  // 移动端搜索后收起抽屉与移动搜索面板
  mobileSearchOpen.value = false
  mobileMenuOpen.value = false
}

/** 移动端抽屉 */
const mobileMenuOpen = ref(false)
const mobileSearchOpen = ref(false)
function closeMobile(): void {
  mobileMenuOpen.value = false
  mobileSearchOpen.value = false
}

/** 历史浮层（hover 显示，移动端用点击） */
const historyOpen = ref(false)
let historyCloseTimer: number | null = null
function openHistory(): void {
  if (historyCloseTimer !== null) {
    window.clearTimeout(historyCloseTimer)
    historyCloseTimer = null
  }
  historyOpen.value = true
}
function deferCloseHistory(): void {
  if (historyCloseTimer !== null) {
    window.clearTimeout(historyCloseTimer)
  }
  historyCloseTimer = window.setTimeout(() => {
    historyOpen.value = false
    historyCloseTimer = null
  }, 200)
}
function toggleHistory(): void {
  historyOpen.value = !historyOpen.value
}

/** 历史前 8 条（list 已是 timeStamp desc，最新在前） */
const historyTop = computed(() =>
  historyList.value.slice(0, 8).map((it) => ({
    id: it.id,
    name: it.name,
    picture: it.picture,
    link: it.link,
    source: it.source ?? '',
    episode: it.episode
  }))
)

function goHistoryItem(item: {
  id: string
  link?: string
  source?: string
  episode: string
}): void {
  historyOpen.value = false
  // link 形如 /play?id=...&source=...&episode=...&currentTime=...
  // 直接 router.push 字符串路径，可一次性带上 currentTime 续播
  if (item.link) {
    router.push(item.link)
    return
  }
  router.push({
    path: '/play',
    query: { id: item.id, source: item.source, episode: item.episode }
  })
}

/** 当前路由分类 Pid 高亮 */
const activePid = computed<number | null>(() => {
  const pid = route.query.Pid
  const n = Number(pid)
  return Number.isFinite(n) && n > 0 ? n : null
})

/** route 跳转后自动关闭抽屉 */
function isNavActive(id: number): boolean {
  return activePid.value === id
}
</script>

<template>
  <header
    class="gf-header"
    :class="scrolled ? 'gf-header--scrolled' : 'gf-header--top'"
    :data-route="route.name as string | undefined"
  >
    <div class="gf-header__inner container-page flex items-center gap-[var(--gf-space-4)]">
      <!-- 移动端汉堡 -->
      <button
        class="gf-header__icon-btn md:hidden"
        type="button"
        aria-label="打开菜单"
        @click="mobileMenuOpen = !mobileMenuOpen"
      >
        <BaseIcon :name="mobileMenuOpen ? 'close' : 'menu'" size="22px" />
      </button>

      <!-- 站名 -->
      <RouterLink
        to="/index"
        class="gf-header__brand text-brand-gradient"
        :aria-label="siteName"
        @click="closeMobile"
      >
        {{ siteName }}
      </RouterLink>

      <!-- 主导航（桌面 / TV） -->
      <nav class="gf-header__nav hidden md:flex items-center gap-[var(--gf-space-5)]">
        <RouterLink
          to="/index"
          class="gf-header__nav-link"
          :class="route.name === 'home' ? 'is-active' : ''"
          data-focusable="true"
        >
          首页
        </RouterLink>
        <RouterLink
          v-for="nav in topNav"
          :key="nav.id"
          :to="{ path: '/filmClassify', query: { Pid: nav.id } }"
          class="gf-header__nav-link"
          :class="isNavActive(nav.id) ? 'is-active' : ''"
          data-focusable="true"
        >
          {{ nav.name }}
        </RouterLink>
      </nav>

      <!-- 中部弹性 -->
      <div class="flex-1" />

      <!-- 桌面搜索框 -->
      <form
        class="gf-header__search hidden md:flex items-center"
        role="search"
        @submit.prevent="submitSearch"
      >
        <BaseIcon name="search" size="18px" class="gf-header__search-icon" />
        <input
          v-model="keyword"
          type="search"
          placeholder="搜索影片、剧集、动漫…"
          aria-label="搜索"
          class="gf-header__search-input"
        />
      </form>

      <!-- 移动端搜索图标 -->
      <button
        class="gf-header__icon-btn md:hidden"
        type="button"
        aria-label="搜索"
        @click="mobileSearchOpen = !mobileSearchOpen"
      >
        <BaseIcon :name="mobileSearchOpen ? 'close' : 'search'" size="22px" />
      </button>

      <!-- 历史按钮 + 浮层 -->
      <div
        class="gf-header__history relative hidden md:block"
        @mouseenter="openHistory"
        @mouseleave="deferCloseHistory"
      >
        <button
          class="gf-header__icon-btn"
          type="button"
          aria-label="观看历史"
          aria-haspopup="menu"
          :aria-expanded="historyOpen"
          @click="toggleHistory"
        >
          <BaseIcon name="history" size="22px" />
        </button>
        <Transition name="gf-fade">
          <div
            v-if="historyOpen"
            class="gf-header__history-panel"
            role="menu"
            @mouseenter="openHistory"
            @mouseleave="deferCloseHistory"
          >
            <div class="gf-header__history-title">
              <span>最近观看</span>
              <RouterLink
                to="/history"
                class="text-link text-[var(--gf-fs-sm)]"
                @click="historyOpen = false"
              >
                全部
              </RouterLink>
            </div>
            <ul v-if="historyTop.length" class="gf-header__history-list">
              <li
                v-for="item in historyTop"
                :key="item.id + item.source + item.episode"
              >
                <button
                  type="button"
                  class="gf-header__history-item"
                  data-focusable="true"
                  @click="goHistoryItem(item)"
                >
                  <img
                    v-if="item.picture"
                    :src="item.picture"
                    :alt="item.name"
                    loading="lazy"
                    class="gf-header__history-thumb"
                  />
                  <span class="gf-header__history-meta">
                    <span class="gf-header__history-name">{{ item.name }}</span>
                    <span class="gf-header__history-ep">第 {{ item.episode || '1' }} 集</span>
                  </span>
                </button>
              </li>
            </ul>
            <div v-else class="gf-header__history-empty">
              暂无观看记录
            </div>
          </div>
        </Transition>
      </div>
    </div>

    <!-- 移动端搜索条（展开） -->
    <Transition name="gf-slide-down">
      <form
        v-if="mobileSearchOpen"
        class="gf-header__mobile-search md:hidden"
        role="search"
        @submit.prevent="submitSearch"
      >
        <BaseIcon name="search" size="18px" class="gf-header__search-icon" />
        <input
          v-model="keyword"
          type="search"
          placeholder="搜索影片、剧集、动漫…"
          aria-label="搜索"
          class="gf-header__search-input"
          autofocus
        />
      </form>
    </Transition>

    <!-- 移动端抽屉 -->
    <Transition name="gf-slide-down">
      <nav
        v-if="mobileMenuOpen"
        class="gf-header__mobile-nav md:hidden"
        aria-label="主导航"
      >
        <RouterLink
          to="/index"
          class="gf-header__mobile-link"
          :class="route.name === 'home' ? 'is-active' : ''"
          @click="closeMobile"
        >
          首页
        </RouterLink>
        <RouterLink
          v-for="nav in topNav"
          :key="nav.id"
          :to="{ path: '/filmClassify', query: { Pid: nav.id } }"
          class="gf-header__mobile-link"
          :class="isNavActive(nav.id) ? 'is-active' : ''"
          @click="closeMobile"
        >
          {{ nav.name }}
        </RouterLink>
        <RouterLink
          to="/history"
          class="gf-header__mobile-link"
          @click="closeMobile"
        >
          观看历史
        </RouterLink>
      </nav>
    </Transition>
  </header>
</template>

<style scoped>
.gf-header {
  position: sticky;
  top: 0;
  z-index: var(--gf-z-header);
  width: 100%;
  transition:
    background-color var(--gf-dur-base) var(--gf-ease-standard),
    backdrop-filter var(--gf-dur-base) var(--gf-ease-standard),
    border-color var(--gf-dur-base) var(--gf-ease-standard);
  border-bottom: 1px solid transparent;
}

.gf-header--top {
  background-color: rgba(11, 11, 15, 0);
  background-image: linear-gradient(
    180deg,
    rgba(0, 0, 0, 0.55) 0%,
    rgba(0, 0, 0, 0) 100%
  );
}

.gf-header--scrolled {
  background-color: var(--gf-bg-header-scrolled);
  backdrop-filter: blur(18px) saturate(140%);
  -webkit-backdrop-filter: blur(18px) saturate(140%);
  border-bottom-color: var(--gf-border-subtle);
}

.gf-header__inner {
  height: 56px;
}

@media (min-width: 768px) {
  .gf-header__inner {
    height: 64px;
  }
}

.gf-header__brand {
  font-family: var(--gf-font-display);
  font-size: var(--gf-fs-xl);
  font-weight: var(--gf-fw-black);
  letter-spacing: var(--gf-tracking-tight);
  text-decoration: none;
  white-space: nowrap;
  outline: none;
  border-radius: var(--gf-radius-md);
  padding: 4px 6px;
  margin-left: -6px;
}

.gf-header__nav {
  margin-left: var(--gf-space-4);
}

.gf-header__nav-link {
  position: relative;
  display: inline-flex;
  align-items: center;
  height: 40px;
  padding: 0 4px;
  color: var(--gf-text-secondary);
  font-size: var(--gf-fs-sm);
  font-weight: var(--gf-fw-medium);
  text-decoration: none;
  white-space: nowrap;
  transition: color var(--gf-dur-fast) var(--gf-ease-standard);
  border-radius: var(--gf-radius-sm);
  outline: none;
}

.gf-header__nav-link:hover,
.gf-header__nav-link:focus-visible {
  color: var(--gf-text-primary);
}

.gf-header__nav-link.is-active {
  color: var(--gf-text-primary);
  font-weight: var(--gf-fw-semibold);
}

.gf-header__nav-link.is-active::after {
  content: '';
  position: absolute;
  left: 4px;
  right: 4px;
  bottom: 4px;
  height: 2px;
  background-image: var(--gf-brand-gradient);
  border-radius: 2px;
}

/* 搜索 */
.gf-header__search {
  position: relative;
  height: 40px;
  width: 280px;
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: var(--gf-radius-full);
  padding: 0 14px 0 38px;
  transition:
    background-color var(--gf-dur-fast) var(--gf-ease-standard),
    border-color var(--gf-dur-fast) var(--gf-ease-standard),
    box-shadow var(--gf-dur-fast) var(--gf-ease-standard);
}

.gf-header__search:focus-within {
  background-color: rgba(255, 255, 255, 0.12);
  border-color: rgba(255, 255, 255, 0.24);
  box-shadow: 0 0 0 3px rgba(74, 209, 229, 0.25);
}

.gf-header__search-icon {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--gf-text-muted);
  pointer-events: none;
}

.gf-header__search-input {
  flex: 1;
  height: 100%;
  background: transparent;
  border: none;
  outline: none;
  color: var(--gf-text-primary);
  font-size: var(--gf-fs-sm);
}

.gf-header__search-input::placeholder {
  color: var(--gf-text-muted);
}

@media (min-width: 1440px) {
  .gf-header__search {
    width: 360px;
  }
}

/* 图标按钮 */
.gf-header__icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  min-width: 44px;
  background: transparent;
  border: none;
  border-radius: var(--gf-radius-md);
  color: var(--gf-text-secondary);
  cursor: pointer;
  transition:
    background-color var(--gf-dur-fast) var(--gf-ease-standard),
    color var(--gf-dur-fast) var(--gf-ease-standard);
}

.gf-header__icon-btn:hover,
.gf-header__icon-btn:focus-visible {
  background-color: rgba(255, 255, 255, 0.08);
  color: var(--gf-text-primary);
}

/* 历史浮层 */
.gf-header__history-panel {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  width: 320px;
  background-color: var(--gf-bg-surface);
  border: 1px solid var(--gf-border-subtle);
  border-radius: var(--gf-radius-lg);
  box-shadow: var(--gf-shadow-lg);
  padding: var(--gf-space-3);
  z-index: var(--gf-z-dropdown);
}

.gf-header__history-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--gf-space-2) var(--gf-space-2);
  font-size: var(--gf-fs-sm);
  font-weight: var(--gf-fw-semibold);
  color: var(--gf-text-primary);
  border-bottom: 1px solid var(--gf-border-subtle);
}

.gf-header__history-list {
  list-style: none;
  margin: 0;
  padding: var(--gf-space-2) 0 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 360px;
  overflow-y: auto;
}

.gf-header__history-item {
  display: flex;
  width: 100%;
  align-items: center;
  gap: var(--gf-space-3);
  padding: var(--gf-space-2);
  background: transparent;
  border: none;
  border-radius: var(--gf-radius-md);
  color: var(--gf-text-primary);
  cursor: pointer;
  text-align: left;
  transition: background-color var(--gf-dur-fast) var(--gf-ease-standard);
}

.gf-header__history-item:hover,
.gf-header__history-item:focus-visible {
  background-color: rgba(255, 255, 255, 0.06);
  outline: none;
}

.gf-header__history-thumb {
  width: 56px;
  height: 36px;
  object-fit: cover;
  border-radius: var(--gf-radius-sm);
  background-color: var(--gf-bg-elevated);
  flex-shrink: 0;
}

.gf-header__history-meta {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.gf-header__history-name {
  font-size: var(--gf-fs-sm);
  font-weight: var(--gf-fw-medium);
  color: var(--gf-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gf-header__history-ep {
  font-size: var(--gf-fs-xs);
  color: var(--gf-text-muted);
  margin-top: 2px;
}

.gf-header__history-empty {
  padding: var(--gf-space-6) var(--gf-space-2);
  text-align: center;
  color: var(--gf-text-muted);
  font-size: var(--gf-fs-sm);
}

/* 移动端搜索条 / 抽屉 */
.gf-header__mobile-search {
  position: relative;
  display: flex;
  align-items: center;
  margin: 0 var(--gf-gutter-mobile) var(--gf-space-3);
  height: 40px;
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: var(--gf-radius-full);
  padding: 0 14px 0 38px;
}

.gf-header__mobile-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--gf-space-2) var(--gf-gutter-mobile) var(--gf-space-3);
  background-color: rgba(11, 11, 15, 0.96);
  border-bottom: 1px solid var(--gf-border-subtle);
}

.gf-header__mobile-link {
  display: flex;
  align-items: center;
  height: 48px;
  padding: 0 var(--gf-space-3);
  border-radius: var(--gf-radius-md);
  color: var(--gf-text-secondary);
  font-size: var(--gf-fs-md);
  font-weight: var(--gf-fw-medium);
  text-decoration: none;
}

.gf-header__mobile-link:hover,
.gf-header__mobile-link:focus-visible,
.gf-header__mobile-link.is-active {
  background-color: rgba(255, 255, 255, 0.06);
  color: var(--gf-text-primary);
}

.gf-header__mobile-link.is-active {
  background-image: linear-gradient(
    90deg,
    rgba(155, 73, 231, 0.18),
    rgba(74, 209, 229, 0.08)
  );
}

/* 过渡 */
.gf-fade-enter-active,
.gf-fade-leave-active {
  transition: opacity var(--gf-dur-fast) var(--gf-ease-standard),
    transform var(--gf-dur-fast) var(--gf-ease-standard);
}
.gf-fade-enter-from,
.gf-fade-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

.gf-slide-down-enter-active,
.gf-slide-down-leave-active {
  transition: opacity var(--gf-dur-base) var(--gf-ease-standard),
    transform var(--gf-dur-base) var(--gf-ease-standard);
  overflow: hidden;
}
.gf-slide-down-enter-from,
.gf-slide-down-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>

<style>
/* TV 模式覆盖：高度放大、字号放大、强制实色背景（避免透明导航被忽略） */
[data-mode='tv'] .gf-header__inner {
  height: 96px;
  padding-inline: var(--gf-tv-safe);
}
[data-mode='tv'] .gf-header__brand {
  font-size: var(--gf-fs-2xl);
}
[data-mode='tv'] .gf-header__nav-link {
  height: 56px;
  font-size: var(--gf-fs-md);
  padding: 0 var(--gf-space-3);
}
[data-mode='tv'] .gf-header__icon-btn {
  width: 56px;
  height: 56px;
}
[data-mode='tv'] .gf-header__search {
  height: 56px;
  width: 420px;
  font-size: var(--gf-fs-md);
}
[data-mode='tv'] .gf-header__search-input {
  font-size: var(--gf-fs-md);
}
/* TV 默认实色 header（不依赖滚动），避免与 hero 冲突 */
[data-mode='tv'] .gf-header {
  background-color: var(--gf-bg-header-scrolled);
  border-bottom-color: var(--gf-border-subtle);
}
[data-mode='tv'] .gf-header--top {
  background-image: none;
}
</style>
