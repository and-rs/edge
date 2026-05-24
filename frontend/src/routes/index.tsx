import { animate, stagger } from "animejs"
import ChevronsRight from "lucide-solid/icons/chevrons-right"
import { For, onMount } from "solid-js"
import { Brand } from "~/components/brand"
import { SignalTestDrive } from "~/components/signal-test-drive"
import { PowerButton } from "~/components/ui/power-button"

const audience = ["Operators", "Desks", "Research"]
const power = ["Sources", "Markets", "AI", "ARC USDC"]
const coreTech = [
  "Go",
  "SolidStart",
  "Connect-RPC",
  "Postgres",
  "OpenAI",
  "Arc USDC",
]

const HomeRoute = () => {
  let heroRef: HTMLElement | undefined
  let subRef: HTMLParagraphElement | undefined
  let cardOneRef: HTMLDivElement | undefined
  let cardTwoRef: HTMLDivElement | undefined
  let cardThreeRef: HTMLDivElement | undefined
  let endgameRef: HTMLElement | undefined

  const animateCardOne = () => {
    if (!cardOneRef) return
    animate(cardOneRef.querySelectorAll("[data-word]"), {
      translateY: [14, 0],
      opacity: [0.35, 1],
      delay: stagger(55),
      duration: 420,
      ease: "outQuint",
    })
  }

  const animateCardTwo = () => {
    if (!cardTwoRef) return
    const chips = cardTwoRef.querySelectorAll("[data-chip]")
    animate(chips, {
      translateX: [12, 0],
      opacity: [0.35, 1],
      delay: stagger(70),
      duration: 360,
      ease: "outQuint",
    })
  }

  const animateCardThree = () => {
    if (!cardThreeRef) return
    animate(cardThreeRef.querySelectorAll("[data-power]"), {
      scale: [0.92, 1],
      opacity: [0.35, 1],
      delay: stagger(65),
      duration: 360,
      ease: "outQuint",
    })
  }

  const animateEndgame = () => {
    if (!endgameRef) return
    animate(endgameRef.querySelectorAll("[data-core-tech]"), {
      translateY: [10, 0],
      scale: [0.96, 1],
      opacity: [0.35, 1],
      delay: stagger(55),
      duration: 360,
      ease: "outQuint",
    })
  }

  const jumpToSignalTest = (event: MouseEvent) => {
    event.preventDefault()
    const signalTest = document.getElementById("signal-test-drive")
    if (!signalTest) return
    window.history.replaceState(null, "", "#signal-test-drive")
    signalTest.scrollIntoView({ behavior: "smooth", block: "start" })
  }

  onMount(() => {
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return
    if (!heroRef || !subRef || !endgameRef) return

    animate(heroRef, {
      translateY: [18, 0],
      opacity: [0, 1],
      duration: 500,
      ease: "outQuint",
    })
    animate(subRef, {
      translateY: [12, 0],
      opacity: [0, 1],
      delay: 80,
      duration: 420,
      ease: "outQuint",
    })
    animate([cardOneRef, cardTwoRef, cardThreeRef].filter(Boolean), {
      translateY: [20, 0],
      opacity: [0, 1],
      delay: stagger(80, { start: 140 }),
      duration: 420,
      ease: "outQuint",
    })
    animate(endgameRef, {
      translateY: [18, 0],
      opacity: [0, 1],
      delay: 300,
      duration: 420,
      ease: "outQuint",
    })

    animateCardOne()
    setTimeout(animateCardTwo, 120)
    setTimeout(animateCardThree, 220)
    setTimeout(animateEndgame, 320)
  })

  return (
    <main class="px-4 pt-6 pb-3">
      <div class="flex flex-col gap-2">
        <section ref={heroRef} class="flex flex-col gap-8 py-8 bento-cell">
          <Brand size="lg" text="Iridium Edge" logoSrc="/iridium-edge.png" />
          <div class="flex flex-col gap-4 max-w-3xl">
            <p class="text-2xl leading-tight md:text-3xl">
              Event-to-market intelligence for crypto and prediction-market
              operators.
            </p>
            <p ref={subRef} class="text-base opacity-80 md:text-lg">
              Raw events in. Ranked market signals out. Premium access in USDC.
            </p>

            <PowerButton
              as="a"
              href="#signal-test-drive"
              class="w-min"
              icon={<ChevronsRight class="size-7" strokeWidth={1.5} />}
              onClick={jumpToSignalTest}
            >
              open signal test
            </PowerButton>
          </div>
        </section>

        <section class="grid gap-2 md:grid-cols-3">
          <div
            ref={cardOneRef}
            class="flex flex-col gap-6 justify-between bento-cell min-h-56"
            onMouseEnter={animateCardOne}
          >
            <div class="text-sm uppercase opacity-70">What it is</div>
            <div class="flex flex-col gap-2 text-xl font-semibold md:text-4xl text-primary">
              <span data-word>Signal</span>
              <span data-word>Engine</span>
              <span data-word class="opacity-55">
                for events
              </span>
            </div>
          </div>

          <div
            ref={cardTwoRef}
            class="flex flex-col gap-6 justify-between bento-cell min-h-56"
            onMouseEnter={animateCardTwo}
          >
            <div class="flex gap-3 justify-between items-center">
              <div class="text-sm uppercase opacity-70">Who it is for</div>
            </div>
            <div class="flex flex-wrap gap-2">
              <For each={audience}>
                {(item) => (
                  <span data-chip class="badge-outline">
                    {item}
                  </span>
                )}
              </For>
            </div>
            <div class="text-xl leading-tight md:text-2xl">
              Faster event triage for market operators.
            </div>
          </div>

          <div
            ref={cardThreeRef}
            class="flex flex-col gap-6 justify-between bento-cell min-h-56"
            onMouseEnter={animateCardThree}
          >
            <div class="text-sm uppercase opacity-70">What powers it</div>
            <div class="flex flex-wrap gap-2">
              <For each={power}>
                {(item) => (
                  <span data-power class="badge-muted">
                    {item}
                  </span>
                )}
              </For>
            </div>
            <div class="text-xl leading-tight md:text-2xl">
              Sources, markets, AI, payments.
            </div>
          </div>
        </section>

        <section
          ref={endgameRef}
          class="flex flex-col gap-4 bento-cell"
          onMouseEnter={animateEndgame}
        >
          <div class="text-sm uppercase opacity-70">Endgame</div>
          <h3>Decision software for event-driven markets.</h3>
          <p class="max-w-3xl opacity-85">
            Ingest cross-market events, map them to live contracts, rank what
            matters, and deliver operator-ready theses through premium USDC
            access. With a performance & value oriented stack.
          </p>
          <div class="flex flex-wrap gap-2">
            <For each={coreTech}>
              {(item) => (
                <span data-core-tech class="badge-outline">
                  {item}
                </span>
              )}
            </For>
          </div>
        </section>

        <SignalTestDrive />
      </div>
    </main>
  )
}

export default HomeRoute
