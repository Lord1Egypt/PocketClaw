package onboard

import (
	"fmt"

	"github.com/spf13/cobra"

	picoclaw "github.com/sipeed/picoclaw"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

var embeddedFiles = picoclaw.OnboardWorkspace

func NewOnboardCommand() *cobra.Command {
	var encrypt bool

	cmd := &cobra.Command{
		Use:     "onboard",
		Aliases: []string{"o"},
		Short:   "Initialize picoclaw configuration and workspace",
		// Run without subcommands → original onboard flow
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				onboard(encrypt)
			} else {
				_ = cmd.Help()
			}
		},
	}

	cmd.Flags().BoolVar(&encrypt, "enc", false,
		"Enable credential encryption (generates SSH key and prompts for passphrase)")

	cmd.AddCommand(&cobra.Command{
		Use:    "ensure-workspace",
		Short:  "Seed missing bundled workspace files without replacing user files",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.LoadConfig(internal.GetConfigPath())
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			return copyMissingEmbeddedToTarget(cfg.WorkspacePath())
		},
	})

	return cmd
}
