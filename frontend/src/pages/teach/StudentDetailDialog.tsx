import { useQueries, useQuery } from '@tanstack/react-query'
import { CheckCircle2, Circle } from 'lucide-react'
import { api, call } from '@/api/client'
import type { components } from '@/api/schema'
import { StatusBadge } from '@/components/StatusBadge'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Progress } from '@/components/ui/progress'
import { formatDate } from '@/lib/format'

type Quiz = components['schemas']['models.Quiz']
type Student = components['schemas']['models.StudentProgress']

/** One student of the course: the lessons done and left, and the result of every quiz. */
export function StudentDetailDialog({ courseId, student, onClose }: { courseId: string; student: Student; onClose: () => void }) {
  const studentId = student.student_id!

  const course = useQuery({
    queryKey: ['course', courseId],
    queryFn: () => call(api.GET('/courses/{courseId}', { params: { path: { courseId } } })),
  })
  const progress = useQuery({
    queryKey: ['students', courseId, studentId, 'progress'],
    queryFn: () => call(api.GET('/courses/{courseId}/students/{studentId}/progress', { params: { path: { courseId, studentId } } })),
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

  const quizzes: Quiz[] = [...(courseQuizzes.data ?? []), ...moduleQuizzes.flatMap((query) => query.data ?? [])]

  const attempts = useQueries({
    queries: quizzes.map((quiz) => ({
      queryKey: ['attempts', quiz.id],
      queryFn: () => call(api.GET('/quizzes/{quizId}/attempts', { params: { path: { quizId: quiz.id! } } })),
    })),
  })

  const done = new Set(progress.data?.completed_lesson_ids ?? [])
  const percent = progress.data?.progress_percent ?? 0

  // best finished attempt of this student for every quiz
  const results = quizzes.map((quiz, index) => {
    const mine = (attempts[index]?.data ?? []).filter((attempt) => attempt.student_id === studentId && attempt.completed_at)
    const best = mine.length ? Math.max(...mine.map((attempt) => attempt.score ?? 0)) : null

    return { quiz, mine, best, passed: best !== null && best >= (quiz.pass_threshold ?? 0) }
  })
  const taken = results.filter((result) => result.best !== null)
  const average = taken.length ? Math.round(taken.reduce((sum, result) => sum + (result.best ?? 0), 0) / taken.length) : null

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[92vh] overflow-y-auto sm:max-w-5xl">
        <DialogHeader>
          <div className="flex items-center gap-4">
            <Avatar className="size-14">
              <AvatarFallback className="bg-primary/10 text-lg font-medium text-primary">{student.full_name?.slice(0, 2).toUpperCase()}</AvatarFallback>
            </Avatar>
            <div className="min-w-0">
              <DialogTitle className="text-xl">{student.full_name}</DialogTitle>
              <DialogDescription>
                {student.email} · last activity {formatDate(student.last_activity_at)}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        {(course.isPending || progress.isPending) && <LoadingBlock rows={3} />}
        {course.isError && <ErrorBlock error={course.error} />}
        {progress.isError && <ErrorBlock error={progress.error} />}

        {course.data && progress.data && (
          <div className="space-y-6">
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              <Tile label="Course progress" value={`${Math.round(percent)}%`}>
                <Progress value={percent} aria-label="Course progress" />
              </Tile>
              <Tile label="Lessons done" value={`${progress.data.completed_lessons ?? 0} / ${progress.data.total_lessons ?? 0}`} />
              <Tile label="Quizzes passed" value={`${results.filter((result) => result.passed).length} / ${results.length}`} />
              <Tile label="Average quiz score" value={average === null ? '–' : `${average}%`} />
            </div>

            <div className="grid gap-6 lg:grid-cols-2">
              <section className="space-y-3" aria-label="Lessons">
                <h3 className="font-semibold">Lessons</h3>
                {modules.map((module, index) => {
                  const lessons = module.lessons ?? []
                  const finished = lessons.filter((lesson) => done.has(lesson.id!)).length

                  return (
                    <div key={module.id} className="overflow-hidden rounded-xl border">
                      <div className="flex items-center justify-between gap-3 border-b bg-muted/40 px-4 py-2.5">
                        <h4 className="truncate text-sm font-medium">
                          {index + 1}. {module.title}
                        </h4>
                        <span className="shrink-0 text-xs text-muted-foreground">
                          {finished} / {lessons.length}
                        </span>
                      </div>
                      <ul className="divide-y">
                        {lessons.map((lesson) => (
                          <li key={lesson.id} className="flex items-center gap-2.5 px-4 py-2 text-sm">
                            {done.has(lesson.id!) ? (
                              <CheckCircle2 className="size-4 shrink-0 text-green-600" aria-label="Done" />
                            ) : (
                              <Circle className="size-4 shrink-0 text-muted-foreground" aria-label="Not done" />
                            )}
                            <span className={done.has(lesson.id!) ? '' : 'text-muted-foreground'}>{lesson.title}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  )
                })}
              </section>

              <section className="space-y-3" aria-label="Quiz results">
                <h3 className="font-semibold">Quiz results</h3>
                {results.length === 0 && <p className="text-sm text-muted-foreground">This course has no quizzes.</p>}
                {results.map(({ quiz, mine, best, passed }) => (
                  <div key={quiz.id} className="space-y-3 rounded-xl border p-4">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <span className="font-medium">{quiz.title}</span>
                      {best === null ? <StatusBadge value="pending" label="Not taken" /> : <StatusBadge value={passed ? 'passed' : 'failed'} />}
                    </div>

                    {best !== null && (
                      <div className="space-y-1">
                        <div className="relative h-2 overflow-hidden rounded-full bg-muted">
                          <div className={`h-full rounded-full ${passed ? 'bg-green-500' : 'bg-red-500'}`} style={{ width: `${Math.min(best, 100)}%` }} />
                          <div className="absolute inset-y-0 w-0.5 bg-foreground/60" style={{ left: `${quiz.pass_threshold ?? 0}%` }} title="Pass mark" />
                        </div>
                        <p className="text-xs text-muted-foreground">
                          Best {best}% · pass mark {quiz.pass_threshold}% · {mine.length} of {quiz.max_attempts} attempts used
                        </p>
                      </div>
                    )}

                    {mine.length > 0 && (
                      <ul className="divide-y rounded-lg border text-sm">
                        {mine.map((attempt) => (
                          <li key={attempt.id} className="flex items-center justify-between gap-3 px-3 py-1.5">
                            <span className="text-muted-foreground">
                              #{attempt.attempt_number} · {formatDate(attempt.completed_at)}
                            </span>
                            <span className={attempt.passed ? 'font-medium text-green-700' : 'font-medium text-red-700'}>{attempt.score}%</span>
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                ))}
              </section>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

function Tile({ label, value, children }: { label: string; value: string; children?: React.ReactNode }) {
  return (
    <div className="space-y-2 rounded-xl border bg-muted/30 p-4">
      <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</p>
      <p className="text-2xl font-semibold">{value}</p>
      {children}
    </div>
  )
}
