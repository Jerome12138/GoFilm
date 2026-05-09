# 用户中心 API 接入指南 (注册 / 登录 / 观看历史 / 收藏)

> 服务端版本：`feature/redesign` 分支，提交 `4e55a25` 之后  
> 受众：前端 (`client-v2`)

本文档列出本次为普通用户新增的所有接口、请求/响应结构、鉴权要求与典型调用流程。所有接口都遵循已有的统一返回封装。

---

## 0. 通用约定

### 0.1 BaseURL

与现有接口保持一致，由部署环境决定（默认 `http://127.0.0.1:3601`）。

### 0.2 鉴权方式

- 登录成功后服务端在响应头返回 `new-token`，前端自行存储。
- 后续鉴权请求需在请求头携带：

  ```
  auth-token: <token>
  ```

- token 临近过期会被服务端自动续签，新的 token 通过响应头 `new-token` 返回。前端拦截器收到 `new-token` 应覆盖本地存储。
- token 无效或同账号在其他设备登录会返回 HTTP `401`，前端应清除本地 token 并跳到登录页。

### 0.3 统一响应格式

```json
{
  "code": 0,
  "data": { ... },
  "msg": "ok"
}
```

- `code = 0` 业务成功
- `code = -1` 业务失败（HTTP 仍是 200，错误信息在 `msg`）
- HTTP `401`：未登录 / token 失效（在 `AuthToken` 中间件层面拦截）

### 0.4 分页参数

GET 列表类接口统一使用 query：

| 参数 | 含义 | 默认值 | 上限 |
|---|---|---|---|
| `current` | 当前页 (从 1 开始) | 1 | — |
| `pageSize` | 每页条数 | 20 | 100 |

返回中带回 `page` 字段：

```json
{ "pageSize": 20, "current": 1, "pageCount": 5, "total": 87 }
```

---

## 1. 注册

### 1.1 `POST /user/register` —— 公共

**Body**

```json
{
  "userName": "alice123",
  "password": "Abc1234!",
  "email":    "alice@example.com",
  "nickName": "Alice"
}
```

| 字段 | 必填 | 校验规则 |
|---|---|---|
| `userName` | ✅ | 4-20 位英文/数字/下划线 (`^[A-Za-z0-9_]{4,20}$`)，全局唯一 |
| `password` | ✅ | 8-12 位，必须含数字、大写字母、小写字母、特殊字符 (`!@#~$%^&*()+\|_`) |
| `email` | ❌ | 若填则需符合 `xxx@xxx.xxx` 格式，全局唯一 |
| `nickName` | ❌ | 默认取 `userName` |

**成功响应**

```json
{
  "code": 0,
  "msg": "注册成功",
  "data": {
    "id": 10001,
    "userName": "alice123",
    "email": "alice@example.com",
    "gender": 0,
    "nickName": "Alice",
    "avatar": "",
    "status": 0
  }
}
```

> 服务端不会自动登录，前端需要主动跳转登录页或紧接着调用登录接口。

**典型失败 `msg`**

- `用户名格式不合法 (4-20 位英文/数字/下划线)`
- `密码长度不符合规范, 必须为8-12位` / `密码必须包含数字` / `密码必须包含大写字母` 等
- `邮箱格式不正确`
- `用户名已被占用`
- `邮箱已被注册`

---

## 2. 登录 / 登出 / 修改密码 / 个人信息

### 2.1 `POST /user/login` —— 公共

兼容旧路径 `POST /login`，二者均可。

**Body**

```json
{ "userName": "alice123", "password": "Abc1234!" }
```

> `userName` 字段也支持传 email；后端用 `WHERE user_name = ? OR email = ?` 查找。

**响应**

- 成功：`code=0`，HTTP Header 增加 `new-token: <jwt>`
- 失败：`用户信息不存在 / 用户名或密码错误`

### 2.2 `GET /user/logout` —— 鉴权

清除服务端记录的 token；前端同时清除本地。

### 2.3 `POST /user/changePassword` —— 鉴权

```json
{ "password": "旧密码", "newPassword": "新密码符合规则" }
```

`newPassword` 同样跑 `ValidPwd` 校验。

### 2.4 `GET /user/info` —— 鉴权

返回当前登录用户的脱敏信息：

```json
{
  "code": 0,
  "data": {
    "id": 10001,
    "userName": "alice123",
    "email": "alice@example.com",
    "gender": 0,
    "nickName": "Alice",
    "avatar": "",
    "status": 0
  }
}
```

> 与后台接口 `GET /manage/user/info` 复用同一 controller，普通用户也可调用 `/user/info`。

---

## 3. 观看历史

服务端按 `(user_id, mid)` 唯一存储，重复观看同一影片只保留最近一条记录（通过 ON DUPLICATE KEY UPDATE 实现 upsert）。

### 3.1 `POST /user/history` —— 鉴权

