package main

import "github.com/spf13/cobra"

// selfUpdateCommands is empty on Android. PocketClaw's Core is packaged in the
// APK and updated only by installing a new APK: a binary that downloads and
// swaps its own executable has no place there, and official F-Droid does not
// allow an app to fetch executables outside its own update channel. Leaving the
// command out also keeps pkg/updater and its self-replacement library out of
// the Android link entirely (TestAndroidCoreLinksNoSelfUpdater).
func selfUpdateCommands() []*cobra.Command {
	return nil
}
