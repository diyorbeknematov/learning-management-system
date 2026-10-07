import { useState, type ReactNode } from 'react'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'

/** A button that asks "are you sure?" before it does something that cannot be undone. */
export function ConfirmButton({
  children,
  title,
  description,
  confirmLabel = 'Delete',
  onConfirm,
  pending,
  size = 'sm',
  variant = 'ghost',
}: {
  children: ReactNode
  title: string
  description?: string
  confirmLabel?: string
  onConfirm: () => void
  pending?: boolean
  size?: 'xs' | 'sm' | 'default'
  variant?: 'ghost' | 'outline' | 'destructive'
}) {
  const [open, setOpen] = useState(false)

  return (
    <>
      <Button type="button" size={size} variant={variant} disabled={pending} onClick={() => setOpen(true)}>
        {children}
      </Button>
      <AlertDialog open={open} onOpenChange={setOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{title}</AlertDialogTitle>
            {description && <AlertDialogDescription>{description}</AlertDialogDescription>}
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => {
                setOpen(false)
                onConfirm()
              }}
            >
              {confirmLabel}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
