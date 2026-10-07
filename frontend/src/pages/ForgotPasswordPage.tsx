import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link } from 'react-router'
import { z } from 'zod'
import { api, call } from '@/api/client'
import { AuthCard } from '@/components/AuthCard'
import { FormField } from '@/components/FormField'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/lib/query'
import { applyApiErrors, email } from '@/lib/validation'

const schema = z.object({ email })

type Values = z.infer<typeof schema>

export default function ForgotPasswordPage() {
  const [sentTo, setSentTo] = useState<string | null>(null)
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
      await call(api.POST('/auth/forgot-password', { body: { email: values.email } }))
      setSentTo(values.email)
    } catch (error) {
      if (!applyApiErrors(error, setError)) setFailure(errorMessage(error))
    }
  }

  if (sentTo) {
    return (
      <AuthCard title="Check your email" footer={<Link to="/login" className="underline">Back to log in</Link>}>
        <p className="text-sm">
          We sent a link to <strong>{sentTo}</strong>. It works for a short time and only once.
        </p>
      </AuthCard>
    )
  }

  return (
    <AuthCard
      title="Forgot the password?"
      description="Enter your email and we will send you a link to choose a new one."
      footer={<Link to="/login" className="underline">Back to log in</Link>}
    >
      <form onSubmit={handleSubmit(submit)} className="space-y-4" noValidate>
        <FormField label="Email" type="email" autoComplete="email" error={errors.email} {...register('email')} />

        {failure && (
          <p role="alert" className="text-sm text-destructive">
            {failure}
          </p>
        )}

        <Button type="submit" className="w-full" disabled={isSubmitting}>
          {isSubmitting ? 'Sending…' : 'Send the link'}
        </Button>
      </form>
    </AuthCard>
  )
}
