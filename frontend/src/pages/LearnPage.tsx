import { useMutation, useQuery, useQueries, useQueryClient } from '@tanstack/react-query'
import { Check, ChevronRight, Circle, FileText, ListChecks, PlayCircle } from 'lucide-react'
import { Link, Navigate, useNavigate, useParams } from 'react-router'
import { toast } from 'sonner'
import { api, call } from '@/api/client'
import { useMyEnrollments } from '@/api/queries'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { Badge } from '@/components/ui/badge'
import { Button, buttonVariants } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { duration } from '@/lib/format'
import { errorMessage } from '@/lib/query'
import { cn } from '@/lib/utils'

function Materials({ lessonId }: { lessonId: string }) {
  const materials = useQuery({
    queryKey: ['materials', lessonId],
    queryFn: () => call(api.GET('/lessons/{lessonId}/materials', { params: { path: { lessonId } } })),
  })

  if (materials.isPending) return <LoadingBlock rows={2} />
  if (materials.isError) return <ErrorBlock error={materials.error} />
  if (materials.data.length === 0) return <p className="text-sm text-muted-foreground">This lesson has no materials yet.</p>

  return (
    <div className="space-y-4">
      {materials.data.map((material) => (
        <article key={material.id} className="rounded-lg border p-4">
          {material.type === 'text' && <p className="whitespace-pre-line leading-relaxed">{material.content}</p>}

          {material.type === 'video' && (
            <a href={material.content} target="_blank" rel="noreferrer" className="inline-flex items-center gap-2 underline">
              <PlayCircle className="size-5" /> Watch the video
            </a>
          )}

          {material.type === 'file' && (
            <a href={material.file_url} target="_blank" rel="noreferrer" className="inline-flex items-center gap-2 underline">
              <FileText className="size-5" /> {material.file_name ?? 'Download the file'}
              {material.file_size ? (
                <span className="text-xs text-muted-foreground">({(material.file_size / 1024 / 1024).toFixed(1)} MB)</span>
              ) : null}
            </a>
          )}
        </article>
      ))}
    </div>
  )
}

