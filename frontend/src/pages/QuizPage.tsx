import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, Clock, RotateCcw, Target, XCircle } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router'
import { toast } from 'sonner'
import { api, call } from '@/api/client'
import type { components } from '@/api/schema'
import { useAuth } from '@/auth/context'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { formatDate } from '@/lib/format'
import { errorMessage } from '@/lib/query'

type AttemptView = components['schemas']['models.AttemptView']
type AttemptResult = components['schemas']['models.AttemptResult']
type Question = components['schemas']['models.Question']

function pad(n: number) {
  return String(n).padStart(2, '0')
}

/** Counts down to the end of the attempt; calls onEnd once when the time is up. */
function Countdown({ endsAt, onEnd }: { endsAt: number; onEnd: () => void }) {
  const [left, setLeft] = useState(() => Math.max(0, endsAt - Date.now()))

  useEffect(() => {
    const timer = setInterval(() => {
      const remaining = Math.max(0, endsAt - Date.now())

      setLeft(remaining)

      if (remaining === 0) {
        clearInterval(timer)
        onEnd()
      }
    }, 1000)

    return () => clearInterval(timer)
  }, [endsAt, onEnd])

  const seconds = Math.ceil(left / 1000)

  return (
    <span className={left < 60_000 ? 'font-medium text-destructive' : 'font-medium'} role="timer" aria-label="Time left">
      {pad(Math.floor(seconds / 60))}:{pad(seconds % 60)}
    </span>
  )
}

function Taking({
  attempt,
  timeLimit,
  onSubmitted,
}: {
  attempt: AttemptView
  timeLimit: number
  onSubmitted: (result: AttemptResult, questions: Question[]) => void
}) {
  const questions = useMemo(() => attempt.questions ?? [], [attempt])
  const [answers, setAnswers] = useState<Record<string, string[]>>({})
  const [endsAt] = useState(() => new Date(attempt.started_at ?? Date.now()).getTime() + timeLimit * 60_000)

  const submit = useMutation({
    mutationFn: () =>
      call(
        api.POST('/attempts/{attemptId}/submit', {
          params: { path: { attemptId: attempt.id! } },
          body: { answers: questions.map((q) => ({ question_id: q.id!, option_ids: answers[q.id!] ?? [] })) },
        }),
      ),
    onSuccess: (result) => onSubmitted(result, questions),
    onError: (error) => toast.error(errorMessage(error)),
  })

  function choose(question: Question, optionId: string) {
    setAnswers((current) => {
      const selected = current[question.id!] ?? []

      if (question.type === 'multiple_choice') {
        return { ...current, [question.id!]: selected.includes(optionId) ? selected.filter((id) => id !== optionId) : [...selected, optionId] }
      }

      return { ...current, [question.id!]: [optionId] }
    })
  }

  const answered = questions.filter((q) => (answers[q.id!]?.length ?? 0) > 0).length

  return (
    <form
      className="space-y-6"
      onSubmit={(e) => {
        e.preventDefault()
        submit.mutate()
      }}
    >
      <div className="sticky top-0 z-10 flex items-center justify-between rounded-lg border bg-background p-3">
        <span className="text-sm">
          Answered {answered} of {questions.length}
        </span>
        <Countdown endsAt={endsAt} onEnd={() => submit.mutate()} />
      </div>

      {questions.map((question, index) => (
        <fieldset key={question.id} className="space-y-3 rounded-lg border p-4">
          <legend className="px-1 font-medium">
            {index + 1}. {question.text}
          </legend>
          {question.type === 'multiple_choice' && <p className="text-xs text-muted-foreground">Choose all that apply.</p>}

          {question.options?.map((option) => {
            const multiple = question.type === 'multiple_choice'

            return (
              <label key={option.id} className="flex cursor-pointer items-center gap-3 rounded-md px-2 py-1.5 hover:bg-muted">
                <input
                  type={multiple ? 'checkbox' : 'radio'}
                  name={question.id}
                  checked={answers[question.id!]?.includes(option.id!) ?? false}
                  onChange={() => choose(question, option.id!)}
                  className="size-4"
                />
                <span className="text-sm">{option.option_text}</span>
              </label>
            )
          })}
        </fieldset>
      ))}

      <Button type="submit" disabled={submit.isPending}>
        {submit.isPending ? 'Sending…' : 'Finish the quiz'}
      </Button>
    </form>
  )
}

function Result({ result, questions, threshold }: { result: AttemptResult; questions: Question[]; threshold: number }) {
  const byQuestion = new Map(result.results?.map((r) => [r.question_id, r]))

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-3">
            {result.passed ? <CheckCircle2 className="size-6 text-green-600" /> : <XCircle className="size-6 text-destructive" />}
            {result.passed ? 'You passed' : 'You did not pass'}
          </CardTitle>
          <CardDescription>
            Your score: <strong>{result.score}%</strong> (to pass: {threshold}%)
          </CardDescription>
        </CardHeader>
      </Card>

      <ol className="space-y-2">
        {questions.map((question, index) => {
          const correct = byQuestion.get(question.id)?.correct

          return (
            <li key={question.id} className="flex items-start gap-2 text-sm">
              {correct ? (
                <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-green-600" aria-label="Correct" />
              ) : (
                <XCircle className="mt-0.5 size-4 shrink-0 text-destructive" aria-label="Wrong" />
              )}
              <span>
                {index + 1}. {question.text}
              </span>
            </li>
          )
        })}
      </ol>
    </div>
  )
}

