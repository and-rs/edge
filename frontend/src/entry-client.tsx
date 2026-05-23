// @refresh reload
import { mount, StartClient } from "@solidjs/start/client"
import "virtual:uno.css"

const mountApp = () => {
  const root = document.getElementById("app")
  if (!(root instanceof HTMLElement)) {
    throw new Error("Root element #app not found")
  }

  return mount(() => <StartClient />, root)
}

export default mountApp

mountApp()
