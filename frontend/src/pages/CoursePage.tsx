import { useQuery } from '@tanstack/react-query'
import { Clock, GraduationCap, Layers, Lock, PlayCircle, Users } from 'lucide-react'
import { Link, useParams } from 'react-router'
import { api, call } from '@/api/client'
import { hasRole, useAuth } from '@/auth/context'
import { CourseAction, useEnrollment } from '@/components/CourseAction'
import { CourseCover } from '@/components/CourseCard'
import { Rating } from '@/components/Rating'
import { Reviews } from '@/components/Reviews'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { capitalize, duration, fullName, money, plural } from '@/lib/format'

export default function CoursePage() {
  const { courseId = '' } = useParams()
  const { user } = useAuth()

  const course = useQuery({
    queryKey: ['course', courseId],
    queryFn: () => call(api.GET('/courses/{courseId}', { params: { path: { courseId } } })),
  })

  const enrollment = useEnrollment(courseId)

  if (course.isPending) return <LoadingBlock rows={4} />
  if (course.isError) return <ErrorBlock error={course.error} />

  const data = course.data
  const instructor = data.instructor
  const canManage = user && (user.role_name === 'SuperAdmin' || user.id === data.instructor_id)

  return (
    <div className="grid gap-10 lg:grid-cols-[minmax(0,1fr)_24rem] xl:grid-cols-[minmax(0,1fr)_28rem]">
      <div className="min-w-0 space-y-10">
        <div className="rounded-2xl bg-gradient-to-br from-primary/10 via-background to-background p-6 sm:p-8">
          <div className="space-y-4">
            <div className="flex flex-wrap gap-2">
              {data.category_name && <Badge variant="outline">{data.category_name}</Badge>}
              {data.difficulty && <Badge variant="secondary">{capitalize(data.difficulty)}</Badge>}
              {data.status === 'draft' && <Badge>Draft: only you can see it</Badge>}
            </div>

            <h1 className="text-3xl font-semibold leading-tight">{data.title}</h1>

            <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground">
              <Rating value={data.avg_rating} count={data.review_count} />
              <span className="inline-flex items-center gap-1">
                <Users className="size-4" /> {plural(data.enrollment_count, 'student')}
              </span>
              <span className="inline-flex items-center gap-1">
                <Layers className="size-4" /> {plural(data.lesson_count, 'lesson')}
              </span>
              <span className="inline-flex items-center gap-1">
                <Clock className="size-4" /> {duration(data.total_duration)}
              </span>
            </div>

            {data.description && <p className="whitespace-pre-line leading-relaxed">{data.description}</p>}
          </div>
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
            <Accordion multiple defaultValue={data.modules?.map((m) => m.id)} className="rounded-lg border px-4">
              {data.modules?.map((module) => (
                <AccordionItem key={module.id} value={module.id}>
                  <AccordionTrigger>
                    <span className="flex-1 text-left">{module.title}</span>
                    <span className="mr-2 text-xs font-normal text-muted-foreground">{plural(module.lessons?.length, 'lesson')}</span>
                  </AccordionTrigger>
                  <AccordionContent>
                    <ul className="space-y-1">
                      {module.lessons?.map((lesson) => {
                        const open = lesson.is_preview || Boolean(enrollment) || canManage
                        const target = enrollment ? `/learn/${courseId}/lessons/${lesson.id}` : `/courses/${courseId}/preview/${lesson.id}`

                        return (
                          <li key={lesson.id}>
                            {open ? (
                              <Link to={target} className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm no-underline! hover:bg-muted">
                                <PlayCircle className="size-4 shrink-0 text-primary" />
                                <span className="flex-1">{lesson.title}</span>
                                {lesson.is_preview && !enrollment && <Badge variant="secondary">Free preview</Badge>}
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

      <aside className="order-first space-y-5 lg:sticky lg:top-20 lg:order-none lg:self-start">
        <Card className="overflow-hidden pt-0 shadow-md">
          <CourseCover url={data.cover_url} title={data.title} className="h-44 w-full object-cover" />
          <CardContent className="space-y-3 pt-2">
            <p className="text-3xl font-semibold">{money(data.price)}</p>
            <CourseAction courseId={courseId} price={data.price} from={`/courses/${courseId}`} />
            {canManage && (
              <Link to={`/teach/courses/${courseId}`} className={buttonVariants({ variant: 'outline' }) + ' w-full'}>
                Manage this course
              </Link>
            )}
            {data.language && <p className="text-sm text-muted-foreground">Language: {data.language}</p>}
          </CardContent>
        </Card>

        {instructor && (
          <div className="space-y-3 px-1" aria-label="Instructor">
            <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Instructor</p>
            <div className="flex items-center gap-3">
              <Avatar className="size-12">
                <AvatarImage src={instructor.avatar_url} alt="" />
                <AvatarFallback>{fullName(instructor).slice(0, 2).toUpperCase()}</AvatarFallback>
              </Avatar>
              <div className="min-w-0 space-y-0.5">
                <p className="truncate font-medium">{fullName(instructor)}</p>
                <Rating value={instructor.avg_rating} />
              </div>
            </div>
            {instructor.bio && <p className="line-clamp-3 text-sm text-muted-foreground">{instructor.bio}</p>}
          </div>
        )}
      </aside>
    </div>
  )
}
