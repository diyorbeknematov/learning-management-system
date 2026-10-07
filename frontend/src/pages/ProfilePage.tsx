import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { useNavigate } from 'react-router'
import { toast } from 'sonner'
import { z } from 'zod'
import { api, call } from '@/api/client'
import { useAuth } from '@/auth/context'
import { FormField } from '@/components/FormField'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { errorMessage } from '@/lib/query'
import { applyApiErrors, email, password, required, username } from '@/lib/validation'

const profileSchema = z.object({
  first_name: required('First name'),
  last_name: required('Last name'),
  username,
  email,
  bio: z.string().max(2000, 'At most 2000 characters'),
})

type ProfileValues = z.infer<typeof profileSchema>

function ProfileForm() {
  const { user, setUser } = useAuth()

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isDirty },
  } = useForm<ProfileValues>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      first_name: user?.first_name ?? '',
      last_name: user?.last_name ?? '',
      username: user?.username ?? '',
      email: user?.email ?? '',
      bio: user?.bio ?? '',
    },
  })

  const save = useMutation({
    mutationFn: (values: ProfileValues) => call(api.PUT('/users/me', { body: { ...values, bio: values.bio.trim() } })),
    onSuccess: (updated) => {
      setUser(updated)
      toast.success('Profile saved')
    },
    onError: (error) => {
      if (!applyApiErrors(error, setError)) toast.error(errorMessage(error))
    },
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>Profile</CardTitle>
        <CardDescription>
          Your role: <Badge variant="secondary">{user?.role_name}</Badge>
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit((values) => save.mutate(values))} className="space-y-4" noValidate>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField label="First name" error={errors.first_name} {...register('first_name')} />
            <FormField label="Last name" error={errors.last_name} {...register('last_name')} />
            <FormField label="Username" error={errors.username} {...register('username')} />
            <FormField label="Email" type="email" error={errors.email} {...register('email')} />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="bio">About you</Label>
            <textarea
              id="bio"
              rows={4}
              className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
              {...register('bio')}
            />
            {errors.bio && <p className="text-sm text-destructive">{errors.bio.message}</p>}
          </div>

          <Button type="submit" disabled={!isDirty || save.isPending}>
            {save.isPending ? 'Saving…' : 'Save'}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

const passwordSchema = z
  .object({ old_password: z.string().min(1, 'Enter the current password'), new_password: password, confirm: z.string() })
  .refine((v) => v.new_password === v.confirm, { path: ['confirm'], message: 'The passwords do not match' })

type PasswordValues = z.infer<typeof passwordSchema>

function PasswordForm() {
  const { logout } = useAuth()
  const navigate = useNavigate()

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<PasswordValues>({ resolver: zodResolver(passwordSchema) })

  const change = useMutation({
    mutationFn: (values: PasswordValues) =>
      call(api.PUT('/users/me/password', { body: { old_password: values.old_password, new_password: values.new_password } })),
    onSuccess: async () => {
      // the backend ends every session when the password changes
      await logout()
      toast.success('Password changed. Log in with the new password.')
      navigate('/login')
    },
    onError: (error) => {
      if (!applyApiErrors(error, setError)) toast.error(errorMessage(error))
    },
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>Password</CardTitle>
        <CardDescription>After the change you log in again, on every device.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit((values) => change.mutate(values))} className="max-w-sm space-y-4" noValidate>
          <FormField label="Current password" type="password" autoComplete="current-password" error={errors.old_password} {...register('old_password')} />
          <FormField label="New password" type="password" autoComplete="new-password" error={errors.new_password} {...register('new_password')} />
          <FormField label="Repeat the new password" type="password" autoComplete="new-password" error={errors.confirm} {...register('confirm')} />

          <Button type="submit" disabled={change.isPending}>
            {change.isPending ? 'Saving…' : 'Change the password'}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

export default function ProfilePage() {
  return (
    <div className="space-y-6">
      <ProfileForm />
      <PasswordForm />
    </div>
  )
}
