import { Plus, X } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

/** A list of short points: type one, press Enter or "Add", and it joins the list below. */
export function ListInput({
  id,
  label,
  placeholder,
  items,
  onChange,
}: {
  id: string
  label: string
  placeholder: string
  items: string[]
  onChange: (items: string[]) => void
}) {
  const [text, setText] = useState('')

  const add = () => {
    const value = text.trim()

    if (!value) return

    onChange([...items, value])
    setText('')
  }

  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <div className="flex gap-2">
        <Input
          id={id}
          value={text}
          placeholder={placeholder}
          className="h-11 text-base"
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              add()
            }
          }}
        />
        <Button type="button" variant="outline" className="h-11" disabled={!text.trim()} onClick={add}>
          <Plus /> Add
        </Button>
      </div>

      {items.length > 0 && (
        <ul className="divide-y rounded-lg border bg-muted/30">
          {items.map((item, index) => (
            <li key={`${item}-${index}`} className="flex items-center gap-2 px-3 py-2 text-sm">
              <span className="min-w-0 flex-1 break-words">{item}</span>
              <Button type="button" size="icon-xs" variant="ghost" aria-label={`Remove ${item}`} onClick={() => onChange(items.filter((_, i) => i !== index))}>
                <X />
              </Button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
