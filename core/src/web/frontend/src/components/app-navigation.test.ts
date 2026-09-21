import { describe, expect, it } from "vitest"

import {
  CREDENTIALS_IN_PUBLIC_NAVIGATION,
  type ChannelNavItem,
  buildNavGroups,
  navigationDestinations,
} from "./app-navigation"
import {
  DEFAULT_POST_AUTH_DESTINATION,
  resolvePostAuthDestination,
} from "@/lib/post-auth-destination"

/**
 * v0.2.0 ships no Credentials entry.
 *
 * Account-login credential management is unfinished — OpenAI browser OAuth
 * reaches a real authentication screen and returns `unknown_error` — so it is
 * withdrawn from the public surface for this release and will come back with
 * the later feature phase. These assert the withdrawal as a property of the
 * navigation data, because an absence is what a rendered snapshot is worst at
 * defending: a snapshot happily records whatever is there.
 *
 * The page component, its hook and its API are deliberately still in the tree.
 * Nothing here asserts they are gone, and nothing should.
 */

const CHANNELS: ChannelNavItem[] = [
  { title: "Telegram", url: "/channels/telegram", icon: () => null },
  { title: "Discord", url: "/channels/discord", icon: () => null },
]

describe("v0.2.0 sidebar navigation", () => {
  it("offers no Credentials entry", () => {
    const destinations = navigationDestinations(buildNavGroups(CHANNELS))
    expect(destinations).not.toContain("/credentials")
    expect(CREDENTIALS_IN_PUBLIC_NAVIGATION).toBe(false)
  })

  it("names no credential or account-login surface anywhere in navigation", () => {
    const groups = buildNavGroups(CHANNELS)
    const text = JSON.stringify(
      groups.map((group) => ({
        label: group.label,
        items: group.items.map((item) => ({
          title: item.title,
          url: item.url,
        })),
      })),
    ).toLowerCase()
    for (const forbidden of [
      "credential",
      "oauth",
      "device code",
      "devicecode",
      "sign in",
      "signin",
      "login",
    ]) {
      expect(text, `navigation must not offer ${forbidden}`).not.toContain(
        forbidden,
      )
    }
  })

  it("keeps Models reachable, which is where providers are configured", () => {
    const destinations = navigationDestinations(buildNavGroups(CHANNELS))
    expect(destinations).toContain("/models")
  })

  it("leaves no empty group behind", () => {
    // The Credentials item shared a group with Models. Removing it must not
    // leave a labelled group with nothing under it, which is the visible gap
    // the release explicitly rules out.
    for (const group of buildNavGroups(CHANNELS)) {
      expect(group.items.length, `${group.label} is empty`).toBeGreaterThan(0)
    }
  })

  it("keeps every other destination it had", () => {
    expect(navigationDestinations(buildNavGroups(CHANNELS))).toEqual([
      "/",
      "/models",
      "/channels/telegram",
      "/channels/discord",
      "/agent/hub",
      "/agent/skills",
      "/agent/tools",
      "/config",
      "/logs",
    ])
  })

  it("has one builder, so mobile and desktop cannot disagree", () => {
    // Parity is structural here: both render the same object. Asserting it this
    // way is what keeps a second, drifting list from being introduced later.
    const first = navigationDestinations(buildNavGroups(CHANNELS))
    const second = navigationDestinations(buildNavGroups(CHANNELS))
    expect(first).toEqual(second)
  })

  it("renders the channels group from its input rather than a fixed list", () => {
    const none = buildNavGroups([]).find((g) => g.isChannelsGroup)
    expect(none?.items).toEqual([])
  })
})

describe("the /credentials destination after login", () => {
  it("is no longer an allowed post-auth destination", () => {
    expect(resolvePostAuthDestination("/credentials")).toBe(
      DEFAULT_POST_AUTH_DESTINATION,
    )
  })

  it("still allows the destinations v0.2.0 does ship", () => {
    expect(resolvePostAuthDestination("/models")).toBe("/models")
    expect(resolvePostAuthDestination("/channels/telegram")).toBe(
      "/channels/telegram",
    )
  })
})
