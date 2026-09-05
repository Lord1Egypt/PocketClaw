/**
 * The Vision / Image Model control.
 *
 * Unset is a state, not a gap: with no vision model configured an image turn
 * goes to the default model, which is what every install did before this
 * setting existed. The picker draws from the same configured-provider source as
 * Default and Fallback, so an unconfigured provider template cannot appear here
 * any more than it can there.
 */
import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"

import type { ModelInfo, ModelProviderOption } from "@/api/models"

const setVisionModel = vi.fn()
const materializeModel = vi.fn()
const fetchUpstreamModels = vi.fn()
const applyGatewayConfigIfRequired = vi.fn()

vi.mock("@/api/models", () => ({
  setVisionModel: (...args: unknown[]) => setVisionModel(...args),
  materializeModel: (...args: unknown[]) => materializeModel(...args),
  fetchUpstreamModels: (...args: unknown[]) => fetchUpstreamModels(...args),
}))
vi.mock("@/lib/restart-required", () => ({
  applyGatewayConfigIfRequired: (...args: unknown[]) =>
    applyGatewayConfigIfRequired(...args),
}))
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

import i18n from "@/i18n"

const { VisionModelSection } = await import("./vision-model-section")

function entry(overrides: Partial<ModelInfo> = {}): ModelInfo {
  return {
    index: 0,
    model_name: "m",
    provider: "openai",
    model: "m-id",
    api_key: "",
    enabled: true,
    available: true,
    status: "available",
    is_default: false,
    is_virtual: false,
    default_model_allowed: true,
    ...overrides,
  }
}

/// Two configured providers and four shipped keyless templates.
const models: ModelInfo[] = [
  entry({
    index: 0,
    model_name: "opencode-primary",
    provider: "opencode",
    model: "deepseek-v4-flash-free",
    api_base: "https://opencode.example/v1",
  }),
  entry({
    index: 1,
    model_name: "gemini-primary",
    provider: "gemini",
    model: "gemini-3.7-flash",
    api_base: "https://gemini.example/v1beta",
  }),
  entry({
    index: 2,
    model_name: "azure-gpt5",
    provider: "azure",
    model: "deployment",
    status: "unconfigured",
    available: false,
  }),
  entry({
    index: 3,
    model_name: "cerebras-llama",
    provider: "cerebras",
    model: "llama-3.3-70b",
    status: "unconfigured",
    available: false,
  }),
  entry({
    index: 4,
    model_name: "ollama-llama3",
    provider: "ollama",
    model: "llama3",
    status: "unconfigured",
    available: false,
  }),
  entry({
    index: 5,
    model_name: "ark-code",
    provider: "volcengine",
    model: "ark",
    status: "unconfigured",
    available: false,
  }),
]

const providerOptions: ModelProviderOption[] = [
  {
    id: "opencode",
    display_name: "OpenCode",
    default_api_base: "https://opencode.example/v1",
    empty_api_key_allowed: false,
    create_allowed: true,
    default_model_allowed: true,
    supports_fetch: true,
  },
  {
    id: "gemini",
    display_name: "Google Gemini",
    default_api_base: "https://gemini.example/v1beta",
    empty_api_key_allowed: false,
    create_allowed: true,
    default_model_allowed: true,
    supports_fetch: true,
  },
  // Advertises listing but is unconfigured, so it must never be queried.
  {
    id: "azure",
    display_name: "Azure",
    default_api_base: "",
    empty_api_key_allowed: false,
    create_allowed: true,
    default_model_allowed: true,
    supports_fetch: true,
  },
]

function renderSection(visionModelName = "") {
  return render(
    <VisionModelSection
      models={models}
      visionModelName={visionModelName}
      defaultModelName="opencode-primary"
      providerOptions={providerOptions}
      onSaved={vi.fn()}
    />,
  )
}

async function openPicker() {
  await userEvent.click(
    screen.getByRole("button", { name: i18n.t("models.visionModel.change") }),
  )
  await waitFor(() =>
    expect(
      screen.getByPlaceholderText(i18n.t("models.discovery.searchPlaceholder")),
    ).toBeTruthy(),
  )
}

