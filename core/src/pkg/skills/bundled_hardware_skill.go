package skills

import (
	"runtime"
	"strings"
)

// The bundled hardware skill teaches the model to drive I2C and SPI peripherals
// through the `i2c` and `spi` tools. Those tools are deliberately unavailable on
// PocketClaw Android — an unrooted phone exposes no /dev/i2c-* or /dev/spidev*
// nodes to an app UID — so listing the skill there advertises a capability the
// agent cannot exercise, and invites it to try.
//
// This is a narrow product filter, not a permission system. The skill's declared
// `requires.tools` metadata is not parsed anywhere in the tree, and teaching the
// loader to read and enforce it would change what every other skill means. The
// bundled skill is identified instead, precisely, by the two frontmatter fields
// the loader already reads.
const (
	bundledHardwareSkillName = "hardware"
	// The bundled skill names the boards it is written for. A skill a user
	// wrote themselves and happened to call "hardware" does not carry this, so
	// it stays visible.
	bundledHardwareSkillMarker = "sipeed"
)

// isBundledHardwareSkill reports whether a listed skill is the one PocketClaw
// ships for Sipeed boards.
func isBundledHardwareSkill(info SkillInfo) bool {
	if !strings.EqualFold(strings.TrimSpace(info.Name), bundledHardwareSkillName) {
		return false
	}
	return strings.Contains(strings.ToLower(info.Description), bundledHardwareSkillMarker)
}

// skillHiddenOnPlatform decides whether a skill is withheld from discovery on a
// given platform. The platform is a parameter so the rule can be tested for a
// platform the test is not running on.
//
// Hiding happens at listing rather than at install or onboarding on purpose: an
// installation upgraded from an earlier PocketClaw already has the skill on
// disk, and declining to copy it for new users would leave every existing user
// still advertising it.
func skillHiddenOnPlatform(goos string, info SkillInfo) bool {
	if goos != "android" {
		return false
	}
	return isBundledHardwareSkill(info)
}

func skillHidden(info SkillInfo) bool {
	return skillHiddenOnPlatform(runtime.GOOS, info)
}
