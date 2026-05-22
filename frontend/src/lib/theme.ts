export const HEX_COLORS = {
  foreground: { light: "#070b0d", dark: "#f8fbfc" },
  background: { light: "#f8fbfc", dark: "#070b0d" },
  card: { light: "#edf5f7", dark: "#0c1317" },
  primary: { light: "#5493b6", dark: "#4989ab" },
  light: { light: "#5e94b3", dark: "#5e94b3" },
  secondary: { light: "#95c2d6", dark: "#29576a" },
  accent: { light: "#6ba9cc", dark: "#337094" },
  muted: { light: "#c6dbe7", dark: "#1d3644" },
  success: { light: "#519749", dark: "#6fb668" },
} as const

// Normalize hex to color(srgb X X X)
export function toSrgb(hex: string): string {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex)
  if (!result) return hex
  const [r, g, b] = [1, 2, 3].map((i) => parseInt(result[i], 16) / 255)
  return `color(srgb ${r} ${g} ${b})`
}

export const THEME_COLORS = Object.fromEntries(
  Object.entries(HEX_COLORS).map(([key, modes]) => [
    key,
    {
      light: toSrgb(modes.light),
      dark: toSrgb(modes.dark),
    },
  ]),
) as typeof HEX_COLORS

export type ThemeMode = "light" | "dark"

export const getThemeClient = (): ThemeMode => {
  return (
    (document.documentElement.getAttribute("data-kb-theme") as ThemeMode) ||
    "light"
  )
}
