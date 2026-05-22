import {
  ColorModeProvider,
  ColorModeScript,
  cookieStorageManagerSSR,
} from "@kobalte/core"
import { Link, MetaProvider } from "@solidjs/meta"
import { Router } from "@solidjs/router"
import { FileRoutes } from "@solidjs/start/router"
import { isServer, Suspense } from "solid-js/web"
import { getCookie } from "vinxi/http"
import { Layout } from "./components/layout/layout"

function getThemeServer() {
  "use server"
  const colorMode = getCookie("kb-color-mode")
  return colorMode ? `kb-color-mode=${colorMode}` : ""
}

const App = () => {
  const storageManager = cookieStorageManagerSSR(
    isServer ? getThemeServer() : document.cookie,
  )

  return (
    <Router
      root={(props) => (
        <MetaProvider>
          <Link
            rel="apple-touch-icon"
            sizes="180x180"
            href="/apple-touch-icon.png"
          />
          <Link
            rel="icon"
            type="image/png"
            sizes="32x32"
            href="/favicon-32x32.png"
          />
          <Link
            rel="icon"
            type="image/png"
            sizes="16x16"
            href="/favicon-16x16.png"
          />
          <Link rel="manifest" href="/site.webmanifest" />
          <ColorModeScript storageType={storageManager.type} />
          <ColorModeProvider storageManager={storageManager}>
            <Suspense>
              <Layout>{props.children}</Layout>
            </Suspense>
          </ColorModeProvider>
        </MetaProvider>
      )}
    >
      <FileRoutes />
    </Router>
  )
}

export default App
