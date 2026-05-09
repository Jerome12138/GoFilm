<script setup lang="ts">
import { computed, ref } from 'vue'
import * as filmApi from '@/api/film'
import type {
  ClassifySearchResp,
  ClassifyTagItem,
  FilmListItem
} from '@/types/film'
import FilmGrid from '@/components/film/FilmGrid.vue'
import FilmFilterBar, {
  type FilterGroup
} from '@/components/film/FilmFilterBar.vue'
import { useQuerySync } from '@/composables/useQuerySync'
import { useAbortable } from '@/composables/useAbortable'

/**
 * /filmClassifySearch?Pid=&Category=&Plot=&Area=&Language=&Year=&Sort=&current=
 * STORY-012 分类筛选页
 *
 * 注意：query 字段大小写严格保留旧站约定
 *  Pid / Category / Plot / Area / Language / Year / Sort / current
 */

type QueryShape = {
  Pid: string
  Category: string
  Plot: string
  Area: string
  Language: string
  Year: string
  Sort: string
  current: number
} & Record<string, string | number>

const initial: QueryShape = {
  Pid: '',
  Category: '',
  Plot: '',
  Area: '',
  Language: '',
  Year: '',
  Sort: '',
  current: 1
}

const { params, push } = useQuerySync<QueryShape>(initial, {
  path: '/filmClassifySearch',
  onChange: () => {
    void load()
  }
})

const resp = ref<ClassifySearchResp>({
  title: { id: 0, name: '' },
  list: [],
  page: { pageSize: 49, current: 1, pageCount: 0, total: 0 },
  search: { sortList: [], titles: {}, tags: {} },
  params: {
    Pid: '',
    Category: '',
    Plot: '',
    Area: '',
    Language: '',
    Year: '',
    Sort: ''
  }
})

const loading = ref<boolean>(true)
const loaded = ref<boolean>(false)
const errorMsg = ref<string>('')
const abortable = useAbortable()

async function load(): Promise<void> {
  const pid = (params.value.Pid || '').trim()
  if (!pid) {
    errorMsg.value = '缺少分类参数 Pid'
    loading.value = false
    loaded.value = true
    return
  }
  loading.value = true
  errorMsg.value = ''
  const signal = abortable.refresh()
  try {
    const r = await filmApi.searchClassify(
      {
        Pid: pid,
        Category: params.value.Category || '',
        Plot: params.value.Plot || '',
        Area: params.value.Area || '',
        Language: params.value.Language || '',
        Year: params.value.Year || '',
        Sort: params.value.Sort || '',
        current: params.value.current || 1
      },
      { signal }
    )
    resp.value = r
    loaded.value = true
  } catch (e) {
    if ((e as { name?: string })?.name === 'CanceledError') return
    errorMsg.value = '加载失败，请稍后重试'
    loaded.value = true
  } finally {
    loading.value = false
  }
}
// useQuerySync 的 onChange 会在路由 query 变化时触发 load；
// 进入页面（即使无 query 变化）也立即 load 一次保证首屏渲染。
void load()

const films = computed<FilmListItem[]>(() => resp.value.list || [])

/**
 * 把后端返回的 sortList / titles / tags 拼成 FilterGroup[]
 * 用 params 中当前值作为 current
 */
const filterGroups = computed<FilterGroup[]>(() => {
  const search = resp.value.search
  if (!search?.sortList?.length) return []
  return search.sortList
    .map<FilterGroup | null>((key) => {
      const tagList: ClassifyTagItem[] = search.tags?.[key] || []
      if (tagList.length === 0) return null
      return {
        key,
        title: search.titles?.[key] || key,
        options: tagList.map((t) => ({
          value: String(t.Value ?? ''),
          label: String(t.Name ?? '')
        })),
        current: String(params.value[key] ?? '')
      }
    })
    .filter((g): g is FilterGroup => g !== null)
})

function onFilterChange(payload: { key: string; value: string | number }): void {
  const allowed = ['Category', 'Plot', 'Area', 'Language', 'Year', 'Sort']
  if (!allowed.includes(payload.key)) return
  const next: Partial<QueryShape> = {
    [payload.key]: String(payload.value)
  } as Partial<QueryShape>
  // 切换筛选项 → current 重置 1
  next.current = 1
  void push(next)
}

