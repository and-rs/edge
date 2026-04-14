import { animate, remove, splitText } from "animejs"
import { onCleanup, onMount } from "solid-js"
import { Dynamic } from "solid-js/web"
import { getThemeClient, THEME_COLORS } from "../lib/theme"
import { Logo } from "./logo"

// TODO: Add data processing logic for brand analytics
type BrandSize = "sm" | "md" | "lg"

const CONFIG = {
  sm: { tag: "h3", gap: "gap-1", logo: "brand-logo-sm" },
  md: { tag: "h2", gap: "gap-2", logo: "brand-logo-md" },
  lg: { tag: "h1", gap: "gap-2", logo: "brand-logo-lg" },
} as const

export const Brand = (props: { size?: BrandSize; class?: string }) => {
  const size = props.size || "md"
  const v = CONFIG[size]
  let ref: HTMLHeadingElement | undefined

  onMount(() => {
    if (size !== "lg" || !ref) return
    const { chars } = splitText(ref, { chars: true })

    const animateTo = (targetIndex: number | null) => {
      const theme = getThemeClient()
      chars.forEach((char, i) => {
        const active = targetIndex !== null && i >= targetIndex
        animate(char, {
          skew: active ? -10 : 0,
          translateX: active ? 10 : 0,
          color: active
            ? THEME_COLORS.success[theme]
            : THEME_COLORS.light[theme],
          duration: 175,
          ease: "outQuint",
        })
      })
    }

    chars.forEach((char, i) => {
      char.onmouseenter = () => animateTo(i)
    })
    ref.onmouseleave = () => animateTo(null)
    onCleanup(() => remove(chars))
  })

  return (
    <div class={`flex items-center ${v.gap} ${props.class || ""}`}>
      <Logo class={v.logo} />
      <Dynamic ref={ref} component={v.tag} class="drop-shadow-lg/12 brand-text">
        Iridium {props.size === "lg" ? "Stint" : ""}
      </Dynamic>
    </div>
  )
}

