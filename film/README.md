# Film Build

## 1. 目录结构说明

- `data/` 服务容器相关的数据与配置
  - `nginx/`
    - `html/` 放 `client-v2/` 构建后的 dist 静态资源
    - `nginx.conf` 反代后端 API + SPA history fallback
  - `redis/`
    - `redis.conf` redis 远程访问 / 密码
- `docker-compose.yml` 编排 nginx / film / mysql / redis
- `Dockerfile` 构建 film 后端镜像 (Go), 上下文为 **仓库根**, 直接 COPY 根 `server/` 进镜像 — 不再保留 `film/server/` 快照, 避免双份维护

```text
GoFilm/
├─ server/                # 后端唯一源码 (docker 构建直接拿这里)
├─ client-v2/             # 前端唯一源码 (pnpm build 后产物拷到下面 html/)
└─ film/
   ├─ data/
   │  ├─ nginx/
   │  │  ├─ html/         # 来源: client-v2/dist
   │  │  └─ nginx.conf
   │  └─ redis/
   │     └─ redis.conf
   ├─ docker-compose.yml  # build.context: .. (仓库根)
   ├─ Dockerfile          # COPY server/ /opt/server/
   └─ README.md
```

> 历史上此处有过 `film/server/` 旧快照, 已删除. 现在 docker compose build 直接读
> 仓库根 `server/`, 不再需要手动同步.

## 2. 程序构建运行

### 1. 环境准备

1.  Linux 服务器
2.  安装 docker, docker compose 服务
    - Centos 安装 Docker Engine  [官方文档链接](https://docs.docker.com/engine/install/centos/)
    - Ubuntu 安装 Docker Engine   [官方文档链接](https://docs.docker.com/engine/install/ubuntu/)

```shell
# Centos 系统安装Docker Engine示例
# 1. 卸载旧版本Docker
sudo yum remove docker \
                  docker-client \
                  docker-client-latest \
                  docker-common \
                  docker-latest \
                  docker-latest-logrotate \
                  docker-logrotate \
                  docker-engine

#2. 设置存储库
sudo yum install -y yum-utils
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
#3. 安装最新版本Docker
$ sudo yum install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
#启动 Docker 服务
sudo systemctl start docker
```

### 2. 启动流程

> 默认配置流程:

1. 把**整个仓库**上传到服务器 (例如 `/opt/GoFilm`); film 依赖根 `server/` 与 `client-v2/`, 单独传 `film/` 是不够的.
2. **构建前端静态资源**

   ```bash
   cd /opt/GoFilm/client-v2
   pnpm install --frozen-lockfile
   pnpm build
   cp -r dist/* /opt/GoFilm/film/data/nginx/html/
   ```
3. **切换后端 DSN 到 docker 网络** (启用 `mysql:3306`):

   编辑 `/opt/GoFilm/server/config/DataConfig.go`, 注释掉 `192.168.20.10:3307` 那行, 启用 `mysql:3306` 那行 (文件里两行注释切换). Redis 同理改 `RedisAddr = "redis:6379"`.
4. **构建并启动**

   ```bash
   cd /opt/GoFilm/film
   docker compose build       # 上下文是 .. (仓库根), 自动拉 server/ 与 go mod
   docker compose up -d
   docker ps                  # 确认 film_nginx / film_api / film_mysql / film_redis 4 容器都在跑
   ```
5. 等待 3~8 分钟初始化, 访问后台 `http://xxx.xxx.xxx/manage`, 默认 `admin / admin` (登录后立即改密).
6. 在 `采集管理` 启动一次采集; 前台 `http://xxx.xxx.xxx/index` 看效果.

### 3. 服务配置信息修改

- **后端代码**: 直接改根 `server/` 然后 `docker compose build film` 重建 (不要再改 `film/server/`, 该目录已删除).
- mysql 用户名/密码/端口: 改 `docker-compose.yml` 同时同步 `server/config/DataConfig.go` 里的 DSN.
- redis: `/film/data/redis/redis.conf` + `server/config/DataConfig.go` 里的 `RedisAddr` 双写.
- nginx: `/film/data/nginx/nginx.conf`.

>注意事项

-  mysql 和 redis 服务配置修改后需要同步修改根 `server/config/DataConfig.go` 中的连接地址和账户名信息

```go
## 配置使用的用户名密码信息需和ocker-compose.yml文件中设置的一致
const (
	// mysql服务配置信息修改
	mysqlDsn = "用户名:密码$@(服务名:服务端口)/FilmSite?charset=utf8mb4&parseTime=True&loc=Local"

	/*
		redis 配置信息
		RedisAddr host:port
		RedisPassword redis访问密码
		RedisDBNo 使用第几号库
	*/
	RedisAddr = `服务名:服务端口`
	RedisPassword = `密码`
	RedisDBNo = 0
)
## docker-compose.yml (设置服务的启动端口和服务名以及账户密码信息)
mysql:
    container_name: film_mysql
    image: mysql
    ports:
    - 3610:3306
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: FilmSite
 redis:
    container_name: film_redis
    image: redis
    ports:
      - 3620:6379
```



### 4. 常见问题

1.  CPU 架构为 ARM 的服务器部署时 需修改Dockerfile 文件中的 `GOARCH=amd64` 为 `GOARCH=arm`
2.  服务器内存偏小时, 可能自行将redis容器关闭, 需在宿主机 `/etc/sysctl.conf` 文件中追加 `vm.overcommit_memory = 1` 配置, 并执行 `sysctl vm.overcommit_memory=1` 使其生效



### 5. 管理后台基本使用说明

- 访问 http://xxx.xxx.xxx/manage 进行登录,  用户名 密码: `admin admin` , 登录成功后自行修改
- 使用 `采集管理 -> 影视采集` 功能进行采集站信息的添加和更新,  系统初始化时有预留站点信息, 自行斟酌选择
- 首先选择一个站点为主站点, 然后选择 采集一周, 一天, 或 全部, 进行主站点的数据采集, (前台数据全来自于主站点, 因此主站点只需要存在一个, 否则会冲突)
- 主站点信息采集完成后则可在 影片管理 -> 影视分类 中进行主页分类展示信息的设置, 选择需要展示的分类信息, 以及分类的名称管理 (自行摸索如何使用)
- 附属站点即影片的多个播放来源, 可自行进行选择性的采集添加
- 定时任务: 
  - 系统默认添加一条规则, 但未启用, 该规则为每20分钟更新一次所有已开启的采集站的近3小时内更新的数据, 开启后则基本满足资源更新需求
  - 可自定义定时任务, 使用相关功能可自主选择对某些站点进行定时更新功能

