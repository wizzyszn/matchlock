import { Check, CircleX, Clock, Users } from 'lucide-react'

import { cn } from '@/lib/utils'
import type { WagerStatus } from '@/lib/api'

const STATUS_CONFIG: Record<
  WagerStatus,
  {
    label: string
    icon: typeof Clock
    className: string
  }
> = {
  open: {
    label: 'Open',
    icon: Clock,
    className: 'border-status-open/25 bg-status-open-bg text-status-open',
  },
  matched: {
    label: 'Matched',
    icon: Users,
    className: 'border-status-matched/25 bg-status-matched-bg text-status-matched',
  },
  settled: {
    label: 'Settled',
    icon: Check,
    className: 'border-status-settled/25 bg-status-settled-bg text-status-settled',
  },
  cancelled: {
    label: 'Cancelled',
    icon: CircleX,
    className:
      'border-status-cancelled/25 bg-status-cancelled-bg text-status-cancelled',
  },
}

export interface WagerStatusBadgeProps {
  status: WagerStatus
  className?: string
}

export function WagerStatusBadge({ status, className }: WagerStatusBadgeProps) {
  const config = STATUS_CONFIG[status]
  const Icon = config.icon

  return (
    <span
      className={cn(
        'inline-flex h-6 items-center gap-1 rounded-full border px-2.5 text-xs font-medium',
        config.className,
        className,
      )}
    >
      <Icon className="size-3" aria-hidden />
      {config.label}
    </span>
  )
}