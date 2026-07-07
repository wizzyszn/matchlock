import { X } from 'lucide-react'
import { useEffect, useRef, type ReactNode } from 'react'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export interface DialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  children: ReactNode
  className?: string
  /** When false, backdrop click and Escape do not close the dialog. */
  dismissible?: boolean
}

export function Dialog({
  open,
  onOpenChange,
  title,
  description,
  children,
  className,
  dismissible = true,
}: DialogProps) {
  const panelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return

    const onKeyDown = (event: KeyboardEvent) => {
      if (dismissible && event.key === 'Escape') onOpenChange(false)
    }

    document.addEventListener('keydown', onKeyDown)
    document.body.style.overflow = 'hidden'
    panelRef.current?.focus()

    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = ''
    }
  }, [open, onOpenChange, dismissible])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center sm:items-center">
      {dismissible ? (
        <button
          type="button"
          className="absolute inset-0 bg-foreground/20"
          aria-label="Close dialog"
          onClick={() => onOpenChange(false)}
        />
      ) : (
        <div className="absolute inset-0 bg-foreground/20" aria-hidden />
      )}
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="dialog-title"
        aria-describedby={description ? 'dialog-description' : undefined}
        tabIndex={-1}
        className={cn(
          'relative z-10 w-full max-w-md rounded-t-xl border bg-card p-6 shadow-sahara sm:rounded-xl',
          className,
        )}
      >
        <div className="mb-4 flex items-start justify-between gap-4">
          <div>
            <h2 id="dialog-title" className="font-heading text-2xl">
              {title}
            </h2>
            {description ? (
              <p
                id="dialog-description"
                className="mt-1 text-sm text-muted-foreground"
              >
                {description}
              </p>
            ) : null}
          </div>
          {dismissible ? (
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label="Close"
              onClick={() => onOpenChange(false)}
            >
              <X className="size-4" />
            </Button>
          ) : null}
        </div>
        {children}
      </div>
    </div>
  )
}