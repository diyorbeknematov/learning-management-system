import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link, useNavigate } from 'react-router'
import { z } from 'zod'
import { useAuth } from '@/auth/context'
import { AuthCard } from '@/components/AuthCard'
import { FormField } from '@/components/FormField'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/lib/query'
import { applyApiErrors, email, password, required, username } from '@/lib/validation'

const schema = z.object({
  first_name: required('First name'),
  last_name: required('Last name'),
  username,
  email,
  password,
})

type Values = z.infer<typeof schema>

export default function RegisterPage() {
  const { register: signUp } = useAuth()
  const navigate = useNavigate()
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
      await signUp(values)
      navigate('/', { replace: true })
    } catch (error) {
      if (!applyApiErrors(error, setError)) setFailure(errorMessage(error))
    }
  }

  return (
    <AuthCard
      title="Create an account"
      description="You will join as a student."
      footer={
        <span>
          Already have an account?{' '}
          <Link to="/login" className="underline">
            Log in
          </Link>
        </span>
      }
    >
      <form onSubmit={handleSubmit(submit)} className="space-y-4" noValidate>
        <div className="grid grid-cols-2 gap-3">
          <FormField label="First name" autoComplete="given-name" error={errors.first_name} {...register('first_name')} />
          <FormField label="Last name" autoComplete="family-name" error={errors.last_name} {...register('last_name')} />
        </div>
        <FormField label="Username" autoComplete="username" error={errors.username} {...register('username')} />
        <FormField label="Email" type="email" autoComplete="email" error={errors.email} {...register('email')} />
        <FormField label="Password" type="password" autoComplete="new-password" error={errors.password} {...register('password')} />

        {failure && (
          <p role="alert" className="text-sm text-destructive">
            {failure}
          </p>
        )}

        <Button type="submit" className="w-full" disabled={isSubmitting}>
          {isSubmitting ? 'Creating…' : 'Sign up'}
        </Button>
      </form>
    </AuthCard>
  )
}
