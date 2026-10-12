<template>
  <div>
    <div class="page-header">
      <n-button type="primary" @click="openCreate">新增用户</n-button>
    </div>

    <!-- 筛选 / 排序 -->
    <n-card size="small" :bordered="false" style="margin-bottom: 12px">
      <n-space :wrap="false" align="center">
        <n-input v-model:value="filterKeyword" placeholder="账号 / 姓名(模糊)" clearable style="width: 200px" @keyup.enter="loadList" @clear="loadList" />
        <n-select v-model:value="filterRole" placeholder="角色" clearable style="width: 130px" :options="roleOptions" @update:value="loadList" />
        <n-select v-model:value="filterStatus" placeholder="状态" clearable style="width: 110px" :options="statusOptions" @update:value="loadList" />
        <n-select
          v-model:value="filterSuperiorID"
          placeholder="上级"
          clearable
          filterable
          style="width: 180px"
          :options="superiorFilterOptions"
          @update:value="loadList"
        />
        <n-divider vertical />
        <span style="font-size: 12px; color: #999">排序</span>
        <n-select v-model:value="sortBy" style="width: 130px" :options="sortByOptions" @update:value="loadList" />
        <n-select v-model:value="sortOrder" style="width: 90px" :options="sortOrderOptions" @update:value="loadList" />
        <n-divider vertical />
        <n-button @click="loadList">查询</n-button>
        <n-button quaternary @click="resetFilter">重置</n-button>
      </n-space>
    </n-card>

    <n-data-table
      :columns="columns"
      :data="users"
      :pagination="pagination"
      :loading="loading"
      :bordered="false"
      @update:page="(p: number) => { pagination.page = p; loadList() }"
      @update:page-size="(s: number) => { pagination.page = s; pagination.page = 1; loadList() }"
    />

    <!-- 新增/编辑弹窗 -->
    <n-modal v-model:show="showForm" preset="card" :title="editing.id ? '编辑用户' : '新增用户'" style="width: 600px">
      <n-form :model="editing" label-placement="left" label-width="100">
        <n-form-item label="账号" v-if="!editing.id">
          <n-input v-model:value="editing.username" placeholder="登录账号" />
        </n-form-item>
        <n-form-item label="密码" v-if="!editing.id">
          <n-input v-model:value="editing.password" type="password" placeholder="至少 6 位" show-password-on="click" />
        </n-form-item>
        <n-form-item label="真实姓名">
          <n-input v-model:value="editing.real_name" />
        </n-form-item>
        <n-form-item label="角色">
          <n-select v-model:value="editing.role" :options="roleOptions" @update:value="onRoleChange" />
        </n-form-item>
        <n-form-item label="上级">
          <n-select
            v-model:value="editing.superior_id"
            :options="superiorOptions"
            placeholder="选择上级(主管/组长);不选则直属顶层"
            clearable
            filterable
          />
        </n-form-item>
        <n-form-item label="状态">
          <n-select v-model:value="editing.status" :options="[{label:'启用',value:'enabled'},{label:'禁用',value:'disabled'}]" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showForm = false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="handleSubmit">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 重置密码 -->
    <n-modal v-model:show="showResetPwd" preset="card" title="重置密码" style="width: 400px">
      <n-form>
        <n-form-item label="新密码">
          <n-input v-model:value="newPassword" type="password" placeholder="至少 6 位" show-password-on="click" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showResetPwd = false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="handleResetPwd">确认</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, h } from 'vue'
import { NDataTable, NButton, NTag, NModal, NForm, NFormItem, NInput, NSelect, NSpace, NCard, NDivider, useMessage, type DataTableColumns } from 'naive-ui'
import { userApi, type User, type Role } from '@/api/modules'

const message = useMessage()
const users = ref<User[]>([])
const loading = ref(false)
const showForm = ref(false)
const showResetPwd = ref(false)
const submitting = ref(false)
const newPassword = ref('')
const resetUserId = ref<number | null>(null)

// 所有启用用户的精简列表(独立于分页,用于上级下拉候选)
// 确保选中的上级就算不在当前页也能显示真实名字
interface UserOption { id: number; username: string; real_name: string; role: string; status: string }
const userOptions = ref<UserOption[]>([])

const editing = ref<any>({
  id: null,
  username: '',
  password: '',
  real_name: '',
  role: 'a_staff',
  superior_id: null,
  status: 'enabled'
})

const pagination = ref({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: true, pageSizes: [10, 20, 50] })

const roleOptions = [
  { label: '超级管理员', value: 'admin' },
  { label: '前端主管', value: 'a_manager' },
  { label: '前端组长', value: 'a_lead' },
  { label: '前端员工', value: 'a_staff' },
  { label: '后端主管', value: 'b_manager' },
  { label: '后端组长', value: 'b_lead' },
  { label: '后端员工', value: 'b_staff' }
]

const statusOptions = [
  { label: '启用', value: 'enabled' },
  { label: '禁用', value: 'disabled' }
]

const sortByOptions = [
  { label: 'ID', value: 'id' },
  { label: '账号', value: 'username' },
  { label: '姓名', value: 'real_name' },
  { label: '角色', value: 'role' },
  { label: '状态', value: 'status' },
  { label: '最后登录', value: 'last_login_at' },
  { label: '创建时间', value: 'created_at' }
]

const sortOrderOptions = [
  { label: '降序', value: 'desc' },
  { label: '升序', value: 'asc' }
]

