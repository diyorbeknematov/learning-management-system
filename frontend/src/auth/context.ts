import { createContext, useContext } from 'react'
import type { components } from '@/api/schema'

export type User = components['schemas']['models.User']
export type RegisterBody = components['schemas']['models.RegisterRequest']

export type Role = 'SuperAdmin' | 'Instructor' | 'Student'

export type AuthState = {
  user: User | null
  /** true while the stored tokens are being checked after a page load */
  loading: boolean
  login: (username: string, password: string) => Promise<void>
  register: (body: RegisterBody) => Promise<void>
  logout: () => Promise<void>
  setUser: (user: User) => void
}

export const AuthContext = createContext<AuthState | null>(null)

export function useAuth(): AuthState {
  const state = useContext(AuthContext)
  if (!state) throw new Error('useAuth must be used inside AuthProvider')

  return state
}

/** Whether the user may do what the role allows; a SuperAdmin can do what an Instructor can. */
export function hasRole(user: User | null, ...roles: Role[]): boolean {
  if (!user?.role_name) return false

  const role = user.role_name as Role

  return roles.includes(role) || (role === 'SuperAdmin' && roles.includes('Instructor'))
}
