import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate } from 'react-router'
import { z } from 'zod'
import { api, call } from '@/api/client'
import { useCategories } from '@/api/queries'
import type { components } from '@/api/schema'
import { useAuth } from '@/auth/context'
import { FormField } from '@/components/FormField'
import { ImageUpload } from '@/components/ImageUpload'
import { NativeSelect } from '@/components/NativeSelect'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { useApiMutation } from '@/lib/mutations'
import { applyApiErrors, required } from '@/lib/validation'

type CourseDetail = components['schemas']['models.CourseDetail']

const number = (label: string) =>
  z.number({ error: `${label} must be a number` }).min(0, `${label} cannot be negative`)

const schema = z.object({
  title: required('Title'),
  category_id: z.string().min(1, 'Choose a category'),
  description: z.string(),
  difficulty: z.enum(['', 'beginner', 'intermediate', 'advanced']),
  language: z.string(),
  price: number('Price'),
  total_duration: number('Duration'),
  learning_outcomes: z.string(),
  requirements: z.string(),
  payout_type: z.enum(['', 'percentage', 'fixed']),
  payout_value: number('Payout'),
})

type Values = z.infer<typeof schema>

const lines = (text: string) =>
  text
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)

/** The form of a course: to create one, or to change the one given. */
export function CourseDetailsForm({ course }: { course?: CourseDetail }) {
  const { user } = useAuth()
  const navigate = useNavigate()
  const categories = useCategories()
  const isAdmin = user?.role_name === 'SuperAdmin'
  const [cover, setCover] = useState<string | undefined>()

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isDirty },
  } = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      title: course?.title ?? '',
      category_id: course?.category_id ?? '',
      description: course?.description ?? '',
      difficulty: course?.difficulty ?? '',
      language: course?.language ?? '',
      price: course?.price ?? 0,
      total_duration: course?.total_duration ?? 0,
      learning_outcomes: course?.learning_outcomes?.map((item) => item.content).join('\n') ?? '',
      requirements: course?.requirements?.map((item) => item.content).join('\n') ?? '',
      payout_type: course?.payout_type ?? '',
      payout_value: course?.payout_value ?? 0,
    },
  })

  const save = useApiMutation(
    (values: Values) => {
      const body = {
        title: values.title.trim(),
        category_id: values.category_id,
        description: values.description.trim(),
        difficulty: values.difficulty || undefined,
        language: values.language.trim() || undefined,
        price: values.price,
        total_duration: values.total_duration,
        learning_outcomes: lines(values.learning_outcomes),
        requirements: lines(values.requirements),
        cover,
        // only a SuperAdmin sets the payout; the API refuses it from others
        ...(isAdmin && values.payout_type ? { payout_type: values.payout_type, payout_value: values.payout_value } : {}),
      }

      return course
        ? call(api.PUT('/courses/{courseId}', { params: { path: { courseId: course.id! } }, body }))
        : call(api.POST('/courses', { body }))
    },
    {
      success: course ? 'Course saved' : 'Course created',
      invalidate: [['courses'], ['course']],
      onSuccess: (saved) => {
        if (!course) navigate(`/teach/courses/${saved.id}`)
      },
      onError: (error) => applyApiErrors(error, setError),
    },
  )

  return (
    <form onSubmit={handleSubmit((values) => save.mutate(values))} className="max-w-2xl space-y-5" noValidate>
      <FormField label="Title" error={errors.title} {...register('title')} />

      <div className="space-y-1.5">
        <Label htmlFor="description">Description</Label>
        <Textarea id="description" rows={5} {...register('description')} />
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label htmlFor="category_id">Category</Label>
          <NativeSelect id="category_id" aria-invalid={errors.category_id ? true : undefined} {...register('category_id')}>
            <option value="">Choose…</option>
            {categories.data?.items?.map((category) => (
              <option key={category.id} value={category.id}>
                {category.name}
              </option>
            ))}
          </NativeSelect>
          {errors.category_id && <p className="text-sm text-destructive">{errors.category_id.message}</p>}
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="difficulty">Level</Label>
          <NativeSelect id="difficulty" {...register('difficulty')}>
            <option value="">Not set</option>
            <option value="beginner">Beginner</option>
            <option value="intermediate">Intermediate</option>
            <option value="advanced">Advanced</option>
          </NativeSelect>
        </div>

        <FormField label="Language" placeholder="English" error={errors.language} {...register('language')} />
        <FormField label="Total duration (minutes)" type="number" min={0} error={errors.total_duration} {...register('total_duration', { valueAsNumber: true })} />
        <FormField label="Price (USD, 0 is free)" type="number" min={0} step="0.01" error={errors.price} {...register('price', { valueAsNumber: true })} />
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label htmlFor="learning_outcomes">What students learn</Label>
          <Textarea id="learning_outcomes" rows={5} placeholder="One point on each line" {...register('learning_outcomes')} />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="requirements">Requirements</Label>
          <Textarea id="requirements" rows={5} placeholder="One point on each line" {...register('requirements')} />
        </div>
      </div>

      <ImageUpload purpose="course_cover" label="Cover image" previewUrl={course?.cover_url} onUploaded={setCover} />

      {isAdmin && (
        <fieldset className="grid gap-4 rounded-lg border p-4 sm:grid-cols-2">
          <legend className="px-1 text-sm font-medium">Instructor payout (only you see this)</legend>
          <div className="space-y-1.5">
            <Label htmlFor="payout_type">Type</Label>
            <NativeSelect id="payout_type" {...register('payout_type')}>
              <option value="">No payout</option>
              <option value="percentage">Percentage of the price</option>
              <option value="fixed">Fixed amount per student</option>
            </NativeSelect>
          </div>
          <FormField label="Value" type="number" min={0} step="0.01" error={errors.payout_value} {...register('payout_value', { valueAsNumber: true })} />
        </fieldset>
      )}

      <Button type="submit" disabled={save.isPending || (Boolean(course) && !isDirty && !cover)}>
        {save.isPending ? 'Saving…' : course ? 'Save changes' : 'Create the course'}
      </Button>
    </form>
  )
}