describe("the vision picker offers configured providers only", () => {
  beforeEach(async () => {
    vi.clearAllMocks()
    await i18n.changeLanguage("en")
    fetchUpstreamModels.mockResolvedValue({ models: [], total: 0 })
    applyGatewayConfigIfRequired.mockResolvedValue(undefined)
  })

  it("groups by the two providers the user configured", async () => {
    renderSection()
    await openPicker()
    expect(screen.getByText("OpenCode")).toBeTruthy()
    expect(screen.getByText("Google Gemini")).toBeTruthy()
  })

  it.each(["Azure", "cerebras", "ollama", "volcengine"])(
    "never lists the unconfigured %s template",
    async (label) => {
      renderSection()
      await openPicker()
      expect(screen.queryByText(label)).toBeNull()
    },
  )

  it("lists live-discovered models under the provider that reported them", async () => {
    fetchUpstreamModels.mockImplementation(
      async ({ provider }: { provider: string }) =>
        provider === "gemini"
          ? { models: [{ id: "gemini-3.7-pro-vision" }], total: 1 }
          : { models: [], total: 0 },
    )
    renderSection()
    await openPicker()
    await waitFor(() =>
      expect(screen.getByText("gemini-3.7-pro-vision")).toBeTruthy(),
    )
    // And the configured one is still there, marked as such.
    expect(screen.getByText("gemini-3.7-flash")).toBeTruthy()
    expect(
      screen.getAllByText(i18n.t("models.discovery.configured")).length,
    ).toBeGreaterThan(0)
    expect(
      screen.getAllByText(i18n.t("models.discovery.discovered")).length,
    ).toBeGreaterThan(0)
  })

  it("never sends a credential when discovering", async () => {
    renderSection()
    await waitFor(() => expect(fetchUpstreamModels).toHaveBeenCalled())
    for (const call of fetchUpstreamModels.mock.calls) {
      const req = call[0] as { api_key?: string; model_index?: number }
      expect(req.api_key).toBeUndefined()
      expect(typeof req.model_index).toBe("number")
    }
  })

  it("keeps one provider's failure off the others", async () => {
    fetchUpstreamModels.mockImplementation(
      async ({ provider }: { provider: string }) => {
        if (provider === "gemini") throw new Error("502")
        return { models: [{ id: "opencode-discovered" }], total: 1 }
      },
    )
    renderSection()
    await openPicker()

    await waitFor(() =>
      expect(screen.getByText(i18n.t("models.discovery.failed"))).toBeTruthy(),
    )
    // The healthy provider's discovered model still arrived...
    expect(screen.getByText("opencode-discovered")).toBeTruthy()
    // ...and the failing provider kept its configured entry.
    expect(screen.getByText("gemini-3.7-flash")).toBeTruthy()
    // The global catalog is still nowhere.
    expect(screen.queryByText("Azure")).toBeNull()
  })

  it("retries just the failing provider", async () => {
    fetchUpstreamModels.mockImplementation(
      async ({ provider }: { provider: string }) => {
        if (provider === "gemini") throw new Error("502")
        return { models: [], total: 0 }
      },
    )
    renderSection()
    await openPicker()
    await waitFor(() =>
      expect(screen.getByText(i18n.t("models.discovery.failed"))).toBeTruthy(),
    )

    fetchUpstreamModels.mockResolvedValue({
      models: [{ id: "recovered-model" }],
      total: 1,
    })
    fireEvent.click(
      screen.getByRole("button", { name: i18n.t("models.discovery.retry") }),
    )
    await waitFor(() =>
      expect(screen.getByText("recovered-model")).toBeTruthy(),
    )
  })
})

