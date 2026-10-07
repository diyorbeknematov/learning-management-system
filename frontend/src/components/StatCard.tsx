import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { Link } from 'react-router'
import { cn } from '@/lib/utils'

const tones = {
  indigo: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-500/20 dark:text-indigo-300',
  green: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-300',
  amber: 'bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-300',
  rose: 'bg-rose-100 text-rose-700 dark:bg-rose-500/20 dark:text-rose-300',
  sky: 'bg-sky-100 text-sky-700 dark:bg-sky-500/20 dark:text-sky-300',
}

/** A number with a label and an icon; a link when it has somewhere to lead. */
export function StatCard({
  label,
  value,
  icon: Icon,
  tone = 'indigo',
  to,
}: {
  label: string
  value: ReactNode
  icon: LucideIcon
  tone?: keyof typeof tones
  to?: string
}) {
  const body = (
    <div className={cn('flex items-center gap-4 rounded-xl border bg-card p-5', to && 'transition hover:border-primary hover:shadow-sm')}>
      <span className={cn('flex size-12 shrink-0 items-center justify-center rounded-xl', tones[tone])}>
        <Icon className="size-6" />
      </span>
      <div className="min-w-0">
        <p className="truncate text-sm text-muted-foreground">{label}</p>
        <p className="truncate text-2xl font-semibold">{value}</p>
      </div>
    </div>
  )

  return to ? (
    <Link to={to} className="rounded-xl outline-none focus-visible:ring-3 focus-visible:ring-ring/50">
      {body}
    </Link>
  ) : (
    body
  )
}
