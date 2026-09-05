package coresource

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Live channel reconciliation only runs when the gateway's config watcher is
// armed, and Core leaves gateway.hot_reload off by default because a server
// deployment should not reload itself. PocketClaw Android wants the opposite:
// saving a channel setting in the Dashboard has to apply without the user
// stopping and starting the Gateway by hand.
//
// That difference is expressed at the host boundary, in the environment the
// Android service hands the managed Core process. These two guards pin both
// halves of the decision, because nothing else would notice either one being
// undone: dropping the env entry makes every channel setting silently require a
// manual restart again, and flipping the Core default would change behaviour for
// every non-Android deployment.

var androidHotReloadEnvPattern = regexp.MustCompile(
	`"PICOCLAW_GATEWAY_HOT_RELOAD"\s*to\s*"true"`,
)

// TestAndroidManagedGatewayEnablesHotReload proves a normal PocketClaw Android
// install starts its managed Gateway with hot reload on, without the user
// editing gateway.hot_reload.
func TestAndroidManagedGatewayEnablesHotReload(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout")
	}

	servicePath := filepath.Join(root, "android", "app", "src", "main", "kotlin",
		"com", "lord1egypt", "pocketclaw", "service", "PicoClawService.kt")
	service, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("cannot read the Android service: %v", err)
	}

	if !androidHotReloadEnvPattern.Match(service) {
		t.Error("the Android managed Gateway environment does not set " +
			"PICOCLAW_GATEWAY_HOT_RELOAD=true; live channel reconciliation would " +
			"not run and every channel setting would need a manual Gateway restart")
	}

	// The entry has to be in the map the gateway process actually inherits.
	if !strings.Contains(string(service), "fun buildEnvironment(") {
		t.Fatal("buildEnvironment is gone; this guard is checking the wrong place")
	}
	envStart := strings.Index(string(service), "val environment = mutableMapOf(")
	if envStart < 0 {
		t.Fatal("the managed-process environment map is gone; this guard is stale")
	}
	envEnd := strings.Index(string(service)[envStart:], "\n            )")
	if envEnd < 0 {
		t.Fatal("cannot delimit the environment map; this guard is stale")
	}
	if !androidHotReloadEnvPattern.MatchString(string(service)[envStart : envStart+envEnd]) {
		t.Error("PICOCLAW_GATEWAY_HOT_RELOAD is set somewhere other than the " +
			"managed-process environment map, so the Gateway may not inherit it")
	}
}

// TestCoreHotReloadDefaultStaysOff keeps the Android decision from leaking into
// Core. Enabling it globally would change behaviour for every server deployment,
// which is exactly what the host-boundary override exists to avoid.
func TestCoreHotReloadDefaultStaysOff(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout")
	}

	defaults, err := os.ReadFile(filepath.Join(root, "core", "src", "pkg", "config", "defaults.go"))
	if err != nil {
		t.Fatalf("cannot read Core defaults: %v", err)
	}
	if !strings.Contains(string(defaults), "HotReload: false") {
		t.Error("the Core-wide gateway.hot_reload default is no longer false; " +
			"PocketClaw Android enables hot reload at its own host boundary and " +
			"must not change it for other deployments")
	}
}
