import { ArrowRight, Check, X } from 'lucide-react'
import { Link } from 'react-router-dom'
import logoUrl from '@/assets/g17.svg'

const NAV_LINKS = [
  { href: '#workflow', label: 'How it Works' },
  { href: '#advantage', label: 'The Advantage' },
  { href: '#roadmap', label: 'Protocol Roadmap' },
]

const WORKFLOW_STEPS = [
  { step: '01', title: 'Pick a match', description: 'Browse live World Cup fixtures with real-time status and market availability.' },
  { step: '02', title: 'Challenge or accept', description: 'Create a wager on your side, or take the opposite side of an open challenge.' },
  { step: '03', title: 'Stakes lock on-chain', description: 'USDC moves securely to a program-controlled vault PDA.' },
  { step: '04', title: 'Settle automatically', description: 'When the match ends, TxLINE proves the result for 2x payout.' },
]

const TRADITIONAL_DRAWBACKS = [
  '5% to 10% house edge',
  'Opaque settlement systems',
  'Forced custody of funds',
]

const MATCHLOCK_BENEFITS = [
  '0% house edge (True PvP)',
  'Automated cryptographic settlement',
  'Fully non-custodial',
]

export function HomePage() {
  const scrollTo = (id: string) => {
    const el = document.getElementById(id)
    if (el) el.scrollIntoView({ behavior: 'smooth' })
  }

  return (
    <div className="min-h-screen bg-[#282828] text-white">
      {/* NAV */}
      <nav className="fixed top-8 left-1/2 -translate-x-1/2 bg-[#2c2c2c] border border-[#373737] rounded-[100px] py-4 px-10 flex items-center justify-between w-[1282px] max-w-[95vw] z-50">
        <div className="flex items-center gap-3">
          <img src={logoUrl} alt="" className="h-7 w-auto object-contain" />
          <span className="font-heading text-3xl font-bold">Matchlock</span>
        </div>
        <div className="hidden lg:flex gap-8 text-sm font-medium">
          <Link to="/" className="text-[#e64040] font-semibold">Home</Link>
          {NAV_LINKS.map(({ href, label }) => (
            <a
              key={href}
              href={href}
              onClick={(e) => { e.preventDefault(); scrollTo(href.slice(1)) }}
              className="hover:text-white transition-colors"
            >
              {label}
            </a>
          ))}
        </div>
        <div className="flex items-center gap-4">
          <Link
            to="/login"
            className="px-8 py-3 rounded-full border border-white/30 hover:bg-white/5 transition text-sm font-medium"
          >
            Login
          </Link>
          <Link
            to="/login"
            className="px-8 py-3 bg-gradient-to-r from-[#e64040] to-[#e66140] rounded-full font-medium text-sm"
          >
            Enter Matchlock
          </Link>
        </div>
      </nav>

      {/* HERO */}
      <section className="relative min-h-screen flex items-center justify-center overflow-hidden pt-20">
        <div className="absolute inset-0 bg-gradient-to-b from-[#e64040]/5 to-transparent opacity-20" />

        <div className="relative z-10 text-center max-w-4xl px-6">
          <h1 className="text-7xl md:text-8xl leading-none font-medium tracking-tighter mb-6">
            Challenge another Player.<br />
            <span className="font-heading text-[110px] md:text-[130px] leading-none bg-clip-text text-transparent bg-gradient-to-r from-white via-white to-[#999]">
              Settle on-chain.
            </span>
          </h1>
          <p className="text-xl text-white/80 max-w-lg mx-auto mb-10">
            Peer-to-peer sports wagers with trustless settlement powered by TxLINE oracle data. No house edge. Your funds, your control.
          </p>
          <div className="flex justify-center gap-4">
            <Link
              to="/login"
              className="px-10 py-4 bg-gradient-to-r from-[#e64040] to-[#e66140] rounded-full text-lg font-medium inline-flex items-center gap-2"
            >
              Enter Matchlock <ArrowRight className="w-5 h-5" />
            </Link>
            <button
              onClick={() => scrollTo('workflow')}
              className="px-10 py-4 border border-white/30 rounded-full text-lg font-medium hover:bg-white/5"
            >
              See how it works
            </button>
          </div>
        </div>

        {/* Match Example */}
        <div className="absolute bottom-12 left-1/2 -translate-x-1/2 bg-[#2c2c2c] border border-[#373737] rounded-[58px] w-[989px] max-w-[95vw] h-[235px] flex items-center justify-center">
          <div className="flex items-center gap-10 md:gap-20">
            <div className="text-center">
              <div className="w-20 h-20 mx-auto bg-[#383838] rounded-full flex items-center justify-center mb-4">
                <span className="text-2xl font-bold">ARG</span>
              </div>
              <div className="bg-[#3b312e] border border-[#e66140]/30 px-10 py-2 rounded-xl font-bold text-3xl">ARG</div>
            </div>

            <div className="text-center">
              <div className="bg-white/70 text-black font-bold text-5xl px-10 py-4 rounded-2xl mb-4">VS</div>
              <div className="text-[#e64d40] font-medium">Open &bull; 1,250.00 USDC</div>
            </div>

            <div className="text-center">
              <div className="w-20 h-20 mx-auto bg-[#383838] rounded-full flex items-center justify-center mb-4">
                <span className="text-2xl font-bold">POR</span>
              </div>
              <div className="bg-[#3b312e] border border-[#e66140]/30 px-10 py-2 rounded-xl font-bold text-3xl">POR</div>
            </div>
          </div>
        </div>
      </section>

      {/* PROTOCOL WORKFLOW */}
      <section id="workflow" className="py-32 bg-[#282828]">
        <div className="max-w-6xl mx-auto px-6">
          <div className="flex justify-center mb-4">
            <div className="bg-gradient-to-r from-[#e64040] to-[#e64e40] h-1.5 w-28 rounded-full" />
          </div>
          <h2 className="text-center text-5xl font-medium tracking-tighter mb-4 font-heading">Protocol Workflow</h2>
          <p className="text-center text-white/70 text-xl max-w-md mx-auto">A secure, transparent, and decentralized flow from challenge to payout.</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-8 max-w-5xl mx-auto mt-20 px-6">
          {WORKFLOW_STEPS.map(({ step, title, description }) => (
            <div key={step} className="bg-[#2c2c2c] border border-[#373737] p-10 rounded-3xl">
              <div className="text-4xl font-bold text-[#e64040] mb-6">{step}.</div>
              <h3 className="text-3xl font-semibold mb-3">{title}</h3>
              <p className="text-white/70">{description}</p>
            </div>
          ))}
        </div>
      </section>

      {/* THE ADVANTAGE */}
      <section id="advantage" className="py-32 bg-[#282828]">
        <div className="max-w-6xl mx-auto px-6 text-center">
          <div className="flex justify-center mb-4">
            <div className="bg-gradient-to-r from-[#e64040] to-[#e64e40] h-1.5 w-28 rounded-full" />
          </div>
          <h2 className="text-5xl font-medium tracking-tighter mb-6 font-heading">The Advantage</h2>
          <p className="max-w-2xl mx-auto text-white/80 text-xl">Traditional platforms are designed to take a cut. Matchlock is designed to connect peers.</p>
        </div>

        <div className="max-w-5xl mx-auto mt-16 grid grid-cols-1 md:grid-cols-2 gap-8 px-6">
          <div className="bg-[#2c2c2c] p-10 rounded-3xl border border-[#373737]">
            <h3 className="text-red-400 font-bold mb-8 text-xl">TRADITIONAL SPORTSBOOKS</h3>
            <ul className="space-y-6 text-white/70">
              {TRADITIONAL_DRAWBACKS.map((item) => (
                <li key={item} className="flex gap-4">
                  <X className="w-5 h-5 text-red-500 shrink-0 mt-0.5" />
                  {item}
                </li>
              ))}
            </ul>
          </div>

          <div className="bg-[#2c2c2c] p-10 rounded-3xl border border-[#e64040]/30 relative">
            <div className="absolute -top-3 right-8 bg-emerald-600 text-xs px-4 py-1 rounded-full">Verified</div>
            <h3 className="text-[#e64040] font-bold mb-8 text-xl">MATCHLOCK PROTOCOL</h3>
            <ul className="space-y-6">
              {MATCHLOCK_BENEFITS.map((item) => (
                <li key={item} className="flex gap-4">
                  <Check className="w-5 h-5 text-emerald-400 shrink-0 mt-0.5" />
                  {item}
                </li>
              ))}
            </ul>
          </div>
        </div>
      </section>

      {/* FOOTER */}
      <footer className="bg-[#282828] py-16 border-t border-white/10">
        <div className="max-w-6xl mx-auto px-6 flex flex-col md:flex-row justify-between items-center gap-8">
          <div className="text-center md:text-left">
            <div className="flex items-center gap-3 mb-2 justify-center md:justify-start">
              <img src={logoUrl} alt="" className="h-7 w-auto object-contain" />
              <span className="font-heading text-3xl font-bold">Matchlock</span>
            </div>
            <p className="text-white/60">Peer-to-peer sports wagers on Solana.</p>
          </div>
          <div className="flex gap-8 text-sm">
            <Link to="#" className="hover:text-white transition-colors">Docs</Link>
            <Link to="/terms" className="hover:text-white transition-colors">Terms</Link>
            <Link to="#" className="hover:text-white transition-colors">Privacy</Link>
          </div>
        </div>
      </footer>
    </div>
  )
}
