import { useAnchorWallet, useConnection } from '@solana/wallet-adapter-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { PublicKey } from '@solana/web3.js'

import { useConfig } from '@/hooks/use-api'
import type { Side } from '@/lib/api'
import { mapTransactionError } from '@/lib/errors'
import { getProgram, getUsdcMint } from '@/lib/anchor'

import { useApi } from '@/hooks/use-api'
import { useWalletLinkStatus } from '@/hooks/use-wallet-link-status'
import {
  buildAcceptWagerTransaction,
  buildCancelWagerTransaction,
  buildClaimWagerTransaction,
  buildMakeWagerTransaction,
  sendTransaction,
  simulateTransaction,
} from '@/lib/wager-tx'

export type TxAction = 'make' | 'accept' | 'cancel' | 'claim'

export function useWagerMutations() {
  const { connection } = useConnection()
  const wallet = useAnchorWallet()
  const { canTransact, connected, needsLink, conflict } = useWalletLinkStatus()
  const config = useConfig()
  const api = useApi()
  const queryClient = useQueryClient()

  const invalidateWagers = async () => {
    await queryClient.invalidateQueries({ queryKey: ['wagers'] })
    await queryClient.invalidateQueries({ queryKey: ['tokenBalance'] })
  }

  const makeWager = useMutation({
    mutationFn: async (input: {
      matchId: string
      stake: bigint
      makerSide: Side
      invitedTaker?: PublicKey
    }) => {
      if (!wallet?.publicKey) {
        throw new Error('Connect your wallet on Profile first.')
      }
      if (conflict) {
        throw new Error(
          'This wallet is linked to another Matchlock account. Switch wallet on Profile.',
        )
      }
      if (needsLink) {
        throw new Error('Link your connected wallet to your account on Profile.')
      }
      const program = getProgram(connection, wallet)
      const stablecoinMint = getUsdcMint(config)
      const matchBytes = Buffer.from(input.matchId, 'utf8')
      const [wagerPubkey] = PublicKey.findProgramAddressSync(
        [Buffer.from('wager'), wallet.publicKey.toBuffer(), matchBytes],
        program.programId,
      )

      const tx = await buildMakeWagerTransaction({
        program,
        connection,
        wallet,
        matchId: input.matchId,
        stake: input.stake,
        makerSide: input.makerSide,
        stablecoinMint,
        invitedTaker: input.invitedTaker,
      })

      await simulateTransaction(connection, wallet, tx)
      const signature = await sendTransaction(connection, wallet, tx)
      return { signature, wagerPubkey: wagerPubkey.toBase58() }
    },
    onSuccess: invalidateWagers,
  })

  const acceptWager = useMutation({
    mutationFn: async (input: {
      wagerPubkey: string
      maker: string
      takerSide: Side
    }) => {
      if (!wallet?.publicKey) {
        throw new Error('Connect your wallet on Profile first.')
      }
      if (conflict) {
        throw new Error(
          'This wallet is linked to another Matchlock account. Switch wallet on Profile.',
        )
      }
      if (needsLink) {
        throw new Error('Link your connected wallet to your account on Profile.')
      }
      const program = getProgram(connection, wallet)
      const stablecoinMint = getUsdcMint(config)

      const tx = await buildAcceptWagerTransaction({
        program,
        wallet,
        wagerPubkey: new PublicKey(input.wagerPubkey),
        maker: new PublicKey(input.maker),
        takerSide: input.takerSide,
        stablecoinMint,
      })

      await simulateTransaction(connection, wallet, tx)
      return sendTransaction(connection, wallet, tx)
    },
    onSuccess: invalidateWagers,
  })

  const cancelWager = useMutation({
    mutationFn: async (input: { wagerPubkey: string }) => {
      if (!wallet?.publicKey) {
        throw new Error('Connect your wallet on Profile first.')
      }
      if (conflict) {
        throw new Error(
          'This wallet is linked to another Matchlock account. Switch wallet on Profile.',
        )
      }
      if (needsLink) {
        throw new Error('Link your connected wallet to your account on Profile.')
      }
      const program = getProgram(connection, wallet)
      const stablecoinMint = getUsdcMint(config)

      const tx = await buildCancelWagerTransaction({
        program,
        wallet,
        wagerPubkey: new PublicKey(input.wagerPubkey),
        stablecoinMint,
      })

      await simulateTransaction(connection, wallet, tx)
      return sendTransaction(connection, wallet, tx)
    },
    onSuccess: invalidateWagers,
  })

  const claimWager = useMutation({
    mutationFn: async (input: { wagerPubkey: string }) => {
      if (!wallet?.publicKey) {
        throw new Error('Connect your wallet on Profile first.')
      }
      if (conflict) {
        throw new Error(
          'This wallet is linked to another Matchlock account. Switch wallet on Profile.',
        )
      }
      if (needsLink) {
        throw new Error('Link your connected wallet to your account on Profile.')
      }
      const proof = await api.getWagerSettlementProof(input.wagerPubkey)
      const program = getProgram(connection, wallet)
      const stablecoinMint = getUsdcMint(config)

      const tx = await buildClaimWagerTransaction({
        program,
        wallet,
        wagerPubkey: new PublicKey(input.wagerPubkey),
        proof,
        stablecoinMint,
      })

      await simulateTransaction(connection, wallet, tx)
      return sendTransaction(connection, wallet, tx)
    },
    onSuccess: invalidateWagers,
  })

  return {
    makeWager,
    acceptWager,
    cancelWager,
    claimWager,
    mapError: mapTransactionError,
    isWalletReady: canTransact,
    walletConnected: connected,
    walletNeedsLink: needsLink,
    walletConflict: conflict,
  }
}