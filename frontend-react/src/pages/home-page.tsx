import { useState, useEffect, useRef } from 'react'
import { Link } from 'react-router-dom'
import { motion, AnimatePresence, useMotionValue, useTransform, animate } from 'framer-motion'
import { Menu, X, XCircle, CheckCircle, BarChart3, Clock, FileText, Trophy } from 'lucide-react'
import Lenis from 'lenis'
import 'lenis/dist/lenis.css'

/* ───────────────────────── Data ───────────────────────── */

const NAV_LINKS = [
  { label: 'Home', href: '#' },
  { label: 'How it Works', href: '#how-it-works' },
  { label: 'The Advantage', href: '#advantage' },
  { label: 'Protocol Roadmap', href: '#roadmap' },
]

const TICKER_ITEMS = [
  '· Powered by TxLINE',
  '· Non-Custodial',
  '· Built on Solana',
  '· Powered by TxLINE',
  '· Non-Custodial',
  '· Built on Solana',
  '· Powered by TxLINE',
  '· Non-Custodial',
  '· Built on Solana',
]

const WORKFLOW_STEPS = [
  {
    number: '01.',
    icon: '/icons/ball.png',
    title: 'PICK A MATCH',
    description: 'Browse live World Cup fixtures with real-time status and market availability.',
  },
  {
    number: '02.',
    icon: '/icons/chess.png',
    title: 'CHALLENGE OR ACCEPT',
    description: 'Create a wager on your side, or take the opposite side of an open challenge from the ledger.',
  },
  {
    number: '03.',
    icon: '/icons/lock.png',
    title: 'STAKES LOCK ON-CHAIN',
    description: 'USDC/tokens escrow to a program-controlled vault (PDA) — never a platform hot wallet.',
  },
  {
    number: '04.',
    icon: '/icons/cup.png',
    title: 'SETTLE AUTOMATICALLY',
    description: 'When the match ends, TxLINE proves the result via cryptographic receipt for 2x payout.',
  },
]

const TRADITIONAL_ITEMS = [
  '5% to 10% house edge built into odds',
  'Opaque backend settlement systems',
  'Operator discretion on withdrawal times',
  'Forced custody of user funds',
]

const MATCHLOCK_ITEMS = [
  '0% house edge (True PvP environment)',
  'Automated cryptographic proof settlement',
  'Cancel open challenges for full refund',
  'Fully non-custodial; funds held in PDA',
]

const SPORTS = [
  { emoji: '⚽', label: 'FOOTBALL' },
  { emoji: '🏀', label: 'BASKETBALL' },
  { emoji: '🥊', label: 'BOXING' },
  { emoji: '🎾', label: 'TENNIS' },
  { emoji: '🏎️', label: 'FORMULA 1' },
  { emoji: '🎯', label: '...& OTHERS' },
]

const ROADMAP_ITEMS = [
  {
    status: 'Coming Soon',
    icon: Clock,
    title: 'Prediction Pools & AMM',
    description:
      'Shared liquidity pools allowing for fractional positions in high-volume markets alongside traditional PvP.',
  },
  {
    status: 'Coming Soon',
    icon: FileText,
    title: 'Verifiable Receipt UI',
    description:
      'Interactive Merkle proof trees directly in the app to audit and verify every settlement on TxLINE.',
  },
  {
    status: 'Coming Soon',
    icon: Trophy,
    title: 'Leaderboards & Gamification',
    description:
      'On-chain standings backed by win rate and net PnL, with exclusive tiered badges for protocol power users.',
  },
]

const containerVariants = {
  hidden: {},
  visible: { transition: { staggerChildren: 0.15 } },
}

const cardVariants = {
  hidden: { opacity: 0, y: 30 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.5 } },
}

/* ───────────────────── Animated Counter ────────────────── */

