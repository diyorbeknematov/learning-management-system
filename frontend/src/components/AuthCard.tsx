import type { ReactNode } from 'react'

/** The heading, the form and the line below it of the login, register and password pages. */
export function AuthCard({
  title,
  description,
  footer,
  children,
}: {
  title: string
  description?: string
  footer?: ReactNode
  children: ReactNode
}) {
  return (
    <div className="space-y-6">
      <div className="space-y-1.5">
        <h1 className="text-3xl font-bold tracking-tight">{title}</h1>
        {description && <p className="text-muted-foreground">{description}</p>}
      </div>

      {children}

      {footer && <div className="text-sm text-muted-foreground [&_a]:font-medium [&_a]:text-primary">{footer}</div>}
    </div>
  )
}
