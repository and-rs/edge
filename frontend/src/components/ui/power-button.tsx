import type { ButtonRootProps } from "@kobalte/core/button"
import type { PolymorphicProps } from "@kobalte/core/polymorphic"
import { animate } from "animejs"
import {
  type JSX,
  onCleanup,
  onMount,
  splitProps,
  type ValidComponent,
} from "solid-js"
import { Button } from "./button"

type PowerButtonProps<T extends ValidComponent = "button"> =
  ButtonRootProps<T> & {
    class?: string
    children: JSX.Element
    icon?: JSX.Element
  }

export const PowerButton = <T extends ValidComponent = "button">(
  props: PolymorphicProps<T, PowerButtonProps<T>>,
) => {
  const [local, rest] = splitProps(props as PowerButtonProps, [
    "children",
    "class",
    "icon",
  ])
  let rootRef: HTMLButtonElement | HTMLAnchorElement | undefined
  let glowRef: HTMLSpanElement | undefined
  let beamRef: HTMLSpanElement | undefined
  let iconRef: HTMLSpanElement | undefined

  onMount(() => {
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return
    if (!rootRef || !glowRef || !beamRef) return

    const glowAnimation = animate(glowRef, {
      opacity: [0.18, 0.42, 0.18],
      scale: [0.96, 1.04, 0.96],
      duration: 2600,
      ease: "inOutQuad",
      loop: true,
    })

    const beamAnimation = animate(beamRef, {
      translateX: ["-140%", "300%"],
      opacity: [0, 0.28, 0],
      duration: 2200,
      delay: 250,
      ease: "inOutQuad",
      loop: true,
    })

    const iconAnimation = iconRef
      ? animate(iconRef, {
          translateX: [0, 4, 0],
          duration: 1200,
          ease: "inOutQuad",
          loop: true,
        })
      : null

    const handleEnter = () => {
      animate(rootRef, {
        scale: [1, 1.02, 1],
        duration: 320,
        ease: "outQuint",
      })
    }

    rootRef.addEventListener("mouseenter", handleEnter)

    onCleanup(() => {
      glowAnimation.pause()
      beamAnimation.pause()
      iconAnimation?.pause()
      rootRef?.removeEventListener("mouseenter", handleEnter)
    })
  })

  return (
    <Button
      ref={rootRef}
      size="lg"
      class={`relative isolate overflow-hidden whitespace-nowrap uppercase ${local.class || ""}`}
      {...rest}
    >
      <span
        ref={glowRef}
        aria-hidden="true"
        class="absolute rounded pointer-events-none inset--1 bg-primary/12 blur-md"
      />
      <span
        ref={beamRef}
        aria-hidden="true"
        class="absolute inset-y-0 left-0 w-20 from-transparent to-transparent pointer-events-none bg-linear-to-r via-primary/55"
      />
      <span class="flex relative gap-2 items-center z-1">
        <span>{local.children}</span>
        <span ref={iconRef} class="flex items-center" data-power-button-icon>
          {local.icon}
        </span>
      </span>
    </Button>
  )
}
