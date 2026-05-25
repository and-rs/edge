import { MatchType, StageStatus, type Stage } from "~/api/signals/v1/signals_pb"

export const revealClass =
  "motion-safe:animate-[signal-reveal_520ms_cubic-bezier(0.22,1,0.36,1)_both]"

export const streamPulseClass =
  "motion-safe:animate-[signal-pulse_1400ms_ease-in-out_infinite]"

export const stageToneClass = (stage?: Stage): string => {
  switch (stage?.status) {
    case StageStatus.READY:
      return "border-success text-emerald-700 dark:text-emerald-300"
    case StageStatus.RUNNING:
      return "border-secondary text-secondary"
    case StageStatus.DISABLED:
    case StageStatus.SKIPPED:
      return "border-muted text-muted-foreground"
    case StageStatus.MISCONFIGURED:
    case StageStatus.INVALID:
      return "border-amber-500/40 text-amber-700 dark:text-amber-300"
    case StageStatus.TIMEOUT:
    case StageStatus.UNAVAILABLE:
      return "border-red-500/40 text-red-700 dark:text-red-300"
    default:
      return "border-muted text-foreground"
  }
}

export const matchToneClass = (matchType: MatchType): string => {
  switch (matchType) {
    case MatchType.MARKET_LINKED:
      return "border-success"
    case MatchType.WATCHLIST:
      return "border-secondary"
    default:
      return "border-muted"
  }
}
