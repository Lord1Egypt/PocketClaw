package picoclaw

import "embed"

// OnboardWorkspace embeds the default onboarding workspace template.
//
// Keeping this embed at the module root lets us source files directly from the
// tracked `workspace/` tree, instead of relying on a generated copy inside
// `cmd/...` that may be absent in clean checkouts and CI lint runs.
//
//go:embed workspace
var OnboardWorkspace embed.FS
