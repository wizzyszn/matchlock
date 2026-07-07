import { useAnchorWallet, useConnection } from '@solana/wallet-adapter-react'
import { PublicKey } from '@solana/web3.js'
import { useCallback, useMemo } from 'react'

import { useConfig } from '@/hooks/use-api'
import type { SettlementProof, Side } from '@/lib/api'
import { getProgram, getUsdcMint } from '@/lib/anchor'
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
  const stablecoinMint = useMemo(() => getUsdcMint(config), [config])

  const buildMake = useCallback(
    async (input: {
      matchId: string
      stake: bigint
      makerSide: Side
      invitedTaker?: PublicKey
    }) => {
      if (!wallet?.publicKey || !program) return null
      return buildMakeWagerTransaction({
        program,
        connection,
        wallet,
        matchId: input.matchId,
        stake: input.stake,
        makerSide: input.makerSide,
        stablecoinMint,
        invitedTaker: input.invitedTaker,
      })
    },
    [connection, program, stablecoinMint, wallet],
  )

  const buildAccept = useCallback(
    async (input: {
      wagerPubkey: string
      maker: string
      takerSide: Side
    }) => {
      if (!wallet?.publicKey || !program) return null
      return buildAcceptWagerTransaction({
        program,
        wallet,
        wagerPubkey: new PublicKey(input.wagerPubkey),
        maker: new PublicKey(input.maker),
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