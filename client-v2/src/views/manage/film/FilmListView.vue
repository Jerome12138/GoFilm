<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { manageApi } from '@/api'
import type { FilmListItem } from '@/types/film'
import ManageTable from '@/components/manage/ManageTable.vue'
import ManageInput from '@/components/manage/ManageInput.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseImage from '@/components/base/BaseImage.vue'
import BasePagination from '@/components/base/BasePagination.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'

const router = useRouter()
const rows = ref<FilmListItem[]>([])
const loading = ref(true)
const total = ref(0)
const pageSize = ref(20)
const params = reactive({ keyword: '', current: 1, pageSize: 20 })

const columns = [
  { key: 'picture' as const, label: '海报', width: '90px' },
  { key: 'name' as const, label: '名称' },
  { key: 'cName' as const, label: '分类', width: '120px' },
  { key: 'year' as const, label: '年份', width: '80px' },
  { key: 'area' as const, label: '地区', width: '100px' }
]

async function load(): Promise<void> {
  loading.value = true
  try {
    const resp = await manageApi.film.searchList({ ...params })
    rows.value = resp.list ?? []
    total.value = resp.total ?? 0
    pageSize.value = resp.size ?? params.pageSize
  } finally {
    loading.value = false
  }
}

function search(): void {
  params.current = 1
  load()
}

function changePage(p: number): void {
  params.current = p
  load()
}

function viewDetail(row: FilmListItem): void {
  router.push({ path: '/manage/film/detail', query: { id: String(row.id) } })
}

onMounted(load)
</script>

<template>
  <div class="flex flex-col gap-[var(--gf-space-4)]">
    <ManageTable
      :columns="columns"
      :rows="rows"
      row-key="id"
      :loading="loading"
      empty="未找到影片"
    >
      <template #toolbar>
        <h2 class="text-lg font-[var(--gf-fw-semibold)]">影片管理</h2>
        <div class="flex gap-[var(--gf-space-2)] flex-wrap">
          <ManageInput v-model="params.keyword" placeholder="影片名关键字" @keydown.enter="search" />
          <BaseButton variant="gradient" size="sm" @click="search">
            <BaseIcon name="search" size="16px" /> 搜索
          </BaseButton>
          <BaseButton variant="ghost" size="sm" @click="router.push('/manage/film/add')">
            <BaseIcon name="plus" size="16px" /> 新增
          </BaseButton>
        </div>
      </template>

      <template #cell="{ row, col }">
        <div v-if="col.key === 'picture'" class="w-[60px] h-[80px] rounded-[var(--gf-radius-md)] overflow-hidden">
          <BaseImage :src="row.picture" :alt="row.name" ratio="3/4" />
        </div>
        <span v-else-if="col.key === 'name'" class="font-[var(--gf-fw-medium)]">
          {{ row.name }}
        </span>
        <span v-else>{{ row[col.key] ?? '—' }}</span>
      </template>

      <template #actions="{ row }">
        <div class="flex gap-[var(--gf-space-1)] justify-end">
          <BaseButton variant="ghost" size="sm" @click="viewDetail(row)">查看</BaseButton>
        </div>
      </template>
    </ManageTable>

    <BasePagination
      :current="params.current"
      :page-size="pageSize"
      :total="total"
      @change="changePage"
    />
  </div>
</template>
