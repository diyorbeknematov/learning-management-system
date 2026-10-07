const dollars = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' })
const dateFormat = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })

/** 0 or nothing is "Free". */
export function money(amount?: number | null): string {
  return amount ? dollars.format(amount) : 'Free'
}

/** A sum of money, also when it is zero. */
export function moneyExact(amount?: number | null): string {
  return dollars.format(amount ?? 0)
}

/** Minutes as "1 h 30 min". */
export function duration(minutes?: number | null): string {
  if (!minutes) return '—'

  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60

  if (hours === 0) return `${rest} min`

  return rest === 0 ? `${hours} h` : `${hours} h ${rest} min`
}

export function formatDate(value?: string | null): string {
  if (!value) return '—'

  const date = new Date(value)

  return Number.isNaN(date.getTime()) ? '—' : dateFormat.format(date)
}

export function fullName(person?: { first_name?: string; last_name?: string; username?: string } | null): string {
  if (!person) return ''

  return `${person.first_name ?? ''} ${person.last_name ?? ''}`.trim() || person.username || ''
}

export function capitalize(text?: string | null): string {
  return text ? text.charAt(0).toUpperCase() + text.slice(1) : ''
}
