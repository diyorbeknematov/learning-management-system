// The tokens of the logged in user. They live in localStorage, so a reload keeps
// the session; the access token is short-lived and is renewed with the refresh
// token (see client.ts).

const ACCESS = 'lms.access_token'
const REFRESH = 'lms.refresh_token'

function read(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

export const tokens = {
  access: () => read(ACCESS),
  refresh: () => read(REFRESH),

  set(access: string, refresh: string) {
    try {
      localStorage.setItem(ACCESS, access)
      localStorage.setItem(REFRESH, refresh)
    } catch {
      // storage is blocked: the session lasts until the page is closed
    }
  },

  clear() {
    try {
      localStorage.removeItem(ACCESS)
      localStorage.removeItem(REFRESH)
    } catch {
      // nothing to clear
    }
  },
}
