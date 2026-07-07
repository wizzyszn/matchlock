import { BN } from '@coral-xyz/anchor'

export type TxlineProofNode = {
  hash: string | number[]
  isRightSibling: boolean
}

export type TxlineStatValidation = {
  summary: {
    fixtureId: number
    updateStats: {
      updateCount: number
      minTimestamp: number
      maxTimestamp: number
    }
    eventStatsSubTreeRoot: string | number[]
  }
  subTreeProof: TxlineProofNode[]
  mainTreeProof: TxlineProofNode[]
  statToProve: { key: number; value: number; period: number }
  eventStatRoot: string | number[]
  statProof: TxlineProofNode[]
}

function decodeHash32(value: string | number[]): number[] {
  if (Array.isArray(value)) {
    if (value.length !== 32) throw new Error('Expected 32-byte hash array')
    return [...value]
  }
  if (value.length === 64 && /^[0-9a-f]+$/i.test(value)) {
    const out: number[] = []
    for (let i = 0; i < 32; i++) {
      out.push(Number.parseInt(value.slice(i * 2, i * 2 + 2), 16))
    }
    return out
  }
  const binary = atob(value)
  const bytes = Uint8Array.from(binary, (c) => c.charCodeAt(0))
  if (bytes.length !== 32) throw new Error('Expected 32-byte base64 hash')
  return Array.from(bytes)
}

function mapProofNodes(nodes: TxlineProofNode[]) {
  return nodes.map((node) => ({
    hash: decodeHash32(node.hash),
    isRightSibling: node.isRightSibling,
  }))
}

export function validationFromTxlineApi(validation: TxlineStatValidation) {
  return {
    ts: new BN(validation.summary.updateStats.minTimestamp),
    fixtureSummary: {
      fixtureId: new BN(validation.summary.fixtureId),
      updateStats: {
        updateCount: validation.summary.updateStats.updateCount,
        minTimestamp: new BN(validation.summary.updateStats.minTimestamp),
        maxTimestamp: new BN(validation.summary.updateStats.maxTimestamp),
      },
      eventsSubTreeRoot: decodeHash32(validation.summary.eventStatsSubTreeRoot),
    },
    fixtureProof: mapProofNodes(validation.subTreeProof),
    mainTreeProof: mapProofNodes(validation.mainTreeProof),
    predicate: {
      threshold: 0,
      comparison: { greaterThan: {} },
    },
    statA: {
      statToProve: {
        key: validation.statToProve.key,
        value: validation.statToProve.value,
        period: validation.statToProve.period,
      },
      eventStatRoot: decodeHash32(validation.eventStatRoot),
      statProof: mapProofNodes(validation.statProof),
    },
    statB: null,
    op: null,
  }
}

export function decodeMerkleRoot(base64: string): number[] {
  const binary = atob(base64)
  const bytes = Uint8Array.from(binary, (c) => c.charCodeAt(0))
  if (bytes.length !== 32) throw new Error('Expected 32-byte merkle root')
  return Array.from(bytes)
}

export function toAnchorWinningSide(code: number) {
  switch (code) {
    case 0:
      return { home: {} }
    case 1:
      return { away: {} }
    case 2:
      return { draw: {} }
    default:
      throw new Error(`Unknown winning side code: ${code}`)
  }
}