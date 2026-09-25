package sources

import (
	"go/build"
	"testing"
)

// PC-DEF-080: the Android Core must compile the inert monitor, never the one
// that shells out to udevadm, even though GOOS=android satisfies "linux".
func TestAndroidCompilesTheInertUSBMonitor(t *testing.T) {
	android := build.Default
	android.GOOS = "android"
	android.GOARCH = "arm64"

	for file, want := range map[string]bool{"usb_linux.go": false, "usb_stub.go": true} {
		got, err := android.MatchFile(".", file)
		if err != nil {
			t.Fatalf("MatchFile(%s): %v", file, err)
		}
		if got != want {
			t.Fatalf("android build includes %s = %v, want %v", file, got, want)
		}
	}
}
