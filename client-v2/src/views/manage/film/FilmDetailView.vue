<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { filmApi } from '@/api'
import type { FilmDetailResp } from '@/types/film'
import BaseSkeleton from '@/components/base/BaseSkeleton.vue'
import BaseImage from '@/components/base/BaseImage.vue'
import BaseTag from '@/components/base/BaseTag.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseEmpty from '@/components/base/BaseEmpty.vue'

const route = useRoute()
const router = useRouter()
const data = ref<FilmDetailResp | null>(null)
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  const id = String(route.query.id ?? '')
  if (!id) {
    error.value = '缺少影片 ID'
    loading.value = false
    return
  }
  try {
    data.value = await filmApi.getFilmDetail(id)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <section class="bg-surface rounded-card shadow-card p-[var(--gf-space-6)] max-w-[1100px]">
    <header class="flex items-center justify-between mb-[var(--gf-space-5)]">
      <h2 class="text-lg font-[var(--gf-fw-semibold)]">影片详情</h2>
      <BaseButton variant="ghost" size="sm" @click="router.back()">返回</BaseButton>
    </header>

    <div v-if="loading" class="flex flex-col gap-[var(--gf-space-3)]">
      <BaseSkeleton shape="rect" height="240px" />
      <BaseSkeleton v-for="i in 4" :key="i" shape="rect" height="20px" />
    </div>

    <BaseEmpty v-else-if="error || !data" :description="error || '未找到影片'" />

    <article v-else class="flex flex-col md:flex-row gap-[var(--gf-space-6)]">
      <div class="w-full md:w-[200px] shrink-0 rounded-[var(--gf-radius-lg)] overflow-hidden">
        <BaseImage :src="data.detail.picture" :alt="data.detail.name" ratio="3/4" />
      </div>
      <div class="flex-1 flex flex-col gap-[var(--gf-space-3)]">
        <h3 class="text-[var(--gf-fs-2xl)] font-[var(--gf-fw-bold)]">
          {{ data.detail.name }}
        </h3>
        <div class="flex flex-wrap gap-[var(--gf-space-2)]">
          <BaseTag v-if="data.detail.descriptor.cName" variant="purple">
            {{ data.detail.descriptor.cName }}
          </BaseTag>
          <BaseTag v-if="data.detail.descriptor.year">
            {{ data.detail.descriptor.year }}
          </BaseTag>
          <BaseTag v-if="data.detail.descriptor.area">
            {{ data.detail.descriptor.area }}
          </BaseTag>
        </div>
        <p class="text-sm text-secondary">
          <span class="text-muted">导演：</span>{{ data.detail.descriptor.director || '—' }}
        </p>
        <p class="text-sm text-secondary">
          <span class="text-muted">主演：</span>{{ data.detail.descriptor.actor || '—' }}
        </p>
        <p class="text-sm text-secondary">
          <span class="text-muted">上映：</span>{{ data.detail.descriptor.releaseDate || '—' }}
        </p>
        <p class="text-sm text-muted whitespace-pre-line max-h-[260px] overflow-auto">
          {{ data.detail.descriptor.content }}
        </p>
        <div class="text-xs text-muted">
          播放源：{{ data.detail.list?.length ?? 0 }} 个
        </div>
      </div>
    </article>
  </section>
</template>
