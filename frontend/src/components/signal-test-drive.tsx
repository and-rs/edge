import type { Timestamp } from "@bufbuild/protobuf/wkt"
import { For, onMount, Show } from "solid-js"
import { createStore } from "solid-js/store"
import {
  MatchType,
  type Signal,
  SignalUpdateType,
  type Stage,
  StageStatus,
} from "~/api/signals/v1/signals_pb"
import { signalsClient } from "~/lib/rpc"
import { Button } from "./ui/button"

const CACHE_KEY = "iridium-edge.signal-test-drive.v1"
const CACHE_TTL_MS = 60 * 60 * 1000

type CachedTimestamp = {
  seconds: string
  nanos: number
}

type CachedStage = {
  status: StageStatus
  label: string
  detail: string
}

type CachedSignal = Omit<Signal, "publishedAt" | "aiJudgment"> & {
  publishedAt?: CachedTimestamp
  aiJudgment?: CachedStage
}

type SignalCache = {
  savedAt: number
  eventIngest?: CachedStage
  marketIngest?: CachedStage
  aiJudgment?: CachedStage
  summary: string
  failureReason: string
  signals: CachedSignal[]
}

const cacheAvailable = (): boolean => typeof window !== "undefined"

const cacheTimestamp = (timestamp?: Timestamp): CachedTimestamp | undefined => {
  if (!timestamp) return undefined
  return {
    seconds: timestamp.seconds?.toString() ?? "0",
    nanos: timestamp.nanos ?? 0,
  }
}

const hydrateTimestamp = (timestamp?: CachedTimestamp): Timestamp | undefined => {
  if (!timestamp) return undefined
  return {
    seconds: BigInt(timestamp.seconds),
    nanos: timestamp.nanos,
  } as Timestamp
}

const cacheStage = (stage?: Stage): CachedStage | undefined =>
  stage
    ? {
        status: stage.status,
        label: stage.label,
        detail: stage.detail,
      }
    : undefined

const hydrateStage = (stage?: CachedStage): Stage | undefined =>
  stage ? ({ ...stage } as Stage) : undefined

const cacheSignal = (signal: Signal): CachedSignal => ({
  ...signal,
  publishedAt: cacheTimestamp(signal.publishedAt),
  aiJudgment: cacheStage(signal.aiJudgment),
})

const hydrateSignal = (signal: CachedSignal): Signal =>
  ({
    ...signal,
    publishedAt: hydrateTimestamp(signal.publishedAt),
    aiJudgment: hydrateStage(signal.aiJudgment),
  }) as Signal

const formatPublishedAt = (publishedAt?: Timestamp): string => {
  if (!publishedAt?.seconds) return "unknown"
  return new Date(Number(publishedAt.seconds) * 1000).toLocaleString()
}

