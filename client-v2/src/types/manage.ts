/** 站点基础信息 */
export interface SiteBasic {
  siteName: string
  logo: string
  keyword: string
  description: string
  filing: string
  domain?: string
}

/** 采集源 */
export interface CollectSource {
  id: number
  name: string
  url: string
  /** json / xml */
  type: string
  resultModel: string
  state: boolean
  syncPictures?: boolean
}

/** 采集源 options（用于下拉） */
export interface CollectOption {
  id: number
  name: string
}

/** 定时任务 */
export interface CronTask {
  id: number
  name: string
  cron: string
  jobType: string
  state: boolean
  remark?: string
}

/** 影片分类 */
export interface FilmClass {
  id: number
  pid: number
  name: string
  ename?: string
  show: boolean
  sort?: number
  children?: FilmClass[]
}

/** 文件 */
export interface FileItem {
  id: string
  name: string
  url: string
  size: number
  type: string
  createdAt: number
}

/** 仪表盘统计 */
export interface DashboardStat {
  filmCount: number
  collectCount: number
  cronCount: number
  diskUsage?: { used: number; total: number }
}
