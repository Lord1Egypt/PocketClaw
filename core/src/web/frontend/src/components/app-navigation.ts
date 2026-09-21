import {
  IconAtom,
  IconListDetails,
  IconMessageCircle,
  IconSearch,
  IconSettings,
  IconSparkles,
  IconTools,
} from "@tabler/icons-react"
import type * as React from "react"

/**
 * What the Dashboard's sidebar offers, as data.
 *
 * Kept as a pure builder so the navigation surface can be asserted directly
 * rather than inferred from a rendered tree. The thing that most needs
 * asserting is what is *absent*: v0.2.0 ships no Credentials entry, and an
 * absence is exactly what a snapshot of a rendered sidebar is worst at
 * defending.
 */

export interface NavItem {
  title: string
  url: string
  icon: React.ComponentType<{ className?: string }>
  translateTitle?: boolean
}

export interface NavGroup {
  label: string
  defaultOpen: boolean
  items: NavItem[]
  isChannelsGroup?: boolean
}

/** One channel row, as `useSidebarChannels` produces it. */
export interface ChannelNavItem {
  title: string
  url: string
  icon: React.ComponentType<{ className?: string }>
}

/**
 * Account-login credential management is **not part of the v0.2.0 surface**.
 *
 * The page and its API remain in the tree — see `components/credentials/` — and
 * they are the starting point for the later feature phase (Google account
 * login, Claude and ChatGPT/Codex subscription login, and the finished
 * credential management UI). What that work does not yet have is a finished
 * flow: OpenAI browser OAuth reaches a real authentication screen and comes back
 * `unknown_error`. A stable release does not show a user a door that does not
 * open, so the entry point is withdrawn rather than decorated with a disabled
 * button or a "coming soon" label — both of which are still a door.
 *
 * Providers configured by **API key** are unaffected and keep their own,
 * finished path: Models → Add/Manage provider. That is where key entry,
 * rotation and Set Default live, and none of it routes through this page.
 */
export const CREDENTIALS_IN_PUBLIC_NAVIGATION = false

/**
 * Builds the sidebar's groups for a given set of channel rows.
 *
 * Pure: the same channels always produce the same navigation, on every client.
 * Mobile and desktop render this one structure, so parity is a property of
 * there being a single builder rather than of two lists agreeing.
 */
export function buildNavGroups(channelItems: ChannelNavItem[]): NavGroup[] {
  return [
    {
      label: "navigation.chat",
      defaultOpen: true,
      items: [
        {
          title: "navigation.chat",
          url: "/",
          icon: IconMessageCircle,
          translateTitle: true,
        },
      ],
    },
    {
      label: "navigation.model_group",
      defaultOpen: true,
      items: [
        {
          title: "navigation.models",
          url: "/models",
          icon: IconAtom,
          translateTitle: true,
        },
      ],
    },
    {
      label: "navigation.channels_group",
      defaultOpen: true,
      items: channelItems.map((item) => ({
        title: item.title,
        url: item.url,
        icon: item.icon,
        translateTitle: false,
      })),
      isChannelsGroup: true,
    },
    {
      label: "navigation.agent_group",
      defaultOpen: true,
      items: [
        {
          title: "navigation.hub",
          url: "/agent/hub",
          icon: IconSearch,
          translateTitle: true,
        },
        {
          title: "navigation.skills",
          url: "/agent/skills",
          icon: IconSparkles,
          translateTitle: true,
        },
        {
          title: "navigation.tools",
          url: "/agent/tools",
          icon: IconTools,
          translateTitle: true,
        },
      ],
    },
    {
      label: "navigation.services",
      defaultOpen: true,
      items: [
        {
          title: "navigation.config",
          url: "/config",
          icon: IconSettings,
          translateTitle: true,
        },
        {
          title: "navigation.logs",
          url: "/logs",
          icon: IconListDetails,
          translateTitle: true,
        },
      ],
    },
  ]
}

/** Every destination the sidebar can navigate to, flattened. */
export function navigationDestinations(groups: NavGroup[]): string[] {
  return groups.flatMap((group) => group.items.map((item) => item.url))
}
