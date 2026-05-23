import { animate, utils } from "animejs"
import { createClient } from "@connectrpc/connect"
import { createConnectTransport } from "@connectrpc/connect-web"
import { batch, onCleanup, onMount, Show } from "solid-js"
import { createStore } from "solid-js/store"
import { FlightService } from "~/api/flight/v1/flight_pb"

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

export const FlightDiagnostics = () => {
  const [data, setData] = createStore({
    cpu: 0,
    frontendMemBytes: null as number | null,
    backendMemBytes: 0,
    rtt: 0,
    loading: false,
    error: "",
  })
  let progressRef: HTMLSpanElement | undefined
  let metricsRef: HTMLDivElement | undefined
  let intervalId: ReturnType<typeof setInterval> | undefined

  const animateRefresh = () => {
    const reduceMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches
    if (progressRef) {
      animate(progressRef, {
        scaleX: [1, 0],
        duration: 10000,
        ease: "linear",
      })
    }
    if (reduceMotion) return
    if (metricsRef) {
      animate(metricsRef.querySelectorAll("[data-metric]"), {
        translateY: [6, 0],
        opacity: [0.45, 1],
        delay: utils.stagger(35),
        duration: 280,
        ease: "outQuint",
      })
    }
  }

  const probe = async () => {
    if (data.loading) return
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
          error: "",
        })
      })
      animateRefresh()
    } catch (err) {
      setData({
        loading: false,
        error: err instanceof Error ? err.message : "Probe failed",
      })
      console.error("Probe failed:", err)
    }
  }

  onMount(() => {
    void probe()
    intervalId = setInterval(() => {
      void probe()
    }, 10000)
  })

  onCleanup(() => {
    if (intervalId) clearInterval(intervalId)
  })

  return (
    <div class="flex flex-col gap-2 p-2 rounded w-fit bg-accent/10">
      <div class="flex flex-col gap-2">
        <div class="flex gap-3 justify-between items-center opacity-70">
          <span class="text-xs uppercase">System health</span>
          <Show when={data.loading}>
            <span class="font-mono text-xs opacity-60">Refreshing</span>
          </Show>
        </div>
        <div class="overflow-hidden w-full h-1 rounded bg-current/10">
          <span ref={progressRef} class="block h-full origin-left bg-primary" />
        </div>
      </div>
      <div ref={metricsRef} class="flex gap-x-6 font-mono text-sm">
        <div>
          <span>STATS</span>
          <div data-metric class="flex gap-2">
            <span class="opacity-60">latency</span>
            <span>{data.rtt}ms</span>
          </div>
          <div data-metric class="flex gap-2">
            <span class="opacity-60">back cpu</span>
            <span>{data.cpu.toFixed(1)}%</span>
          </div>
        </div>
        <div>
          <span>MEM</span>
          <div data-metric class="flex gap-2">
            <span class="opacity-60">front</span>
            <span>{formatBytes(data.frontendMemBytes)}</span>
          </div>
          <div data-metric class="flex gap-2">
            <span class="opacity-60">back</span>
            <span>{formatBytes(data.backendMemBytes)}</span>
          </div>
        </div>
      </div>
      <Show when={data.error}>
        {(error) => (
          <div class="font-mono text-red-600 dark:text-red-400">{error()}</div>
        )}
      </Show>
    </div>
  )
}