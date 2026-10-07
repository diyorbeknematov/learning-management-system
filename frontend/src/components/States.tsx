import type { ReactNode } from 'react'
import { Skeleton } from '@/components/ui/skeleton'
import { errorMessage } from '@/lib/query'

export function LoadingBlock({ rows = 3 }: { rows?: number }) {
  return (
    <div className="space-y-3" aria-busy="true">
      {Array.from({ length: rows }, (_, i) => (
        <Skeleton key={i} className="h-16 w-full" />
      ))}
    </div>
  )
}

export function ErrorBlock({ error }: { error: unknown }) {
  return (
    <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">
      {errorMessage(error)}
    </p>
  )
}

export function Empty({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="rounded-lg border border-dashed p-8 text-center">
      <p className="font-medium">{title}</p>
      {children && <div className="mt-2 text-sm text-muted-foreground">{children}</div>}
    </div>
  )
}

/** Loading, error and empty states of a list in one place. */
export function ListState({
  query,
  empty,
  rows,
  children,
}: {
  query: { isPending: boolean; isError: boolean; error: unknown }
  empty?: ReactNode
  rows?: number
  children: ReactNode
}) {
  if (query.isPending) return <LoadingBlock rows={rows} />
  if (query.isError) return <ErrorBlock error={query.error} />
  if (empty) return <>{empty}</>

  return <>{children}</>
}
