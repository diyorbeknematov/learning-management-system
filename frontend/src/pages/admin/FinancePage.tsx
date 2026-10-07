import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { api, call } from '@/api/client'
import { DateRange } from '@/components/DateRange'
import { NativeSelect } from '@/components/NativeSelect'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { moneyExact } from '@/lib/format'

type Group = 'day' | 'week' | 'month'

function Figure({ label, value, tone }: { label: string; value?: number; tone?: 'good' | 'bad' }) {
  return (
    <Card>
      <CardHeader>
        <CardDescription>{label}</CardDescription>
        <CardTitle className={tone === 'bad' ? 'text-2xl text-destructive' : 'text-2xl'}>{moneyExact(value)}</CardTitle>
      </CardHeader>
    </Card>
  )
}

export default function FinancePage() {
  const [range, setRange] = useState({ from: '', to: '' })
  const [group, setGroup] = useState<Group>('month')

  const summary = useQuery({
    queryKey: ['finance', range, group],
    queryFn: () => call(api.GET('/finance', { params: { query: { from: range.from || undefined, to: range.to || undefined, group_by: group } } })),
  })

  const expenses = useQuery({
    queryKey: ['finance', 'expenses', range, group],
    queryFn: () => call(api.GET('/finance/expenses', { params: { query: { from: range.from || undefined, to: range.to || undefined, group_by: group } } })),
  })

  const points = summary.data?.points ?? []
  const largest = Math.max(1, ...points.map((point) => Math.max(point.revenue ?? 0, point.expenses ?? 0)))

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">Finance</h1>

      <div className="grid max-w-xl grid-cols-3 gap-3">
        <DateRange from={range.from} to={range.to} onChange={setRange} />
        <div className="space-y-1.5">
          <Label htmlFor="group">Group by</Label>
          <NativeSelect id="group" value={group} onChange={(e) => setGroup(e.target.value as Group)}>
            <option value="day">Day</option>
            <option value="week">Week</option>
            <option value="month">Month</option>
          </NativeSelect>
        </div>
      </div>

      {summary.isPending && <LoadingBlock rows={2} />}
      {summary.isError && <ErrorBlock error={summary.error} />}

      {summary.data && (
        <>
          <div className="grid gap-4 sm:grid-cols-3">
            <Figure label="Revenue" value={summary.data.revenue} />
            <Figure label="Paid to instructors" value={summary.data.expenses} />
            <Figure label="Net profit" value={summary.data.net_profit} tone={(summary.data.net_profit ?? 0) < 0 ? 'bad' : 'good'} />
          </div>

          {points.length === 0 ? (
            <p className="text-sm text-muted-foreground">No payments in this period.</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Period</TableHead>
                  <TableHead className="w-64">Revenue and payouts</TableHead>
                  <TableHead className="text-right">Revenue</TableHead>
                  <TableHead className="text-right">Payouts</TableHead>
                  <TableHead className="text-right">Profit</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {points.map((point) => (
                  <TableRow key={point.period}>
                    <TableCell>{point.period?.slice(0, 10)}</TableCell>
                    <TableCell>
                      <div className="space-y-1" aria-hidden>
                        <div className="h-2 rounded bg-primary" style={{ width: `${((point.revenue ?? 0) / largest) * 100}%` }} />
                        <div className="h-2 rounded bg-amber-400" style={{ width: `${((point.expenses ?? 0) / largest) * 100}%` }} />
                      </div>
                    </TableCell>
                    <TableCell className="text-right">{moneyExact(point.revenue)}</TableCell>
                    <TableCell className="text-right">{moneyExact(point.expenses)}</TableCell>
                    <TableCell className="text-right font-medium">{moneyExact(point.profit)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </>
      )}

      {expenses.data && (expenses.data.payouts?.length ?? 0) > 0 && (
        <section className="space-y-2">
          <h2 className="text-lg font-medium">Payouts to instructors</h2>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Date</TableHead>
                <TableHead>Rule</TableHead>
                <TableHead className="text-right">Amount</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {expenses.data.payouts?.slice(0, 50).map((payout) => (
                <TableRow key={payout.id}>
                  <TableCell>{payout.created_at?.slice(0, 10)}</TableCell>
                  <TableCell>{payout.type === 'percentage' ? `${payout.value}% of the price` : `${moneyExact(payout.value)} per student`}</TableCell>
                  <TableCell className="text-right">{moneyExact(payout.amount)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </section>
      )}
    </div>
  )
}
