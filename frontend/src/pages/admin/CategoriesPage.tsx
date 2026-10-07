import { Plus, Tags } from 'lucide-react'
import { useState } from 'react'
import { api, call } from '@/api/client'
import { useCategories } from '@/api/queries'
import type { components } from '@/api/schema'
import { ConfirmButton } from '@/components/ConfirmButton'
import { PageHeader } from '@/components/PageHeader'
import { TableCard } from '@/components/Panels'
import { Empty, ErrorBlock, LoadingBlock } from '@/components/States'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { formatDate, plural } from '@/lib/format'
import { useApiMutation } from '@/lib/mutations'

type Category = components['schemas']['models.Category']

function CategoryDialog({ category, onClose }: { category?: Category; onClose: () => void }) {
  const [name, setName] = useState(category?.name ?? '')
  const [description, setDescription] = useState(category?.description ?? '')

  const save = useApiMutation(
    () => {
      const body = { name: name.trim(), description: description.trim() }

      return category ? call(api.PUT('/categories/{id}', { params: { path: { id: category.id! } }, body })) : call(api.POST('/categories', { body }))
    },
    { success: category ? 'Category saved' : 'Category created', invalidate: [['categories'], ['courses']], onSuccess: onClose },
  )

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <form
          className="grid gap-4"
          onSubmit={(e) => {
            e.preventDefault()
            save.mutate(undefined)
          }}
        >
          <DialogHeader>
            <DialogTitle>{category ? 'Edit the category' : 'New category'}</DialogTitle>
          </DialogHeader>
          <div className="space-y-1.5">
            <Label htmlFor="category-name">Name</Label>
            <Input id="category-name" value={name} maxLength={100} onChange={(e) => setName(e.target.value)} required autoFocus />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="category-description">Description (optional)</Label>
            <Input id="category-description" value={description} onChange={(e) => setDescription(e.target.value)} />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={save.isPending || !name.trim()}>
              {save.isPending ? 'Saving…' : 'Save'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function Row({ category, onEdit }: { category: Category; onEdit: () => void }) {
  const remove = useApiMutation(() => call(api.DELETE('/categories/{id}', { params: { path: { id: category.id! } } })), {
    success: 'Category deleted',
    invalidate: [['categories'], ['courses']],
  })

  return (
    <TableRow>
      <TableCell>
        <div className="flex items-center gap-3">
          <span className="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <Tags className="size-4" />
          </span>
          <span className="font-medium">{category.name}</span>
        </div>
      </TableCell>
      <TableCell className="text-muted-foreground">{category.description || '—'}</TableCell>
      <TableCell className="text-muted-foreground">{formatDate(category.created_at)}</TableCell>
      <TableCell className="text-right">
        <div className="flex justify-end gap-1">
          <Button size="xs" variant="outline" onClick={onEdit}>
            Edit
          </Button>
          <ConfirmButton
            size="xs"
            title={`Delete "${category.name}"?`}
            description="A category that has courses cannot be deleted."
            onConfirm={() => remove.mutate(undefined)}
            pending={remove.isPending}
          >
            Delete
          </ConfirmButton>
        </div>
      </TableCell>
    </TableRow>
  )
}

export default function CategoriesPage() {
  const categories = useCategories()
  const [dialog, setDialog] = useState<{ category?: Category } | null>(null)

  return (
    <div className="space-y-6">
      <PageHeader
        title="Categories"
        description={categories.data ? `${plural(categories.data.total, 'category', 'categories')}. Courses are grouped by them in the catalog.` : undefined}
        actions={
          <Button onClick={() => setDialog({})}>
            <Plus /> New category
          </Button>
        }
      />

      {categories.isPending && <LoadingBlock />}
      {categories.isError && <ErrorBlock error={categories.error} />}
      {categories.data?.items?.length === 0 && <Empty title="No categories yet" />}

      {(categories.data?.items?.length ?? 0) > 0 && (
        <TableCard>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Description</TableHead>
                <TableHead>Created</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {categories.data?.items?.map((category) => (
                <Row key={category.id} category={category} onEdit={() => setDialog({ category })} />
              ))}
            </TableBody>
          </Table>
        </TableCard>
      )}

      {dialog && <CategoryDialog category={dialog.category} onClose={() => setDialog(null)} />}
    </div>
  )
}
