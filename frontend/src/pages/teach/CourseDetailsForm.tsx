import { zodResolver } from '@hookform/resolvers/zod'
import { CheckCircle2, DollarSign, Eye, FilePlus2, ListChecks, Rocket, Layers } from 'lucide-react'
import { useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { useNavigate } from 'react-router'
import { z } from 'zod'
import { api, call } from '@/api/client'
import { useCategories } from '@/api/queries'
import type { components } from '@/api/schema'
import { useAuth } from '@/auth/context'
import { CourseCard } from '@/components/CourseCard'
import { FormField } from '@/components/FormField'
import { ImageUpload } from '@/components/ImageUpload'
import { NativeSelect } from '@/components/NativeSelect'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { fullName } from '@/lib/format'
import { useApiMutation } from '@/lib/mutations'
import { applyApiErrors, required } from '@/lib/validation'

type CourseDetail = components['schemas']['models.CourseDetail']

const number = (label: string) => z.number({ error: `${label} must be a number` }).min(0, `${label} cannot be negative`)

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
  const [coverPreview, setCoverPreview] = useState<string | undefined>()

  const {
    register,
    handleSubmit,
    setError,
    control,
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

  const live = useWatch({ control })
  const category = categories.data?.items?.find((item) => item.id === live.category_id)

  // the course as a student will see it on the card
  const preview = {
    id: course?.id,
    title: live.title?.trim() || 'The title of your course',
    category_name: category?.name,
    instructor_name: course?.instructor_name ?? fullName(user),
    cover_url: coverPreview ?? course?.cover_url,
    price: Number.isFinite(live.price) ? live.price : 0,
    difficulty: live.difficulty || undefined,
    total_duration: Number.isFinite(live.total_duration) ? live.total_duration : 0,
    lesson_count: course?.lesson_count ?? 0,
    avg_rating: course?.avg_rating ?? 0,
    review_count: course?.review_count ?? 0,
  } as components['schemas']['models.CourseListItem']

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

      return course ? call(api.PUT('/courses/{courseId}', { params: { path: { courseId: course.id! } }, body })) : call(api.POST('/courses', { body }))
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
    <form onSubmit={handleSubmit((values) => save.mutate(values))} noValidate className="grid gap-6 xl:grid-cols-[minmax(0,48rem)_24rem] xl:justify-center">
      <div className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Basics</CardTitle>
            <CardDescription>The title and the description are the first things a student reads.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            <FormField label="Title" large error={errors.title} {...register('title')} />

            <div className="space-y-1.5">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                rows={6}
                className="min-h-32 px-3.5 py-3 text-base"
                placeholder="What is the course about, and for whom?"
                {...register('description')}
              />
            </div>

            <div className="grid gap-5 md:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="learning_outcomes">What students learn</Label>
                <Textarea
                  id="learning_outcomes"
                  rows={6}
                  className="min-h-32 px-3.5 py-3 text-base"
                  placeholder="One point on each line"
                  {...register('learning_outcomes')}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="requirements">Requirements</Label>
                <Textarea
                  id="requirements"
                  rows={6}
                  className="min-h-32 px-3.5 py-3 text-base"
                  placeholder="One point on each line"
                  {...register('requirements')}
                />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="gap-3 py-4">
          <CardHeader>
            <CardTitle>Details</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
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
            <FormField
              label="Duration (minutes)"
              type="number"
              min={0}
              error={errors.total_duration}
              {...register('total_duration', { valueAsNumber: true })}
            />
          </CardContent>
        </Card>

        <Card className="gap-3 py-4">
          <CardHeader>
            <CardTitle>Price</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <FormField
              label="Price (USD, 0 is free)"
              large
              icon={DollarSign}
              type="number"
              min={0}
              step="0.01"
              error={errors.price}
              {...register('price', { valueAsNumber: true })}
            />

            {isAdmin && (
              <div className="space-y-4 border-t pt-4">
                <p className="text-sm font-medium">Instructor payout (only you see this)</p>
                <div className="space-y-1.5">
                  <Label htmlFor="payout_type">Type</Label>
                  <NativeSelect id="payout_type" {...register('payout_type')}>
                    <option value="">No payout</option>
                    <option value="percentage">Percentage of the price</option>
                    <option value="fixed">Fixed amount per student</option>
                  </NativeSelect>
                </div>
                <FormField
                  label="Value"
                  large
                  type="number"
                  min={0}
                  step="0.01"
                  error={errors.payout_value}
                  {...register('payout_value', { valueAsNumber: true })}
                />
              </div>
            )}
          </CardContent>
        </Card>

        {!course && (
          <Card>
            <CardHeader>
              <CardTitle>What happens next</CardTitle>
              <CardDescription>A course goes live in four steps.</CardDescription>
            </CardHeader>
            <CardContent>
              <ol className="space-y-4">
                {[
                  [FilePlus2, 'Create the course', 'The title, the description, the price and the cover: this page.'],
                  [Layers, 'Add the content', 'Modules and lessons with text, video links and files.'],
                  [ListChecks, 'Add a quiz', 'Optional. A final quiz decides if a student gets the certificate.'],
                  [Rocket, 'Publish', 'Only then students can see and join the course.'],
                ].map(([Icon, title, text], index) => {
                  const I = Icon as typeof CheckCircle2

                  return (
                    <li key={title as string} className="flex gap-3">
                      <span
                        className={
                          index === 0
                            ? 'flex size-8 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground'
                            : 'flex size-8 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground'
                        }
                      >
                        <I className="size-4" />
                      </span>
                      <div>
                        <p className="text-sm font-medium">
                          {index + 1}. {title as string}
                        </p>
                        <p className="text-sm text-muted-foreground">{text as string}</p>
                      </div>
                    </li>
                  )
                })}
              </ol>
            </CardContent>
          </Card>
        )}

        <div className="flex items-center gap-3">
          <Button type="submit" size="lg" disabled={save.isPending || (Boolean(course) && !isDirty && !cover)}>
            {save.isPending ? 'Saving…' : course ? 'Save changes' : 'Create the course'}
          </Button>
          {!course && <p className="text-sm text-muted-foreground">You add the lessons and quizzes next.</p>}
        </div>
      </div>

      <div className="space-y-6 xl:sticky xl:top-24 xl:self-start">
        <section aria-label="Preview" className="space-y-2">
          <p className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
            <Eye className="size-4" /> How it looks in the catalog
          </p>
          <div className="pointer-events-none" aria-hidden>
            <CourseCard course={preview} />
          </div>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Cover image</CardTitle>
            <CardDescription>Shown on the course card and the course page.</CardDescription>
          </CardHeader>
          <CardContent>
            <ImageUpload
              purpose="course_cover"
              label="Cover"
              previewUrl={course?.cover_url}
              onUploaded={(key, url) => {
                setCover(key)
                setCoverPreview(url)
              }}
            />
          </CardContent>
        </Card>
      </div>
    </form>
  )
}