// The place where a student studies: the lessons of the course on the left, the
// lesson with its materials on the right.
export default function LearnPage() {
  const { courseId = '', lessonId } = useParams()
  const navigate = useNavigate()
  const client = useQueryClient()
  const enrollments = useMyEnrollments()

  const course = useQuery({
    queryKey: ['course', courseId],
    queryFn: () => call(api.GET('/courses/{courseId}', { params: { path: { courseId } } })),
  })

  const progress = useQuery({
    queryKey: ['progress', courseId],
    queryFn: () => call(api.GET('/courses/{courseId}/progress', { params: { path: { courseId } } })),
  })

  const courseQuizzes = useQuery({
    queryKey: ['quizzes', 'course', courseId],
    queryFn: () => call(api.GET('/courses/{courseId}/quizzes', { params: { path: { courseId } } })),
  })

  const modules = course.data?.modules ?? []

  const moduleQuizzes = useQueries({
    queries: modules.map((module) => ({
      queryKey: ['quizzes', 'module', module.id],
      queryFn: () => call(api.GET('/modules/{moduleId}/quizzes', { params: { path: { moduleId: module.id! } } })),
    })),
  })

  const done = new Set(progress.data?.completed_lesson_ids ?? [])
  const lessons = modules.flatMap((module) => module.lessons ?? [])
  const enrollment = enrollments.data?.items?.find((item) => item.course_id === courseId)

  const toggle = useMutation({
    mutationFn: (input: { id: string; completed: boolean }) =>
      call(api.POST('/lessons/{lessonId}/progress', { params: { path: { lessonId: input.id } }, body: { completed: input.completed } })),
    onSuccess: (_, input) => {
      client.invalidateQueries({ queryKey: ['progress', courseId] })
      client.invalidateQueries({ queryKey: ['enrollments'] })
      client.invalidateQueries({ queryKey: ['certificates'] })

      const index = lessons.findIndex((lesson) => lesson.id === input.id)
      const next = lessons[index + 1]

      // after "done" the next lesson opens by itself
      if (input.completed && next) navigate(`/learn/${courseId}/lessons/${next.id}`)
    },
    onError: (error) => toast.error(errorMessage(error)),
  })

  if (course.isPending || progress.isPending) return <LoadingBlock rows={4} />
  if (course.isError) return <ErrorBlock error={course.error} />
  if (progress.isError) return <ErrorBlock error={progress.error} />

  // no lesson in the address: the first one that is not done
  if (!lessonId) {
    const first = lessons.find((lesson) => !done.has(lesson.id!)) ?? lessons[0]

    if (first) return <Navigate to={`/learn/${courseId}/lessons/${first.id}`} replace />
  }

  const current = lessons.find((lesson) => lesson.id === lessonId)
  const percent = progress.data.progress_percent ?? 0

  return (
    <div className="grid gap-8 lg:grid-cols-[20rem_1fr]">
      <aside className="space-y-4">
        <div className="space-y-2">
          <Link to={`/courses/${courseId}`} className="font-semibold hover:underline">
            {course.data.title}
          </Link>
          <Progress value={percent} aria-label="Course progress" />
          <p className="text-xs text-muted-foreground">
            {progress.data.completed_lessons ?? 0} of {progress.data.total_lessons ?? 0} lessons · {Math.round(percent)}%
          </p>
        </div>

        {enrollment?.status === 'completed' && (
          <Link to="/certificates" className={buttonVariants({ variant: 'outline' }) + ' w-full'}>
            Your certificate
          </Link>
        )}

        <nav aria-label="Lessons" className="space-y-4">
          {modules.map((module, moduleIndex) => (
            <div key={module.id} className="space-y-1">
              <h2 className="text-sm font-medium">{module.title}</h2>
              <ul>
                {module.lessons?.map((lesson) => (
                  <li key={lesson.id}>
                    <Link
                      to={`/learn/${courseId}/lessons/${lesson.id}`}
                      className={cn(
                        'flex items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-muted',
                        lesson.id === lessonId && 'bg-muted font-medium',
                      )}
                    >
                      {done.has(lesson.id!) ? (
                        <Check className="size-4 shrink-0 text-green-600" aria-label="Done" />
                      ) : (
                        <Circle className="size-4 shrink-0 text-muted-foreground/50" aria-label="Not done" />
                      )}
                      <span className="flex-1">{lesson.title}</span>
                      <span className="text-xs text-muted-foreground">{duration(lesson.duration)}</span>
                    </Link>
                  </li>
                ))}
                {moduleQuizzes[moduleIndex]?.data?.map((quiz) => (
                  <li key={quiz.id}>
                    <Link to={`/quizzes/${quiz.id}`} className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-muted">
                      <ListChecks className="size-4 shrink-0 text-primary" /> {quiz.title}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}

          {(courseQuizzes.data?.length ?? 0) > 0 && (
            <div className="space-y-1">
              <h2 className="text-sm font-medium">Final quiz</h2>
              <ul>
                {courseQuizzes.data?.map((quiz) => (
                  <li key={quiz.id}>
                    <Link to={`/quizzes/${quiz.id}`} className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-muted">
                      <ListChecks className="size-4 shrink-0 text-primary" /> {quiz.title}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </nav>
      </aside>

      <section className="min-w-0 space-y-6">
        {!current ? (
          <p className="text-muted-foreground">This course has no lessons yet.</p>
        ) : (
          <>
            <div className="flex flex-wrap items-center gap-3">
              <h1 className="flex-1 text-2xl font-semibold">{current.title}</h1>
              {current.is_preview && <Badge variant="secondary">Preview</Badge>}
            </div>

            <Materials lessonId={current.id!} />

            <div className="flex flex-wrap items-center gap-3 border-t pt-4">
              <Button
                disabled={toggle.isPending}
                variant={done.has(current.id!) ? 'outline' : 'default'}
                onClick={() => toggle.mutate({ id: current.id!, completed: !done.has(current.id!) })}
              >
                {done.has(current.id!) ? 'Mark as not done' : 'Mark as done'}
              </Button>

              {(() => {
                const next = lessons[lessons.findIndex((lesson) => lesson.id === current.id) + 1]

                return next ? (
                  <Link to={`/learn/${courseId}/lessons/${next.id}`} className={buttonVariants({ variant: 'ghost' })}>
                    Next lesson <ChevronRight />
                  </Link>
                ) : null
              })()}
            </div>
          </>
        )}
      </section>
    </div>
  )
}
