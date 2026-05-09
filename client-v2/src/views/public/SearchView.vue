<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import * as filmApi from '@/api/film'
import type { FilmListItem, BackendPage } from '@/types/film'
import FilmGrid from '@/components/film/FilmGrid.vue'
import { useQuerySync } from '@/composables/useQuerySync'
import { useAbortable } from '@/composables/useAbortable'
import { useViewMode } from '@/composables/useViewMode'

/**
 * /search?search=xxx&current=1
 * STORY-011 关键字搜索页：顶部搜索框 + 结果列表 + 分页
 */

const router = useRouter()
const { isMobile } = useViewMode()

const { params, push } = useQuerySync<{ search: string; current: number }>(
  { search: '', current: 1 },
  {
    path: '/search',
    onChange: () => {
      void load()
    }
  }
)

const inputKeyword = ref<string>(params.value.search)
const list = ref<FilmListItem[]>([])
const page = ref<BackendPage>({ pageSize: 10, current: 1, pageCount: 0, total: 0 })
const loading = ref(false)
const loaded = ref(false)
const errorMsg = ref<string>('')

const abortable = useAbortable()

const oldSearch = computed(() => params.value.search)

watch(
  () => params.value.search,
  (v) => {
    inputKeyword.value = v
  },
  { immediate: true }
)

async function load(): Promise<void> {
  const keyword = (params.value.search || '').trim()
  if (!keyword) {
    list.value = []
    page.value = { pageSize: 10, current: 1, pageCount: 0, total: 0 }
    loaded.value = true
    return
  }
  loading.value = true
  errorMsg.value = ''
  const signal = abortable.refresh()
  try {
    const resp = await filmApi.searchFilm(
      {
        keyword,
        current: params.value.current || 1
      },
      { signal }
    )
    list.value = resp.list ?? []
    page.value = resp.page ?? page.value
    loaded.value = true
  } catch (e) {
    // 401/网络错误已被 http 拦截器 toast；这里仅清空业务态
    if ((e as { name?: string })?.name === 'CanceledError') return
    list.value = []
    page.value = { pageSize: 10, current: 1, pageCount: 0, total: 0 }
    errorMsg.value = ''
    loaded.value = true
  } finally {
    loading.value = false
  }
}

function submitSearch(): void {
  const k = inputKeyword.value.trim()
  // 允许提交空字符串：清空 URL 中的 search，回到引导态
  if (!k) {
    if (params.value.search) {
      void push({ search: '', current: 1 })
    }
    return
  }
  void push({ search: k, current: 1 })
}

