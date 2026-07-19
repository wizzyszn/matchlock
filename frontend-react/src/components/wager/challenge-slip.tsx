import { useWallet } from '@solana/wallet-adapter-react'
import { PublicKey } from '@solana/web3.js'
import { Loader2, Search } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { toast } from 'sonner'

import { useMediaQuery } from '@/hooks/use-media-query'
import usdtLogo from '@/assets/usdt-svgrepo-com.svg'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  ConfirmTxDialog,
  type ConfirmTxDetails,
} from '@/components/wager/confirm-tx-dialog'
import { DuelFrame } from '@/components/wager/duel-frame'
import { FixturePicker } from '@/components/wager/fixture-picker'
import { OutcomePicker } from '@/components/wager/outcome-picker'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { PageHeader, PageHeaderHeading, PageHeaderDescription } from '@/components/ui/page-header'
import { useAuthMutations } from '@/hooks/mutations/use-auth-mutations'
import { useMatchesQuery } from '@/hooks/queries/use-matches'
import { useSessionQuery } from '@/hooks/queries/use-session'
import { useWalletLinkStatus } from '@/hooks/use-wallet-link-status'
import { useTokenBalanceQuery } from '@/hooks/queries/use-token-balance'
import { useWagerMutations } from '@/hooks/mutations/use-wager-mutations'
import type { UserLookup } from '@/lib/api'
import { useTxFeeEstimate } from '@/hooks/use-tx-fee-estimate'
import { useWagerTxBuilders } from '@/hooks/use-wager-tx-builders'
import { useStablecoinLabel } from '@/hooks/use-stablecoin-label'
import type { Match, Side } from '@/lib/api'
import { baseUnitsToUsdt, formatUsdt, usdtToBaseUnits } from '@/lib/format'
import {
  classifyMatch,
  formatKickoffClock,
  formatKickoffDate,
  matchLabels,
} from '@/lib/match-display'
import { sideLabel } from '@/lib/wager-sides'

// ─── Zod schema ──────────────────────────────────────────────────────────────

const slipSchema = z.object({
  stake: z
    .string()
    .min(1, 'Stake is required')
    .refine(
      (v) => Number.isFinite(Number(v)) && Number(v) > 0,
      'Must be a positive amount',
    ),
  friendEmail: z
    .string()
    .email('Enter a valid email address')
    .or(z.literal(''))
    .optional(),
})

type SlipFormValues = z.infer<typeof slipSchema>

// ─── Types ────────────────────────────────────────────────────────────────────

type ChallengeSlipProps = {
  match: Match
  initialSide?: Side
  open?: boolean
  onOpenChange?: (open: boolean) => void
  variant?: 'dialog' | 'page'
}

// ─── Body ─────────────────────────────────────────────────────────────────────

