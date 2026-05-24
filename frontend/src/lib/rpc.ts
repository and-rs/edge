import { createClient } from "@connectrpc/connect"
import { createConnectTransport } from "@connectrpc/connect-web"
import { DiagnosticsService } from "~/api/diagnostics/v1/diagnostics_pb"
import { SignalsService } from "~/api/signals/v1/signals_pb"

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

export const diagnosticsClient = createClient(DiagnosticsService, transport)
export const signalsClient = createClient(SignalsService, transport)
