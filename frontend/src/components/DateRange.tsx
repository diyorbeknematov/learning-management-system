import { endOfMonth, format, startOfMonth, startOfYear, subDays } from 'date-fns'
import { DatePicker } from '@/components/DatePicker'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'

type Range = { from: string; to: string }

const day = 'yyyy-MM-dd'

// the periods are counted from the day the page was opened
const opened = new Date()

/** From and to days, each picked from a calendar; empty means no limit. */
export function DateRange({ from, to, onChange }: Range & { onChange: (value: Range) => void }) {
  return (
    <>
      <div className="space-y-1.5">
        <Label htmlFor="range-from">From</Label>
        <DatePicker id="range-from" value={from} max={to || undefined} placeholder="Any day" onChange={(value) => onChange({ from: value, to })} />
      </div>
      <div className="space-y-1.5">
        <Label htmlFor="range-to">To</Label>
        <DatePicker id="range-to" value={to} min={from || undefined} placeholder="Any day" onChange={(value) => onChange({ from, to: value })} />
      </div>
    </>
  )
}

/** One click for the periods that are asked for most. */
export function RangePresets({ from, to, onChange }: Range & { onChange: (value: Range) => void }) {
  const today = opened

  const presets: { label: string; range: Range }[] = [
    { label: 'Last 7 days', range: { from: format(subDays(today, 6), day), to: format(today, day) } },
    { label: 'Last 30 days', range: { from: format(subDays(today, 29), day), to: format(today, day) } },
    { label: 'This month', range: { from: format(startOfMonth(today), day), to: format(endOfMonth(today), day) } },
    { label: 'This year', range: { from: format(startOfYear(today), day), to: format(today, day) } },
    { label: 'All time', range: { from: '', to: '' } },
  ]

  return (
    <div className="flex min-h-10 flex-wrap items-center gap-1.5" role="group" aria-label="Period">
      {presets.map((preset) => {
        const active = preset.range.from === from && preset.range.to === to

        return (
          <Button
            key={preset.label}
            type="button"
            size="sm"
            variant={active ? 'default' : 'outline'}
            className="rounded-full"
            onClick={() => onChange(preset.range)}
          >
            {preset.label}
          </Button>
        )
      })}
    </div>
  )
}
