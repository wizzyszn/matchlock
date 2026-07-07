import { cn } from '@/lib/utils'

const TXLINE_LOGO = '/txline-logo.jpg'

export type PoweredByTxLineProps = {
  className?: string
  size?: 'sm' | 'md'
}

export function PoweredByTxLine({
  className,
  size = 'md',
}: PoweredByTxLineProps) {
  const logoSize = size === 'sm' ? 'size-5' : 'size-7'

  return (
    <a
      href="https://txodds.com"
      target="_blank"
      rel="noopener noreferrer"
      className={cn(
        'inline-flex items-center gap-2 rounded-md text-muted-foreground transition-colors',
        'hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        className,
      )}
      aria-label="Powered by TxLINE — opens txodds.com in a new tab"
    >
      <span className={cn('text-xs font-medium tracking-wide', size === 'md' && 'text-sm')}>
        Powered by
      </span>
      <img
        src={TXLINE_LOGO}
        alt=""
        aria-hidden
        className={cn(logoSize, 'rounded-full object-cover ring-1 ring-border')}
      />
      <span className={cn('font-semibold tracking-tight text-foreground', size === 'sm' ? 'text-xs' : 'text-sm')}>
        TxLINE
      </span>
    </a>
  )
}