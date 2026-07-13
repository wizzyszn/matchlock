import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from 'react-router-dom'
import { Check, Clock, Loader2, Swords } from 'lucide-react'

import { buttonVariants } from '@/components/ui/button'
import { PageHeader, PageHeaderHeading, PageHeaderDescription } from '@/components/ui/page-header'
import { useSessionQuery } from '@/hooks/queries/use-session'
import { useWagerQuery } from '@/hooks/queries/use-wagers'
import { useApi } from '@/hooks/use-api'
import type { WagerInvite } from '@/lib/api'
import { formatStakeBaseUnits } from '@/lib/format'
import { queryKeys } from '@/lib/query-keys'
import { cn } from '@/lib/utils'

function inviteSideLabel(
  side: 'home' | 'draw' | 'away' | 'unset',
  homeTeam?: string,
  awayTeam?: string,
) {
  switch (side) {
    case 'home': return homeTeam ?? 'Home'
    case 'away': return awayTeam ?? 'Away'
    case 'draw': return 'Draw'
    case 'unset': return 'TBD'
  }
}

function inviteDisplayStatus(wagerStatus: string | undefined): { label: string; variant: 'pending' | 'accepted' | 'declined' } {
  if (!wagerStatus || wagerStatus === 'open') return { label: 'Pending', variant: 'pending' }
  if (wagerStatus === 'matched' || wagerStatus === 'settled') return { label: 'Accepted', variant: 'accepted' }
  return { label: 'Declined', variant: 'declined' }
}

function InviteCard({ invite, sessionEmail }: { invite: WagerInvite; sessionEmail: string }) {
  const navigate = useNavigate()
  const isRecipient = invite.recipient_email === sessionEmail
  const matchup =
    invite.home_team && invite.away_team
      ? `${invite.home_team} vs ${invite.away_team}`
      : `Match ${invite.match_id}`

  const { data: wager } = useWagerQuery(
    invite.wager_pubkey ? invite.wager_pubkey : undefined,
  )

  const status = inviteDisplayStatus(wager?.status)
  const isAccepted = status.variant === 'accepted'

  return (
    <li
      className={cn(
        'rounded-lg border bg-card p-4 space-y-2',
        isAccepted && 'cursor-pointer transition-colors hover:bg-muted/20',
      )}
      onClick={isAccepted ? () => navigate(`/my-wagers/${invite.wager_pubkey}`) : undefined}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="font-medium">{matchup}</p>
        <span
          className={cn(
            'inline-flex h-6 items-center gap-1 rounded-full border px-2.5 text-xs font-medium capitalize',
            status.variant === 'accepted'
              ? 'border-status-matched/25 bg-status-matched-bg text-status-matched'
              : 'border px-2 py-0.5',
          )}
        >
          {status.variant === 'accepted' ? (
            <Check className="size-3" aria-hidden />
          ) : (
            <Clock className="size-3" aria-hidden />
          )}
          {status.label}
        </span>
      </div>
      <p className="text-sm text-muted-foreground">
        {isRecipient
          ? `From ${invite.maker_email}`
          : `To ${invite.recipient_email}`}
        {' · '}
        {inviteSideLabel(invite.maker_side, invite.home_team, invite.away_team)}{' '}
        · {formatStakeBaseUnits(invite.stake)}
      </p>
      {invite.wager_pubkey && isRecipient && status.variant === 'pending' ? (
        <Link
          to={`/my-wagers/${invite.wager_pubkey}`}
          className={buttonVariants({ size: 'sm' })}
          onClick={(e) => e.stopPropagation()}
        >
          <Swords className="size-3.5 mr-1" aria-hidden />
          View open wager
        </Link>
      ) : null}
    </li>
  )
}

export function InvitesPage() {
  const api = useApi()
  const { data: session } = useSessionQuery()

  const invites = useQuery({
    queryKey: queryKeys.auth.invites,
    queryFn: () => api.listInvites(),
    enabled: Boolean(session),
  })

  if (!session) {
    return (
      <div className="text-center">
        <PageHeader className="mb-4">
          <PageHeaderHeading>Challenges</PageHeaderHeading>
          <PageHeaderDescription className="mx-auto">
            Sign in to send and receive direct challenges.
          </PageHeaderDescription>
        </PageHeader>
        <Link to="/login" className={buttonVariants()}>
          Sign in
        </Link>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader>
        <PageHeaderHeading>Challenges</PageHeaderHeading>
        <PageHeaderDescription>
          Direct head-to-head invites sent to your email.
        </PageHeaderDescription>
      </PageHeader>

      {invites.isLoading ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" aria-hidden />
          Loading invites…
        </div>
      ) : null}

      {invites.data?.length ? (
        <ul className="space-y-3">
          {invites.data.map((invite) => (
            <InviteCard key={invite.id} invite={invite} sessionEmail={session.email} />
          ))}
        </ul>
      ) : invites.isSuccess ? (
        <p className="text-sm text-muted-foreground">No challenges yet.</p>
      ) : null}
    </div>
  )
}
