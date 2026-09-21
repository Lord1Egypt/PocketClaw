import { Outlet, createRootRoute, useRouterState } from "@tanstack/react-router"
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import { getLauncherAuthStatus } from "@/api/launcher-auth"
import { AppLayout } from "@/components/app-layout"
import { initializeChatStore } from "@/features/chat/controller"
import { isLauncherAuthPathname } from "@/lib/launcher-login-path"
import { launcherLoginUrlFor } from "@/lib/post-auth-destination"

const RootLayout = () => {
  const { t } = useTranslation()
  // Prefer the real address bar path: stale embedded bundles may not register
  // /launcher-login or /launcher-setup in the route tree, which would otherwise
  // keep AppLayout + gateway polling → 401 → launcherFetch redirect loop.
  const routerState = useRouterState({
    select: (s) => ({
      pathname: s.location.pathname,
      matches: s.matches,
    }),
  })

  const windowPath =
    typeof globalThis.location !== "undefined"
      ? globalThis.location.pathname || "/"
      : routerState.pathname

  const isAuthPage =
    isLauncherAuthPathname(windowPath) ||
    isLauncherAuthPathname(routerState.pathname) ||
    routerState.matches.some(
      (m) => m.routeId === "/launcher-login" || m.routeId === "/launcher-setup",
    )

  const [authError, setAuthError] = useState<string | null>(null)

  // Session guard: proactively check auth status on every page load.
  useEffect(() => {
    if (isAuthPage) return
    void getLauncherAuthStatus()
      .then((s) => {
        if (!s.initialized) {
          globalThis.location.assign("/launcher-setup")
        } else if (!s.authenticated) {
          // PC-DEF-059. Remember where the user was going. Opening Manage Models
          // from native Settings lands on /models; without this the login page
          // sent them to / and the destination was lost.
          globalThis.location.assign(
            launcherLoginUrlFor(
              globalThis.location.pathname,
              globalThis.location.search,
            ),
          )
        }
      })
      .catch((err: unknown) => {
        // On 401/403, redirect to login — the session is invalid.
        // On 5xx (e.g. 503 when the auth store is unavailable) or network errors,
        // do NOT redirect: a subsequent successful login would loop straight back here.
        // launcherFetch handles 401 on real API calls regardless.
        if (err instanceof Error && /^status 40[13]$/.test(err.message)) {
          globalThis.location.assign(
            launcherLoginUrlFor(
              globalThis.location.pathname,
              globalThis.location.search,
            ),
          )
        } else {
          setAuthError(
            err instanceof Error
              ? err.message
              : t("header.authError.unavailable"),
          )
        }
      })
  }, [isAuthPage, t])

  useEffect(() => {
    if (isAuthPage) {
      return
    }
    initializeChatStore()
  }, [isAuthPage])

  if (isAuthPage) {
    return (
      <>
        <Outlet />
        {import.meta.env.DEV ? <TanStackRouterDevtools /> : null}
      </>
    )
  }

  return (
    <>
      {authError && (
        <div className="bg-destructive text-destructive-foreground fixed inset-x-0 top-0 z-[100] flex items-center justify-between px-4 py-2 text-sm shadow-md">
          <span>{t("header.authError.banner", { message: authError })}</span>
          <button
            className="ml-4 opacity-70 hover:opacity-100"
            onClick={() => setAuthError(null)}
            aria-label={t("common.dismiss")}
          >
            ✕
          </button>
        </div>
      )}
      <AppLayout>
        <Outlet />
        {import.meta.env.DEV ? <TanStackRouterDevtools /> : null}
      </AppLayout>
    </>
  )
}

export const Route = createRootRoute({ component: RootLayout })
