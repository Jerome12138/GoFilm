<script setup lang="ts">
import { useUIStore } from '@/stores/ui'
import { useSiteStore } from '@/stores/site'
import { storeToRefs } from 'pinia'
import BaseIcon from '@/components/base/BaseIcon.vue'

const emit = defineEmits<{
  (e: 'toggle-collapsed'): void
}>()

interface MenuItem {
  path: string
  label: string
  icon?: string
}

interface MenuGroup {
  title: string
  icon: string
  items: MenuItem[]
}

const uiStore = useUIStore()
const siteStore = useSiteStore()
const { sidebarCollapsed } = storeToRefs(uiStore)

function handleToggle(): void {
  uiStore.toggleSidebar()
  emit('toggle-collapsed')
}

const groups: MenuGroup[] = [
  { title: '概览', icon: 'home', items: [{ path: '/manage/index', label: '仪表盘' }] },
  {
    title: '影视',
    icon: 'film',
    items: [
      { path: '/manage/film', label: '影片列表' },
      { path: '/manage/film/class', label: '分类管理' },
      { path: '/manage/film/add', label: '新增影片' }
    ]
  },
  {
    title: '采集',
    icon: 'magic',
    items: [
      { path: '/manage/collect/index', label: '采集源' },
      { path: '/manage/cron/index', label: '定时任务' }
    ]
  },
  {
    title: '文件',
    icon: 'folder',
    items: [
      { path: '/manage/file/upload', label: '文件上传' },
      { path: '/manage/file/gallery', label: '文件库' }
    ]
  },
  {
    title: '系统',
    icon: 'settings',
    items: [{ path: '/manage/system/webSite', label: '站点配置' }]
  }
]
</script>

<template>
  <aside
    class="bg-[#191a23] border-r border-subtle h-full transition-[width] duration-[var(--gf-dur-base)] overflow-y-auto flex flex-col"
    :class="sidebarCollapsed ? 'w-[64px]' : 'w-[220px]'"
  >
    <button
      type="button"
      class="gf-sidebar__brand px-[var(--gf-space-4)] py-[var(--gf-space-5)] border-b border-subtle flex items-center gap-[var(--gf-space-3)] w-full bg-transparent border-0 cursor-pointer text-left"
      :title="sidebarCollapsed ? '展开侧栏' : '折叠侧栏'"
      :aria-label="sidebarCollapsed ? '展开侧栏' : '折叠侧栏'"
      @click="handleToggle"
    >
      <span
        class="font-[var(--gf-fw-bold)] italic text-brand-gradient text-[var(--gf-fs-lg)] truncate flex-1"
      >
        {{ sidebarCollapsed ? 'GF' : siteStore.basic?.siteName || 'GoFilm' }}
      </span>
      <BaseIcon
        v-if="!sidebarCollapsed"
        name="chevron-left"
        size="16px"
        class="text-muted shrink-0"
      />
    </button>
    <nav class="py-[var(--gf-space-3)] flex-1">
      <div
        v-for="group in groups"
        :key="group.title"
        class="mb-[var(--gf-space-3)]"
      >
        <div
          v-if="!sidebarCollapsed"
          class="px-[var(--gf-space-4)] py-[var(--gf-space-2)] text-xs text-muted uppercase tracking-wider flex items-center gap-[var(--gf-space-2)]"
        >
          <BaseIcon :name="group.icon" size="14px" />
          {{ group.title }}
        </div>
        <RouterLink
          v-for="item in group.items"
          :key="item.path"
          :to="item.path"
          class="flex items-center gap-[var(--gf-space-3)] px-[var(--gf-space-4)] py-[var(--gf-space-3)] text-secondary hover:bg-elevated hover:text-primary transition-colors"
          active-class="bg-[image:var(--gf-brand-gradient)] text-white shadow-purple-glow"
          data-focusable="true"
        >
          <BaseIcon
            v-if="sidebarCollapsed"
            :name="group.icon"
            size="20px"
          />
          <span :class="{ 'sr-only': sidebarCollapsed }">{{ item.label }}</span>
        </RouterLink>
      </div>
    </nav>
  </aside>
</template>
