import type { ComponentProps } from 'react'
import type { FieldError } from 'react-hook-form'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type Props = ComponentProps<typeof Input> & {
  label: string
  error?: FieldError
}

/** A label, an input and the message about it. */
export function FormField({ label, error, id, ...input }: Props) {
  const fieldId = id ?? input.name

  return (
    <div className="space-y-1.5">
      <Label htmlFor={fieldId}>{label}</Label>
      <Input id={fieldId} aria-invalid={error ? true : undefined} aria-describedby={error ? `${fieldId}-error` : undefined} {...input} />
      {error && (
        <p id={`${fieldId}-error`} className="text-sm text-destructive">
          {error.message}
        </p>
      )}
    </div>
  )
}
