import { useQuery } from '@tanstack/react-query'
import { CircleDollarSign, Coins, TrendingUp } from 'lucide-react'
import { useState } from 'react'
import { api, call } from '@/api/client'
import { DateRange } from '@/components/DateRange'
import { NativeSelect } from '@/components/NativeSelect'
import { PageHeader } from '@/components/PageHeader'
import { TableCard, Toolbar } from '@/components/Panels'
import { StatCard } from '@/components/StatCard'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { Label } from '@/components/ui/label'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { moneyExact } from '@/lib/format'

type Group = 'day' | 'week' | 'month'
type Point = { period?: string; revenue?: number; expenses?: number; profit?: number }

/** Two bars for every period: what came in and what went to the instructors. */
function Chart({ points }: { points: Point[] }) {
  const largest = Math.max(1, ...points.map((point) => Math.max(point.revenue ?? 0, point.expenses ?? 0)))

  return (
    <TableCard
      title="Revenue and payouts"
      actions={
        <div className="flex items-center gap-4 text-xs text-muted-foreground">
          <span className="inline-flex items-center gap-1.5">
            <span className="size-2.5 rounded-sm bg-primary" /> Revenue
          </span>
          <span className="inline-flex items-center gap-1.5">
            <span className="size-2.5 rounded-sm bg-amber-400" /> Payouts
          </span>
        </div>
      }
    >
      <div className="flex h-64 items-end gap-3 overflow-x-auto px-5 pb-4 pt-8" role="img" aria-label="Revenue and payouts by period">
        {points.map((point) => (
          <div
            key={point.period}
            className="flex h-full min-w-16 flex-1 flex-col justify-end gap-2"
            title={`${point.period?.slice(0, 10)}: ${moneyExact(point.revenue)} revenue, ${moneyExact(point.expenses)} payouts`}
          >
            <div className="flex flex-1 items-end justify-center gap-1.5">
              <div
                className="w-1/3 max-w-8 rounded-t bg-primary"
                style={{ height: `${((point.revenue ?? 0) / largest) * 100}%`, minHeight: point.revenue ? 4 : 0 }}
              />
              <div
                className="w-1/3 max-w-8 rounded-t bg-amber-400"
                style={{ height: `${((point.expenses ?? 0) / largest) * 100}%`, minHeight: point.expenses ? 4 : 0 }}
              />
            </div>
            <p className="text-center text-xs text-muted-foreground">{point.period?.slice(0, 10)}</p>
          </div>
        ))}
      </div>
    </TableCard>
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

  return (
    <div className="space-y-6">
      <PageHeader title="Finance" description="Money in and out: what students paid and what instructors earned." />

      <Toolbar className="max-w-2xl">
        <div className="grid flex-1 grid-cols-3 gap-3">
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
      </Toolbar>

      {summary.isPending && <LoadingBlock rows={2} />}
      {summary.isError && <ErrorBlock error={summary.error} />}

      {summary.data && (
        <>
          <div className="grid gap-4 md:grid-cols-3">
            <StatCard label="Revenue" value={moneyExact(summary.data.revenue)} icon={CircleDollarSign} tone="green" />
            <StatCard label="Paid to instructors" value={moneyExact(summary.data.expenses)} icon={Coins} tone="amber" />
            <StatCard
              label="Net profit"
              value={moneyExact(summary.data.net_profit)}
              icon={TrendingUp}
              tone={(summary.data.net_profit ?? 0) < 0 ? 'rose' : 'indigo'}
            />
          </div>

          {points.length === 0 ? (
            <p className="rounded-xl border border-dashed p-10 text-center text-muted-foreground">No payments in this period.</p>
          ) : (
            <>
              <Chart points={points} />

              <TableCard title="By period">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Period</TableHead>
                      <TableHead className="text-right">Revenue</TableHead>
                      <TableHead className="text-right">Payouts</TableHead>
                      <TableHead className="text-right">Profit</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {points.map((point) => (
                      <TableRow key={point.period}>
                        <TableCell>{point.period?.slice(0, 10)}</TableCell>
                        <TableCell className="text-right">{moneyExact(point.revenue)}</TableCell>
                        <TableCell className="text-right">{moneyExact(point.expenses)}</TableCell>
                        <TableCell className="text-right font-semibold">{moneyExact(point.profit)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableCard>
            </>
          )}
        </>
      )}

      {expenses.data && (expenses.data.payouts?.length ?? 0) > 0 && (
        <TableCard title="Payouts to instructors">
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
                  <TableCell className="text-right font-semibold">{moneyExact(payout.amount)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableCard>
      )}
    </div>
  )
}
