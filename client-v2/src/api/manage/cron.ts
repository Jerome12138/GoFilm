import { http } from '../http'
import type { CronTask } from '@/types/manage'

/** GET /api/manage/cron/list 定时任务列表 */
export const list = (): Promise<CronTask[]> =>
  http.get<unknown, CronTask[]>('/manage/cron/list')

/** GET /api/manage/cron/find 定时任务详情（id: string） */
export const find = (id: string): Promise<CronTask> =>
  http.get<unknown, CronTask>('/manage/cron/find', { params: { id } })

/** GET /api/manage/cron/del 删除定时任务（id: string） */
export const remove = (id: string): Promise<void> =>
  http.get<unknown, void>('/manage/cron/del', { params: { id } })

/** POST /api/manage/cron/add 新增定时任务 */
export const add = (data: CronTask): Promise<void> =>
  http.post<unknown, void>('/manage/cron/add', data)

/** POST /api/manage/cron/update 修改定时任务 */
export const update = (data: CronTask): Promise<void> =>
  http.post<unknown, void>('/manage/cron/update', data)

/**
 * POST /api/manage/cron/change 切换状态
 * 后端期望完整 FilmCollectTask 对象（至少 id/state）
 */
export const change = (data: CronTask): Promise<void> =>
  http.post<unknown, void>('/manage/cron/change', data)
