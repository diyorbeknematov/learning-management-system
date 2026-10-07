import { useQuery } from '@tanstack/react-query'
import { Copy, Download, ExternalLink } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router'
import { toast } from 'sonner'
import { api, call, downloadFile } from '@/api/client'
import { CertificatePreview } from '@/components/CertificatePreview'
import { PageHeader } from '@/components/PageHeader'
import { Empty, ErrorBlock, LoadingBlock } from '@/components/States'
import { Button } from '@/components/ui/button'
import { formatDate, plural } from '@/lib/format'
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

  async function copy(number: string) {
    try {
      await navigator.clipboard.writeText(`${window.location.origin}/verify/${number}`)
      toast.success('The link is copied')
    } catch {
      toast.error('The link could not be copied')
    }
  }

  return (
    <div className="space-y-8">
      <PageHeader
        title="My certificates"
        description={
          certificates.data
            ? `${plural(certificates.data.length, 'certificate')}. Download the PDF or share the link: anybody can check it.`
            : 'Download a certificate or share its public page.'
        }
      />

      {certificates.isPending && <LoadingBlock rows={2} />}
      {certificates.isError && <ErrorBlock error={certificates.error} />}

      {certificates.data?.length === 0 && (
        <Empty title="No certificates yet">A certificate is issued when you finish all the lessons and pass the final quiz.</Empty>
      )}

      <div className="grid gap-8 md:grid-cols-2 2xl:grid-cols-3">
        {certificates.data?.map((certificate) => (
          <article key={certificate.id} className="space-y-4">
            <Link
              to={`/verify/${certificate.unique_id}`}
              className="block rounded-xl outline-none transition hover:-translate-y-0.5 hover:shadow-xl focus-visible:ring-3 focus-visible:ring-ring/50"
            >
              <CertificatePreview
                student={certificate.student_name}
                course={certificate.course_title}
                instructor={certificate.instructor_name}
                date={certificate.completion_date}
                number={certificate.unique_id}
              />
            </Link>

            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="min-w-0">
                <h2 className="truncate font-semibold">{certificate.course_title}</h2>
                <p className="text-sm text-muted-foreground">Completed on {formatDate(certificate.completion_date)}</p>
              </div>
              <div className="flex flex-wrap gap-2">
                <Button size="sm" disabled={busy === certificate.id} onClick={() => download(certificate.id!, certificate.unique_id ?? 'file')}>
                  <Download /> {busy === certificate.id ? 'Preparing…' : 'Download PDF'}
                </Button>
                <Button size="sm" variant="outline" onClick={() => copy(certificate.unique_id!)}>
                  <Copy /> Copy link
                </Button>
                <Link
                  to={`/verify/${certificate.unique_id}`}
                  className="inline-flex h-7 items-center gap-1 px-2 text-sm font-medium text-primary hover:underline"
                >
                  <ExternalLink className="size-3.5" /> Public page
                </Link>
              </div>
            </div>
          </article>
        ))}
      </div>
    </div>
  )
}
