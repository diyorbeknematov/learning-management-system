import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { api, call } from '@/api/client'
import { DateRange } from '@/components/DateRange'
import { Pagination } from '@/components/Pagination'
import { Empty, ErrorBlock, LoadingBlock } from '@/components/States'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { formatDate, moneyExact } from '@/lib/format'

const LIMIT = 20

export default function PaymentsPage() {
  const [range, setRange] = useState({ from: '', to: '' })
  const [page, setPage] = useState(1)

  const payments = useQuery({
    queryKey: ['payments', range, page],
    queryFn: () => call(api.GET('/payments', { params: { query: { from: range.from || undefined, to: range.to || undefined, page, limit: LIMIT } } })),
    placeholderData: (previous) => previous,
  })

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">Payments</h1>

      <div className="grid max-w-md grid-cols-2 gap-3">
        <DateRange
          from={range.from}
          to={range.to}
          onChange={(next) => {
            setRange(next)
            setPage(1)
          }}
        />
      </div>

      {payments.isPending && <LoadingBlock />}
      {payments.isError && <ErrorBlock error={payments.error} />}
      {payments.data?.items?.length === 0 && <Empty title="No payments in this period" />}

      {(payments.data?.items?.length ?? 0) > 0 && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Paid</TableHead>
              <TableHead>Enrollment</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Amount</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {payments.data?.items?.map((payment) => (
              <TableRow key={payment.id}>
                <TableCell>{formatDate(payment.paid_at)}</TableCell>
                <TableCell className="font-mono text-xs">{payment.enrollment_id?.slice(0, 8)}</TableCell>
                <TableCell>
                  <Badge variant="secondary">{payment.status}</Badge>
                </TableCell>
                <TableCell className="text-right font-medium">{moneyExact(payment.amount)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <Pagination page={page} limit={LIMIT} total={payments.data?.total ?? 0} onPage={setPage} />
    </div>
  )
}
