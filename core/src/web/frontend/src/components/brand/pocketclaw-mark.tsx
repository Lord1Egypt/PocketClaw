/**
 * The PocketClaw mark and its lockups.
 *
 * A geometric mark on a 24x24 grid, 2-unit stroke, one colour: an open
 * container with two claw arms rising out of it. The negative space between
 * the arms is the aperture — the same bracket the active navigation row, the
 * focus ring and the assistant message rail are drawn with.
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
      {/* the pocket: a squared container, open at the top */}
      <path d="M4 12v8h16v-8" />
      {/* the claw: two arms rising and splaying outward */}
      <path d="M8.5 14V8L5.5 3.5" />
      <path d="M15.5 14V8l3-4.5" />
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
