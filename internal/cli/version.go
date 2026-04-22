package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aegis-sec/aegis/internal/version"
)

func cmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print aegis version and schema version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("aegis %s  schema=%d  commit=%s  built=%s\n",
				version.Version, version.SchemaVersion, version.Commit, version.Date)
		},
	}
}
