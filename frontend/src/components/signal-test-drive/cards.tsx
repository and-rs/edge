import type { Timestamp } from "@bufbuild/protobuf/wkt"
import CalendarDays from "lucide-solid/icons/calendar-days"
import ExternalLink from "lucide-solid/icons/external-link"
import Newspaper from "lucide-solid/icons/newspaper"
import Sparkles from "lucide-solid/icons/sparkles"
import LoaderCircle from "lucide-solid/icons/loader-circle"
import { Show } from "solid-js"
import { StageStatus, type Signal, type Stage } from "~/api/signals/v1/signals_pb"
import { Button } from "../ui/button"
import {
  matchToneClass,
  revealClass,
  stageToneClass,
  streamPulseClass,
} from "./styles"

const badgeClass =
  "border border-secondary/50 bg-secondary px-2 py-1 text-xs font-semibold uppercase tracking-[0.18em] text-secondary-foreground"
const actionButtonClass =
  "border border-secondary/50 bg-secondary font-semibold uppercase tracking-[0.18em] text-secondary-foreground hover:bg-secondary/90"

const formatPublishedAt = (publishedAt?: Timestamp): string => {
  if (!publishedAt?.seconds) return "unknown"
  return new Date(Number(publishedAt.seconds) * 1000).toLocaleString()
}

const formatVolume = (volume: number): string => {
  if (!volume) return "unknown"
  return new Intl.NumberFormat("en", {
    notation: "compact",
    maximumFractionDigits: 1,
  }).format(volume)
}

const SectionLabel = (props: { children: string }) => (
  <div class="text-[0.65rem] font-medium uppercase tracking-[0.24em] text-muted-foreground">
    {props.children}
  </div>
)

const AILoader = () => (
  <div class={`grid gap-3 ${streamPulseClass}`}>
    <div class="flex items-center gap-2 text-[0.65rem] font-medium uppercase tracking-[0.24em] text-secondary">
      <LoaderCircle class="size-3.5 animate-spin" />
      <span>AI judgment loading</span>
    </div>
    <div class="h-4 w-[82%] bg-muted" />
    <div class="h-4 w-full bg-muted/80" />
    <div class="h-4 w-[74%] bg-muted/70" />
  </div>
)

export const LoadingStream = () => (
  <div class={`grid gap-2 md:grid-cols-3 ${streamPulseClass}`}>
    <div class="flex items-center gap-2 border border-muted p-3 text-xs uppercase tracking-[0.24em] text-secondary">
      <LoaderCircle class="size-3.5 animate-spin" />
      <span>finding events</span>
    </div>
    <div class="flex items-center gap-2 border border-muted p-3 text-xs uppercase tracking-[0.24em] text-muted-foreground">
      <LoaderCircle class="size-3.5 animate-spin" />
      <span>linking markets</span>
    </div>
    <div class="flex items-center gap-2 border border-muted p-3 text-xs uppercase tracking-[0.24em] text-muted-foreground">
      <LoaderCircle class="size-3.5 animate-spin" />
      <span>revealing edge</span>
    </div>
  </div>
)

export const StageCard = (props: { stage: Stage }) => (
  <div
    class={`flex flex-col gap-2 border border-muted p-3 ${stageToneClass(props.stage)} ${revealClass}`}
  >
    <div class="flex items-center gap-2 text-xs uppercase opacity-70 tracking-[0.24em]">
      <Show when={props.stage.status === StageStatus.RUNNING}>
        <LoaderCircle class="size-3.5 animate-spin" />
      </Show>
      <span>{props.stage.label}</span>
    </div>
    <div class="text-sm opacity-80">{props.stage.detail}</div>
  </div>
)

