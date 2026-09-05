package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bundledHardwareFrontmatter is the frontmatter PocketClaw ships in
// workspace/skills/hardware/SKILL.md. The identity check reads the same two
// fields the loader parses, so a reword upstream must fail loudly here rather
// than silently stop hiding the skill — see
// TestFilterStillMatchesTheShippedBundledSkill.
const bundledHardwareFrontmatter = `---
name: hardware
description: Read and control I2C and SPI peripherals on Sipeed boards (LicheeRV Nano, MaixCAM, NanoKVM).
homepage: https://wiki.sipeed.com/hardware/en/lichee/RV_Nano/1_intro.html
metadata: {"nanobot":{"emoji":"🔧","requires":{"tools":["i2c","spi"]}}}
---

# Hardware (I2C / SPI)

Use the ` + "`i2c`" + ` and ` + "`spi`" + ` tools.
`

func writeSkill(t *testing.T, dir, name, content string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// listedSkillNamesOn runs real discovery for a chosen platform.
func listedSkillNamesOn(loader *SkillsLoader, goos string) map[string]bool {
	names := make(map[string]bool)
	for _, s := range loader.listSkillsForPlatform(goos) {
		names[s.Name] = true
	}
	return names
}

// E. An installation upgraded from an earlier PocketClaw already has the skill
//
//	on disk. Declining to copy it during onboarding would do nothing for those
//	users, which is why the filter lives at listing.
func TestUpgradedWorkspaceDoesNotExposeTheBundledHardwareSkillOnAndroid(t *testing.T) {
	workspace := t.TempDir()
	skillsDir := filepath.Join(workspace, "skills")
	writeSkill(t, skillsDir, "hardware", bundledHardwareFrontmatter)
	writeSkill(t, skillsDir, "summarize", "---\nname: summarize\ndescription: Summarize a document.\n---\n\n# Summarize\n")

	loader := NewSkillsLoader(workspace, "", "")

	// The file is on disk, exactly as it is for an installation upgraded from an
	// earlier PocketClaw.
	if !listedSkillNamesOn(loader, "linux")["hardware"] {
		t.Fatal("the bundled hardware skill is not on disk; this test would prove nothing")
	}

	onAndroid := listedSkillNamesOn(loader, "android")
	if onAndroid["hardware"] {
		t.Error("an upgraded workspace still advertises the bundled hardware skill on Android")
	}
	if !onAndroid["summarize"] {
		t.Error("an unrelated bundled skill disappeared on Android")
	}
}

// The rendered summary is what actually reaches the prompt, so assert on the
// text rather than only on the listing that feeds it.
func TestSkillsSummaryOmitsTheBundledHardwareSkillOnAndroid(t *testing.T) {
	workspace := t.TempDir()
	skillsDir := filepath.Join(workspace, "skills")
	writeSkill(t, skillsDir, "hardware", bundledHardwareFrontmatter)
	writeSkill(t, skillsDir, "summarize", "---\nname: summarize\ndescription: Summarize a document.\n---\n\n# Summarize\n")

	loader := NewSkillsLoader(workspace, "", "")

	var androidSummary strings.Builder
	for _, info := range loader.listSkillsForPlatform("android") {
		androidSummary.WriteString(info.Name + " " + info.Description + "\n")
	}
	rendered := strings.ToLower(androidSummary.String())

	if strings.Contains(rendered, "sipeed") || strings.Contains(rendered, "i2c") {
		t.Fatalf("the Android skills summary still advertises host-bus hardware:\n%s",
			androidSummary.String())
	}
	if !strings.Contains(rendered, "summarize") {
		t.Fatal("unrelated skills must still reach the summary")
	}
}

// F. A fresh workspace is covered by the same rule.
func TestFreshWorkspaceDoesNotExposeTheBundledHardwareSkillOnAndroid(t *testing.T) {
	workspace := t.TempDir()
	writeSkill(t, filepath.Join(workspace, "skills"), "hardware", bundledHardwareFrontmatter)

	loader := NewSkillsLoader(workspace, "", "")
	if listedSkillNamesOn(loader, "android")["hardware"] {
		t.Fatal("a fresh install would still advertise the bundled hardware skill on Android")
	}
}

// G. Every other platform keeps the skill: it is genuinely useful on the Linux
//
//	boards upstream targets.
func TestNonAndroidPlatformsKeepTheBundledHardwareSkill(t *testing.T) {
	workspace := t.TempDir()
	writeSkill(t, filepath.Join(workspace, "skills"), "hardware", bundledHardwareFrontmatter)
	loader := NewSkillsLoader(workspace, "", "")

	for _, goos := range []string{"linux", "darwin", "windows", "freebsd"} {
		if !listedSkillNamesOn(loader, goos)["hardware"] {
			t.Errorf("the bundled hardware skill was hidden on %s", goos)
		}
	}
}

// H. The filter is scoped to the skill PocketClaw ships, not to a name. A user's
//
//	own skills are untouched, including one they chose to call "hardware".
func TestUnrelatedSkillsAreUnaffectedOnAndroid(t *testing.T) {
	workspace := t.TempDir()
	skillsDir := filepath.Join(workspace, "skills")
	writeSkill(t, skillsDir, "hardware", bundledHardwareFrontmatter)
	writeSkill(t, skillsDir, "my-notes", "---\nname: my-notes\ndescription: Keep my notes tidy.\n---\n\n# Notes\n")
	writeSkill(t, skillsDir, "workshop",
		"---\nname: workshop\ndescription: My own hardware bench notes for the garage.\n---\n\n# Bench\n")

	loader := NewSkillsLoader(workspace, "", "")
	onAndroid := listedSkillNamesOn(loader, "android")
	if onAndroid["hardware"] {
		t.Error("the bundled hardware skill must be hidden on Android")
	}
	for _, name := range []string{"my-notes", "workshop"} {
		if !onAndroid[name] {
			t.Errorf("unrelated skill %q was hidden on Android", name)
		}
	}

	// A skill a user wrote and named "hardware" does not carry the bundled
	// skill's board list, so it stays visible.
	userAuthored := SkillInfo{
		Name:        "hardware",
		Description: "Notes about my 3D printer wiring.",
	}
	if skillHiddenOnPlatform("android", userAuthored) {
		t.Error("a user-authored skill named hardware was hidden on Android")
	}
}

// The identity check reads the shipped file's own frontmatter. If that skill is
// reworded upstream, this fails rather than the filter silently lapsing.
func TestFilterStillMatchesTheShippedBundledSkill(t *testing.T) {
	root := repoRootForSkills()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout")
	}
	shipped := filepath.Join(root, "core", "src", "workspace", "skills", "hardware", "SKILL.md")
	if _, err := os.Stat(shipped); err != nil {
		t.Skipf("bundled hardware skill not present: %v", err)
	}

	workspace := t.TempDir()
	content, err := os.ReadFile(shipped)
	if err != nil {
		t.Fatalf("read shipped skill: %v", err)
	}
	writeSkill(t, filepath.Join(workspace, "skills"), "hardware", string(content))

	loader := NewSkillsLoader(workspace, "", "")
	if listedSkillNamesOn(loader, "android")["hardware"] {
		t.Fatal("the shipped hardware skill no longer matches the Android filter; " +
			"its name or description changed and the filter must be updated with it")
	}
}

func repoRootForSkills() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "pubspec.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
