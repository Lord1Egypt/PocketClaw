import { readFileSync } from "node:fs"
import { resolve } from "node:path"

import { fireEvent, render } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

type TestLogEntry = {
  id: string
  line: string
}

const gatewayLogState = vi.hoisted(() => ({ logs: [] as TestLogEntry[] }))

vi.mock("@/hooks/use-gateway-logs", () => ({
  useGatewayLogs: () => ({
    clearLogs: vi.fn(),
    clearing: false,
    logs: gatewayLogState.logs,
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

type ViewportMetrics = {
  clientHeight: number
  scrollHeight: number
  scrollTop: number
  writes: number[]
}

function configureViewport(
  viewport: HTMLDivElement,
  initial: Omit<ViewportMetrics, "writes">,
) {
  const metrics: ViewportMetrics = { ...initial, writes: [] }

  Object.defineProperties(viewport, {
    clientHeight: {
      configurable: true,
      get: () => metrics.clientHeight,
    },
    scrollHeight: {
      configurable: true,
      get: () => metrics.scrollHeight,
    },
    scrollTop: {
      configurable: true,
      get: () => metrics.scrollTop,
      set: (value: number) => {
        const maximum = Math.max(metrics.scrollHeight - metrics.clientHeight, 0)
        metrics.scrollTop = Math.max(0, Math.min(value, maximum))
        metrics.writes.push(metrics.scrollTop)
      },
    },
  })

  return metrics
}

function viewportIn(container: HTMLElement) {
  return container.querySelector<HTMLDivElement>(
    '[data-slot="scroll-area-viewport"]',
  )!
}

describe("Core Web Console Logs page", () => {
  beforeEach(() => {
    gatewayLogState.logs = []
  })

  it("renders the shared brand-safe plain-text representation", () => {
    gatewayLogState.logs = contract.map((fixture, index) => ({
      id: `7:${index}`,
      line: fixture.input,
    }))

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

  it("renders Telegram API operations without token fragments", () => {
    const token = "123456789:AAExampleSecretTokenValue"
    gatewayLogState.logs = [
      {
        id: "12:0",
        line: `DBG telego bot.go:247 > API call to: "https://api.telegram.org/bot${token}/getMe"`,
      },
      {
        id: "12:1",
        line: `DBG telego bot.go:247 > API call to: "https://api.telegram.org/bot${token}/getUpdates"`,
      },
      {
        id: "12:2",
        line: `DBG telego bot.go:245 > API call to: "https://api.telegram.org/bot${token}/sendMessage", with data: {"chat_id":42}`,
      },
      {
        id: "12:3",
        line: `ERR telego bot.go:170 > Post "https://api.telegram.org/bot${token}/editMessageText": context deadline exceeded`,
      },
      {
        id: "12:4",
        line: `ERR telego bot.go:170 > POST "https://telegram.example/v1/bot${token}/answerCustomQuery" 503 53.616µs`,
      },
    ]

    const { container } = render(<LogsPage />)
    const rendered = container.textContent ?? ""

    for (const detail of [
      "Telegram API call: getMe",
      "Telegram API call: getUpdates",
      "Telegram API call: sendMessage",
      "Telegram API call: editMessageText",
      "Telegram API call: answerCustomQuery",
      "context deadline exceeded",
      "503 53.616µs",
    ]) {
      expect(rendered).toContain(detail)
    }
    for (const fragment of [
      token,
      "123456789:",
      "AAExample",
      "TokenValue",
      "AAEx****alue",
    ]) {
      expect(rendered).not.toContain(fragment)
    }
  })

  it("normalizes only classified legacy startup identities", () => {
    gatewayLogState.logs = [
      {
        id: "13:0",
        line: "DBG pid pidfile.go:112 > wrote pid file: /data/user/0/app/files/.picoclaw/.picoclaw.pid success",
      },
      {
        id: "13:1",
        line: "DBG channels manager.go:1101 > Attempting to initialize channel channel=pico type=pico",
      },
      {
        id: "13:2",
        line: "INF channels manager.go:1291 > Webhook handler registered channel=pico path=/pico/",
      },
      {
        id: "13:3",
        line: "INF channels manager.go:1347 > Starting channel channel=pico",
      },
      {
        id: "13:4",
        line: `WRN channels base.go:131 > SECURITY: Channel allows EVERYONE (allow_from is empty) channel=pico hint="Set allow_from to your ID, or use '*' to explicitly acknowledge open access."`,
      },
      {
        id: "13:5",
        line: "INF pico pico.go:248 > Starting Pico Protocol channel",
      },
      {
        id: "13:6",
        line: "WRN agent agent_outbound.go:205 > Failed to publish pico reasoning channel=pico error=timeout",
      },
    ]

    const { container } = render(<LogsPage />)
    const rendered = container.textContent ?? ""

    for (const expected of [
      "Gateway PID file written successfully",
      "channel=pocketclaw type=pocketclaw",
      "channel=pocketclaw path=<internal>",
      "Starting channel channel=pocketclaw",
      "SECURITY: Channel allows EVERYONE",
      "Starting PocketClaw realtime channel",
      "Failed to publish realtime reasoning channel=pocketclaw error=timeout",
    ]) {
      expect(rendered).toContain(expected)
    }
    for (const forbidden of [
      ".picoclaw.pid",
      "channel=pico",
      "type=pico",
      "/pico/",
      "Pico Protocol",
    ]) {
      expect(rendered).not.toContain(forbidden)
    }
  })

  it("does not rewrite unrelated pico substrings", () => {
    const unrelated =
      "INF picometer picophone.go:42 > compatibility pico_client.go topic=pico-test filename=my-pico-notes.txt archive=.picoclaw.pid.backup"
    gatewayLogState.logs = [{ id: "14:0", line: unrelated }]

    const { container } = render(<LogsPage />)
    expect(container.textContent).toContain(unrelated)
  })

  it("keeps a following viewport pinned to the bottom before paint", () => {
    gatewayLogState.logs = [
      { id: "8:0", line: "first" },
      { id: "8:1", line: "second" },
    ]
    const { container, rerender } = render(<LogsPage />)
    const viewport = viewportIn(container)
    const metrics = configureViewport(viewport, {
      clientHeight: 100,
      scrollHeight: 240,
      scrollTop: 140,
    })

    fireEvent.scroll(viewport)
    metrics.scrollHeight = 300
    gatewayLogState.logs = [
      ...gatewayLogState.logs,
      { id: "8:2", line: "new log" },
    ]
    rerender(<LogsPage />)

    expect(metrics.scrollTop).toBe(200)
    expect(metrics.writes).toEqual([200])
  })

  it("preserves scrollTop when the user has scrolled upward", () => {
    gatewayLogState.logs = [
      { id: "9:0", line: "first" },
      { id: "9:1", line: "second" },
    ]
    const { container, rerender } = render(<LogsPage />)
    const viewport = viewportIn(container)
    const metrics = configureViewport(viewport, {
      clientHeight: 100,
      scrollHeight: 300,
      scrollTop: 42,
    })

    fireEvent.scroll(viewport)
    metrics.scrollHeight = 380
    gatewayLogState.logs = [
      ...gatewayLogState.logs,
      { id: "9:2", line: "new log" },
    ]
    rerender(<LogsPage />)

    expect(metrics.scrollTop).toBe(42)
    expect(metrics.writes).toEqual([])
  })

  it("retains the same DOM node and text for an unchanged long entry", () => {
    const longLine =
      'DBG telegram telego bot.go:173 > API response {"description":"رسالة طويلة ✅","duration":"53.616µs","payload":"' +
      "x".repeat(300) +
      '"}'
    gatewayLogState.logs = [{ id: "10:27", line: longLine }]
    const { container, rerender } = render(<LogsPage />)
    const viewport = viewportIn(container)
    const metrics = configureViewport(viewport, {
      clientHeight: 100,
      scrollHeight: 400,
      scrollTop: 50,
    })
    fireEvent.scroll(viewport)

    const before = container.querySelector('[data-log-entry-id="10:27"]')
    metrics.scrollHeight = 460
    gatewayLogState.logs = [
      ...gatewayLogState.logs,
      { id: "10:28", line: "next event" },
    ]
    rerender(<LogsPage />)
    const after = container.querySelector('[data-log-entry-id="10:27"]')

    expect(after).toBe(before)
    expect(after?.textContent).toBe(longLine)
    expect(metrics.scrollTop).toBe(50)
    expect(metrics.writes).toEqual([])
  })

  it("does not change scrollTop across repeated polls with no new logs", () => {
    gatewayLogState.logs = [{ id: "11:0", line: "unchanged" }]
    const { container, rerender } = render(<LogsPage />)
    const viewport = viewportIn(container)
    const metrics = configureViewport(viewport, {
      clientHeight: 100,
      scrollHeight: 300,
      scrollTop: 60,
    })
    fireEvent.scroll(viewport)

    rerender(<LogsPage />)
    rerender(<LogsPage />)

    expect(metrics.scrollTop).toBe(60)
    expect(metrics.writes).toEqual([])
  })
})
