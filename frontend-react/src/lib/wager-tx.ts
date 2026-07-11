import { BN, type Idl, type Program } from '@coral-xyz/anchor'
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  TOKEN_PROGRAM_ID,
  createAssociatedTokenAccountIdempotentInstruction,
  getAssociatedTokenAddressSync,
} from '@solana/spl-token'
import {
  ComputeBudgetProgram,
  Connection,
  PublicKey,
  SystemProgram,
  Transaction,
  type TransactionSignature,
} from '@solana/web3.js'

import type { SettlementProof, Side } from '@/lib/api'
import {
  decodeMerkleRoot,
  toAnchorWinningSide,
  validationFromTxlineApi,
  type TxlineStatValidation,
} from '@/lib/settle-validation'

export type AnchorWallet = {
  publicKey: PublicKey
  signTransaction: (tx: Transaction) => Promise<Transaction>
}

function toAnchorSide(side: Side) {
  switch (side) {
    case 'home':
      return { home: {} }
    case 'away':
      return { away: {} }
    case 'draw':
      return { draw: {} }
    case 'unset':
      throw new Error('Cannot use unset side as an instruction argument')
  }
}

function findConfigPda(programId: PublicKey): PublicKey {
  const [pda] = PublicKey.findProgramAddressSync(
    [Buffer.from('config')],
    programId,
  )
  return pda
}

function findWagerPda(
  programId: PublicKey,
  maker: PublicKey,
  matchId: string,
  nonce: bigint,
): PublicKey {
  const matchBytes = Buffer.from(matchId, 'utf8')
  const nonceBuf = Buffer.alloc(8)
  nonceBuf.writeBigUInt64LE(nonce)
  const [pda] = PublicKey.findProgramAddressSync(
    [Buffer.from('wager'), maker.toBuffer(), matchBytes, nonceBuf],
    programId,
  )
  return pda
}

function findVaultPda(programId: PublicKey, wager: PublicKey): PublicKey {
  const [pda] = PublicKey.findProgramAddressSync(
    [Buffer.from('vault'), wager.toBuffer()],
    programId,
  )
  return pda
}

export async function estimateTransactionFee(
  connection: Connection,
  wallet: AnchorWallet,
  tx: Transaction,
): Promise<number> {
  const { blockhash, lastValidBlockHeight } =
    await connection.getLatestBlockhash('confirmed')

  const prepared = new Transaction({
    feePayer: wallet.publicKey,
    blockhash,
    lastValidBlockHeight,
  })
  prepared.add(...tx.instructions)

  const fee = await connection.getFeeForMessage(prepared.compileMessage())
  if (fee.value === null) {
    throw new Error('Network fee estimate unavailable')
  }

  // Cross-check with simulation when possible (same tx shape users will sign).
  try {
    const simulation = await connection.simulateTransaction(prepared)
    if (simulation.value.err) {
      return fee.value
    }
    const simFee = (simulation.value as { fee?: number }).fee
    if (typeof simFee === 'number' && simFee > 0) {
      return simFee
    }
  } catch {
    // Fall back to getFeeForMessage result.
  }

  return fee.value
}

async function prepareTransaction(
  connection: Connection,
  wallet: AnchorWallet,
  instructions: Parameters<Transaction['add']>[0][],
): Promise<Transaction> {
  const tx = new Transaction().add(...instructions)
  const { blockhash, lastValidBlockHeight } =
    await connection.getLatestBlockhash('confirmed')
  tx.recentBlockhash = blockhash
  tx.feePayer = wallet.publicKey
  tx.lastValidBlockHeight = lastValidBlockHeight
  return tx
}

export async function simulateTransaction(
  connection: Connection,
  wallet: AnchorWallet,
  tx: Transaction,
): Promise<void> {
  const prepared = await prepareTransaction(
    connection,
    wallet,
    tx.instructions,
  )
  const simulation = await connection.simulateTransaction(prepared)

  if (simulation.value.err) {
    const logs = simulation.value.logs?.join('\n') ?? ''
    throw new Error(
      `Simulation failed: ${JSON.stringify(simulation.value.err)}${logs ? `\n${logs}` : ''}`,
    )
  }
}

export async function sendTransaction(
  connection: Connection,
  wallet: AnchorWallet,
  tx: Transaction,
): Promise<TransactionSignature> {
  const prepared = await prepareTransaction(
    connection,
    wallet,
    tx.instructions,
  )
  const signed = await wallet.signTransaction(prepared)
  const signature = await connection.sendRawTransaction(signed.serialize(), {
    skipPreflight: false,
    preflightCommitment: 'confirmed',
  })
  await connection.confirmTransaction(signature, 'confirmed')
  return signature
}

export type MakeWagerParams = {
  program: Program<Idl>
  connection: Connection
  wallet: AnchorWallet
  matchId: string
  stake: bigint
  makerSide: Side
  participant1IsHome: boolean
  stablecoinMint: PublicKey
  invitedTaker?: PublicKey
  nonce?: bigint
}

export type BuildMakeWagerTransactionResult = {
  tx: Transaction
  wagerPubkey: PublicKey
}

