import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation } from '@tanstack/react-query'
import { Lock, Mail, User } from 'lucide-react'
import { useForm, useWatch } from 'react-hook-form'
import { useNavigate } from 'react-router'
import { toast } from 'sonner'
import { z } from 'zod'
import { api, call } from '@/api/client'
import { useAuth } from '@/auth/context'
import { FormField } from '@/components/FormField'
import { ImageUpload } from '@/components/ImageUpload'
import { PasswordRules } from '@/components/PasswordRules'
import { StatusBadge } from '@/components/StatusBadge'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { formatDate, fullName } from '@/lib/format'
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

function Header() {
  const { user } = useAuth()
  const name = fullName(user)

  return (
    <div className="overflow-hidden rounded-2xl border bg-card">
      <div className="h-32 bg-gradient-to-r from-primary via-indigo-500 to-sky-400 sm:h-40" />
      <div className="flex flex-wrap items-end gap-5 px-6 pb-6">
        <Avatar className="-mt-14 size-28 border-4 border-card text-3xl shadow-md">
          <AvatarImage src={user?.avatar_url} alt="" />
          <AvatarFallback>{name.slice(0, 2).toUpperCase()}</AvatarFallback>
        </Avatar>
        <div className="min-w-0 flex-1 space-y-1 pt-3">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="text-2xl font-bold tracking-tight">{name}</h1>
            <StatusBadge value={user?.role_name} />
          </div>
          <p className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground">
            <span className="inline-flex items-center gap-1.5">
              <User className="size-4" /> @{user?.username}
            </span>
            <span className="inline-flex items-center gap-1.5">
              <Mail className="size-4" /> {user?.email}
            </span>
            <span>Member since {formatDate(user?.created_at)}</span>
          </p>
        </div>
      </div>
    </div>
  )
}

function PhotoCard() {
  const { user, setUser } = useAuth()

  const save = useMutation({
    mutationFn: (avatar: string) => call(api.PUT('/users/me', { body: { avatar } })),
    onSuccess: (updated) => {
      setUser(updated)
      toast.success('Photo saved')
    },
    onError: (error) => toast.error(errorMessage(error)),
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>Photo</CardTitle>
        <CardDescription>It is shown in the menu and, for instructors, on the course page.</CardDescription>
      </CardHeader>
      <CardContent>
        <ImageUpload purpose="avatar" label="Your photo" previewUrl={user?.avatar_url} onUploaded={(key) => save.mutate(key)} />
      </CardContent>
    </Card>
  )
}

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
        <CardTitle>Personal information</CardTitle>
        <CardDescription>This is what other people see about you.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit((values) => save.mutate(values))} className="space-y-5" noValidate>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField label="First name" large error={errors.first_name} {...register('first_name')} />
            <FormField label="Last name" large error={errors.last_name} {...register('last_name')} />
            <FormField label="Username" large icon={User} error={errors.username} {...register('username')} />
            <FormField label="Email" large icon={Mail} type="email" error={errors.email} {...register('email')} />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="bio">About you</Label>
            <Textarea id="bio" rows={4} placeholder="A few words about you" {...register('bio')} />
            {errors.bio && <p className="text-sm text-destructive">{errors.bio.message}</p>}
          </div>

          <Button type="submit" size="lg" disabled={!isDirty || save.isPending}>
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
    control,
    formState: { errors },
  } = useForm<PasswordValues>({ resolver: zodResolver(passwordSchema) })

  const typed = useWatch({ control, name: 'new_password' })

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
        <form onSubmit={handleSubmit((values) => change.mutate(values))} className="max-w-md space-y-4" noValidate>
          <FormField
            label="Current password"
            large
            icon={Lock}
            type="password"
            autoComplete="current-password"
            error={errors.old_password}
            {...register('old_password')}
          />
          <FormField
            label="New password"
            large
            icon={Lock}
            type="password"
            autoComplete="new-password"
            error={errors.new_password}
            {...register('new_password')}
          />
          <PasswordRules value={typed} />
          <FormField
            label="Repeat the new password"
            large
            icon={Lock}
            type="password"
            autoComplete="new-password"
            error={errors.confirm}
            {...register('confirm')}
          />

          <Button type="submit" size="lg" disabled={change.isPending}>
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
      <Header />

      <Tabs defaultValue="personal">
        <TabsList>
          <TabsTrigger value="personal">Personal info</TabsTrigger>
          <TabsTrigger value="security">Security</TabsTrigger>
        </TabsList>

        <TabsContent value="personal" className="pt-4">
          <div className="grid gap-6 lg:grid-cols-[22rem_minmax(0,1fr)]">
            <PhotoCard />
            <ProfileForm />
          </div>
        </TabsContent>

        <TabsContent value="security" className="pt-4">
          <PasswordForm />
        </TabsContent>
      </Tabs>
    </div>
  )
}