function ChallengeSlipBody({
  match,
  initialSide = 'home',
  onSuccess,
}: {
  match: Match
  initialSide?: Side
  onSuccess?: () => void
}) {
  const navigate = useNavigate()
  const { publicKey } = useWallet()
  const { data: balance = BigInt(0) } = useTokenBalanceQuery()
  const { data: session } = useSessionQuery()
  const { lookupUser, createInvite } = useAuthMutations()
  const { makeWager, mapError, isWalletReady, walletNeedsLink } = useWagerMutations()
  const walletStatus = useWalletLinkStatus()
  const { buildMake, wallet } = useWagerTxBuilders()
  const stablecoin = useStablecoinLabel()

  const [side, setSide] = useState<Side>(initialSide)
  const [challengeMode, setChallengeMode] = useState<'open' | 'friend'>('open')
  const [friendLookup, setFriendLookup] = useState<UserLookup | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [confirmDetails, setConfirmDetails] = useState<ConfirmTxDetails | null>(null)
  const [txError, setTxError] = useState<string | null>(null)
  const [signature, setSignature] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors },
  } = useForm<SlipFormValues>({
    resolver: zodResolver(slipSchema),
    defaultValues: { stake: '', friendEmail: '' },
  })

  const stakeRaw = watch('stake')
  const friendEmail = watch('friendEmail') ?? ''

  useEffect(() => {
    setSide(initialSide)
  }, [initialSide, match.match_id])

  const labels = matchLabels(match)
  const stakeUsdt = Number.parseFloat(stakeRaw)
  const stakeValid = Number.isFinite(stakeUsdt) && stakeUsdt > 0
  const balanceUsdt = baseUnitsToUsdt(balance)
  const insufficientBalance = stakeValid && stakeUsdt > balanceUsdt
  const matchPhase = classifyMatch(match)
  const canChallenge = matchPhase === 'upcoming' || matchPhase === 'live'
  const friendEmailNormalized = friendEmail.trim().toLowerCase()
  const isSelfChallenge =
    Boolean(session) &&
    friendEmailNormalized.length > 3 &&
    friendEmailNormalized === session?.email.toLowerCase()
  const friendReady =
    challengeMode === 'open' ||
    (Boolean(session) && friendLookup !== null && !isSelfChallenge)

  const handleLookupFriend = async () => {
    const email = friendEmail.trim()
    if (session && email.toLowerCase() === session.email.toLowerCase()) {
      setFriendLookup({
        email,
        has_account: true,
        user_id: session.id,
        primary_wallet: walletStatus.primaryWallet?.pubkey,
      })
      return
    }
    const result = await lookupUser.mutateAsync(email)
    setFriendLookup(result)
  }

  const openConfirm = () => {
    if (challengeMode === 'friend' && !friendReady) return
    if (!stakeValid || insufficientBalance || !canChallenge) return
    setConfirmDetails({
      action: 'make',
      matchLabel: labels.league,
      sideLabel: sideLabel(side, match),
      stakeUsdt,
      payoutUsdt: stakeUsdt * 2,
    })
    setTxError(null)
    setSignature(null)
    setDialogOpen(true)
  }

  const estimateFee = useCallback(async () => {
    if (!confirmDetails || confirmDetails.action !== 'make' || !stakeValid) return null
    const invitedTaker =
      challengeMode === 'friend' && friendLookup?.primary_wallet
        ? new PublicKey(friendLookup.primary_wallet)
        : undefined
    return buildMake({
      matchId: match.match_id,
      stake: usdtToBaseUnits(stakeUsdt),
      makerSide: side,
      participant1IsHome: match.participant1_is_home,
      invitedTaker,
    })
  }, [buildMake, challengeMode, confirmDetails, friendLookup?.primary_wallet, match.match_id, match.participant1_is_home, side, stakeUsdt, stakeValid])

  const feeEstimate = useTxFeeEstimate({
    enabled: dialogOpen && confirmDetails?.action === 'make',
    estimateKey: `make-${match.match_id}-${side}-${stakeUsdt}`,
    buildTx: estimateFee,
  })

  const handleConfirm = async () => {
    if (!stakeValid) return
    setTxError(null)
    try {
      const invitedTaker =
        challengeMode === 'friend' && friendLookup?.primary_wallet
          ? new PublicKey(friendLookup.primary_wallet)
          : undefined
      const { signature, wagerPubkey } = await makeWager.mutateAsync({
        matchId: match.match_id,
        stake: usdtToBaseUnits(stakeUsdt),
        makerSide: side,
        participant1IsHome: match.participant1_is_home,
        invitedTaker,
      })

      if (challengeMode === 'friend' && session && friendEmail.trim()) {
        try {
          await createInvite.mutateAsync({
            recipient_email: friendEmail.trim(),
            wager_pubkey: wagerPubkey,
            match_id: match.match_id,
            maker_side: side,
            stake: Number(usdtToBaseUnits(stakeUsdt)),
            home_team: labels.homeTeam,
            away_team: labels.awayTeam,
          })
        } catch {
          toast.warning('Wager created, but the invite could not be sent.')
        }
      }

      setSignature(signature)
      setDialogOpen(false)
      reset()
      setFriendLookup(null)
      toast.success('Wager created successfully.')
      onSuccess?.()
      navigate('/my-wagers')
    } catch (error) {
      const message = mapError(error)
      setTxError(message)
      toast.error(message)
    }
  }

  return (
    <>
      {/* Match identity */}
      <div className="space-y-1 mb-6">
        <DuelFrame
          home={labels.homeTeam}
          away={labels.awayTeam}
          league={labels.league}
          isLive={labels.isLive}
          size="compact"
          layout="inline"
        />
        <div className="flex items-center justify-center gap-2 pt-1 text-xs text-muted-foreground">
          <span>{formatKickoffDate(match)}</span>
          <span aria-hidden>·</span>
          <span className="tabular-nums font-medium text-foreground">{formatKickoffClock(match)}</span>
        </div>
      </div>

      {!canChallenge ? (
        <p className="rounded-md border border-dashed bg-muted/40 px-3 py-3 text-sm text-muted-foreground text-center">
          This fixture is no longer available for wagering. Pick an upcoming or live match.
        </p>
      ) : (
        <form onSubmit={handleSubmit(openConfirm)} noValidate className="space-y-6">

          {/* Outcome */}
          <OutcomePicker
            match={match}
            sides={['home', 'draw', 'away']}
            selected={side}
            onSelect={setSide}
            hint="Reference odds from TxLINE — your opponent takes one of the other outcomes."
          />

          {/* Who can accept */}
          <div className="space-y-3">
            <p className="text-sm font-medium">Who can accept?</p>
            <div className="flex gap-2">
              <Button
                type="button"
                size="sm"
                variant={challengeMode === 'open' ? 'default' : 'outline'}
                className="rounded-full"
                onClick={() => setChallengeMode('open')}
              >
                Anyone
              </Button>
              <Button
                type="button"
                size="sm"
                variant={challengeMode === 'friend' ? 'default' : 'outline'}
                className="rounded-full"
                onClick={() => setChallengeMode('friend')}
              >
                A friend
              </Button>
            </div>

            {challengeMode === 'friend' && (
              <div className="space-y-2">
                {!session ? (
                  <p className="text-sm text-muted-foreground">
                    <Link to="/login" className="text-primary hover:underline">Sign in</Link>{' '}
                    to challenge someone by email.
                  </p>
                ) : (
                  <>
                    <div className="flex gap-2">
                      <Input
                        type="email"
                        placeholder="friend@example.com"
                        className="flex-1"
                        {...register('friendEmail')}
                        onChange={(e) => {
                          void register('friendEmail').onChange(e)
                          setFriendLookup(null)
                        }}
                      />
                      <Button
                        type="button"
                        size="icon"
                        variant="outline"
                        className="shrink-0"
                        disabled={!friendEmail.trim() || lookupUser.isPending}
                        onClick={() => void handleLookupFriend()}
                        aria-label="Find friend"
                      >
                        {lookupUser.isPending ? <Loader2 className="size-4 animate-spin" /> : <Search className="size-4 text-muted-foreground" />}
                      </Button>
                    </div>
                    {errors.friendEmail && (
                      <p className="text-xs text-destructive">{errors.friendEmail.message}</p>
                    )}
                    {friendLookup && (
                      <div className="rounded-md bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
                        {isSelfChallenge ? (
                          <p>Direct self-challenges aren&apos;t supported — use <strong>Anyone</strong>.</p>
                        ) : friendLookup.primary_wallet ? (
                          <p>
                            Found — only{' '}
                            <span className="font-mono text-foreground">
                              {friendLookup.primary_wallet.slice(0, 4)}…{friendLookup.primary_wallet.slice(-4)}
                            </span>{' '}
                            can accept.
                          </p>
                        ) : (
                          <p>They&apos;ll receive an invite by email once you create this wager.</p>
                        )}
                      </div>
                    )}
                  </>
                )}
              </div>
            )}
          </div>

          {/* Stake */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <label htmlFor="challenge-stake" className="text-sm font-medium flex items-center gap-1.5">
                <img src={usdtLogo} alt="USDT" className="size-4" />
                Stake · {stablecoin}
              </label>
              {isWalletReady && (
                <button
                  type="button"
                  className="text-xs text-primary underline-offset-4 hover:underline"
                  onClick={() => setValue('stake', balanceUsdt.toString())}
                >
                  Max {formatUsdt(balanceUsdt)}
                </button>
              )}
            </div>
            <Input
              id="challenge-stake"
              type="number"
              min="0"
              inputMode="decimal"
              placeholder="0.00"
              {...register('stake')}
            />
            {errors.stake && (
              <p className="text-xs text-destructive">{errors.stake.message}</p>
            )}
            {!errors.stake && insufficientBalance && (
              <p className="text-xs text-destructive">Insufficient {stablecoin} balance.</p>
            )}
          </div>

          {/* Summary */}
          {stakeValid && !insufficientBalance && (
            <div className="space-y-1 text-sm">
              <div className="flex justify-between text-muted-foreground">
                <span>You back</span>
                <span className="font-medium text-foreground">{sideLabel(side, match)}</span>
              </div>
              <div className="flex justify-between text-muted-foreground">
                <span>If you win</span>
                <span className="font-semibold text-primary tabular-nums">
                  <img src={usdtLogo} alt="" className="inline-block size-3.5 -mt-px mr-0.5" />
                  {formatUsdt(stakeUsdt * 2)} {stablecoin}
                </span>
              </div>
            </div>
          )}

          {/* CTA */}
          <Button
            type="submit"
            className="w-full min-h-11"
            size="lg"
            disabled={
              !isWalletReady ||
              walletNeedsLink ||
              !stakeValid ||
              insufficientBalance ||
              !canChallenge ||
              !friendReady ||
              makeWager.isPending
            }
          >
            {makeWager.isPending ? (
              <>
                <Loader2 className="size-4 animate-spin" />
                Creating…
              </>
            ) : (
              <>
                {/* <Swords className="size-4" aria-hidden /> */}
                Create challenge
              </>
            )}
          </Button>
        </form>
      )}

      <ConfirmTxDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        details={confirmDetails}
        pending={makeWager.isPending || createInvite.isPending}
        error={txError}
        signature={signature}
        feeEstimate={feeEstimate}
        feePayerAddress={wallet?.publicKey?.toBase58() ?? publicKey?.toBase58()}
        onConfirm={() => void handleConfirm()}
      />
    </>
  )
}

