package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// PocketClaw-managed guidance.
//
// A workspace file is the user's. The seeded templates are only a starting
// point, and after the first seed PocketClaw never rewrites them — an install
// created a year ago keeps the AGENT.md it was given. That is the right
// ownership rule and it has one consequence: anything the *product* needs the
// model to know cannot live in those files, because it would reach only the
// installs that happened to be seeded after it was written. The device
// validated on 2026-09-08 still carried an AGENT.md predating the Managed
// Runtime, so that agent had never been told the Managed Runtime exists.
//
// So guidance splits by owner rather than by topic:
//
//   - user-owned: who the assistant is, how it should sound, what the user
//     wants from it — AGENT.md, SOUL.md, USER.md, and memory. Seeded once,
//     never touched again, and never merged.
//   - PocketClaw-managed: how this build's own capabilities behave. It ships
//     inside the binary as a prompt part, so upgrading the app upgrades the
//     guidance, on every install, with no file to migrate and nothing of the
//     user's to overwrite.
//
// This file holds the second kind.

// managedGuidanceVersion identifies the revision of the text below. It is
// stamped into the seeded bootstrap metadata so a later version can tell what a
// workspace was seeded against without diffing prose.
const managedGuidanceVersion = 1

// managedRuntimeGuidance is what every agent is told about this build's
// Managed Runtime, regardless of when its workspace was created.
//
// COMPATIBILITY: the tool name, the directory wording and the `runtime` action
// names below are user-visible product surface.
// RECHECK AFTER FULL NAMESPACE MIGRATION.
const managedRuntimeGuidance = "# Managed Runtime\n" + `
This device provides a fixed catalog of verified command-line tools through the
` + "`runtime`" + ` tool. ` + "`action=list`" + ` is the authority on what exists here.

- Before concluding that a command is missing, check ` + "`action=list`" + `. A tool absent
  from PATH may still be in the catalog — typically ` + "`git`" + `, ` + "`gh`" + `, ` + "`rg`" + `, ` + "`jq`" + `,
  ` + "`sqlite3`" + ` and ` + "`curl`" + `, alongside the standard system tools.
- Nothing can be installed. Executables run only from the app package or the
  system image, both fixed when PocketClaw was installed. "Unavailable" is
  final: solve the task with what is there, or tell the user plainly that this
  device cannot do it. Never download a binary, never make a file executable.
- Never modify anything under the managed runtime directories.
- A Skill describing a tool is not evidence that the tool exists here. Check
  ` + "`action=list`" + `, and say so accurately when it does not.
- Runtime tools run without a shell. Arguments pass through verbatim, so pipes,
  redirection, globs and ` + "`$(...)`" + ` do nothing; use several calls rather than one
  composed command line.
- Never put a credential in an argument: no token inside a URL, no ` + "`-u`" + `, no
  ` + "`--password`" + `. PocketClaw supplies GitHub credentials to ` + "`git`" + ` and ` + "`gh`" + ` itself.
  If a command needs authentication and none is configured, say so.

These facts describe the build you are running and are authoritative for what
this device can do: where a workspace file says otherwise about PocketClaw's
tools or runtime, it is out of date. Persona, tone and the user's preferences
remain the workspace's.`

// managedGuidancePart is the prompt part carrying the text above.
func managedGuidancePart() PromptPart {
	return PromptPart{
		ID:      "capability.managed_runtime",
		Layer:   PromptLayerCapability,
		Slot:    PromptSlotTooling,
		Source:  PromptSource{ID: PromptSourceManagedGuidance, Name: "managed_runtime"},
		Title:   "managed runtime guidance",
		Content: managedRuntimeGuidance,
		Stable:  true,
		Cache:   PromptCacheEphemeral,
	}
}

// supersededManagedSections are sha256 digests of managed sections that earlier
// versions of PocketClaw seeded *into* AGENT.md, before this guidance moved into
// the binary.
//
// A workspace seeded by one of those versions still has the text in its own
// AGENT.md. We will not edit that file — it is the user's — but sending both
// copies would waste the tokens twice and, once this text is revised, put two
// versions of the same instructions in one prompt.
//
// Matching by digest is what makes that safe: it recognises only the exact text
// PocketClaw itself wrote. The moment a user changes one character of their copy
// it stops matching, their words are kept, and the managed part is added
// alongside them — because at that point it is their instruction, not ours.
var supersededManagedSections = map[string]int{
	// AGENT.md "## Managed Runtime" as seeded from the first Managed Runtime
	// release through 0.2.0+58. The literal it digests is pinned in
	// managed_guidance_legacy.go and must never be regenerated from the current
	// wording — it names bytes already on users' devices.
	//
	// COMPATIBILITY: RECHECK AFTER FULL NAMESPACE MIGRATION, and do not rewrite
	// the entry then either.
	"47b63011a55eaa659470f2ab09d05532e9942800848020a1cfe443e6f21aca76": 1,
}

// managedSectionHeadings are the headings whose sections may be superseded.
var managedSectionHeadings = []string{"## Managed Runtime"}

// stripSupersededManagedGuidance removes from a workspace file body any section
// that PocketClaw itself seeded and now delivers as a managed prompt part. It
// reports which versions it dropped, for logging.
//
// The file on disk is never touched. This only shapes what is sent to the model.
func stripSupersededManagedGuidance(body string) (string, []int) {
	var dropped []int
	result := body
	for _, heading := range managedSectionHeadings {
		section, rest, ok := cutManagedSection(result, heading)
		if !ok {
			continue
		}
		version, superseded := supersededManagedSections[digestOf(section)]
		if !superseded {
			continue
		}
		dropped = append(dropped, version)
		result = rest
	}
	return result, dropped
}

// cutManagedSection returns the trimmed section beginning at heading and the
// body with that section removed. A section runs until the next line starting
// with "## " or the end of the body.
func cutManagedSection(body, heading string) (section, rest string, ok bool) {
	lines := strings.Split(body, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimRight(line, " \t") == heading {
			start = i
			break
		}
	}
	if start < 0 {
		return "", body, false
	}

	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}

	section = strings.TrimSpace(strings.Join(lines[start:end], "\n"))
	remaining := append(append([]string{}, lines[:start]...), lines[end:]...)
	return section, strings.Join(remaining, "\n"), true
}

func digestOf(s string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(s)))
	return hex.EncodeToString(sum[:])
}

// ManagedGuidanceVersion exposes the revision of the managed guidance this
// binary carries, so the bootstrap record can state what a workspace was seeded
// alongside.
func ManagedGuidanceVersion() int { return managedGuidanceVersion }
