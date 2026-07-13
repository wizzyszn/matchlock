import { ChallengeHistoryPanel } from '@/components/wager/challenge-history-panel'
import { PageHeader, PageHeaderHeading, PageHeaderDescription } from '@/components/ui/page-header'

export function HistoryPage() {
  return (
    <section aria-labelledby="history-heading">
      <PageHeader>
        <PageHeaderHeading id="history-heading">Challenge History</PageHeaderHeading>
        <PageHeaderDescription>
          Review open, matched, settled, and voided bets with status, result, and date filters.
        </PageHeaderDescription>
      </PageHeader>
      <ChallengeHistoryPanel />
    </section>
  )
}
