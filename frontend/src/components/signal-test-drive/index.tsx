import { create } from "@bufbuild/protobuf"
import ChevronsRight from "lucide-solid/icons/chevrons-right"
import LoaderCircle from "lucide-solid/icons/loader-circle"
import { For, Show, createSignal, onCleanup, onMount } from "solid-js"
import { createStore } from "solid-js/store"
import {
  SignalHuntStateSchema,
  type Signal,
  type SignalHuntState,
  type Stage,
} from "~/api/signals/v1/signals_pb"
import { signalsClient } from "~/lib/rpc"
import { Button } from "../ui/button"
import {
  SIGNAL_HUNT_COOLDOWN_MS,
  clearSignalCache,
  readSignalCache,
  readSignalCooldown,
  writeSignalCache,
  writeSignalCooldown,
} from "./cache"
import { LoadingStream, SignalCard, StageCard } from "./cards"

type SignalTestDriveState = SignalHuntState & {
  loading: boolean
  loaded: boolean
  error: string
}

const createInitialState = (): SignalTestDriveState => ({
  ...create(SignalHuntStateSchema),
  loading: false,
  loaded: false,
  error: "",
})

export const SignalTestDrive = () => {
  const [data, setData] = createStore<SignalTestDriveState>(
    createInitialState(),
  )
  const [cooldownRemainingMs, setCooldownRemainingMs] = createSignal(0)

  const stages = (): Stage[] =>
    [
      data.stages?.eventIngest,
      data.stages?.marketIngest,
      data.stages?.aiJudgment,
    ].filter((stage): stage is Stage => Boolean(stage))

  const snapshot = (signals: Signal[] = data.signals): SignalHuntState =>
    create(SignalHuntStateSchema, {
      stages: data.stages,
      summary: data.summary,
      signals,
    })

  const upsertSignal = (signal: Signal): Signal[] => {
    const index = data.signals.findIndex(
      (current) => current.sourceUrl === signal.sourceUrl,
    )
    if (index >= 0) {
      const signals = data.signals.map((current, currentIndex) =>
        currentIndex === index ? signal : current,
      )
      setData("signals", index, signal)
      return signals
    }

    const signals = [...data.signals, signal]
    setData("signals", data.signals.length, signal)
    return signals
  }

  const hydrateCache = () => {
    const cached = readSignalCache()
    if (cached) setData({ ...cached, loaded: true, loading: false, error: "" })
  }

  const refreshCooldown = () => {
    const remainingMs = Math.max(0, readSignalCooldown() - Date.now())
    setCooldownRemainingMs(remainingMs)
  }

  onMount(() => {
    hydrateCache()
    refreshCooldown()
    const interval = window.setInterval(refreshCooldown, 1000)
    onCleanup(() => window.clearInterval(interval))
  })

  const persist = (signals?: Signal[]) => writeSignalCache(snapshot(signals))

  const startCooldown = () => {
    const cooldownUntil = Date.now() + SIGNAL_HUNT_COOLDOWN_MS
    writeSignalCooldown(cooldownUntil)
    setCooldownRemainingMs(SIGNAL_HUNT_COOLDOWN_MS)
  }

  const loadSignals = async () => {
    if (data.loading || cooldownRemainingMs() > 0) return
    setData({ ...createInitialState(), loading: true })
    clearSignalCache()

    let completed = false

    try {
      for await (const update of signalsClient.streamSignals({})) {
        let signals: Signal[] | undefined

        switch (update.event.case) {
          case "stages":
            setData("stages", update.event.value)
            persist()
            break
          case "signal":
            signals = upsertSignal(update.event.value)
            persist(signals)
            break
          case "summary":
            setData("summary", update.event.value)
            persist()
            break
          case "done":
            completed = true
            setData({ loading: false, loaded: true })
            startCooldown()
            persist(signals)
            break
        }
      }

      if (completed) return
      setData({ loading: false, loaded: true })
      persist()
    } catch (err) {
      setData({
        loading: false,
        error: err instanceof Error ? err.message : "Signal stream failed",
      })
      console.error("Signal stream failed:", err)
    }
  }

  const cooldownLabel = () => {
    const seconds = Math.ceil(cooldownRemainingMs() / 1000)
    if (data.loading) return "streaming signal hunt"
    if (seconds > 0) return `refresh in ${seconds}s`
    if (data.loaded) return "refresh signal hunt"
    return "start signal hunt"
  }

  return (
    <section id="signal-test-drive" class="flex flex-col gap-3 bento-cell">
      <span class="">
        Honestly I am simply gathering from kalshi and coinbase and doing an AI
        model pass with deepseek. I could not finish. I could not produce
        anything with ARC. I think I had good ideas. I will continue to work on
        this idea even after the hackathon.
      </span>

      <span>
        I believe in news/events grouping, and I believe in USDC payments. You
        can perhaps cheack PLAN.md if you haven't I think it has most if not all
        my vision.
      </span>
      <div class="flex flex-row gap-2">
        <Button
          onClick={loadSignals}
          disabled={data.loading || cooldownRemainingMs() > 0}
          size="lg"
          class="gap-2 px-5 font-semibold uppercase border disabled:opacity-60 disabled:cursor-not-allowed border-secondary/50 bg-secondary tracking-[0.18em] text-secondary-foreground hover:bg-secondary/90"
        >
          <Show
            when={data.loading || cooldownRemainingMs() > 0}
            fallback={<ChevronsRight class="size-4" />}
          >
            <LoaderCircle class="animate-spin size-4" />
          </Show>
          {cooldownLabel()}
        </Button>
      </div>

      <Show when={data.loading && data.signals.length === 0}>
        <LoadingStream />
      </Show>

      <Show when={data.loaded || data.loading || data.signals.length > 0}>
        <div class="flex flex-col gap-3">
          <Show when={data.summary?.text}>
            {(summary) => (
              <div class="flex flex-col gap-3 p-4 border border-muted bg-card">
                <div class="py-1 px-2 text-xs font-semibold uppercase border w-fit border-secondary/50 bg-secondary tracking-[0.18em] text-secondary-foreground">
                  pipeline summary
                </div>
                <div class="text-lg leading-8 text-foreground/95">
                  {summary()}
                </div>
              </div>
            )}
          </Show>

          <div class="grid gap-2 md:grid-cols-3">
            <For each={stages()}>{(stage) => <StageCard stage={stage} />}</For>
          </div>

          <Show when={data.summary?.failureReason}>
            {(reason) => (
              <div class="p-3 font-mono text-xs text-amber-600 border dark:text-amber-400 border-amber-500/30 bg-amber-500/8">
                {reason()}
              </div>
            )}
          </Show>

          <div class="flex flex-col gap-3">
            <For each={data.signals}>
              {(signal) => <SignalCard signal={signal} />}
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

