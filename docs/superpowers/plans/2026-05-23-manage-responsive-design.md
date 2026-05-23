# Manage 后台响应式 — 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 GoFilm /manage/* 后台在 mobile / tablet / desktop 三档下都可用且触摸友好。

**Architecture:** 混合策略 — useViewMode 提供 mode hook (JS 分支), UnoCSS `md:`/`lg:` 断点处理样式排版; 既有 uiStore.sidebarCollapsed 复用为桌面 toggle, tablet 强制 icon-rail。新建 ManageSheet 封装弹窗三态 (modal/sheet/fullsheet)。ManageTable 加 mobileVariant=card 默认 + 智能 fallback。

**Tech Stack:** Vue 3.5 / TypeScript / UnoCSS / Pinia / Vitest / @vue/test-utils / Playwright

**Spec:** `docs/superpowers/specs/2026-05-23-manage-responsive-design.md` (commit b417d02)

**部署/工作环境:**
- 所有编辑和 commit 都在远程服务器: `ubuntu@43.156.77.237:~/gofilm/`
- git author: `jerome12138 <1030155707@qq.com>` (已 repo-scoped 配置)
- 构建+热部署: `cd ~/gofilm/film && sudo docker compose up -d --build nginx`
- 单测: `cd ~/gofilm/client-v2 && pnpm install && pnpm test`
- 每个 task 的 ssh 命令前缀: `ssh -i ~/.ssh/id_ed25519 ubuntu@43.156.77.237` (省略, 实际跑加上)

---

## 文件结构地图

| 文件 | 责任 | 阶段 |
|---|---|---|
| `client-v2/src/composables/useViewMode.ts` | 扩展四态 ViewMode + isTablet/isNarrow + 三档 detectMode/onResize | P0 |
| `client-v2/src/components/layout/ManageLayout.vue` | 去 data-mode="desktop", 接入 mode, drawerOpen 状态, main 区 padding 三档 | P0 |
| `client-v2/src/components/layout/ManageHeader.vue` | 汉堡按钮按 mode 分支: mobile emit toggle-drawer, desktop 仍 toggleSidebar | P0 |
| `client-v2/src/components/layout/ManageSidebar.vue` | 加 variant: drawer/icon-rail/full prop, drawer 模式覆盖层 | P1 |
| `client-v2/src/components/manage/ManageSheet.vue` | **新建** modal/sheet/fullsheet 三态封装 | P2 |
| `client-v2/src/components/manage/ManageTable.vue` | 加 mobileVariant: card prop, card 模式 div 渲染 | P3 |
| `client-v2/src/views/manage/collect/CollectListView.vue` | BaseDialog → ManageSheet | P2 |
| `client-v2/src/views/manage/cron/CronListView.vue` | BaseDialog → ManageSheet | P2 |
| `client-v2/src/views/manage/film/FilmClassView.vue` | BaseDialog → ManageSheet | P2 |
| `client-v2/src/components/layout/ManageHeader.vue` (二次) | BaseDialog 改密 → ManageSheet | P2 |
| `client-v2/tests/composables/useViewMode.spec.ts` | **新建** 单测 useViewMode 三档逻辑 | P0 |
| `client-v2/tests/components/ManageSheet.spec.ts` | **新建** 单测 ManageSheet 形态切换 | P2 |
| `client-v2/tests/components/ManageTable.spec.ts` | **新建** 单测 card 模式渲染 | P3 |
| `client-v2/tests/e2e/manage-responsive.spec.ts` | **新建** Playwright 3 视口截屏 | P5 |
| `client-v2/uno.config.ts` | (确认) breakpoints 已含 md/lg, 无需改 | P0 |

---
# P0 / 基建 — useViewMode 四态 + Layout 接入

## Task 1: 扩展 useViewMode 为四态 + 加 isTablet/isNarrow

**Files:**
- Modify: `client-v2/src/composables/useViewMode.ts`
- Test: `client-v2/tests/composables/useViewMode.spec.ts` (新建)

- [ ] **Step 1: 写失败测试** — 文件 `client-v2/tests/composables/useViewMode.spec.ts`