上报或更新一条观看记录。前端建议在以下时机调用：

- 切换剧集 / 切换播放源时（首次进入也算）
- 播放过程中节流上报（建议 30 秒一次）
- 离开播放页 / 暂停超过阈值时

**Body**

```json
{
  "mid": 12345,
  "cid": 6,
  "pid": 1,
  "name": "影片名称",
  "picture": "https://.../poster.jpg",
  "playFrom": "siteA",
  "playFromName": "主线路",
  "episode": 2,
  "episodeName": "第3集",
  "progress": 480,
  "duration": 2700
}
```

| 字段 | 必填 | 含义 |
|---|---|---|
| `mid` | ✅ | 影片 ID (`SearchInfo.Mid`)，>0 |
| `name` | ✅ | 影片名 (空字符串拒绝) |
| `cid` / `pid` | ❌ | 二级 / 一级分类 ID，便于卡片直接展示 |
| `picture` | ❌ | 封面 URL |
| `playFrom` | ❌ | 站点 ID (`PlayLinkVo.Id`) |
| `playFromName` | ❌ | 站点名 |
| `episode` | ❌ | 集索引（0 起） |
| `episodeName` | ❌ | 集名（用于"上次看到 第3集"） |
| `progress` | ❌ | 已观看秒数 |
| `duration` | ❌ | 总时长（秒），可用作进度百分比展示 |

**成功响应**

```json
{ "code": 0, "msg": "已记录观看历史" }
```

### 3.2 `GET /user/history?current=1&pageSize=20` —— 鉴权

按更新时间倒序分页拉取：

```json
{
  "code": 0,
  "msg": "获取观看历史成功",
  "data": {
    "list": [
      {
        "id": 1,
        "userId": 10001,
        "mid": 12345,
        "cid": 6, "pid": 1,
        "name": "影片名称", "picture": "...",
        "playFrom": "siteA", "playFromName": "主线路",
        "episode": 2, "episodeName": "第3集",
        "progress": 480, "duration": 2700,
        "createdAt": "2026-05-09T22:11:00+08:00",
        "updatedAt": "2026-05-09T22:38:00+08:00"
      }
    ],
    "page": { "pageSize": 20, "current": 1, "pageCount": 1, "total": 1 }
  }
}
```

### 3.3 `DELETE /user/history?id=<historyId>` 或 `?mid=<filmMid>` —— 鉴权

二选一：

- `?id=` 删除单条记录（推荐）
- `?mid=` 按影片 ID 删（适合"我不再想保留这部影片的历史"）

两者都缺时返回 `缺少 id 或 mid`。

服务端内部带 `WHERE user_id = ?` 防越权，安全可信。

### 3.4 `DELETE /user/history/clear` —— 鉴权

清空当前用户全部历史。响应：

```json
{ "code": 0, "msg": "观看历史已清空" }
```

---

## 4. 收藏

按 `(user_id, mid)` 唯一存储，重复添加幂等。

### 4.1 `POST /user/favorite` —— 鉴权

**Body**

```json
{
  "mid": 12345,
  "cid": 6,
  "pid": 1,
  "name": "影片名称",
  "picture": "https://.../poster.jpg",
  "remarks": "更新至第8集"
}
```

| 字段 | 必填 | 含义 |
|---|---|---|
| `mid` | ✅ | 影片 ID |
| `name` | ✅ | 影片名 |
| 其它 | ❌ | 同上 |

重复添加同一 `mid` 不会报错（`ON CONFLICT DO NOTHING`），前端可以按"再点一次取消"或者"按钮按下后乐观切换状态"两种风格实现。

### 4.2 `DELETE /user/favorite?mid=<filmMid>` —— 鉴权

```json
{ "code": 0, "msg": "已取消收藏" }
```

### 4.3 `GET /user/favorite?current=1&pageSize=20` —— 鉴权

```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 1,
        "userId": 10001,
        "mid": 12345,
        "cid": 6, "pid": 1,
        "name": "影片名称", "picture": "...",
        "remarks": "更新至第8集",
        "createdAt": "2026-05-09T22:11:00+08:00",
        "updatedAt": "2026-05-09T22:11:00+08:00"
      }
    ],
    "page": { "pageSize": 20, "current": 1, "pageCount": 1, "total": 1 }
  }
}
```

### 4.4 `GET /user/favorite/check?mid=<filmMid>` —— 鉴权

进入详情页时调用，决定收藏按钮初始态：

```json
{ "code": 0, "msg": "ok", "data": { "favorited": true } }
```

> 缺少 `mid` 返回 `code=-1, msg="缺少 mid"`。

---

## 5. 典型前端流程

### 5.1 注册 → 登录 → 拉个人信息

