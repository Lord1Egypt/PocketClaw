package api

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func toolSupportByName(items []toolSupportItem) map[string]toolSupportItem {
	byName := make(map[string]toolSupportItem, len(items))
	for _, item := range items {
		byName[item.Name] = item
	}
	return byName
}

// A. PocketClaw Android offers no host-bus tools. They cannot work there — an
//
//	unrooted phone exposes no /dev/i2c-*, /dev/spidev* or /dev/tty* to an app
//	UID — so three permanently unusable switches are worse than none.
func TestAndroidToolLibraryOffersNoHardwareTools(t *testing.T) {
	cfg := config.DefaultConfig()
	got := toolSupportByName(buildToolSupportForPlatform(cfg, "android"))

	for _, name := range []string{"i2c", "spi", "serial"} {
		if _, present := got[name]; present {
			t.Errorf("%s is offered on Android; it cannot work there", name)
		}
	}
}

// The same holds when a persisted config claims they are enabled: the entry is
// gone, not merely shown as disabled.
func TestAndroidToolLibraryHidesHardwareToolsEvenWhenConfigEnablesThem(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.I2C.Enabled = true
	cfg.Tools.SPI.Enabled = true
	cfg.Tools.Serial.Enabled = true

	got := toolSupportByName(buildToolSupportForPlatform(cfg, "android"))
	for _, name := range []string{"i2c", "spi", "serial"} {
		if _, present := got[name]; present {
			t.Errorf("%s reappeared on Android because the config enabled it", name)
		}
	}
}

// B. This is a hardware-category rule, not a "hide disabled tools" rule. An
//
//	unrelated tool that is off must still be listed so the user can turn it on.
func TestAndroidToolLibraryKeepsUnrelatedDisabledTools(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.Tools.IsToolEnabled("spawn_status") {
		t.Fatal("spawn_status is expected to default to disabled; pick another control")
	}

	got := toolSupportByName(buildToolSupportForPlatform(cfg, "android"))
	item, present := got["spawn_status"]
	if !present {
		t.Fatal("an unrelated disabled tool disappeared from the Android Tool Library")
	}
	if item.Status != "disabled" {
		t.Fatalf("spawn_status status = %q, want disabled", item.Status)
	}

	// And the catalog is otherwise intact: only the three hardware entries go.
	full := toolSupportByName(buildToolSupportForPlatform(cfg, "linux"))
	if len(full)-len(got) != 3 {
		t.Fatalf("Android dropped %d entries, want exactly 3", len(full)-len(got))
	}
}

// C. Every other platform keeps today's behaviour exactly, including the
//
//	reason codes the UI renders.
func TestNonAndroidHardwareToolStatusesAreUnchanged(t *testing.T) {
	enabled := config.DefaultConfig()
	enabled.Tools.I2C.Enabled = true
	enabled.Tools.SPI.Enabled = true
	enabled.Tools.Serial.Enabled = true

	for _, test := range []struct {
		goos      string
		hwStatus  string
		hwReason  string
		serStatus string
		serReason string
	}{
		{goos: "linux", hwStatus: "enabled", serStatus: "enabled"},
		{goos: "darwin", hwStatus: "blocked", hwReason: "requires_linux", serStatus: "enabled"},
		{goos: "windows", hwStatus: "blocked", hwReason: "requires_linux", serStatus: "enabled"},
		{
			goos: "freebsd", hwStatus: "blocked", hwReason: "requires_linux",
			serStatus: "blocked", serReason: "requires_serial_platform",
		},
	} {
		t.Run(test.goos, func(t *testing.T) {
			got := toolSupportByName(buildToolSupportForPlatform(enabled, test.goos))

			for _, name := range []string{"i2c", "spi"} {
				item, present := got[name]
				if !present {
					t.Fatalf("%s must still be offered on %s", name, test.goos)
				}
				if item.Status != test.hwStatus || item.ReasonCode != test.hwReason {
					t.Fatalf("%s on %s = %q/%q, want %q/%q",
						name, test.goos, item.Status, item.ReasonCode, test.hwStatus, test.hwReason)
				}
			}

			serial, present := got["serial"]
			if !present {
				t.Fatalf("serial must still be offered on %s", test.goos)
			}
			if serial.Status != test.serStatus || serial.ReasonCode != test.serReason {
				t.Fatalf("serial on %s = %q/%q, want %q/%q",
					test.goos, serial.Status, serial.ReasonCode, test.serStatus, test.serReason)
			}
		})
	}

	// Disabled-by-default still reads as disabled off Android, not blocked.
	off := toolSupportByName(buildToolSupportForPlatform(config.DefaultConfig(), "linux"))
	for _, name := range []string{"i2c", "spi", "serial"} {
		if off[name].Status != "disabled" {
			t.Fatalf("%s on linux with default config = %q, want disabled", name, off[name].Status)
		}
	}
}
