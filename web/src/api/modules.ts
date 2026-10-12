import http from './client'

// ============ 认证 ============
export interface LoginParams {
  username: string
  password: string
  turnstile_token?: string
}

export interface LoginResp {
  access_token: string
  refresh_token: string
  expires_at: string
  user: User
}

export const authApi = {
  login: (data: LoginParams) => http.post<any, any>('/auth/login', data),
  refresh: (refreshToken: string) => http.post<any, any>('/auth/refresh', { refresh_token: refreshToken }),
  me: () => http.get<any, any>('/auth/me'),
  logout: () => http.post<any, any>('/auth/logout')
}

// ============ 用户 ============
export type Role = 'admin' | 'a_manager' | 'a_lead' | 'a_staff' | 'b_manager' | 'b_lead' | 'b_staff'
export type UserStatus = 'enabled' | 'disabled'

export interface User {
  id: number
  username: string
  real_name: string
  role: Role
  dept_id: number | null
  group_id: number | null
  superior_id: number | null        // 归属 / 上级(指向另一用户)
  status: UserStatus
  last_login_at: string | null
  created_at: string
  dept?: Department
  superior?: Pick<User, 'id' | 'username' | 'real_name' | 'role'>
}

export const userApi = {
  list: (params?: any) => http.get<any, any>('/users', { params }),
  create: (data: any) => http.post<any, any>('/users', data),
  get: (id: number) => http.get<any, any>(`/users/${id}`),
  update: (id: number, data: any) => http.put<any, any>(`/users/${id}`, data),
  delete: (id: number) => http.delete<any, any>(`/users/${id}`),
  resetPassword: (id: number, new_password: string) => http.post<any, any>(`/users/${id}/reset-password`, { new_password }),
  // 所有启用用户的精简列表(供上级筛选 / 新建编辑时下拉用,不分页)
  options: () => http.get<any, any>('/users/options')
}

// ============ 部门 ============
export type DeptType = 'company' | 'dept_a' | 'dept_b' | 'group'

export interface Department {
  id: number
  name: string
  parent_id: number | null
  type: DeptType
  children?: Department[]
}

export const deptApi = {
  list: () => http.get<any, any>('/departments'),
  create: (data: any) => http.post<any, any>('/departments', data),
  get: (id: number) => http.get<any, any>(`/departments/${id}`),
  update: (id: number, data: any) => http.put<any, any>(`/departments/${id}`, data),
  delete: (id: number) => http.delete<any, any>(`/departments/${id}`)
}

// ============ 游戏 ============
export type GameStatus = 'enabled' | 'disabled'
export type PartnerType = '' | 'api' | 'webhook'

export interface Game {
  id: number
  code: string
  name: string
  company: string
  status: GameStatus
  remark: string
  partner_type: PartnerType
  partner_config: Record<string, any>
  partner_enabled: boolean
}

export const gameApi = {
  list: (params?: any) => http.get<any, any>('/games', { params }),
  create: (data: any) => http.post<any, any>('/games', data),
  get: (id: number) => http.get<any, any>(`/games/${id}`),
  update: (id: number, data: any) => http.put<any, any>(`/games/${id}`, data),
  delete: (id: number) => http.delete<any, any>(`/games/${id}`)
}

// ============ 字典 ============
export type DictCategory = 'channel' | 'country' | 'community' | 'region' | 'follow_type'
export type DictStatus = 'enabled' | 'disabled'

export interface Dict {
  id: number
  category: DictCategory
  code: string
  label: string
  sort: number
  status: DictStatus
  remark: string
}

export const dictApi = {
  categories: () => http.get<any, any>('/dicts/categories'),
  list: (params?: any) => http.get<any, any>('/dicts', { params }),
  create: (data: any) => http.post<any, any>('/dicts', data),
  get: (id: number) => http.get<any, any>(`/dicts/${id}`),
  update: (id: number, data: any) => http.put<any, any>(`/dicts/${id}`, data),
  delete: (id: number) => http.delete<any, any>(`/dicts/${id}`)
}

// ============ 广播模板 ============
export interface BroadcastTemplate {
  id: number
  name: string
  template: string
  enabled: boolean
  weight: number
  remark: string
}

export const broadcastApi = {
  list: () => http.get<any, any>('/broadcast-templates'),
  create: (data: any) => http.post<any, any>('/broadcast-templates', data),
  get: (id: number) => http.get<any, any>(`/broadcast-templates/${id}`),
  update: (id: number, data: any) => http.put<any, any>(`/broadcast-templates/${id}`, data),
  delete: (id: number) => http.delete<any, any>(`/broadcast-templates/${id}`),
  render: (data: any) => http.post<any, any>('/broadcast-templates/render', data)
}

