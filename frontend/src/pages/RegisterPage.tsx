import { zodResolver } from '@hookform/resolvers/zod'
import { Lock, Mail, User } from 'lucide-react'
import { useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { Link, useNavigate } from 'react-router'
import { z } from 'zod'
import { useAuth } from '@/auth/context'
import { AuthCard } from '@/components/AuthCard'
import { FormField } from '@/components/FormField'
import { PasswordRules } from '@/components/PasswordRules'
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
    control,
    formState: { errors, isSubmitting },
  } = useForm<Values>({ resolver: zodResolver(schema) })

  const typed = useWatch({ control, name: 'password' })

  async function submit(values: Values) {
    setFailure(null)

    try {
      await signUp(values)
      navigate('/dashboard', { replace: true })
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
          <FormField label="First name" large autoComplete="given-name" error={errors.first_name} {...register('first_name')} />
          <FormField label="Last name" large autoComplete="family-name" error={errors.last_name} {...register('last_name')} />
        </div>
        <FormField
          label="Username"
          large
          icon={User}
          hint="3 to 30 letters, digits, . _ -"
          autoComplete="username"
          error={errors.username}
          {...register('username')}
        />
        <FormField
          label="Email"
          large
          icon={Mail}
          type="email"
          placeholder="you@example.com"
          autoComplete="email"
          error={errors.email}
          {...register('email')}
        />
        <FormField label="Password" large icon={Lock} type="password" autoComplete="new-password" error={errors.password} {...register('password')} />
        <PasswordRules value={typed} />

        {failure && (
          <p role="alert" className="text-sm text-destructive">
            {failure}
          </p>
        )}

        <Button type="submit" size="lg" className="w-full" disabled={isSubmitting}>
          {isSubmitting ? 'Creating…' : 'Sign up'}
        </Button>
      </form>
    </AuthCard>
  )
}