function AnimatedCounter({ from = 0, to = 1250, duration = 1.5, delay = 1 }: { from?: number; to?: number; duration?: number; delay?: number }) {
  const [displayValue, setDisplayValue] = useState(from.toFixed(2))
  const count = useMotionValue(from)
  const formatted = useTransform(count, (v: number) =>
    v.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 }),
  )

  useEffect(() => {
    const timeout = setTimeout(() => {
      const controls = animate(count, to, { duration, ease: 'easeOut' })
      return () => controls.stop()
    }, delay * 1000)
    return () => clearTimeout(timeout)
  }, [])

  useEffect(() => {
    return formatted.on('change', (v: string) => setDisplayValue(v))
  }, [formatted])

  return <>{displayValue} USDC</>
}

/* ───────────────────── Home Page ────────────────────────── */

export function HomePage() {
  const [mobileOpen, setMobileOpen] = useState(false)
  const [activeLink, setActiveLink] = useState('#')
  const [activeTab, setActiveTab] = useState<'traditional' | 'matchlock'>('matchlock')

  /* smooth scroll with Lenis */
  useEffect(() => {
    const lenis = new Lenis()
    function raf(time: number) {
      lenis.raf(time)
      requestAnimationFrame(raf)
    }
    requestAnimationFrame(raf)
    return () => lenis.destroy()
  }, [])

  /* scroll-spy for nav */
  useEffect(() => {
    const sections = NAV_LINKS.map((link) => ({
      href: link.href,
      el: link.href === '#' ? null : document.querySelector(link.href),
    }))

    const handleScroll = () => {
      const scrollY = window.scrollY + 120
      for (let i = sections.length - 1; i >= 0; i--) {
        const section = sections[i]
        if (section.el && (section.el as HTMLElement).offsetTop <= scrollY) {
          setActiveLink(section.href)
          return
        }
      }
      setActiveLink('#')
    }

    window.addEventListener('scroll', handleScroll, { passive: true })
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  const handleNavClick = (href: string) => {
    setActiveLink(href)
    setMobileOpen(false)
  }

  return (
    <div className="min-h-screen bg-[#282828] text-white">
      {/* ─── NAVBAR ─── */}
      <div className="fixed top-0 left-0 right-0 flex justify-center px-4 md:px-6 pt-4 z-[500]">
        <motion.nav
          initial={{ y: -20, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          transition={{ duration: 0.5 }}
          className="w-full max-w-[1200px] bg-[#2C2C2C] backdrop-blur-md border border-[#373737] rounded-full px-4 md:px-6 py-3 flex items-center justify-between relative"
        >
          <div className="flex items-center gap-1">
            <img src="/logo/logo.png" alt="Matchlock" className="h-6" />
            <span
              className="text-white font-bold leading-none ml-1 hidden sm:inline"
              style={{ fontFamily: "'EB Garamond', serif", fontSize: '24px' }}
            >
              Matchlock
            </span>
          </div>

          <div className="hidden md:flex items-center gap-8">
            {NAV_LINKS.map((link) => (
              <a
                key={link.label}
                href={link.href}
                onClick={() => handleNavClick(link.href)}
                className={`text-sm transition-colors ${
                  activeLink === link.href ? 'text-[#e8473a]' : 'text-gray-400 hover:text-white'
                }`}
              >
                {link.label}
              </a>
            ))}
          </div>

          <div className="hidden md:flex items-center gap-4">
            <Link to="/login" className="text-sm text-gray-400 hover:text-white transition-colors">
              Login
            </Link>
            <Link
              to="/login"
              className="bg-[#e8473a] hover:bg-[#d63d31] text-white text-sm font-medium px-5 py-2.5 rounded-full transition-colors"
            >
              Enter Matchlock
            </Link>
          </div>

          <button onClick={() => setMobileOpen(!mobileOpen)} className="flex md:hidden text-white p-1">
            {mobileOpen ? <X size={22} /> : <Menu size={22} />}
          </button>
        </motion.nav>

        <AnimatePresence>
          {mobileOpen && (
            <motion.div
              initial={{ opacity: 0, y: -10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ duration: 0.2 }}
              className="absolute top-20 left-4 right-4 bg-[#2C2C2C] border border-[#373737] rounded-3xl p-6 flex flex-col gap-4 md:hidden z-50"
            >
              {NAV_LINKS.map((link) => (
                <a
                  key={link.label}
                  href={link.href}
                  onClick={() => handleNavClick(link.href)}
                  className={`text-sm transition-colors ${
                    activeLink === link.href ? 'text-[#e8473a]' : 'text-gray-400 hover:text-white'
                  }`}
                >
                  {link.label}
                </a>
              ))}
              <div className="border-t border-white/10 pt-4 flex flex-col gap-4">
                <Link to="/login" className="text-sm text-gray-400 hover:text-white transition-colors">
                  Login
                </Link>
                <Link
                  to="/login"
                  className="bg-[#e8473a] hover:bg-[#d63d31] text-white text-sm font-medium px-5 py-2.5 rounded-full transition-colors text-center"
                >
                  Enter Matchlock
                </Link>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* ─── HERO ─── */}
      <section className="relative pt-32 pb-16 flex flex-col items-center text-center overflow-hidden">
        <div
          className="absolute inset-0 pointer-events-none opacity-[0.06]"
          style={{
            backgroundImage: 'url(/herobg.png)',
            backgroundSize: 'cover',
            backgroundPosition: 'center',
            backgroundRepeat: 'no-repeat',
          }}
        />
        <div
          className="absolute inset-0 pointer-events-none opacity-[0.02]"
          style={{
            backgroundImage: 'url(/herobg2.png)',
            backgroundSize: 'cover',
            backgroundPosition: 'center',
            backgroundRepeat: 'no-repeat',
          }}
        />

        {/* Soccer ball decoration */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: [0, 1, 1, 1] }}
          transition={{
            duration: 1.8,
            delay: 0.3,
            ease: [0.22, 1, 0.36, 1],
            times: [0, 0.5, 0.75, 1],
          }}
          className="absolute left-0 top-[50px] pointer-events-none select-none hidden lg:block z-[100]"
        >
          <motion.img
            src="/soccer.png"
            alt=""
            className="md:w-[300px] lg:w-[400px] xl:w-[500px] h-auto"
            animate={{ y: [0, -8, 0, -4, 0] }}
            transition={{
              duration: 3,
              delay: 2.5,
              repeat: Infinity,
              repeatDelay: 4,
              ease: 'easeInOut',
            }}
          />
          <motion.div
            className="absolute inset-0 rounded-full"
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: [0, 0.4, 0], scale: [0.8, 1.3, 1.5] }}
            transition={{ duration: 0.6, delay: 1.2, ease: 'easeOut' }}
            style={{ background: 'radial-gradient(circle, rgba(232,71,58,0.3) 0%, transparent 70%)' }}
          />
        </motion.div>

        <div className="relative z-10 px-4">
          <h1 className="text-5xl md:text-6xl lg:text-7xl font-medium text-white leading-none mb-0 tracking-[-2px] md:whitespace-nowrap overflow-hidden">
            {['Challenge', 'another', 'Player.'].map((word, i) => (
              <motion.span
                key={word}
                initial={{ y: '100%', opacity: 0 }}
                animate={{ y: '0%', opacity: 1 }}
                transition={{
                  duration: 0.6,
                  delay: 0.15 + i * 0.12,
                  ease: [0.22, 1, 0.36, 1],
                }}
                className="inline-block mr-[0.3em]"
              >
                {word}
              </motion.span>
            ))}
          </h1>

          <motion.h1
            initial={{ opacity: 0, y: 20, filter: 'blur(12px)' }}
            animate={{ opacity: 1, y: 0, filter: 'blur(0px)' }}
            transition={{ duration: 0.8, delay: 0.65, ease: [0.22, 1, 0.36, 1] }}
            className="text-5xl md:text-6xl lg:text-7xl leading-none mb-8 tracking-[-2px]"
            style={{ fontStyle: 'italic', fontFamily: "'EB Garamond', serif" }}
          >
            <span className="text-white">Settle on-chain.</span>
          </motion.h1>

          <motion.p
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 1.0, ease: 'easeOut' }}
            className="text-gray-400 text-sm md:text-[18px] md:max-w-[620px] mx-auto mb-8 leading-relaxed font-normal text-center"
          >
            Peer-to-peer sports wagers with trustless settlement powered by TxLINE oracle data. No house edge. Your funds, your control.
          </motion.p>

          <motion.div
            initial={{ opacity: 0, y: 15 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 1.2, ease: 'easeOut' }}
            className="flex items-center justify-center gap-4 mb-12"
          >
            <Link
              to="/login"
              className="bg-[#e8473a] hover:bg-[#d63d31] text-white font-medium px-6 py-3 rounded-full transition-colors text-sm"
            >
              Enter Matchlock
            </Link>
            <a
              href="#how-it-works"
              className="text-gray-300 hover:text-white text-sm transition-colors flex items-center gap-1"
            >
              See how it works
            </a>
          </motion.div>
        </div>

        {/* Match Card */}
        <motion.div
          initial={{ opacity: 0, y: 50, scale: 0.95 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          transition={{ duration: 0.8, delay: 1.4, ease: [0.22, 1, 0.36, 1] }}
          className="relative z-10 w-full max-w-[989px] px-4"
        >
          <div
            className="bg-[#2C2C2C] rounded-[30px] md:rounded-[58px] px-5 md:px-16 py-5 md:py-6 overflow-hidden"
            style={{ border: '0.8px solid #373737' }}
          >
            <div className="flex items-center justify-center mb-5">
              <motion.div
                initial={{ opacity: 0, x: -30 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.5, delay: 1.9, ease: [0.22, 1, 0.36, 1] }}
                className="relative left-1 w-[60px] h-[60px] md:w-[79px] md:h-[79px] rounded-full bg-[#383838] flex items-center justify-center text-xl md:text-3xl overflow-hidden"
              >
                <img className=" h-[45px] md:h-[60px]  object-cover" src="/icons/arg.png" alt="" />
              </motion.div>
              <motion.span
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.4, delay: 2.0 }}
                className="bg-[#e6614016] py-1 md:w-22 w-15"
              >
                <span className="text-white font-bold text-lg md:text-[27.68px] leading-none">ARG</span>
              </motion.span>

              <motion.div
                initial={{ opacity: 0, scale: 0 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ duration: 0.4, delay: 2.15, type: 'spring', stiffness: 300, damping: 20 }}
                className="font-bold flex items-center justify-center text-black"
                style={{
                  width: '50px',
                  height: '60px',
                  borderRadius: '14px',
                  border: '0.72px solid rgba(255,255,255,0.15)',
                  fontSize: '18px',
                  lineHeight: '100%',
                  backgroundColor: '#FFFFFFC9',
                }}
              >
                <span className="hidden md:inline" style={{ fontSize: '27.68px' }}>VS</span>
                <span className="md:hidden">VS</span>
              </motion.div>

              <motion.span
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.4, delay: 2.0 }}
                className="bg-[#e6614016] py-1 md:w-22 w-15"
              >
                <span className="text-white font-bold text-lg md:text-[27.68px] leading-none">POR</span>
              </motion.span>
              <motion.div
                initial={{ opacity: 0, x: 30 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.5, delay: 1.9, ease: [0.22, 1, 0.36, 1] }}
                className="relative right-1 w-[60px] h-[60px] md:w-[79px] md:h-[79px] rounded-full bg-[#383838] flex items-center justify-center text-xl md:text-3xl overflow-hidden"
              >
                <img className=" h-[45px] md:h-[60px]  object-cover" src="/icons/por.png" alt="" />
              </motion.div>
            </div>

            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.5, delay: 2.3 }}
              className="mb-4 w-full md:w-[80%] m-auto"
            >
              <div className="flex justify-between text-xs text-gray-500 mb-2">
                <span>Market Stake</span>
                <span><AnimatedCounter from={0} to={1250} duration={1.5} delay={2.5} /></span>
              </div>
              <div className="w-full h-1.5 bg-gray-700 rounded-full overflow-hidden">
                <motion.div
                  className="h-full rounded-full"
                  initial={{ width: '0%' }}
                  animate={{ width: '45%' }}
                  transition={{ duration: 1.5, delay: 2.5, ease: 'easeOut' }}
                  style={{ background: 'linear-gradient(90deg, #e8473a, #f0a060)' }}
                />
              </div>
            </motion.div>

            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.4, delay: 2.6 }}
              className="flex items-center justify-center gap-3"
            >
              <span className="text-[#e8473a] text-sm">Open</span>
              <button className="flex items-center gap-2 text-white text-sm font-medium hover:text-gray-300 transition-colors">
                Lock Game <span>&rarr;</span>
              </button>
            </motion.div>
          </div>
        </motion.div>
      </section>

      {/* ─── TICKER ─── */}
      <div className="w-full overflow-hidden py-5">
        <div
          className="relative border-y border-white/10 bg-[#222222] flex items-center z-[200]"
          style={{
            height: '86px',
            transform: 'rotate(-1.47deg)',
            marginLeft: '-32px',
            width: 'calc(100% + 64px)',
          }}
        >
          <motion.div
            className="flex gap-8 whitespace-nowrap"
            animate={{ x: ['0%', '-50%'] }}
            transition={{ duration: 20, repeat: Infinity, ease: 'linear' }}
          >
            {[...TICKER_ITEMS, ...TICKER_ITEMS].map((item, i) => (
              <span key={i} className="text-gray-500 text-sm flex-shrink-0">
                {item}
              </span>
            ))}
          </motion.div>
        </div>
      </div>

      {/* ─── PROTOCOL WORKFLOW ─── */}
      <section id="how-it-works" className="py-24 px-8 max-w-[1440px] mx-auto">
        <div className="w-12 h-[3px] bg-[#e8473a] mb-6" />

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-16 items-start">
          <motion.div
            initial={{ opacity: 0, x: -30 }}
            whileInView={{ opacity: 1, x: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.6 }}
          >
            <h2 className="text-4xl md:text-5xl font-medium text-white mb-4 tracking-[-2px]">
              Protocol Workflow
            </h2>
            <p className="text-gray-400 text-base md:text-[20px] max-w-sm font-light">
              A secure, transparent, and decentralized flow from challenge to payout.
            </p>
          </motion.div>

          <motion.div
            className="grid grid-cols-1 sm:grid-cols-2 gap-4"
            variants={containerVariants}
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
          >
            {WORKFLOW_STEPS.map((step) => (
              <motion.div
                key={step.number}
                variants={cardVariants}
                className="bg-[#2C2C2C] border border-white/5 rounded-xl p-6 hover:border-[#e8473a]/30 transition-colors"
              >
                <span
                  className="text-[#e8473a] font-bold text-[22px] leading-none tracking-[-2.19px] mb-4 inline-flex items-center justify-center"
                  style={{
                    width: '47px',
                    height: '43px',
                    borderRadius: '13.12px',
                    border: '0.55px solid #E6724012',
                    backgroundColor: 'rgba(230, 114, 64, 0.07)',
                  }}
                >
                  {step.number}
                </span>
                <div className="w-10 h-10 rounded-lg bg-white/5 flex items-center justify-center mb-4">
                  <img src={step.icon} alt="" className="w-8 h-8 object-cover" />
                </div>
                <h3 className="text-white font-bold text-[16px] leading-[20px] tracking-[-0.45px] uppercase mb-2">
                  {step.title}
                </h3>
                <p className="text-gray-500 text-xs leading-tight">{step.description}</p>
              </motion.div>
            ))}
          </motion.div>
        </div>
      </section>

      {/* ─── ADVANTAGE ─── */}
      <section id="advantage" className="py-24 px-4 md:px-8 max-w-[1440px] mx-auto">
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="text-center mb-16"
        >
          <div className="w-16 h-[3px] m-auto bg-[#e8473a] mb-6" />
          <h2 className="text-3xl md:text-[48px] font-medium text-white mb-4 tracking-[-2px] leading-none">
            The Advantage
          </h2>
          <p className="text-gray-400 text-base md:text-[18px] md:max-w-[620px] mx-auto font-light leading-relaxed">
            Traditional platforms are designed to take a cut. Matchlock is designed to connect peers. We've removed the middleman, the house edge, and the counterparty risk.
          </p>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.2 }}
          className="max-w-3xl mx-auto bg-[#2C2C2C] border border-white/5 rounded-2xl overflow-hidden"
        >
          <div className="grid grid-cols-2 relative">
            <motion.div
              className="absolute bottom-0 h-[2px] bg-[#e8473a]"
              animate={{ left: activeTab === 'traditional' ? '0%' : '50%' }}
              transition={{ duration: 0.3, ease: 'easeInOut' }}
              style={{ width: '50%' }}
            />
            <button
              onClick={() => setActiveTab('traditional')}
              className="px-4 md:px-8 py-4 md:py-5 text-left border-b border-white/5 transition-colors duration-300 text-gray-400 hover:text-white"
              style={{ backgroundColor: activeTab === 'traditional' ? '#E6724008' : 'transparent' }}
            >
              <span className="text-xs md:text-sm font-semibold tracking-wide uppercase">
                Traditional Sportsbooks
              </span>
            </button>
            <button
              onClick={() => setActiveTab('matchlock')}
              className="px-4 md:px-8 py-4 md:py-5 text-left border-b border-white/5 transition-colors duration-300 flex flex-wrap items-center gap-1.5 md:gap-2 text-[#e8473a]"
              style={{ backgroundColor: activeTab === 'matchlock' ? '#E6724008' : 'transparent' }}
            >
              <span className="text-xs md:text-sm font-semibold tracking-wide uppercase">
                Matchlock Protocol
              </span>
              <span className="bg-green-500/10 text-green-400 text-[10px] md:text-xs px-1.5 md:px-2 py-0.5 rounded-full flex items-center gap-1 whitespace-nowrap">
                <CheckCircle size={10} /> Verified
              </span>
            </button>
          </div>

          {/* Desktop: both columns */}
          <div className="hidden md:grid md:grid-cols-2">
            <div
              className="px-8 py-6 flex flex-col gap-5 transition-colors duration-300 border-r border-white/5"
              style={{ backgroundColor: activeTab === 'traditional' ? '#E6724008' : 'transparent' }}
            >
              {TRADITIONAL_ITEMS.map((item, i) => (
                <div key={i} className="flex items-center gap-3">
                  <XCircle size={20} className="text-red-400/60 shrink-0" />
                  <span className="text-gray-400 text-base">{item}</span>
                </div>
              ))}
            </div>
            <div
              className="px-8 py-6 flex flex-col gap-5 transition-colors duration-300"
              style={{ backgroundColor: activeTab === 'matchlock' ? '#E6724008' : 'transparent' }}
            >
              {MATCHLOCK_ITEMS.map((item, i) => (
                <div key={i} className="flex items-center gap-3">
                  <CheckCircle size={20} className="text-green-400/60 shrink-0" />
                  <span className="text-gray-300 text-base">{item}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Mobile: active tab only */}
          <div className="md:hidden px-6 py-6">
            <AnimatePresence mode="wait">
              {activeTab === 'traditional' ? (
                <motion.div
                  key="traditional"
                  initial={{ opacity: 0, x: -15 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: 15 }}
                  transition={{ duration: 0.2 }}
                  className="flex flex-col gap-5"
                >
                  {TRADITIONAL_ITEMS.map((item, i) => (
                    <div key={i} className="flex items-center gap-3">
                      <XCircle size={20} className="text-red-400/60 shrink-0" />
                      <span className="text-gray-400 text-sm">{item}</span>
                    </div>
                  ))}
                </motion.div>
              ) : (
                <motion.div
                  key="matchlock"
                  initial={{ opacity: 0, x: 15 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: -15 }}
                  transition={{ duration: 0.2 }}
                  className="flex flex-col gap-5"
                >
                  {MATCHLOCK_ITEMS.map((item, i) => (
                    <div key={i} className="flex items-center gap-3">
                      <CheckCircle size={20} className="text-green-400/60 shrink-0" />
                      <span className="text-gray-300 text-sm">{item}</span>
                    </div>
                  ))}
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        </motion.div>
      </section>

      {/* ─── DASHBOARD OVERVIEW ─── */}
      <section className="py-24 px-8 max-w-[1440px] mx-auto">
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center"
        >
          <div>
            <span className="text-[#e8473a] text-xs font-semibold tracking-widest uppercase mb-4 block">
              Dashboard Overview
            </span>
            <h2 className="text-4xl md:text-5xl font-medium text-white leading-tight mb-4 tracking-[-2px]">
              Transparent ledgers.
              <br />
              Plain rules.
            </h2>
            <p className="text-gray-400 text-base md:text-[20px] max-w-md font-light">
              No confusing bookmaker odds. Just simple head-to-head challenges.
            </p>
          </div>

          <div className="bg-[#1e1e1e] border border-white/5 rounded-2xl p-8 flex items-center justify-center min-h-[280px]">
            <div className="text-gray-600 flex flex-col items-center gap-4">
              <BarChart3 size={80} strokeWidth={1} />
            </div>
          </div>
        </motion.div>
      </section>

      {/* ─── ACTIVE SPORTS ─── */}
      <section className="py-16 px-8 max-w-[1440px] mx-auto">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5 }}
          className="text-center"
        >
          <div className="flex items-center gap-4 mb-8">
            <div className="flex-1 h-px bg-white/10" />
            <span className="text-gray-500 text-xs tracking-widest uppercase">Active Sports</span>
            <div className="flex-[10] h-px bg-white/10" />
          </div>

          <div className="flex flex-wrap gap-3">
            {SPORTS.map((sport) => (
              <motion.div
                key={sport.label}
                whileHover={{ scale: 1.05 }}
                className="flex items-center gap-2.5 bg-[#2C2C2C] border border-white/5 rounded-full px-6 py-3 hover:border-[#e8473a]/30 transition-colors cursor-default"
              >
                <span className="text-lg">{sport.emoji}</span>
                <span className="text-white text-sm font-medium tracking-wide">{sport.label}</span>
              </motion.div>
            ))}
          </div>
        </motion.div>
      </section>

      {/* ─── ROADMAP ─── */}
      <section id="roadmap" className="py-24 px-8 max-w-[1440px] mx-auto">
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="mb-12"
        >
          <div className="w-12 h-[3px] bg-[#e8473a] mb-6" />

          <div className="flex items-end justify-between flex-wrap gap-4 mb-12">
            <div>
              <h2 className="text-4xl md:text-[48px] font-medium text-white mb-3 tracking-[-2px] leading-none">
                Protocol Roadmap
              </h2>
              <p className="text-gray-400 text-base md:text-[20px] max-w-md font-light">
                Expanding the boundaries of decentralized
                <br />
                peer-to-peer prediction markets.
              </p>
            </div>
            <a
              href="#"
              className="text-[#e8473a] text-sm hover:text-[#ff6b5e] transition-colors rounded-full px-4 py-2"
              style={{ backgroundColor: '#E661400F', border: '1px solid #E6614021' }}
            >
              Phase 1 Live: PvP Markets
            </a>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {ROADMAP_ITEMS.map((item, i) => (
              <motion.div
                key={item.title}
                initial={{ opacity: 0, y: 30 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ duration: 0.5, delay: i * 0.1 }}
                className="bg-[#2C2C2C] border border-white/5 rounded-[51px] p-6 hover:border-[#e8473a]/30 transition-colors"
              >
                <div className="flex items-center justify-between mb-4">
                  <span className="text-[#A4A4A4] text-[10px] font-semibold tracking-widest uppercase bg-[#303030] px-3 py-1 rounded-full">
                    {item.status}
                  </span>
                  <div
                    className="flex items-center justify-center"
                    style={{
                      width: '40px',
                      height: '38px',
                      borderRadius: '17px',
                      border: '1px solid #353535',
                    }}
                  >
                    <item.icon size={16} className="text-[#e8473a] opacity-60" />
                  </div>
                </div>
                <h3 className="text-white font-semibold text-[22px] leading-none tracking-normal mb-3">
                  {item.title}
                </h3>
                <p className="text-[#A4A4A4] text-[16px] leading-[1.4] font-normal">{item.description}</p>
              </motion.div>
            ))}
          </div>
        </motion.div>
      </section>

      {/* ─── FOOTER ─── */}
      <footer>
        <div className="max-w-[1440px] mx-auto px-8 py-12">
          <div className="flex items-end justify-between flex-wrap gap-8 mb-12">
            <div>
              <div className="flex items-center gap-2 mb-2" style={{ lineHeight: 1 }}>
                <img src="/logo/logo.png" alt="Matchlock" className="h-6" />
                <span
                  className="text-white font-bold leading-none"
                  style={{ fontFamily: "'EB Garamond', serif", fontSize: '32.86px' }}
                >
                  Matchlock
                </span>
              </div>
              <p className="text-gray-500 text-sm font-light">Peer-to-peer sports wagers on Solana.</p>
            </div>

            <div className="flex items-center gap-6">
              <a href="#" className="text-gray-400 text-sm hover:text-white transition-colors">
                Docs
              </a>
              <Link to="/terms" className="text-gray-400 text-sm hover:text-white transition-colors">
                Terms of Service
              </Link>
              <a href="#" className="text-gray-400 text-sm hover:text-white transition-colors">
                Privacy
              </a>
            </div>
          </div>

          <div className="flex items-center justify-between flex-wrap gap-4 pt-8 border-t border-white/5">
            <p className="text-gray-600 text-xs">
              Powered by <span className="text-gray-400">TxLINE</span> Oracle Data
            </p>

            <div className="flex items-center gap-4">
              {/* Discord */}
              <a href="#" className="text-gray-500 hover:text-white transition-colors">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M20.317 4.3698a19.7913 19.7913 0 00-4.8851-1.5152.0741.0741 0 00-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 00-.0785-.037 19.7363 19.7363 0 00-4.8852 1.515.0699.0699 0 00-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 00.0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 00.0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 00-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 01-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 01.0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 01.0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 01-.0066.1276 12.2986 12.2986 0 01-1.873.8914.0766.0766 0 00-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 00.0842.0286c1.961-.6067 3.9495-1.5219 6.0023-3.0294a.077.077 0 00.0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 00-.0312-.0286z" />
                </svg>
              </a>
              {/* Telegram */}
              <a href="#" className="text-gray-500 hover:text-white transition-colors">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M11.944 0A12 12 0 0 0 0 12a12 12 0 0 0 12 12 12 12 0 0 0 12-12A12 12 0 0 0 12 0a12 12 0 0 0-.056 0zm4.962 7.224c.1-.002.321.023.465.14a.506.506 0 0 1 .171.325c.016.093.036.306.02.472-.18 1.898-.962 6.502-1.36 8.627-.168.9-.499 1.201-.82 1.23-.696.065-1.225-.46-1.9-.902-1.056-.693-1.653-1.124-2.678-1.8-1.185-.78-.417-1.21.258-1.91.177-.184 3.247-2.977 3.307-3.23.007-.032.014-.15-.056-.212s-.174-.041-.249-.024c-.106.024-1.793 1.14-5.061 3.345-.48.33-.913.49-1.302.48-.428-.008-1.252-.241-1.865-.44-.752-.245-1.349-.374-1.297-.789.027-.216.325-.437.893-.663 3.498-1.524 5.83-2.529 6.998-3.014 3.332-1.386 4.025-1.627 4.476-1.635z" />
                </svg>
              </a>
              {/* X / Twitter */}
              <a href="#" className="text-gray-500 hover:text-white transition-colors">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
                </svg>
              </a>
            </div>
          </div>
        </div>
      </footer>
    </div>
  )
}
