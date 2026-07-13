import { MyWagersPanel } from '@/components/wager/my-wagers-panel'
import { PageHeader, PageHeaderHeading, PageHeaderDescription } from '@/components/ui/page-header'

export function MyWagersPage() {
  return (
    <section aria-labelledby="my-wagers-heading">
      <PageHeader>
        <PageHeaderHeading id="my-wagers-heading">Your wagers</PageHeaderHeading>
        <PageHeaderDescription>
          Track open, matched, and settled positions for your connected wallet.
        </PageHeaderDescription>
      </PageHeader>
      <MyWagersPanel />
    </section>
  )
}