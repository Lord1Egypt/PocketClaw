import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { RouterProvider, createRouter } from "@tanstack/react-router"
import { StrictMode } from "react"
import ReactDOM from "react-dom/client"

import { AppProviders } from "./app-providers"
import { AppErrorBoundary } from "./components/app-error-boundary"
import { markAppReady } from "./lib/app-readiness"
import "./i18n"
import "./index.css"
import { routeTree } from "./routeTree.gen"

const queryClient = new QueryClient()

const router = createRouter({
  routeTree,
  context: {
    queryClient,
  },
})

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router
  }
}

const rootElement = document.getElementById("root")!
if (!rootElement.innerHTML) {
  const root = ReactDOM.createRoot(rootElement)
  root.render(
    <StrictMode>
      <AppErrorBoundary>
        <AppProviders>
          <QueryClientProvider client={queryClient}>
            <RouterProvider router={router} />
          </QueryClientProvider>
        </AppProviders>
      </AppErrorBoundary>
    </StrictMode>,
  )

  // Marked after the first paint, so the flag means "this page rendered", not
  // merely "this bundle parsed". The Android host reads it on resume to tell a
  // live console from a WebView whose renderer was killed while backgrounded.
  requestAnimationFrame(() => markAppReady())
}
