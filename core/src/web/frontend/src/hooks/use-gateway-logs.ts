import { useAtomValue } from "jotai"
import { useEffect, useRef, useState } from "react"

import { clearGatewayLogs, getGatewayLogs } from "@/api/gateway"
import { gatewayAtom } from "@/store/gateway"

export type GatewayLogEntry = {
  id: string
  line: string
}

function createLogEntries(
  lines: string[],
  runId: number,
  total: number,
): GatewayLogEntry[] {
  const firstOffset = Math.max(total - lines.length, 0)

  return lines.map((line, index) => ({
    id: `${runId}:${firstOffset + index}`,
    line,
  }))
}

export function useGatewayLogs() {
  const [logs, setLogs] = useState<GatewayLogEntry[]>([])
  const [clearing, setClearing] = useState(false)
  const logOffsetRef = useRef(0)
  const logRunIdRef = useRef(-1)
  const syncTokenRef = useRef(0)

  const gateway = useAtomValue(gatewayAtom)

  const clearLogs = async () => {
    setClearing(true)
    try {
      const data = await clearGatewayLogs()
      syncTokenRef.current += 1
      setLogs([])
      logOffsetRef.current = data.log_total ?? 0
      if (data.log_run_id !== undefined) {
        logRunIdRef.current = data.log_run_id
      }
    } catch {
      // Ignore clear failures silently to avoid noisy transient errors.
    } finally {
      setClearing(false)
    }
  }

  useEffect(() => {
    let mounted = true
    let timeout: ReturnType<typeof setTimeout>

    const fetchLogs = async () => {
      if (
        !mounted ||
        !["running", "starting", "restarting", "stopping"].includes(
          gateway.status,
        )
      ) {
        if (mounted) {
          timeout = setTimeout(fetchLogs, 1000)
        }
        return
      }

      try {
        const requestToken = syncTokenRef.current
        const requestOffset = logOffsetRef.current
        const requestRunId = logRunIdRef.current
        const data = await getGatewayLogs({
          log_offset: requestOffset,
          log_run_id: requestRunId,
        })

        if (!mounted || requestToken !== syncTokenRef.current) {
          return
        }

        const responseRunId = data.log_run_id ?? requestRunId

        if (responseRunId !== requestRunId) {
          logRunIdRef.current = responseRunId
          logOffsetRef.current = 0
          if (data.logs) {
            const total = data.log_total ?? data.logs.length
            setLogs(createLogEntries(data.logs, responseRunId, total))
            logOffsetRef.current = total
          }
        } else if (data.logs && data.logs.length > 0) {
          const total =
            data.log_total ?? logOffsetRef.current + data.logs.length
          const nextLogs = createLogEntries(data.logs, responseRunId, total)
          setLogs((prev) => [...prev, ...nextLogs])
          logOffsetRef.current = total
        }
      } catch {
        // Ignore simple fetch errors during polling.
      } finally {
        if (mounted) {
          timeout = setTimeout(fetchLogs, 1000)
        }
      }
    }

    fetchLogs()

    return () => {
      mounted = false
      clearTimeout(timeout)
    }
  }, [gateway.status])

  return {
    clearLogs,
    clearing,
    logs,
  }
}
