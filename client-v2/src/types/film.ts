/**
 * 影视相关 DTO（来自架构文档第 8 节，对照后端字段）
 */

/** 顶级 / 子级分类导航 */
export interface NavCategory {
  id: number
  pid: number
  name: string
  show: boolean
  children?: NavCategory[]
}

/** 列表项（首页 Row / 搜索 / 分类筛选） */
export interface FilmListItem {
  id: string | number
  /** 旧站 hot list 字段 */
  mid?: string
  name: string
  picture: string
  remarks?: string
  year?: string
  area?: string
  /** 分类名（部分接口返回） */
  cName?: string
  cid?: number
  pid?: number
}

/** 详情描述区 */
export interface FilmDescriptor {
  cName: string
  /** 逗号分隔的剧情/题材标签 */
  classTag?: string
  director?: string
  actor?: string
  releaseDate?: string
  area?: string
  year?: string
  remarks?: string
  content?: string
  dbScore?: string | number
}

/** 单集 */
export interface PlayEpisode {
  episode: string
  /** 唯一标识 / m3u8 链接 */
  link: string
}

/** 一个播放源（站内可能多源） */
export interface PlaySource {
  /** sourceId */
  id: string
  /** 播放源名 */
  name: string
  linkList: PlayEpisode[]
}

/** 详情主对象 */
export interface FilmDetail extends FilmListItem {
  descriptor: FilmDescriptor
  list: PlaySource[]
}

/** 详情接口响应 */
export interface FilmDetailResp {
  detail: FilmDetail
  relate: FilmListItem[]
}

/** 播放页接口响应 */
export interface PlayInfo {
  /** m3u8 / mp4 真实链接 */
  src: string
  /** 类型，例如 application/x-mpegURL */
  type?: string
  episode: string
  link: string
  prev?: { episode: string; link: string } | null
  next?: { episode: string; link: string } | null
}

/** 分类首页（最新 / 排行 / 最近更新） */
export interface ClassifyData {
  pid: number
  category: NavCategory
  newest: FilmListItem[]
  ranking: FilmListItem[]
  recent: FilmListItem[]
}

/** 分类筛选 query */
export interface ClassifySearchParams {
  Pid: number
  Category?: number | ''
  Plot?: string
  Area?: string
  Language?: string
  Year?: string
  Sort?: string
  current?: number
}

/** 首页聚合 */
export interface IndexPageData {
  banner: FilmListItem[]
  content: Array<{
    nav: NavCategory
    movies: FilmListItem[]
    hot: FilmListItem[]
  }>
}

/** 后台影片搜索参数 */
export interface ManageFilmSearchParams {
  current?: number
  size?: number
  keyword?: string
  classId?: number
  pid?: number
  state?: boolean
}

/** 爬虫分类封面项 */
export interface ClassCoverItem {
  id: number
  name: string
  cover: string
}

/** 后台影片新增 payload */
export interface FilmAddPayload {
  name: string
  pid: number
  cid: number
  picture?: string
  area?: string
  year?: string
  director?: string
  actor?: string
  content?: string
  remarks?: string
  list?: PlaySource[]
}
