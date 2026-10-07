import { useQuery } from '@tanstack/react-query'
import { Plus, Search } from 'lucide-react'
import { useEffect, useState } from 'react'
import { api, call } from '@/api/client'
import type { components } from '@/api/schema'
import { useAuth } from '@/auth/context'
import { ConfirmButton } from '@/components/ConfirmButton'
import { NativeSelect } from '@/components/NativeSelect'
import { Pagination } from '@/components/Pagination'
import { Empty, ErrorBlock, LoadingBlock } from '@/components/States'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { fullName } from '@/lib/format'
import { useApiMutation } from '@/lib/mutations'
import { applyApiErrors } from '@/lib/validation'
import { useForm } from 'react-hook-form'

type User = components['schemas']['models.User']
type Role = 'SuperAdmin' | 'Instructor' | 'Student'

const LIMIT = 20

function UserDialog({ user, onClose }: { user?: User; onClose: () => void }) {
  const { register, handleSubmit, setError, formState: { errors } } = useForm({
    defaultValues: {
      first_name: user?.first_name ?? '',
      last_name: user?.last_name ?? '',
      username: user?.username ?? '',
      email: user?.email ?? '',
      password: '',
      role: (user?.role_name ?? 'Instructor') as Role,
    },
  })

  const save = useApiMutation(
    (values: { first_name: string; last_name: string; username: string; email: string; password: string; role: Role }) =>
      user
        ? call(
            api.PUT('/users/{id}', {
              params: { path: { id: user.id! } },
              body: { first_name: values.first_name, last_name: values.last_name, username: values.username, email: values.email, role: values.role },
            }),
          )
        : call(api.POST('/users', { body: values })),
    {
      success: user ? 'User saved' : 'User created',
      invalidate: [['users']],
      onSuccess: onClose,
      onError: (error) => applyApiErrors(error, setError),
    },
  )

  const field = (name: 'first_name' | 'last_name' | 'username' | 'email' | 'password', label: string, type = 'text') => (
    <div className="space-y-1.5">
      <Label htmlFor={`user-${name}`}>{label}</Label>
      <Input id={`user-${name}`} type={type} aria-invalid={errors[name] ? true : undefined} {...register(name)} />
      {errors[name] && <p className="text-sm text-destructive">{errors[name]?.message}</p>}
    </div>
  )

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <form className="grid gap-3" onSubmit={handleSubmit((values) => save.mutate(values))}>
          <DialogHeader>
            <DialogTitle>{user ? 'Edit the user' : 'New user'}</DialogTitle>
          </DialogHeader>
          <div className="grid grid-cols-2 gap-3">
            {field('first_name', 'First name')}
            {field('last_name', 'Last name')}
          </div>
          {field('username', 'Username')}
          {field('email', 'Email', 'email')}
          {!user && field('password', 'Password', 'password')}
          <div className="space-y-1.5">
            <Label htmlFor="user-role">Role</Label>
            <NativeSelect id="user-role" {...register('role')}>
              <option value="Student">Student</option>
              <option value="Instructor">Instructor</option>
              <option value="SuperAdmin">SuperAdmin</option>
            </NativeSelect>
          </div>
          {user && <p className="text-xs text-muted-foreground">If the role changes, the user must log in again.</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={save.isPending}>
              {save.isPending ? 'Saving…' : 'Save'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function UserRow({ user, isMe, onEdit }: { user: User; isMe: boolean; onEdit: () => void }) {
  const blocked = user.status === 'blocked'

  const status = useApiMutation(
    () => call(api.PATCH('/users/{id}/status', { params: { path: { id: user.id! } }, body: { status: blocked ? 'active' : 'blocked' } })),
    { success: blocked ? 'User unblocked' : 'User blocked and signed out', invalidate: [['users']] },
  )
  const remove = useApiMutation(() => call(api.DELETE('/users/{id}', { params: { path: { id: user.id! } } })), {
    success: 'User deleted',
    invalidate: [['users']],
  })

  return (
    <TableRow>
      <TableCell>
        <div className="font-medium">{fullName(user)}</div>
        <div className="text-xs text-muted-foreground">
          @{user.username} · {user.email}
        </div>
      </TableCell>
      <TableCell>
        <Badge variant="secondary">{user.role_name}</Badge>
      </TableCell>
      <TableCell>{blocked ? <Badge variant="outline">Blocked</Badge> : <Badge>Active</Badge>}</TableCell>
      <TableCell className="text-right">
        <div className="flex justify-end gap-1">
          <Button size="xs" variant="outline" onClick={onEdit}>
            Edit
          </Button>
          {!isMe && (
            <>
              <Button size="xs" variant="outline" disabled={status.isPending} onClick={() => status.mutate(undefined)}>
                {blocked ? 'Unblock' : 'Block'}
              </Button>
              <ConfirmButton size="xs" title={`Delete ${fullName(user)}?`} description="The user cannot log in any more." onConfirm={() => remove.mutate(undefined)} pending={remove.isPending}>
                Delete
              </ConfirmButton>
            </>
          )}
        </div>
      </TableCell>
    </TableRow>
  )
}

export default function UsersPage() {
  const { user: me } = useAuth()
  const [search, setSearch] = useState('')
  const [typed, setTyped] = useState('')
  const [role, setRole] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [dialog, setDialog] = useState<{ user?: User } | null>(null)

  useEffect(() => {
    const timer = setTimeout(() => {
      setSearch(typed)
      setPage(1)
    }, 350)

    return () => clearTimeout(timer)
  }, [typed])

  const users = useQuery({
    queryKey: ['users', { search, role, status, page }],
    queryFn: () =>
      call(
        api.GET('/users', {
          params: { query: { search: search || undefined, role: (role || undefined) as Role | undefined, status: (status || undefined) as 'active' | 'blocked' | undefined, page, limit: LIMIT } },
        }),
      ),
    placeholderData: (previous) => previous,
  })

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-2xl font-semibold">Users</h1>
        <Button onClick={() => setDialog({})}>
          <Plus /> New user
        </Button>
      </div>

      <div className="grid gap-3 sm:grid-cols-4">
        <div className="relative sm:col-span-2">
          <Search className="pointer-events-none absolute left-2.5 top-2 size-4 text-muted-foreground" />
          <Input className="pl-8" value={typed} onChange={(e) => setTyped(e.target.value)} placeholder="Name, username or email" aria-label="Search users" />
        </div>
        <NativeSelect aria-label="Role" value={role} onChange={(e) => { setRole(e.target.value); setPage(1) }}>
          <option value="">All roles</option>
          <option value="Student">Students</option>
          <option value="Instructor">Instructors</option>
          <option value="SuperAdmin">SuperAdmins</option>
        </NativeSelect>
        <NativeSelect aria-label="Status" value={status} onChange={(e) => { setStatus(e.target.value); setPage(1) }}>
          <option value="">Any status</option>
          <option value="active">Active</option>
          <option value="blocked">Blocked</option>
        </NativeSelect>
      </div>

      {users.isPending && <LoadingBlock />}
      {users.isError && <ErrorBlock error={users.error} />}
      {users.data?.items?.length === 0 && <Empty title="No users found" />}

      {(users.data?.items?.length ?? 0) > 0 && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>User</TableHead>
              <TableHead>Role</TableHead>
              <TableHead>Status</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {users.data?.items?.map((user) => (
              <UserRow key={user.id} user={user} isMe={user.id === me?.id} onEdit={() => setDialog({ user })} />
            ))}
          </TableBody>
        </Table>
      )}

      <Pagination page={page} limit={LIMIT} total={users.data?.total ?? 0} onPage={setPage} />

      {dialog && <UserDialog user={dialog.user} onClose={() => setDialog(null)} />}
    </div>
  )
}
