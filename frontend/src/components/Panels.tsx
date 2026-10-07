import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

/** The box around a table: rounded, with a quiet header. */
export function TableCard({ title, actions, children, className }: { title?: ReactNode; actions?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <section className={cn('overflow-hidden rounded-xl border bg-card', className)}>
      {(title || actions) && (
        <header className="flex items-center justify-between gap-3 border-b px-5 py-4">
          <h2 className="font-semibold">{title}</h2>
          {actions}
        </header>
      )}
      {children}
    </section>
  )
}

/** The box around the filters of a list. */
export function Toolbar({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn('flex flex-wrap items-end gap-3 rounded-xl border bg-card p-4', className)}>{children}</div>
}
