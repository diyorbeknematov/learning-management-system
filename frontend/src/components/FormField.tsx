import { Eye, EyeOff, type LucideIcon } from 'lucide-react'
import { useState, type ComponentProps } from 'react'
import type { FieldError } from 'react-hook-form'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'

type Props = ComponentProps<typeof Input> & {
  label: string
  error?: FieldError
  /** an icon at the start of the field */
  icon?: LucideIcon
  /** a taller field with larger text, for the sign-in pages */
  large?: boolean
  hint?: string
}

/** A label, an input and the message about it. A password can be shown with the eye. */
export function FormField({ label, error, id, icon: Icon, large, hint, className, type, ...input }: Props) {
  const fieldId = id ?? input.name
  const [shown, setShown] = useState(false)
  const isPassword = type === 'password'

  return (
    <div className="space-y-1.5">
      <Label htmlFor={fieldId} className={large ? 'text-sm font-medium' : undefined}>
        {label}
      </Label>

      <div className="relative">
        {Icon && <Icon className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />}

        <Input
          id={fieldId}
          type={isPassword && shown ? 'text' : type}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? `${fieldId}-error` : undefined}
          className={cn(large && 'h-11 rounded-lg text-base md:text-base', Icon && 'pl-10', isPassword && 'pr-11', className)}
          {...input}
        />

        {isPassword && (
          <button
            type="button"
            onClick={() => setShown((value) => !value)}
            className="absolute right-1.5 top-1/2 flex size-8 -translate-y-1/2 items-center justify-center rounded-md text-muted-foreground outline-none hover:bg-muted hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50"
            aria-label={shown ? 'Hide' : 'Show'}
            aria-pressed={shown}
          >
            {shown ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
          </button>
        )}
      </div>

      {error ? (
        <p id={`${fieldId}-error`} className="text-sm text-destructive">
          {error.message}
        </p>
      ) : (
        hint && <p className="text-xs text-muted-foreground">{hint}</p>
      )}
    </div>
  )
}
