import { readFileSync } from "node:fs"
import { resolve } from "node:path"

import { render } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const gatewayLogState = vi.hoisted(() => ({ logs: [] as string[] }))

vi.mock("@/hooks/use-gateway-logs", () => ({
  useGatewayLogs: () => ({
    clearLogs: vi.fn(),
    clearing: false,
    logs: gatewayLogState.logs,
  }),
}))

vi.mock("@/hooks/use-log-wrap-columns", () => ({
  useLogWrapColumns: () => ({
    contentRef: { current: null },
    measureRef: { current: null },
    wrapColumns: 120,
  }),
}))

vi.mock("@/components/page-header", () => ({
  PageHeader: ({ title }: { title: string }) => <h1>{title}</h1>,
}))

vi.mock("@/components/logs/log-level-select", () => ({
  LogLevelSelect: () => null,
}))

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

const { LogsPage } = await import("@/components/logs/logs-page")

type LogContractCase = {
  expected: string
  input: string
  name: string
}

const contract = JSON.parse(
  readFileSync(
    resolve(
      process.cwd(),
      "../../../../test/fixtures/user_visible_log_contract.json",
    ),
    "utf8",
  ),
) as LogContractCase[]

describe("Core Web Console Logs page", () => {
  beforeEach(() => {
    gatewayLogState.logs = []
  })

  it("renders the shared brand-safe plain-text representation", () => {
    gatewayLogState.logs = contract.map((fixture) => fixture.input)

    const { container } = render(<LogsPage />)
    const rendered = container.textContent ?? ""

    expect(rendered).toContain("مدة 53.616µs ✅")
    expect(rendered).toContain("English عربي 😊")
    expect(rendered).toContain("GET /internal realtime connection 500")
    expect(rendered).toContain("Starting gateway process")
    expect(rendered).toContain(
      "INF realtime realtime.go:1013 > WebSocket client connected",
    )
    expect(rendered).not.toContain("\x1b")
    expect(rendered).not.toContain("[38;2;")
    expect(rendered).not.toContain("/pico/ws")
    expect(rendered).not.toContain("libpicoclaw.so")
    expect(rendered).not.toContain("PicoClaw")
    expect(rendered).not.toContain("picoclaw")
    expect(rendered).not.toContain("Sipeed")
    expect(rendered).not.toContain("sipeed")
    expect(rendered).not.toContain("INF pico pico.go:1013")
    expect(rendered).not.toContain("�")

    for (const fixture of contract) {
      if (fixture.expected) {
        expect(rendered, fixture.name).toContain(fixture.expected)
      }
    }
  })
})
