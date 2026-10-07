import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { api, call } from '@/api/client'
import { DateRange, RangePresets } from '@/components/DateRange'
import { PageHeader } from '@/components/PageHeader'
import { Pagination } from '@/components/Pagination'
import { TableCard, Toolbar } from '@/components/Panels'
import { Empty, ErrorBlock, LoadingBlock } from '@/components/States'
import { StatusBadge } from '@/components/StatusBadge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { formatDate, moneyExact, plural } from '@/lib/format'

const LIMIT = 20

export default function PaymentsPage() {
  const [range, setRange] = useState({ from: '', to: '' })
  const [page, setPage] = useState(1)

  function change(next: { from: string; to: string }) {
    setRange(next)
    setPage(1)
  }

  const payments = useQuery({
    queryKey: ['payments', range, page],
    queryFn: () => call(api.GET('/payments', { params: { query: { from: range.from || undefined, to: range.to || undefined, page, limit: LIMIT } } })),
    placeholderData: (previous) => previous,
  })

  return (
    <div className="space-y-6">
      <PageHeader
        title="Payments"
        description={payments.data ? `${plural(payments.data.total, 'payment')} in this period` : 'What students paid for the courses'}
      />

      <Toolbar className="border-0 bg-transparent p-0">
        <div className="grid w-80 grid-cols-2 gap-3">
          <DateRange from={range.from} to={range.to} onChange={change} />
        </div>
        <RangePresets from={range.from} to={range.to} onChange={change} />
      </Toolbar>

      {payments.isPending && <LoadingBlock />}
      {payments.isError && <ErrorBlock error={payments.error} />}
      {payments.data?.items?.length === 0 && <Empty title="No payments in this period" />}

      {(payments.data?.items?.length ?? 0) > 0 && (
        <TableCard>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Paid on</TableHead>
                <TableHead>Enrollment</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Amount</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {payments.data?.items?.map((payment) => (
                <TableRow key={payment.id}>
                  <TableCell>{formatDate(payment.paid_at)}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{payment.enrollment_id?.slice(0, 8)}</TableCell>
                  <TableCell>
                    <StatusBadge value={payment.status} />
                  </TableCell>
                  <TableCell className="text-right font-semibold">{moneyExact(payment.amount)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableCard>
      )}

      <Pagination page={page} limit={LIMIT} total={payments.data?.total ?? 0} onPage={setPage} />
    </div>
  )
}
