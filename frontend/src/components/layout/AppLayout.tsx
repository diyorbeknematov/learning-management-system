import { Link, NavLink, Outlet } from 'react-router'
import { useAuth } from '@/auth/context'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { buttonVariants } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Toaster } from '@/components/ui/sonner'
import { cn } from '@/lib/utils'
import { useNavigate } from 'react-router'

function UserMenu() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  if (!user) return null

  const name = `${user.first_name ?? ''} ${user.last_name ?? ''}`.trim() || user.username
  const initials = ((user.first_name?.[0] ?? '') + (user.last_name?.[0] ?? '') || (user.username?.[0] ?? '?')).toUpperCase()

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="rounded-full outline-none focus-visible:ring-3 focus-visible:ring-ring/50" aria-label="Account menu">
        <Avatar>
          <AvatarImage src={user.avatar_url} alt="" />
          <AvatarFallback>{initials}</AvatarFallback>
        </Avatar>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuGroup>
          <DropdownMenuLabel>
            <div className="font-medium">{name}</div>
            <div className="text-xs font-normal opacity-70">{user.role_name}</div>
          </DropdownMenuLabel>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => navigate('/profile')}>Profile</DropdownMenuItem>
        <DropdownMenuItem
          onClick={async () => {
            await logout()
            navigate('/')
          }}
        >
          Log out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export function AppLayout() {
  const { user, loading } = useAuth()

  return (
    <div className="flex min-h-screen flex-col">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-5xl items-center gap-6 px-4">
          <Link to="/" className="font-semibold">
            LMS
          </Link>

          <nav className="flex flex-1 items-center gap-4 text-sm">
            <NavLink to="/" end className={({ isActive }) => cn('hover:underline', isActive && 'font-medium')}>
              Courses
            </NavLink>
            {(user?.role_name === 'Instructor' || user?.role_name === 'SuperAdmin') && (
              <NavLink to="/teach/courses" className={({ isActive }) => cn('hover:underline', isActive && 'font-medium')}>
                Teach
              </NavLink>
            )}
            {user?.role_name === 'SuperAdmin' && (
              <>
                {[
                  ['/admin/users', 'Users'],
                  ['/admin/categories', 'Categories'],
                  ['/admin/payments', 'Payments'],
                  ['/admin/finance', 'Finance'],
                  ['/admin/reports', 'Reports'],
                ].map(([to, label]) => (
                  <NavLink key={to} to={to} className={({ isActive }) => cn('hover:underline', isActive && 'font-medium')}>
                    {label}
                  </NavLink>
                ))}
              </>
            )}
            {user?.role_name === 'Student' && (
              <>
                <NavLink to="/my-courses" className={({ isActive }) => cn('hover:underline', isActive && 'font-medium')}>
                  My courses
                </NavLink>
                <NavLink to="/certificates" className={({ isActive }) => cn('hover:underline', isActive && 'font-medium')}>
                  Certificates
                </NavLink>
              </>
            )}
          </nav>

          {!loading &&
            (user ? (
              <UserMenu />
            ) : (
              <div className="flex gap-2">
                <Link to="/login" className={buttonVariants({ variant: 'ghost' })}>
                  Log in
                </Link>
                <Link to="/register" className={buttonVariants()}>
                  Sign up
                </Link>
              </div>
            ))}
        </div>
      </header>

      <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-8">
        <Outlet />
      </main>

      <Toaster />
    </div>
  )
}
