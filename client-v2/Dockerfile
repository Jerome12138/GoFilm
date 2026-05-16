# 多阶段: node 构建前端 → nginx 静态托管.
# 上下文 = 仓库根 (compose build.context: ..)
FROM node:20-alpine AS build

WORKDIR /app

# 先拷锁文件再拷源码, 复用 docker layer 缓存
COPY client-v2/package.json client-v2/package-lock.json* ./
RUN npm ci --no-audit --no-fund

COPY client-v2/ ./
# build:no-check 跳过 vue-tsc 类型检查, 生产构建假设 CI 已经跑过 type-check
RUN npm run build:no-check

# 运行镜像: nginx 静态托管 + /api/ 反代到 server.
FROM nginx:1.27-alpine

# nginx.conf 走仓库版本, 不再依赖宿主机 /opt/film/data/nginx 挂载
COPY film/data/nginx/nginx.conf /etc/nginx/nginx.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80

# nginx 镜像自带 wget, healthcheck 直接探 /
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1/ >/dev/null 2>&1 || exit 1
