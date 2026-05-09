import { http } from '../http'
import type { CronTask } from '@/types/manage'

/** GET /api/manage/cron/list 定时任务列表 */
export const list = (): Promise<CronTask[]> =>
  http.get<unknown, CronTask[]>('/manage/cron/list')

/** GET /api/manage/cron/find 定时任务详情 */
export const find = (id: number): Promise<CronTask> =>
  http.get<unknown, CronTask>('/manage/cron/find', { params: { id } })

/** GET /api/manage/cron/del 删除定时任务 */
export const remove = (id: number): Promise<void> =>
  http.get<unknown, void>('/manage/cron/del', { params: { id } })

/** POST /api/manage/cron/add 新增定时任务 */
export const add = (data: CronTask): Promise<void> =>
  http.post<unknown, void>('/manage/cron/add', data)

/** POST /api/manage/cron/update 修改定时任务 */
export const update = (data: CronTask): Promise<void> =>
  http.post<unknown, void>('/manage/cron/update', data)

/** POST /api/manage/cron/change 切换状态 */
export const change = (data: { id: number; status: boolean }): Promise<void> =>
  http.post<unknown, void>('/manage/cron/change', data)
