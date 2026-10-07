import { useQuery } from '@tanstack/react-query'
import { CircleDollarSign, Coins, TrendingUp } from 'lucide-react'
import { useState } from 'react'
import { api, call } from '@/api/client'
import { DateRange, RangePresets } from '@/components/DateRange'
import { NativeSelect } from '@/components/NativeSelect'
import { PageHeader } from '@/components/PageHeader'
import { TableCard, Toolbar } from '@/components/Panels'
import { StatCard } from '@/components/StatCard'
import { ErrorBlock, LoadingBlock } from '@/components/States'
import { Label } from '@/components/ui/label'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { moneyExact } from '@/lib/format'
import { format } from 'date-fns'
import { BarsChart, ChartCard, Donut, RankedBars, TimeChart } from '@/components/charts'

type Group = 'day' | 'week' | 'month'

/** Days up to a month, weeks up to four months, months for anything longer or without limits. */
function autoGroup(range: { from: string; to: string }): Group {
  if (!range.from || !range.to) return 'month'

  const days = (new Date(range.to).getTime() - new Date(range.from).getTime()) / 86_400_000

  if (days <= 31) return 'day'
  if (days <= 120) return 'week'

  return 'month'
}
export default function FinancePage() {
  const [range, setRange] = useState({ from: '', to: '' })
  // 'auto' picks days for a short period and months for a long one, so the lines have points to join
  const [chosen, setChosen] = useState<Group | 'auto'>('auto')
  const group = chosen === 'auto' ? autoGroup(range) : chosen

  const summary = useQuery({
    queryKey: ['finance', range, group],
    queryFn: () => call(api.GET('/finance', { params: { query: { from: range.from || undefined, to: range.to || undefined, group_by: group } } })),
  })

  const expenses = useQuery({
    queryKey: ['finance', 'expenses', range, group],
    queryFn: () => call(api.GET('/finance/expenses', { params: { query: { from: range.from || undefined, to: range.to || undefined, group_by: group } } })),
  })

  const query = { from: range.from || undefined, to: range.to || undefined }

  const byCourse = useQuery({
    queryKey: ['report', 'revenue', range],
    queryFn: () => call(api.GET('/reports/revenue', { params: { query } })),
  })
  const byInstructor = useQuery({
    queryKey: ['report', 'instructors', range],
    queryFn: () => call(api.GET('/reports/instructors', { params: { query } })),
  })
  const enrollments = useQuery({
    queryKey: ['report', 'enrollments', range, group],
    queryFn: () => call(api.GET('/reports/enrollments', { params: { query: { ...query, group_by: group } } })),
  })

  const points = summary.data?.points ?? []

  // a short label for a period: "Oct 2026" when grouped by month, "7 Oct" otherwise
  const label = (period?: string) => {
    if (!period) return ''

    const date = new Date(period)

    return Number.isNaN(date.getTime()) ? period.slice(0, 10) : format(date, group === 'month' ? 'MMM yyyy' : 'd MMM')
  }

  const series = points.map((point) => ({ label: label(point.period), revenue: point.revenue ?? 0, expenses: point.expenses ?? 0, profit: point.profit ?? 0 }))
  const topCourses = [...(byCourse.data ?? [])].sort((a, b) => (b.revenue ?? 0) - (a.revenue ?? 0))
  const courseShares = [
    ...topCourses.slice(0, 5).map((row) => ({ name: row.course_title ?? '', value: row.revenue ?? 0 })),
    { name: 'Others', value: topCourses.slice(5).reduce((sum, row) => sum + (row.revenue ?? 0), 0) },
  ]
  const instructors = [...(byInstructor.data ?? [])].sort((a, b) => (b.revenue ?? 0) - (a.revenue ?? 0)).slice(0, 6)
  const trend = (enrollments.data?.trend ?? []).map((row) => ({ label: label(row.period), count: row.count ?? 0 }))

  return (
    <div className="space-y-6">
      <PageHeader title="Finance" description="Money in and out: what students paid and what instructors earned." />

      <Toolbar className="border-0 bg-transparent p-0">
        <div className="grid w-[30rem] max-w-full grid-cols-3 gap-3">
          <DateRange from={range.from} to={range.to} onChange={setRange} />
          <div className="space-y-1.5">
            <Label htmlFor="group">Group by</Label>
            <NativeSelect id="group" className="h-10 bg-muted/50" value={chosen} onChange={(e) => setChosen(e.target.value as Group | 'auto')}>
              <option value="auto">Automatic</option>
              <option value="day">Day</option>
              <option value="week">Week</option>
              <option value="month">Month</option>
            </NativeSelect>
          </div>
        </div>
        <RangePresets from={range.from} to={range.to} onChange={setRange} />
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
              <div className="grid gap-6 xl:grid-cols-3">
                <div className="xl:col-span-2">
                  <ChartCard title="Revenue, payouts and profit" description="How the money moved over time">
                    <TimeChart
                      area
                      data={series}
                      series={[
                        { key: 'revenue', label: 'Revenue' },
                        { key: 'expenses', label: 'Payouts' },
                        { key: 'profit', label: 'Profit' },
                      ]}
                    />
                  </ChartCard>
                </div>
                <ChartCard title="Where the revenue goes" description="Payouts to instructors and what stays">
                  <Donut
                    total={moneyExact(summary.data.revenue)}
                    caption="revenue"
                    data={[
                      { name: 'Paid to instructors', value: summary.data.expenses ?? 0 },
                      { name: 'Net profit', value: Math.max(summary.data.net_profit ?? 0, 0) },
                    ]}
                  />
                </ChartCard>
              </div>

              <div className="grid gap-6 xl:grid-cols-2">
                <ChartCard title="Revenue and payouts by period" description="Side by side for every period">
                  <BarsChart
                    data={series}
                    series={[
                      { key: 'revenue', label: 'Revenue' },
                      { key: 'expenses', label: 'Payouts' },
                    ]}
                  />
                </ChartCard>
                <ChartCard title="New enrollments" description="How many students joined a course in each period">
                  <TimeChart data={trend} money={false} series={[{ key: 'count', label: 'Enrollments', color: '#10b981' }]} area />
                </ChartCard>
              </div>

              <div className="grid gap-6 xl:grid-cols-2">
                <ChartCard title="Revenue by course" description="The five best courses and the rest">
                  <Donut data={courseShares} total={moneyExact(summary.data.revenue)} caption="in total" />
                </ChartCard>
                <ChartCard title="Revenue by instructor" description="Who brings the most">
                  <RankedBars data={instructors.map((row) => ({ name: row.full_name ?? '', value: row.revenue ?? 0 }))} />
                </ChartCard>
              </div>

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
