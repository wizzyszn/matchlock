import { useWallet } from '@solana/wallet-adapter-react'
import { PublicKey } from '@solana/web3.js'
import { Loader2, Swords } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { useMediaQuery } from '@/hooks/use-media-query'


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
import { baseUnitsToUsdc, formatUsdc, usdcToBaseUnits } from '@/lib/format'
import {
  classifyMatch,
  formatKickoffClock,
  formatKickoffDate,
  matchLabels,
} from '@/lib/match-display'
import { sideLabel } from '@/lib/wager-sides'

type ChallengeSlipProps = {
  match: Match
  initialSide?: Side
  open?: boolean
  onOpenChange?: (open: boolean) => void
  variant?: 'dialog' | 'page'
}

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
  const { makeWager, mapError, isWalletReady, walletNeedsLink } =
    useWagerMutations()
  const walletStatus = useWalletLinkStatus()
  const { buildMake, wallet } = useWagerTxBuilders()
  const stablecoin = useStablecoinLabel()

  const [side, setSide] = useState<Side>(initialSide)
  const [challengeMode, setChallengeMode] = useState<'open' | 'friend'>('open')
  const [friendEmail, setFriendEmail] = useState('')
  const [friendLookup, setFriendLookup] = useState<UserLookup | null>(null)
  const [stakeInput, setStakeInput] = useState('')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [confirmDetails, setConfirmDetails] = useState<ConfirmTxDetails | null>(
    null,
  )
  const [txError, setTxError] = useState<string | null>(null)
  const [signature, setSignature] = useState<string | null>(null)

  useEffect(() => {
    setSide(initialSide)
  }, [initialSide, match.match_id])

  const labels = matchLabels(match)
  const stakeUsdc = Number.parseFloat(stakeInput)
  const stakeValid = Number.isFinite(stakeUsdc) && stakeUsdc > 0
  const balanceUsdc = baseUnitsToUsdc(balance)
  const insufficientBalance = stakeValid && stakeUsdc > balanceUsdc
  const canChallenge = classifyMatch(match) === 'upcoming' || classifyMatch(match) === 'live'
  const friendEmailNormalized = friendEmail.trim().toLowerCase()
  const isSelfChallenge =
    Boolean(session) &&
    friendEmailNormalized.length > 3 &&
    friendEmailNormalized === session?.email.toLowerCase()

  const friendReady =
    challengeMode === 'open' ||
    (Boolean(session) &&
      friendLookup !== null &&
      !isSelfChallenge)

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

  const handleOpenConfirm = () => {
    if (challengeMode === 'friend' && !friendReady) return
    if (!stakeValid || insufficientBalance || !canChallenge) return

    setConfirmDetails({
      action: 'make',
      matchLabel: labels.league,
      sideLabel: sideLabel(side, match),
      stakeUsdc,
      payoutUsdc: stakeUsdc * 2,
    })
    setTxError(null)
    setSignature(null)
    setDialogOpen(true)
  }

  const estimateFee = useCallback(async () => {
    if (!confirmDetails || confirmDetails.action !== 'make' || !stakeValid) {
      return null
    }
    const invitedTaker =
      challengeMode === 'friend' && friendLookup?.primary_wallet
        ? new PublicKey(friendLookup.primary_wallet)
        : undefined
    return buildMake({
      matchId: match.match_id,
      stake: usdcToBaseUnits(stakeUsdc),
      makerSide: side,
      invitedTaker,
    })
  }, [
    buildMake,
    challengeMode,
    confirmDetails,
    friendLookup?.primary_wallet,
    match.match_id,
    side,
    stakeUsdc,
    stakeValid,
  ])

  const feeEstimate = useTxFeeEstimate({
    enabled: dialogOpen && confirmDetails?.action === 'make',
    estimateKey: `make-${match.match_id}-${side}-${stakeUsdc}`,
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
        stake: usdcToBaseUnits(stakeUsdc),
        makerSide: side,
        invitedTaker,
      })

      if (challengeMode === 'friend' && session && friendEmail.trim()) {
        try {
          await createInvite.mutateAsync({
            recipient_email: friendEmail.trim(),
            wager_pubkey: wagerPubkey,
            match_id: match.match_id,
            maker_side: side,
            stake: Number(usdcToBaseUnits(stakeUsdc)),
            home_team: labels.homeTeam,
            away_team: labels.awayTeam,
          })
        } catch {
          toast.warning('Wager created, but the invite could not be sent.')
        }
      }

      setSignature(signature)
      setDialogOpen(false)
      setStakeInput('')
      setFriendEmail('')
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
      <div className="space-y-5">
        <DuelFrame
          home={labels.homeTeam}
          away={labels.awayTeam}
          league={labels.league}
          isLive={labels.isLive}
          size="compact"
          layout="inline"
        />

        <div className="flex items-center justify-center gap-3 text-xs text-muted-foreground">
          <span>{formatKickoffDate(match)}</span>
          <span aria-hidden>·</span>
          <span className="tabular-nums font-medium text-foreground">
            {formatKickoffClock(match)}
          </span>
        </div>

        {!canChallenge ? (
          <p className="rounded-md border border-dashed bg-muted/40 px-3 py-2 text-sm text-muted-foreground">
            This fixture is finished — pick an upcoming or live match to challenge.
          </p>
        ) : (
          <>
            <OutcomePicker
              match={match}
              sides={['home', 'draw', 'away']}
              selected={side}
              onSelect={setSide}
              hint="Reference odds from TxLINE — your opponent takes one of the other outcomes."
            />

            <div className="space-y-3 rounded-lg border bg-card p-3">
              <p className="text-sm font-medium">Who can accept?</p>
              <div className="flex flex-wrap gap-2">
                <Button
                  type="button"
                  size="sm"
                  variant={challengeMode === 'open' ? 'default' : 'outline'}
                  onClick={() => setChallengeMode('open')}
                >
                  Anyone
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant={challengeMode === 'friend' ? 'default' : 'outline'}
                  onClick={() => setChallengeMode('friend')}
                >
                  A friend
                </Button>
              </div>
              {challengeMode === 'friend' ? (
                <div className="space-y-2">
                  {!session ? (
                    <p className="text-sm text-muted-foreground">
                      <Link to="/login" className="text-primary hover:underline">
                        Sign in
                      </Link>{' '}
                      to challenge someone by email.
                    </p>
                  ) : (
                    <>
                      <Input
                        type="email"
                        placeholder="friend@example.com"
                        value={friendEmail}
                        onChange={(e) => {
                          setFriendEmail(e.target.value)
                          setFriendLookup(null)
                        }}
                      />
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        disabled={!friendEmail.trim() || lookupUser.isPending}
                        onClick={() => void handleLookupFriend()}
                      >
                        {lookupUser.isPending ? 'Looking up…' : 'Find friend'}
                      </Button>
                      {friendLookup ? (
                        <div className="rounded-md border bg-muted/20 px-3 py-2 text-sm">
                          {isSelfChallenge ? (
                            walletStatus.primaryWallet ? (
                              <p className="text-muted-foreground">
                                Direct self-challenges are not supported. Use{' '}
                                <strong>Anyone</strong> for an open wager.
                              </p>
                            ) : (
                              <p>
                                That&apos;s your email —{' '}
                                <Link
                                  to="/profile"
                                  className="font-medium text-primary hover:underline"
                                >
                                  link your wallet on Profile
                                </Link>{' '}
                                before using friend challenges.
                              </p>
                            )
                          ) : friendLookup.primary_wallet ? (
                            <p className="text-muted-foreground">
                              Friend found — only{' '}
                              <span className="font-mono">
                                {friendLookup.primary_wallet.slice(0, 4)}…
                                {friendLookup.primary_wallet.slice(-4)}
                              </span>{' '}
                              can accept.
                            </p>
                          ) : (
                            <p className="text-muted-foreground">
                              They&apos;ll receive a challenge invite by email
                              once you create this wager.
                            </p>
                          )}
                        </div>
                      ) : null}
                    </>
                  )}
                </div>
              ) : null}
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between gap-2">
                <label htmlFor="challenge-stake" className="text-sm font-medium">
                  Stake ({stablecoin})
                </label>
                {isWalletReady ? (
                  <button
                    type="button"
                    className="text-xs font-medium text-primary underline-offset-4 hover:underline"
                    onClick={() => setStakeInput(balanceUsdc.toString())}
                  >
                    Max {formatUsdc(balanceUsdc)}
                  </button>
                ) : null}
              </div>
              <Input
                id="challenge-stake"
                type="number"
                min="0"
                step="0.000001"
                inputMode="decimal"
                placeholder="0.00"
                value={stakeInput}
                onChange={(e) => setStakeInput(e.target.value)}
              />
              {insufficientBalance ? (
                <p className="text-sm text-destructive">
                  Insufficient {stablecoin} balance.
                </p>
              ) : null}
            </div>

            <div className="rounded-lg border border-dashed bg-muted/30 px-4 py-3 text-sm">
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">You back</span>
                <span className="font-medium">{sideLabel(side, match)}</span>
              </div>
              <div className="mt-2 flex justify-between gap-4">
                <span className="text-muted-foreground">If you win</span>
                <span className="tabular-nums font-semibold text-primary">
                  {stakeValid ? `${formatUsdc(stakeUsdc * 2)} ${stablecoin}` : '—'}
                </span>
              </div>
            </div>
          </>
        )}
      </div>

      <div className="mt-4 flex flex-col gap-2 sm:flex-row">
        <Button
          className="min-h-11 flex-1"
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
          onClick={handleOpenConfirm}
        >
          {makeWager.isPending ? (
            <>
              <Loader2 className="size-4 animate-spin" />
              Creating…
            </>
          ) : (
            <>
              <Swords className="size-4" aria-hidden />
              Create challenge
            </>
          )}
        </Button>
      </div>



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
          className="w-full sm:max-w-[420px] overflow-y-auto px-4 sm:px-6"
        >
          <SheetHeader className="mb-6 text-left">
            <SheetTitle>Create challenge</SheetTitle>
            <SheetDescription className="text-xs">
              Pick your outcome, set your stake, and post an open PvP wager.
            </SheetDescription>
          </SheetHeader>
          <div className="pb-6 w-full max-w-full">
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
          className="max-h-[90dvh] overflow-y-auto rounded-t-2xl border-t px-4 pb-8 pt-6"
        >
          <div className="mb-4 flex items-center justify-center">
            <div className="h-1.5 w-12 rounded-full bg-border" />
          </div>
          <SheetHeader className="mb-4 text-left px-0">
            <SheetTitle>Create challenge</SheetTitle>
            <SheetDescription className="text-xs">
              Pick your outcome, set your stake, and post an open PvP wager.
            </SheetDescription>
          </SheetHeader>
          <div className="w-full max-w-full">
            <ChallengeSlipBody
              match={match}
              initialSide={initialSide}
              onSuccess={() => onOpenChange(false)}
            />
          </div>
        </SheetContent>
      )}
    </Sheet>
  )
}

