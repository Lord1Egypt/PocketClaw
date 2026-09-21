import { createFileRoute, redirect } from "@tanstack/react-router"

/**
 * `/credentials` is not a v0.2.0 surface.
 *
 * Account-login credential management is unfinished — OpenAI browser OAuth
 * reaches a real authentication screen and returns `unknown_error` — so the
 * navigation entry is withdrawn for this release. The page component and its
 * API stay in the tree for the later feature phase; only the way in is closed.
 *
 * The route itself is kept and redirected rather than deleted, because a
 * bookmark or a stale embedded bundle can still ask for this path and the one
 * thing it must not do is render unfinished auth controls. `/models` is the
 * destination because it is where provider access is actually configured in
 * v0.2.0 — a user who asked for Credentials wanted to set up a provider, and
 * that is the finished path to it.
 *
 * `beforeLoad` runs before the component, so nothing here mounts, no OAuth
 * request is made, and no button is rendered on the way past.
 */
export const Route = createFileRoute("/credentials")({
  beforeLoad: () => {
    throw redirect({ to: "/models" })
  },
})
