import { useQuery } from '@tanstack/react-query'
import { MoreHorizontal, Plus, Search, SlidersHorizontal, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { api, call } from '@/api/client'
import type { components } from '@/api/schema'
import { useAuth } from '@/auth/context'
import { PageHeader } from '@/components/PageHeader'
import { TableCard } from '@/components/Panels'
import { StatusBadge } from '@/components/StatusBadge'
import { NativeSelect } from '@/components/NativeSelect'
import { Pagination } from '@/components/Pagination'
import { Empty, ErrorBlock, LoadingBlock } from '@/components/States'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { formatDate, fullName, plural } from '@/lib/format'
import { useApiMutation } from '@/lib/mutations'
import { applyApiErrors } from '@/lib/validation'
import { useForm } from 'react-hook-form'

type User = components['schemas']['models.User']
type Role = 'SuperAdmin' | 'Instructor' | 'Student'

// a rounded, compact select for the filters
const pill = 'h-9 w-auto min-w-36 rounded-full bg-background pr-8'

const LIMIT = 20

function UserDialog({ user, onClose }: { user?: User; onClose: () => void }) {
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm({
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
  const [asking, setAsking] = useState(false)

  const status = useApiMutation(
    () => call(api.PATCH('/users/{id}/status', { params: { path: { id: user.id! } }, body: { status: blocked ? 'active' : 'blocked' } })),
    { success: blocked ? 'User unblocked' : 'User blocked and signed out', invalidate: [['users']] },
  )
  const remove = useApiMutation(() => call(api.DELETE('/users/{id}', { params: { path: { id: user.id! } } })), {
    success: 'User deleted',
    invalidate: [['users']],
  })

  const name = fullName(user)

  return (
    <TableRow>
      <TableCell>
        <div className="flex items-center gap-3">
          <Avatar>
            <AvatarImage src={user.avatar_url} alt="" />
            <AvatarFallback>{name.slice(0, 2).toUpperCase()}</AvatarFallback>
          </Avatar>
          <div className="min-w-0">
            <div className="font-medium">{name}</div>
            <div className="text-xs text-muted-foreground">
              @{user.username} · {user.email}
            </div>
          </div>
        </div>
      </TableCell>
      <TableCell>
        <StatusBadge value={user.role_name} />
      </TableCell>
      <TableCell>
        <StatusBadge value={user.status} />
      </TableCell>
      <TableCell className="text-muted-foreground">{formatDate(user.created_at)}</TableCell>
      <TableCell className="text-right">
        <div className="flex justify-end gap-2">
          <Button size="sm" variant="outline" onClick={onEdit}>
            Edit
          </Button>
          {!isMe && (
            <DropdownMenu>
              <DropdownMenuTrigger
                className="inline-flex size-7 items-center justify-center rounded-lg border outline-none hover:bg-muted focus-visible:ring-3 focus-visible:ring-ring/50"
                aria-label={`More actions for ${name}`}
              >
                <MoreHorizontal className="size-4" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => status.mutate(undefined)}>{blocked ? 'Unblock' : 'Block'}</DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem className="text-destructive" onClick={() => setAsking(true)}>
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>

        <AlertDialog open={asking} onOpenChange={setAsking}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete {name}?</AlertDialogTitle>
              <AlertDialogDescription>The user cannot log in any more.</AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction
                variant="destructive"
                onClick={() => {
                  setAsking(false)
                  remove.mutate(undefined)
                }}
              >
                Delete
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
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
          params: {
            query: {
              search: search || undefined,
              role: (role || undefined) as Role | undefined,
              status: (status || undefined) as 'active' | 'blocked' | undefined,
              page,
              limit: LIMIT,
            },
          },
        }),
      ),
    placeholderData: (previous) => previous,
  })

  return (
    <div className="space-y-6">
      <PageHeader
        title="Users"
        description={users.data ? `${plural(users.data.total, 'user')} on the platform` : 'People who use the platform'}
        actions={
          <Button onClick={() => setDialog({})}>
            <Plus /> New user
          </Button>
        }
      />

      <div className="space-y-4">
        <div className="relative w-full max-w-md">
          <Search className="pointer-events-none absolute left-4 top-3 size-4 text-muted-foreground" />
          <Input
            className="h-10 rounded-full pl-10"
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
            placeholder="Name, username or email"
            aria-label="Search users"
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <span className="mr-1 inline-flex items-center gap-1.5 text-sm font-medium">
            <SlidersHorizontal className="size-4" /> Filters
          </span>

          <NativeSelect
            aria-label="Role"
            className={pill}
            value={role}
            onChange={(e) => {
              setRole(e.target.value)
              setPage(1)
            }}
          >
            <option value="">All roles</option>
            <option value="Student">Students</option>
            <option value="Instructor">Instructors</option>
            <option value="SuperAdmin">SuperAdmins</option>
          </NativeSelect>

          <NativeSelect
            aria-label="Status"
            className={pill}
            value={status}
            onChange={(e) => {
              setStatus(e.target.value)
              setPage(1)
            }}
          >
            <option value="">Any status</option>
            <option value="active">Active</option>
            <option value="blocked">Blocked</option>
          </NativeSelect>

          {(role || status || typed) && (
            <Button
              variant="ghost"
              size="sm"
              className="rounded-full"
              onClick={() => {
                setRole('')
                setStatus('')
                setTyped('')
                setSearch('')
                setPage(1)
              }}
            >
              <X /> Clear
            </Button>
          )}

          <p className="ml-auto text-sm text-muted-foreground" aria-live="polite">
            {users.data ? plural(users.data.total, 'user') : ' '}
          </p>
        </div>
      </div>

      {users.isPending && <LoadingBlock />}
      {users.isError && <ErrorBlock error={users.error} />}
      {users.data?.items?.length === 0 && <Empty title="No users found" />}

      {(users.data?.items?.length ?? 0) > 0 && (
        <TableCard>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>User</TableHead>
                <TableHead>Role</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Joined</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.data?.items?.map((user) => (
                <UserRow key={user.id} user={user} isMe={user.id === me?.id} onEdit={() => setDialog({ user })} />
              ))}
            </TableBody>
          </Table>
        </TableCard>
      )}

      <Pagination page={page} limit={LIMIT} total={users.data?.total ?? 0} onPage={setPage} />

      {dialog && <UserDialog user={dialog.user} onClose={() => setDialog(null)} />}
    </div>
  )
}
