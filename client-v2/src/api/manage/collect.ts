import { http } from '../http'
import type { CollectOption, CollectParams, CollectSource } from '@/types/manage'

/** GET /api/manage/collect/list 采集源列表（后端返回 FilmSource 数组） */
export const list = (): Promise<CollectSource[]> =>
  http.get<unknown, CollectSource[]>('/manage/collect/list')

/** GET /api/manage/collect/options 采集源 options（下拉用） */
export const options = (): Promise<CollectOption[]> =>
  http.get<unknown, CollectOption[]>('/manage/collect/options')

/** GET /api/manage/collect/find 采集源详情（id: string） */
export const find = (id: string): Promise<CollectSource> =>
  http.get<unknown, CollectSource>('/manage/collect/find', { params: { id } })

/** GET /api/manage/collect/del 删除采集源（id: string） */
export const remove = (id: string): Promise<void> =>
  http.get<unknown, void>('/manage/collect/del', { params: { id } })

/** POST /api/manage/collect/add 新增采集源 */
export const add = (data: CollectSource): Promise<void> =>
  http.post<unknown, void>('/manage/collect/add', data)

/** POST /api/manage/collect/update 修改采集源 */
export const update = (data: CollectSource): Promise<void> =>
  http.post<unknown, void>('/manage/collect/update', data)

/**
 * POST /api/manage/collect/change 切换采集源状态 / 同步图片配置
 * 后端期望完整 FilmSource，至少 id/state/syncPictures
 */
export const change = (data: CollectSource): Promise<void> =>
  http.post<unknown, void>('/manage/collect/change', data)

/**
 * POST /api/manage/collect/test 测试采集源连通（后端期望完整 FilmSource）
 * 成功无 body 数据；失败由拦截器抛 BizError，调用方用 try/catch 区分成功/失败
 */
export const test = (data: CollectSource): Promise<void> =>
  http.post<unknown, void>('/manage/collect/test', data)

/** POST /api/manage/spider/start 启动爬虫（CollectParams） */
export const startSpider = (data: CollectParams): Promise<void> =>
  http.post<unknown, void>('/manage/spider/start', data)

/**
 * GET /api/manage/spider/class/cover 重置/覆盖影片分类（动作类接口）
 * 后端实际行为是触发分类采集并覆盖本地分类数据，不返回数据；
 * 成功后建议调用方刷新分类列表 / nav store。
 */
export const spiderClassCover = (): Promise<void> =>
  http.get<unknown, void>('/manage/spider/class/cover')