```typescript
import { describe, it, expect, beforeEach, vi } from "vitest"

describe("useViewMode 四档检测", () => {
  beforeEach(() => {
    localStorage.removeItem("gf-mode")
    document.documentElement.removeAttribute("data-mode")
    vi.resetModules()
  })

  it("视口 600px → mobile", async () => {
    Object.defineProperty(window, "innerWidth", { value: 600, configurable: true })
    const m = await import("@/composables/useViewMode")
    m.installViewMode()
    const v = m.useViewMode()
    expect(v.mode.value).toBe("mobile")
    expect(v.isMobile.value).toBe(true)
    expect(v.isTablet.value).toBe(false)
    expect(v.isNarrow.value).toBe(true)
  })

  it("视口 900px → tablet", async () => {
    Object.defineProperty(window, "innerWidth", { value: 900, configurable: true })
    const m = await import("@/composables/useViewMode")
    m.installViewMode()
    const v = m.useViewMode()
    expect(v.mode.value).toBe("tablet")
    expect(v.isTablet.value).toBe(true)
    expect(v.isNarrow.value).toBe(true)
  })

  it("视口 1280px → desktop", async () => {
    Object.defineProperty(window, "innerWidth", { value: 1280, configurable: true })
    const m = await import("@/composables/useViewMode")
    m.installViewMode()
    const v = m.useViewMode()
    expect(v.mode.value).toBe("desktop")
    expect(v.isNarrow.value).toBe(false)
  })
})
```

- [ ] **Step 2: 跑测试确认 fail** — `cd ~/gofilm/client-v2 && pnpm test useViewMode -- --run`. Expected: FAIL (isTablet not exported, etc.)

- [ ] **Step 3: 实现** — 修改 `client-v2/src/composables/useViewMode.ts`

  1. type 扩展:
  ```typescript
  export type ViewMode = "mobile" | "tablet" | "desktop" | "tv"
  export type PersistedMode = "mobile" | "desktop" | "tv"
  ```

  2. `detectMode` 末尾改三档:
  ```typescript
  const w = window.innerWidth
  if (w < 768) return "mobile"
  if (w < 1024) return "tablet"
  return "desktop"
  ```

  3. `onResize` 内 next 改三档:
  ```typescript
  const next: ViewMode = detectTV() ? "tv" : w < 768 ? "mobile" : w < 1024 ? "tablet" : "desktop"
  ```

  4. `setMode` 入参类型收紧 `PersistedMode | null`

  5. 返回值加 isTablet / isNarrow + 类型签名:
  ```typescript
  isTablet: computed(() => mode.value === "tablet"),
  isNarrow: computed(() => mode.value === "mobile" || mode.value === "tablet")
  ```

- [ ] **Step 4: 跑测试确认 pass** — Expected: 3/3 PASS

- [ ] **Step 5: 提交**

```bash
cd ~/gofilm && git -c safe.directory=$PWD add \
  client-v2/src/composables/useViewMode.ts \
  client-v2/tests/composables/useViewMode.spec.ts && \
git -c safe.directory=$PWD commit -m "feat(client-v2): useViewMode 扩展四态 (P0)"
```

---

## Task 2: ManageLayout 接入 mode + drawerOpen + 三档 padding

**Files:** Modify `client-v2/src/components/layout/ManageLayout.vue`

- [ ] **Step 1: script setup 改**

```typescript
import { onMounted, ref } from "vue"
import { useViewMode } from "@/composables/useViewMode"
// ...
const { mode, isMobile, isTablet } = useViewMode()
const drawerOpen = ref(false)
function closeDrawer(): void { drawerOpen.value = false }
```

- [ ] **Step 2: template 改 (去硬编码 + 三档 padding + 传 variant)**

```vue
<template>
  <div class="min-h-screen flex flex-col bg-base text-primary" :data-mode="mode">
    <ManageHeader :show-hamburger="isMobile" @toggle-drawer="drawerOpen = !drawerOpen" />
    <div class="flex-1 flex overflow-hidden relative">
      <ManageSidebar
        :variant="isMobile ? \"drawer\" : isTablet ? \"icon-rail\" : \"full\""
        :open="drawerOpen"
        @close="closeDrawer"
      />
      <main class="flex-1 overflow-x-auto overflow-y-auto p-[var(--gf-space-3)] md:p-[var(--gf-space-4)] lg:p-[var(--gf-space-6)]">
        <slot />
      </main>
    </div>
  </div>
</template>
```

