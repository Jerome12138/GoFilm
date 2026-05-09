<script setup lang="ts">
import { ref } from 'vue'
import * as filmApi from '@/api/film'
import type { ClassifyData, FilmListItem } from '@/types/film'
import FilmRow from '@/components/film/FilmRow.vue'
import { useQuerySync } from '@/composables/useQuerySync'
import { useAbortable } from '@/composables/useAbortable'

/**
 * /filmClassify?Pid=xxx
 * STORY-012 分类首页：最新上映 / 排行榜 / 最近更新
 */

const { params } = useQuerySync<{ Pid: string }>(
  { Pid: '' },
  {
    path: '/filmClassify',
    onChange: () => {
      void load()
    }
  }
)

const data = ref<ClassifyData>({
  title: { id: 0, name: '' },
  content: { news: [], top: [], recent: [] }
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
  abortable.refresh()
  try {
    const resp = await filmApi.getClassify(pid)
    data.value = resp
    loaded.value = true
  } catch (e) {
    if ((e as { name?: string })?.name === 'CanceledError') return
    errorMsg.value = '加载失败，请稍后重试'
    loaded.value = true
  } finally {
    loading.value = false
  }
}

void load()

function moreLink(sort: string): {
  path: string
  query: Record<string, string>
} {
  return {
    path: '/filmClassifySearch',
    query: { Pid: String(data.value.title.id || params.value.Pid), Sort: sort }
  }
}

function isReady(items: FilmListItem[] | undefined): boolean {
  return Array.isArray(items) && items.length > 0
}
</script>

<template>
  <div class="gf-classify container-page py-[var(--gf-space-6)]">
    <!-- 顶部 title 切换 -->
    <header
      v-if="data.title.name"
      class="gf-classify__title flex items-center gap-[var(--gf-space-3)] mb-[var(--gf-space-8)]"
    >
      <RouterLink
        :to="{ path: '/filmClassify', query: { Pid: String(data.title.id) } }"
        class="gf-classify__title-active"
        data-focusable="true"
        tabindex="0"
      >
        {{ data.title.name }}
      </RouterLink>
      <span class="gf-classify__title-divider" aria-hidden="true">|</span>
      <RouterLink
        :to="{ path: '/filmClassifySearch', query: { Pid: String(data.title.id) } }"
        class="gf-classify__title-link"
        data-focusable="true"
        tabindex="0"
      >
        {{ data.title.name }}库
      </RouterLink>
    </header>

    <!-- 错误态 -->
    <BaseEmpty
      v-if="!loading && errorMsg"
      :title="errorMsg"
      description="请检查链接中的 Pid 参数"
    />

    <!-- 骨架 -->
    <div
      v-else-if="loading && !loaded"
      class="flex flex-col gap-[var(--gf-space-8)]"
    >
      <div v-for="n in 3" :key="n" class="flex flex-col gap-[var(--gf-space-3)]">
        <BaseSkeleton width="220px" height="32px" />
        <div class="flex gap-[var(--gf-space-3)] overflow-hidden">
          <BaseSkeleton
            v-for="i in 6"
            :key="i"
            width="160px"
            height="240px"
            ratio="2/3"
          />
        </div>
      </div>
    </div>

    <!-- 三个 Row -->
    <div
      v-else
      class="gf-classify__rows flex flex-col gap-[var(--gf-space-10)]"
    >
      <FilmRow
        v-if="isReady(data.content.news)"
        title="最新上映"
        :more-link="moreLink('release_stamp')"
        :items="data.content.news"
      />
      <FilmRow
        v-if="isReady(data.content.top)"
        title="排行榜"
        :more-link="moreLink('hits')"
        :items="data.content.top"
      />
      <FilmRow
        v-if="isReady(data.content.recent)"
        title="最近更新"
        :more-link="moreLink('update_stamp')"
        :items="data.content.recent"
      />

      <BaseEmpty
        v-if="
          !isReady(data.content.news) &&
          !isReady(data.content.top) &&
          !isReady(data.content.recent)
        "
        title="该分类暂无影片"
        description="切换到分类库浏览更多内容"
      >
        <template #action>
          <BaseButton
            v-if="data.title.id"
            variant="gradient"
            size="md"
            @click="
              $router.push({
                path: '/filmClassifySearch',
                query: { Pid: String(data.title.id) }
              })
            "
          >
            前往 {{ data.title.name }}库
          </BaseButton>
        </template>
      </BaseEmpty>
    </div>
  </div>
</template>

<style scoped>
.gf-classify__title {
  flex-wrap: wrap;
}
.gf-classify__title-active,
.gf-classify__title-link {
  text-decoration: none;
  font-size: var(--gf-fs-2xl);
  font-weight: var(--gf-fw-bold);
  outline: none;
  transition: opacity var(--gf-dur-fast) var(--gf-ease-standard);
  border-radius: var(--gf-radius-sm);
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
