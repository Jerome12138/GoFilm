# 06 前端交接 — STORY-006 + STORY-007（基础组件 + 影视业务组件）

> 阶段：base 组件 + film 业务组件 + 首页视觉验证  
> 完成时间：2026-05-08  
> 工作目录：`D:/Git/GoFilm/client-v2/`

## 1. 范围

- STORY-006：通用基础组件（自动注册到全局，src/components/base/）
- STORY-007：影视业务组件（src/components/film/）
- HomeView 改造为 mock 数据驱动的视觉验证页

布局壳（PublicLayout / PublicHeader / PublicFooter）、其余 view、API 接入均不在本任务范围。

## 2. 落地文件清单

### 新增 base 组件（共 8 个，含临时新增的 BaseIcon）

| 文件 | 用途 |
|---|---|
| `src/components/base/BaseButton.vue` | 通用按钮，5 variant × 4 size，icon slot / loading |
| `src/components/base/BaseImage.vue` | IntersectionObserver 懒加载 + 错误回退占位 |
| `src/components/base/BaseSkeleton.vue` | 骨架屏，rect / circle / text，count 重复 |
| `src/components/base/BasePagination.vue` | 分页 chip，移动端简化 |
| `src/components/base/BaseEmpty.vue` | 空状态（slot icon / title / description / action） |
| `src/components/base/BaseTag.vue` | 标签，7 variant × 3 size，outlined |
| `src/components/base/BaseDialog.vue` | Teleport 弹窗，ESC 关闭，scale 动画 |
| `src/components/base/BaseIcon.vue` | inline SVG 图标集合（10 个常用） |

保留：`BasePagePlaceholder.vue` / `BaseToastContainer.vue`。

### 新增 film 业务组件（7 个）

| 文件 | 关键 props / emits |
|---|---|
| `src/components/film/FilmCard.vue` | `item: FilmListItem`, `score?`, `lazy?`, `showTitleBelow?` |
| `src/components/film/HeroCarousel.vue` | `items: FilmListItem[]`, `interval?`, `showArrows?` |
| `src/components/film/FilmRow.vue` | `title?`, `moreLink?`, `items[]`, `itemKey?`；slot `item` |
| `src/components/film/FilmGrid.vue` | `items[]`, `gap?`；slot `item` |
| `src/components/film/FilmFilterBar.vue` | `groups: FilterGroup[]`；emit `change({key, value})` |
| `src/components/film/EpisodeTabs.vue` | `sources`, `currentSourceId?`, `currentEpisode?`, `watchedLinks?`；emit `select / change-source` |
| `src/components/film/RelatedList.vue` | `items: FilmListItem[]`, `title?` |

### 改造文件

- `src/views/public/HomeView.vue` — mock 数据驱动，HeroCarousel + 3 个 FilmRow（仅作视觉验证，STORY-008 替换为 `/api/index`）
- `src/components/base/BaseTag.vue` — rgba arbitrary value 改用 scoped class（unocss arbitrary value 解析坑）
- `src/components/base/BaseImage.vue` — `transition-opacity duration-[var(...)] ease-[var(...)]` 移到 scoped CSS

### 配置改动

- `uno.config.ts`
  - 临时禁用 `presetIcons`（preset-icons 0.62.4 + carbon/mdi 在 build 阶段输出非法 CSS）
  - 业务图标改走 BaseIcon 内置 inline SVG 集合

## 3. 关键决策与坑点（务必周知后续开发者）

### 3.1 preset-icons 暂时禁用
- **现象**：build 报 `[unocss:global:build:scan] [postcss] Missed semicolon`，无论用 `i-carbon-search`、`i-mdi-image-off` 都失败
- **临时方案**：禁用 preset-icons，BaseIcon 组件内置 10 个常用 SVG（play / info / chevron-left|right / search / close / image / menu / plus / minus），通过 `<BaseIcon name="play" size="20px" />` 调用
- **后续优化**（建议放到 STORY-009 或单独 ticket）：升级 unocss 至 ≥ 0.65（与 preset-icons 一起升），或迁移到 `unplugin-icons`

### 3.2 unocss arbitrary value 不要乱用
- ❌ `ease-[var(--gf-ease-standard)]` — preset-uno 不识别任意值给 transition-timing-function
- ❌ `bg-[rgba(155,73,231,0.2)]` — 含逗号的 rgba 在 attributify 模式下偶发解析问题
- ❌ `transition-[background-color,box-shadow,...]` — 多值 transition 不可靠
- ✅ 把这些写到 `<style scoped>` 内的 transition 声明里
- ✅ 简单 token：`gap-[var(--gf-space-4)]` / `text-[var(--gf-fs-md)]` / `rounded-[var(--gf-radius-md)]` 都 OK

### 3.3 Vue scoped 中没有 `:global()`
- 这是 CSS Modules 语法，Vue SFC scoped 不支持
- 正确做法：在同一 SFC 末尾追加一个 unscoped `<style>` 块写 `[data-mode='tv'] .xxx`

### 3.4 TV 模式覆盖
- 所有组件已按 04-tv-addendum 加 `data-focusable="true"`、TV 默认显示箭头 / 标题、focus-visible 强对比环
- TV 字号 / 间距覆盖完全在 `theme.css` 的 `[data-mode='tv']` 段，组件层不再重复覆盖
- HeroCarousel 自动播放间隔在 TV 模式下从 4s 变为 6s（用 `useViewMode().isTV` 判定）

### 3.5 路由跳转规范
- FilmCard 用 `<RouterLink :to="{path:'/filmDetail',query:{link:String(id)}}">`
- HeroCarousel CTA 用 `router.push({...})`
- 绝不 `<a href>` / `location.href`

## 4. 验证

```
cd client-v2
pnpm exec vue-tsc -p tsconfig.app.json --noEmit   # ✓
pnpm build                                         # ✓ ~7s
pnpm dev                                           # http://localhost:3600
```

视觉确认：访问 `/index` 应看到：
- 顶部 60vh~70vh Hero 自动轮播（3 张）
- 下方 3 个横向滚动 Row（mock 14 张/行）
- 卡片 hover 放大 + 浮层标题
- TV 模式（`?mode=tv`）下卡片标题常驻、箭头默认可见

## 5. 给下游开发者的提示

1. **新增图标**：直接编辑 `BaseIcon.vue` 的 PATHS map，加新条目，types 自动收紧。不要恢复用 `i-carbon-*`。
2. **新增 base 组件**：放到 `src/components/base/` 自动注册全局，view / film 内不需要 import。
3. **新增 film 组件**：放到 `src/components/film/`，view 内手动 import（这部分未配置自动注册）。
4. **API 接入 STORY-008**：`HomeView.vue` 中替换 mock 为 `useFilmIndex()` composable + `/api/index`，组件不动。
5. **筛选页**：直接用 `FilmFilterBar` + `FilmGrid` + `BasePagination`，配 `useQuerySync` 双向绑定 query。
6. **详情页**：`FilmDetailHeader` 视觉规范见 `components-spec.md` 第 5 节，本 story 未交付（属 view 层），由下一个 story 用 BaseImage / BaseTag / BaseButton + EpisodeTabs + RelatedList 拼装。

## 6. 待办（不在本 story 范围）

- `BaseIcon` 图标集合需要扩充（heart / star / play-fill / pause / volume / settings / user 等）
- 升级 unocss 重新启用 preset-icons（或迁移 unplugin-icons），届时回退本次的 inline SVG
- 添加 BaseDialog 单元测试（Teleport / ESC / scrollLock）
- 加 Storybook 或 docs 页面预览 base / film 组件（建议放到 STORY-013 视觉打磨阶段）