export const SignalCard = (props: { signal: Signal }) => {
  const hasAISummary = () =>
    Boolean(
      props.signal.aiJudgment?.status === StageStatus.READY &&
        props.signal.whyItMatters.trim(),
    )
  const aiSummary = () => (hasAISummary() ? props.signal.whyItMatters : "")
  const isAIPending = () => props.signal.aiJudgment?.status === StageStatus.RUNNING
  const marketQuestion = () => props.signal.marketQuestion.trim()
  const hasMarket = () =>
    Boolean(
      props.signal.marketUrl &&
        marketQuestion() &&
        props.signal.marketVenue &&
        !props.signal.linkStateLabel.toLowerCase().includes("no live"),
    )
  const noisyMarketQuestion = () => {
    const question = marketQuestion()
    return question.includes(",") || question.toLowerCase().startsWith("yes ")
  }
  const marketSummary = () =>
    noisyMarketQuestion()
      ? "Live Kalshi market found for this signal. Open market for exact contract details."
      : marketQuestion()

  return (
    <article
      class={`border border-muted bg-card shadow-sm shadow-black/5 transition-all hover:border-secondary/70 ${matchToneClass(props.signal.finalMatchType)} ${revealClass}`}
    >
      <div class="grid gap-5 p-4 md:p-5">
        <div class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
          <div class="min-w-0 space-y-3">
            <div class="flex flex-wrap items-center gap-2 text-xs">
              <span class={badgeClass}>{props.signal.linkStateLabel}</span>
              <span class="text-muted-foreground">
                Edge score {props.signal.score.toFixed(1)}
              </span>
            </div>
            <h3 class="max-w-4xl text-xl font-semibold leading-tight tracking-tight text-foreground">
              {props.signal.headline}
            </h3>
          </div>

          <Button
            as="a"
            href={props.signal.sourceUrl}
            target="_blank"
            rel="noreferrer"
            size="sm"
            class={`shrink-0 whitespace-nowrap ${actionButtonClass}`}
          >
            Read source
          </Button>
        </div>

        <Show when={hasAISummary() || isAIPending()}>
          <section class="grid gap-3 border-t border-muted pt-5">
            <div class="flex items-center gap-2 text-[0.65rem] font-medium uppercase tracking-[0.24em] text-muted-foreground">
              <Sparkles class="size-3.5" />
              <span>Why this matters</span>
            </div>
            <Show when={hasAISummary()} fallback={<AILoader />}>
              <p class="max-w-4xl text-lg leading-8 text-foreground/95">
                {aiSummary()}
              </p>
            </Show>
          </section>
        </Show>

        <div class="flex flex-wrap items-center gap-x-5 gap-y-2 border-t border-muted pt-4 text-sm">
          <div class="inline-flex items-center gap-2 text-foreground">
            <Newspaper class="size-4 text-muted-foreground" />
            <span class="font-medium">{props.signal.eventSource}</span>
          </div>
          <div class="inline-flex items-center gap-2 text-muted-foreground">
            <CalendarDays class="size-4" />
            <span>{formatPublishedAt(props.signal.publishedAt)}</span>
          </div>
          <a
            class="inline-flex items-center gap-2 text-secondary underline-offset-4 hover:underline"
            href={props.signal.sourceUrl}
            target="_blank"
            rel="noreferrer"
          >
            <ExternalLink class="size-4" />
            <span>Original report</span>
          </a>
        </div>

        <Show when={hasMarket()}>
          <section class="grid gap-3 border-t border-muted bg-background px-4 py-4 md:grid-cols-[minmax(0,1fr)_auto] md:items-start">
            <div class="min-w-0">
              <SectionLabel>Matched market</SectionLabel>
              <div class="mt-2 max-w-3xl text-base leading-7 text-foreground">
                {marketSummary()}
              </div>
              <div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-muted-foreground">
                <span>{props.signal.marketVenue}</span>
                <Show when={props.signal.marketStatus}>
                  {(status) => <span>{status()}</span>}
                </Show>
                <Show when={props.signal.marketVolume24h}>
                  <span>24h volume {formatVolume(props.signal.marketVolume24h)}</span>
                </Show>
              </div>
            </div>
            <Button
              as="a"
              href={props.signal.marketUrl}
              target="_blank"
              rel="noreferrer"
              size="sm"
              class={`whitespace-nowrap ${actionButtonClass}`}
            >
              Open market
            </Button>
          </section>
        </Show>
      </div>
    </article>
  )
}
