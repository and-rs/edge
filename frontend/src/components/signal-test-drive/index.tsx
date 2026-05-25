import { create } from "@bufbuild/protobuf"
import { For, onMount, Show } from "solid-js"
import { createStore } from "solid-js/store"
import {
  type ListSignalsResponse,
  ListSignalsResponseSchema,
  type Signal,
  SignalUpdateType,
  type Stage,
} from "~/api/signals/v1/signals_pb"
import { signalsClient } from "~/lib/rpc"
import { Button } from "../ui/button"
import { clearSignalCache, readSignalCache, writeSignalCache } from "./cache"
import { LoadingStream, SignalCard, StageCard } from "./cards"

type SignalTestDriveState = ListSignalsResponse & {
  loading: boolean
  loaded: boolean
  error: string
}

const createInitialState = (): SignalTestDriveState => ({
  ...create(ListSignalsResponseSchema),
  loading: false,
  loaded: false,
  error: "",
})

export const SignalTestDrive = () => {
  const [data, setData] = createStore<SignalTestDriveState>(
    createInitialState(),
  )

  const stages = (): Stage[] =>
    [data.eventIngest, data.marketIngest, data.aiJudgment].filter(
      (stage): stage is Stage => Boolean(stage),
    )

  const snapshot = (signals: Signal[] = data.signals): ListSignalsResponse =>
    create(ListSignalsResponseSchema, {
      eventIngest: data.eventIngest,
      marketIngest: data.marketIngest,
      aiJudgment: data.aiJudgment,
      summary: data.summary,
      failureReason: data.failureReason,
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

  onMount(hydrateCache)

  const persist = (signals?: Signal[]) => writeSignalCache(snapshot(signals))

  const loadSignals = async () => {
    setData({ ...createInitialState(), loading: true })
    clearSignalCache()

    try {
      for await (const update of signalsClient.streamSignals({})) {
        let signals: Signal[] | undefined

        if (update.eventIngest) setData("eventIngest", update.eventIngest)
        if (update.marketIngest) setData("marketIngest", update.marketIngest)
        if (update.aiJudgment) setData("aiJudgment", update.aiJudgment)
        if (update.signal) signals = upsertSignal(update.signal)
        if (update.summary) setData("summary", update.summary)
        if (update.failureReason) setData("failureReason", update.failureReason)

        if (
          update.eventIngest ||
          update.marketIngest ||
          update.aiJudgment ||
          update.signal ||
          update.summary ||
          update.failureReason
        ) {
          persist(signals)
        }

        if (update.type === SignalUpdateType.DONE || update.done) {
          setData({ loading: false, loaded: true })
          persist(signals)
        }
      }

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

  return (
    <section id="signal-test-drive" class="flex flex-col gap-3 bento-cell">
      <div class="flex flex-row gap-2">
        <Button
          onClick={loadSignals}
          disabled={data.loading}
          class="w-min uppercase whitespace-nowrap"
        >
          {data.loading
            ? "streaming..."
            : data.loaded
              ? "refresh test drive"
              : "load test drive"}
        </Button>
      </div>

      <Show when={data.loading && data.signals.length === 0}>
        <LoadingStream />
      </Show>

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
            <For each={stages()}>{(stage) => <StageCard stage={stage} />}</For>
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
