import { GraduationCap } from 'lucide-react'
import { Link, NavLink, Outlet, useNavigate } from 'react-router'
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
import { container } from './container'

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
        <DropdownMenuItem onClick={() => navigate('/dashboard')}>Dashboard</DropdownMenuItem>
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

const year = new Date().getFullYear()

const link = ({ isActive }: { isActive: boolean }) =>
  cn('whitespace-nowrap rounded-md px-2.5 py-1.5 text-sm text-muted-foreground transition hover:bg-muted hover:text-foreground', isActive && 'bg-muted font-medium text-foreground')

export function AppLayout() {
  const { user, loading } = useAuth()
  const role = user?.role_name

  return (
    <div className="flex min-h-screen flex-col">
      <header className="sticky top-0 z-40 border-b bg-background/85 backdrop-blur">
        <div className={`${container} flex h-14 items-center gap-4`}>
          <Link to="/" className="flex shrink-0 items-center gap-2 font-semibold">
            <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <GraduationCap className="size-5" />
            </span>
            <span className="hidden sm:inline">LMS Academy</span>
          </Link>

          <nav className="flex min-w-0 flex-1 items-center gap-1 overflow-x-auto" aria-label="Main">
            <NavLink to="/courses" className={link}>
              Courses
            </NavLink>
            {user && (
              <NavLink to="/dashboard" className={link}>
                Dashboard
              </NavLink>
            )}
            {role === 'Student' && (
              <>
                <NavLink to="/my-courses" className={link}>
                  My courses
                </NavLink>
                <NavLink to="/certificates" className={link}>
                  Certificates
                </NavLink>
              </>
            )}
            {(role === 'Instructor' || role === 'SuperAdmin') && (
              <NavLink to="/teach/courses" className={link}>
                Teach
              </NavLink>
            )}
            {role === 'SuperAdmin' &&
              [
                ['/admin/users', 'Users'],
                ['/admin/categories', 'Categories'],
                ['/admin/payments', 'Payments'],
                ['/admin/finance', 'Finance'],
                ['/admin/reports', 'Reports'],
              ].map(([to, label]) => (
                <NavLink key={to} to={to} className={link}>
                  {label}
                </NavLink>
              ))}
          </nav>

          {!loading &&
            (user ? (
              <UserMenu />
            ) : (
              <div className="flex shrink-0 gap-2">
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

      <main className="flex-1">
        <Outlet />
      </main>

      <footer className="border-t">
        <div className={`${container} flex flex-wrap items-center justify-between gap-2 py-6 text-sm text-muted-foreground`}>
          <span>© {year} LMS Academy</span>
          <nav className="flex gap-4" aria-label="Footer">
            <Link to="/courses" className="hover:text-foreground">
              Courses
            </Link>
            <Link to="/verify" className="hover:text-foreground">
              Check a certificate
            </Link>
          </nav>
        </div>
      </footer>

      <Toaster />
    </div>
  )
}
