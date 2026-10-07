import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

/** From and to dates (a day each); empty means no limit. */
export function DateRange({ from, to, onChange }: { from: string; to: string; onChange: (value: { from: string; to: string }) => void }) {
  return (
    <>
      <div className="space-y-1.5">
        <Label htmlFor="range-from">From</Label>
        <Input id="range-from" type="date" value={from} max={to || undefined} onChange={(e) => onChange({ from: e.target.value, to })} />
      </div>
      <div className="space-y-1.5">
        <Label htmlFor="range-to">To</Label>
        <Input id="range-to" type="date" value={to} min={from || undefined} onChange={(e) => onChange({ from, to: e.target.value })} />
      </div>
    </>
  )
}
