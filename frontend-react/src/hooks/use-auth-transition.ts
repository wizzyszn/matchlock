import { useEffect, useState } from 'react'

const DEFAULT_MIN_MS = 2200

export function useAuthTransition(active: boolean, minMs = DEFAULT_MIN_MS) {
  const [visible, setVisible] = useState(active)

  useEffect(() => {
    if (!active) {
      setVisible(false)
      return
    }

    setVisible(true)
    const timer = window.setTimeout(() => setVisible(false), minMs)
    return () => window.clearTimeout(timer)
  }, [active, minMs])

  return visible
}

export function delay(ms: number) {
  return new Promise<void>((resolve) => {
    window.setTimeout(resolve, ms)
  })
}