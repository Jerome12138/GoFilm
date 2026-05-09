# 06 前端交接 — STORY-008 + STORY-009 + 布局壳打磨

> 阶段：HomeView 接 API + FilmDetailView 实装 + PublicHeader/Footer/Layout 重写
> 完成时间：2026-05-08
> 工作目录：`D:/Git/GoFilm/client-v2/`
> 前置文档：`doc/handover/06-frontend.md`（STORY-006/007 base + film 组件交付）

---

## 1. 范围

- **STORY-008**：HomeView 替换 mock，接 `/api/index`
- **STORY-009**：FilmDetailView 替换占位，接 `/api/filmDetail?id=`
- **布局壳打磨**：PublicHeader / PublicFooter 重写，PublicLayout 调整
- **App.vue**：站点信息预热并行化

不在本 story 范围：PlayView、收藏功能、分享功能、TV D-pad 联调、Storybook。

---

## 2. 落地文件清单

### 重写 (4 个)

| 文件 | 关键改动 |
|---|---|
| `src/components/layout/PublicHeader.vue` | 滚动透明→实色；左站名 gradient；中搜索 pill；右导航 + 历史浮层 + 移动汉堡；TV 高度 96 + 安全区 |
| `src/components/layout/PublicFooter.vue` | 极简三列（移动单列居中）；站名 + 描述 + 站点导航 + 备案号 + 版权 |
| `src/components/layout/PublicLayout.vue` | main 区 `flex: 1 0 auto` + 不同 mode 的 min-height 计算 |
| `src/views/public/HomeView.vue` | 接入 `filmApi.getIndex()`；Hero（banner 优先）+ 多行 FilmRow + lg 桌面热播旁栏；骨架屏 + 错误重试 |
| `src/views/public/FilmDetailView.vue` | 接入 `filmApi.getFilmDetail()`；模糊海报 hero + 信息区 + 集数 + 相关推荐；剧情展开/收起 |

### 改动 (2 个)

| 文件 | 改动 |
|---|---|
| `src/App.vue` | onMounted 内 `Promise.all([siteStore.ensureLoaded(), navStore.ensureLoaded()])` 并行预热 |
| `src/components/base/BaseIcon.vue` | 扩展 17 个图标（history / star / heart / share / user / home / arrow-left / film / chevron-up/down / pause / play-circle / skip-next / skip-prev / volume-up/down / autoplay） |

---

## 3. 数据流

### HomeView
```
onMounted → filmApi.getIndex() → 返回 IndexPageData
  ├── banner[]            → HeroCarousel.items
  └── content[].{nav, movies, hot}
       ├── 每个 movies → 一个 FilmRow（title=nav.name, more=/filmClassify?Pid=nav.id）
       └── 所有 hot 合并去重前 12 → 桌面 ≥1024 旁栏热播榜
```

### FilmDetailView
```
route.query.link → filmApi.getFilmDetail(id) → { detail, relate }
  detail
    ├── picture       → hero 模糊背景 + 海报
    ├── name          → 标题
    ├── descriptor    → 评分 / chips / 导演 / 主演 / 上映 / 地区 / 剧情
    └── list[]        → EpisodeTabs (sources)
                         点击集数 → router.push('/play', { id, source, episode })
  relate              → RelatedList
```

---

## 4. 编码守则对照（已落实）

| 守则 | 落实点 |
|---|---|
| API 必经 `api/*.ts` | `filmApi.getIndex()` / `filmApi.getFilmDetail(id)` |
| 路由 query 出入口集中 | 全部走 `router.push({ path, query })` |
| 不在 template 拼 query | 详情卡 / 集数 / 历史项均用对象 to |
| 列表数据 ref 整体替换 | `state.value = { loading, errored, data }` |
| onMounted 并行请求 | App.vue 用 Promise.all 预热 site + nav |
| 不直接读 cookie | Header 通过 `useHistoryStore().list` |

---

## 5. 关键决策（与坑）

