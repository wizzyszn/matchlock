import { useAnchorWallet, useConnection } from '@solana/wallet-adapter-react'
import { PublicKey } from '@solana/web3.js'
import { useCallback, useMemo } from 'react'

import { useConfig } from '@/hooks/use-api'
import type { SettlementProof, Side } from '@/lib/api'
import { getProgram, getUsdtMint } from '@/lib/anchor'
import {
  buildAcceptWagerTransaction,
  buildCancelWagerTransaction,
  buildClaimWagerTransaction,
  buildMakeWagerTransaction,
} from '@/lib/wager-tx'

export function useWagerTxBuilders() {
  const { connection } = useConnection()
  const wallet = useAnchorWallet()
  const config = useConfig()

  const program = useMemo(
    () => (wallet ? getProgram(connection, wallet) : null),
    [connection, wallet],
  )
  const stablecoinMint = useMemo(() => getUsdtMint(config), [config])

  const buildMake = useCallback(
    async (input: {
      matchId: string
      stake: bigint
      makerSide: Side
      participant1IsHome: boolean
      invitedTaker?: PublicKey
    }) => {
      if (!wallet?.publicKey || !program) return null
      return (await buildMakeWagerTransaction({
        program,
        connection,
        wallet,
        matchId: input.matchId,
        stake: input.stake,
        makerSide: input.makerSide,
        participant1IsHome: input.participant1IsHome,
        stablecoinMint,
        invitedTaker: input.invitedTaker,
      })).tx
    },
    [connection, program, stablecoinMint, wallet],
  )

  const buildAccept = useCallback(
    async (input: {
      wagerPubkey: string
      maker: string
      matchId: string
      takerSide: Side
    }) => {
      if (!wallet?.publicKey || !program) return null
      return buildAcceptWagerTransaction({
        program,
        wallet,
        wagerPubkey: new PublicKey(input.wagerPubkey),
        maker: new PublicKey(input.maker),
        matchId: input.matchId,
        takerSide: input.takerSide,
        stablecoinMint,
      })
    },
    [program, stablecoinMint, wallet],
  )

  const buildCancel = useCallback(
    async (input: { wagerPubkey: string }) => {
      if (!wallet?.publicKey || !program) return null
      return buildCancelWagerTransaction({
        program,
        wallet,
        wagerPubkey: new PublicKey(input.wagerPubkey),
        stablecoinMint,
      })
    },
    [program, stablecoinMint, wallet],
  )

  const buildClaim = useCallback(
    async (input: { wagerPubkey: string; proof: SettlementProof }) => {
      if (!wallet?.publicKey || !program) return null
      return buildClaimWagerTransaction({
        program,
        wallet,
        wagerPubkey: new PublicKey(input.wagerPubkey),
        proof: input.proof,
        stablecoinMint,
      })
    },
    [program, stablecoinMint, wallet],
  )

  return {
    wallet,
    buildMake,
    buildAccept,
    buildCancel,
    buildClaim,
  }
}
