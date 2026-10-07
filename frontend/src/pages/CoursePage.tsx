import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Clock, GraduationCap, Layers, Users } from 'lucide-react'
import { Link, useNavigate, useParams } from 'react-router'
import { toast } from 'sonner'
import { api, call } from '@/api/client'
import { useMyEnrollments } from '@/api/queries'
import { hasRole, useAuth } from '@/auth/context'
import { CourseCover } from '@/components/CourseCard'
import { Rating } from '@/components/Rating'
import { Reviews } from '@/components/Reviews'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { capitalize, duration, fullName, money } from '@/lib/format'
import { errorMessage } from '@/lib/query'
import { ApiError } from '@/api/client'

export default function CoursePage() {
  const { courseId = '' } = useParams()
  const { user } = useAuth()
  const navigate = useNavigate()
  const client = useQueryClient()

  const course = useQuery({
    queryKey: ['course', courseId],
    queryFn: () => call(api.GET('/courses/{courseId}', { params: { path: { courseId } } })),
  })

  const enrollments = useMyEnrollments()
  const isStudent = user?.role_name === 'Student'
  const enrollment = enrollments.data?.items?.find((item) => item.course_id === courseId && item.status !== 'dropped')

  const enroll = useMutation({
    mutationFn: () => call(api.POST('/courses/{courseId}/enrollments', { params: { path: { courseId } } })),
    onSuccess: () => {
      toast.success('You are enrolled')
      client.invalidateQueries({ queryKey: ['enrollments'] })
      client.invalidateQueries({ queryKey: ['course', courseId] })
      navigate(`/learn/${courseId}`)
    },
    onError: (error) => {
      // already enrolled on another device or tab
      if (error instanceof ApiError && error.status === 409) {
        client.invalidateQueries({ queryKey: ['enrollments'] })
      }
      toast.error(errorMessage(error))
    },
  })

  if (course.isPending) return <LoadingBlock rows={4} />
  if (course.isError) return <ErrorBlock error={course.error} />

  const data = course.data
  const instructor = data.instructor
  const canManage = user && (user.role_name === 'SuperAdmin' || user.id === data.instructor_id)

  function action() {
    if (!user) {
      return (
        <Link to="/login" state={{ from: `/courses/${courseId}` }} className={buttonVariants({ size: 'lg' }) + ' w-full'}>
          Log in to enroll
        </Link>
      )
    }

    if (enrollment) {
      return (
        <Link to={`/learn/${courseId}`} className={buttonVariants({ size: 'lg' }) + ' w-full'}>
          {enrollment.status === 'completed' ? 'Review the course' : 'Continue learning'}
        </Link>
      )
    }

    if (isStudent) {
      return (
        <Button size="lg" className="w-full" disabled={enroll.isPending} onClick={() => enroll.mutate()}>
          {enroll.isPending ? 'Enrolling…' : data.price ? `Enroll for ${money(data.price)}` : 'Enroll for free'}
        </Button>
      )
    }

    return null
  }

  return (
    <div className="space-y-10">
      <div className="grid gap-8 lg:grid-cols-3">
        <div className="space-y-4 lg:col-span-2">
          <div className="flex flex-wrap gap-2">
            {data.category_name && <Badge variant="outline">{data.category_name}</Badge>}
            {data.difficulty && <Badge variant="secondary">{capitalize(data.difficulty)}</Badge>}
            {data.status === 'draft' && <Badge>Draft: only you can see it</Badge>}
          </div>

          <h1 className="text-3xl font-semibold leading-tight">{data.title}</h1>

          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground">
            <Rating value={data.avg_rating} count={data.review_count} />
            <span className="inline-flex items-center gap-1">
              <Users className="size-4" /> {data.enrollment_count ?? 0} students
            </span>
            <span className="inline-flex items-center gap-1">
              <Layers className="size-4" /> {data.lesson_count ?? 0} lessons
            </span>
            <span className="inline-flex items-center gap-1">
              <Clock className="size-4" /> {duration(data.total_duration)}
            </span>
          </div>

          {data.description && <p className="whitespace-pre-line leading-relaxed">{data.description}</p>}
        </div>

        <aside>
          <Card className="overflow-hidden">
            <CourseCover url={data.cover_url} title={data.title} className="h-44 w-full object-cover" />
            <CardContent className="space-y-3 pt-2">
              <p className="text-3xl font-semibold">{money(data.price)}</p>
              {action()}
              {canManage && (
                <Link to={`/teach/courses/${courseId}`} className={buttonVariants({ variant: 'outline' }) + ' w-full'}>
                  Manage this course
                </Link>
              )}
              {data.language && <p className="text-sm text-muted-foreground">Language: {data.language}</p>}
            </CardContent>
          </Card>
        </aside>
      </div>

      {(data.learning_outcomes?.length ?? 0) > 0 && (
        <section className="space-y-3">
          <h2 className="text-xl font-semibold">What you will learn</h2>
          <ul className="grid gap-2 sm:grid-cols-2">
            {data.learning_outcomes?.map((item) => (
              <li key={item.id} className="flex gap-2 text-sm">
                <GraduationCap className="mt-0.5 size-4 shrink-0 text-primary" />
                {item.content}
              </li>
            ))}
          </ul>
        </section>
      )}

      {(data.requirements?.length ?? 0) > 0 && (
        <section className="space-y-3">
          <h2 className="text-xl font-semibold">Requirements</h2>
          <ul className="list-disc space-y-1 pl-5 text-sm">
            {data.requirements?.map((item) => (
              <li key={item.id}>{item.content}</li>
            ))}
          </ul>
        </section>
      )}

      <section className="space-y-3">
        <h2 className="text-xl font-semibold">Course content</h2>
        {(data.modules?.length ?? 0) === 0 ? (
          <p className="text-sm text-muted-foreground">The content is not added yet.</p>
        ) : (
          <Accordion multiple defaultValue={data.modules?.slice(0, 1).map((m) => m.id)} className="rounded-lg border px-4">
            {data.modules?.map((module) => (
              <AccordionItem key={module.id} value={module.id}>
                <AccordionTrigger>
                  <span className="flex-1 text-left">{module.title}</span>
                  <span className="mr-2 text-xs font-normal text-muted-foreground">{module.lessons?.length ?? 0} lessons</span>
                </AccordionTrigger>
                <AccordionContent>
                  <ul className="space-y-1">
                    {module.lessons?.map((lesson) => (
                      <li key={lesson.id} className="flex items-center gap-2 py-1 text-sm">
                        <span className="flex-1">{lesson.title}</span>
                        {lesson.is_preview && <Badge variant="secondary">Preview</Badge>}
                        <span className="text-xs text-muted-foreground">{duration(lesson.duration)}</span>
                      </li>
                    ))}
                  </ul>
                </AccordionContent>
              </AccordionItem>
            ))}
          </Accordion>
        )}
      </section>

      {instructor && (
        <section className="space-y-3">
          <h2 className="text-xl font-semibold">Your instructor</h2>
          <div className="flex gap-4">
            <Avatar className="size-14">
              <AvatarImage src={instructor.avatar_url} alt="" />
              <AvatarFallback>{fullName(instructor).slice(0, 2).toUpperCase()}</AvatarFallback>
            </Avatar>
            <div className="space-y-1">
              <p className="font-medium">{fullName(instructor)}</p>
              <Rating value={instructor.avg_rating} />
              {instructor.bio && <p className="text-sm text-muted-foreground">{instructor.bio}</p>}
            </div>
          </div>
        </section>
      )}

      <Reviews courseId={courseId} canReview={Boolean(enrollment) && hasRole(user, 'Student')} />
    </div>
  )
}
