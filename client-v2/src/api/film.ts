import { http } from './http'
import type { PaginationResp } from '@/types/api'
import type {
  ClassifyData,
  ClassifySearchParams,
  FilmDetailResp,
  FilmListItem,
  IndexPageData,
  NavCategory,
  PlayInfo
} from '@/types/film'
import type { SiteBasic } from '@/types/manage'

/** GET /api/index 首页（轮播 / 分类 / 热门聚合） */
export const getIndex = (): Promise<IndexPageData> =>
  http.get<unknown, IndexPageData>('/index')

/** GET /api/navCategory 顶级分类导航 */
export const getNavCategory = (): Promise<NavCategory[]> =>
  http.get<unknown, NavCategory[]>('/navCategory')

/** GET /api/config/basic 站点基础信息 */
export const getSiteBasic = (): Promise<SiteBasic> =>
  http.get<unknown, SiteBasic>('/config/basic')

/** GET /api/filmDetail 影片详情（含相关推荐） */
export const getFilmDetail = (id: string): Promise<FilmDetailResp> =>
  http.get<unknown, FilmDetailResp>('/filmDetail', { params: { id } })

/** GET /api/filmPlayInfo 播放信息 */
export const getPlayInfo = (params: {
  id: string
  playFrom: string
  episode: string
}): Promise<PlayInfo> =>
  http.get<unknown, PlayInfo>('/filmPlayInfo', { params })

/** GET /api/filmClassify 分类首页（最新 / 排行 / 最近更新） */
export const getClassify = (Pid: number): Promise<ClassifyData> =>
  http.get<unknown, ClassifyData>('/filmClassify', { params: { Pid } })

/** GET /api/filmClassifySearch 分类筛选 */
export const searchClassify = (
  params: ClassifySearchParams
): Promise<PaginationResp<FilmListItem>> =>
  http.get<unknown, PaginationResp<FilmListItem>>('/filmClassifySearch', { params })

/** GET /api/searchFilm 关键字搜索 */
export const searchFilm = (params: {
  keyword: string
  current?: number
}): Promise<PaginationResp<FilmListItem>> =>
  http.get<unknown, PaginationResp<FilmListItem>>('/searchFilm', { params })
