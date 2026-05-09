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
  /** 豆瓣评分（部分接口/mock 返回，1-10） */
  dbScore?: string | number
  /** 通用评分别名 —— 后端历史字段：score / rating */
  score?: string | number
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

/** 播放页接口响应（对照后端 IndexController.FilmPlayInfo） */
export interface PlayInfo {
  detail: FilmDetail
  /** 当前集（episode + link） */
  current: PlayEpisode
  /** 当前播放源 ID（与 detail.list[i].id 对应） */
  currentPlayFrom: string
  /** 当前集索引（detail.list[i].linkList[idx]） */
  currentEpisode: number
  /** 相关推荐 */
  relate: FilmListItem[]
}

/** 分类标题（顶部分类导航上下文） */
export interface ClassifyTitle {
  id: number
  pid?: number
  name: string
  show?: boolean
}

/** 分类首页（最新 / 排行 / 最近更新） — 后端字段：title + content.{news, top, recent} */
export interface ClassifyData {
  title: ClassifyTitle
  content: {
    news: FilmListItem[]
    top: FilmListItem[]
    recent: FilmListItem[]
  }
}

/** 分类筛选 query —— 严格保持后端首字母大写 */
export interface ClassifySearchParams {
  Pid: number | string
  Category?: number | string | ''
  Plot?: string
  Area?: string
  Language?: string
  Year?: string | number
  Sort?: string
  current?: number
}

/** 后端分页对象（与 system.Page 保持一致） */
export interface BackendPage {
  pageSize: number
  current: number
  pageCount: number
  total: number
}

/** 搜索 / 筛选用列表项 tag（Name/Value） */
export interface ClassifyTagItem {
  Name: string
  Value: string | number
}

/** 分类筛选响应（后端 FilmTagSearch） */
export interface ClassifySearchResp {
  title: ClassifyTitle
  list: FilmListItem[]
  page: BackendPage
  search: {
    sortList: string[]
    titles: Record<string, string>
    tags: Record<string, ClassifyTagItem[]>
  }
  params: {
    Pid: string
    Category: string
    Plot: string
    Area: string
    Language: string
    Year: string
    Sort: string
  }
}

/** 关键字搜索响应（后端 SearchFilm） */
export interface SearchFilmResp {
  list: FilmListItem[]
  page: BackendPage
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

/** 后台影片搜索参数（对齐后端 FilmSearchPage Query 名） */
export interface ManageFilmSearchParams {
  current?: number
  pageSize?: number
  /** 后端字段名 name（影片名关键字） */
  name?: string
  /** 一级分类 ID */
  pid?: number
  /** 二级分类 ID */
  cid?: number
  plot?: string
  area?: string
  language?: string
  year?: number
  remarks?: string
}

/** 后台影片搜索响应（嵌套 paging） */
export interface ManageFilmSearchResp {
  list: FilmListItem[]
  options?: unknown
  params: {
    paging: BackendPage
    [k: string]: unknown
  }
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
  enName?: string
  subTitle?: string
  initial?: string
  pid: number
  cid: number
  picture?: string
  area?: string
  year?: string
  director?: string
  actor?: string
  classTag?: string
  content?: string
  remarks?: string
  state?: string
  playFrom?: string
  downFrom?: string
  playLink?: string
  downloadLink?: string
  list?: PlaySource[]
}