- [ ] **Step 3: 提交** — `git commit -m "feat(client-v2): ManageLayout 去硬编码 data-mode, 接入 useViewMode (P0)"`

---

## Task 3: ManageHeader 汉堡按钮按 mode 分支

**Files:** Modify `client-v2/src/components/layout/ManageHeader.vue`

- [ ] **Step 1: 加 props + emits**

```typescript
const props = defineProps<{ showHamburger?: boolean }>()
const emit = defineEmits<{ (e: "toggle-drawer"): void }>()
```

- [ ] **Step 2: 改 menu 按钮**

```vue
<button class="text-white/90 hover:text-white text-xl min-h-[44px] min-w-[44px] flex items-center justify-center"
  @click="props.showHamburger ? emit(\"toggle-drawer\") : uiStore.toggleSidebar()">
  <BaseIcon name="menu" size="22px" />
  <span class="sr-only">{{ props.showHamburger ? "打开菜单" : "切换侧栏" }}</span>
</button>
```

- [ ] **Step 3: 跑现有测试无回归** — `pnpm test -- --run`

- [ ] **Step 4: 提交** — `git commit -m "feat(client-v2): ManageHeader 汉堡按钮按 mode 分支 + 44pt 触摸目标 (P0)"`

---
# P1 / ManageSidebar 三变体

## Task 4: ManageSidebar 加 variant + drawer 模式

**Files:**
- Modify: `client-v2/src/components/layout/ManageSidebar.vue`
- Test: `client-v2/tests/components/ManageSidebar.spec.ts` (新建)

- [ ] **Step 1: script setup 加 props + effectiveCollapsed**

```typescript
import { computed } from "vue"

const props = defineProps<{
  variant?: "drawer" | "icon-rail" | "full"
  open?: boolean
}>()
const emit = defineEmits<{
  (e: "toggle-collapsed"): void
  (e: "close"): void
}>()

const effectiveCollapsed = computed(() => {
  if (props.variant === "icon-rail") return true
  if (props.variant === "drawer") return false
  return sidebarCollapsed.value
})
```

- [ ] **Step 2: template 改根元素 + 加遮罩**

```vue
<template>
  <div v-if="props.variant === \"drawer\" && props.open"
    class="fixed inset-0 bg-black/60 z-[var(--gf-z-overlay,90)]"
    @click="emit(\"close\")" />
  <aside
    class="bg-[#191a23] border-r border-subtle h-full transition-[transform,width] duration-[var(--gf-dur-base)] overflow-y-auto flex flex-col"
    :class="[
      props.variant === \"drawer\"
        ? \"fixed inset-y-0 left-0 z-[var(--gf-z-drawer,100)] w-[75%] max-w-[280px]\"
        : effectiveCollapsed ? \"w-[64px]\" : \"w-[220px]\",
      props.variant === \"drawer\" && !props.open ? \"-translate-x-full\" : \"translate-x-0\"
    ]">
    <!-- (现有 brand button 和 nav, 把 sidebarCollapsed 替换为 effectiveCollapsed) -->
    <!-- 导航 item 加 @click="props.variant === \"drawer\" && emit(\"close\")" -->
  </aside>
</template>
```

- [ ] **Step 3: 写单测**

```typescript
import { describe, it, expect, beforeEach } from "vitest"
import { mount } from "@vue/test-utils"
import { createPinia, setActivePinia } from "pinia"
import ManageSidebar from "@/components/layout/ManageSidebar.vue"

describe("ManageSidebar variant", () => {
  beforeEach(() => setActivePinia(createPinia()))

  it("drawer + open=true → 遮罩存在", () => {
    const w = mount(ManageSidebar, { props: { variant: "drawer", open: true } })
    expect(w.find(".fixed.inset-0").exists()).toBe(true)
  })

  it("drawer + open=false → -translate-x-full", () => {
    const w = mount(ManageSidebar, { props: { variant: "drawer", open: false } })
    expect(w.find("aside").classes()).toContain("-translate-x-full")
  })

  it("icon-rail → w-[64px]", () => {
    const w = mount(ManageSidebar, { props: { variant: "icon-rail" } })
    expect(w.find("aside").classes()).toContain("w-[64px]")
  })

  it("full → w-[220px]", () => {
    const w = mount(ManageSidebar, { props: { variant: "full" } })
    expect(w.find("aside").classes()).toContain("w-[220px]")
  })
})
```