function onPaginate(p: number): void {
  void push({ search: params.value.search, current: p })
  if (typeof window !== 'undefined') {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

function play(id: string | number): void {
  void router.push({
    path: '/play',
    query: { id: String(id), source: '0', episode: '0' }
  })
}

// 首次进入或直接刷新带 query 的 URL：触发一次 load
void load()
</script>

<template>
  <div class="gf-search container-page py-[var(--gf-space-6)]">
    <!-- 顶部搜索框 -->
    <div class="gf-search__form mx-auto max-w-[640px] mb-[var(--gf-space-8)]">
      <div class="gf-search__input-wrap">
        <BaseIcon name="search" size="20px" class="gf-search__icon" />
        <input
          v-model="inputKeyword"
          class="gf-search__input"
          type="search"
          placeholder="输入关键字搜索 动漫 / 剧集 / 电影"
          aria-label="搜索影片"
          @keydown.enter.prevent="submitSearch"
        />
        <BaseButton
          variant="gradient"
          size="md"
          class="gf-search__btn"
          aria-label="搜索"
          @click="submitSearch"
        >
          <template #icon>
            <BaseIcon name="search" size="18px" />
          </template>
          <span class="hidden sm:inline">搜索</span>
        </BaseButton>
      </div>
    </div>

    <!-- 结果区 -->
    <section v-if="oldSearch" class="gf-search__result">
      <header v-if="!loading" class="mb-[var(--gf-space-6)]">
        <h2
          class="text-[var(--gf-fs-xl)] font-[var(--gf-fw-bold)] text-primary mb-[var(--gf-space-1)]"
        >
          {{ oldSearch }}
        </h2>
        <p class="text-secondary text-[var(--gf-fs-sm)]">
          找到 <strong class="text-primary">{{ page.total }}</strong> 部与
          "{{ oldSearch }}" 相关的影视作品
        </p>
      </header>

      <!-- loading 骨架 -->
      <div v-if="loading && list.length === 0" class="flex flex-col gap-[var(--gf-space-4)]">
        <BaseSkeleton :count="5" height="180px" />
      </div>

      <!-- 移动端：列表卡片 -->
      <div v-else-if="isMobile && list.length > 0" class="gf-search__list-mobile flex flex-col gap-[var(--gf-space-4)]">
        <article
          v-for="m in list"
          :key="String(m.id)"
          class="gf-search__row-mobile"
        >
          <RouterLink
            :to="{ path: '/filmDetail', query: { link: String(m.id) } }"
            class="gf-search__poster-link"
            data-focusable="true"
            tabindex="0"
            :aria-label="m.name"
          >
            <BaseImage :src="m.picture" :alt="m.name" ratio="2/3" fit="cover" />
          </RouterLink>
          <div class="gf-search__info">
            <h3 class="gf-search__name">{{ m.name }}</h3>
            <div class="gf-search__tags">
              <BaseTag v-if="m.cName" variant="brand" size="xs">{{ m.cName }}</BaseTag>
              <BaseTag v-if="m.year" size="xs">{{ m.year }}</BaseTag>
              <BaseTag v-if="m.area" size="xs">{{ m.area }}</BaseTag>
            </div>
            <p v-if="m.remarks" class="gf-search__line">{{ m.remarks }}</p>
            <BaseButton
              variant="gradient"
              size="sm"
              class="gf-search__play"
              @click.stop="play(m.id)"
            >
              <template #icon>
                <BaseIcon name="play" size="14px" />
              </template>
              立即播放
            </BaseButton>
          </div>
        </article>
      </div>

      <!-- 桌面：每行 2 个详情卡 -->
      <FilmGrid
        v-else-if="!isMobile && list.length > 0"
        :items="list"
        :gap="'var(--gf-space-5)'"
      >
        <template #item="{ item }">
          <article class="gf-search__row-desktop">
            <RouterLink
              :to="{ path: '/filmDetail', query: { link: String(item.id) } }"
              class="gf-search__poster-link"
              data-focusable="true"
              tabindex="0"
              :aria-label="item.name"
            >
              <BaseImage
                :src="item.picture"
                :alt="item.name"
                ratio="2/3"
                fit="cover"
              />
            </RouterLink>
            <div class="gf-search__info">
              <h3 class="gf-search__name">{{ item.name }}</h3>
              <div class="gf-search__tags">
                <BaseTag v-if="item.cName" variant="brand" size="xs">
                  {{ item.cName }}
                </BaseTag>
                <BaseTag v-if="item.year" size="xs">{{ item.year }}</BaseTag>
                <BaseTag v-if="item.area" size="xs">{{ item.area }}</BaseTag>
              </div>
              <p v-if="item.remarks" class="gf-search__line">{{ item.remarks }}</p>
              <BaseButton
                variant="gradient"
                size="sm"
                class="gf-search__play"
                @click.stop="play(item.id)"
              >
                <template #icon>
                  <BaseIcon name="play" size="14px" />
                </template>
                立即播放
              </BaseButton>
            </div>
          </article>
        </template>
      </FilmGrid>

      <!-- 分页 -->
      <div
        v-if="!loading && page.total > 0"
        class="gf-search__pagination mt-[var(--gf-space-8)] flex justify-center"
      >
        <BasePagination
          :current="page.current || 1"
          :page-size="page.pageSize || 10"
          :total="page.total"
          @change="onPaginate"
        />
      </div>

      <!-- 空状态 -->
      <BaseEmpty
        v-if="!loading && loaded && list.length === 0"
        title="未查询到对应影片"
        description="换一个关键词试试，或者从首页分类发现内容"
      >
        <template #action>
          <BaseButton variant="gradient" size="md" @click="router.push('/index')">
            返回首页
          </BaseButton>
        </template>
      </BaseEmpty>
    </section>

    <!-- 未输入关键字 -->
    <BaseEmpty
      v-else
      title="开始你的搜索"
      description="输入片名 / 演员 / 关键字，发现更多影视作品"
    />
  </div>
</template>

<style scoped>
.gf-search__input-wrap {
  position: relative;
  display: flex;
  align-items: center;
  background-color: var(--gf-bg-elevated);
  border-radius: var(--gf-radius-full);
  padding: 6px 6px 6px var(--gf-space-5);
  border: 1px solid transparent;
  transition: border-color var(--gf-dur-fast) var(--gf-ease-standard);
}
.gf-search__input-wrap:focus-within {
  border-color: var(--gf-brand-primary);
  box-shadow: var(--gf-shadow-purple-glow);
}

.gf-search__icon {
  color: var(--gf-text-muted);
  flex-shrink: 0;
  margin-right: var(--gf-space-3);
}

.gf-search__input {
  flex: 1;
  height: 44px;
  background-color: transparent;
  border: none;
  outline: none;
  color: var(--gf-text-primary);
  font-size: var(--gf-fs-md);
  padding: 0;
  min-width: 0;
}

.gf-search__input::placeholder {
  color: var(--gf-text-muted);
}

.gf-search__btn {
  flex-shrink: 0;
  margin-left: var(--gf-space-2);
}

.gf-search__row-desktop,
.gf-search__row-mobile {
  display: flex;
  gap: var(--gf-space-4);
  background-color: var(--gf-bg-elevated);
  border-radius: var(--gf-radius-lg);
  padding: var(--gf-space-4);
  transition: background-color var(--gf-dur-fast) var(--gf-ease-standard);
}

.gf-search__row-mobile {
  padding: var(--gf-space-3);
}

.gf-search__row-desktop:hover,
.gf-search__row-mobile:hover {
  background-color: rgba(255, 255, 255, 0.05);
}

.gf-search__poster-link {
  display: block;
  flex-shrink: 0;
  width: 120px;
  border-radius: var(--gf-radius-md);
  overflow: hidden;
  outline: none;
}
.gf-search__row-desktop .gf-search__poster-link {
  width: 160px;
}
.gf-search__poster-link:focus-visible {
  box-shadow: var(--gf-shadow-focus-ring);
}

.gf-search__info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--gf-space-2);
}

.gf-search__name {
  font-size: var(--gf-fs-lg);
  font-weight: var(--gf-fw-bold);
  color: var(--gf-text-primary);
  line-height: var(--gf-lh-snug);
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.gf-search__tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--gf-space-1);
}

.gf-search__line {
  color: var(--gf-text-secondary);
  font-size: var(--gf-fs-sm);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.gf-search__play {
  align-self: flex-start;
  margin-top: auto;
}

@media (max-width: 767px) {
  .gf-search__poster-link {
    width: 100px;
  }
  .gf-search__name {
    font-size: var(--gf-fs-md);
  }
}
</style>

<style>
[data-mode='tv'] .gf-search__input {
  font-size: var(--gf-fs-lg);
  height: 56px;
}
[data-mode='tv'] .gf-search__poster-link:focus-visible {
  box-shadow: var(--gf-shadow-focus-ring), var(--gf-shadow-hover);
}
</style>
