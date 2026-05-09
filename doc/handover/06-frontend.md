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

---

# 06-2 STORY-011 + STORY-012（Search / Classify / ClassifySearch + useQuerySync）

> 完成时间：2026-05-08
> 工作目录：`D:/Git/GoFilm/client-v2/`

## 1. 范围

- STORY-011：SearchView（关键字搜索 `/search`）
- STORY-012：ClassifyView（分类首页 `/filmClassify`） + ClassifySearchView（分类筛选 `/filmClassifySearch`）
- 新增 `useQuerySync` composable（统一处理 URL query <-> 数据 ref 双向绑定）

## 2. 落地文件

| 文件 | 状态 |
|---|---|
| `src/composables/useQuerySync.ts` | 新增 |
| `src/views/public/SearchView.vue` | 重写（占位 → 实装） |
| `src/views/public/ClassifyView.vue` | 重写 |
| `src/views/public/ClassifySearchView.vue` | 重写 |
| `src/types/film.ts` | 调整 ClassifyData / 新增 BackendPage / ClassifySearchResp / SearchFilmResp / ClassifyTitle / ClassifyTagItem |
| `src/api/film.ts` | getClassify / searchClassify / searchFilm 返回类型对齐后端实际字段 |

## 3. useQuerySync 用法

```ts
const { params, push, replace } = useQuerySync<{
  search: string
  current: number
}>(
  { search: '', current: 1 },
  { path: '/search', onChange: () => load() }
)
// 读：params.value.search
// 写：push({ search: 'xxx', current: 1 })
// 浏览器前进/后退会自动触发 watch route.query → 同步 params + onChange
```

要点：
- 初始值同时决定字段名 + 字段类型（数字字段读 query 时自动 Number()）
- skip undefined / null / '' 字段（不写入 URL，也不带回 params）
- push 后内部 suppressNext 跳过自身回调，不会重复 onChange
- 业务里**不要再写 `watch(route, ...)` 触发请求** — 用 onChange / 直接 watch params

## 4. 后端字段对照表（非常重要）

| 接口 | 实际响应（后端返回）|
|---|---|
| `/searchFilm` | `{ list: FilmListItem[], page: { pageSize, current, pageCount, total } }` |
| `/filmClassify` | `{ title: { id, pid, name, show }, content: { news, top, recent } }` |
| `/filmClassifySearch` | `{ title, list, page, search: { sortList, titles, tags }, params }` |

`ClassifyData.content.news/top/recent` 与 04 期占位类型 `newest/ranking/recent` 不一致 — 已修正为后端约定。

`search.tags[key]` 列表项字段为 `Name / Value`（旧站约定），FilmFilterBar 传入时已映射为 `{ value, label }`。

## 5. URL query 大小写硬约束

旧站 `/filmClassifySearch` query 字段使用首字母大写：

```
Pid / Category / Plot / Area / Language / Year / Sort
```

`current` 是小写（与 SearchView 一致）。

useQuerySync 在内部按 initial 中的字段名直接读写，因此**只要在调用方写对就 OK**，不要在 watch / push 处把它转成 camelCase。

## 6. 后端字段疑问（待联调确认）

1. `filmClassify` 返回的 `title` 对象，前端目前推断字段为 `{ id, pid?, name, show? }`。架构文档第 8 节未明确 DTO，已写在 `ClassifyTitle` 兜底，需要联调时核对。
2. `filmClassifySearch.search.tags[key]` 列表项字段 `Name / Value`（首字母大写）— 与旧站对齐。如改名将影响 FilmFilterBar 渲染。
3. `searchFilm` 后端硬编码 PageSize=10、`filmClassifySearch` PageSize=49 — 前端 BasePagination 已配合处理。
4. `searchFilm` 关键字搜索为空时后端会返回 code != 0 + msg："暂无相关影片信息"。当前响应拦截器会 toast。建议改为返回 code=0 + 空列表 + total=0，避免每次空结果都 toast。**已用 SearchView 的 try/catch 兜底**，不会页面卡死，但用户体验仍差，建议后端调整。

## 7. 给下游开发者的提示

1. 任何带 query 的页面（详情页 `link`、播放页 `id/source/episode/currentTime`）都建议改用 useQuerySync。
2. 加新筛选维度（如 Score）：
   - 后端先在 `search.sortList` 加 `Score`，加 `titles.Score = "评分"`，加 `tags.Score = [{Name, Value}, ...]`
   - 前端无需改 ClassifySearchView — initial 中加 `Score: ''`，FilmFilterBar 自动渲染
3. 切换页码 / 筛选项之后会 `window.scrollTo({ top: 0, behavior: 'smooth' })`，TV 端如有遥控器焦点回落需求需另外实现（不在本 story 范围）。
4. ClassifySearchView 的"无符合条件"空状态在 `loaded && !errorMsg && films.length===0` 时显示，避免首次未到达时闪屏。

## 8. 验证

```
cd client-v2
pnpm exec vue-tsc -p tsconfig.app.json --noEmit   # ✓
pnpm build                                         # ✓ 10.22s
```

产物：
- SearchView 6.12 KB / ClassifyView 3.44 KB / ClassifySearchView 5.03 KB
