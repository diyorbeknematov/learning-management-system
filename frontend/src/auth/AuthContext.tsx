import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api, call } from '@/api/client'
import { onSessionEnded, tokens } from '@/api/tokens'
import { queryClient } from '@/lib/query'
import { AuthContext, type RegisterBody, type User } from './context'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  // true until we know whether the stored tokens still log somebody in
  const [loading, setLoading] = useState(() => Boolean(tokens.access() || tokens.refresh()))

  useEffect(() => {
    if (!loading) return

    call(api.GET('/users/me'))
      .then(setUser)
      .catch(() => tokens.clear())
      .finally(() => setLoading(false))
  }, [loading])

  // the tokens could not be renewed in the middle of the work
  useEffect(
    () =>
      onSessionEnded(() => {
        setUser(null)
        queryClient.clear()
      }),
    [],
  )

  const login = useCallback(async (username: string, password: string) => {
    const result = await call(api.POST('/auth/login', { body: { username, password } }))

    tokens.set(result.access_token ?? '', result.refresh_token ?? '')
    setUser(result.user ?? null)
  }, [])

  const register = useCallback(async (body: RegisterBody) => {
    const result = await call(api.POST('/auth/register', { body }))

    tokens.set(result.access_token ?? '', result.refresh_token ?? '')
    setUser(await call(api.GET('/users/me')))
  }, [])

  const logout = useCallback(async () => {
    const refreshToken = tokens.refresh()

    if (refreshToken) {
      // the server forgets the refresh token; the user leaves even if this fails
      await api.POST('/auth/logout', { body: { refresh_token: refreshToken } }).catch(() => undefined)
    }

    tokens.clear()
    setUser(null)
    queryClient.clear()
  }, [])

  const value = useMemo(
    () => ({ user, loading, login, register, logout, setUser }),
    [user, loading, login, register, logout],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
