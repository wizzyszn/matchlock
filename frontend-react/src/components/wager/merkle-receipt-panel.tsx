import { CheckCircle2 } from 'lucide-react'
import { MerkleProofMap } from './merkle-proof-map'

export type MerkleReceiptPanelProps = {
  fixtureId: number
}

export function MerkleReceiptPanel({ fixtureId }: MerkleReceiptPanelProps) {
  return (
    <div className="flex w-full flex-col h-full bg-transparent">
      <div className="mb-6 space-y-1.5 px-4 sm:px-0">
        <h2 className="flex items-center gap-2 text-xl font-semibold tracking-tight text-foreground">
          Verifiable Resolution
        </h2>
        <p className="text-sm text-muted-foreground max-w-xl">
          Cryptographic guarantee that your match outcomes originate directly from the TxODDS feed and were confirmed in an on-chain Merkle Root.
        </p>
      </div>

      <div className="flex flex-col flex-1 relative min-h-[400px]">
        <MerkleProofMap fixtureId={fixtureId} />
        
        <div className="mt-8 flex rounded-md bg-muted/20 p-4 text-xs text-muted-foreground border border-border/40 max-w-2xl mx-auto w-full">
          <CheckCircle2 className="mr-2 mt-0.5 size-4 shrink-0 text-status-settled" />
          <p>
            You can take these hashes and use standard on-chain validation to mathematically confirm
            TxODDS data integrity without needing to trust an intermediate relayer.
          </p>
        </div>
      </div>
    </div>
  )
}