- [ ] **Step 4: 跑测试 + 部署 + 浏览器 smoke**

```bash
cd ~/gofilm/client-v2 && pnpm test ManageSidebar -- --run  # 4/4 PASS
cd ~/gofilm/film && sudo docker compose up -d --build nginx
```

浏览器 http://43.156.77.237/manage/index, devtools 三视口验证:
- 375px: 汉堡 → drawer 滑出 → 点遮罩关闭
- 900px: 60px icon-rail 常驻
- 1280px: 220px 完整菜单 (或用户已折叠的 64px)

- [ ] **Step 5: 提交** — `git commit -m "feat(client-v2): ManageSidebar 三变体 (P1)"`

---
# P2 / ManageSheet + 4 个 BaseDialog 迁移

## Task 5: 新建 ManageSheet 组件

**Files:**
- Create: `client-v2/src/components/manage/ManageSheet.vue`
- Test: `client-v2/tests/components/ManageSheet.spec.ts`

- [ ] **Step 1: 写失败测试**

```typescript
import { describe, it, expect, beforeEach } from "vitest"
import { mount } from "@vue/test-utils"
import { createPinia, setActivePinia } from "pinia"
import ManageSheet from "@/components/manage/ManageSheet.vue"

describe("ManageSheet 形态", () => {
  beforeEach(() => setActivePinia(createPinia()))

  it("desktop → .gf-sheet--modal", async () => {
    Object.defineProperty(window, "innerWidth", { value: 1280, configurable: true })
    const w = mount(ManageSheet, { props: { modelValue: true, title: "T" } })
    await w.vm.$nextTick()
    expect(w.find(".gf-sheet--modal").exists()).toBe(true)
  })

  it("mobile + sheet → .gf-sheet--sheet", async () => {
    Object.defineProperty(window, "innerWidth", { value: 375, configurable: true })
    const w = mount(ManageSheet, { props: { modelValue: true, mobileMode: "sheet" } })
    await w.vm.$nextTick()
    expect(w.find(".gf-sheet--sheet").exists()).toBe(true)
  })

  it("mobile + fullsheet → .gf-sheet--fullsheet", async () => {
    Object.defineProperty(window, "innerWidth", { value: 375, configurable: true })
    const w = mount(ManageSheet, { props: { modelValue: true, mobileMode: "fullsheet" } })
    await w.vm.$nextTick()
    expect(w.find(".gf-sheet--fullsheet").exists()).toBe(true)
  })

  it("点遮罩 emit update:modelValue false + close", async () => {
    Object.defineProperty(window, "innerWidth", { value: 1280, configurable: true })
    const w = mount(ManageSheet, { props: { modelValue: true } })
    await w.vm.$nextTick()
    await w.find(".gf-sheet__overlay").trigger("click")
    expect(w.emitted("update:modelValue")?.[0]).toEqual([false])
    expect(w.emitted("close")).toBeTruthy()
  })
})
```

- [ ] **Step 2: 实现 ManageSheet.vue** (按 spec 三态规则)

完整代码见 spec 中 ManageSheet 部分 + 上面单测期望的 CSS class 名 (`gf-sheet__overlay`, `gf-sheet--modal`, `gf-sheet--sheet`, `gf-sheet--fullsheet`)。
关键点:
  - `<Teleport to="body">` 避免 z-index 嵌套问题
  - `variant` 基于 `isMobile` 和 `mobileMode` 算出
  - sheet 模式: `bottom-0 max-h-[75vh]` + grabber
  - fullsheet 模式: `absolute inset-0` + 顶部 header (← 返回 / title / slot header-action)
  - footer slot 在三种 variant 下都支持

- [ ] **Step 3: 跑测试 4/4 PASS**

- [ ] **Step 4: 提交** — `git commit -m "feat(client-v2): 新建 ManageSheet 三态弹窗 (P2)"`

---

## Task 6: 迁移 FilmClassView 试点

**Files:** Modify `client-v2/src/views/manage/film/FilmClassView.vue`

- [ ] **Step 1: 读现有用法** — `grep -n -B 2 -A 5 BaseDialog ~/gofilm/client-v2/src/views/manage/film/FilmClassView.vue`

- [ ] **Step 2: 替换 import + 标签**

