import type { ReactNode } from "react"

import { SidebarTrigger } from "@/components/ui/sidebar"
import { cn } from "@/lib/utils"

interface PageHeaderProps {
  title: string
  titleExtra?: ReactNode
  children?: ReactNode
  className?: string
}

export function PageHeader({
  title,
  titleExtra,
  children,
  className,
}: PageHeaderProps) {
  return (
    <div
      className={cn(
        // Wraps instead of clipping: at phone width the title plus two
        // actions (Models) do not fit one row (PC-DEF-089).
        "z-40 flex min-h-14 shrink-0 flex-wrap items-center justify-between gap-x-4 gap-y-2 px-6 pt-2",
        className,
      )}
    >
      <div className="flex items-center gap-4">
        {/* Desktop only. Here the sidebar genuinely is a collapsible panel,
            so this keeps the panel glyph and the panel wording; the mobile
            control in the app header is the one that navigates. */}
        <SidebarTrigger className="border-pc-line bg-pc-surface-1 text-pc-muted hover:bg-pc-surface-3 hover:text-pc-text hidden size-9 rounded-md border sm:flex [&>svg]:size-5" />
        <h2 className="text-pc-text text-[1.375rem] font-semibold tracking-[-0.01em]">
          {title}
        </h2>
        {titleExtra}
      </div>
      {children && (
        <div className="ms-auto flex flex-wrap items-center gap-2">
          {children}
        </div>
      )}
    </div>
  )
}
