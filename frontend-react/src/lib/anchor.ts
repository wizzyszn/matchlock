import { AnchorProvider, Program, type Idl, type Wallet } from '@coral-xyz/anchor'
import { Connection, PublicKey, type Transaction } from '@solana/web3.js'

import type { AppConfig } from '@/lib/config'
import idl from '@/idl/blockchain.json'

export type AnchorCompatibleWallet = {
  publicKey: PublicKey
  signTransaction: (tx: Transaction) => Promise<Transaction>
  signAllTransactions: (txs: Transaction[]) => Promise<Transaction[]>
}

export function getProgramId(config: Pick<AppConfig, 'programId'>): PublicKey {
  return new PublicKey(config.programId)
}

export function getUsdcMint(config: Pick<AppConfig, 'usdcMint'>): PublicKey {
  return new PublicKey(config.usdcMint)
}

export function createAnchorProvider(
  connection: Connection,
  wallet: AnchorCompatibleWallet,
): AnchorProvider {
  return new AnchorProvider(connection, wallet as Wallet, {
    commitment: 'confirmed',
    preflightCommitment: 'confirmed',
  })
}

export function getProgram(
  connection: Connection,
  wallet: AnchorCompatibleWallet,
): Program<Idl> {
  const provider = createAnchorProvider(connection, wallet)
  return new Program(idl as Idl, provider)
}