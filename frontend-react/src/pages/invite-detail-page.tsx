import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Check, Clock, Loader2, ThumbsUp, ThumbsDown, ArrowLeft, SwordsIcon } from 'lucide-react'

import { Button, buttonVariants } from '@/components/ui/button'
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

function InviteDetail({ invite, sessionEmail }: { invite: WagerInvite; sessionEmail: string }) {
  const navigate = useNavigate()
  const api = useApi()
  const queryClient = useQueryClient()
  const isRecipient = invite.recipient_email === sessionEmail

  const { data: wager } = useWagerQuery(
    invite.wager_pubkey ? invite.wager_pubkey : undefined,
  )

  const status = inviteDisplayStatus(wager?.status)

  const updateInvite = useMutation({
    mutationFn: ({ id, status }: { id: string; status: 'accepted' | 'declined' }) =>
      api.updateInvite(id, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.auth.invites })
    },
  })

  const matchup =
    invite.home_team && invite.away_team
      ? `${invite.home_team} vs ${invite.away_team}`
      : `Match ${invite.match_id}`

  const isPending = invite.status === 'pending' && (!wager || wager.status === 'open')

  return (
    <div className="mx-auto max-w-lg space-y-6">
      <Button variant="ghost" size="sm" onClick={() => navigate('/invites')}>
        <ArrowLeft className="size-4 mr-1" /> Back to invites
      </Button>

      <div className="rounded-lg border bg-card p-6 space-y-5">
        <div className="space-y-1">
          <p className="text-sm text-muted-foreground">Challenge</p>
          <h2 className="text-xl font-semibold">{matchup}</h2>
        </div>

        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <p className="text-muted-foreground">From</p>
            <p className="font-medium truncate">{invite.maker_email}</p>
          </div>
          <div>
            <p className="text-muted-foreground">To</p>
            <p className="font-medium truncate">{invite.recipient_email}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Side</p>
            <p className="font-medium capitalize">{inviteSideLabel(invite.maker_side, invite.home_team, invite.away_team)}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Stake</p>
            <p className="font-medium">{formatStakeBaseUnits(invite.stake)}</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span
            className={cn(
              'inline-flex h-6 items-center gap-1 rounded-full border px-2.5 text-xs font-medium capitalize',
              status.variant === 'accepted'
                ? 'border-status-matched/25 bg-status-matched-bg text-status-matched'
                : status.variant === 'declined'
                  ? 'border-destructive/25 bg-destructive/10 text-destructive'
                  : 'border-border',
            )}
          >
            {status.variant === 'accepted' ? (
              <Check className="size-3" aria-hidden />
            ) : status.variant === 'declined' ? (
              <ThumbsDown className="size-3" aria-hidden />
            ) : (
              <Clock className="size-3" aria-hidden />
            )}
            {invite.status === 'pending' && (!wager || wager.status === 'open') ? 'Pending' : status.label}
          </span>
        </div>

        {isRecipient && isPending ? (
          <div className="flex gap-3 pt-2">
            <Button
              className="flex-1"
              disabled={updateInvite.isPending}
              onClick={() =>
                updateInvite.mutate(
                  { id: invite.id, status: 'accepted' },
                  {
                    onSuccess: () => {
                      if (invite.wager_pubkey) {
                        navigate(`/my-wagers/${invite.wager_pubkey}`)
                      }
                    },
                  },
                )
              }
            >
              {updateInvite.isPending ? (
                <Loader2 className="size-4 mr-1 animate-spin" />
              ) : (
                <ThumbsUp className="size-4 mr-1" />
              )}
              Accept
            </Button>
            <Button
              variant="outline"
              className="flex-1"
              disabled={updateInvite.isPending}
              onClick={() =>
                updateInvite.mutate({ id: invite.id, status: 'declined' })
              }
            >
              <ThumbsDown className="size-4 mr-1" />
              Decline
            </Button>
          </div>
        ) : null}

        {invite.wager_pubkey ? (
          <Link
            to={`/my-wagers/${invite.wager_pubkey}`}
            className={buttonVariants({ variant: 'outline', className: 'w-full' })}
          >
            <SwordsIcon className="size-4 mr-1" />
            View wager
          </Link>
        ) : null}
      </div>
    </div>
  )
}

export function InviteDetailPage() {
  const { id } = useParams()
  const api = useApi()
  const { data: session } = useSessionQuery()

  const inviteDetail = useQuery({
    queryKey: queryKeys.auth.invite(id!),
    queryFn: () => api.getInvite(id!),
    enabled: Boolean(id && session),
  })

  if (!session || inviteDetail.isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" aria-hidden />
        Loading invite…
      </div>
    )
  }

  if (inviteDetail.isError) {
    return (
      <div className="text-center">
        <PageHeader className="mb-4">
          <PageHeaderHeading>Invite not found</PageHeaderHeading>
          <PageHeaderDescription className="mx-auto">
            This invite may have expired or you don&apos;t have access to it.
          </PageHeaderDescription>
        </PageHeader>
        <Link to="/invites" className={buttonVariants({ variant: 'outline' })}>
          Back to invites
        </Link>
      </div>
    )
  }

  if (inviteDetail.data) {
    return <InviteDetail invite={inviteDetail.data} sessionEmail={session.email} />
  }

  return null
}
