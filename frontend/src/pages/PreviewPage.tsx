import { useQuery } from '@tanstack/react-query'
import { ArrowLeft, Lock, PlayCircle } from 'lucide-react'
import { Link, Navigate, useMatch, useParams } from 'react-router'
import { api, call } from '@/api/client'
import { useAuth } from '@/auth/context'
import { CourseAction, useEnrollment } from '@/components/CourseAction'
import { LessonMaterials } from '@/components/LessonMaterials'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { Badge } from '@/components/ui/badge'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { duration, money } from '@/lib/format'
import { cn } from '@/lib/utils'

// A lesson that is open to everybody (a free preview), with the content of the
// course beside it: the visitor sees what is inside before they enroll.
export default function PreviewPage() {
  const { courseId = '', lessonId = '' } = useParams()
  const { user } = useAuth()
  const enrollment = useEnrollment(courseId)
  const teach = Boolean(useMatch('/teach/courses/:courseId/preview/:lessonId'))
  const base = teach ? `/teach/courses/${courseId}/preview` : `/courses/${courseId}/preview`

  const course = useQuery({
    queryKey: ['course', courseId],
    queryFn: () => call(api.GET('/courses/{courseId}', { params: { path: { courseId } } })),
  })

  if (course.isPending) return <LoadingBlock rows={4} />
  if (course.isError) return <ErrorBlock error={course.error} />

  // those who study the course have their own page
  if (enrollment && !teach) return <Navigate to={`/learn/${courseId}/lessons/${lessonId}`} replace />

  const data = course.data
  const canManage = Boolean(user && (user.role_name === 'SuperAdmin' || user.id === data.instructor_id))
  const lessons = data.modules?.flatMap((module) => module.lessons ?? []) ?? []
  const current = lessons.find((lesson) => lesson.id === lessonId)

  return (
    <div className="space-y-6">
      {teach && (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-primary/20 bg-primary/5 px-4 py-3 text-sm">
          <span className="text-muted-foreground">This is how a student sees your lesson.</span>
          <Link to={`/teach/courses/${courseId}`} className={buttonVariants({ size: 'sm' })}>
            <ArrowLeft /> Back to the course editor
          </Link>
        </div>
      )}
      <Link
        to={teach ? `/teach/courses/${courseId}/preview` : `/courses/${courseId}`}
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="size-4" /> {data.title}
      </Link>

      <div className="grid gap-8 lg:grid-cols-[20rem_1fr]">
        <aside className="space-y-4">
          <nav aria-label="Course content" className="space-y-4">
            {data.modules?.map((module) => (
              <div key={module.id} className="space-y-1">
                <h2 className="text-sm font-medium">{module.title}</h2>
                <ul>
                  {module.lessons?.map((lesson) => {
                    const open = lesson.is_preview || canManage

                    return (
                      <li key={lesson.id}>
                        {open ? (
                          <Link
                            to={`${base}/${lesson.id}`}
                            className={cn(
                              'flex items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-muted',
                              lesson.id === lessonId && 'bg-muted font-medium',
                            )}
                          >
                            <PlayCircle className="size-4 shrink-0 text-primary" />
                            <span className="flex-1">{lesson.title}</span>
                            <span className="text-xs text-muted-foreground">{duration(lesson.duration)}</span>
                          </Link>
                        ) : (
                          <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground">
                            <Lock className="size-4 shrink-0" aria-label="Locked" />
                            <span className="flex-1">{lesson.title}</span>
                            <span className="text-xs">{duration(lesson.duration)}</span>
                          </div>
                        )}
                      </li>
                    )
                  })}
                </ul>
              </div>
            ))}
          </nav>
        </aside>

        <section className="min-w-0 space-y-6">
          {!current ? (
            <p className="text-muted-foreground">This lesson was not found in the course.</p>
          ) : (
            <>
              <div className="flex flex-wrap items-center gap-3">
                <h1 className="flex-1 text-2xl font-semibold">{current.title}</h1>
                {current.is_preview && <Badge variant="secondary">Free preview</Badge>}
              </div>

              <LessonMaterials lessonId={current.id!} />
            </>
          )}

          {!canManage && (
            <Card className="bg-gradient-to-br from-primary/10 to-background">
              <CardHeader>
                <CardTitle>Like what you see?</CardTitle>
                <CardDescription>
                  Enroll to open all {data.lesson_count ?? lessons.length} lessons, take the quizzes and earn the certificate.
                  {data.price ? ` The course costs ${money(data.price)}.` : ' The course is free.'}
                </CardDescription>
              </CardHeader>
              <CardContent className="max-w-xs">
                <CourseAction courseId={courseId} price={data.price} from={`/courses/${courseId}/preview/${lessonId}`} />
              </CardContent>
            </Card>
          )}
        </section>
      </div>
    </div>
  )
}