```ts
// 1. 注册
await http.post('/user/register', {
  userName: 'alice123',
  password: 'Abc1234!',
  email: 'alice@example.com',
})

// 2. 登录
const loginResp = await http.post('/user/login', {
  userName: 'alice123',
  password: 'Abc1234!',
})
// 拦截器读取 response.headers['new-token'] 并存到 localStorage

// 3. 拉信息（需 auth-token）
const me = await http.get('/user/info')
```

### 5.2 详情页：判断收藏 + 上报观看进度

```ts
// 进入详情页
const { data: { favorited } } = await http.get('/user/favorite/check', { params: { mid } })

// 用户点击 "收藏"
await http.post('/user/favorite', { mid, name, picture, cid, pid })

// 用户播放过程中节流上报 (示例: 每 30s)
setInterval(() => {
  http.post('/user/history', {
    mid, cid, pid, name, picture,
    playFrom: currentSiteId,
    playFromName: currentSiteName,
    episode: currentEpisodeIndex,
    episodeName: currentEpisodeName,
    progress: Math.floor(player.currentTime),
    duration: Math.floor(player.duration),
  })
}, 30_000)
```

### 5.3 个人中心：历史 / 收藏列表

```ts
// 历史
const history = await http.get('/user/history', { params: { current: 1, pageSize: 20 } })

// 收藏
const fav = await http.get('/user/favorite', { params: { current: 1, pageSize: 20 } })
```

### 5.4 列表项跳转回播放页

历史接口返回的字段足以恢复播放上下文：

```ts
// 用户点击历史卡片
router.push({
  name: 'FilmPlay',
  query: {
    id: item.mid,
    playFrom: item.playFrom,
    episode: item.episode,
  },
})
```

进入播放页后可读取 `item.progress` 把播放器 seek 到对应位置。

---

## 6. 错误码速查

| 场景 | HTTP | code | msg 示例 |
|---|---|---|---|
| 缺少 `auth-token` | 401 | 0 | `用户未授权,请先登录` |
| token 解析失败 | 401 | 0 | `身份信息无效, 请重新登录` |
| token 在别处登录被踢 | 401 | 0 | `账号在其它设备登录,身份验证信息失效,请重新登录!!!` |
| 注册参数不合法 | 200 | -1 | `用户名格式不合法...` 等 |
| 上报历史缺 `mid`/`name` | 200 | -1 | `影片信息不完整` |
| 删历史无 `id`/`mid` | 200 | -1 | `缺少 id 或 mid` |
| 收藏检查无 `mid` | 200 | -1 | `缺少 mid` |
| 取消收藏无 `mid` | 200 | -1 | `缺少 mid` |

---

## 7. 数据库 (供参考)

新增两张表，启动时由 `TableInIt` 自动迁移：

```sql
-- user_histories: (user_id, mid) UNIQUE
CREATE TABLE user_histories (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  user_id INT UNSIGNED NOT NULL,
  mid BIGINT NOT NULL,
  cid BIGINT, pid BIGINT,
  name VARCHAR(255), picture VARCHAR(255),
  play_from VARCHAR(64), play_from_name VARCHAR(64),
  episode INT, episode_name VARCHAR(64),
  progress BIGINT, duration BIGINT,
  created_at DATETIME, updated_at DATETIME,
  UNIQUE KEY idx_user_mid_history (user_id, mid)
);

-- user_favorites: (user_id, mid) UNIQUE
CREATE TABLE user_favorites (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  user_id INT UNSIGNED NOT NULL,
  mid BIGINT NOT NULL,
  cid BIGINT, pid BIGINT,
  name VARCHAR(255), picture VARCHAR(255),
  remarks VARCHAR(64),
  created_at DATETIME, updated_at DATETIME,
  UNIQUE KEY idx_user_mid_fav (user_id, mid)
);
```

两张表都使用硬删除（不带 `deleted_at`），保证唯一约束在用户取消收藏 / 清空历史后立即释放。

---

## 8. FAQ

**Q: 注册后是否自动登录？**  
A: 否。后端只建账号，token 由独立的登录请求下发，便于前端保留"注册成功 → 登录页"或"注册成功 → 自动跳到首页"两种产品策略。

**Q: 同一影片连续上报观看进度，会不会产生大量历史记录？**  
A: 不会。`(user_id, mid)` 唯一约束 + ON DUPLICATE KEY UPDATE，永远只保留最新一条。

**Q: 用户切换设备/重登录后历史和收藏会丢吗？**  
A: 不会。两张表按 `user_id` 关联，不依赖客户端 token / 设备。

**Q: 如何避免收藏按钮重复点击带来的并发写？**  
A: 后端 `OnConflict DoNothing`，重复 INSERT 不会报错。前端可以乐观地 `setState(true)` 后再调接口。

**Q: 历史记录支持游客模式吗？**  
A: 不支持。所有 `/user/history` 和 `/user/favorite` 路由都挂在 `AuthToken` 中间件下。游客本地观看历史建议前端用 localStorage 维护，登录后选择性同步到服务端。
