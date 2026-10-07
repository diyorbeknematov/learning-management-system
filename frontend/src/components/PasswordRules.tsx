import { Check, Circle } from 'lucide-react'
import { cn } from '@/lib/utils'

const rules = [
  { label: 'At least 8 characters', test: (v: string) => v.length >= 8 },
  { label: 'A capital letter', test: (v: string) => /\p{Lu}/u.test(v) },
  { label: 'A small letter', test: (v: string) => /\p{Ll}/u.test(v) },
  { label: 'A digit', test: (v: string) => /\p{Nd}/u.test(v) },
  { label: 'A special character', test: (v: string) => /[\p{P}\p{S}]/u.test(v) },
]

/** The rules of a password, ticked while the person types. */
export function PasswordRules({ value }: { value?: string }) {
  const text = value ?? ''

  return (
    <ul className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs" aria-label="Requirements">
      {rules.map((rule) => {
        const ok = rule.test(text)

        return (
          <li key={rule.label} className={cn('flex items-center gap-1.5', ok ? 'text-emerald-600' : 'text-muted-foreground')}>
            {ok ? <Check className="size-3.5" /> : <Circle className="size-3" />} {rule.label}
          </li>
        )
      })}
    </ul>
  )
}
