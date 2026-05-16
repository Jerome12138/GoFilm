<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useSiteStore, useNavStore, useHistoryStore } from '@/stores'
import { useUserStore } from '@/stores/user'
import { useViewMode } from '@/composables/useViewMode'
import BaseIcon from '@/components/base/BaseIcon.vue'
import BaseDialog from '@/components/base/BaseDialog.vue'

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
const userStore = useUserStore()
const { isTV } = useViewMode()

const { basic } = storeToRefs(siteStore)
const { list: navList } = storeToRefs(navStore)
const { list: historyList } = storeToRefs(historyStore)
const { isLoggedIn, isAdmin, displayName, info: userInfo } = storeToRefs(userStore)

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

/** TV 历史 Dialog（全屏抽屉，替代 hover 浮层） */
const tvHistoryDialog = ref(false)
function openTvHistory(): void {
  tvHistoryDialog.value = true
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

/** 用户菜单（已登录） */
const userMenuOpen = ref(false)
let userMenuTimer: number | null = null
function openUserMenu(): void {
  if (userMenuTimer !== null) {
    window.clearTimeout(userMenuTimer)
    userMenuTimer = null
  }
  userMenuOpen.value = true
}
function deferCloseUserMenu(): void {
  if (userMenuTimer !== null) {
    window.clearTimeout(userMenuTimer)
  }
  userMenuTimer = window.setTimeout(() => {
    userMenuOpen.value = false
    userMenuTimer = null
  }, 200)
}
function toggleUserMenu(): void {
  userMenuOpen.value = !userMenuOpen.value
}
function closeUserMenu(): void {
  userMenuOpen.value = false
  if (userMenuTimer !== null) {
    window.clearTimeout(userMenuTimer)
    userMenuTimer = null
  }
}

const userAvatar = computed(() => {
  const a = userInfo.value?.avatar
  if (a && a !== 'empty') return a
  // 默认根据用户名生成 dicebear avatar，保持稳定
  const seed = userInfo.value?.userName || userInfo.value?.username || 'guest'
  return `https://api.dicebear.com/7.x/avataaars/svg?seed=${encodeURIComponent(seed)}`
})

function gotoLogin(): void {
  closeMobile()
  closeUserMenu()
  router.push({
    path: '/login',
    query: { redirect: route.fullPath }
  })
}

async function handleLogout(): Promise<void> {
  closeUserMenu()
  await userStore.logout()
  // 退出后留在当前页面（如果是受保护页则前面 401 拦截器已处理跳转）
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
        data-focusable="true"
        tabindex="0"
        @click="mobileMenuOpen = !mobileMenuOpen"
      >
        <BaseIcon :name="mobileMenuOpen ? 'close' : 'menu'" size="22px" />
      </button>

      <!-- 站名 -->
      <RouterLink
        to="/index"
        class="gf-header__brand text-brand-gradient"
        :aria-label="siteName"
        data-focusable="true"
        tabindex="0"
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
          tabindex="0"
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
          data-focusable="true"
          tabindex="0"
        />
      </form>

      <!-- 移动端搜索图标 -->
      <button
        class="gf-header__icon-btn md:hidden"
        type="button"
        aria-label="搜索"
        data-focusable="true"
        tabindex="0"
        @click="mobileSearchOpen = !mobileSearchOpen"
      >
        <BaseIcon :name="mobileSearchOpen ? 'close' : 'search'" size="22px" />
      </button>

      <!-- 历史按钮 + 浮层（TV 模式走 Dialog 抽屉） -->
      <div
        class="gf-header__history relative hidden md:block"
        @mouseenter="!isTV && openHistory()"
        @mouseleave="!isTV && deferCloseHistory()"
      >
        <button
          class="gf-header__icon-btn"
          type="button"
          aria-label="观看历史"
          aria-haspopup="menu"
          data-focusable="true"
          tabindex="0"
          :aria-expanded="isTV ? tvHistoryDialog : historyOpen"
          @click="isTV ? openTvHistory() : toggleHistory()"
        >
          <BaseIcon name="history" size="22px" />
        </button>
        <Transition name="gf-fade">
          <div
            v-if="historyOpen && !isTV"
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

      <!-- 用户菜单：未登录显示"登录"按钮，已登录显示头像 dropdown -->
      <button
        v-if="!isLoggedIn"
        class="gf-header__login-btn"
        type="button"
        data-focusable="true"
        tabindex="0"
        @click="gotoLogin"
      >
        <BaseIcon name="user" size="18px" />
        <span class="hidden md:inline">登录</span>
      </button>

      <div
        v-else
        class="gf-header__user relative"
        @mouseenter="!isTV && openUserMenu()"
        @mouseleave="!isTV && deferCloseUserMenu()"
      >
        <button
          class="gf-header__user-btn"
          type="button"
          data-focusable="true"
          tabindex="0"
          aria-haspopup="menu"
          :aria-expanded="userMenuOpen"
          @click="toggleUserMenu"
        >
          <img
            :src="userAvatar"
            :alt="displayName"
            class="gf-header__avatar"
          />
          <span class="gf-header__username hidden lg:inline">
            {{ displayName }}
          </span>
          <BaseIcon name="chevron-down" size="14px" class="hidden lg:inline" />
        </button>

        <Transition name="gf-fade">
          <div
            v-if="userMenuOpen"
            class="gf-header__user-panel"
            role="menu"
            @mouseenter="openUserMenu"
            @mouseleave="deferCloseUserMenu"
          >
            <div class="gf-header__user-header">
              <img :src="userAvatar" :alt="displayName" class="gf-header__user-avatar" />
              <div class="gf-header__user-info">
                <div class="gf-header__user-name">{{ displayName }}</div>
                <div class="gf-header__user-role">
                  {{ isAdmin ? '管理员' : '普通用户' }}
                </div>
              </div>
            </div>

            <RouterLink
              to="/history"
              class="gf-header__user-item"
              data-focusable="true"
              tabindex="0"
              @click="closeUserMenu"
            >
              <BaseIcon name="history" size="16px" />
              观看历史
            </RouterLink>

            <RouterLink
              to="/favorites"
              class="gf-header__user-item"
              data-focusable="true"
              tabindex="0"
              @click="closeUserMenu"
            >
              <BaseIcon name="heart" size="16px" />
              我的收藏
            </RouterLink>

            <RouterLink
              v-if="isAdmin"
              to="/manage/index"
              class="gf-header__user-item"
              data-focusable="true"
              tabindex="0"
              @click="closeUserMenu"
            >
              <BaseIcon name="settings" size="16px" />
              后台管理
            </RouterLink>

            <button
              type="button"
              class="gf-header__user-item gf-header__user-item--danger"
              data-focusable="true"
              tabindex="0"
              @click="handleLogout"
            >
              <BaseIcon name="logout" size="16px" />
              退出登录
            </button>
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
          data-focusable="true"
          tabindex="0"
          autofocus
        />
      </form>
    </Transition>

    <!-- 移动端抽屉: 左侧浮层 + 遮罩, 不占文档流不再把内容向下推 -->
    <Transition name="gf-mobile-overlay-fade">
      <div
        v-if="mobileMenuOpen"
        class="gf-header__mobile-overlay md:hidden"
        @click="closeMobile"
      />
    </Transition>
    <Transition name="gf-slide-left">
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
        <RouterLink
          to="/favorites"
          class="gf-header__mobile-link"
          @click="closeMobile"
        >
          我的收藏
        </RouterLink>
        <RouterLink
          v-if="isLoggedIn && isAdmin"
          to="/manage/index"
          class="gf-header__mobile-link"
          @click="closeMobile"
        >
          后台管理
        </RouterLink>
        <RouterLink
          v-if="!isLoggedIn"
          :to="{ path: '/login', query: { redirect: route.fullPath } }"
          class="gf-header__mobile-link"
          @click="closeMobile"
        >
          登录
        </RouterLink>
        <button
          v-else
          type="button"
          class="gf-header__mobile-link gf-header__mobile-link--danger"
          @click="closeMobile(); handleLogout()"
        >
          退出登录（{{ displayName }}）
        </button>
      </nav>
    </Transition>
  </header>

  <!-- TV 模式：历史抽屉 Dialog（全屏宽度，避免 hover 浮层依赖鼠标） -->
  <BaseDialog
    v-if="isTV"
    v-model:visible="tvHistoryDialog"
    title="最近观看"
    width="800px"
  >
    <ul v-if="historyTop.length" class="gf-tv-history__list">
      <li
        v-for="item in historyTop"
        :key="item.id + item.source + item.episode"
      >
        <button
          type="button"
          class="gf-tv-history__item"
          data-focusable="true"
          tabindex="0"
          @click="goHistoryItem(item); tvHistoryDialog = false"
        >
          <img
            v-if="item.picture"
            :src="item.picture"
            :alt="item.name"
            loading="lazy"
            class="gf-tv-history__thumb"
          />
          <span class="gf-tv-history__meta">
            <span class="gf-tv-history__name">{{ item.name }}</span>
            <span class="gf-tv-history__ep">第 {{ item.episode || '1' }} 集</span>
          </span>
        </button>
      </li>
    </ul>
    <div v-else class="gf-tv-history__empty">暂无观看记录</div>
    <template #footer>
      <RouterLink
        to="/history"
        class="text-link"
        data-focusable="true"
        tabindex="0"
        @click="tvHistoryDialog = false"
      >
        查看全部历史
      </RouterLink>
    </template>
  </BaseDialog>
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

/* 用户菜单 / 登录按钮 */
.gf-header__login-btn {
  display: inline-flex;
  align-items: center;
  gap: var(--gf-space-2);
  height: 36px;
  padding: 0 var(--gf-space-3);
  border-radius: var(--gf-radius-full);
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.14);
  color: var(--gf-text-primary);
  font-size: var(--gf-fs-sm);
  font-weight: var(--gf-fw-medium);
  cursor: pointer;
  transition:
    background-color var(--gf-dur-fast) var(--gf-ease-standard),
    border-color var(--gf-dur-fast) var(--gf-ease-standard);
}
.gf-header__login-btn:hover,
.gf-header__login-btn:focus-visible {
  background-color: rgba(255, 255, 255, 0.14);
  border-color: rgba(255, 255, 255, 0.24);
  outline: none;
}

.gf-header__user-btn {
  display: inline-flex;
  align-items: center;
  gap: var(--gf-space-2);
  height: 40px;
  padding: 2px var(--gf-space-2);
  border-radius: var(--gf-radius-full);
  background: transparent;
  border: 1px solid transparent;
  color: var(--gf-text-secondary);
  cursor: pointer;
  transition:
    background-color var(--gf-dur-fast) var(--gf-ease-standard),
    border-color var(--gf-dur-fast) var(--gf-ease-standard);
}
.gf-header__user-btn:hover,
.gf-header__user-btn:focus-visible {
  background-color: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.14);
  color: var(--gf-text-primary);
  outline: none;
}