export function ChallengeSlipPage({
  matchId,
  initialSide,
}: {
  matchId?: string
  initialSide?: Side
}) {
  const { data: matches, isLoading } = useMatchesQuery()

  const openMatches = useMemo(
    () =>
      (matches ?? []).filter(
        (match) => classifyMatch(match) !== 'finished',
      ),
    [matches],
  )

  const [selectedId, setSelectedId] = useState(matchId ?? '')
  useEffect(() => {
    if (matchId) setSelectedId(matchId)
  }, [matchId])

  const selectedMatch =
    openMatches.find((match) => match.match_id === selectedId) ??
    openMatches[0]

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
          Check back when upcoming or live matches are available on the schedule.
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg space-y-5">
      <div>
        <h2 className="font-heading text-3xl sm:text-4xl">Create challenge</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          Challenge another player on a 1-X-2 outcome. They take one of the
          remaining sides at the same stake.
        </p>
      </div>

      <div className="space-y-2">
        <span className="text-sm font-medium">Choose a fixture</span>
        <FixturePicker
          matches={openMatches}
          selectedId={selectedMatch?.match_id ?? ''}
          onSelect={setSelectedId}
        />
      </div>

      {selectedMatch ? (
        <div className="rounded-xl border border-border bg-card p-5 shadow-sahara sm:p-7">
          <ChallengeSlipBody
            match={selectedMatch}
            initialSide={initialSide}
          />
        </div>
      ) : null}
    </div>
  )
}
