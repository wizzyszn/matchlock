import { TeamFlag, type TeamFlagProps } from '@/components/wager/team-flag'
import { cn } from '@/lib/utils'

export type TeamRowProps = {
  name: string
  code?: TeamFlagProps['code']
  flagSize?: TeamFlagProps['size']
  className?: string
}

export function TeamRow({
  name,
  code,
  flagSize = 'sm',
  className,
}: TeamRowProps) {
  return (
    <div className={cn('flex min-w-0 items-center gap-2', className)}>
      <TeamFlag name={name} code={code} size={flagSize} />
      <span className="truncate font-medium text-foreground">{name}</span>
    </div>
  )
}