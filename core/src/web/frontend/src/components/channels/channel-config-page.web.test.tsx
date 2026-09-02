/**
 * Channels → Web, through the component the physical APK renders.
 *
 * The Web channel is PocketClaw's internal realtime transport — the console's
 * Chat page talks to it over /pico/ws — and its allow-list is not a setting.
 * pkg/channels/pico stamps every inbound sender itself and enforces a hardcoded
 * owner allow-list "regardless of client payload fields or a stale/permissive
 * on-disk allowlist", while the console backend and the Android host both
 * rewrite the on-disk value on every provisioning pass.
 *
 * So the field showed an internal principal — pico-user — that a user could
 * neither meaningfully change nor remove. These assert it is gone from the UI
 * and that nothing about the stored value changed.
 */
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"

import en from "@/i18n/locales/en.json"

const getChannelsCatalog = vi.fn()
const getChannelConfig = vi.fn()
const patchAppConfig = vi.fn()

vi.mock("@/api/channels", () => ({
  getChannelsCatalog: () => getChannelsCatalog(),
  getChannelConfig: (name: string) => getChannelConfig(name),
  patchAppConfig: (patch: unknown) => patchAppConfig(patch),
}))

vi.mock("@/hooks/use-gateway", () => ({
  useGateway: () => ({ state: "running" }),
}))

vi.mock("@/store/gateway", () => ({ refreshGatewayState: vi.fn() }))

vi.mock("@/lib/restart-required", () => ({
  showSaveSuccessOrRestartToast: vi.fn(),
}))

vi.mock("@/components/page-header", () => ({
  PageHeader: ({ title }: { title: string }) => <h1>{title}</h1>,
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

const { ChannelConfigPage } = await import(
  "@/components/channels/channel-config-page"
)

const WEB_CHANNEL = { name: "pico", display_name: "Web", config_key: "pico" }

/** What a provisioned install actually has on disk. */
const PROVISIONED_CONFIG = {
  enabled: true,
  allow_from: ["pico-user"],
  token: "",
  port: 18790,
}

function arrange(config: Record<string, unknown> = PROVISIONED_CONFIG) {
  getChannelsCatalog.mockResolvedValue({ channels: [WEB_CHANNEL] })
  getChannelConfig.mockResolvedValue({
    config,
    configured_secrets: ["token"],
    config_key: "pico",
  })
}

async function renderWebPage() {
  render(<ChannelConfigPage channelName="pico" />)
  await waitFor(() =>
    expect(screen.getByRole("heading", { name: "Web" })).toBeDefined(),
  )
}

describe("Channels → Web", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("does not show the Allow From control", async () => {
    arrange()
    await renderWebPage()

    expect(screen.queryByText(en.channels.field.allowFrom)).toBeNull()
    expect(screen.queryByText(en.channels.form.desc.allowFrom)).toBeNull()
  })

  it("shows no pico-user value anywhere on the page", async () => {
    const { container } = render(<ChannelConfigPage channelName="pico" />)
    arrange()
    await waitFor(() =>
      expect(screen.getByRole("heading", { name: "Web" })).toBeDefined(),
    )

    expect(container.textContent).not.toContain("pico-user")
  })

  it("shows no PicoClaw branding in the Web channel UI", async () => {
    arrange()
    const { container } = render(<ChannelConfigPage channelName="pico" />)
    await waitFor(() =>
      expect(screen.getAllByRole("heading", { name: "Web" }).length).toBeGreaterThan(0),
    )

    // "Pico" is the upstream name for this transport; the user sees "Web".
    expect(container.textContent).not.toMatch(/picoclaw/i)
    expect(container.textContent).not.toMatch(/\bpico[-_]/i)
  })

  it("still renders the channel's real settings", async () => {
    // Hiding one misleading control must not blank the page.
    arrange()
    await renderWebPage()

    expect(screen.getByRole("heading", { name: "Web" })).toBeDefined()
    expect(screen.getByText(en.channels.page.enableLabel)).toBeDefined()
  })

  it("keeps the stored allow_from when the page is saved", async () => {
    // The value is authoritative to the provisioners, not to this form. Hiding
    // the control must not amount to clearing the field on the next save.
    arrange()
    await renderWebPage()

    const toggle = screen.getByRole("switch")
    await userEvent.click(toggle)

    const save = screen.getByRole("button", { name: en.common.save })
    await waitFor(() => expect(save.hasAttribute("disabled")).toBe(false))
    await userEvent.click(save)

    await waitFor(() => expect(patchAppConfig).toHaveBeenCalledTimes(1))
    const payload = patchAppConfig.mock.calls[0][0] as {
      channel_list: { pico: Record<string, unknown> }
    }
    // List fields submit as canonical JSON arrays. What matters here is that
    // the principal survived the save rather than being cleared by a control
    // the user can no longer see.
    expect(payload.channel_list.pico.allow_from).toEqual(["pico-user"])
  })

  it("still shows Allow From for channels where it is a real setting", async () => {
    // The field is hidden for the Web channel specifically, not removed.
    getChannelsCatalog.mockResolvedValue({
      channels: [{ name: "irc", display_name: "IRC", config_key: "irc" }],
    })
    getChannelConfig.mockResolvedValue({
      config: { enabled: true, server: "irc.example.org", allow_from: ["someone"] },
      configured_secrets: [],
      config_key: "irc",
    })

    render(<ChannelConfigPage channelName="irc" />)
    await waitFor(() =>
      expect(screen.getByText(en.channels.field.allowFrom)).toBeDefined(),
    )
  })
})
