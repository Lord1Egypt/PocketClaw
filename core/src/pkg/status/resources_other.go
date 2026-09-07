//go:build !linux

package status

// readResources reports zeroes on platforms without a Linux /proc.
//
// Zero means "not available here", and the Status screen renders it as
// unavailable rather than as a measurement. PocketClaw's shipping target is
// Android, so this fallback exists to keep desktop builds and tests compiling
// rather than to serve a supported configuration.
func readResources() Resources {
	return Resources{}
}
