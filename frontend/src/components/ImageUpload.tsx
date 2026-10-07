import { ImagePlus } from 'lucide-react'
import { useRef, useState } from 'react'
import { toast } from 'sonner'
import { uploadFile, type UploadPurpose } from '@/api/upload'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/lib/query'

/** Picks an image, uploads it and tells the form its key; shows what is chosen. */
export function ImageUpload({
  purpose,
  previewUrl,
  onUploaded,
  label,
}: {
  purpose: Extract<UploadPurpose, 'avatar' | 'course_cover'>
  previewUrl?: string
  onUploaded: (objectKey: string) => void
  label: string
}) {
  const input = useRef<HTMLInputElement>(null)
  const [busy, setBusy] = useState(false)
  const [local, setLocal] = useState<string | undefined>()

  async function pick(file?: File) {
    if (!file) return

    setBusy(true)

    try {
      const uploaded = await uploadFile(purpose, file)

      setLocal(URL.createObjectURL(file))
      onUploaded(uploaded.objectKey)
    } catch (error) {
      toast.error(errorMessage(error))
    } finally {
      setBusy(false)
      if (input.current) input.current.value = ''
    }
  }

  const shown = local ?? previewUrl

  return (
    <div className="space-y-2">
      <p className="text-sm font-medium">{label}</p>
      <div className="flex items-center gap-4">
        {shown ? (
          <img src={shown} alt="" className="h-24 w-40 rounded-lg border object-cover" />
        ) : (
          <div className="flex h-24 w-40 items-center justify-center rounded-lg border border-dashed text-muted-foreground">
            <ImagePlus className="size-6" />
          </div>
        )}
        <div>
          <input ref={input} type="file" accept="image/png,image/jpeg,image/webp" className="sr-only" id={`upload-${purpose}`} onChange={(e) => pick(e.target.files?.[0])} />
          <Button type="button" variant="outline" size="sm" disabled={busy} onClick={() => input.current?.click()}>
            {busy ? 'Uploading…' : shown ? 'Change the image' : 'Choose an image'}
          </Button>
          <p className="mt-1 text-xs text-muted-foreground">PNG, JPEG or WebP.</p>
        </div>
      </div>
    </div>
  )
}
