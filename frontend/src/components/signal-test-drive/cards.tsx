import type { Timestamp } from "@bufbuild/protobuf/wkt"
import { Show } from "solid-js"
import type { Signal, Stage } from "~/api/signals/v1/signals_pb"
import {
  matchToneClass,
  revealClass,
  stageToneClass,
  streamPulseClass,
} from "./styles"

const formatPublishedAt = (publishedAt?: Timestamp): string => {
  if (!publishedAt?.seconds) return "unknown"
  return new Date(Number(publishedAt.seconds) * 1000).toLocaleString()
}

export const LoadingStream = () => (
  <div class={`grid gap-2 md:grid-cols-3 ${streamPulseClass}`}>
    <div class="p-3 text-xs uppercase border border-muted tracking-[0.24em] text-secondary">
      finding events
    </div>
    <div class="p-3 text-xs uppercase border border-muted tracking-[0.24em] text-muted-foreground">
      linking markets
    </div>
    <div class="p-3 text-xs uppercase border border-muted tracking-[0.24em] text-muted-foreground">
      revealing edge
    </div>
  </div>
)

export const StageCard = (props: { stage: Stage }) => (
  <div
    class={`flex flex-col gap-1 border border-muted p-3 ${stageToneClass(props.stage)} ${revealClass}`}
  >
    <div class="text-xs uppercase opacity-70 tracking-[0.24em]">
      {props.stage.label}
    </div>
    <div class="text-sm opacity-80">{props.stage.detail}</div>
  </div>
)

export const SignalCard = (props: { signal: Signal }) => (
  <a
    class={`flex flex-col gap-3 rounded border border-muted p-3 transition-all hover:border-secondary ${matchToneClass(props.signal.finalMatchType)} ${revealClass}`}
    href={props.signal.marketUrl || props.signal.sourceUrl}
    target="_blank"
    rel="noreferrer"
  >
    <div class="flex flex-col gap-1">
      <div class="text-muted-foreground">
        {props.signal.linkStateLabel} · score {props.signal.score.toFixed(1)}
      </div>
      <div class="font-medium">{props.signal.headline}</div>
      <div class="text-sm opacity-80">
        {formatPublishedAt(props.signal.publishedAt)}
      </div>
    </div>

    <div class="text-sm">{props.signal.thesis}</div>
    <div class="text-sm opacity-80">{props.signal.whyItMatters}</div>

    <div class="grid gap-2 text-xs opacity-80 md:grid-cols-2">
      <div>market: {props.signal.marketQuestion}</div>
      <div>ai: {props.signal.aiJudgment?.label}</div>
    </div>
    <Show when={props.signal.failureReason}>
      {(reason) => (
        <div class="font-mono text-xs text-amber-600 dark:text-amber-400">
          {reason()}
        </div>
      )}
    </Show>
  </a>
)
