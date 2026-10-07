import type { ReactNode } from 'react'
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { moneyExact } from '@/lib/format'

/** The colours of the charts; the same series has the same colour everywhere. */
export const colors = ['#4f46e5', '#10b981', '#f59e0b', '#f43f5e', '#0ea5e9', '#8b5cf6']

const legendText = (value: string) => <span className="text-sm text-foreground">{value}</span>

const axis = { fontSize: 12, fill: 'var(--muted-foreground)' }
const grid = 'var(--border)'

type Value = number | string | readonly (number | string)[] | undefined

const asMoney = (value: Value) => moneyExact(Number(Array.isArray(value) ? value[0] : value))

/** A box with a title for one chart. */
export function ChartCard({ title, description, height = 280, children }: { title: string; description?: string; height?: number; children: ReactNode }) {
  return (
    <section className="rounded-xl border bg-card p-5">
      <header className="mb-4 space-y-0.5">
        <h2 className="font-semibold">{title}</h2>
        {description && <p className="text-sm text-muted-foreground">{description}</p>}
      </header>
      <div style={{ height }} className="w-full">
        {children}
      </div>
    </section>
  )
}

export function NoData() {
  return <p className="flex h-full items-center justify-center text-sm text-muted-foreground">No data for this period.</p>
}

type Series = { key: string; label: string; color?: string }

/** Lines over time, with a soft fill below them when `area` is set. */
export function TimeChart({ data, series, area, money = true }: { data: Record<string, unknown>[]; series: Series[]; area?: boolean; money?: boolean }) {
  if (data.length === 0) return <NoData />

  const format = money ? asMoney : (value: Value) => String(value)
  const tick = (value: number) => (money ? `$${value}` : String(value))

  const common = { data, margin: { top: 8, right: 12, left: 0, bottom: 0 } }
  const parts = (
    <>
      <CartesianGrid stroke={grid} strokeDasharray="3 3" vertical={false} />
      <XAxis dataKey="label" tick={axis} tickLine={false} axisLine={false} />
      <YAxis tick={axis} tickLine={false} axisLine={false} width={52} tickFormatter={tick} allowDecimals={false} />
      <Tooltip formatter={(value) => format(value as Value)} />
      <Legend iconType="circle" formatter={legendText} />
    </>
  )

  return (
    <ResponsiveContainer width="100%" height="100%">
      {area ? (
        <AreaChart {...common}>
          <defs>
            {series.map((item, i) => (
              <linearGradient key={item.key} id={`fill-${item.key}`} x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={item.color ?? colors[i]} stopOpacity={0.35} />
                <stop offset="95%" stopColor={item.color ?? colors[i]} stopOpacity={0} />
              </linearGradient>
            ))}
          </defs>
          {parts}
          {series.map((item, i) => (
            <Area
              key={item.key}
              type="monotone"
              dataKey={item.key}
              name={item.label}
              stroke={item.color ?? colors[i]}
              strokeWidth={2.5}
              fill={`url(#fill-${item.key})`}
            />
          ))}
        </AreaChart>
      ) : (
        <LineChart {...common}>
          {parts}
          {series.map((item, i) => (
            <Line
              key={item.key}
              type="monotone"
              dataKey={item.key}
              name={item.label}
              stroke={item.color ?? colors[i]}
              strokeWidth={2.5}
              dot={{ r: 3 }}
              activeDot={{ r: 5 }}
            />
          ))}
        </LineChart>
      )}
    </ResponsiveContainer>
  )
}

/** Bars side by side for every period. */
export function BarsChart({ data, series, money = true }: { data: Record<string, unknown>[]; series: Series[]; money?: boolean }) {
  if (data.length === 0) return <NoData />

  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart data={data} margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
        <CartesianGrid stroke={grid} strokeDasharray="3 3" vertical={false} />
        <XAxis dataKey="label" tick={axis} tickLine={false} axisLine={false} />
        <YAxis tick={axis} tickLine={false} axisLine={false} width={52} tickFormatter={(v: number) => (money ? `$${v}` : String(v))} allowDecimals={false} />
        <Tooltip formatter={(value) => (money ? asMoney(value as Value) : String(value))} cursor={{ fill: 'var(--muted)', opacity: 0.5 }} />
        <Legend iconType="circle" formatter={legendText} />
        {series.map((item, i) => (
          <Bar key={item.key} dataKey={item.key} name={item.label} fill={item.color ?? colors[i]} radius={[6, 6, 0, 0]} maxBarSize={36} />
        ))}
      </BarChart>
    </ResponsiveContainer>
  )
}

/** Shares of a whole as a ring with the total in the middle. */
export function Donut({ data, total, caption }: { data: { name: string; value: number }[]; total?: string; caption?: string }) {
  const shown = data.filter((item) => item.value > 0)

  if (shown.length === 0) return <NoData />

  return (
    <div className="relative h-full">
      <ResponsiveContainer width="100%" height="100%">
        <PieChart>
          <Pie data={shown} dataKey="value" nameKey="name" innerRadius="62%" outerRadius="88%" paddingAngle={2} stroke="none">
            {shown.map((item, i) => (
              <Cell key={item.name} fill={colors[i % colors.length]} />
            ))}
          </Pie>
          <Tooltip formatter={(value) => asMoney(value as Value)} />
          <Legend iconType="circle" verticalAlign="bottom" formatter={legendText} />
        </PieChart>
      </ResponsiveContainer>
      {total && (
        <div className="pointer-events-none absolute inset-x-0 top-[38%] -translate-y-1/2 text-center">
          <p className="text-xl font-bold">{total}</p>
          {caption && <p className="text-xs text-muted-foreground">{caption}</p>}
        </div>
      )}
    </div>
  )
}

/** One bar for each name, the longest on top. */
export function RankedBars({ data, money = true }: { data: { name: string; value: number }[]; money?: boolean }) {
  if (data.length === 0 || data.every((item) => item.value === 0)) return <NoData />

  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart data={data} layout="vertical" margin={{ top: 4, right: 16, left: 0, bottom: 0 }}>
        <CartesianGrid stroke={grid} strokeDasharray="3 3" horizontal={false} />
        <XAxis type="number" tick={axis} tickLine={false} axisLine={false} tickFormatter={(v: number) => (money ? `$${v}` : String(v))} allowDecimals={false} />
        <YAxis type="category" dataKey="name" tick={axis} tickLine={false} axisLine={false} width={120} />
        <Tooltip formatter={(value) => (money ? asMoney(value as Value) : String(value))} cursor={{ fill: 'var(--muted)', opacity: 0.5 }} />
        <Bar dataKey="value" radius={[0, 6, 6, 0]} maxBarSize={22}>
          {data.map((item, i) => (
            <Cell key={item.name} fill={colors[i % colors.length]} />
          ))}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  )
}
