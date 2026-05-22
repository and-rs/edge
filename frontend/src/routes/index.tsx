import { Brand } from "~/components/brand"
import { FlightDiagnostics } from "~/components/flight"

const HomeRoute = () => {
  return (
    <main class="px-4 pt-4 pb-3">
      <div class="flex flex-col gap-3">
        <div class="flex flex-col gap-4 md:col-span-3 bento-cell md:h-min">
          <Brand size="lg" text="Iridium Edge" logoSrc="/iridium-edge.png" />
          <span class="text-xl">
            Event-to-market intelligence for crypto and prediction-market
            operators.
          </span>
        </div>

        <FlightDiagnostics />
      </div>
    </main>
  )
}

export default HomeRoute
