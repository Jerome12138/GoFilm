import { http } from '../http'
import type { PaginationResp } from '@/types/api'
import type { CollectOption, CollectSource } from '@/types/manage'
import type { ClassCoverItem } from '@/types/film'

/** GET /api/manage/collect/list 采集源列表（分页） */
export const list = (params?: {
  current?: number
  size?: number
  keyword?: string
}): Promise<PaginationResp<CollectSource>> =>
  http.get<unknown, PaginationResp<CollectSource>>('/manage/collect/list', { params })

/** GET /api/manage/collect/options 采集源 options（下拉用） */
export const options = (): Promise<CollectOption[]> =>
  http.get<unknown, CollectOption[]>('/manage/collect/options')

/** GET /api/manage/collect/find 采集源详情 */
export const find = (id: number): Promise<CollectSource> =>
  http.get<unknown, CollectSource>('/manage/collect/find', { params: { id } })

/** GET /api/manage/collect/del 删除采集源 */
export const remove = (id: number): Promise<void> =>
  http.get<unknown, void>('/manage/collect/del', { params: { id } })

/** POST /api/manage/collect/add 新增采集源 */
export const add = (data: CollectSource): Promise<void> =>
  http.post<unknown, void>('/manage/collect/add', data)

/** POST /api/manage/collect/update 修改采集源 */
export const update = (data: CollectSource): Promise<void> =>
  http.post<unknown, void>('/manage/collect/update', data)

/** POST /api/manage/collect/change 切换采集源状态 */
export const change = (data: { id: number; status: boolean }): Promise<void> =>
  http.post<unknown, void>('/manage/collect/change', data)

/** POST /api/manage/collect/test 测试采集源连通 */
export const test = (data: { url: string }): Promise<{ ok: boolean; msg: string }> =>
  http.post<unknown, { ok: boolean; msg: string }>('/manage/collect/test', data)

/** POST /api/manage/spider/start 启动爬虫 */
export const startSpider = (data: { sourceId: number; mode: string }): Promise<void> =>
  http.post<unknown, void>('/manage/spider/start', data)

/** GET /api/manage/spider/class/cover 爬虫分类封面 */
export const spiderClassCover = (): Promise<ClassCoverItem[]> =>
  http.get<unknown, ClassCoverItem[]>('/manage/spider/class/cover')
