import { ChallengeHistoryPanel } from '@/components/wager/challenge-history-panel'

export function HistoryPage() {
  return (
    <section aria-labelledby="history-heading">
      <div className="mb-8">
        <h2
          id="history-heading"
          className="font-heading text-3xl leading-tight sm:text-4xl"
        >
          Challenge History
        </h2>
        <p className="mt-2 max-w-prose text-sm text-muted-foreground">
          Review your settled and voided challenges.
        </p>
      </div>
      <ChallengeHistoryPanel />
    </section>
  )
}
