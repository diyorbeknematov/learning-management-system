import { QueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { ApiError } from '@/api/client'

/** The text to show for a failed call. */
export function errorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message

  return 'Something went wrong. Check the connection and try again.'
}

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      // a 4xx answer will not change by asking again
      retry: (count, error) => !(error instanceof ApiError && error.status < 500) && count < 2,
      refetchOnWindowFocus: false,
    },
    mutations: {
      onError: (error) => {
        // the forms show the errors of their own fields; a rate limit is for everybody
        if (error instanceof ApiError && error.status === 429) toast.error(error.message)
      },
    },
  },
})
