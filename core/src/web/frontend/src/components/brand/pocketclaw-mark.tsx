/**
 * The PocketClaw mark and its lockups.
 *
 * A geometric mark on a 24x24 grid, 2-unit stroke, one colour: an open
 * container with two claw arms rising out of it. The negative space between
 * the arms is the aperture — the same bracket the active navigation row, the
 * focus ring and the assistant message rail are drawn with.
 *
 * The mouth carries a short inward lip and the arms are shorter than they
 * first were. Without the lip the container ended in two bare stroke caps, so
 * below about 24px the four verticals read at equal weight and the whole mark
 * looked like a crown or a fork rather than a pocket. Measured against the
 * first geometry at 16, 20, 24, 32, 48 and 64px, the lip is what makes the
 * container close optically at every one of them. The concept is unchanged.
 *
 * It is inline SVG rather than a file so it inherits `currentColor` and is
 * correct in both themes from one asset, with no flash of the wrong logo. The
 * raster favicons in `public/` are generated from this same geometry by
 * `scripts/generate-brand-assets.py`.
 */
import { cn } from "@/lib/utils"

interface MarkProps {
  className?: string
  /** Stroke weight in grid units. 2 at UI sizes, lighter when drawn large. */
  strokeWidth?: number
}

export function PocketClawMark({ className, strokeWidth = 2 }: MarkProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={strokeWidth}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={cn("size-6", className)}
      aria-hidden="true"
      focusable="false"
    >
      {/* the pocket: a squared container whose mouth turns inward, so the
          opening reads as a lip rather than as two bare stroke ends */}
      <path d="M7.5 12H4v8h16v-8h-3.5" />
      {/* the claw: two arms rising out of the mouth and splaying outward */}
      <path d="M10 13.5V10L6.5 4.5" />
      <path d="M14 13.5V10l3.5-5.5" />
    </svg>
  )
}

interface LockupProps {
  className?: string
  /** Accessible name. The mark itself is decorative inside the lockup. */
  label: string
}

/**
 * Mark plus wordmark, for the sidebar header and the desktop toolbar. The
 * wordmark is live text in Inter SemiBold rather than an image, so it scales
 * with the user's text size and stays legible at any zoom.
 */
export function PocketClawLockup({ className, label }: LockupProps) {
  return (
    <span
      className={cn("inline-flex items-center gap-2", className)}
      role="img"
      aria-label={label}
    >
      <PocketClawMark className="text-pc-claw size-6 shrink-0" />
      <span className="text-[0.9375rem] font-semibold tracking-[-0.02em]">
        PocketClaw
      </span>
    </span>
  )
}
