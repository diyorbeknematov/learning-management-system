import { useQuery } from '@tanstack/react-query'
import { hasRole, useAuth } from '@/auth/context'
import { api, call } from './client'

/** All categories (for filters and forms). */
export function useCategories() {
  return useQuery({
    queryKey: ['categories'],
    queryFn: () => call(api.GET('/categories', { params: { query: { limit: 100 } } })),
    staleTime: 5 * 60_000,
  })
}

/** The courses the logged in student studies; empty for everybody else. */
export function useMyEnrollments() {
  const { user } = useAuth()
  const isStudent = hasRole(user, 'Student') && user?.role_name === 'Student'

  return useQuery({
    queryKey: ['enrollments', 'me'],
    queryFn: () => call(api.GET('/enrollments/me', { params: { query: { limit: 100 } } })),
    enabled: isStudent,
  })
}