1. **Header 滚动监听**：`window.addEventListener('scroll', ..., { passive: true })`，阈值 `scrollY > 12`；卸载时移除。比 IntersectionObserver 简单且不会被 sticky 父级影响。
2. **历史浮层 hover 防闪退**：`setTimeout(close, 200)`，鼠标进入浮层 `clearTimeout`。
3. **Header `--top` 状态**：保留 `linear-gradient(180deg, rgba(0,0,0,0.55), 0)` 淡蒙版，不完全透明，避免亮色 hero 海报上白字看不清。
4. **TV 模式 Header 实色**：直接覆盖为 `--gf-bg-header-scrolled`，去掉淡蒙版（10-foot UI 不需要"沉浸感"）。
5. **FilmDetail hero 背景**：
   - `background-image: url('${heroBg}')`（动态绑定，不能用 attr()）
   - `filter: blur(40px) brightness(0.4)`
   - `inset: -40px` + `transform: scale(1.1)` 防止 blur 边缘透明断层
   - `isolation: isolate` 防止层叠上下文污染
6. **剧情清洗正则**：`/(&.*?;)|( )|(　　)|(\n)|(<[^>]+>)/g`，把旧站的贪婪 `&.*;` 改为 `&.*?;`。
7. **导演/主演切片**：`split(/[,，、\/\s]+/)` 兼容中英文逗号、顿号、斜杠、空格。
8. **HomeView 旁栏 deep**：`:deep(.gf-film-row > header.container-page)` 在 ≥ lg 双列布局下覆盖 FilmRow 内部 container 的 padding，让 row 与 sidebar 平铺。少数必须穿透的场景。
9. **HistoryRecord 兼容**：旧站 cookie 项是 `{ id, name, link, episode, timeStamp, picture? }`，Header 浮层优先 `router.push(item.link)` 利用旧站完整 link 续播。

---

## 6. 后端对接疑问（联调时校对）

| 字段 | 当前处理 | 联调点 |
|---|---|---|
| `IndexPageData.banner` | 不存在则回退 `content[0].movies.slice(0, 5)` | 旧后端是否总返回 banner？ |
| `FilmDetail.descriptor.dbScore` | `Number(s).toFixed(1)`，0/空串不显示 | 后端字段类型是 string 还是 number？ |
| `FilmDetail.descriptor.classTag` | 用 `[,，、\/\s]+` 切片 | 后端实际分隔符 |
| `FilmDetail.list[i].id` | 作为 `source` query 传给 /play | 与 PlayInfo.playFrom 是否一致 |
| `HistoryRecord.link` | 优先 push 字符串保留 currentTime | 链接格式是否稳定 |

---

## 7. 验证

```bash
cd client-v2
pnpm exec vue-tsc -p tsconfig.app.json --noEmit   # ✓
pnpm build                                         # ✓ 10.01s
```

产物体积参考：
- HomeView 7.35 KB / 1.36 KB css
- FilmDetailView 8.62 KB / 1.61 KB css
- 全量 index.css 35.22 KB

浏览器手动验证（无后端时）：
- `/index` → API 失败 → BaseEmpty「加载失败」+ 重试按钮
- `/index?mock` → 无影响（不再走 mock）
- `/filmDetail` 无 link → BaseEmpty「请从首页或搜索进入」
- `/filmDetail?link=xxx` → API 失败 → BaseEmpty「影片不存在或加载失败」
- 滚动 Home → Header 切实色背景
- 移动端 viewport → Header 收成两个图标 + 抽屉

---

## 8. 给下游开发者的提示

1. **PlayView 接入**（下一个 story）：`route.query.{id, source, episode, currentTime?}`，用 `filmApi.getPlayInfo({ id, playFrom: source, episode })`。需要在播放完毕时调 `useHistoryStore().record({ ... })` 写入完整 link 字符串，沿用旧站 link 格式 `/play?id=&source=&episode=&currentTime=`。
2. **收藏功能**：FilmDetail 的"收藏"按钮当前为静态；后续接入 `/api/favorite/*` API 时，按现有 BaseButton 写法替换 `@click` 即可。
3. **分享 Dialog**：用 `BaseDialog`，复制当前页面 url（`window.location.href`）到剪贴板。
4. **新增图标**：直接编辑 `BaseIcon.vue` 的 PATHS map（24×24 viewBox 单色填充）。
5. **TV 焦点环**：`data-focusable="true"` 已加在导航 / 集数 / 历史项；如新增交互元素请同步加。
6. **修改 App.vue 预热**：如需新增全站 store 预热，加到 `Promise.all([...])` 数组里即可。
