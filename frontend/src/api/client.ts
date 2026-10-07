import createClient, { type Middleware } from 'openapi-fetch'
import type { components, paths } from './schema'
import { tokens } from './tokens'

// The address of the API. In development and behind the nginx of the frontend
// image /api is proxied to the backend; set VITE_API_URL to call another server.
const baseUrl = import.meta.env.VITE_API_URL ?? '/api/v1'

export const api = createClient<paths>({ baseUrl })

/** An answer of the API that is not a success: {"success": false, "error": {...}}. */
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  /** The wrong fields of a request and why, when the backend says so. */
  readonly fields: Record<string, string>

  constructor(status: number, code: string, message: string, fields: Record<string, string> = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
  }
}

type Failure = { error?: { code?: string; message?: string; fields?: Record<string, string> } }

/**
 * Turns the result of a call into its data, or throws an ApiError:
 *
 *   const courses = await call(api.GET('/courses'))
 *
 * Every success of the API is {"success": true, "data": ...}; this returns the
 * "data".
 */
export async function call<T extends { data?: unknown }>(
  request: Promise<{ data?: T; error?: unknown; response: Response }>,
): Promise<NonNullable<T['data']>> {
  const { data, error, response } = await request

  if (!response.ok) {
    const failure = (error ?? {}) as Failure

    throw new ApiError(
      response.status,
      failure.error?.code ?? 'INTERNAL',
      failure.error?.message ?? response.statusText,
      failure.error?.fields,
    )
  }

  return data?.data as NonNullable<T['data']>
}

// --- the access token and its renewal

type TokenPair = components['schemas']['models.TokenPair']

// The requests as they were sent, so that one that was refused because its
// access token expired can be sent again with a new one.
const sent = new WeakMap<Request, Request>()

let refreshing: Promise<boolean> | null = null

/** Gets a new token pair with the refresh token; false when that is not possible. */
function refresh(): Promise<boolean> {
  refreshing ??= (async () => {
    const refreshToken = tokens.refresh()
    if (!refreshToken) return false

    try {
      const response = await fetch(`${baseUrl}/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      })
      if (!response.ok) return false

      const body = (await response.json()) as { data?: TokenPair }
      if (!body.data?.access_token || !body.data.refresh_token) return false

      tokens.set(body.data.access_token, body.data.refresh_token)
      return true
    } catch {
      return false
    } finally {
      refreshing = null
    }
  })()

  return refreshing
}

const auth: Middleware = {
  onRequest({ request }) {
    sent.set(request, request.clone())

    const access = tokens.access()
    if (access) request.headers.set('Authorization', `Bearer ${access}`)

    return request
  },

  async onResponse({ request, response }) {
    // the login and the renewal answer 401 for a wrong password, not for an old token
    if (response.status !== 401 || !tokens.refresh() || new URL(request.url).pathname.includes('/auth/')) {
      return response
    }

    const original = sent.get(request)

    if (original && (await refresh())) {
      original.headers.set('Authorization', `Bearer ${tokens.access()}`)

      return fetch(original)
    }

    tokens.clear()

    return response
  },
}

api.use(auth)
