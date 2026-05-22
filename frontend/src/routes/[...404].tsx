import { A } from "@solidjs/router"

const NotFoundRoute = () => {
  return (
    <main class="m-4 text-center bento-cell">
      <h1 class="my-4">Not Found</h1>
      <p class="font-mono text-2xl">
        <span>404 - </span>
        <A class="underline" href="/">
          Go back home
        </A>
      </p>
    </main>
  )
}

export default NotFoundRoute
