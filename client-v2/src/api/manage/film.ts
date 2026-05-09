import { http } from '../http'
import type {
  FilmAddPayload,
  ManageFilmSearchParams,
  ManageFilmSearchResp
} from '@/types/film'
import type { FilmClass } from '@/types/manage'

/**
 * GET /api/manage/film/search/list 后台影片搜索（分页）
 * 后端响应嵌套 `params.paging`，前端按 ManageFilmSearchResp 接收
 */
export const searchList = (
  params: ManageFilmSearchParams
): Promise<ManageFilmSearchResp> =>
  http.get<unknown, ManageFilmSearchResp>('/manage/film/search/list', { params })

/** GET /api/manage/film/class/tree 分类树 */
export const classTree = (): Promise<FilmClass[]> =>
  http.get<unknown, FilmClass[]>('/manage/film/class/tree')

/** GET /api/manage/film/class/find 分类详情 */
export const classFind = (id: number): Promise<FilmClass> =>
  http.get<unknown, FilmClass>('/manage/film/class/find', { params: { id } })

/** GET /api/manage/film/class/del 删除分类 */
export const classDel = (id: number): Promise<void> =>
  http.get<unknown, void>('/manage/film/class/del', { params: { id } })

/** POST /api/manage/film/class/update 新增 / 修改分类 */
export const classUpdate = (data: FilmClass): Promise<void> =>
  http.post<unknown, void>('/manage/film/class/update', data)

/** POST /api/manage/film/add 新增影片 */
export const add = (data: FilmAddPayload): Promise<void> =>
  http.post<unknown, void>('/manage/film/add', data)