```diff
- import BaseDialog from "@/components/base/BaseDialog.vue"
+ import ManageSheet from "@/components/manage/ManageSheet.vue"
- <BaseDialog v-model:visible="dialogOpen" title="编辑分类">
+ <ManageSheet v-model="dialogOpen" title="编辑分类" mobile-mode="sheet">
```

注意 prop 差异: `v-model:visible` → `v-model`。footer slot 命名相同。

- [ ] **Step 3: 部署 + 浏览器手动 smoke** — desktop modal + mobile sheet 两态

- [ ] **Step 4: 提交** — `git commit -m "feat(client-v2): FilmClassView 改用 ManageSheet (P2 试点)"`

---

## Task 7: 批量迁移剩 3 个 (CollectListView + CronListView + ManageHeader)

> 试点跑通后, 批量迁移。每个调用方同 Task 6 模式。

- [ ] **Step 1: CollectListView.vue 迁移** — 看字段数决定 mobileMode (默认 sheet, >5 字段改 fullsheet)
- [ ] **Step 2: CronListView.vue 迁移** — 同上
- [ ] **Step 3: ManageHeader.vue 改密弹窗迁移** — 3 字段, mobileMode=sheet。注意是 layout 组件, 影响所有 manage 页
- [ ] **Step 4: 浏览器统一 smoke** — 4 弹窗 × 2 视口 = 8 测试点
- [ ] **Step 5: 提交** — `git commit -m "feat(client-v2): 迁移剩余 3 个 BaseDialog → ManageSheet (P2 完成)"`

---
# P3 / ManageTable mobileVariant=card 默认

## Task 8: ManageTable 加 mobileVariant prop + card 渲染

**Files:**
- Modify: `client-v2/src/components/manage/ManageTable.vue`
- Test: `client-v2/tests/components/ManageTable.spec.ts`

- [ ] **Step 1: 写失败测试**

```typescript
import { describe, it, expect, beforeEach } from "vitest"
import { mount } from "@vue/test-utils"
import { createPinia, setActivePinia } from "pinia"
import ManageTable from "@/components/manage/ManageTable.vue"

interface Row { id: number; name: string; email: string }
const cols = [{ key: "name" as const, label: "用户名" }, { key: "email" as const, label: "邮箱" }]
const rows: Row[] = [{ id: 1, name: "admin", email: "a@a.com" }, { id: 2, name: "jerry", email: "j@j.com" }]

describe("ManageTable card mode", () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    Object.defineProperty(window, "innerWidth", { value: 375, configurable: true })
  })

  it("mobile + 默认 mobileVariant=card → .gf-card-list", async () => {
    const w = mount(ManageTable<Row>, { props: { columns: cols, rows, rowKey: "id" } })
    await w.vm.$nextTick()
    expect(w.find(".gf-card-list").exists()).toBe(true)
    expect(w.find("table").exists()).toBe(false)
  })

  it("desktop → 仍渲染 <table>", async () => {
    Object.defineProperty(window, "innerWidth", { value: 1280, configurable: true })
    const w = mount(ManageTable<Row>, { props: { columns: cols, rows, rowKey: "id" } })
    await w.vm.$nextTick()
    expect(w.find("table").exists()).toBe(true)
  })

  it("#mobile-card slot 完全覆盖", async () => {
    const w = mount(ManageTable<Row>, {
      props: { columns: cols, rows, rowKey: "id" },
      slots: { "mobile-card": "<div class=\"custom-card\">{{ params.row.name }}</div>" }
    })
    await w.vm.$nextTick()
    expect(w.findAll(".custom-card")).toHaveLength(2)
  })
})
```

- [ ] **Step 2: 改 ManageTable.vue**

script setup 加:
```typescript
import { computed } from "vue"
import { useViewMode } from "@/composables/useViewMode"

const props = withDefaults(defineProps<ManageTableProps<T> & {
  mobileVariant?: "card" | "scroll"
}>(), { mobileVariant: "card" })

const { isMobile } = useViewMode()
const useCardMode = computed(() => isMobile.value && props.mobileVariant === "card")
```

defineSlots 增加:
```typescript
defineSlots<{
  cell(props: { row: T; col: Column<T>; value: unknown }): unknown
  actions(props: { row: T }): unknown
  toolbar(): unknown
  "mobile-card"(props: { row: T; index: number }): unknown
}>()
```

