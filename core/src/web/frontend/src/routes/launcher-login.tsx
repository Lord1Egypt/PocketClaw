import { IconMoon, IconSun } from "@tabler/icons-react"
import { createFileRoute } from "@tanstack/react-router"
import * as React from "react"
import { useTranslation } from "react-i18next"

import {
  getLauncherAuthStatus,
  postLauncherDashboardLogin,
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
import { Label } from "@/components/ui/label"
import { useTheme } from "@/hooks/use-theme"
import { readPostAuthDestination } from "@/lib/post-auth-destination"

function LauncherLoginPage() {
  const { t } = useTranslation()
  const { theme, toggleTheme } = useTheme()
  const [password, setPassword] = React.useState("")
  const [submitting, setSubmitting] = React.useState(false)
  const [error, setError] = React.useState("")

  // Captured once from the URL rather than re-read per submit, so a wrong
  // password followed by the right one still lands on the original destination.
  const [destination] = React.useState(() =>
    readPostAuthDestination(globalThis.location.search),
  )

  // If the password store has never been initialized, go to setup instead.
  React.useEffect(() => {
    void getLauncherAuthStatus()
      .then((s) => {
        if (!s.initialized) {
          globalThis.location.assign("/launcher-setup")
        }
      })
      .catch(() => {
        /* network error — stay on login page */
      })
  }, [])

  const loginWithPassword = React.useCallback(
    async (passwordValue: string) => {
      setError("")
      setSubmitting(true)
      try {
        const result = await postLauncherDashboardLogin(passwordValue)
        if (result.ok) {
          // PC-DEF-059. Return to whatever was requested before the session
          // expired, validated against the route set — `next` arrives from a URL
          // and is untrusted. An unrecognised destination falls back to home.
          globalThis.location.assign(destination)
          return
        }
        if (result.status === 409) {
          globalThis.location.assign("/launcher-setup")
          return
        }
        if (result.status === 401) {
          setError(t("launcherLogin.errorInvalid"))
          return
        }
        setError(result.error)
      } catch {
        setError(t("launcherLogin.errorNetwork"))
      } finally {
        setSubmitting(false)
      }
    },
    [t, destination],
  )

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    await loginWithPassword(password)
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
            <CardTitle>{t("launcherLogin.title")}</CardTitle>
            <CardDescription>{t("launcherLogin.description")}</CardDescription>
          </CardHeader>
          <CardContent>
            <form className="flex flex-col gap-4" onSubmit={onSubmit}>
              <div className="flex flex-col gap-2">
                <Label htmlFor="launcher-password">
                  {t("launcherLogin.passwordLabel")}
                </Label>
                <Input
                  id="launcher-password"
                  name="password"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={t("launcherLogin.passwordPlaceholder")}
                />
              </div>
              <Button type="submit" disabled={submitting}>
                {submitting ? t("labels.loading") : t("launcherLogin.submit")}
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

export const Route = createFileRoute("/launcher-login")({
  component: LauncherLoginPage,
})
