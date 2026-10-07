import { useMutation, useQueryClient, type QueryKey } from '@tanstack/react-query'
import { toast } from 'sonner'
import { errorMessage } from './query'

/**
 * A mutation that shows an error as a message, shows `success` when it works and
 * refreshes the queries in `invalidate` so the screen shows the new data.
 */
export function useApiMutation<Variables, Result>(
  mutationFn: (variables: Variables) => Promise<Result>,
  options: {
    invalidate?: QueryKey[]
    success?: string
    onSuccess?: (result: Result, variables: Variables) => void
    /** return true when the error was handled (a form shows its own fields) */
    onError?: (error: unknown) => boolean | void
  } = {},
) {
  const client = useQueryClient()

  return useMutation({
    mutationFn,
    onSuccess: (result, variables) => {
      if (options.success) toast.success(options.success)

      for (const queryKey of options.invalidate ?? []) client.invalidateQueries({ queryKey })

      options.onSuccess?.(result, variables)
    },
    onError: (error) => {
      if (options.onError?.(error) !== true) toast.error(errorMessage(error))
    },
  })
}
