import { useLayoutEffect, useMemo, useRef } from "react"
import { useTranslation } from "react-i18next"

import { PlainLogLine } from "@/components/logs/plain-log-line"
import { ScrollArea } from "@/components/ui/scroll-area"
import type { GatewayLogEntry } from "@/hooks/use-gateway-logs"
import { normalizeUserVisibleLog } from "@/lib/plain-text-log"

const AUTO_SCROLL_THRESHOLD_PX = 24

function isNearBottom(viewport: HTMLDivElement) {
  const distanceToBottom =
    viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight

  return distanceToBottom <= AUTO_SCROLL_THRESHOLD_PX
}

type LogsPanelProps = {
  logs: GatewayLogEntry[]
}

export function LogsPanel({ logs }: LogsPanelProps) {
  const { t } = useTranslation()
  const scrollAreaRef = useRef<HTMLDivElement>(null)
  const viewportRef = useRef<HTMLDivElement | null>(null)
  const shouldStickToBottomRef = useRef(true)
  const visibleLogs = useMemo(
    () =>
      logs
        .map(({ id, line }) => ({ id, line: normalizeUserVisibleLog(line) }))
        .filter(({ line }) => Boolean(line)),
    [logs],
  )

  useLayoutEffect(() => {
    const scrollArea = scrollAreaRef.current
    const viewport = scrollArea?.querySelector<HTMLDivElement>(
      '[data-slot="scroll-area-viewport"]',
    )

    if (!viewport) {
      return
    }

    viewportRef.current = viewport

    const updateStickToBottom = () => {
      shouldStickToBottomRef.current = isNearBottom(viewport)
    }

    viewport.scrollTop = viewport.scrollHeight
    viewport.addEventListener("scroll", updateStickToBottom)

    return () => {
      viewport.removeEventListener("scroll", updateStickToBottom)
      if (viewportRef.current === viewport) {
        viewportRef.current = null
      }
    }
  }, [])

  useLayoutEffect(() => {
    const viewport = viewportRef.current
    if (!viewport) {
      return
    }

    if (shouldStickToBottomRef.current) {
      viewport.scrollTop = viewport.scrollHeight
    }
  }, [logs])

  return (
    <div className="relative flex-1 overflow-hidden rounded-lg border border-zinc-800 bg-zinc-950 text-zinc-100">
      <ScrollArea ref={scrollAreaRef} className="h-full">
        <div className="relative p-4 font-mono text-sm leading-relaxed [overflow-anchor:none]">
          {visibleLogs.length === 0 ? (
            <div className="text-zinc-500 italic">{t("pages.logs.empty")}</div>
          ) : (
            visibleLogs.map((log) => (
              <PlainLogLine key={log.id} id={log.id} line={log.line} />
            ))
          )}
        </div>
      </ScrollArea>
    </div>
  )
}
