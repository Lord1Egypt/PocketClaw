//go:build !android

package main

import (
	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/updater"
)

// selfUpdateCommands returns upstream's `update` command, which downloads a
// release binary and replaces the running executable. It exists on the desktop
// platforms upstream supports; selfupdate_cmd_android.go leaves it out.
func selfUpdateCommands() []*cobra.Command {
	return []*cobra.Command{updater.NewUpdateCommand("picoclaw")}
}