const stageToneClass = (stage?: Stage): string => {
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

const matchToneClass = (matchType: MatchType): string => {
  switch (matchType) {
    case MatchType.MARKET_LINKED:
      return "border-success"
    case MatchType.WATCHLIST:
      return "border-secondary"
    default:
      return "border-muted"
  }
}

export const SignalTestDrive = () => {
  const [data, setData] = createStore({
    loading: false,
    loaded: false,
    eventIngest: undefined as Stage | undefined,
    marketIngest: undefined as Stage | undefined,
    aiJudgment: undefined as Stage | undefined,
    summary: "",
    failureReason: "",
    signals: [] as Signal[],
    error: "",
  })

  const writeCache = () => {
    if (!cacheAvailable()) return
    const cache: SignalCache = {
      savedAt: Date.now(),
      eventIngest: cacheStage(data.eventIngest),
      marketIngest: cacheStage(data.marketIngest),
      aiJudgment: cacheStage(data.aiJudgment),
      summary: data.summary,
      failureReason: data.failureReason,
      signals: data.signals.map(cacheSignal),
    }
    window.sessionStorage.setItem(CACHE_KEY, JSON.stringify(cache))
  }

  const clearCache = () => {
    if (!cacheAvailable()) return
    window.sessionStorage.removeItem(CACHE_KEY)
  }

  const hydrateCache = () => {
    if (!cacheAvailable()) return
    const rawCache = window.sessionStorage.getItem(CACHE_KEY)
    if (!rawCache) return

    try {
      const cache = JSON.parse(rawCache) as SignalCache
      if (Date.now() - cache.savedAt > CACHE_TTL_MS) {
        clearCache()
        return
      }
      setData({
        loading: false,
        loaded: true,
        eventIngest: hydrateStage(cache.eventIngest),
        marketIngest: hydrateStage(cache.marketIngest),
        aiJudgment: hydrateStage(cache.aiJudgment),
        summary: cache.summary,
        failureReason: cache.failureReason,
        signals: cache.signals.map(hydrateSignal),
        error: "",
      })
    } catch (err) {
      clearCache()
      console.error("Signal cache hydration failed:", err)
    }
  }

  onMount(hydrateCache)

  const stages = (): Stage[] =>
    [data.eventIngest, data.marketIngest, data.aiJudgment].filter(
      (stage): stage is Stage => Boolean(stage),
    )

  const upsertSignal = (signal: Signal) => {
    let nextSignals = data.signals
    const index = data.signals.findIndex(
      (current) => current.sourceUrl === signal.sourceUrl,
    )
    if (index >= 0) {
      setData("signals", index, signal)
      nextSignals = data.signals.map((current, currentIndex) =>
        currentIndex === index ? signal : current,
      )
    } else {
      setData("signals", data.signals.length, signal)
      nextSignals = [...data.signals, signal]
    }
    return nextSignals
  }

  const loadSignals = async () => {
    setData({
      loading: true,
      loaded: false,
      eventIngest: undefined,
      marketIngest: undefined,
      aiJudgment: undefined,
      summary: "",
      failureReason: "",
      signals: [],
      error: "",
    })
    clearCache()

    try {
      for await (const update of signalsClient.streamSignals({})) {
        if (update.eventIngest) setData("eventIngest", update.eventIngest)
        if (update.marketIngest) setData("marketIngest", update.marketIngest)
        if (update.aiJudgment) setData("aiJudgment", update.aiJudgment)
        if (update.signal) upsertSignal(update.signal)
        if (update.summary) setData("summary", update.summary)
        if (update.failureReason) setData("failureReason", update.failureReason)
        if (update.eventIngest || update.marketIngest || update.aiJudgment || update.signal || update.summary || update.failureReason) {
          writeCache()
        }
        if (update.type === SignalUpdateType.DONE || update.done) {
          setData({ loading: false, loaded: true })
          writeCache()
        }
      }
      setData({ loading: false, loaded: true })
      writeCache()
    } catch (err) {
      setData({
        loading: false,
        error: err instanceof Error ? err.message : "Signal stream failed",
      })
      console.error("Signal stream failed:", err)
    }
  }

  return (
    <section id="signal-test-drive" class="flex flex-col gap-3 bento-cell">
      <div class="flex flex-row gap-2">
        <Button
          onClick={loadSignals}
          disabled={data.loading}
          class="w-min whitespace-nowrap"
        >
          {data.loading
            ? "STREAMING..."
            : data.loaded
              ? "REFRESH TEST DRIVE"
              : "LOAD TEST DRIVE"}
        </Button>
      </div>

      <Show when={data.loaded || data.loading || data.signals.length > 0}>
        <div class="flex flex-col gap-3">
          <Show when={data.summary}>
            {(summary) => (
              <div class="flex flex-col gap-2 p-3 border border-muted">
                <div class="text-xs uppercase tracking-[0.24em] text-muted-foreground">
                  pipeline summary
                </div>
                <div>{summary()}</div>
              </div>
            )}
          </Show>

          <div class="grid gap-2 md:grid-cols-3">
            <For each={stages()}>
              {(stage) => (
                <div
                  class={`flex flex-col gap-1 border p-3 ${stageToneClass(stage)}`}
                >
                  <div class="text-xs uppercase opacity-70 tracking-[0.24em]">
                    {stage.label}
                  </div>
                  <div class="text-sm opacity-80">{stage.detail}</div>
                </div>
              )}
            </For>
          </div>

          <Show when={data.failureReason}>
            {(reason) => (
              <div class="font-mono text-xs text-amber-600 dark:text-amber-400">
                {reason()}
              </div>
            )}
          </Show>

          <div class="flex flex-col gap-3">
            <For each={data.signals}>
              {(signal: Signal) => (
                <a
                  class={`flex flex-col gap-3 rounded border p-3 transition-all hover:border-secondary ${matchToneClass(signal.finalMatchType)}`}
                  href={signal.marketUrl || signal.sourceUrl}
                  target="_blank"
                  rel="noreferrer"
                >
                  <div class="flex flex-col gap-1">
                    <div class="text-muted-foreground">
                      {signal.linkStateLabel} · score {signal.score.toFixed(1)}
                    </div>
                    <div class="font-medium">{signal.headline}</div>
                    <div class="text-sm opacity-80">
                      {formatPublishedAt(signal.publishedAt)}
                    </div>
                  </div>
                  <div class="text-sm">{signal.thesis}</div>
                  <div class="text-sm opacity-80">{signal.whyItMatters}</div>
                  <div class="grid gap-2 text-xs opacity-80 md:grid-cols-2">
                    <div>market: {signal.marketQuestion}</div>
                    <div>ai: {signal.aiJudgment?.label}</div>
                  </div>
                  <Show when={signal.failureReason}>
                    {(reason) => (
                      <div class="font-mono text-xs text-amber-600 dark:text-amber-400">
                        {reason()}
                      </div>
                    )}
                  </Show>
                </a>
              )}
            </For>
          </div>
        </div>
      </Show>

      <Show when={data.error}>
        {(error) => (
          <div class="font-mono text-sm text-red-600 dark:text-red-400">
            {error()}
          </div>
        )}
      </Show>
    </section>
  )
}
