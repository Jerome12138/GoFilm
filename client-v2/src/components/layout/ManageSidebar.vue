<script setup lang="ts">
import { useUIStore } from '@/stores'
import { storeToRefs } from 'pinia'

interface MenuItem {
  path: string
  label: string
}

interface MenuGroup {
  title: string
  items: MenuItem[]
}

const uiStore = useUIStore()
const { sidebarCollapsed } = storeToRefs(uiStore)

const groups: MenuGroup[] = [
  {
    title: '概览',
    items: [{ path: '/manage/index', label: '仪表盘' }]
  },
  {
    title: '影视',
    items: [
      { path: '/manage/film', label: '影片列表' },
      { path: '/manage/film/class', label: '分类管理' },
      { path: '/manage/film/add', label: '新增影片' }
    ]
  },
  {
    title: '采集',
    items: [
      { path: '/manage/collect/index', label: '采集源' },
      { path: '/manage/cron/index', label: '定时任务' }
    ]
  },
  {
    title: '文件',
    items: [
      { path: '/manage/file/upload', label: '上传' },
      { path: '/manage/file/gallery', label: '文件库' }
    ]
  },
  {
    title: '系统',
    items: [{ path: '/manage/system/webSite', label: '站点配置' }]
  }
]
</script>

<template>
  <aside
    class="bg-surface border-r border-subtle h-full transition-[width] duration-[var(--gf-dur-base)] overflow-y-auto"
    :class="sidebarCollapsed ? 'w-[60px]' : 'w-[220px]'"
  >
    <nav class="py-[var(--gf-space-4)]">
      <div
        v-for="group in groups"
        :key="group.title"
        class="mb-[var(--gf-space-4)]"
      >
        <div
          v-if="!sidebarCollapsed"
          class="px-[var(--gf-space-4)] py-[var(--gf-space-2)] text-xs text-muted uppercase tracking-wider"
        >
          {{ group.title }}
        </div>
        <RouterLink
          v-for="item in group.items"
          :key="item.path"
          :to="item.path"
          class="block px-[var(--gf-space-4)] py-[var(--gf-space-3)] text-secondary hover:bg-elevated hover:text-primary transition-colors"
          active-class="bg-elevated text-primary border-l-2 border-brand"
        >
          {{ sidebarCollapsed ? item.label.slice(0, 1) : item.label }}
        </RouterLink>
      </div>
    </nav>
  </aside>
</template>
