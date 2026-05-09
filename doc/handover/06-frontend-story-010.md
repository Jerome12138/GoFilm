# 06 前端交接 — STORY-010（PlayView 播放页）

> 阶段：用户端最复杂页面 — video.js 播放器 + 多源切换 + 历史记录写入
> 完成时间：2026-05-08
> 工作目录：`D:/Git/GoFilm/client-v2/`

## 1. 范围

- STORY-010：PlayView.vue 完整重写
- 配套：usePlayer composable / useFilmHistory composable / history store 重构
- 修正：`types/film.ts::PlayInfo` 类型与后端实际响应对齐
- 联动：PublicHeader 历史浮层跳转改为带续播参数的 link push

## 2. 落地文件

| 文件 | 状态 | 说明 |
|---|---|---|
| `src/composables/usePlayer.ts` | 新建 | video.js 实例封装；命令式 API；shallowRef 实例；ready 前 on() 排队 |
| `src/composables/useFilmHistory.ts` | 新建 | PlayView 卸载 / beforeunload / 切集时调 store.record；buildPlayLink 工具 |
| `src/stores/history.ts` | 重写 | 改为旧站 `{ [filmId]: ... }` 对象映射；双写 cookie + localStorage；兼容旧数组结构 |
| `src/types/film.ts` | 修正 | `PlayInfo` 改为 `{ detail, current, currentPlayFrom, currentEpisode, relate }`，原定义全错 |
| `src/views/public/PlayView.vue` | 重写 | 播放器 + 标题 + 自动连播 + 集数 + 简介 + 相关推荐 + 错误态 |
| `src/components/base/BaseIcon.vue` | 扩展 | 加 7 个图标（pause / play-circle / skip-next / skip-prev / volume-up / volume-down / autoplay） |
| `src/components/layout/PublicHeader.vue` | 改造 | 历史浮层跳转改 router.push(link)，保留 currentTime |
| `src/assets/play.png` | 复制 | 从旧站复制的 poster 占位图 |

## 3. 关键决策与坑点

### 3.1 PlayInfo 类型修正
旧 `types/film.ts::PlayInfo` 定义为 `{ src, episode, link, prev, next }`，**完全不符合后端响应**。检查 `server/controller/IndexController.go::FilmPlayInfo` 后改为：
```ts
interface PlayInfo {
  detail: FilmDetail
  current: PlayEpisode  // { episode, link }
  currentPlayFrom: string
  currentEpisode: number
  relate: FilmListItem[]
}
```

### 3.2 video.js 实例不要被 reactive 包
- 用 `shallowRef<Player>` 存实例
- video.js 内部上千个属性，深 proxy 化会导致严重性能问题甚至栈溢出
- 模板内绑定播放器属性时只读 composable 暴露的派生 ref（paused / currentTime / duration）

### 3.3 切换 src 不重建 player
```ts
// usePlayer 内部
watch(() => unwrap(opts.src), (next) => {
  player.value.src({ src: next, type })
})
```
`player.dispose()` 只在卸载时调用一次。这样 volume / playbackRate / fullscreen 状态都保留，用户体验好。

### 3.4 cookie 结构兼容旧站
- 旧站约定：`filmHistory` cookie 为 `{ [id]: { name, link, episode, timeStamp } }`
- 新站新增 source / episodeIndex / currentTime / picture（旧站读时忽略多余字段）
- store 同时双写 cookie + localStorage（cookie 4KB 限制）
- 启动加载顺序：cookie → localStorage 兜底
- 兼容早期 STORY-006 的 array 形式自动转 map

### 3.5 集数切换 - 不刷新页面
- 触发 `selectEpisode(...)` 后流程：
  1. flushHistory()（写当前进度到 cookie）
  2. currentSourceId / currentEpisodeIndex 更新
  3. currentSrc.value 改变 → usePlayer watch 触发 player.src(...)
  4. router.replace 同步 query（**replace 而非 push**，避免历史栈污染）
  5. nextTick 后 playerPlay()
