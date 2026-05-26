import { create } from "@bufbuild/protobuf"
import type { Timestamp } from "@bufbuild/protobuf/wkt"
import {
  PipelineStagesSchema,
  SignalHuntStateSchema,
  SignalHuntSummarySchema,
  SignalSchema,
  StageSchema,
  type PipelineStages,
  type Signal,
  type SignalHuntState,
  type SignalHuntSummary,
  type Stage,
} from "~/api/signals/v1/signals_pb"

const CACHE_KEY = "iridium-edge.signal-test-drive.v1"
const CACHE_TTL_MS = 60 * 60 * 1000
const COOLDOWN_KEY = "iridium-edge.signal-test-drive.cooldown.v1"
export const SIGNAL_HUNT_COOLDOWN_MS = 30 * 1000

type CachedTimestamp = {
  seconds: string
  nanos: number
}

type CachedStage = Pick<Stage, "status" | "label" | "detail">

type CachedSignal = Omit<Signal, "publishedAt" | "aiJudgment"> & {
  publishedAt?: CachedTimestamp
  aiJudgment?: CachedStage
}

type CachedSummary = Pick<SignalHuntSummary, "text" | "failureReason">

type SignalCache = {
  savedAt: number
  stages?: {
    eventIngest?: CachedStage
    marketIngest?: CachedStage
    aiJudgment?: CachedStage
  }
  summary?: CachedSummary
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

const cacheStages = (stages?: PipelineStages) =>
  stages
    ? {
        eventIngest: cacheStage(stages.eventIngest),
        marketIngest: cacheStage(stages.marketIngest),
        aiJudgment: cacheStage(stages.aiJudgment),
      }
    : undefined

const hydrateStages = (stages?: SignalCache["stages"]): PipelineStages | undefined =>
  stages
    ? create(PipelineStagesSchema, {
        eventIngest: hydrateStage(stages.eventIngest),
        marketIngest: hydrateStage(stages.marketIngest),
        aiJudgment: hydrateStage(stages.aiJudgment),
      })
    : undefined

const cacheSummary = (summary?: SignalHuntSummary): CachedSummary | undefined =>
  summary
    ? {
        text: summary.text,
        failureReason: summary.failureReason,
      }
    : undefined

const hydrateSummary = (summary?: CachedSummary): SignalHuntSummary | undefined =>
  summary ? create(SignalHuntSummarySchema, summary) : undefined

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

export const writeSignalCache = (state: SignalHuntState): void => {
  if (!cacheAvailable()) return
  const cache: SignalCache = {
    savedAt: Date.now(),
    stages: cacheStages(state.stages),
    summary: cacheSummary(state.summary),
    signals: state.signals.map(cacheSignal),
  }
  window.sessionStorage.setItem(CACHE_KEY, JSON.stringify(cache))
}

export const clearSignalCache = (): void => {
  if (!cacheAvailable()) return
  window.sessionStorage.removeItem(CACHE_KEY)
}

export const writeSignalCooldown = (cooldownUntil: number): void => {
  if (!cacheAvailable()) return
  window.localStorage.setItem(COOLDOWN_KEY, String(cooldownUntil))
}

export const readSignalCooldown = (): number => {
  if (!cacheAvailable()) return 0
  const rawCooldown = window.localStorage.getItem(COOLDOWN_KEY)
  if (!rawCooldown) return 0
  const cooldownUntil = Number(rawCooldown)
  return Number.isFinite(cooldownUntil) ? cooldownUntil : 0
}

export const readSignalCache = (): SignalHuntState | undefined => {
  if (!cacheAvailable()) return undefined
  const rawCache = window.sessionStorage.getItem(CACHE_KEY)
  if (!rawCache) return undefined

  try {
    const cache = JSON.parse(rawCache) as SignalCache
    if (Date.now() - cache.savedAt > CACHE_TTL_MS) {
      clearSignalCache()
      return undefined
    }
    return create(SignalHuntStateSchema, {
      stages: hydrateStages(cache.stages),
      summary: hydrateSummary(cache.summary),
      signals: cache.signals.map(hydrateSignal),
    })
  } catch (err) {
    clearSignalCache()
    console.error("Signal cache hydration failed:", err)
    return undefined
  }
}