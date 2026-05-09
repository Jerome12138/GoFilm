# gofilm-client-v2

GoFilm Vue3 重构客户端（用户端 + 管理端 + TV 模式）。

## 技术栈

- Vue 3.5 + Vite 5 + TypeScript 5
- Pinia 2 + Vue Router 4
- UnoCSS（preset-uno + attributify + icons）
- axios + @vueuse/core + video.js

## 开发

```bash
pnpm install
pnpm dev          # 启动 :3600，代理 /api → 127.0.0.1:3601
pnpm type-check   # 类型检查
pnpm build        # 产物输出 dist/
```

## 目录约定

详见 `doc/bmad/architecture-redesign.md` 第 3 节与
`doc/handover/03-arch-handover.md` 的"文件落点速查表"。

- `src/api/`     接口模块（业务域拆分）
- `src/stores/`  Pinia 全局状态
- `src/router/`  路由（public + manage 双布局）
- `src/views/`   页面视图（按业务域归档）
- `src/components/{base,layout,film}/` 自动注册的 base 组件 + 布局壳 + 影视业务组件
- `src/composables/` 组合式工具
- `src/types/`   DTO / 接口类型
- `src/utils/`   通用工具

## 三套视图模式

通过 `<html data-mode="mobile|desktop|tv">` 切换：

- 自动检测：UA + 视口宽度 + hover 能力
- 强制：`localStorage.setItem('gf-mode','tv')` 或 URL `?mode=tv`
- 详见 `src/composables/useViewMode.ts`

## 编码守则

阅读 `doc/handover/03-arch-handover.md` 第 2、3 节。
违反"反模式清单"的代码 review 会被打回。
