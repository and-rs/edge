import type { JSX } from "solid-js"
import { Footer } from "./footer"
import { Nav } from "./nav"

export const Layout = (props: { children: JSX.Element }) => {
  return (
    <>
      <Nav />
      <div class="pt-22 max-w-800px mx-a">
        {props.children}
        <Footer />
      </div>
    </>
  )
}
