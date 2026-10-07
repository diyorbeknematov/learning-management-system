import { useQuery } from '@tanstack/react-query'
import { Award, Download } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router'
import { toast } from 'sonner'
import { api, call, downloadFile } from '@/api/client'
import { Empty, ErrorBlock, LoadingBlock } from '@/components/States'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { formatDate } from '@/lib/format'
import { errorMessage } from '@/lib/query'

export default function CertificatesPage() {
  const [busy, setBusy] = useState<string | null>(null)

  const certificates = useQuery({
    queryKey: ['certificates', 'me'],
    queryFn: () => call(api.GET('/certificates/me')),
  })

  async function download(id: string, number: string) {
    setBusy(id)

    try {
      await downloadFile(`/certificates/${id}/download`, `certificate-${number}.pdf`)
    } catch (error) {
      toast.error(errorMessage(error))
    } finally {
      setBusy(null)
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">My certificates</h1>

      {certificates.isPending && <LoadingBlock rows={2} />}
      {certificates.isError && <ErrorBlock error={certificates.error} />}

      {certificates.data?.length === 0 && (
        <Empty title="No certificates yet">A certificate is issued when you finish all the lessons and pass the final quiz.</Empty>
      )}

      <div className="grid gap-4 sm:grid-cols-2">
        {certificates.data?.map((certificate) => (
          <Card key={certificate.id}>
            <CardHeader>
              <CardTitle className="flex items-start gap-2">
                <Award className="mt-0.5 size-5 shrink-0 text-amber-500" />
                {certificate.course_title}
              </CardTitle>
              <CardDescription>
                Completed on {formatDate(certificate.completion_date)}
                {certificate.instructor_name ? ` · ${certificate.instructor_name}` : ''}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <p className="font-mono text-xs text-muted-foreground">{certificate.unique_id}</p>
              <div className="flex flex-wrap gap-2">
                <Button size="sm" disabled={busy === certificate.id} onClick={() => download(certificate.id!, certificate.unique_id ?? 'file')}>
                  <Download /> {busy === certificate.id ? 'Preparing…' : 'Download PDF'}
                </Button>
                <Link to={`/verify/${certificate.unique_id}`} className="inline-flex h-7 items-center px-2 text-sm underline">
                  Public page
                </Link>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