// ─── Dialog wrapper ───────────────────────────────────────────────────────────

export function ChallengeSlipDialog({
  match,
  initialSide = 'home',
  open,
  onOpenChange,
}: Omit<ChallengeSlipProps, 'variant' | 'open' | 'onOpenChange'> & {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const isDesktop = useMediaQuery('(min-width: 40rem)')

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      {isDesktop ? (
        <SheetContent
          side="right"
          className="w-full sm:max-w-[400px] overflow-y-auto px-5 sm:px-7"
        >
          <SheetHeader className="mb-6 text-left">
            <SheetTitle>Create challenge</SheetTitle>
            <SheetDescription className="text-xs">
              Pick your outcome, set your stake, and post an open PvP wager.
            </SheetDescription>
          </SheetHeader>
          <div className="pb-8 w-full">
            <ChallengeSlipBody
              match={match}
              initialSide={initialSide}
              onSuccess={() => onOpenChange(false)}
            />
          </div>
        </SheetContent>
      ) : (
        <SheetContent
          side="bottom"
          showCloseButton={false}
          className="max-h-[90dvh] overflow-y-auto rounded-t-2xl border-t px-5 pb-10 pt-6"
        >
          <div className="mb-4 flex items-center justify-center">
            <div className="h-1.5 w-12 rounded-full bg-border" />
          </div>
          <SheetHeader className="mb-6 text-left px-0">
            <SheetTitle>Create challenge</SheetTitle>
            <SheetDescription className="text-xs">
              Pick your outcome, set your stake, and post an open PvP wager.
            </SheetDescription>
          </SheetHeader>
          <ChallengeSlipBody
            match={match}
            initialSide={initialSide}
            onSuccess={() => onOpenChange(false)}
          />
        </SheetContent>
      )}
    </Sheet>
  )
}