export default function QuizPage() {
  const { quizId = '' } = useParams()
  const { user } = useAuth()
  const client = useQueryClient()
  const isStudent = user?.role_name === 'Student'

  const [active, setActive] = useState<AttemptView | null>(null)
  const [finished, setFinished] = useState<{ result: AttemptResult; questions: Question[] } | null>(null)

  const quiz = useQuery({
    queryKey: ['quiz', quizId],
    queryFn: () => call(api.GET('/quizzes/{quizId}', { params: { path: { quizId } } })),
  })

  const attempts = useQuery({
    queryKey: ['attempts', quizId],
    queryFn: () => call(api.GET('/quizzes/{quizId}/attempts', { params: { path: { quizId } } })),
  })

  const start = useMutation({
    mutationFn: () => call(api.POST('/quizzes/{quizId}/attempts', { params: { path: { quizId } } })),
    onSuccess: (attempt) => {
      setFinished(null)
      setActive(attempt)
    },
    onError: (error) => toast.error(errorMessage(error)),
  })

  if (quiz.isPending) return <LoadingBlock rows={3} />
  if (quiz.isError) return <ErrorBlock error={quiz.error} />

  const data = quiz.data
  const used = attempts.data?.length ?? 0
  const left = (data.max_attempts ?? 0) - used
  const passed = attempts.data?.some((attempt) => attempt.passed)

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <Card className="gap-0 overflow-hidden py-0">
        <div className="bg-gradient-to-r from-primary to-indigo-500 px-6 py-8 text-primary-foreground">
          <p className="text-xs font-semibold uppercase tracking-widest opacity-80">Quiz</p>
          <h1 className="text-3xl font-bold tracking-tight">{data.title}</h1>
          {data.description && <p className="mt-1 opacity-90">{data.description}</p>}
        </div>
        <div className="grid grid-cols-3 divide-x border-b text-center">
          {[
            [Clock, `${data.time_limit} min`, 'Time'],
            [Target, `${data.pass_threshold}%`, 'To pass'],
            [RotateCcw, `${used} of ${data.max_attempts}`, 'Attempts used'],
          ].map(([Icon, value, label]) => {
            const I = Icon as typeof Clock

            return (
              <div key={label as string} className="space-y-0.5 px-2 py-4">
                <I className="mx-auto size-5 text-primary" />
                <p className="text-lg font-semibold">{value as string}</p>
                <p className="text-xs text-muted-foreground">{label as string}</p>
              </div>
            )
          })}
        </div>
        {passed && (
          <p className="bg-emerald-50 px-6 py-3 text-sm font-medium text-emerald-800 dark:bg-emerald-500/10 dark:text-emerald-300">
            Passed: you can go on with the course.
          </p>
        )}
      </Card>

      {active ? (
        <Taking
          attempt={active}
          timeLimit={data.time_limit ?? 0}
          onSubmitted={(result, questions) => {
            setActive(null)
            setFinished({ result, questions })
            client.invalidateQueries({ queryKey: ['attempts', quizId] })
            client.invalidateQueries({ queryKey: ['progress'] })
            client.invalidateQueries({ queryKey: ['enrollments'] })
            client.invalidateQueries({ queryKey: ['certificates'] })
          }}
        />
      ) : (
        <>
          {finished && <Result result={finished.result} questions={finished.questions} threshold={data.pass_threshold ?? 0} />}

          {isStudent && (
            <Button disabled={start.isPending || (left <= 0 && !attempts.data?.some((a) => !a.completed_at))} onClick={() => start.mutate()}>
              {start.isPending ? 'Starting…' : attempts.data?.some((a) => !a.completed_at) ? 'Continue the attempt' : used > 0 ? 'Try again' : 'Start the quiz'}
            </Button>
          )}
          {isStudent && left <= 0 && !attempts.data?.some((a) => !a.completed_at) && (
            <p className="text-sm text-muted-foreground">You have no attempts left.</p>
          )}

          {attempts.isError && <ErrorBlock error={attempts.error} />}

          {(attempts.data?.length ?? 0) > 0 && (
            <section className="space-y-2">
              <h2 className="font-medium">Your attempts</h2>
              <ul className="divide-y rounded-lg border">
                {attempts.data?.map((attempt) => (
                  <li key={attempt.id} className="flex items-center gap-3 px-4 py-2 text-sm">
                    <span className="w-16">#{attempt.attempt_number}</span>
                    <span className="flex-1 text-muted-foreground">{formatDate(attempt.started_at)}</span>
                    {attempt.completed_at ? (
                      <>
                        <span>{attempt.score}%</span>
                        <Badge variant={attempt.passed ? 'default' : 'outline'}>{attempt.passed ? 'Passed' : 'Failed'}</Badge>
                      </>
                    ) : (
                      <Badge variant="outline">In progress</Badge>
                    )}
                  </li>
                ))}
              </ul>
            </section>
          )}
        </>
      )}

      {!isStudent && (
        <Card>
          <CardContent className="pt-4 text-sm text-muted-foreground">Only students take quizzes.</CardContent>
        </Card>
      )}
    </div>
  )
}
