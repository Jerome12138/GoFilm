/** 站点基础信息 */
export interface SiteBasic {
  siteName: string
  logo: string
  keyword: string
  description: string
  filing: string
  domain?: string
  /** 后端 BasicConfig 可能还有以下字段（联调时校对） */
  record?: string
  copyright?: string
  security?: string
}

/** 接口返回类型：0=JSON 1=XML（与后端 CollectResultModel 对齐） */
export type CollectResultModel = 0 | 1

/** 采集站等级：0=主站 1=附属（与后端 SourceGrade 对齐） */
export type SourceGrade = 0 | 1

/** 采集资源类型：0=视频 1=文章 2=演员 3=角色 4=网站（与后端 ResourceType 对齐） */
export type CollectType = 0 | 1 | 2 | 3 | 4

/** 采集源（对齐后端 FilmSource） */
export interface CollectSource {
  /** 唯一 ID（后端字符串 uuid） */
  id: string
  /** 采集站点备注名 */
  name: string
  /** 采集链接（后端字段名 uri） */
  uri: string
  /** 接口返回类型 0=json 1=xml */
  resultModel: CollectResultModel
  /** 站点等级 0=主站 1=附属 */
  grade: SourceGrade
  /** 是否同步图片 */
  syncPictures: boolean
  /** 采集资源类型 0..4 */
  collectType: CollectType
  /** 是否启用 */
  state: boolean
  /** 采集时间间隔 单位 ms */
  interval: number
}

/** 采集源 options（用于下拉） */
export interface CollectOption {
  id: string
  name: string
}

/** 采集执行参数（对齐后端 CollectParams） */
export interface CollectParams {
  /** 资源站 id（单个） */
  id: string
  /** 资源站 id 列表（批量时） */
  ids: string[]
  /** 采集时长（小时） */
  time: number
  /** 是否批量 */
  batch: boolean
}

/** 定时任务（对齐后端 FilmCollectTask） */
export interface CronTask {
  /** 唯一 uid */
  id: string
  /** 关联的采集站 id 列表 */
  ids: string[]
  /** 采集时长（小时） */
  time: number
  /** cron 表达式 */
  spec: string
  /** 任务类型 0=自动更新已启用站点 1=更新 ids */
  model: 0 | 1
  /** 启用 */
  state: boolean
  /** 备注 */
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

/** 文件项（FileGallery 列表项） */
export interface FileItem {
  id: number | string
  name: string
  url: string
  size?: number
  type?: string
  createdAt?: number
}

/** 后端通用分页 */
export interface BackendPage {
  total: number
  current: number
  pageSize: number
  pageCount?: number
}

/** PhotoWall 响应：{ list, page } */
export interface PhotoWallResp {
  list: FileItem[]
  page: BackendPage
}

/** 仪表盘统计（PRD Q3 后端待实现，前端兜底） */
export interface DashboardStat {
  filmCount?: number
  collectCount?: number
  cronCount?: number
  diskUsage?: { used: number; total: number }
}
