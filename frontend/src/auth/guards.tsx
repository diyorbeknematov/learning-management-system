import { Navigate, Outlet, useLocation } from 'react-router'
import { Skeleton } from '@/components/ui/skeleton'
import { hasRole, useAuth, type Role } from './context'

function Waiting() {
  return (
    <div className="mx-auto max-w-md space-y-3 p-8">
      <Skeleton className="h-8 w-1/2" />
      <Skeleton className="h-24 w-full" />
    </div>
  )
}

/** Pages for logged in users; the others are sent to the login page and brought back after it. */
export function RequireAuth({ roles }: { roles?: Role[] }) {
  const { user, loading } = useAuth()
  const location = useLocation()

  if (loading) return <Waiting />

  if (!user) return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />

  if (roles && !hasRole(user, ...roles)) return <Navigate to="/" replace />

  return <Outlet />
}

/** Pages only for visitors (login, register): a logged in user has nothing to do there. */
export function GuestOnly() {
  const { user, loading } = useAuth()

  if (loading) return <Waiting />

  return user ? <Navigate to="/dashboard" replace /> : <Outlet />
}
