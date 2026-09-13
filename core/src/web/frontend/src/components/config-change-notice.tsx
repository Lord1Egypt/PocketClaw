import {
  IconAlertCircle,
  IconDeviceFloppy,
  IconRefresh,
} from "@tabler/icons-react"

import { cn } from "@/lib/utils"

interface ConfigChangeNoticeProps {
  kind: "save" | "restart"
  title: string
  description?: string
  className?: string
}

export function ConfigChangeNotice({
  kind,
  title,
  description,
  className,
}: ConfigChangeNoticeProps) {
  const Icon =
    kind === "restart"
      ? IconRefresh
      : kind === "save"
        ? IconDeviceFloppy
        : IconAlertCircle

  return (
    <div
      // The background is the soft (14% alpha) warning token, never the solid
      // one: `bg-pc-warning` with `text-pc-warning` paints the label in the
      // same colour as the surface behind it, which is a filled amber rectangle
      // with no readable content. PC-DEF-033.
      className={cn(
        "flex items-start gap-3 rounded-lg border px-3 py-2 text-sm",
        "border-pc-warning/30 bg-pc-warning-soft text-pc-warning",
        className,
      )}
    >
      <Icon className="mt-0.5 size-4 shrink-0" />
      <div className="min-w-0">
        <p className="font-medium">{title}</p>
        {description && (
          <p className="mt-0.5 text-xs/5 opacity-85">{description}</p>
        )}
      </div>
    </div>
  )
}
