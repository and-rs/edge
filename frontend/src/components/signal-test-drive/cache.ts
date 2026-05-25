import { create } from "@bufbuild/protobuf"
import type { Timestamp } from "@bufbuild/protobuf/wkt"
import {
  ListSignalsResponseSchema,
  SignalSchema,
  StageSchema,
  type ListSignalsResponse,
  type Signal,
  type Stage,
} from "~/api/signals/v1/signals_pb"

const CACHE_KEY = "iridium-edge.signal-test-drive.v1"
const CACHE_TTL_MS = 60 * 60 * 1000

type CachedTimestamp = {
  seconds: string
  nanos: number
}

type CachedStage = Pick<Stage, "status" | "label" | "detail">

type CachedSignal = Omit<Signal, "publishedAt" | "aiJudgment"> & {
  publishedAt?: CachedTimestamp
  aiJudgment?: CachedStage
}

type SignalCache = Pick<
  ListSignalsResponse,
  "summary" | "failureReason"
> & {
  savedAt: number
  eventIngest?: CachedStage
  marketIngest?: CachedStage
  aiJudgment?: CachedStage
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
  stage ? create(StageSchema, stage) : undefined

const cacheSignal = (signal: Signal): CachedSignal => ({
  ...signal,
  publishedAt: cacheTimestamp(signal.publishedAt),
  aiJudgment: cacheStage(signal.aiJudgment),
})

const hydrateSignal = (signal: CachedSignal): Signal =>
  create(SignalSchema, {
    ...signal,
    publishedAt: hydrateTimestamp(signal.publishedAt),
    aiJudgment: hydrateStage(signal.aiJudgment),
  })

export const writeSignalCache = (response: ListSignalsResponse): void => {
  if (!cacheAvailable()) return
  const cache: SignalCache = {
    savedAt: Date.now(),
    eventIngest: cacheStage(response.eventIngest),
    marketIngest: cacheStage(response.marketIngest),
    aiJudgment: cacheStage(response.aiJudgment),
    summary: response.summary,
    failureReason: response.failureReason,
    signals: response.signals.map(cacheSignal),
  }
  window.sessionStorage.setItem(CACHE_KEY, JSON.stringify(cache))
}

export const clearSignalCache = (): void => {
  if (!cacheAvailable()) return
  window.sessionStorage.removeItem(CACHE_KEY)
}

export const readSignalCache = (): ListSignalsResponse | undefined => {
  if (!cacheAvailable()) return undefined
  const rawCache = window.sessionStorage.getItem(CACHE_KEY)
  if (!rawCache) return undefined

  try {
    const cache = JSON.parse(rawCache) as SignalCache
    if (Date.now() - cache.savedAt > CACHE_TTL_MS) {
      clearSignalCache()
      return undefined
    }
    return create(ListSignalsResponseSchema, {
      eventIngest: hydrateStage(cache.eventIngest),
      marketIngest: hydrateStage(cache.marketIngest),
      aiJudgment: hydrateStage(cache.aiJudgment),
      summary: cache.summary,
      failureReason: cache.failureReason,
      signals: cache.signals.map(hydrateSignal),
    })
  } catch (err) {
    clearSignalCache()
    console.error("Signal cache hydration failed:", err)
    return undefined
  }
}