- watch route.query 仅在 path === '/play' 时响应，且只在影片 id 变化时调 loadPlayInfo()，否则做幂等同步

### 3.6 键盘事件统一 handler
- 桌面 + TV 共用同一个 `handleKeydown`
- 先调 `normalizeDpadKey(e)` 把 D-pad keyCode 19/20/21/22/23 归一化为 ArrowUp/Down/Left/Right/Enter
- 输入框 / contenteditable 聚焦时跳过，避免抢用户输入
- Space/Enter 在 button/a 元素聚焦时跳过，让原生交互生效
- Escape 走 `router.push('/filmDetail', { link: id })` —— 维持 SPA 体验

### 3.7 视频源错误降级
- on('error') → 取模找下一个 source → selectEpisode
- 保护：
  - sources.length ≤ 1 直接返回
  - nextIdx === curIdx 直接返回（避免单源时无限循环）
  - 集数索引在新源做 `Math.min(idx, list.length - 1)` 截断

### 3.8 BaseIcon 扩展
- 加 7 个图标：pause / play-circle / skip-next / skip-prev / volume-up / volume-down / autoplay
- 仍走 inline SVG（preset-icons 在 unocss 0.62.4 仍未启用）

### 3.9 历史浮层跳转 - 续播
- PublicHeader 历史项原来用 `router.push({ path:'/play', query:{...} })` 拼参，丢失 currentTime
- 改为 `router.push(item.link)` —— store 写入时已包含 `&currentTime=NN`
- 路由守卫无影响，依然命中 `/play` 路由

## 4. 编码守则遵守

- API 调用仅经 `filmApi.getPlayInfo`
- 路由跳转全部 `router.push/replace`
- watch 仅 `route.query.id/source/episode`，不全量 watch route
- 不对 reactive 整体赋值
- `onBeforeRouteLeave` + `onScopeDispose` 双 dispose player
- 错误 UI 走 BaseEmpty，toast 由 http 拦截器统一
- TV 模式样式写在末尾 unscoped `<style>` 块（不能用 `:global()`）

## 5. 验证

```
cd client-v2
pnpm exec vue-tsc -p tsconfig.app.json --noEmit  # ✓
pnpm build                                         # ✓ ~20s
```

构建产物：
- `PlayView-*.js` 11.46 KB / `PlayView-*.css` 3.01 KB
- **video-vendor-*.js 683.26 KB**（gzip 204.72 KB）— 之前为空，本期填充
- 总 build 时间 20.16s

## 6. 后端字段对接疑问（请回复）

1. **`PlayInfo` 字段**：本期已根据 `IndexController.go::FilmPlayInfo` 更正类型。请确认接口实际响应字段名 / 大小写与代码一致：
   - `detail`（含 `list: PlaySource[]`，`descriptor`，`id`，`name`，`picture`，`pid`，`cid`）
   - `current: { episode, link }`
   - `currentPlayFrom: string`
   - `currentEpisode: number`
   - `relate: FilmListItem[]`
2. **`detail.list[].id`**：是否 = `MovieDetail.PlayFrom[i]`？前端用其作 sourceId 唯一键。
3. **`current.link`**：是直接的 m3u8 / mp4 url，还是有时是 iframe 跳转地址？前端目前直接喂给 video.js，前者可工作，后者需要前端 detect 并切 iframe。
4. **空 list 容错**：当 `detail.list` 为空数组时，应该如何处理？当前会渲染空播放器（黑屏）。

## 7. 待办（不在本 story 范围）

1. PlayView 接 AbortController（getPlayInfo 加 signal 支持）
2. HistoryView 真实卡片渲染（store.list 已就绪）
3. video.js 控件 D-pad 焦点接管（当前依赖原生焦点，TV 上控件可能不易聚焦）
4. 多语言（i18n）：当前所有文案 hardcoded 中文，待全站 i18n 框架落地后回写
5. 进度续播 UX：当前从 query 读 currentTime 一次性 seek，可加"上次看到 12:34，继续观看"提示