.gf-header__avatar {
  width: 32px;
  height: 32px;
  border-radius: 9999px;
  object-fit: cover;
  background-color: var(--gf-bg-elevated);
  border: 1px solid rgba(255, 255, 255, 0.14);
}

.gf-header__username {
  font-size: var(--gf-fs-sm);
  font-weight: var(--gf-fw-medium);
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gf-header__user-panel {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  width: 240px;
  background-color: var(--gf-bg-surface);
  border: 1px solid var(--gf-border-subtle);
  border-radius: var(--gf-radius-lg);
  box-shadow: var(--gf-shadow-lg);
  padding: var(--gf-space-3);
  z-index: var(--gf-z-dropdown);
}

.gf-header__user-header {
  display: flex;
  align-items: center;
  gap: var(--gf-space-3);
  padding: var(--gf-space-2);
  border-bottom: 1px solid var(--gf-border-subtle);
  margin-bottom: var(--gf-space-2);
}
.gf-header__user-avatar {
  width: 44px;
  height: 44px;
  border-radius: 9999px;
  object-fit: cover;
  background-color: var(--gf-bg-elevated);
}
.gf-header__user-info {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}
.gf-header__user-name {
  font-size: var(--gf-fs-md);
  font-weight: var(--gf-fw-semibold);
  color: var(--gf-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gf-header__user-role {
  font-size: var(--gf-fs-xs);
  color: var(--gf-text-muted);
  margin-top: 2px;
}

.gf-header__user-item {
  display: flex;
  width: 100%;
  align-items: center;
  gap: var(--gf-space-3);
  padding: var(--gf-space-3) var(--gf-space-2);
  background: transparent;
  border: none;
  border-radius: var(--gf-radius-md);
  color: var(--gf-text-secondary);
  font-size: var(--gf-fs-sm);
  cursor: pointer;
  text-align: left;
  text-decoration: none;
  transition: background-color var(--gf-dur-fast) var(--gf-ease-standard);
}
.gf-header__user-item:hover,
.gf-header__user-item:focus-visible {
  background-color: rgba(255, 255, 255, 0.06);
  color: var(--gf-text-primary);
  outline: none;
}
.gf-header__user-item--danger {
  color: var(--gf-danger);
}
.gf-header__user-item--danger:hover {
  background-color: rgba(255, 71, 87, 0.12);
}

.gf-header__mobile-link--danger {
  color: var(--gf-danger);
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

/* 移动端抽屉: 从左滑出, fixed 定位不占文档流, 不再向下挤压主内容 */
.gf-header__mobile-overlay {
  position: fixed;
  inset: 0;
  background-color: rgba(0, 0, 0, 0.5);
  z-index: 80;
}

.gf-header__mobile-nav {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: min(82vw, 320px);
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--gf-space-4) var(--gf-space-3) var(--gf-space-4);
  background-color: rgba(11, 11, 15, 0.98);
  border-right: 1px solid var(--gf-border-subtle);
  box-shadow: 12px 0 32px rgba(0, 0, 0, 0.5);
  z-index: 90;
  overflow-y: auto;
  /* 给顶部留出 header 高度, 让用户视觉上能"看到 header 还在" */
  padding-top: calc(var(--gf-header-height, 60px) + var(--gf-space-3));
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

/* 移动端抽屉: 从左侧滑入 + 遮罩淡入 */
.gf-slide-left-enter-active,
.gf-slide-left-leave-active {
  transition: transform var(--gf-dur-base) var(--gf-ease-standard);
}
.gf-slide-left-enter-from,
.gf-slide-left-leave-to {
  transform: translateX(-100%);
}
.gf-mobile-overlay-fade-enter-active,
.gf-mobile-overlay-fade-leave-active {
  transition: opacity var(--gf-dur-base) var(--gf-ease-standard);
}
.gf-mobile-overlay-fade-enter-from,
.gf-mobile-overlay-fade-leave-to {
  opacity: 0;
}
</style>

<style>
/* TV history Dialog 内容 */
.gf-tv-history__list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--gf-space-3);
  max-height: 60vh;
  overflow-y: auto;
}
.gf-tv-history__item {
  display: flex;
  width: 100%;
  align-items: center;
  gap: var(--gf-space-3);
  padding: var(--gf-space-3);
  background: var(--gf-bg-elevated);
  border: none;
  border-radius: var(--gf-radius-md);
  color: var(--gf-text-primary);
  cursor: pointer;
  text-align: left;
  outline: none;
  transition: background-color var(--gf-dur-fast) var(--gf-ease-standard);
}
.gf-tv-history__item:focus,
.gf-tv-history__item:focus-visible {
  outline: none;
  background-color: rgba(255, 255, 255, 0.08);
}
.gf-tv-history__thumb {
  width: 96px;
  height: 64px;
  object-fit: cover;
  border-radius: var(--gf-radius-sm);
  background-color: var(--gf-bg-base);
  flex-shrink: 0;
}
.gf-tv-history__meta {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}
.gf-tv-history__name {
  font-size: var(--gf-fs-md);
  font-weight: var(--gf-fw-semibold);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gf-tv-history__ep {
  font-size: var(--gf-fs-sm);
  color: var(--gf-text-muted);
  margin-top: 4px;
}
.gf-tv-history__empty {
  padding: var(--gf-space-8) var(--gf-space-2);
  text-align: center;
  color: var(--gf-text-muted);
  font-size: var(--gf-fs-md);
}

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

/* TV 焦点环：导航 / 图标 / 搜索 */
[data-mode='tv'] .gf-header__nav-link:focus,
[data-mode='tv'] .gf-header__nav-link:focus-visible {
  outline: none;
  box-shadow: 0 0 0 4px var(--gf-brand-cyan);
  border-radius: var(--gf-radius-sm);
  color: var(--gf-text-primary);
}
[data-mode='tv'] .gf-header__icon-btn:focus,
[data-mode='tv'] .gf-header__icon-btn:focus-visible {
  outline: none;
  box-shadow: 0 0 0 4px var(--gf-brand-cyan);
  background-color: rgba(255, 255, 255, 0.12);
  color: var(--gf-text-primary);
}
[data-mode='tv'] .gf-header__search:focus-within {
  box-shadow: 0 0 0 4px var(--gf-brand-cyan);
  background-color: rgba(255, 255, 255, 0.12);
}
[data-mode='tv'] .gf-header__brand:focus,
[data-mode='tv'] .gf-header__brand:focus-visible {
  outline: none;
  box-shadow: 0 0 0 4px var(--gf-brand-cyan);
}
</style>
