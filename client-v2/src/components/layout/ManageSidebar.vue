<script setup lang="ts">
import { computed } from 'vue'
import { useUIStore } from '@/stores/ui'
import { useSiteStore } from '@/stores/site'
import { storeToRefs } from 'pinia'
import BaseIcon from '@/components/base/BaseIcon.vue'

const props = withDefaults(defineProps<{
  variant?: 'drawer' | 'icon-rail' | 'full'
  open?: boolean
}>(), {
  variant: 'full',
  open: false
})

const emit = defineEmits<{
  (e: 'toggle-collapsed'): void
  (e: 'close'): void
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

// tablet=icon-rail 强制 collapsed; drawer 总是展开; 其余按用户偏好
const effectiveCollapsed = computed(() => {
  if (props.variant === 'icon-rail') return true
  if (props.variant === 'drawer') return false
  return sidebarCollapsed.value
})

function handleBrandClick(): void {
  if (props.variant === 'drawer') {
    emit('close')
  } else if (props.variant === 'full') {
    uiStore.toggleSidebar()
    emit('toggle-collapsed')
  }
  // icon-rail 模式 brand 点击无操作 (强制 collapsed)
}

function handleItemClick(): void {
  if (props.variant === 'drawer') {
    emit('close')
  }
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
  <!-- drawer 遮罩 (仅 mobile 模式 + open 时) -->
  <div
    v-if="props.variant === 'drawer' && props.open"
    class="fixed inset-0 bg-black/60 z-[90]"
    @click="emit('close')"
  />
  <aside
    class="bg-[#191a23] border-r border-subtle overflow-y-auto flex flex-col transition-[transform,width] duration-[var(--gf-dur-base)]"
    :class="[
      props.variant === 'drawer'
        ? 'fixed inset-y-0 left-0 z-[100] w-[75%] max-w-[280px] h-screen'
        : ['h-full', effectiveCollapsed ? 'w-[64px]' : 'w-[220px]'],
      props.variant === 'drawer' && !props.open ? '-translate-x-full' : 'translate-x-0'
    ]"
  >
    <button
      type="button"
      class="gf-sidebar__brand px-[var(--gf-space-4)] py-[var(--gf-space-5)] border-b border-subtle flex items-center gap-[var(--gf-space-3)] w-full bg-transparent border-0 cursor-pointer text-left min-h-[44px]"
      :title="props.variant === 'drawer' ? '关闭菜单' : effectiveCollapsed ? '展开侧栏' : '折叠侧栏'"
      :aria-label="props.variant === 'drawer' ? '关闭菜单' : effectiveCollapsed ? '展开侧栏' : '折叠侧栏'"
      @click="handleBrandClick"
    >
      <span
        class="font-[var(--gf-fw-bold)] italic text-brand-gradient text-[var(--gf-fs-lg)] truncate flex-1"
      >
        {{ effectiveCollapsed ? 'GF' : siteStore.basic?.siteName || 'GoFilm' }}
      </span>
      <BaseIcon
        v-if="!effectiveCollapsed"
        :name="props.variant === 'drawer' ? 'close' : 'chevron-left'"
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
          v-if="!effectiveCollapsed"
          class="px-[var(--gf-space-4)] py-[var(--gf-space-2)] text-xs text-muted uppercase tracking-wider flex items-center gap-[var(--gf-space-2)]"
        >
          <BaseIcon :name="group.icon" size="14px" />
          {{ group.title }}
        </div>
        <RouterLink
          v-for="item in group.items"
          :key="item.path"
          :to="item.path"
          class="flex items-center gap-[var(--gf-space-3)] px-[var(--gf-space-4)] py-[var(--gf-space-3)] text-secondary hover:bg-elevated hover:text-primary transition-colors min-h-[44px]"
          active-class="bg-[image:var(--gf-brand-gradient)] text-white shadow-purple-glow"
          data-focusable="true"
          @click="handleItemClick"
        >
          <BaseIcon
            v-if="effectiveCollapsed"
            :name="group.icon"
            size="20px"
          />
          <span :class="{ 'sr-only': effectiveCollapsed }">{{ item.label }}</span>
        </RouterLink>
      </div>
    </nav>
  </aside>
</template>
