import { Star } from 'lucide-react'
import { cn } from '@/lib/utils'

/** Stars for a rating from 0 to 5, with the number and the count of reviews. */
export function Rating({ value, count, className }: { value?: number; count?: number; className?: string }) {
  const rounded = Math.round(value ?? 0)

  return (
    <span className={cn('inline-flex items-center gap-1 text-sm', className)}>
      <span className="inline-flex" aria-hidden>
        {[1, 2, 3, 4, 5].map((n) => (
          <Star key={n} className={cn('size-4', n <= rounded ? 'fill-amber-400 text-amber-400' : 'text-muted-foreground/40')} />
        ))}
      </span>
      <span className="font-medium">{value ? value.toFixed(1) : 'New'}</span>
      {count !== undefined && <span className="text-muted-foreground">({count})</span>}
      <span className="sr-only">rating {value?.toFixed(1) ?? 'none'} of 5</span>
    </span>
  )
}
