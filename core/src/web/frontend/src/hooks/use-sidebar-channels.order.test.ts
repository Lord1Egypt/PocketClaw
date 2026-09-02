/**
 * Channel ordering in the sidebar.
 *
 * The previous physical test failed because the experimental Agent Channel sat
 * at the bottom of ~17 unconfigured entries, behind the show-more toggle, and
 * the tester never found it — they exercised the Self-Chat launcher instead and
 * reasonably read the silence as a broken transport. These pin the fix.
 */
import { describe, expect, it } from "vitest"

import { CHANNEL_IMPORTANCE_TAIL } from "@/hooks/use-sidebar-channels"

describe("channel ordering", () => {
  it("puts the WhatsApp Agent Channel directly beside WhatsApp Self-Chat", () => {
    const selfChat = CHANNEL_IMPORTANCE_TAIL.indexOf("whatsapp_self_chat")
    const agent = CHANNEL_IMPORTANCE_TAIL.indexOf("whatsapp_agent")

    expect(selfChat).toBeGreaterThanOrEqual(0)
    expect(agent).toBeGreaterThanOrEqual(0)
    expect(agent - selfChat).toBe(1)
  })

  it("does not restore the retired WhatsApp cards", () => {
    // These asked a phone user for a bridge URL and a session store path.
    expect(CHANNEL_IMPORTANCE_TAIL).not.toContain("whatsapp")
    expect(CHANNEL_IMPORTANCE_TAIL).not.toContain("whatsapp_native")
  })
})
