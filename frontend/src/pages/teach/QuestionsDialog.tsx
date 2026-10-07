import { useQuery } from '@tanstack/react-query'
import { Check, Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { api, call } from '@/api/client'
import { ConfirmButton } from '@/components/ConfirmButton'
import { NativeSelect } from '@/components/NativeSelect'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useApiMutation } from '@/lib/mutations'

type QuestionType = 'single_choice' | 'multiple_choice' | 'true_false'
type Option = { option_text: string; is_correct: boolean }
type Question = { id?: string; text?: string; type?: QuestionType; options?: { option_text?: string; is_correct?: boolean }[] }

const typeLabels: Record<QuestionType, string> = {
  single_choice: 'One right answer',
  multiple_choice: 'Several right answers',
  true_false: 'True or false',
}

const trueFalse: Option[] = [
  { option_text: 'True', is_correct: true },
  { option_text: 'False', is_correct: false },
]

const blank = (): Option[] => [
  { option_text: '', is_correct: true },
  { option_text: '', is_correct: false },
]

function QuestionForm({ initial, submitting, onSubmit, onCancel }: { initial?: Question; submitting: boolean; onSubmit: (value: { text: string; type: QuestionType; options: Option[] }) => void; onCancel?: () => void }) {
  const [text, setText] = useState(initial?.text ?? '')
  const [type, setType] = useState<QuestionType>(initial?.type ?? 'single_choice')
  const [options, setOptions] = useState<Option[]>(
    initial?.options?.map((o) => ({ option_text: o.option_text ?? '', is_correct: Boolean(o.is_correct) })) ?? blank(),
  )

  function changeType(next: QuestionType) {
    setType(next)

    if (next === 'true_false') setOptions(trueFalse)
    else if (type === 'true_false') setOptions(blank())
    else if (next === 'single_choice') {
      // exactly one right option: keep the first one that was right
      const first = options.findIndex((o) => o.is_correct)
      setOptions(options.map((o, i) => ({ ...o, is_correct: i === Math.max(first, 0) })))
    }
  }

  function setCorrect(index: number, checked: boolean) {
    setOptions(options.map((o, i) => (type === 'multiple_choice' ? (i === index ? { ...o, is_correct: checked } : o) : { ...o, is_correct: i === index })))
  }

  function submit(e: React.FormEvent) {
    e.preventDefault()

    const filled = type === 'true_false' ? options : options.filter((o) => o.option_text.trim())

    if (filled.length < 2) return toast.error('A question needs at least two options')
    if (!filled.some((o) => o.is_correct)) return toast.error('Mark the right answer')

    onSubmit({ text: text.trim(), type, options: filled.map((o) => ({ ...o, option_text: o.option_text.trim() })) })
  }

  return (
    <form onSubmit={submit} className="space-y-3 rounded-lg border p-3">
      <div className="grid gap-3 sm:grid-cols-[1fr_12rem]">
        <div className="space-y-1.5">
          <Label htmlFor="question-text">Question</Label>
          <Input id="question-text" value={text} onChange={(e) => setText(e.target.value)} required />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="question-type">Type</Label>
          <NativeSelect id="question-type" value={type} onChange={(e) => changeType(e.target.value as QuestionType)}>
            {Object.entries(typeLabels).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </NativeSelect>
        </div>
      </div>

      <fieldset className="space-y-2">
        <legend className="text-sm font-medium">Options (mark the right ones)</legend>
        {options.map((option, index) => (
          <div key={index} className="flex items-center gap-2">
            <input
              type={type === 'multiple_choice' ? 'checkbox' : 'radio'}
              name="correct"
              checked={option.is_correct}
              onChange={(e) => setCorrect(index, e.target.checked)}
              className="size-4"
              aria-label={`Option ${index + 1} is right`}
            />
            <Input
              value={option.option_text}
              readOnly={type === 'true_false'}
              onChange={(e) => setOptions(options.map((o, i) => (i === index ? { ...o, option_text: e.target.value } : o)))}
              aria-label={`Option ${index + 1}`}
            />
            {type !== 'true_false' && options.length > 2 && (
              <Button type="button" size="icon-sm" variant="ghost" aria-label="Remove the option" onClick={() => setOptions(options.filter((_, i) => i !== index))}>
                <Trash2 />
              </Button>
            )}
          </div>
        ))}
        {type !== 'true_false' && (
          <Button type="button" size="sm" variant="outline" onClick={() => setOptions([...options, { option_text: '', is_correct: false }])}>
            <Plus /> Add an option
          </Button>
        )}
      </fieldset>

      <div className="flex gap-2">
        <Button type="submit" disabled={submitting || !text.trim()}>
          {submitting ? 'Saving…' : initial ? 'Save the question' : 'Add the question'}
        </Button>
        {onCancel && (
          <Button type="button" variant="ghost" onClick={onCancel}>
            Cancel
          </Button>
        )}
      </div>
    </form>
  )
}

/** The questions of a quiz: see, add, change, delete. */
export function QuestionsDialog({ quizId, title, onClose }: { quizId: string; title: string; onClose: () => void }) {
  const [editing, setEditing] = useState<string | null>(null)
  // a new key empties the form of a new question after one was added
  const [formKey, setFormKey] = useState(0)
  const key = ['questions', quizId]

  const questions = useQuery({
    queryKey: key,
    queryFn: () => call(api.GET('/quizzes/{quizId}/questions', { params: { path: { quizId } } })),
  })

  const add = useApiMutation(
    (body: { text: string; type: QuestionType; options: Option[] }) => call(api.POST('/quizzes/{quizId}/questions', { params: { path: { quizId } }, body })),
    { success: 'Question added', invalidate: [key], onSuccess: () => setFormKey((n) => n + 1) },
  )
  const update = useApiMutation(
    (input: { id: string; body: { text: string; type: QuestionType; options: Option[] } }) =>
      call(api.PUT('/questions/{questionId}', { params: { path: { questionId: input.id } }, body: input.body })),
    { success: 'Question saved', invalidate: [key], onSuccess: () => setEditing(null) },
  )
  const remove = useApiMutation((id: string) => call(api.DELETE('/questions/{questionId}', { params: { path: { questionId: id } } })), {
    success: 'Question deleted',
    invalidate: [key],
  })

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Questions: {title}</DialogTitle>
          <DialogDescription>Students get the questions and the options in a random order.</DialogDescription>
        </DialogHeader>

        {questions.isPending && <LoadingBlock rows={2} />}
        {questions.isError && <ErrorBlock error={questions.error} />}
        {questions.data?.length === 0 && <p className="text-sm text-muted-foreground">No questions yet.</p>}

        <ol className="space-y-3">
          {questions.data?.map((question, index) =>
            editing === question.id ? (
              <li key={question.id}>
                <QuestionForm initial={question} submitting={update.isPending} onCancel={() => setEditing(null)} onSubmit={(body) => update.mutate({ id: question.id!, body })} />
              </li>
            ) : (
              <li key={question.id} className="space-y-2 rounded-lg border p-3">
                <div className="flex items-start gap-2">
                  <p className="flex-1 font-medium">
                    {index + 1}. {question.text}
                  </p>
                  <Badge variant="secondary">{typeLabels[question.type as QuestionType]}</Badge>
                  <Button size="xs" variant="outline" onClick={() => setEditing(question.id!)}>
                    Edit
                  </Button>
                  <ConfirmButton size="xs" title="Delete this question?" onConfirm={() => remove.mutate(question.id!)} pending={remove.isPending}>
                    Delete
                  </ConfirmButton>
                </div>
                <ul className="space-y-1 text-sm">
                  {question.options?.map((option) => (
                    <li key={option.id} className="flex items-center gap-2">
                      {option.is_correct ? <Check className="size-4 text-green-600" aria-label="Right" /> : <span className="size-4" />}
                      <span className={option.is_correct ? 'font-medium' : ''}>{option.option_text}</span>
                    </li>
                  ))}
                </ul>
              </li>
            ),
          )}
        </ol>

        <QuestionForm key={formKey} submitting={add.isPending} onSubmit={(body) => add.mutate(body)} />
      </DialogContent>
    </Dialog>
  )
}
