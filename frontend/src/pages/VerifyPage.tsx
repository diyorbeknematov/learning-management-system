import { useQuery } from '@tanstack/react-query'
import { BadgeCheck, SearchX } from 'lucide-react'
import { useState } from 'react'
import { useNavigate, useParams } from 'react-router'
import { ApiError, api, call } from '@/api/client'
import { CertificatePreview } from '@/components/CertificatePreview'
import { LoadingBlock } from '@/components/States'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'

// The public page a certificate (and its QR code) points to: anybody can check
// that a certificate is real.
export default function VerifyPage() {
  const { uniqueId } = useParams()
  const navigate = useNavigate()
  const [typed, setTyped] = useState(uniqueId ?? '')

  const result = useQuery({
    queryKey: ['verify', uniqueId],
    queryFn: () => call(api.GET('/certificates/verify/{uniqueId}', { params: { path: { uniqueId: uniqueId! } } })),
    enabled: Boolean(uniqueId),
    retry: false,
  })

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h1 className="text-2xl font-semibold">Check a certificate</h1>

      <form
        className="flex gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          if (typed.trim()) navigate(`/verify/${encodeURIComponent(typed.trim())}`)
        }}
      >
        <Input value={typed} onChange={(e) => setTyped(e.target.value)} placeholder="LMS-XXXX-XXXX-XXXX" aria-label="Certificate number" />
        <Button type="submit">Check</Button>
      </form>

      {result.isFetching && <LoadingBlock rows={1} />}

      {result.data && (
        <div className="space-y-4">
          <p className="flex items-center gap-2 text-lg font-semibold text-emerald-700">
            <BadgeCheck className="size-6" /> This certificate is real
          </p>
          <CertificatePreview
            student={result.data.student_name}
            course={result.data.course_title}
            instructor={result.data.instructor_name}
            date={result.data.completion_date}
            number={result.data.unique_id}
          />
        </div>
      )}

      {result.isError && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <SearchX className="size-6 text-destructive" />
              {result.error instanceof ApiError && result.error.status === 404 ? 'No such certificate' : 'Could not check it'}
            </CardTitle>
            <CardDescription>Check the number and try again.</CardDescription>
          </CardHeader>
        </Card>
      )}
    </div>
  )
}
