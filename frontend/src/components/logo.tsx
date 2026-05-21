import { animate } from "animejs"
import { createSignal, splitProps } from "solid-js"

export const Logo = (props: {
  class?: string
  src?: string
  alt?: string
  ariaLabel?: string
}) => {
  const [local, rest] = splitProps(props, ["class", "src", "alt", "ariaLabel"])
  const [rotations, setRotations] = createSignal(0)

  let logoRef: HTMLButtonElement | undefined

  const handleRotate = (e: MouseEvent) => {
    e.stopPropagation()
    if (!logoRef) return
    setRotations((c) => c + 1)
    animate(logoRef, {
      rotate: rotations() * 360,
      ease: "out(4)",
      duration: 1500,
    })
  }

  return (
    <button
      type="button"
      ref={logoRef}
      onClick={handleRotate}
      class={`z-10 aspect-square text-primary ${local.class || ""}`}
      aria-label={local.ariaLabel || "Rotate Iridium Edge Logo"}
      tabindex="-1"
      {...rest}
    >
      <img
        src={local.src || "/iridium.png"}
        class="w-full h-full"
        alt={local.alt || "Iridium Edge Logo"}
        draggable={false}
      />
    </button>
  )
}

