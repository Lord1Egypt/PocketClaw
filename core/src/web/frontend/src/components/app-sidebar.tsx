import {
  IconChevronRight,
  IconChevronsDown,
  IconChevronsUp,
} from "@tabler/icons-react"
import { Link, useRouterState } from "@tanstack/react-router"
import * as React from "react"
import { useTranslation } from "react-i18next"

import {
  type NavGroup,
  buildNavGroups,
} from "@/components/app-navigation"
import { PocketClawLockup } from "@/components/brand/pocketclaw-mark"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
  useSidebar,
} from "@/components/ui/sidebar"
import { useSidebarChannels } from "@/hooks/use-sidebar-channels"

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const routerState = useRouterState()
  const { i18n, t } = useTranslation()
  const { isMobile, setOpenMobile } = useSidebar()
  const currentPath = routerState.location.pathname
  const {
    channelItems,
    hasMoreChannels,
    showAllChannels,
    toggleShowAllChannels,
  } = useSidebarChannels({
    language: (i18n.resolvedLanguage ?? i18n.language ?? "").toLowerCase(),
    t,
  })

  const handleNavItemClick = React.useCallback(() => {
    if (isMobile) {
      setOpenMobile(false)
    }
  }, [isMobile, setOpenMobile])

  // v0.2.0 ships no Credentials entry; see app-navigation.ts for why and for
  // what replaces it. The structure is data so that absence is testable.
  const navGroups: NavGroup[] = React.useMemo(
    () => buildNavGroups(channelItems),
    [channelItems],
  )

  return (
    <Sidebar
      {...props}
      className="bg-pc-surface-1 border-e-border border-e"
    >
      {/* A product mark belongs in the sidebar header of a sidebar layout,
          which leaves the top bar for state and actions. */}
      <div className="border-b-border flex h-14 shrink-0 items-center border-b px-4">
        <Link
          to="/"
          onClick={handleNavItemClick}
          className="text-foreground flex items-center rounded-md"
        >
          <PocketClawLockup label={t("header.logoAlt")} />
        </Link>
      </div>
      <SidebarContent className="bg-pc-surface-1 pt-2">
        {navGroups.map((group) => (
          <Collapsible
            key={group.label}
            defaultOpen={group.defaultOpen}
            className="group/collapsible mb-1"
          >
            <SidebarGroup className="px-2 py-0">
              <SidebarGroupLabel asChild>
                <CollapsibleTrigger className="text-pc-faint hover:text-pc-muted pc-micro flex w-full cursor-pointer items-center justify-between rounded-md px-2 py-1.5 transition-colors">
                  <span>{t(group.label)}</span>
                  <IconChevronRight className="size-3.5 opacity-50 transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                </CollapsibleTrigger>
              </SidebarGroupLabel>
              <CollapsibleContent>
                <SidebarGroupContent className="pt-1">
                  <SidebarMenu>
                    {group.items.map((item) => {
                      const isActive =
                        currentPath === item.url ||
                        (item.url !== "/" &&
                          currentPath.startsWith(`${item.url}/`))
                      return (
                        <SidebarMenuItem key={item.title}>
                          <SidebarMenuButton
                            asChild
                            isActive={isActive}
                            onClick={handleNavItemClick}
                            data-tour={
                              item.url === "/models" ? "models-nav" : undefined
                            }
                            className={`h-9 rounded-md border-s-2 px-3 transition-colors ${
                              isActive
                                ? "border-s-pc-claw bg-pc-claw-soft text-pc-text font-medium"
                                : "hover:bg-pc-surface-2 border-s-transparent text-pc-muted"
                            }`}
                          >
                            <Link to={item.url}>
                              <item.icon
                                className={`size-4.5 ${isActive ? "text-pc-claw" : ""}`}
                              />
                              <span>
                                {item.translateTitle === false
                                  ? item.title
                                  : t(item.title)}
                              </span>
                            </Link>
                          </SidebarMenuButton>
                        </SidebarMenuItem>
                      )
                    })}
                    {group.isChannelsGroup && hasMoreChannels && (
                      <SidebarMenuItem key="channels-more-toggle">
                        <SidebarMenuButton
                          onClick={toggleShowAllChannels}
                          className="text-pc-muted hover:bg-pc-surface-2 h-9 rounded-md border-s-2 border-s-transparent px-3"
                        >
                          {showAllChannels ? (
                            <IconChevronsUp className="size-4.5" />
                          ) : (
                            <IconChevronsDown className="size-4.5" />
                          )}
                          <span>
                            {showAllChannels
                              ? t("navigation.show_less_channels")
                              : t("navigation.show_more_channels")}
                          </span>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    )}
                  </SidebarMenu>
                </SidebarGroupContent>
              </CollapsibleContent>
            </SidebarGroup>
          </Collapsible>
        ))}
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  )
}
