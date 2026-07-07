import { MyWagersPanel } from '@/components/wager/my-wagers-panel'

export function MyWagersPage() {
  return (
    <section aria-labelledby="my-wagers-heading">
      <div className="mb-8">
        <h2
          id="my-wagers-heading"
          className="font-heading text-3xl leading-tight sm:text-4xl"
        >
          Your wagers
        </h2>
        <p className="mt-2 max-w-prose text-sm text-muted-foreground">
          Track open, matched, and settled positions for your connected wallet.
        </p>
      </div>
      <MyWagersPanel />
    </section>
  )
}