import { createClient } from "@connectrpc/connect"
import { createConnectTransport } from "@connectrpc/connect-web"
import { For, Show } from "solid-js"
import { createStore } from "solid-js/store"
import { FlightService } from "~/api/flight/v1/flight_pb"
import { Button } from "./ui/button"

type PublishedAt = { seconds?: bigint | number; nanos?: number }
type SignalItem = {
  headline: string
  source: string
  sourceUrl: string
  publishedAt?: PublishedAt
  marketQuestion: string
  marketUrl: string
  whyItMatters: string
  score: number
}

const getApiBaseUrl = (): string => {
  const envBaseUrl = import.meta.env.VITE_API_BASE_URL?.trim()
  if (envBaseUrl) return envBaseUrl
  if (!import.meta.env.SSR && typeof window !== "undefined") {
    if (import.meta.env.DEV) {
      return `${window.location.protocol}//${window.location.hostname}:8080`
    }
    return window.location.origin
  }
  return "http://127.0.0.1:8080"
}

const transport = createConnectTransport({ baseUrl: getApiBaseUrl() })
const client = createClient(FlightService, transport)

const formatPublishedAt = (publishedAt?: PublishedAt): string => {
  if (!publishedAt?.seconds) return "unknown"
  return new Date(Number(publishedAt.seconds) * 1000).toLocaleString()
}

export const SignalTestDrive = () => {
  const [data, setData] = createStore({
    signalsLoading: false,
    signals: [] as SignalItem[],
    error: "",
  })

  const loadSignals = async () => {
    setData({ signalsLoading: true, error: "" })

    try {
      const response = await client.listSignals({})
      setData({
        signalsLoading: false,
        signals: response.signals,
      })
    } catch (err) {
      setData({
        signalsLoading: false,
        error: err instanceof Error ? err.message : "Signal fetch failed",
      })
      console.error("Signal fetch failed:", err)
    }
  }

  return (
    <section id="signal-test-drive" class="flex flex-col gap-3 bento-cell">
      <div class="flex flex-row gap-2">
        <Button
          onClick={loadSignals}
          disabled={data.signalsLoading}
          class="w-min whitespace-nowrap"
        >
          {data.signalsLoading ? "LOADING..." : "LOAD TEST DRIVE"}
        </Button>
      </div>
      <Show when={data.signals.length > 0}>
        <div class="flex flex-col gap-3">
          <For each={data.signals}>
            {(signal) => (
              <a
                class="flex flex-col gap-2 rounded border border-muted p-3 transition-all hover:border-secondary"
                href={signal.marketUrl || signal.sourceUrl}
                target="_blank"
                rel="noreferrer"
              >
                <div class="flex flex-col gap-1">
                  <div class="text-muted-foreground">
                    {signal.source} · score {signal.score.toFixed(1)}
                  </div>
                  <div class="font-medium">{signal.headline}</div>
                  <div class="text-sm opacity-80">
                    {formatPublishedAt(signal.publishedAt)}
                  </div>
                </div>
                <div class="text-sm">{signal.whyItMatters}</div>
                <div class="font-mono text-sm opacity-80">
                  {signal.marketQuestion}
                </div>
              </a>
            )}
          </For>
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
