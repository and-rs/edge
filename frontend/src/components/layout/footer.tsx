import Github from "lucide-solid/icons/github"
import Linkedin from "lucide-solid/icons/linkedin"
import { For } from "solid-js"
import { NavLink } from "~/components/ui/navlink"
import { FlightDiagnostics } from "~/components/flight"
import { Brand } from "../brand"
const INTERNAL_ROUTES = [{ href: "/", label: "Home" }]

const EXTERNAL_LINKS = [
  {
    href: "https://www.linkedin.com/company/iridium-tech",
    label: "LinkedIn",
    icon: Linkedin,
  },
  {
    href: "https://www.github.com/and-rs/iridium",
    label: "GitHub",
    icon: Github,
  },
]

export const Footer = () => {
  return (
    <footer class="flex flex-col-reverse gap-12 justify-between mx-4 mb-4 sm:flex-row bento-cell">
      <div class="flex flex-col gap-4 justify-between md:flex-[2/3]">
        <Brand size="md" />

        <div class="flex flex-col gap-3">
          <FlightDiagnostics />
          <span>Copyright (c) 2026 Iridium. All Rights Reserved.</span>
        </div>
      </div>

      <div class="flex flex-col gap-6 mr-14 md:flex-[1/3]">
        <div class="flex flex-col gap-2">
          <span class="text-xl font-bold text-primary">Read more</span>
          <For each={INTERNAL_ROUTES}>
            {(route) => (
              <NavLink
                class="text-base"
                href={route.href}
                label={route.label}
              />
            )}
          </For>
        </div>

        <div class="flex flex-col gap-2">
          <span class="text-xl font-bold text-primary">Other sites</span>
          <For each={EXTERNAL_LINKS}>
            {(link) => (
              <NavLink
                class="text-base"
                href={link.href}
                label={link.label}
                icon={link.icon}
              />
            )}
          </For>
        </div>
      </div>
    </footer>
  )
}
