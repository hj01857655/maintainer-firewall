import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api'
import { Button } from '../components/Button'
import { Card } from '../components/Card'
import { Loading } from '../components/Loading'
import { Modal } from '../components/Modal'
import { Table, TableHeader, TableBody, TableHead, TableRow, TableCell } from '../components/Table'
import { cn } from '../utils/cn'

interface AdminUser {
  id: number
  username: string
  role: string
  permissions: string[]
  is_active: boolean
  created_at: string
  updated_at: string
  last_login_at?: string
}

interface CreateUserRequest {
  username: string
  password: string
  role: string
  permissions: string[]
  is_active: boolean
}

interface UpdateUserRequest {
  username: string
  role: string
  permissions: string[]
  is_active: boolean
}

interface UpdatePasswordRequest {
  current_password: string
  new_password: string
}

export function UsersPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showEditModal, setShowEditModal] = useState(false)
  const [showPasswordModal, setShowPasswordModal] = useState(false)
  const [selectedUser, setSelectedUser] = useState<AdminUser | null>(null)
  const [createForm, setCreateForm] = useState<CreateUserRequest>({
    username: '',
    password: '',
    role: 'viewer',
    permissions: ['read'],
    is_active: true,
  })
  const [editForm, setEditForm] = useState<UpdateUserRequest>({
    username: '',
    role: 'viewer',
    permissions: ['read'],
    is_active: true,
  })
  const [passwordForm, setPasswordForm] = useState<UpdatePasswordRequest>({
    current_password: '',
    new_password: '',
  })

  // 获取用户列表
  const { data: usersData, isLoading, error } = useQuery({
    queryKey: ['users'],
    queryFn: async () => {
      const response = await api.get('/api/users?limit=100')
      return response
    },
    retry: 1, // 只重试1次，避免无限loading
  })

  // 创建用户
  const createMutation = useMutation({
    mutationFn: async (user: CreateUserRequest) => {
      const response = await api.post('/api/users', user)
      return response
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      setShowCreateModal(false)
      setCreateForm({
        username: '',
        password: '',
        role: 'viewer',
        permissions: ['read'],
        is_active: true,
      })
    },
  })

  // 更新用户
  const updateMutation = useMutation({
    mutationFn: async ({ id, user }: { id: number; user: UpdateUserRequest }) => {
      const response = await api.put(`/api/users/${id}`, user)
      return response
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      setShowEditModal(false)
      setSelectedUser(null)
    },
  })

  // 更新密码
  const updatePasswordMutation = useMutation({
    mutationFn: async ({ id, password }: { id: number; password: UpdatePasswordRequest }) => {
      const response = await api.put(`/api/users/${id}/password`, password)
      return response
    },
    onSuccess: () => {
      setShowPasswordModal(false)
      setSelectedUser(null)
      setPasswordForm({ current_password: '', new_password: '' })
    },
  })

  // 切换用户状态
  const toggleActiveMutation = useMutation({
    mutationFn: async ({ id, isActive }: { id: number; isActive: boolean }) => {
      const response = await api.patch(`/api/users/${id}/active`, { is_active: isActive })
      return response
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })

  // 删除用户
  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      const response = await api.delete(`/api/users/${id}`)
      return response
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })

  const handleCreateUser = () => {
    createMutation.mutate(createForm)
  }

  const handleEditUser = () => {
    if (selectedUser) {
      updateMutation.mutate({ id: selectedUser.id, user: editForm })
    }
  }

  const handleUpdatePassword = () => {
    if (selectedUser) {
      updatePasswordMutation.mutate({ id: selectedUser.id, password: passwordForm })
    }
  }

  const handleToggleActive = (user: AdminUser) => {
    toggleActiveMutation.mutate({ id: user.id, isActive: !user.is_active })
  }

  const handleDeleteUser = (user: AdminUser) => {
    if (confirm(t('users.confirmDelete', { username: user.username }))) {
      deleteMutation.mutate(user.id)
    }
  }

  const openEditModal = (user: AdminUser) => {
    setSelectedUser(user)
    setEditForm({
      username: user.username,
      role: user.role,
      permissions: user.permissions,
      is_active: user.is_active,
    })
    setShowEditModal(true)
  }

  const openPasswordModal = (user: AdminUser) => {
    setSelectedUser(user)
    setPasswordForm({ current_password: '', new_password: '' })
    setShowPasswordModal(true)
  }

  const availablePermissions = ['read', 'write', 'admin']
  const availableRoles = [
    { value: 'viewer', label: t('users.roles.viewer') },
    { value: 'editor', label: t('users.roles.editor') },
    { value: 'admin', label: t('users.roles.admin') },
  ]

  if (isLoading) {
    return <Loading />
  }

  const users = usersData?.users || []

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t('users.title')}</h1>
          <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
            {t('users.description')}
          </p>
        </div>
        <Button onClick={() => setShowCreateModal(true)}>
          {t('users.createUser')}
        </Button>
      </div>

      <Card>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('users.username')}</TableHead>
              <TableHead>{t('users.role')}</TableHead>
              <TableHead>{t('users.permissionsLabel')}</TableHead>
              <TableHead>{t('users.lastLogin')}</TableHead>
              <TableHead>{t('common.actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {users.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center py-8 text-slate-500">
                  {t('users.noUsers')}
                </TableCell>
              </TableRow>
            ) : (
              users.map((user: AdminUser) => (
                <TableRow key={user.id}>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{user.username}</span>
                      {!user.is_active && (
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400">
                          {t('users.status.inactive')}
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className={cn(
                      'inline-flex items-center px-2 py-1 rounded-full text-xs font-medium',
                      user.role === 'admin' && 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400',
                      user.role === 'editor' && 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
                      user.role === 'viewer' && 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400'
                    )}>
                      {t(`users.roles.${user.role}`)}
                    </span>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {user.permissions.map((permission: string) => (
                        <span
                          key={permission}
                          className="inline-flex items-center px-2 py-1 rounded text-xs bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                        >
                          {t(`users.permissions.${permission}`)}
                        </span>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-slate-600 dark:text-slate-400">
                      {user.last_login_at
                        ? new Date(user.last_login_at).toLocaleString()
                        : t('users.never')
                      }
                    </span>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => openEditModal(user)}
                      >
                        {t('common.edit')}
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => openPasswordModal(user)}
                      >
                        {t('users.changePassword')}
                      </Button>
                      <Button
                        size="sm"
                        variant={user.is_active ? 'destructive' : 'default'}
                        onClick={() => handleToggleActive(user)}
                        disabled={toggleActiveMutation.isPending}
                      >
                        {user.is_active ? t('users.deactivate') : t('users.activate')}
                      </Button>
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={() => handleDeleteUser(user)}
                        disabled={deleteMutation.isPending}
                      >
                        {t('common.delete')}
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </Card>

      {/* 创建用户模态框 */}
      <Modal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        title={t('users.createUser')}
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">
              {t('users.username')}
            </label>
            <input
              type="text"
              value={createForm.username}
              onChange={(e) => setCreateForm({ ...createForm, username: e.target.value })}
              className="w-full px-3 py-2 border rounded-md"
              placeholder={t('users.usernamePlaceholder')}
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">
              {t('users.password')}
            </label>
            <input
              type="password"
              value={createForm.password}
              onChange={(e) => setCreateForm({ ...createForm, password: e.target.value })}
              className="w-full px-3 py-2 border rounded-md"
              placeholder={t('users.passwordPlaceholder')}
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">
              {t('users.role')}
            </label>
            <select
              value={createForm.role}
              onChange={(e) => setCreateForm({ ...createForm, role: e.target.value })}
              className="w-full px-3 py-2 border rounded-md"
            >
              {availableRoles.map(role => (
                <option key={role.value} value={role.value}>
                  {role.label}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">
              {t('users.permissionsLabel')}
            </label>
            <div className="space-y-2">
              {availablePermissions.map(permission => (
                <label key={permission} className="flex items-center">
                  <input
                    type="checkbox"
                    checked={createForm.permissions.includes(permission)}
                    onChange={(e) => {
                      if (e.target.checked) {
                        setCreateForm({
                          ...createForm,
                          permissions: [...createForm.permissions, permission]
                        })
                      } else {
                        setCreateForm({
                          ...createForm,
                          permissions: createForm.permissions.filter(p => p !== permission)
                        })
                      }
                    }}
                    className="mr-2"
                  />
                  {t(`users.permissions.${permission}`)}
                </label>
              ))}
            </div>
          </div>

          <div className="flex justify-end gap-2 pt-4">
            <Button
              variant="outline"
              onClick={() => setShowCreateModal(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button
              onClick={handleCreateUser}
              disabled={createMutation.isPending}
            >
              {createMutation.isPending ? t('common.creating') : t('common.create')}
            </Button>
          </div>
        </div>
      </Modal>

      {/* 编辑用户模态框 */}
      <Modal
        isOpen={showEditModal}
        onClose={() => setShowEditModal(false)}
        title={t('users.editUser')}
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">
              {t('users.username')}
            </label>
            <input
              type="text"
              value={editForm.username}
              onChange={(e) => setEditForm({ ...editForm, username: e.target.value })}
              className="w-full px-3 py-2 border rounded-md"
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">
              {t('users.role')}
            </label>
            <select
              value={editForm.role}
              onChange={(e) => setEditForm({ ...editForm, role: e.target.value })}
              className="w-full px-3 py-2 border rounded-md"
            >
              {availableRoles.map(role => (
                <option key={role.value} value={role.value}>
                  {role.label}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">
              {t('users.permissionsLabel')}
            </label>
            <div className="space-y-2">
              {availablePermissions.map(permission => (
                <label key={permission} className="flex items-center">
                  <input
                    type="checkbox"
                    checked={editForm.permissions.includes(permission)}
                    onChange={(e) => {
                      if (e.target.checked) {
                        setEditForm({
                          ...editForm,
                          permissions: [...editForm.permissions, permission]
                        })
                      } else {
                        setEditForm({
                          ...editForm,
                          permissions: editForm.permissions.filter(p => p !== permission)
                        })
                      }
                    }}
                    className="mr-2"
                  />
                  {t(`users.permissions.${permission}`)}
                </label>
              ))}
            </div>
          </div>

          <div className="flex items-center">
            <input
              type="checkbox"
              checked={editForm.is_active}
              onChange={(e) => setEditForm({ ...editForm, is_active: e.target.checked })}
              className="mr-2"
            />
            <label className="text-sm font-medium">
              {t('users.active')}
            </label>
          </div>

          <div className="flex justify-end gap-2 pt-4">
            <Button
              variant="outline"
              onClick={() => setShowEditModal(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button
              onClick={handleEditUser}
              disabled={updateMutation.isPending}
            >
              {updateMutation.isPending ? t('common.updating') : t('common.update')}
            </Button>
          </div>
        </div>
      </Modal>

      {/* 修改密码模态框 */}
      <Modal
        isOpen={showPasswordModal}
        onClose={() => setShowPasswordModal(false)}
        title={t('users.changePassword')}
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">
              {t('users.changePasswordModal.currentPassword')}
            </label>
            <input
              type="password"
              value={passwordForm.current_password}
              onChange={(e) => setPasswordForm({ ...passwordForm, current_password: e.target.value })}
              className="w-full px-3 py-2 border rounded-md"
              placeholder={t('users.changePasswordModal.currentPasswordPlaceholder')}
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">
              {t('users.changePasswordModal.newPassword')}
            </label>
            <input
              type="password"
              value={passwordForm.new_password}
              onChange={(e) => setPasswordForm({ ...passwordForm, new_password: e.target.value })}
              className="w-full px-3 py-2 border rounded-md"
              placeholder={t('users.changePasswordModal.newPasswordPlaceholder')}
            />
          </div>

          <div className="flex justify-end gap-2 pt-4">
            <Button
              variant="outline"
              onClick={() => setShowPasswordModal(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button
              onClick={handleUpdatePassword}
              disabled={updatePasswordMutation.isPending}
            >
              {updatePasswordMutation.isPending ? t('common.updating') : t('users.changePassword')}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
