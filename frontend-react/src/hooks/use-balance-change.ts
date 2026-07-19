import { useEffect, useRef, useState } from "react";

export interface BalanceChange {
  delta: number;
  direction: "up" | "down";
}

export function useBalanceChange(current: number) {
  const prev = useRef(current);
  const [change, setChange] = useState<BalanceChange | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout>>();

  useEffect(() => {
    const diff = current - prev.current;
    prev.current = current;

    if (diff === 0) return;

    clearTimeout(timer.current);
    setChange({ delta: Math.abs(diff), direction: diff > 0 ? "up" : "down" });

    timer.current = setTimeout(() => setChange(null), 3500);
  }, [current]);

  return change;
}
