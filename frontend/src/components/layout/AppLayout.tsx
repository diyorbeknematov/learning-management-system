import { Award, BookOpen, GraduationCap, LayoutDashboard, LogOut, User as UserIcon } from 'lucide-react'
import { Link, NavLink, Outlet, useNavigate } from 'react-router'
import { useAuth } from '@/auth/context'
import { StatusBadge } from '@/components/StatusBadge'
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
      <DropdownMenuTrigger
        className="rounded-full outline-none transition hover:opacity-80 focus-visible:ring-3 focus-visible:ring-ring/50"
        aria-label="Account menu"
      >
        <Avatar className="size-11">
          <AvatarImage src={user.avatar_url} alt="" />
          <AvatarFallback className="bg-primary/10 text-base font-medium text-primary">{initials}</AvatarFallback>
        </Avatar>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" sideOffset={8} className="w-72 p-0">
        <DropdownMenuGroup>
          <DropdownMenuLabel className="p-0">
            <div className="flex items-center gap-3 border-b bg-muted/40 px-4 py-4">
              <Avatar className="size-12">
                <AvatarImage src={user.avatar_url} alt="" />
                <AvatarFallback className="bg-primary/10 text-base font-medium text-primary">{initials}</AvatarFallback>
              </Avatar>
              <div className="min-w-0 space-y-0.5">
                <p className="truncate text-base font-semibold text-foreground">{name}</p>
                <p className="truncate text-sm font-normal text-muted-foreground">{user.email}</p>
                <StatusBadge value={user.role_name} />
              </div>
            </div>
          </DropdownMenuLabel>
        </DropdownMenuGroup>

        <div className="p-1.5">
          <DropdownMenuItem className="gap-3 py-2" onClick={() => navigate('/dashboard')}>
            <LayoutDashboard className="size-4" /> Dashboard
          </DropdownMenuItem>
          {user.role_name === 'Student' && (
            <>
              <DropdownMenuItem className="gap-3 py-2" onClick={() => navigate('/my-courses')}>
                <BookOpen className="size-4" /> My courses
              </DropdownMenuItem>
              <DropdownMenuItem className="gap-3 py-2" onClick={() => navigate('/certificates')}>
                <Award className="size-4" /> Certificates
              </DropdownMenuItem>
            </>
          )}
          <DropdownMenuItem className="gap-3 py-2" onClick={() => navigate('/profile')}>
            <UserIcon className="size-4" /> Profile and security
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            className="gap-3 py-2 text-destructive focus:text-destructive"
            onClick={async () => {
              await logout()
              navigate('/')
            }}
          >
            <LogOut className="size-4" /> Log out
          </DropdownMenuItem>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

const year = new Date().getFullYear()

const link = ({ isActive }: { isActive: boolean }) =>
  cn(
    'whitespace-nowrap rounded-lg px-3.5 py-2 text-base font-medium text-muted-foreground transition hover:bg-muted hover:text-foreground',
    isActive && 'bg-muted font-medium text-foreground',
  )

export function AppLayout() {
  const { user, loading } = useAuth()
  const role = user?.role_name

  return (
    <div className="flex min-h-screen flex-col">
      <header className="sticky top-0 z-40 border-b bg-background/85 backdrop-blur">
        <div className={`${container} flex h-[4.5rem] items-center gap-6`}>
          <Link to="/" className="flex shrink-0 items-center gap-3 text-xl font-semibold">
            <span className="flex size-10 items-center justify-center rounded-xl bg-primary text-primary-foreground">
              <GraduationCap className="size-6" />
            </span>
            <span className="hidden sm:inline">Edura</span>
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
                <Link to="/login" className={buttonVariants({ variant: 'ghost', size: 'lg' }) + ' h-10 px-4 text-base'}>
                  Log in
                </Link>
                <Link to="/register" className={buttonVariants({ size: 'lg' }) + ' h-10 px-4 text-base'}>
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
          <span>© {year} Edura</span>
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
