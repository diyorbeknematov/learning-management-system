import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link, useSearchParams } from 'react-router'
import { z } from 'zod'
import { api, call } from '@/api/client'
import { AuthCard } from '@/components/AuthCard'
import { FormField } from '@/components/FormField'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/lib/query'
import { applyApiErrors, password } from '@/lib/validation'

const schema = z
  .object({ new_password: password, confirm: z.string() })
  .refine((v) => v.new_password === v.confirm, { path: ['confirm'], message: 'The passwords do not match' })

type Values = z.infer<typeof schema>

// The page of the link in the email: /reset-password?token=...
export default function ResetPasswordPage() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const [done, setDone] = useState(false)
  const [failure, setFailure] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<Values>({ resolver: zodResolver(schema) })

  async function submit(values: Values) {
    setFailure(null)

    try {
      await call(api.POST('/auth/reset-password', { body: { token, new_password: values.new_password } }))
      setDone(true)
    } catch (error) {
      if (!applyApiErrors(error, setError)) setFailure(errorMessage(error))
    }
  }

  if (!token) {
    return (
      <AuthCard title="This link is not valid" footer={<Link to="/forgot-password" className="underline">Ask for a new link</Link>}>
        <p className="text-sm">The link has no token. Open the link from the email again.</p>
      </AuthCard>
    )
  }

  if (done) {
    return (
      <AuthCard title="Password changed">
        <p className="mb-4 text-sm">You can log in with the new password.</p>
        <Link to="/login" className="text-sm underline">
          Go to log in
        </Link>
      </AuthCard>
    )
  }

  return (
    <AuthCard title="Choose a new password" footer={<Link to="/forgot-password" className="underline">Ask for a new link</Link>}>
      <form onSubmit={handleSubmit(submit)} className="space-y-4" noValidate>
        <FormField label="New password" type="password" autoComplete="new-password" error={errors.new_password} {...register('new_password')} />
        <FormField label="Repeat the password" type="password" autoComplete="new-password" error={errors.confirm} {...register('confirm')} />

        {failure && (
          <p role="alert" className="text-sm text-destructive">
            {failure}. The link may have expired or was already used.
          </p>
        )}

        <Button type="submit" size="lg" className="w-full" disabled={isSubmitting}>
          {isSubmitting ? 'Saving…' : 'Change the password'}
        </Button>
      </form>
    </AuthCard>
  )
}