// 筛选状态
const filterKeyword = ref('')
const filterRole = ref<string | null>(null)
const filterStatus = ref<string | null>(null)
const filterSuperiorID = ref<number | null>(null)
const sortBy = ref<string>('id')
const sortOrder = ref<string>('desc')

// 上级候选(全集,来自独立端点 userApi.options,不依赖分页)
function toSelectOption(u: UserOption) {
  const roleLabel = roleOptions.find(r => r.value === u.role)?.label || u.role
  return {
    label: `${u.real_name || u.username} · ${roleLabel}`,
    value: u.id
  }
}

// 顶部筛选条:包含自己在内的所有启用用户
const superiorFilterOptions = computed(() => userOptions.value.map(toSelectOption))

// 弹窗:排除自己
const superiorOptions = computed(() => {
  const selfId = editing.value.id
  return userOptions.value
    .filter(u => u.id !== selfId)
    .map(toSelectOption)
})

function onRoleChange() {
  // 切换角色后,清空上级(避免跨部门误选)
  editing.value.superior_id = null
}

const columns: DataTableColumns<User> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '账号', key: 'username', width: 130 },
  { title: '姓名', key: 'real_name', width: 100 },
  {
    title: '角色',
    key: 'role',
    width: 110,
    render: (row) => h(NTag, { type: 'success', size: 'small' }, () => roleOptions.find(r => r.value === row.role)?.label || row.role)
  },
  {
    title: '上级',
    key: 'superior',
    width: 140,
    render: (row) => {
      const s = row.superior
      if (!s) return h('span', { style: 'color: #94a3b8' }, '—')
      return s.real_name || s.username || `#${s.id}`
    }
  },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: (row) => h(NTag, { type: row.status === 'enabled' ? 'success' : 'default', size: 'small' }, () => row.status === 'enabled' ? '启用' : '禁用')
  },
  {
    title: '最后登录',
    key: 'last_login_at',
    width: 170,
    render: (row) => row.last_login_at ? new Date(row.last_login_at).toLocaleString('zh-CN') : '-'
  },
  {
    title: '操作',
    key: 'actions',
    width: 220,
    render: (row) => h(NSpace, {}, () => [
      h(NButton, { size: 'small', text: true, type: 'primary', onClick: () => openEdit(row) }, () => '编辑'),
      h(NButton, { size: 'small', text: true, onClick: () => openResetPwd(row) }, () => '重置密码'),
      h(NButton, { size: 'small', text: true, type: 'error', onClick: () => handleDelete(row) }, () => '删除')
    ])
  }
]

async function loadList() {
  loading.value = true
  try {
    const params: any = {
      page: pagination.value.page,
      page_size: pagination.value.pageSize
    }
    if (filterKeyword.value.trim()) params.keyword = filterKeyword.value.trim()
    if (filterRole.value) params.role = filterRole.value
    if (filterStatus.value) params.status = filterStatus.value
    if (filterSuperiorID.value) params.superior_id = filterSuperiorID.value
    if (sortBy.value) params.sort_by = sortBy.value
    if (sortOrder.value) params.sort_order = sortOrder.value

    const resp: any = await userApi.list(params)
    users.value = resp.data.list || []
    pagination.value.itemCount = resp.data.total || 0
  } catch {} finally {
    loading.value = false
  }
}

// 拉所有启用用户的精简列表(供上级下拉用,不分页)
async function loadUserOptions() {
  try {
    const resp: any = await userApi.options()
    userOptions.value = resp.data || []
  } catch {}
}

function resetFilter() {
  filterKeyword.value = ''
  filterRole.value = null
  filterStatus.value = null
  filterSuperiorID.value = null
  sortBy.value = 'id'
  sortOrder.value = 'desc'
  pagination.value.page = 1
  loadList()
}

function openCreate() {
  editing.value = {
    id: null, username: '', password: '', real_name: '',
    role: 'a_staff', superior_id: null, status: 'enabled'
  }
  showForm.value = true
}

function openEdit(row: User) {
  editing.value = { ...row, password: '' }
  showForm.value = true
}

async function handleSubmit() {
  submitting.value = true
  try {
    if (editing.value.id) {
      await userApi.update(editing.value.id, {
        real_name: editing.value.real_name,
        role: editing.value.role,
        superior_id: editing.value.superior_id,
        status: editing.value.status
      })
      message.success('更新成功')
    } else {
      await userApi.create({
        username: editing.value.username,
        password: editing.value.password,
        real_name: editing.value.real_name,
        role: editing.value.role,
        superior_id: editing.value.superior_id,
        status: editing.value.status
      })
      message.success('创建成功')
    }
    showForm.value = false
    loadList()
    loadUserOptions() // 候选列表可能变化(新增用户/启停状态)
  } catch {} finally {
    submitting.value = false
  }
}

function openResetPwd(row: User) {
  resetUserId.value = row.id
  newPassword.value = ''
  showResetPwd.value = true
}

async function handleResetPwd() {
  if (!newPassword.value || newPassword.value.length < 6) {
    message.error('密码至少 6 位')
    return
  }
  submitting.value = true
  try {
    await userApi.resetPassword(resetUserId.value!, newPassword.value)
    message.success('密码已重置')
    showResetPwd.value = false
  } catch {} finally {
    submitting.value = false
  }
}

async function handleDelete(row: User) {
  if (!confirm(`确认删除用户「${row.username}」？`)) return
  try {
    await userApi.delete(row.id)
    message.success('已删除')
    loadList()
    loadUserOptions() // 候选列表可能变化(禁用后不再出现)
  } catch {}
}

onMounted(() => {
  loadList()
  loadUserOptions()
})
</script>

<style scoped>
</style>