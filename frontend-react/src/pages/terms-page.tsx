import { Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'

import { Button } from '@/components/ui/button'

export function TermsPage() {
  return (
    <div className="mx-auto min-h-svh max-w-3xl px-4 py-12 sm:py-16">
      <div className="mb-8">
        <Button variant="ghost" size="sm" render={<Link to="/login" className="gap-2" />}>
          <ArrowLeft className="size-4" />
          Back to login
        </Button>
      </div>

      <h1 className="font-heading mb-3 text-4xl tracking-tight">
        Terms & Conditions
      </h1>
      <p className="mb-10 text-sm text-muted-foreground">
        Last updated: July 2026
      </p>

      <div className="space-y-8 text-sm leading-relaxed text-muted-foreground">
        <section className="space-y-2">
          <h2 className="text-base font-semibold text-foreground">1. Acceptance of Terms</h2>
          <p>
            By accessing or using Matchlock ("the Platform"), you agree to be bound by these
            Terms & Conditions. If you do not agree, do not use the Platform.
          </p>
        </section>

        <section className="space-y-2">
          <h2 className="text-base font-semibold text-foreground">2. Eligibility</h2>
          <p>
            You must be at least 18 years old (or the legal age of majority in your
            jurisdiction) to use Matchlock. By using the Platform, you represent and warrant
            that you meet this requirement.
          </p>
        </section>

        <section className="space-y-2">
          <h2 className="text-base font-semibold text-foreground">3. No Real-Money Gambling</h2>
          <p>
            Matchlock is a decentralized prediction protocol built on Solana. Users wager
            using USDC or testnet tokens. The Platform is intended for entertainment and
            skill-based prediction purposes. Nothing on Matchlock constitutes an offer or
            solicitation to engage in real-money gambling.
          </p>
        </section>

        <section className="space-y-2">
          <h2 className="text-base font-semibold text-foreground">4. User Responsibility</h2>
          <p>
            You are solely responsible for your wallet, private keys, and any transactions
            you sign. Matchlock does not custody funds. All wagers are escrowed in
            program-controlled vaults on the Solana blockchain and settled transparently
            via TxLINE oracles.
          </p>
        </section>

        <section className="space-y-2">
          <h2 className="text-base font-semibold text-foreground">5. No Guarantees</h2>
          <p>
            The Platform is provided "as is" without warranties of any kind. We do not
            guarantee uninterrupted access, error-free software, or specific outcomes from
            settlement oracles.
          </p>
        </section>

        <section className="space-y-2">
          <h2 className="text-base font-semibold text-foreground">6. Limitation of Liability</h2>
          <p>
            To the fullest extent permitted by law, Matchlock and its contributors shall
            not be liable for any indirect, incidental, or consequential damages arising
            from your use of the Platform.
          </p>
        </section>

        <section className="space-y-2">
          <h2 className="text-base font-semibold text-foreground">7. Governing Law</h2>
          <p>
            These terms are governed by the laws applicable to decentralized protocols.
            Any disputes shall be resolved through decentralized arbitration where
            applicable.
          </p>
        </section>

        <section className="space-y-2">
          <h2 className="text-base font-semibold text-foreground">8. Changes to Terms</h2>
          <p>
            We may update these terms at any time. Continued use of the Platform after
            changes constitutes acceptance of the revised terms.
          </p>
        </section>

        <div className="border-t border-border pt-6 text-center">
          <p className="text-foreground font-semibold">
            Gamble responsibly. Must be 18+.
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            If you or someone you know has a gambling problem, seek help.
          </p>
        </div>
      </div>
    </div>
  )
}
