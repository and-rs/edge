import { For } from "solid-js"
import { NavLink } from "~/components/ui/navlink"
import { ThemeSwitch } from "./theme-switch"

const INTERNAL_ROUTES = [{ href: "/", label: "Home" }]

export const Nav = () => {
  return (
    <nav class="max-w-800px mx-a">
      <div class="flex fixed gap-4 p-4 m-4 border-2 shadow z-60 border-muted bg-card">
        <For each={INTERNAL_ROUTES}>
          {(route) => <NavLink href={route.href} label={route.label} />}
        </For>
        <ThemeSwitch />
      </div>
    </nav>
  )
}
