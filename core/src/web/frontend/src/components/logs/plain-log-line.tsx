import { useMemo } from "react"

import { wrapPlainTextLogLine } from "@/lib/plain-text-log"

type PlainLogLineProps = {
  line: string
  wrapColumns: number
}

export function PlainLogLine({ line, wrapColumns }: PlainLogLineProps) {
  const wrapped = useMemo(
    () => wrapPlainTextLogLine(line, wrapColumns),
    [line, wrapColumns],
  )

  return <div className="break-normal whitespace-pre-wrap">{wrapped}</div>
}
