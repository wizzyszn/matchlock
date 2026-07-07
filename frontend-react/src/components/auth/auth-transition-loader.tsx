import type { ReactNode } from 'react'

import { cn } from '@/lib/utils'

type AuthTransitionLoaderProps = {
  title: string
  subtitle?: string
  className?: string
  icon?: ReactNode
}

export function AuthTransitionLoader({
  title,
  subtitle,
  className,
  icon,
}: AuthTransitionLoaderProps) {
  return (
    <div
      className={cn(
        'flex min-h-[50vh] flex-col items-center justify-center gap-6 py-16 text-center',
        className,
      )}
      role="status"
      aria-live="polite"
    >
      {icon ?? (
      <div className="relative size-20" aria-hidden>
        <span className="absolute inset-0 rounded-full border-2 border-primary/20" />
        <span className="absolute inset-2 animate-spin rounded-full border-2 border-transparent border-t-primary motion-reduce:animate-none" />
        <span className="absolute inset-5 animate-pulse rounded-full bg-primary/15 motion-reduce:animate-none" />
        <span className="absolute inset-0 flex items-center justify-center font-heading text-lg text-primary">
          M
        </span>
      </div>
      )}
      <div className="space-y-2">
        <p className="font-heading text-2xl tracking-tight">{title}</p>
        {subtitle ? (
          <p className="max-w-sm text-sm text-muted-foreground">{subtitle}</p>
        ) : null}
      </div>
      <div className="flex gap-1.5" aria-hidden>
        {[0, 1, 2].map((i) => (
          <span
            key={i}
            className="size-1.5 animate-pulse rounded-full bg-primary/60 motion-reduce:animate-none"
            style={{ animationDelay: `${i * 200}ms` }}
          />
        ))}
      </div>
    </div>
  )
}