template 在 v-else 分支前插入 card 模式:
```vue
<div v-else-if="useCardMode" class="gf-card-list flex flex-col gap-[var(--gf-space-3)] p-[var(--gf-space-3)]">
  <article v-for="(row, idx) in props.rows" :key="String(row[props.rowKey])"
    class="bg-elevated rounded-[var(--gf-radius-md)] border border-default p-[var(--gf-space-4)] min-h-[44px]">
    <slot name="mobile-card" :row="row" :index="idx">
      <header class="font-[var(--gf-fw-semibold)] text-primary mb-[var(--gf-space-2)]">
        <slot name="cell" :row="row" :col="props.columns[0]" :value="row[props.columns[0].key]">
          {{ row[props.columns[0].key] ?? "—" }}
        </slot>
      </header>
      <dl class="text-sm text-secondary space-y-[var(--gf-space-1)]">
        <div v-for="col in props.columns.slice(1)" :key="col.key" class="flex gap-[var(--gf-space-2)]">
          <dt class="text-muted shrink-0">{{ col.label }}:</dt>
          <dd>
            <slot name="cell" :row="row" :col="col" :value="row[col.key]">
              {{ row[col.key] ?? "—" }}
            </slot>
          </dd>
        </div>
      </dl>
      <div v-if="$slots.actions" class="mt-[var(--gf-space-3)] pt-[var(--gf-space-3)] border-t border-subtle flex gap-[var(--gf-space-2)] justify-end">
        <slot name="actions" :row="row" />
      </div>
    </slot>
  </article>
</div>
```

注意 `v-else` (table 模式) 保持原样, 加 `v-else-if="useCardMode"` 在它前面。

- [ ] **Step 3: 跑测试 3/3 PASS**

- [ ] **Step 4: 部署 + 全 manage 区表格视觉检查** — devtools 375px 过一遍 8 个表格页, 标 fallback 不好看的页面 (留 Task 9 覆盖)

- [ ] **Step 5: 提交** — `git commit -m "feat(client-v2): ManageTable 加 mobileVariant=card 默认 (P3)"`

---

## Task 9: 针对密度低/字段多的页面 覆盖 #mobile-card slot (按需)

**候选** (实际按 Task 8 视觉决定):
- FilmListView: 字段多, 可能 fallback 太长 → 自定义紧凑卡 (海报 + 标题 + 分类 + 状态)
- 其他: 优先 fallback, 视觉不通过再加

- [ ] **Step 1: 列出需要覆盖的页面 (本 plan inline 写)**
- [ ] **Step 2: 为每个页面写 #mobile-card slot** — 见 spec 中 FilmListView 示例
- [ ] **Step 3: 单页 smoke + 单次提交** — `git commit -m "feat(client-v2): <PageName> 自定义 mobile-card slot (P3 优化)"`

---
# P4 / 触摸目标 + 像素零散

## Task 10: 全 manage 区按钮 isNarrow 触发 44pt min-height

**Files:** BaseButton.vue + 全 manage 按钮调用方

- [ ] **Step 1: 评估范围** — `grep -rn BaseButton ~/gofilm/client-v2/src/views/manage ~/gofilm/client-v2/src/components/manage | wc -l`

- [ ] **Step 2: 两种方案二选一**

**方案 A (推荐): 加 CSS 全局规则**

修改 `client-v2/src/assets/styles/theme.css` 或新建 `manage-mobile.css`, 加:
```css
[data-mode="mobile"] .gf-manage button,
[data-mode="mobile"] .gf-manage a[role="button"] { min-height: 44px; }
```

ManageLayout 主区 div 加 `gf-manage` class:
```vue
<main class="gf-manage flex-1 ..."> <slot /></main>
```

**方案 B: BaseButton 加 size=touch prop**

适合更精细控制, 但要修每个调用方。

- [ ] **Step 3: 实施 + 视觉 smoke + 提交** — `git commit -m "feat(client-v2): manage 区按钮 mobile 满足 WCAG 44pt 触摸目标 (P4)"`

---

## Task 11: 表单 input / select 在 mobile 全宽 + 44pt 高度

**Files:** ManageInput.vue / ManageTextarea.vue / ManageSwitch.vue

