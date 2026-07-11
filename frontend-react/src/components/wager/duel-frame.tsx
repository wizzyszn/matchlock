import { TeamFlag } from '@/components/wager/team-flag'
import { cn } from '@/lib/utils'

export interface DuelFrameProps {
  home: string
  away: string
  league?: string
  isLive?: boolean
  scoreLine?: string
  size?: 'editorial' | 'compact' | 'dense'
  layout?: 'stack' | 'inline'
  className?: string
}

function TeamBlock({
  name,
  size,
  inline = false,
  align = 'start',
}: {
  name: string
  size: 'editorial' | 'compact' | 'dense'
  inline?: boolean
  align?: 'start' | 'end'
}) {
  const flagSize =
    size === 'editorial' ? 'xl' : size === 'compact' ? 'lg' : 'md'

  if (inline) {
    return (
      <div
        className={cn(
          'flex min-w-0 items-center gap-2',
          align === 'end' && 'flex-row-reverse justify-start text-right',
        )}
      >
        <TeamFlag name={name} size={flagSize} className="shrink-0" />
        <p
          className={cn(
            'truncate font-heading leading-tight text-foreground',
            size === 'dense' ? 'text-sm' : 'text-base',
          )}
        >
          {name}
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col items-center gap-2">
      <TeamFlag name={name} size={flagSize} />
      <p
        className={cn(
          'font-heading leading-tight text-foreground',
          size === 'editorial'
            ? 'text-3xl sm:text-4xl'
            : size === 'compact'
              ? 'text-xl sm:text-2xl'
              : 'text-base sm:text-lg',
        )}
      >
        {name}
      </p>
    </div>
  )
}

export function DuelFrame({
  home,
  away,
  league,
  isLive = false,
  scoreLine,
  size = 'compact',
  layout = 'stack',
  className,
}: DuelFrameProps) {
  const isInline = layout === 'inline' || size === 'dense'

  return (
    <div className={cn(isInline ? '' : 'text-center', className)}>
      {(league || isLive) && (
        <div
          className={cn(
            'flex items-center gap-2 text-xs text-muted-foreground',
            isInline ? 'mb-2' : 'mb-3 justify-center',
          )}
        >
          {league && <span>{league}</span>}
          {league && isLive && <span aria-hidden>·</span>}
          {isLive && (
            <span className="inline-flex items-center gap-1 font-medium text-status-open">
              <span
                className="size-1.5 rounded-full bg-status-open motion-safe:animate-pulse"
                aria-hidden
              />
              Live
            </span>
          )}
        </div>
      )}

      {isInline ? (
        <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-2 sm:gap-3">
          <TeamBlock name={home} size={size} inline align="end" />
          {scoreLine ? (
            <p
              className={cn(
                'min-w-10 px-0.5 text-center font-semibold leading-none tabular-nums text-foreground',
                size === 'dense' ? 'text-sm' : 'text-base',
              )}
              aria-label={`Score ${scoreLine}`}
            >
              {scoreLine}
            </p>
          ) : (
            <p
              className="px-0.5 text-[10px] font-medium tracking-widest text-muted-foreground uppercase"
              aria-hidden
            >
              vs
            </p>
          )}
          <TeamBlock name={away} size={size} inline />
        </div>
      ) : (
        <>
          <TeamBlock name={home} size={size} />

          <p
            className={cn(
              'my-2 font-medium tracking-widest text-muted-foreground uppercase',
              size === 'editorial' ? 'text-sm' : 'text-xs',
            )}
            aria-hidden
          >
            vs
          </p>

          <TeamBlock name={away} size={size} />
        </>
      )}
    </div>
  )
}