describe("selecting a vision model", () => {
  beforeEach(async () => {
    vi.clearAllMocks()
    await i18n.changeLanguage("en")
    fetchUpstreamModels.mockResolvedValue({ models: [], total: 0 })
    applyGatewayConfigIfRequired.mockResolvedValue(undefined)
    setVisionModel.mockResolvedValue({ status: "ok", image_model: "x" })
    materializeModel.mockResolvedValue({
      status: "ok",
      model_name: "gemini-3.7-pro-vision",
      index: 6,
      created: true,
      role: "vision",
      default_model: "opencode-primary",
      image_model: "gemini-3.7-pro-vision",
      model_fallbacks: [],
    })
  })

  it("points at an already-configured entry by name", async () => {
    renderSection()
    await openPicker()
    await userEvent.click(screen.getByText("gemini-3.7-flash"))

    await waitFor(() =>
      expect(setVisionModel).toHaveBeenCalledWith("gemini-primary"),
    )
    expect(materializeModel).not.toHaveBeenCalled()
  })

  it("materializes a discovered model with the vision role in one call", async () => {
    fetchUpstreamModels.mockImplementation(
      async ({ provider }: { provider: string }) =>
        provider === "gemini"
          ? { models: [{ id: "gemini-3.7-pro-vision" }], total: 1 }
          : { models: [], total: 0 },
    )
    renderSection()
    await openPicker()
    await waitFor(() =>
      expect(screen.getByText("gemini-3.7-pro-vision")).toBeTruthy(),
    )
    await userEvent.click(screen.getByText("gemini-3.7-pro-vision"))

    await waitFor(() => expect(materializeModel).toHaveBeenCalledTimes(1))
    const req = materializeModel.mock.calls[0][0] as {
      role: string
      model: string
      source_index: number
      api_key?: string
    }
    expect(req.role).toBe("vision")
    expect(req.model).toBe("gemini-3.7-pro-vision")
    // The provider instance the model came from, resolved to its model_list
    // index — never a key.
    expect(req.source_index).toBe(1)
    expect(req.api_key).toBeUndefined()
    expect(setVisionModel).not.toHaveBeenCalled()
  })
})

describe("the unset state", () => {
  beforeEach(async () => {
    vi.clearAllMocks()
    await i18n.changeLanguage("en")
    fetchUpstreamModels.mockResolvedValue({ models: [], total: 0 })
    applyGatewayConfigIfRequired.mockResolvedValue(undefined)
    setVisionModel.mockResolvedValue({ status: "ok", image_model: "" })
  })

  it("reads as Auto and explains where image turns go", () => {
    renderSection()
    expect(screen.getByText(i18n.t("models.visionModel.auto"))).toBeTruthy()
    expect(screen.getByText(i18n.t("models.visionModel.autoHint"))).toBeTruthy()
  })

  it("offers no clear action when nothing is configured", () => {
    renderSection()
    expect(
      screen.queryByRole("button", { name: i18n.t("models.visionModel.clear") }),
    ).toBeNull()
  })

  it("clears back to Auto with an empty name", async () => {
    renderSection("gemini-primary")
    expect(screen.getByText("gemini-primary")).toBeTruthy()

    await userEvent.click(
      screen.getByRole("button", { name: i18n.t("models.visionModel.clear") }),
    )
    await waitFor(() => expect(setVisionModel).toHaveBeenCalledWith(""))
  })

  // No provider reports capabilities today, so the honest statement is that it
  // is unknown. A badge guessed from the model name would be a lie the user
  // could route on.
  it("says capability is unknown rather than guessing from the name", () => {
    renderSection()
    expect(
      screen.getByText(i18n.t("models.visionModel.capabilityUnknown")),
    ).toBeTruthy()
  })
})

describe("localization and direction", () => {
  beforeEach(async () => {
    vi.clearAllMocks()
    fetchUpstreamModels.mockResolvedValue({ models: [], total: 0 })
  })

  it.each(["ar", "de", "ja", "ru"])("has translated strings in %s", async (locale) => {
    await i18n.changeLanguage(locale)
    for (const key of [
      "models.visionModel.title",
      "models.visionModel.description",
      "models.visionModel.auto",
      "models.visionModel.change",
      "models.visionModel.clear",
      "models.visionModel.capabilityUnknown",
      "models.discovery.configured",
      "models.discovery.discovered",
      "models.discovery.retry",
    ]) {
      expect(i18n.t(key), `${locale} ${key}`).not.toBe(key)
    }
  })

  it("renders in Arabic and keeps the model identifier left-to-right", async () => {
    await i18n.changeLanguage("ar")
    renderSection("gemini-primary")
    expect(screen.getByText(i18n.t("models.visionModel.title"))).toBeTruthy()
    const identifier = screen.getByText("gemini · gemini-3.7-flash")
    expect(identifier.getAttribute("dir")).toBe("ltr")
    await i18n.changeLanguage("en")
  })
})
