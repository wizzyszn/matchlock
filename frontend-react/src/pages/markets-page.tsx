import { MatchBrowser } from '@/components/wager/match-browser'
import { PageHeader, PageHeaderHeading, PageHeaderDescription } from '@/components/ui/page-header'

export function MarketsPage() {
  return (
    <section aria-labelledby="markets-heading">
      <PageHeader>
        <PageHeaderHeading id="markets-heading">Markets</PageHeaderHeading>
        <PageHeaderDescription>
          Tap a 1-X-2 odds cell or the challenge icon to open a PvP wager slip.
        </PageHeaderDescription>
      </PageHeader>
      <MatchBrowser />
    </section>
  )
}