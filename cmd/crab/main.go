package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/crabkun/crab/internal/app"
	"github.com/crabkun/crab/internal/version"
	"github.com/spf13/cobra"
)

func main() {
	if err := newRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRoot() *cobra.Command {
	var configPath string

	root := &cobra.Command{
		Use:          "crab",
		Short:        "Intranet penetration over a single public TCP port",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return app.Run(ctx, configPath)
		},
	}

	root.PersistentFlags().StringVar(&configPath, "config", "config.json", "config file")

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "crab %s (commit %s, built %s)\n",
				version.Version, version.Commit, version.Date)
		},
	})

	return root
}
