import { http } from '../http'
import type { DashboardStat, SiteBasic } from '@/types/manage'

/** GET /api/manage/index 仪表盘统计 */
export const dashboard = (): Promise<DashboardStat> =>
  http.get<unknown, DashboardStat>('/manage/index')

/** GET /api/manage/config/basic 后台读取站点基础配置 */
export const getBasic = (): Promise<SiteBasic> =>
  http.get<unknown, SiteBasic>('/manage/config/basic')

/** POST /api/manage/config/basic/update 更新站点基础配置 */
export const updateBasic = (data: SiteBasic): Promise<void> =>
  http.post<unknown, void>('/manage/config/basic/update', data)
