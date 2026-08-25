import type { ProviderCatalogEntry } from "./provider-registry"

interface ProviderIconProps {
  provider: Pick<ProviderCatalogEntry, "key" | "label" | "iconSlug" | "domain">
  size?: "sm" | "md"
}

// Provider marks are rendered locally from the provider label. PocketClaw does
// not fetch logos from a CDN or favicon service at runtime: those requests
// disclose which providers a user has configured, and they leave a broken mark
// on a device that is offline or behind a restrictive network.
export function ProviderIcon({ provider, size = "sm" }: ProviderIconProps) {
  const initial = provider.label.trim().charAt(0).toUpperCase() || "?"
  const sizeClass =
    size === "md" ? "size-8 text-sm rounded-md" : "size-4 text-[9px] rounded-sm"

  return (
    <span
      aria-hidden="true"
      className={`inline-flex shrink-0 items-center justify-center border border-black/10 bg-white font-semibold text-black/70 dark:border-white/20 dark:bg-white/10 dark:text-white/80 ${sizeClass}`}
    >
      {initial}
    </span>
  )
}
