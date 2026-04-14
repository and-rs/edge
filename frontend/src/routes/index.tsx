import { Brand } from "~/components/brand"
import { FlightDiagnostics } from "~/components/flight"

// TODO: Add data-heavy logic for improved performance
const NormalizeRoute = () => {
  return (
    <main class="px-4 pt-4 pb-3">
      <div class="flex flex-col gap-3">
        <div class="flex flex-col gap-4 md:col-span-3 bento-cell md:h-min">
          <Brand size="lg" />
          <span class="text-xl">
            Human-in-the-loop normalization, with speed. Like a <i>stint.</i>
          </span>
        </div>

        <FlightDiagnostics />
      </div>
    </main>
  )
}

export default NormalizeRoute

