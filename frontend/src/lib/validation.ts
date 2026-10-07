import { z } from 'zod'
import type { FieldValues, Path, UseFormSetError } from 'react-hook-form'
import { ApiError } from '@/api/client'

// The same rules as the backend, to tell the user before the request. The
// backend checks them again and its answer wins.

export const username = z
  .string()
  .trim()
  .regex(
    /^[A-Za-z0-9][A-Za-z0-9._-]{2,29}$/,
    '3 to 30 characters: Latin letters, digits, . _ - ; start with a letter or a digit',
  )

export const email = z.string().trim().min(1, 'Email is required').max(254).pipe(z.email('Enter a valid email'))

export const password = z
  .string()
  .min(8, 'At least 8 characters')
  .refine((v) => new TextEncoder().encode(v).length <= 72, 'At most 72 bytes')
  .refine((v) => /\p{Lu}/u.test(v), 'A capital letter is needed')
  .refine((v) => /\p{Ll}/u.test(v), 'A small letter is needed')
  .refine((v) => /\p{Nd}/u.test(v), 'A digit is needed')
  .refine((v) => /[\p{P}\p{S}]/u.test(v), 'A special character is needed')

export const required = (label: string) => z.string().trim().min(1, `${label} is required`)

/**
 * Puts the field errors of the backend ({"fields": {"username": "..."}}) on the
 * form. It returns true when the error was about the fields of the form, so the
 * caller shows any other error itself.
 */
export function applyApiErrors<T extends FieldValues>(error: unknown, setError: UseFormSetError<T>): boolean {
  if (!(error instanceof ApiError) || Object.keys(error.fields).length === 0) return false

  for (const [field, message] of Object.entries(error.fields)) {
    setError(field as Path<T>, { type: 'server', message })
  }

  return true
}