// ============ 线索 ============
export type LeadStatus = 'pending' | 'claimed' | 'archived'
export type IsTarget = 'target' | 'not_target'

export interface Lead {
  id: number
  game_id: number
  creator_id: number
  channel: string
  player_username: string
  player_game_name: string
  player_id: string
  region: string
  player_country: string
  player_age: number
  is_target: IsTarget
  community: string
  token_code: string
  screenshots: string[]
  status: LeadStatus
  claimed_by: number | null
  claimed_at: string | null
  released_by: number | null
  released_at: string | null
  extension: Record<string, any>
  remark: string
  created_at: string
  game?: Game
  creator?: User
  claimed_by_user?: User
}

export const leadApi = {
  pool: (params?: any) => http.get<any, any>('/leads/pool', { params }),
  list: (params?: any) => http.get<any, any>('/leads', { params }),
  create: (data: any) => http.post<any, any>('/leads', data),
  get: (id: number) => http.get<any, any>(`/leads/${id}`),
  update: (id: number, data: any) => http.put<any, any>(`/leads/${id}`, data),
  delete: (id: number) => http.delete<any, any>(`/leads/${id}`),
  claim: (id: number) => http.post<any, any>(`/leads/${id}/claim`),
  release: (id: number) => http.post<any, any>(`/leads/${id}/release`),
  archive: (id: number) => http.post<any, any>(`/leads/${id}/archive`)
}

