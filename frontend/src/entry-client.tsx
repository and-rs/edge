// @refresh reload
import { mount, StartClient } from "@solidjs/start/client"
import "virtual:uno.css"

const root = document.getElementById("app")
if (!(root instanceof HTMLElement))
  throw new Error("Root element #app not found")

mount(() => <StartClient />, root)
