<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { manageApi } from '@/api'
import type { FileItem } from '@/types/manage'
import BaseImage from '@/components/base/BaseImage.vue'
import BaseEmpty from '@/components/base/BaseEmpty.vue'
import BaseSkeleton from '@/components/base/BaseSkeleton.vue'
import BasePagination from '@/components/base/BasePagination.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'

const items = ref<FileItem[]>([])
const loading = ref(true)
const total = ref(0)
const pageSize = ref(20)
const params = reactive({ current: 1, size: 20 })

async function load(): Promise<void> {
  loading.value = true
  try {
    const resp = await manageApi.file.list({ ...params })
    items.value = resp.list ?? []
    total.value = resp.total ?? 0
    pageSize.value = resp.size ?? params.size
  } finally {
    loading.value = false
  }
}

async function remove(item: FileItem): Promise<void> {
  if (!confirm(`确认删除「${item.name}」？`)) return
  await manageApi.file.remove(item.id)
  await load()
}

function changePage(p: number): void {
  params.current = p
  load()
}

onMounted(load)
</script>

<template>
  <section class="flex flex-col gap-[var(--gf-space-4)]">
    <header class="flex items-center justify-between">
      <div>
        <h2 class="text-lg font-[var(--gf-fw-semibold)]">文件库</h2>
        <p class="text-sm text-muted">站点已上传的图片 / 视频资源</p>
      </div>
      <BaseButton variant="ghost" size="sm" @click="load">
        <BaseIcon name="refresh" size="16px" /> 刷新
      </BaseButton>
    </header>

    <div
      v-if="loading"
      class="grid grid-cols-2 md:grid-cols-4 xl:grid-cols-6 gap-[var(--gf-space-3)]"
    >
      <BaseSkeleton v-for="i in 8" :key="i" shape="rect" height="160px" />
    </div>

    <BaseEmpty v-else-if="!items.length" description="尚无文件" />

    <div
      v-else
      class="grid grid-cols-2 md:grid-cols-4 xl:grid-cols-6 gap-[var(--gf-space-3)]"
    >
      <article
        v-for="item in items"
        :key="item.id"
        class="bg-surface rounded-[var(--gf-radius-md)] overflow-hidden flex flex-col group"
      >
        <div class="aspect-square bg-elevated">
          <BaseImage :src="item.url" :alt="item.name" />
        </div>
        <div class="p-[var(--gf-space-2)] text-xs flex flex-col gap-[var(--gf-space-1)]">
          <div class="truncate text-secondary">{{ item.name }}</div>
          <div class="flex items-center justify-between text-muted">
            <span>{{ (item.size / 1024).toFixed(1) }} KB</span>
            <button
              class="hover:text-[var(--gf-danger)]"
              type="button"
              data-focusable="true"
              @click="remove(item)"
            >
              删除
            </button>
          </div>
        </div>
      </article>
    </div>

    <BasePagination
      :current="params.current"
      :page-size="pageSize"
      :total="total"
      @change="changePage"
    />
  </section>
</template>
