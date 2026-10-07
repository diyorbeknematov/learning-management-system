import { zodResolver } from '@hookform/resolvers/zod'
import { Lock, User } from 'lucide-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link, useLocation, useNavigate } from 'react-router'
import { z } from 'zod'
import { ApiError } from '@/api/client'
import { useAuth } from '@/auth/context'
import { AuthCard } from '@/components/AuthCard'
import { FormField } from '@/components/FormField'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/lib/query'
import { required } from '@/lib/validation'

const schema = z.object({
  username: required('Username'),
  password: z.string().min(1, 'Password is required'),
})

type Values = z.infer<typeof schema>

export default function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [failure, setFailure] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<Values>({ resolver: zodResolver(schema) })

  async function submit(values: Values) {
    setFailure(null)

    try {
      await login(values.username.trim(), values.password)
      navigate((location.state as { from?: string } | null)?.from ?? '/dashboard', { replace: true })
    } catch (error) {
      // a wrong password, a blocked account or "try again in N seconds"
      setFailure(errorMessage(error))
      if (error instanceof ApiError && error.status >= 500) setFailure('The server has a problem. Try again later.')
    }
  }

  return (
    <AuthCard
      title="Log in"
      description="Welcome back."
      footer={
        <span>
          No account yet?{' '}
          <Link to="/register" className="underline">
            Sign up
          </Link>
        </span>
      }
    >
      <form onSubmit={handleSubmit(submit)} className="space-y-4" noValidate>
        <FormField label="Username" large icon={User} placeholder="Your username" autoComplete="username" error={errors.username} {...register('username')} />
        <FormField
          label="Password"
          large
          icon={Lock}
          type="password"
          placeholder="Your password"
          autoComplete="current-password"
          error={errors.password}
          {...register('password')}
        />

        {failure && (
          <p role="alert" className="text-sm text-destructive">
            {failure}
          </p>
        )}

        <div className="flex justify-end">
          <Link to="/forgot-password" className="text-sm font-medium text-primary hover:underline">
            Forgot the password?
          </Link>
        </div>

        <Button type="submit" size="lg" className="w-full" disabled={isSubmitting}>
          {isSubmitting ? 'Logging in…' : 'Log in'}
        </Button>
      </form>
    </AuthCard>
  )
}
