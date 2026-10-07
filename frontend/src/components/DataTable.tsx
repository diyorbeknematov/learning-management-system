import { TableCard } from '@/components/Panels'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

type Row = Record<string, unknown>

const title = (key: string) => {
  const text = key.replaceAll('_', ' ')

  return text.charAt(0).toUpperCase() + text.slice(1)
}

function cell(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'number') return Number.isInteger(value) ? String(value) : value.toFixed(2)
  if (typeof value === 'boolean') return value ? 'Yes' : 'No'
  if (typeof value === 'string' && /^\d{4}-\d{2}-\d{2}T/.test(value)) return value.slice(0, 10)
  if (typeof value === 'object') return JSON.stringify(value)

  return String(value)
}

/** A table whose columns are the keys of the rows; ids are left out. */
export function DataTable({ rows }: { rows: Row[] }) {
  if (rows.length === 0) return <p className="rounded-xl border border-dashed p-8 text-center text-sm text-muted-foreground">No data for this period.</p>

  const columns = Object.keys(rows[0]).filter((key) => key !== 'id' && !key.endsWith('_id') && !key.endsWith('_ids'))

  return (
    <TableCard>
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((key) => (
              <TableHead key={key}>{title(key)}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row, index) => (
            <TableRow key={index}>
              {columns.map((key) => (
                <TableCell key={key}>{cell(row[key])}</TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableCard>
  )
}

export { title as columnTitle }
