import { format, parse } from 'date-fns'
import { CalendarDays, X } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/lib/utils'

const day = 'yyyy-MM-dd'

// the date the page was opened on; a reference for parsing and for the last year of the calendar
const opened = new Date()

/**
 * A day picked from a calendar that opens under the field. The value is
 * "2026-05-01", or empty when no day is chosen.
 */
export function DatePicker({
  id,
  value,
  onChange,
  placeholder = 'Pick a date',
  min,
  max,
  className,
}: {
  id?: string
  value: string
  onChange: (value: string) => void
  placeholder?: string
  /** the earliest and the latest day that can be chosen */
  min?: string
  max?: string
  className?: string
}) {
  const [open, setOpen] = useState(false)

  const selected = value ? parse(value, day, opened) : undefined
  const from = min ? parse(min, day, opened) : undefined
  const to = max ? parse(max, day, opened) : undefined

  function pickToday() {
    choose(format(new Date(), day))
  }

  function choose(next: string) {
    onChange(next)
    setOpen(false)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        id={id}
        className={cn(
          'flex h-10 w-full items-center gap-2 rounded-lg border border-input bg-muted/50 px-3 text-left text-sm outline-none transition hover:bg-muted focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 aria-expanded:border-ring aria-expanded:ring-3 aria-expanded:ring-ring/30',
          !selected && 'text-muted-foreground',
          className,
        )}
      >
        <CalendarDays className="size-4 shrink-0 text-muted-foreground" />
        <span className="flex-1 truncate">{selected ? format(selected, 'd MMM yyyy') : placeholder}</span>
        {selected && (
          <span
            role="button"
            tabIndex={0}
            aria-label="Clear the date"
            className="-mr-1 flex size-5 items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground"
            onClick={(e) => {
              e.stopPropagation()
              onChange('')
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.stopPropagation()
                onChange('')
              }
            }}
          >
            <X className="size-3.5" />
          </span>
        )}
      </PopoverTrigger>

      <PopoverContent align="start" className="w-auto p-0">
        <Calendar
          mode="single"
          selected={selected}
          defaultMonth={selected ?? to ?? opened}
          onSelect={(date) => choose(date ? format(date, day) : '')}
          disabled={[...(from ? [{ before: from }] : []), ...(to ? [{ after: to }] : [])]}
          captionLayout="dropdown"
          startMonth={new Date(2020, 0)}
          endMonth={new Date(opened.getFullYear() + 1, 11)}
          className="p-3 [--cell-size:--spacing(9)]"
        />
        <div className="flex items-center justify-between border-t px-3 py-2">
          <Button type="button" variant="ghost" size="sm" onClick={() => choose('')}>
            Clear
          </Button>
          <Button type="button" variant="ghost" size="sm" onClick={pickToday}>
            Today
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  )
}
