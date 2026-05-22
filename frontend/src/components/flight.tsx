import { createClient } from "@connectrpc/connect"
import { createConnectTransport } from "@connectrpc/connect-web"
import { batch } from "solid-js"
import { createStore } from "solid-js/store"
import { FlightService } from "~/api/flight/v1/flight_pb"
import { Button } from "./ui/button"

const getApiBaseUrl = (): string => {
  const envBaseUrl = import.meta.env.VITE_API_BASE_URL?.trim()
  if (envBaseUrl) return envBaseUrl
  if (!import.meta.env.SSR && typeof window !== "undefined") {
    if (import.meta.env.DEV)
      return `${window.location.protocol}//${window.location.hostname}:8080`
    return window.location.origin
  }
  return "http://127.0.0.1:8080"
}

const transport = createConnectTransport({ baseUrl: getApiBaseUrl() })

const client = createClient(FlightService, transport)

export const FlightDiagnostics = () => {
  const [data, setData] = createStore({
    cpu: 0,
    frontendMemBytes: null as number | null,
    backendMemBytes: 0,
    rtt: 0,
    loading: false,
    signalsLoading: false,
    signals: [] as Array<{
      headline: string
      source: string
      sourceUrl: string
      publishedAt?: { seconds?: bigint | number; nanos?: number }
      marketQuestion: string
      marketUrl: string
      whyItMatters: string
      score: number
    }>,
    error: "",
  })

  const getFrontendMemoryBytes = (): number | null => {
    const performanceWithMemory = performance as Performance & {
      memory?: { usedJSHeapSize: number }
    }
    return performanceWithMemory.memory?.usedJSHeapSize ?? null
  }

  const formatBytes = (bytes: number | null): string => {
    if (bytes === null) return "n/a"
    if (bytes < 1024) return `${bytes} B`
    const units = ["KB", "MB", "GB", "TB"]
    let value = bytes / 1024
    let unitIndex = 0
    while (value >= 1024 && unitIndex < units.length - 1) {
      value /= 1024
      unitIndex += 1
    }
    return `${value.toFixed(1)} ${units[unitIndex]}`
  }

  const formatPublishedAt = (publishedAt?: { seconds?: bigint | number; nanos?: number }): string => {
    if (!publishedAt?.seconds) return "unknown"
    const seconds = Number(publishedAt.seconds)
    return new Date(seconds * 1000).toLocaleString()
  }

  const probe = async () => {
    setData({ loading: true, error: "" })
    const start = performance.now()

    const now = Date.now()

    try {
      const response = await client.probe({
        clientSentAt: {
          seconds: BigInt(Math.floor(now / 1000)),
          nanos: (now % 1000) * 1e6,
        },
      })

      batch(() => {
        setData({
          cpu: response.cpuPercent,
          frontendMemBytes: getFrontendMemoryBytes(),
          backendMemBytes: Number(response.backendMemoryBytes),
          rtt: Math.floor(performance.now() - start),
          loading: false,
        })
      })
    } catch (err) {
      setData({
        loading: false,
        error: err instanceof Error ? err.message : "Probe failed",
      })
      console.error("Probe failed:", err)
    }
  }

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
    <div class="flex flex-col gap-3 bento-cell">
      <div class="flex flex-row gap-2">
        <Button
          onClick={probe}
          disabled={data.loading}
          class="w-min whitespace-nowrap"
        >
          {data.loading ? "PROBING..." : "RUN DIAGNOSTIC"}
        </Button>
        <Button
          onClick={loadSignals}
          disabled={data.signalsLoading}
          class="w-min whitespace-nowrap"
        >
          {data.signalsLoading ? "LOADING..." : "LOAD TEST DRIVE"}
        </Button>
      </div>
      <div class="flex flex-row gap-6">
        <div>
          <div class="text-xs uppercase">Latency</div>
          <div class="font-mono text-lg">{data.rtt}ms</div>
        </div>
        <div>
          <div class="text-xs uppercase">CPU Usage</div>
          <div class="font-mono text-lg">{data.cpu.toFixed(1)}%</div>
        </div>
        <div>
          <div class="text-xs uppercase">Frontend Memory</div>
          <div class="font-mono text-lg">
            {formatBytes(data.frontendMemBytes)}
          </div>
        </div>
        <div>
          <div class="text-xs uppercase">Backend Memory</div>
          <div class="font-mono text-lg">
            {formatBytes(data.backendMemBytes)}
          </div>
        </div>
      </div>
      <div class="flex flex-col gap-3">
        {data.signals.map((signal) => (
          <a
            class="flex flex-col gap-2 rounded border border-muted p-3 hover:border-secondary transition-all"
            href={signal.marketUrl || signal.sourceUrl}
            target="_blank"
            rel="noreferrer"
          >
            <div class="flex flex-col gap-1">
              <div class="text-sm text-muted-foreground uppercase">
                {signal.source} · score {signal.score.toFixed(1)}
              </div>
              <div class="font-medium">{signal.headline}</div>
              <div class="text-sm opacity-80">{formatPublishedAt(signal.publishedAt)}</div>
            </div>
            <div class="text-sm">{signal.whyItMatters}</div>
            <div class="text-sm font-mono opacity-80">{signal.marketQuestion}</div>
          </a>
        ))}
      </div>
      {data.error ? (
        <div class="font-mono text-sm text-red-600 dark:text-red-400">
          {data.error}
        </div>
      ) : null}
    </div>
  )
}
