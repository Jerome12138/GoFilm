import { http } from './http'
import type {
  FavoriteAddPayload,
  FavoriteCheckResp,
  FavoriteListResp
} from '@/types/favorite'

/**
 * 收藏 API (登录态下使用)
 */

/** POST /api/user/favorite 添加收藏 (后端幂等) */
export const add = (data: FavoriteAddPayload): Promise<void> =>
  http.post<unknown, void>('/user/favorite', data)

/** DELETE /api/user/favorite?mid= 取消收藏 */
export const remove = (mid: number | string): Promise<void> =>
  http.delete<unknown, void>('/user/favorite', { params: { mid } })

/** GET /api/user/favorite 分页拉收藏 */
export const list = (params?: {
  current?: number
  pageSize?: number
}): Promise<FavoriteListResp> =>
  http.get<unknown, FavoriteListResp>('/user/favorite', { params })

/** GET /api/user/favorite/check?mid= 查询是否已收藏 */
export const check = (mid: number | string): Promise<FavoriteCheckResp> =>
  http.get<unknown, FavoriteCheckResp>('/user/favorite/check', { params: { mid } })
