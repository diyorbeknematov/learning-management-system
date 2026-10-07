import { useQuery } from '@tanstack/react-query'
import { Download } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { downloadFile } from '@/api/client'
import { api } from '@/api/client'
import { DataTable, columnTitle } from '@/components/DataTable'
import { DateRange, RangePresets } from '@/components/DateRange'
import { PageHeader } from '@/components/PageHeader'
import { Toolbar } from '@/components/Panels'
import { NativeSelect } from '@/components/NativeSelect'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { errorMessage } from '@/lib/query'
import { ApiError } from '@/api/client'

const reports = [
  { value: 'enrollments', label: 'Enrollments' },
  { value: 'revenue', label: 'Revenue by course' },
  { value: 'students', label: 'Students' },
  { value: 'progress', label: 'Progress' },
  { value: 'quizzes', label: 'Quizzes' },
  { value: 'certificates', label: 'Certificates' },
  { value: 'instructors', label: 'Instructors' },
  { value: 'reviews', label: 'Reviews' },
] as const

type Report = (typeof reports)[number]['value']
type Row = Record<string, unknown>

/** The reports return a list, or an object with several lists and numbers. */
function Result({ data }: { data: unknown }) {
  if (Array.isArray(data)) return <DataTable rows={data as Row[]} />

  const entries = Object.entries(data as Row)

  return (
    <div className="space-y-6">
      {entries.map(([key, value]) =>
        Array.isArray(value) ? (
          <section key={key} className="space-y-2">
            <h2 className="text-lg font-medium">{key === 'rows' ? 'Details' : columnTitle(key)}</h2>
            <DataTable rows={value as Row[]} />
          </section>
        ) : (
          <p key={key} className="text-sm">
            {columnTitle(key)}: <strong>{String(value)}</strong>
          </p>
        ),
      )}
    </div>
  )
}

export default function ReportsPage() {
  const [report, setReport] = useState<Report>('enrollments')
  const [range, setRange] = useState({ from: '', to: '' })
  const [busy, setBusy] = useState(false)

  const query = new URLSearchParams()
  if (range.from) query.set('from', range.from)
  if (range.to) query.set('to', range.to)

  const data = useQuery({
    queryKey: ['report', report, range],
    queryFn: async () => {
      // the reports have different shapes, so the generic client is used
      const {
        data: body,
        error,
        response,
      } = await api.GET(`/reports/${report}` as '/reports/enrollments', { params: { query: { from: range.from || undefined, to: range.to || undefined } } })

      if (!response.ok) {
        const failure = (error ?? {}) as { error?: { code?: string; message?: string } }

        throw new ApiError(response.status, failure.error?.code ?? 'INTERNAL', failure.error?.message ?? response.statusText)
      }

      return (body as { data?: unknown }).data
    },
  })

  async function csv() {
    setBusy(true)

    try {
      query.set('format', 'csv')
      await downloadFile(`/reports/${report}?${query}`, `${report}.csv`)
    } catch (error) {
      toast.error(errorMessage(error))
    } finally {
      setBusy(false)
    }
  }

  const current = reports.find((item) => item.value === report)

  return (
    <div className="space-y-6">
      <PageHeader title="Reports" description="Numbers about students, courses, quizzes and money. Download any of them as a CSV file." />

      <Toolbar className="border-0 bg-transparent p-0">
        <div className="space-y-1.5">
          <Label htmlFor="report">Report</Label>
          <NativeSelect id="report" className="h-10 w-56 bg-muted/50" value={report} onChange={(e) => setReport(e.target.value as Report)}>
            {reports.map((item) => (
              <option key={item.value} value={item.value}>
                {item.label}
              </option>
            ))}
          </NativeSelect>
        </div>
        <div className="grid w-80 grid-cols-2 gap-3">
          <DateRange from={range.from} to={range.to} onChange={setRange} />
        </div>
        <RangePresets from={range.from} to={range.to} onChange={setRange} />
        <Button variant="outline" className="ml-auto h-10 px-4" disabled={busy} onClick={csv}>
          <Download /> {busy ? 'Preparing…' : 'Download CSV'}
        </Button>
      </Toolbar>

      <h2 className="text-xl font-semibold">{current?.label}</h2>

      {data.isPending && <LoadingBlock />}
      {data.isError && <ErrorBlock error={data.error} />}
      {data.data !== undefined && data.data !== null && <Result data={data.data} />}
    </div>
  )
}
