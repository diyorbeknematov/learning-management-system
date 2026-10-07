import { useQueries, useQuery } from '@tanstack/react-query'
import { ListChecks, Pencil, Plus } from 'lucide-react'
import { useState } from 'react'
import { api, call } from '@/api/client'
import type { components } from '@/api/schema'
import { ConfirmButton } from '@/components/ConfirmButton'
import { NativeSelect } from '@/components/NativeSelect'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useApiMutation } from '@/lib/mutations'
import { QuestionsDialog } from './QuestionsDialog'

type Quiz = components['schemas']['models.Quiz']

function QuizDialog({ courseId, modules, quiz, onClose }: { courseId: string; modules: { id?: string; title?: string }[]; quiz?: Quiz; onClose: () => void }) {
  const [scope, setScope] = useState('course')
  const [title, setTitle] = useState(quiz?.title ?? '')
  const [description, setDescription] = useState(quiz?.description ?? '')
  const [timeLimit, setTimeLimit] = useState(String(quiz?.time_limit ?? 30))
  const [threshold, setThreshold] = useState(String(quiz?.pass_threshold ?? 70))
  const [attempts, setAttempts] = useState(String(quiz?.max_attempts ?? 3))

  const save = useApiMutation(
    () => {
      const body = {
        title: title.trim(),
        description: description.trim(),
        time_limit: Number(timeLimit),
        pass_threshold: Number(threshold),
        max_attempts: Number(attempts),
      }

      if (quiz) return call(api.PUT('/quizzes/{quizId}', { params: { path: { quizId: quiz.id! } }, body }))

      return scope === 'course'
        ? call(api.POST('/courses/{courseId}/quizzes', { params: { path: { courseId } }, body }))
        : call(api.POST('/modules/{moduleId}/quizzes', { params: { path: { moduleId: scope } }, body }))
    },
    { success: quiz ? 'Quiz saved' : 'Quiz created', invalidate: [['quizzes']], onSuccess: onClose },
  )

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <form
          className="grid gap-4"
          onSubmit={(e) => {
            e.preventDefault()
            save.mutate(undefined)
          }}
        >
          <DialogHeader>
            <DialogTitle>{quiz ? 'Edit the quiz' : 'New quiz'}</DialogTitle>
          </DialogHeader>

          {!quiz && (
            <div className="space-y-1.5">
              <Label htmlFor="quiz-scope">Where</Label>
              <NativeSelect id="quiz-scope" value={scope} onChange={(e) => setScope(e.target.value)}>
                <option value="course">Final quiz of the course</option>
                {modules.map((module) => (
                  <option key={module.id} value={module.id}>
                    Module: {module.title}
                  </option>
                ))}
              </NativeSelect>
            </div>
          )}

          <div className="space-y-1.5">
            <Label htmlFor="quiz-title">Title</Label>
            <Input id="quiz-title" value={title} onChange={(e) => setTitle(e.target.value)} required autoFocus />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="quiz-description">Description (optional)</Label>
            <Input id="quiz-description" value={description} onChange={(e) => setDescription(e.target.value)} />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="quiz-time">Minutes</Label>
              <Input id="quiz-time" type="number" min={1} value={timeLimit} onChange={(e) => setTimeLimit(e.target.value)} required />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="quiz-threshold">Pass at %</Label>
              <Input id="quiz-threshold" type="number" min={0} max={100} value={threshold} onChange={(e) => setThreshold(e.target.value)} required />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="quiz-attempts">Attempts</Label>
              <Input id="quiz-attempts" type="number" min={1} value={attempts} onChange={(e) => setAttempts(e.target.value)} required />
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={save.isPending || !title.trim()}>
              {save.isPending ? 'Saving…' : 'Save'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function QuizRow({ quiz, onEdit, onQuestions }: { quiz: Quiz; onEdit: () => void; onQuestions: () => void }) {
  const remove = useApiMutation(() => call(api.DELETE('/quizzes/{quizId}', { params: { path: { quizId: quiz.id! } } })), {
    success: 'Quiz deleted',
    invalidate: [['quizzes']],
  })

  return (
    <li className="flex flex-wrap items-center gap-2 px-3 py-2 text-sm">
      <span className="min-w-0 flex-1 font-medium">{quiz.title}</span>
      <Badge variant="secondary">{quiz.time_limit} min</Badge>
      <Badge variant="secondary">pass {quiz.pass_threshold}%</Badge>
      <Badge variant="secondary">{quiz.max_attempts} attempts</Badge>
      <Button size="xs" variant="outline" onClick={onQuestions}>
        <ListChecks /> Questions
      </Button>
      <Button size="icon-xs" variant="ghost" aria-label="Edit the quiz" onClick={onEdit}>
        <Pencil />
      </Button>
      <ConfirmButton size="xs" title="Delete this quiz?" description="Its questions and the attempts of students are deleted too." onConfirm={() => remove.mutate(undefined)} pending={remove.isPending}>
        Delete
      </ConfirmButton>
    </li>
  )
}

/** The quizzes of a course: the final one and the quizzes of its modules. */
export function QuizzesTab({ courseId }: { courseId: string }) {
  const [dialog, setDialog] = useState<{ quiz?: Quiz } | null>(null)
  const [questions, setQuestions] = useState<Quiz | null>(null)

  const course = useQuery({
    queryKey: ['course', courseId],
    queryFn: () => call(api.GET('/courses/{courseId}', { params: { path: { courseId } } })),
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

  if (course.isPending || courseQuizzes.isPending) return <LoadingBlock rows={2} />
  if (course.isError) return <ErrorBlock error={course.error} />
  if (courseQuizzes.isError) return <ErrorBlock error={courseQuizzes.error} />

  const list = (title: string, quizzes?: Quiz[]) => (
    <section className="space-y-2" key={title}>
      <h3 className="font-medium">{title}</h3>
      {quizzes?.length ? (
        <ul className="divide-y rounded-lg border">
          {quizzes.map((quiz) => (
            <QuizRow key={quiz.id} quiz={quiz} onEdit={() => setDialog({ quiz })} onQuestions={() => setQuestions(quiz)} />
          ))}
        </ul>
      ) : (
        <p className="text-sm text-muted-foreground">No quiz.</p>
      )}
    </section>
  )

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground">
          A student gets the certificate when all lessons are done and the final quiz (if there is one) is passed.
        </p>
        <Button onClick={() => setDialog({})}>
          <Plus /> New quiz
        </Button>
      </div>

      {list('Final quiz of the course', courseQuizzes.data)}
      {modules.map((module, index) => list(`Module: ${module.title}`, moduleQuizzes[index]?.data))}

      {dialog && <QuizDialog courseId={courseId} modules={modules} quiz={dialog.quiz} onClose={() => setDialog(null)} />}
      {questions && <QuestionsDialog quizId={questions.id!} title={questions.title ?? ''} onClose={() => setQuestions(null)} />}
    </div>
  )
}
