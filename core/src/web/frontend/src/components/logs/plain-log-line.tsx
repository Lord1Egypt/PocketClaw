import { memo } from "react"

type PlainLogLineProps = {
  id: string
  line: string
}

export const PlainLogLine = memo(function PlainLogLine({
  id,
  line,
}: PlainLogLineProps) {
  return (
    <div
      className="[overflow-wrap:anywhere] whitespace-pre-wrap"
      data-log-entry-id={id}
    >
      {line}
    </div>
  )
})
