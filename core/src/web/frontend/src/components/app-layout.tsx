import type { ReactNode } from "react"
import { Toaster } from "sonner"

import { AppHeader } from "@/components/app-header"
import { AppSidebar } from "@/components/app-sidebar"
import { TourGuide } from "@/components/tour/tour-guide"
import { SidebarProvider } from "@/components/ui/sidebar"
import { TooltipProvider } from "@/components/ui/tooltip"

export function AppLayout({ children }: { children: ReactNode }) {
  return (
    <TooltipProvider>
      <SidebarProvider className="flex h-dvh flex-col overflow-hidden">
        <div className="flex flex-1 overflow-hidden">
          {/* Full height on desktop, so the lockup in its header is the one
              persistent brand anchor and the top bar is purely utility. */}
          <AppSidebar />
          <div className="flex w-full min-w-0 flex-col overflow-hidden">
            <AppHeader />
            <main className="flex min-h-0 w-full max-w-full flex-1 flex-col overflow-hidden">
              {children}
            </main>
          </div>
        </div>
        <Toaster position="bottom-center" />
        <TourGuide />
      </SidebarProvider>
    </TooltipProvider>
  )
}