export async function buildMakeWagerTransaction({
  program,
  wallet,
  matchId,
  stake,
  makerSide,
  participant1IsHome,
  stablecoinMint,
  invitedTaker,
  nonce = BigInt(Date.now()),
}: MakeWagerParams): Promise<BuildMakeWagerTransactionResult> {
  const maker = wallet.publicKey
  const matchBytes = Buffer.from(matchId, 'utf8')
  const config = findConfigPda(program.programId)
  const wager = findWagerPda(program.programId, maker, matchId, nonce)
  const vault = findVaultPda(program.programId, wager)
  const makerStablecoin = getAssociatedTokenAddressSync(stablecoinMint, maker)

  const createAtaIx = createAssociatedTokenAccountIdempotentInstruction(
    maker,
    makerStablecoin,
    maker,
    stablecoinMint,
  )

  const makeIx = await program.methods
    .makeWager(
      matchBytes,
      new BN(stake.toString()),
      toAnchorSide(makerSide),
      invitedTaker ?? PublicKey.default,
      participant1IsHome,
      new BN(nonce.toString()),
    )
    .accounts({
      maker,
      config,
      wager,
      vault,
      makerStablecoin,
      stablecoinMint,
      tokenProgram: TOKEN_PROGRAM_ID,
      associatedTokenProgram: ASSOCIATED_TOKEN_PROGRAM_ID,
      systemProgram: SystemProgram.programId,
    })
    .instruction()

  return { tx: new Transaction().add(createAtaIx, makeIx), wagerPubkey: wager }
}

export type AcceptWagerParams = {
  program: Program<Idl>
  wallet: AnchorWallet
  wagerPubkey: PublicKey
  maker: PublicKey
  takerSide: Side
  stablecoinMint: PublicKey
}

export async function buildAcceptWagerTransaction({
  program,
  wallet,
  wagerPubkey,
  maker,
  takerSide,
  stablecoinMint,
}: AcceptWagerParams): Promise<Transaction> {
  const taker = wallet.publicKey
  const config = findConfigPda(program.programId)
  const vault = findVaultPda(program.programId, wagerPubkey)
  const takerStablecoin = getAssociatedTokenAddressSync(stablecoinMint, taker)

  const createAtaIx = createAssociatedTokenAccountIdempotentInstruction(
    taker,
    takerStablecoin,
    taker,
    stablecoinMint,
  )

  const acceptIx = await program.methods
    .acceptWager(toAnchorSide(takerSide))
    .accounts({
      taker,
      config,
      wager: wagerPubkey,
      maker,
      takerStablecoin,
      vault,
      stablecoinMint,
      tokenProgram: TOKEN_PROGRAM_ID,
      associatedTokenProgram: ASSOCIATED_TOKEN_PROGRAM_ID,
    })
    .instruction()

  return new Transaction().add(createAtaIx, acceptIx)
}

export type CancelWagerParams = {
  program: Program<Idl>
  wallet: AnchorWallet
  wagerPubkey: PublicKey
  stablecoinMint: PublicKey
}

export async function buildCancelWagerTransaction({
  program,
  wallet,
  wagerPubkey,
  stablecoinMint,
}: CancelWagerParams): Promise<Transaction> {
  const maker = wallet.publicKey
  const config = findConfigPda(program.programId)
  const vault = findVaultPda(program.programId, wagerPubkey)
  const makerStablecoin = getAssociatedTokenAddressSync(stablecoinMint, maker)

  const cancelIx = await program.methods
    .cancelWager()
    .accounts({
      maker,
      config,
      wager: wagerPubkey,
      vault,
      makerStablecoin,
      stablecoinMint,
      tokenProgram: TOKEN_PROGRAM_ID,
      associatedTokenProgram: ASSOCIATED_TOKEN_PROGRAM_ID,
    })
    .instruction()

  return new Transaction().add(cancelIx)
}

export type ClaimWagerParams = {
  program: Program<Idl>
  wallet: AnchorWallet
  wagerPubkey: PublicKey
  proof: SettlementProof
  stablecoinMint: PublicKey
}

export async function buildClaimWagerTransaction({
  program,
  wallet,
  wagerPubkey,
  proof,
  stablecoinMint,
}: ClaimWagerParams): Promise<Transaction> {
  const settler = wallet.publicKey
  const config = findConfigPda(program.programId)
  const vault = findVaultPda(program.programId, wagerPubkey)
  const winnerStablecoin = getAssociatedTokenAddressSync(stablecoinMint, settler)
  const txlineProgram = new PublicKey(proof.txline_program_id)
  const dailyScores = new PublicKey(proof.daily_scores_pda)

  const validation = validationFromTxlineApi(
    proof.validation as TxlineStatValidation,
  )
  const merkleRoot = decodeMerkleRoot(proof.merkle_root)
  const winningSide = toAnchorWinningSide(proof.winning_side_code)

  const createAtaIx = createAssociatedTokenAccountIdempotentInstruction(
    settler,
    winnerStablecoin,
    settler,
    stablecoinMint,
  )

  const settleIx = await program.methods
    .settleWager(validation, winningSide, merkleRoot)
    .accounts({
      settler,
      config,
      wager: wagerPubkey,
      vault,
      winner: settler,
      winnerStablecoin,
      stablecoinMint,
      dailyScoresMerkleRoots: dailyScores,
      txlineProgram,
      tokenProgram: TOKEN_PROGRAM_ID,
      associatedTokenProgram: ASSOCIATED_TOKEN_PROGRAM_ID,
    })
    .instruction()

  const computeIx = ComputeBudgetProgram.setComputeUnitLimit({
    units: 1_400_000,
  })

  return new Transaction().add(computeIx, createAtaIx, settleIx)
}