// ============ 文件上传 ============
export const uploadApi = {
  upload: (file: File, subDir = 'screenshots') => {
    const form = new FormData()
    form.append('files', file)
    form.append('sub_dir', subDir)
    return http.post<any, any>('/upload', form, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  }
}

// ============ P7：订单 / 上传模板 / 合作方 ============

export type OrderStatus = 'paid' | 'refunded' | 'pending' | 'cancelled'
export type OrderSource = 'manual' | 'api' | 'webhook'
export type AttributionStatus = 'auto_attributed' | 'manual_attributed' | 'pending_attribution'

export interface Order {
  id: number
  lead_id: number | null
  game_id: number
  player_id: string
  order_no: string
  amount: number
  currency: string
  status: OrderStatus
  order_time: string
  paid_at: string | null
  source: OrderSource
  partner_id: number | null
  attribution_status: AttributionStatus
  attributed_at: string | null
  attributed_by: number | null
  matched_lead_count: number
  is_first_charge: boolean
  remark: string
  raw_payload: Record<string, any>
  import_batch: string
  imported_by: number | null
  created_at: string
  game?: Game
  lead?: Lead
  partner?: Partner
  importer?: User
  attributor?: User
}

export interface OrderStats {
  total_revenue: number
  total_orders: number
  paid_players: number
  arpu: number
  pending_orders: number
  a_team_revenue: number
  a_team_orders: number
  b_team_revenue: number
  b_team_orders: number
}

export const orderApi = {
  list: (params?: any) => http.get<any, any>('/orders', { params }),
  get: (id: number) => http.get<any, any>(`/orders/${id}`),
  stats: (params?: any) => http.get<any, any>('/orders/stats', { params }),
  bind: (id: number, lead_id: number) => http.patch<any, any>(`/orders/${id}/bind`, { lead_id }),
  delete: (id: number) => http.delete<any, any>(`/orders/${id}`),
  import: (gameId: number, templateId: number, file: File) => {
    const form = new FormData()
    form.append('game_id', String(gameId))
    form.append('template_id', String(templateId))
    form.append('file', file)
    return http.post<any, any>('/orders/import', form, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },
  importPreview: (gameId: number, templateId: number, file: File) => {
    const form = new FormData()
    form.append('game_id', String(gameId))
    form.append('template_id', String(templateId))
    form.append('file', file)
    return http.post<any, any>('/orders/import/preview', form, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  }
}

export interface UploadTemplate {
  id: number
  name: string
  game_id: number
  column_mappings: Record<string, string>
  time_format: string
  status_mapping: Record<string, string>
  currency: string
  sample_file_url: string
  remark: string
  created_by: number | null
  game?: Game
  creator?: User
}

export const uploadTemplateApi = {
  list: (params?: any) => http.get<any, any>('/upload-templates', { params }),
  create: (data: any) => http.post<any, any>('/upload-templates', data),
  get: (id: number) => http.get<any, any>(`/upload-templates/${id}`),
  update: (id: number, data: any) => http.put<any, any>(`/upload-templates/${id}`, data),
  delete: (id: number) => http.delete<any, any>(`/upload-templates/${id}`)
}

// ============ 合作方(已合并到 Game.partner_config,以下为兼容保留) ============
export type PartnerType = 'api' | 'webhook'

// ============ P8：操作日志 ============

export interface AuditLog {
  id: number
  user_id: number | null
  username: string
  role: string
  action: string
  target_type: string
  target_id: number | null
  target_label: string
  before: Record<string, any>
  after: Record<string, any>
  extension: Record<string, any>
  ip: string
  user_agent: string
  status: 'success' | 'failure'
  error_message: string
  created_at: string
}

export const auditApi = {
  list: (params?: any) => http.get<any, any>('/audit/logs', { params }),
  get: (id: number) => http.get<any, any>(`/audit/logs/${id}`),
  stats: () => http.get<any, any>('/audit/stats')
}

// ============ P8：注册扩展字段 ============

export interface ExtensionField {
  id: number
  code: string
  label: string
  type: 'text' | 'textarea' | 'number' | 'select' | 'date'
  required: boolean
  enabled: boolean
  sort: number
  options: Record<string, any>
  placeholder: string
  remark: string
  created_by: number | null
  created_at: string
}

export const extensionFieldApi = {
  list: () => http.get<any, any>('/lead-extension-fields'),
  adminList: () => http.get<any, any>('/lead-extension-fields/admin'),
  create: (data: any) => http.post<any, any>('/lead-extension-fields', data),
  update: (id: number, data: any) => http.put<any, any>(`/lead-extension-fields/${id}`, data),
  delete: (id: number) => http.delete<any, any>(`/lead-extension-fields/${id}`)
}

// ============ P9:玩家列表(按 player_id+game_id 聚合)============

export interface PlayerSummary {
  player_id: string
  game_id: number
  game_code?: string
  game_name?: string
  latest_username: string
  latest_country: string
  latest_is_target: string
  lead_count: number
  order_count: number
  paid_order_count: number
  total_order_amount: number
  first_lead_at: string | null
  last_lead_at: string | null
  last_follow_at: string | null
  creator_ids: number[]
  claimed_by_ids: number[]
  lead_ids: number[]
}

export interface PlayerFollowRecord {
  id: number
  lead_id: number
  follower_id: number | null
  follow_type: string
  content: string
  created_at: string
  follower?: { id: number; username: string; real_name: string } | null
}

export interface PlayerDetailResponse {
  summary: PlayerSummary
  leads: Lead[]
  orders: any[]   // 复用现有 Order 类型(orders 模块里有)
  follow_records: PlayerFollowRecord[]
}

export const playerApi = {
  list: (params?: any) => http.get<any, any>('/players', { params }),
  kpi: () => http.get<any, any>('/players/kpi'),
  detail: (player_id: string, game_id: number) =>
    http.get<any, any>(`/players/${encodeURIComponent(player_id)}/${game_id}`)
}

// ============ 列表视图(玩家 / 订单 字段可配置)============

export type ListViewEntityType = 'player' | 'order'

export interface ListViewField {
  code: string
  label: string
  source: string
  width: number
  align?: 'left' | 'center' | 'right'
  formatter?: string
  sortable?: boolean
  visible?: boolean
}

export interface ListView {
  id: number
  entity_type: ListViewEntityType
  name: string
  is_default: boolean
  is_system: boolean
  fields: ListViewField[]
  created_by: number | null
  created_at: string
  updated_at: string
}

export interface ListViewAvailableField {
  code: string
  label: string
  source: string
  formatter: string
  group_key: string
  group_name: string
}

export const listViewApi = {
  // 取默认视图（entity_type + is_default=true 合并查询）
  getDefault: (entity_type: ListViewEntityType) =>
    http.get<any, any>('/list-views', { params: { entity_type, is_default: 'true' } }),

  // 单个视图
  get: (id: number) => http.get<any, any>(`/list-views/${id}`),

  // 列出某 entity 的所有视图
  listByEntity: (entity_type: ListViewEntityType) =>
    http.get<any, any>('/list-views', { params: { entity_type } }),

  // 可用字段池
  availableFields: (entity_type: ListViewEntityType) =>
    http.get<any, any>('/list-views/available-fields', { params: { entity_type } }),

  // 更新（admin）
  update: (id: number, data: { name?: string; fields?: ListViewField[] }) =>
    http.put<any, any>(`/list-views/${id}`, data),

  // 设为默认（admin）
  setDefault: (id: number) => http.post<any, any>(`/list-views/${id}/set-default`)
}