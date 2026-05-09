#!/usr/bin/env bash
###############################################################################
# GoFilm Vue3 重构 - QA 冒烟回归 curl 脚本
# 测试日期: 2026-05-09
# 使用方式:
#   1. 启动后端 (默认 http://127.0.0.1:3601 → 经 vite 代理 /api/* 转发)
#      或直接打到后端: BASE=http://127.0.0.1:3601 bash test-scripts.sh
#      或经 vite dev: BASE=http://127.0.0.1:3600/api bash test-scripts.sh
#   2. 登录获取 token: 修改下方 USERNAME/PASSWORD 后运行 login_and_save_token
#
# 注意：以下接口的 payload / query 命名严格对照后端 Go controller
###############################################################################

BASE="${BASE:-http://127.0.0.1:3601}"
USERNAME="${USERNAME:-admin}"
PASSWORD="${PASSWORD:-admin123}"
TOKEN=""

curl_get() {
  curl -s --noproxy "*" -w "\n[HTTP %{http_code}]\n" "$@"
}
curl_post_json() {
  curl -s --noproxy "*" -w "\n[HTTP %{http_code}]\n" -H "Content-Type: application/json" "$@"
}

###############################################################################
# 1. 用户端公开接口（无需 token）
###############################################################################
public_smoke() {
  echo "=== /index ==="
  curl_get "$BASE/index"
  echo "=== /navCategory ==="
  curl_get "$BASE/navCategory"
  echo "=== /config/basic ==="
  curl_get "$BASE/config/basic"
  echo "=== /filmDetail?id=<id> ==="
  curl_get "$BASE/filmDetail?id=1"
  echo "=== /filmPlayInfo?id=&playFrom=&episode= ==="
  curl_get "$BASE/filmPlayInfo?id=1&playFrom=qiyi&episode=0"
  echo "=== /searchFilm?keyword=战&current=1 ==="
  curl_get "$BASE/searchFilm?keyword=%E6%88%98&current=1"
  echo "=== /filmClassify?Pid=1 ==="
  curl_get "$BASE/filmClassify?Pid=1"
  echo "=== /filmClassifySearch?Pid=1&Sort=update_stamp&current=1 ==="
  curl_get "$BASE/filmClassifySearch?Pid=1&Sort=update_stamp&current=1"
}

###############################################################################
# 2. 异常 / 边界
###############################################################################
edge_cases() {
  echo "=== /filmDetail 缺少 id ==="
  curl_get "$BASE/filmDetail"
  echo "=== /filmDetail 错误 id ==="
  curl_get "$BASE/filmDetail?id=__not_exist__"
  echo "=== /searchFilm 空关键字 ==="
  curl_get "$BASE/searchFilm?keyword=&current=1"
  echo "=== /filmClassify 缺 Pid ==="
  curl_get "$BASE/filmClassify"
  echo "=== /filmClassifySearch 全部空 ==="
  curl_get "$BASE/filmClassifySearch"
}

###############################################################################
# 3. 鉴权
###############################################################################
login_and_save_token() {
  echo "=== /login 登录（注意后端 json tag 是 userName）==="
  # 旧站发 { userName, password }，client-v2 发 { username, password }
  # Go json.Unmarshal 大小写不敏感，两者均能工作；下面同时写两份兼容
  RESP=$(curl -s -i --noproxy "*" -H "Content-Type: application/json" \
    -d "{\"userName\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" \
    "$BASE/login")
  echo "$RESP"
  # 提取 new-token 头
  TOKEN=$(echo "$RESP" | tr -d '\r' | awk '/^new-token:/{print $2}')
  echo "TOKEN=$TOKEN"
}

# ⚠️ 已知 BUG：client-v2 发送 { oldPwd, newPwd }，后端要 { password, newPassword }
# 下面用后端兼容字段做正向回归
change_password_compat() {
  echo "=== /changePassword（后端要求 password / newPassword）==="
  curl_post_json -H "auth-token: $TOKEN" \
    -d '{"password":"oldPwd","newPassword":"NewPwd@123"}' \
    "$BASE/changePassword"
}

