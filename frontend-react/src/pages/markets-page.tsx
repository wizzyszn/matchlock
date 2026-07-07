import { MatchBrowser } from '@/components/wager/match-browser'

export function MarketsPage() {
  return (
    <section aria-labelledby="markets-heading">
      <div className="mb-8">
        <h2
          id="markets-heading"
          className="font-heading text-3xl leading-tight sm:text-4xl"
        >
          Markets
        </h2>
        <p className="mt-2 max-w-prose text-sm text-muted-foreground">
          Tap a 1-X-2 odds cell or the challenge icon to open a PvP wager slip.
        </p>
      </div>
      <MatchBrowser />
    </section>
  )
}