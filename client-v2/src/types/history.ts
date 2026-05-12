/**
 * 观看历史 DTO
 *
 * 与后端契约:
 *   POST /user/history          body: HistoryUpsertPayload
 *   GET  /user/history?current&pageSize    -> HistoryListResp
 *   DELETE /user/history?id= 或 ?mid=
 *   DELETE /user/history/clear
 *
 * RemoteHistoryItem 字段对应 server/model/system/UserHistory (json 标签).
 */

import type { BackendPage } from './film'

/** 后端 UserHistory 序列化形态 */
export interface RemoteHistoryItem {
  id: number
  userId?: number
  mid: number
  cid: number
  pid: number
  name: string
  picture: string
  playFrom: string
  playFromName: string
  episode: number
  episodeName: string
  progress: number
  duration: number
  createdAt?: string
  updatedAt?: string
}

/** GET /user/history 响应 */
export interface HistoryListResp {
  list: RemoteHistoryItem[]
  page: BackendPage
}

/**
 * POST /user/history 请求体
 * 字段名与后端 logic.HistoryUpsertParams (json 标签) 严格一致.
 * mid + name 是后端必填校验项, 其它字段都可缺省.
 */
export interface HistoryUpsertPayload {
  mid: number
  cid?: number
  pid?: number
  name: string
  picture?: string
  playFrom?: string
  playFromName?: string
  episode?: number
  episodeName?: string
  progress?: number
  duration?: number
}