function onPaginate(p: number): void {
  void push({ current: p })
  if (typeof window !== 'undefined') {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}
</script>

<template>
  <div class="gf-classify-search container-page py-[var(--gf-space-6)]">
    <!-- 顶部 title 切换：当前页选中"XX 库" -->
    <header
      v-if="resp.title.name"
      class="gf-classify__title flex items-center gap-[var(--gf-space-3)] mb-[var(--gf-space-6)]"
    >
      <RouterLink
        :to="{ path: '/filmClassify', query: { Pid: String(resp.title.id) } }"
        class="gf-classify__title-link"
        data-focusable="true"
        tabindex="0"
      >
        {{ resp.title.name }}
      </RouterLink>
      <span class="gf-classify__title-divider" aria-hidden="true">|</span>
      <RouterLink
        :to="{ path: '/filmClassifySearch', query: { Pid: String(resp.title.id) } }"
        class="gf-classify__title-active"
        data-focusable="true"
        tabindex="0"
      >
        {{ resp.title.name }}库
      </RouterLink>
    </header>

    <!-- 错误态 -->
    <BaseEmpty
      v-if="!loading && errorMsg"
      :title="errorMsg"
      description="请检查链接中的 Pid 参数"
    />

    <template v-else>
      <!-- 筛选栏 -->
      <FilmFilterBar
        v-if="filterGroups.length > 0"
        :groups="filterGroups"
        class="mb-[var(--gf-space-6)]"
        @change="onFilterChange"
      />

      <!-- 结果统计 -->
      <p
        v-if="resp.page.total > 0"
        class="text-secondary text-[var(--gf-fs-sm)] mb-[var(--gf-space-4)]"
      >
        共 <strong class="text-primary">{{ resp.page.total }}</strong> 部影片
      </p>

      <!-- 骨架 -->
      <div
        v-if="loading && films.length === 0"
        class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-[var(--gf-space-4)]"
      >
        <BaseSkeleton
          v-for="i in 12"
          :key="i"
          height="auto"
          ratio="2/3"
        />
      </div>

      <!-- 网格列表 -->
      <FilmGrid v-else-if="films.length > 0" :items="films" />

      <!-- 分页 -->
      <div
        v-if="!loading && resp.page.total > 0"
        class="mt-[var(--gf-space-8)] flex justify-center"
      >
        <BasePagination
          :current="resp.page.current || 1"
          :page-size="resp.page.pageSize || 49"
          :total="resp.page.total"
          @change="onPaginate"
        />
      </div>

      <!-- 空状态 -->
      <BaseEmpty
        v-if="!loading && loaded && films.length === 0 && !errorMsg"
        title="无符合条件的影片"
        description="试试切换其他筛选条件"
      />
    </template>
  </div>
</template>

<style scoped>
.gf-classify__title-active,
.gf-classify__title-link {
  text-decoration: none;
  font-size: var(--gf-fs-2xl);
  font-weight: var(--gf-fw-bold);
  outline: none;
  border-radius: var(--gf-radius-sm);
  transition: opacity var(--gf-dur-fast) var(--gf-ease-standard);
}

.gf-classify__title-active {
  background-image: var(--gf-brand-gradient);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  -webkit-text-fill-color: transparent;
}

.gf-classify__title-link {
  color: var(--gf-text-secondary);
}
.gf-classify__title-link:hover,
.gf-classify__title-link:focus-visible {
  color: var(--gf-text-primary);
}
.gf-classify__title-active:focus-visible,
.gf-classify__title-link:focus-visible {
  box-shadow: var(--gf-shadow-focus-ring);
}

.gf-classify__title-divider {
  color: var(--gf-text-muted);
  font-size: var(--gf-fs-lg);
}

@media (max-width: 767px) {
  .gf-classify__title-active,
  .gf-classify__title-link {
    font-size: var(--gf-fs-xl);
  }
}
</style>
