import { useCallback, useSyncExternalStore } from "react";

function subscribe(callback: () => void, query: string) {
  const mql = window.matchMedia(query);
  mql.addEventListener("change", callback);
  return () => mql.removeEventListener("change", callback);
}

export function useMediaQuery(query: string): boolean {
  const subscribeMedia = useCallback(
    (cb: () => void) => subscribe(cb, query),
    [query],
  );

  const getSnapshot = useCallback(() => {
    return window.matchMedia(query).matches;
  }, [query]);

  const getServerSnapshot = useCallback(() => false, []);

  return useSyncExternalStore(subscribeMedia, getSnapshot, getServerSnapshot);
}