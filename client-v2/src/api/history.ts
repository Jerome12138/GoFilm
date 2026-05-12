import { http } from './http'
import type {
  HistoryListResp,
  HistoryUpsertPayload
} from '@/types/history'

/**
 * 观看历史 API (登录态下使用; 未登录态由 stores/history 走本地 cookie+LS 兜底)
 *
 * 所有接口都需要 auth-token, 未登录调用会被后端 401, 拦截器统一处理.
 * 这里 silent 配置交给调用方; 默认走全局 loading + toast.
 */

/** POST /api/user/history 上报/更新观看进度 */
export const upsert = (data: HistoryUpsertPayload): Promise<void> =>
  http.post<unknown, void>('/user/history', data)

/** GET /api/user/history 分页拉历史 */
export const list = (params?: {
  current?: number
  pageSize?: number
}): Promise<HistoryListResp> =>
  http.get<unknown, HistoryListResp>('/user/history', { params })

/**
 * DELETE /api/user/history 删除单条
 * 后端: 优先按 ?id= 删, 兜底按 ?mid= 删
 */
export const remove = (params: {
  id?: number | string
  mid?: number | string
}): Promise<void> =>
  http.delete<unknown, void>('/user/history', { params })

/** DELETE /api/user/history/clear 清空当前用户全部历史 */
export const clear = (): Promise<void> =>
  http.delete<unknown, void>('/user/history/clear')