// ─── Page wrapper ─────────────────────────────────────────────────────────────

export function ChallengeSlipPage({
  matchId,
  initialSide,
}: {
  matchId?: string
  initialSide?: Side
}) {
  const { data: matches, isLoading } = useMatchesQuery()

  const openMatches = useMemo(
    () => (matches ?? []).filter((match) => {
      const phase = classifyMatch(match)
      return phase !== 'finished' && phase !== 'pending'
    }),
    [matches],
  )

  const [selectedId, setSelectedId] = useState(matchId ?? '')
  useEffect(() => {
    if (matchId) setSelectedId(matchId)
  }, [matchId])

  const selectedMatch =
    openMatches.find((match) => match.match_id === selectedId) ?? openMatches[0]

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
        Loading fixtures…
      </div>
    )
  }

  if (!openMatches.length) {
    return (
      <div className="rounded-lg border border-dashed bg-muted/40 px-6 py-12 text-center">
        <p className="font-heading text-2xl">No open fixtures</p>
        <p className="mx-auto mt-2 max-w-sm text-sm text-muted-foreground">
          Check back when upcoming or live matches are available.
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-6">
      <PageHeader>
        <PageHeaderHeading>Create challenge</PageHeaderHeading>
        <PageHeaderDescription>
          Challenge another player on a 1‑X‑2 outcome. They take one of the remaining sides at the same stake.
        </PageHeaderDescription>
      </PageHeader>

      <div className="space-y-1.5">
        <span className="text-sm font-medium">Choose a fixture</span>
        <FixturePicker
          matches={openMatches}
          selectedId={selectedMatch?.match_id ?? ''}
          onSelect={setSelectedId}
        />
      </div>

      {selectedMatch && (
        <div className="rounded-xl border border-border bg-card p-5 shadow-sahara sm:p-6">
          <ChallengeSlipBody
            match={selectedMatch}
            initialSide={initialSide}
          />
        </div>
      )}
    </div>
  )
}