# 验证 client-v2 当前发送格式（应失败）
change_password_clientv2_format() {
  echo "=== /changePassword（client-v2 发送格式 oldPwd/newPwd, 预期失败）==="
  curl_post_json -H "auth-token: $TOKEN" \
    -d '{"oldPwd":"old","newPwd":"NewPwd@123"}' \
    "$BASE/changePassword"
}

###############################################################################
# 4. 后台管理接口（需 token）
###############################################################################
manage_smoke() {
  H="auth-token: $TOKEN"
  echo "=== /manage/index ==="
  curl_get -H "$H" "$BASE/manage/index"
  echo "=== /manage/config/basic ==="
  curl_get -H "$H" "$BASE/manage/config/basic"
  echo "=== /manage/user/info ==="
  curl_get -H "$H" "$BASE/manage/user/info"

  echo "=== /manage/collect/list ==="
  curl_get -H "$H" "$BASE/manage/collect/list"
  echo "=== /manage/collect/options ==="
  curl_get -H "$H" "$BASE/manage/collect/options"

  echo "=== /manage/cron/list ==="
  curl_get -H "$H" "$BASE/manage/cron/list"

  echo "=== /manage/film/search/list ==="
  # ⚠️ 注意 query 是 name 不是 keyword（client-v2 传 keyword 会被忽略）
  curl_get -H "$H" "$BASE/manage/film/search/list?name=&pid=0&cid=0&current=1&pageSize=10"

  echo "=== /manage/film/class/tree ==="
  curl_get -H "$H" "$BASE/manage/film/class/tree"

  echo "=== /manage/file/list?current=1 ==="
  curl_get -H "$H" "$BASE/manage/file/list?current=1"

  echo "=== /manage/spider/class/cover ==="
  curl_get -H "$H" "$BASE/manage/spider/class/cover"
}

# ⚠️ 已知 BUG：collect/change 后端期望 FilmSource 完整结构 + state + syncPictures，client-v2 发 { id, status }
collect_change_compat() {
  echo "=== /manage/collect/change（后端期望 id/state/syncPictures）==="
  curl_post_json -H "auth-token: $TOKEN" \
    -d '{"id":"<resourceId>","state":true,"syncPictures":false}' \
    "$BASE/manage/collect/change"
}

# ⚠️ 已知 BUG：collect/test 后端期望 FilmSource 对象，client-v2 发 { url }
collect_test_compat() {
  echo "=== /manage/collect/test（后端期望 FilmSource 完整对象）==="
  curl_post_json -H "auth-token: $TOKEN" \
    -d '{"name":"测试","uri":"https://api.x/json","resultModel":0,"collectType":0}' \
    "$BASE/manage/collect/test"
}

# ⚠️ 已知 BUG：spider/start 后端期望 CollectParams { id, ids, time, batch }，client-v2 发 { sourceId, mode }
spider_start_compat() {
  echo "=== /manage/spider/start（后端期望 id/ids/time/batch）==="
  curl_post_json -H "auth-token: $TOKEN" \
    -d '{"id":"<resourceId>","ids":[],"time":24,"batch":false}' \
    "$BASE/manage/spider/start"
}

###############################################################################
# 主入口
###############################################################################
main() {
  echo "===== A. 用户端公开接口冒烟 ====="
  public_smoke
  echo "===== B. 异常 / 边界 ====="
  edge_cases
  echo "===== C. 登录拿 token ====="
  login_and_save_token
  if [ -z "$TOKEN" ]; then
    echo "登录失败，跳过后台测试。"
    exit 1
  fi
  echo "===== D. 后台管理接口冒烟 ====="
  manage_smoke
  echo "===== E. BUG 复现脚本（需手工逐条执行）====="
  echo "  change_password_compat                # 后端兼容格式应成功"
  echo "  change_password_clientv2_format       # client-v2 当前格式应失败"
  echo "  collect_change_compat                 # 后端兼容格式应成功"
  echo "  collect_test_compat                   # 后端兼容格式应成功"
  echo "  spider_start_compat                   # 后端兼容格式应成功"
}

# 不带参数时直接跑 main；带参数则尝试调用对应函数
if [ "$#" -eq 0 ]; then
  main
else
  "$@"
fi
