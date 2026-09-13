import { IconMoon, IconSun } from "@tabler/icons-react"
import { createFileRoute } from "@tanstack/react-router"
import * as React from "react"
import { useTranslation } from "react-i18next"

import {
  getLauncherAuthStatus,
  postLauncherDashboardSetup,
} from "@/api/launcher-auth"
import { LanguageMenu } from "@/components/language-menu"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { resolveLauncherSetupGate } from "@/lib/launcher-setup-gate"
import { Label } from "@/components/ui/label"
import { useTheme } from "@/hooks/use-theme"

function LauncherSetupPage() {
  const { t } = useTranslation()
  const { theme, toggleTheme } = useTheme()
  const [password, setPassword] = React.useState("")
  const [confirm, setConfirm] = React.useState("")
  const [submitting, setSubmitting] = React.useState(false)
  const [error, setError] = React.useState("")
  // PC-DEF-037. This route used to render the first-run "create a dashboard
  // password" form to anyone who navigated to it, including an unauthenticated
  // visitor on an already-initialized dashboard. The backend always refused the
  // POST, so nothing could be taken over -- but the page invited the attempt and
  // told a stranger that a PocketClaw dashboard lives here. The guard is UX
  // only; authorization stays in the backend (PC-DEF-039).
  const [gate, setGate] = React.useState<"checking" | "first-run" | "leaving">(
    "checking",
  )

  React.useEffect(() => {
    let cancelled = false
    void (async () => {
      let decision
      try {
        decision = resolveLauncherSetupGate(await getLauncherAuthStatus())
      } catch {
        decision = resolveLauncherSetupGate(null)
      }
      if (cancelled) return
      if (decision.render === "first-run") {
        setGate("first-run")
        return
      }
      setGate("leaving")
      globalThis.location.assign(decision.redirectTo)
    })()
    return () => {
      cancelled = true
    }
  }, [])

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setError("")
    if (password !== confirm) {
      setError(t("launcherSetup.errorMismatch"))
      return
    }
    setSubmitting(true)
    try {
      const result = await postLauncherDashboardSetup(password, confirm)
      if (result.ok) {
        globalThis.location.assign("/launcher-login")
        return
      }
      setError(result.error)
    } catch {
      setError(t("launcherSetup.errorNetwork"))
    } finally {
      setSubmitting(false)
    }
  }

  if (gate !== "first-run") {
    return (
      <div
        className="bg-background text-foreground flex min-h-dvh flex-col"
        data-testid="launcher-setup-gate"
      />
    )
  }

  return (
    <div className="bg-background text-foreground flex min-h-dvh flex-col">
      <header className="border-border/50 flex h-14 shrink-0 items-center justify-end gap-2 border-b px-4">
        <LanguageMenu variant="outline" />
        <Button
          variant="outline"
          size="icon"
          type="button"
          onClick={() => toggleTheme()}
          aria-label={
            theme === "dark" ? t("common.lightMode") : t("common.darkMode")
          }
        >
          {theme === "dark" ? (
            <IconSun className="size-4" />
          ) : (
            <IconMoon className="size-4" />
          )}
        </Button>
      </header>

      <div className="flex flex-1 items-center justify-center p-4">
        <Card className="w-full max-w-md" size="sm">
          <CardHeader>
            <CardTitle>{t("launcherSetup.title")}</CardTitle>
            <CardDescription>{t("launcherSetup.description")}</CardDescription>
          </CardHeader>
          <CardContent>
            <form className="flex flex-col gap-4" onSubmit={onSubmit}>
              <div className="flex flex-col gap-2">
                <Label htmlFor="setup-password">
                  {t("launcherSetup.passwordLabel")}
                </Label>
                <Input
                  id="setup-password"
                  name="password"
                  type="password"
                  autoComplete="new-password"
                  required
                  minLength={8}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={t("launcherSetup.passwordPlaceholder")}
                />
              </div>
              <div className="flex flex-col gap-2">
                <Label htmlFor="setup-confirm">
                  {t("launcherSetup.confirmLabel")}
                </Label>
                <Input
                  id="setup-confirm"
                  name="confirm"
                  type="password"
                  autoComplete="new-password"
                  required
                  minLength={8}
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  placeholder={t("launcherSetup.confirmPlaceholder")}
                />
              </div>
              <Button type="submit" disabled={submitting}>
                {submitting ? t("labels.loading") : t("launcherSetup.submit")}
              </Button>
              {error ? (
                <p className="text-destructive text-sm" role="alert">
                  {error}
                </p>
              ) : null}
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

export const Route = createFileRoute("/launcher-setup")({
  component: LauncherSetupPage,
})
