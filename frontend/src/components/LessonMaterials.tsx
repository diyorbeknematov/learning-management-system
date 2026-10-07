import { useQuery } from '@tanstack/react-query'
import { FileText, Lock } from 'lucide-react'
import { ApiError, api, call } from '@/api/client'
import { VideoPlayer } from '@/components/VideoPlayer'
import { ErrorBlock, LoadingBlock } from '@/components/States'

/** The materials of a lesson: text, a link to a video, files. */
export function LessonMaterials({ lessonId, locked }: { lessonId: string; locked?: React.ReactNode }) {
  const materials = useQuery({
    queryKey: ['materials', lessonId],
    queryFn: () => call(api.GET('/lessons/{lessonId}/materials', { params: { path: { lessonId } } })),
    retry: false,
  })

  if (materials.isPending) return <LoadingBlock rows={2} />

  if (materials.isError) {
    if (materials.error instanceof ApiError && materials.error.status === 403) {
      return (
        locked ?? (
          <div className="flex items-center gap-2 rounded-lg border border-dashed p-6 text-sm text-muted-foreground">
            <Lock className="size-4" /> Enroll in the course to open this lesson.
          </div>
        )
      )
    }

    return <ErrorBlock error={materials.error} />
  }

  if (materials.data.length === 0) return <p className="text-sm text-muted-foreground">This lesson has no materials yet.</p>

  // the video comes first, the texts and files under it
  const videos = materials.data.filter((material) => material.type === 'video')
  const others = materials.data.filter((material) => material.type !== 'video')

  return (
    <div className="space-y-4">
      {videos.map((material) => (
        <div key={material.id} className="flex justify-center">
          <VideoPlayer url={material.content ?? ''} />
        </div>
      ))}

      {others.map((material) => (
        <article key={material.id} className="rounded-xl border p-4">
          {material.type === 'text' && <p className="whitespace-pre-line leading-relaxed">{material.content}</p>}

          {material.type === 'file' && (
            <a
              href={material.file_url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-2 font-medium text-primary underline-offset-4 hover:underline"
            >
              <FileText className="size-5" /> {material.file_name ?? 'Download the file'}
              {material.file_size ? (
                <span className="text-xs font-normal text-muted-foreground">({(material.file_size / 1024 / 1024).toFixed(1)} MB)</span>
              ) : null}
            </a>
          )}
        </article>
      ))}
    </div>
  )
}
