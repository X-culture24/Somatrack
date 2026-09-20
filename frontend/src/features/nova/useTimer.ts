import { useEffect, useRef, useState } from "react";

export function useTimer(durationSeconds: number, onExpire?: () => void) {
  const [elapsed, setElapsed] = useState(0);
  const startRef = useRef<number>(Date.now());
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const [running, setRunning] = useState(false);

  function start() {
    startRef.current = Date.now();
    setRunning(true);
  }

  useEffect(() => {
    if (!running) return;
    intervalRef.current = setInterval(() => {
      const secs = Math.floor((Date.now() - startRef.current) / 1000);
      setElapsed(secs);
      if (secs >= durationSeconds) {
        clearInterval(intervalRef.current!);
        onExpire?.();
      }
    }, 1000);
    return () => clearInterval(intervalRef.current!);
  }, [running, durationSeconds]);

  const remaining = Math.max(0, durationSeconds - elapsed);
  const mins = Math.floor(remaining / 60);
  const secs = remaining % 60;
  const formatted = `${String(mins).padStart(2, "0")}:${String(secs).padStart(2, "0")}`;
  const startedAt = new Date(startRef.current).toISOString();
  const urgent = remaining <= 120; // last 2 minutes

  return { elapsed, remaining, formatted, urgent, start, startedAt };
}
