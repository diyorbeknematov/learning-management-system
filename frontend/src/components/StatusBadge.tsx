import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

const tones: Record<string, string> = {
  green: 'border-transparent bg-emerald-100 text-emerald-800 dark:bg-emerald-500/20 dark:text-emerald-300',
  red: 'border-transparent bg-rose-100 text-rose-800 dark:bg-rose-500/20 dark:text-rose-300',
  amber: 'border-transparent bg-amber-100 text-amber-800 dark:bg-amber-500/20 dark:text-amber-300',
  sky: 'border-transparent bg-sky-100 text-sky-800 dark:bg-sky-500/20 dark:text-sky-300',
  violet: 'border-transparent bg-violet-100 text-violet-800 dark:bg-violet-500/20 dark:text-violet-300',
  slate: 'border-transparent bg-slate-100 text-slate-700 dark:bg-slate-500/20 dark:text-slate-300',
}

const byValue: Record<string, keyof typeof tones> = {
  active: 'green',
  paid: 'green',
  published: 'green',
  completed: 'green',
  passed: 'green',
  blocked: 'red',
  failed: 'red',
  dropped: 'red',
  draft: 'amber',
  SuperAdmin: 'violet',
  Instructor: 'sky',
  Student: 'slate',
}

/** A coloured label for a status or a role. */
export function StatusBadge({ value, label }: { value?: string; label?: string }) {
  const tone = tones[byValue[value ?? ''] ?? 'slate']

  return <Badge className={cn('font-medium', tone)}>{label ?? (value ? value.charAt(0).toUpperCase() + value.slice(1) : '')}</Badge>
}
