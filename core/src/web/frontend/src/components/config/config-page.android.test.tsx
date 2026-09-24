/**
 * Config, through the component the Android WebView renders.
 *
 * PocketClaw's Core only ever runs on Android, and three controls on this page
 * could never do anything there: "Enable Devices" and "Monitor USB" drive a
 * udevadm-based USB monitor that an Android app cannot use, and "Launch at
 * Login" wrote a desktop autostart entry and said itself it was unsupported on
 * this platform. "Service Port" was saved and reported as applied, but the
 * Android host always starts the launcher with an explicit -port, so a stored
 * port never takes effect. These assert all four are gone and that saving
 * still round-trips the launcher settings the page no longer shows.
 */
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"

import en from "@/i18n/locales/en.json"

const launcherFetch = vi.fn()
const patchAppConfig = vi.fn()
const getLauncherConfig = vi.fn()
const setLauncherConfig = vi.fn()

vi.mock("@/api/http", () => ({
  launcherFetch: (path: string, init?: RequestInit) =>
    launcherFetch(path, init),
}))

vi.mock("@/api/channels", () => ({
  patchAppConfig: (patch: unknown) => patchAppConfig(patch),
  resetAppConfig: vi.fn(),
}))

vi.mock("@/api/launcher-auth", () => ({
  postLauncherDashboardSetup: vi.fn(),
}))

vi.mock("@/api/system", () => ({
  getLauncherConfig: () => getLauncherConfig(),
  getSystemVersionInfo: async () => ({ version: "0.2.1", go_version: "go" }),
  setLauncherConfig: (payload: unknown) => setLauncherConfig(payload),
}))

vi.mock("@/store/gateway", () => ({ refreshGatewayState: vi.fn() }))

vi.mock("@/lib/restart-required", () => ({
  showSaveSuccessOrRestartToast: vi.fn(),
}))

vi.mock("@/components/page-header", () => ({
  PageHeader: ({ title }: { title: string }) => <h1>{title}</h1>,
}))

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children }: { children: React.ReactNode }) => <a>{children}</a>,
}))

function translate(key: string): string {
  const value = key
    .split(".")
    .reduce<unknown>(
      (node, part) =>
        node && typeof node === "object"
          ? (node as Record<string, unknown>)[part]
          : undefined,
      en,
    )
  return typeof value === "string" ? value : key
}

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: translate }),
}))

const { ConfigPage } = await import("@/components/config/config-page")

const STORED_LAUNCHER = {
  port: 18800,
  public: false,
  allowed_cidrs: [],
  allow_localhost_bypass: true,
  trusted_proxy_cidrs: [],
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <ConfigPage />
    </QueryClientProvider>,
  )
}

describe("Config page on Android", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    launcherFetch.mockImplementation(async () => ({
      ok: true,
      json: async () => ({
        agents: { defaults: { workspace: "/w" } },
        devices: { enabled: true, monitor_usb: true },
      }),
    }))
    getLauncherConfig.mockResolvedValue(STORED_LAUNCHER)
    setLauncherConfig.mockImplementation(async (payload: unknown) => payload)
  })

  it("does not render the Devices card, launch-at-login or the port", async () => {
    renderPage()
    await screen.findByText(en.pages.config.sections.launcher)

    for (const gone of [
      "Devices",
      "Enable Devices",
      "Monitor USB",
      "Launch at Login",
      "Service Port",
    ]) {
      expect(screen.queryByText(gone), gone).toBeNull()
    }
    // The sections that do work on Android are still there.
    expect(screen.getByText(en.pages.config.sections.agent)).toBeTruthy()
    expect(screen.getByText(en.pages.config.sections.cron)).toBeTruthy()
  })

  it("never asks the backend for the retired autostart endpoint", async () => {
    renderPage()
    await screen.findByText(en.pages.config.sections.launcher)
    const paths = launcherFetch.mock.calls.map(([path]) => String(path))
    expect(paths.some((path) => path.includes("autostart"))).toBe(false)
  })

  it("saves launcher settings with the stored port and no devices patch", async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByText(en.pages.config.sections.launcher)

    await user.click(
      screen.getByRole("switch", { name: en.pages.config.lan_access }),
    )
    await user.click(screen.getAllByRole("button", { name: en.common.save })[0])

    await waitFor(() => expect(setLauncherConfig).toHaveBeenCalledTimes(1))
    expect(setLauncherConfig.mock.calls[0][0]).toMatchObject({
      port: 18800,
      public: true,
    })
    for (const [patch] of patchAppConfig.mock.calls) {
      expect(patch).not.toHaveProperty("devices")
    }
  })
})
