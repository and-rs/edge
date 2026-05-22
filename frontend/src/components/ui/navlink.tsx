import { A, useLocation } from "@solidjs/router"
import type { LucideProps } from "lucide-solid"
import { type Component, createMemo, Show } from "solid-js"
import { Dynamic } from "solid-js/web"
import { cn } from "~/lib/utils"

interface NavLinkProps {
  href: string
  label: string
  class?: string
  icon?: Component<LucideProps>
  iconClass?: string
}

export const NavLink = (props: NavLinkProps) => {
  const location = useLocation()
  const isExternal = createMemo(() => props.href.startsWith("http"))
  const isActive = createMemo(() => {
    if (isExternal()) return false
    return props.href === "/"
      ? location.pathname === "/"
      : location.pathname.startsWith(props.href)
  })

  return (
    <A
      href={props.href}
      target={isExternal() ? "_blank" : undefined}
      rel={isExternal() ? "noopener noreferrer" : undefined}
      aria-current={isActive() ? "page" : undefined}
      class={cn(
        "flex items-center text-xl w-min whitespace-nowrap",
        isActive() ? "btn-link-active" : "btn-link-inactive",
        props.class,
      )}
    >
      <Show when={props.icon}>
        <Dynamic
          component={props.icon}
          class={cn("mr-2", props.iconClass ?? "size-5")}
        />
      </Show>
      {props.label}
    </A>
  )
}
