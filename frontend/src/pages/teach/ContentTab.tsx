import { useQuery } from '@tanstack/react-query'
import { ArrowDown, ArrowUp, Clock, Eye, FolderOpen, Layers, Pencil, PlayCircle, Plus } from 'lucide-react'
import { useState } from 'react'
import { api, call } from '@/api/client'
import { ConfirmButton } from '@/components/ConfirmButton'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { duration, plural } from '@/lib/format'
import { useApiMutation } from '@/lib/mutations'
import { MaterialsDialog } from './MaterialsDialog'

type Lesson = { id?: string; title?: string; duration?: number; is_preview?: boolean }

function LessonDialog({ moduleId, lesson, onClose, refresh }: { moduleId: string; lesson?: Lesson; onClose: () => void; refresh: string[] }) {
  const [title, setTitle] = useState(lesson?.title ?? '')
  const [minutes, setMinutes] = useState(String(lesson?.duration ?? 0))
  const [preview, setPreview] = useState(lesson?.is_preview ?? false)

  const save = useApiMutation(
    () => {
      const body = { title: title.trim(), duration: Number(minutes) || 0, is_preview: preview }

      return lesson
        ? call(api.PUT('/lessons/{lessonId}', { params: { path: { lessonId: lesson.id! } }, body }))
        : call(api.POST('/modules/{moduleId}/lessons', { params: { path: { moduleId } }, body }))
    },
    { success: lesson ? 'Lesson saved' : 'Lesson added', invalidate: [refresh], onSuccess: onClose },
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
            <DialogTitle>{lesson ? 'Edit the lesson' : 'New lesson'}</DialogTitle>
          </DialogHeader>
          <div className="space-y-1.5">
            <Label htmlFor="lesson-title">Title</Label>
            <Input id="lesson-title" value={title} onChange={(e) => setTitle(e.target.value)} required autoFocus />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="lesson-duration">Duration (minutes)</Label>
            <Input id="lesson-duration" type="number" min={0} value={minutes} onChange={(e) => setMinutes(e.target.value)} />
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={preview} onChange={(e) => setPreview(e.target.checked)} className="size-4" />
            Free preview: visitors can open this lesson
          </label>
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

function ModuleDialog({
  courseId,
  module,
  onClose,
  refresh,
}: {
  courseId: string
  module?: { id?: string; title?: string; description?: string }
  onClose: () => void
  refresh: string[]
}) {
  const [title, setTitle] = useState(module?.title ?? '')
  const [description, setDescription] = useState(module?.description ?? '')

  const save = useApiMutation(
    () => {
      const body = { title: title.trim(), description: description.trim() }

      return module
        ? call(api.PUT('/modules/{moduleId}', { params: { path: { moduleId: module.id! } }, body }))
        : call(api.POST('/courses/{courseId}/modules', { params: { path: { courseId } }, body }))
    },
    { success: module ? 'Module saved' : 'Module added', invalidate: [refresh], onSuccess: onClose },
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
            <DialogTitle>{module ? 'Edit the module' : 'New module'}</DialogTitle>
          </DialogHeader>
          <div className="space-y-1.5">
            <Label htmlFor="module-title">Title</Label>
            <Input id="module-title" value={title} onChange={(e) => setTitle(e.target.value)} required autoFocus />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="module-description">Description (optional)</Label>
            <Input id="module-description" value={description} onChange={(e) => setDescription(e.target.value)} />
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

type Editing =
  | { kind: 'module'; module?: { id?: string; title?: string; description?: string } }
  | { kind: 'lesson'; moduleId: string; lesson?: Lesson }
  | { kind: 'materials'; lesson: Lesson }
  | null

/** Modules and lessons of a course, in order, with their materials. */
export function ContentTab({ courseId }: { courseId: string }) {
  const [editing, setEditing] = useState<Editing>(null)
  const refresh = ['course', courseId]

  const course = useQuery({
    queryKey: ['course', courseId],
    queryFn: () => call(api.GET('/courses/{courseId}', { params: { path: { courseId } } })),
  })

  const moveModule = useApiMutation(
    (input: { id: string; order: number }) =>
      call(api.PATCH('/modules/{moduleId}/order', { params: { path: { moduleId: input.id } }, body: { order_number: input.order } })),
    { invalidate: [refresh] },
  )
  const deleteModule = useApiMutation((id: string) => call(api.DELETE('/modules/{moduleId}', { params: { path: { moduleId: id } } })), {
    success: 'Module deleted',
    invalidate: [refresh],
  })
  const moveLesson = useApiMutation(
    (input: { id: string; order: number }) =>
      call(api.PATCH('/lessons/{lessonId}/order', { params: { path: { lessonId: input.id } }, body: { order_number: input.order } })),
    { invalidate: [refresh] },
  )
  const deleteLesson = useApiMutation((id: string) => call(api.DELETE('/lessons/{lessonId}', { params: { path: { lessonId: id } } })), {
    success: 'Lesson deleted',
    invalidate: [refresh],
  })

  if (course.isPending) return <LoadingBlock rows={3} />
  if (course.isError) return <ErrorBlock error={course.error} />

  const modules = course.data.modules ?? []

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {plural(course.data.lesson_count, 'lesson')} in {plural(modules.length, 'module')}. A course needs at least one lesson before it can be published.
        </p>
        <Button onClick={() => setEditing({ kind: 'module' })}>
          <Plus /> Add a module
        </Button>
      </div>

      {modules.length === 0 && (
        <div className="rounded-xl border border-dashed bg-card p-10 text-center text-muted-foreground">
          <Layers className="mx-auto mb-2 size-8" />
          No modules yet. Add the first module, then put lessons in it.
        </div>
      )}

      {modules.map((module, moduleIndex) => (
        <Card key={module.id} className="gap-0 overflow-hidden py-0">
          <div className="flex items-center gap-3 border-b bg-muted/40 px-5 py-3.5">
            <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary text-sm font-semibold text-primary-foreground">
              {moduleIndex + 1}
            </span>
            <div className="min-w-0 flex-1">
              <h3 className="truncate text-base font-semibold">{module.title}</h3>
              <p className="text-xs text-muted-foreground">{plural(module.lessons?.length ?? 0, 'lesson')}</p>
            </div>
            <Button
              size="icon-sm"
              variant="ghost"
              aria-label="Move the module up"
              disabled={moduleIndex === 0 || moveModule.isPending}
              onClick={() => moveModule.mutate({ id: module.id!, order: moduleIndex })}
            >
              <ArrowUp />
            </Button>
            <Button
              size="icon-sm"
              variant="ghost"
              aria-label="Move the module down"
              disabled={moduleIndex === modules.length - 1 || moveModule.isPending}
              onClick={() => moveModule.mutate({ id: module.id!, order: moduleIndex + 2 })}
            >
              <ArrowDown />
            </Button>
            <Button size="icon-sm" variant="ghost" aria-label="Edit the module" onClick={() => setEditing({ kind: 'module', module })}>
              <Pencil />
            </Button>
            <ConfirmButton
              title="Delete this module?"
              description="Its lessons and their materials are deleted too."
              onConfirm={() => deleteModule.mutate(module.id!)}
              pending={deleteModule.isPending}
            >
              Delete
            </ConfirmButton>
          </div>

          <CardContent className="p-0">
            {module.lessons?.length === 0 && <p className="px-5 py-4 text-sm text-muted-foreground">No lessons yet.</p>}

            <ul className="divide-y">
              {module.lessons?.map((lesson, lessonIndex) => (
                <li key={lesson.id} className="flex flex-wrap items-center gap-3 px-5 py-3 text-sm transition hover:bg-muted/30">
                  <PlayCircle className="size-5 shrink-0 text-primary" />
                  <span className="min-w-0 flex-1 font-medium">
                    {lessonIndex + 1}. {lesson.title}
                  </span>
                  {lesson.is_preview && (
                    <Badge variant="secondary">
                      <Eye /> Free preview
                    </Badge>
                  )}
                  <span className="flex items-center gap-1 text-xs text-muted-foreground">
                    <Clock className="size-3.5" /> {duration(lesson.duration)}
                  </span>
                  <Button size="xs" variant="outline" onClick={() => setEditing({ kind: 'materials', lesson })}>
                    <FolderOpen /> Materials
                  </Button>
                  <div className="flex items-center">
                    <Button
                      size="icon-xs"
                      variant="ghost"
                      aria-label="Move the lesson up"
                      disabled={lessonIndex === 0 || moveLesson.isPending}
                      onClick={() => moveLesson.mutate({ id: lesson.id!, order: lessonIndex })}
                    >
                      <ArrowUp />
                    </Button>
                    <Button
                      size="icon-xs"
                      variant="ghost"
                      aria-label="Move the lesson down"
                      disabled={lessonIndex === (module.lessons?.length ?? 0) - 1 || moveLesson.isPending}
                      onClick={() => moveLesson.mutate({ id: lesson.id!, order: lessonIndex + 2 })}
                    >
                      <ArrowDown />
                    </Button>
                    <Button
                      size="icon-xs"
                      variant="ghost"
                      aria-label="Edit the lesson"
                      onClick={() => setEditing({ kind: 'lesson', moduleId: module.id!, lesson })}
                    >
                      <Pencil />
                    </Button>
                  </div>
                  <ConfirmButton
                    size="xs"
                    title="Delete this lesson?"
                    description="Its materials are deleted too."
                    onConfirm={() => deleteLesson.mutate(lesson.id!)}
                    pending={deleteLesson.isPending}
                  >
                    Delete
                  </ConfirmButton>
                </li>
              ))}
            </ul>

            <button
              type="button"
              className="flex w-full items-center justify-center gap-2 border-t border-dashed px-5 py-3 text-sm font-medium text-primary transition hover:bg-primary/5"
              onClick={() => setEditing({ kind: 'lesson', moduleId: module.id! })}
            >
              <Plus className="size-4" /> Add a lesson
            </button>
          </CardContent>
        </Card>
      ))}

      {editing?.kind === 'module' && <ModuleDialog courseId={courseId} module={editing.module} refresh={refresh} onClose={() => setEditing(null)} />}
      {editing?.kind === 'lesson' && <LessonDialog moduleId={editing.moduleId} lesson={editing.lesson} refresh={refresh} onClose={() => setEditing(null)} />}
      {editing?.kind === 'materials' && <MaterialsDialog lessonId={editing.lesson.id!} title={editing.lesson.title ?? ''} onClose={() => setEditing(null)} />}
    </div>
  )
}
