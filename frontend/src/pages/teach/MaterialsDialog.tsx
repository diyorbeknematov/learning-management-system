import { useQuery } from '@tanstack/react-query'
import { FileText, Link2, Type } from 'lucide-react'
import { useRef, useState } from 'react'
import { toast } from 'sonner'
import { api, call } from '@/api/client'
import { uploadFile } from '@/api/upload'
import { ConfirmButton } from '@/components/ConfirmButton'
import { NativeSelect } from '@/components/NativeSelect'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { useApiMutation } from '@/lib/mutations'
import { errorMessage } from '@/lib/query'

type Kind = 'text' | 'video' | 'file'

function AddMaterial({ lessonId }: { lessonId: string }) {
  const [kind, setKind] = useState<Kind>('text')
  const [content, setContent] = useState('')
  const [busy, setBusy] = useState(false)
  const file = useRef<HTMLInputElement>(null)

  const add = useApiMutation(
    (body: { type: Kind; content?: string; object_key?: string; file_name?: string }) =>
      call(api.POST('/lessons/{lessonId}/materials', { params: { path: { lessonId } }, body })),
    {
      success: 'Material added',
      invalidate: [['materials', lessonId]],
      onSuccess: () => {
        setContent('')
        if (file.current) file.current.value = ''
      },
    },
  )

  async function submit(e: React.FormEvent) {
    e.preventDefault()

    if (kind !== 'file') {
      add.mutate({ type: kind, content: content.trim() })

      return
    }

    const chosen = file.current?.files?.[0]
    if (!chosen) return toast.error('Choose a file first')

    setBusy(true)

    try {
      const uploaded = await uploadFile('material', chosen)

      add.mutate({ type: 'file', object_key: uploaded.objectKey, file_name: uploaded.fileName })
    } catch (error) {
      toast.error(errorMessage(error))
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={submit} className="space-y-3 rounded-lg border p-3">
      <div className="grid gap-3 sm:grid-cols-[10rem_1fr]">
        <div className="space-y-1.5">
          <Label htmlFor="material-kind">Add</Label>
          <NativeSelect id="material-kind" value={kind} onChange={(e) => setKind(e.target.value as Kind)}>
            <option value="text">Text</option>
            <option value="video">Video link</option>
            <option value="file">File (PDF, slides, video)</option>
          </NativeSelect>
        </div>

        <div className="space-y-1.5">
          {kind === 'text' && (
            <>
              <Label htmlFor="material-content">Text</Label>
              <Textarea id="material-content" rows={4} value={content} onChange={(e) => setContent(e.target.value)} required />
            </>
          )}
          {kind === 'video' && (
            <>
              <Label htmlFor="material-content">Link (http or https)</Label>
              <Input id="material-content" type="url" value={content} onChange={(e) => setContent(e.target.value)} placeholder="https://" required />
            </>
          )}
          {kind === 'file' && (
            <>
              <Label htmlFor="material-file">File</Label>
              <Input id="material-file" type="file" ref={file} accept=".pdf,.ppt,.pptx,video/mp4,video/webm" />
            </>
          )}
        </div>
      </div>

      <Button type="submit" size="sm" disabled={busy || add.isPending}>
        {busy ? 'Uploading…' : add.isPending ? 'Adding…' : 'Add the material'}
      </Button>
    </form>
  )
}

function MaterialRow({ lessonId, material }: { lessonId: string; material: { id?: string; type?: string; content?: string; file_name?: string; file_url?: string } }) {
  const [editing, setEditing] = useState(false)
  const [content, setContent] = useState(material.content ?? '')

  const save = useApiMutation(
    () => call(api.PUT('/materials/{materialId}', { params: { path: { materialId: material.id! } }, body: { content: content.trim() } })),
    { success: 'Material saved', invalidate: [['materials', lessonId]], onSuccess: () => setEditing(false) },
  )

  const remove = useApiMutation(
    () => call(api.DELETE('/materials/{materialId}', { params: { path: { materialId: material.id! } } })),
    { success: 'Material deleted', invalidate: [['materials', lessonId]] },
  )

  const Icon = material.type === 'text' ? Type : material.type === 'video' ? Link2 : FileText

  return (
    <li className="space-y-2 rounded-lg border p-3">
      <div className="flex items-start gap-2">
        <Icon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
        <div className="min-w-0 flex-1 text-sm">
          {editing ? (
            material.type === 'text' ? (
              <Textarea rows={4} value={content} onChange={(e) => setContent(e.target.value)} aria-label="Text" />
            ) : (
              <Input value={content} onChange={(e) => setContent(e.target.value)} aria-label="Link" />
            )
          ) : material.type === 'file' ? (
            <a href={material.file_url} target="_blank" rel="noreferrer" className="underline">
              {material.file_name ?? 'File'}
            </a>
          ) : (
            <p className="line-clamp-3 whitespace-pre-line break-words">{material.content}</p>
          )}
        </div>
        <div className="flex shrink-0 gap-1">
          {material.type !== 'file' &&
            (editing ? (
              <>
                <Button size="xs" disabled={save.isPending} onClick={() => save.mutate(undefined)}>
                  Save
                </Button>
                <Button size="xs" variant="ghost" onClick={() => setEditing(false)}>
                  Cancel
                </Button>
              </>
            ) : (
              <Button size="xs" variant="outline" onClick={() => setEditing(true)}>
                Edit
              </Button>
            ))}
          <ConfirmButton size="xs" title="Delete this material?" onConfirm={() => remove.mutate(undefined)} pending={remove.isPending}>
            Delete
          </ConfirmButton>
        </div>
      </div>
    </li>
  )
}

/** The materials of one lesson: see, add, change, delete. */
export function MaterialsDialog({ lessonId, title, onClose }: { lessonId: string; title: string; onClose: () => void }) {
  const materials = useQuery({
    queryKey: ['materials', lessonId],
    queryFn: () => call(api.GET('/lessons/{lessonId}/materials', { params: { path: { lessonId } } })),
  })

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Materials: {title}</DialogTitle>
          <DialogDescription>Text, a link to a video, or a file. Students see them in this order.</DialogDescription>
        </DialogHeader>

        {materials.isPending && <LoadingBlock rows={2} />}
        {materials.isError && <ErrorBlock error={materials.error} />}
        {materials.data?.length === 0 && <p className="text-sm text-muted-foreground">No materials yet.</p>}

        <ul className="space-y-2">
          {materials.data?.map((material) => (
            <MaterialRow key={material.id} lessonId={lessonId} material={material} />
          ))}
        </ul>

        <AddMaterial lessonId={lessonId} />
      </DialogContent>
    </Dialog>
  )
}