- [ ] **Step 1: ManageInput class 加 mobile 全宽 + min-h**

```vue
class="w-full min-h-[44px] md:min-h-[36px] ..."
```

- [ ] **Step 2: 同步 ManageTextarea / Switch** — Switch 触摸区域至少 44x44

- [ ] **Step 3: 视觉 smoke + 提交** — `git commit -m "feat(client-v2): 表单组件 mobile 全宽 + 44pt 高 (P4)"`

---

# P5 / Playwright 三视口 smoke 测试

## Task 12: Playwright 配三视口 project

**Files:** Modify `client-v2/playwright.config.ts`

- [ ] **Step 1: 检查现有 playwright.config** — `cat ~/gofilm/client-v2/playwright.config.ts`

- [ ] **Step 2: 加 3 project**

```typescript
import { defineConfig, devices } from "@playwright/test"

export default defineConfig({
  testDir: "./tests/e2e",
  use: { baseURL: process.env.E2E_BASE_URL ?? "http://43.156.77.237" },
  projects: [
    { name: "mobile", use: { ...devices["iPhone 12"] } },
    { name: "tablet", use: { viewport: { width: 768, height: 1024 } } },
    { name: "desktop", use: { viewport: { width: 1280, height: 800 } } }
  ]
})
```

- [ ] **Step 3: 提交** — `git commit -m "test(client-v2): playwright 加 mobile/tablet/desktop 三 project (P5)"`

---

## Task 13: 写 manage 响应式 smoke spec + 跑 baseline

**Files:** Create `client-v2/tests/e2e/manage-responsive.spec.ts`

- [ ] **Step 1: 写 spec**

```typescript
import { test, expect } from "@playwright/test"

const PAGES = [
  { path: "/manage/index", name: "dashboard" },
  { path: "/manage/film", name: "film-list" },
  { path: "/manage/film/class", name: "film-class" },
  { path: "/manage/collect/index", name: "collect" },
  { path: "/manage/cron/index", name: "cron" },
  { path: "/manage/file/upload", name: "file-upload" },
  { path: "/manage/file/gallery", name: "file-gallery" },
  { path: "/manage/system/webSite", name: "site-config" }
]

test.beforeEach(async ({ page }) => {
  await page.goto("/login")
  await page.fill("[name=userName]", "admin")
  await page.fill("[name=password]", "admin")
  await page.click("button[type=submit]")
  await page.waitForURL("**/index", { timeout: 10000 })
})

for (const p of PAGES) {
  test(`smoke ${p.name}`, async ({ page }) => {
    await page.goto(p.path)
    await page.waitForLoadState("networkidle")
    await expect(page).toHaveScreenshot(`${p.name}.png`, { fullPage: true })
  })
}
```

- [ ] **Step 2: 首跑 baseline** — `cd ~/gofilm/client-v2 && pnpm playwright test --update-snapshots manage-responsive` (生成 24 截屏 = 8 页 × 3 project)

- [ ] **Step 3: 人工 review snapshots** — `ls tests/e2e/manage-responsive.spec.ts-snapshots/`, scp 回本地一张张过

- [ ] **Step 4: 提交 baseline** — `git commit -m "test(client-v2): manage 响应式 smoke baseline (P5)"`

---

# 完工 checklist

- [ ] P0-P5 所有 task done
- [ ] `pnpm test -- --run` 全过
- [ ] `pnpm playwright test` 全过
- [ ] 真机 iOS Safari + Android Chrome 各跑一遍 manage 区, 关键路径 (登录/查列表/弹窗操作/汉堡菜单) 无明显问题
- [ ] git log 一阶段一组 commits 串成清晰故事

---

# 回滚策略

每阶段独立 commit, 出问题:
```bash
ssh ... "cd ~/gofilm && git -c safe.directory=\$PWD reset --hard <commit_sha>"
ssh ... "cd ~/gofilm/film && sudo docker compose up -d --build nginx"
```

特别注意:
- P0 useViewMode 改动影响公网区 (SearchView 用 isMobile), 若 SearchView 回归立即回滚
- P2 ManageSheet 替换 BaseDialog, 若样式问题 revert 单 view 即可 (BaseDialog 未删)
- P3 ManageTable card 模式默认开, 若哪个表格 fallback 显著破窗, 临时该页传 `mobileVariant="scroll"` 兜底
