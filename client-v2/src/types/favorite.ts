/**
 * 收藏 DTO
 *
 * 与后端契约:
 *   POST   /user/favorite          body: FavoriteAddPayload
 *   DELETE /user/favorite?mid=
 *   GET    /user/favorite?current&pageSize  -> FavoriteListResp
 *   GET    /user/favorite/check?mid=        -> { favorited: boolean }
 */

import type { BackendPage } from './film'

/** 后端 UserFavorite 序列化形态 */
export interface RemoteFavoriteItem {
  id: number
  userId?: number
  mid: number
  cid: number
  pid: number
  name: string
  picture: string
  remarks: string
  createdAt?: string
  updatedAt?: string
}

/** GET /user/favorite 响应 */
export interface FavoriteListResp {
  list: RemoteFavoriteItem[]
  page: BackendPage
}

/** POST /user/favorite 请求体 */
export interface FavoriteAddPayload {
  mid: number
  cid?: number
  pid?: number
  name: string
  picture?: string
  remarks?: string
}

/** GET /user/favorite/check 响应 */
export interface FavoriteCheckResp {
  favorited: boolean
}
