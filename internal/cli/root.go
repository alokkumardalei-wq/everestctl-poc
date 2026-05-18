// Package cli wires the everestctl POC root command together.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/everestctl-poc/internal/backend"
	"github.com/openeverest/everestctl-poc/internal/cli/cluster"
	"github.com/openeverest/everestctl-poc/internal/cli/completion"
	"github.com/openeverest/everestctl-poc/internal/cli/db"
	"github.com/openeverest/everestctl-poc/internal/cli/plugin"
	"github.com/openeverest/everestctl-poc/internal/cli/output"
)

// Version is overridden via -ldflags at release time.
var Version = "0.0.1-poc"

// NewRoot builds the root command tree. The backend and writers are
// injected so tests can swap them.
func NewRoot(b backend.Backend, out, errOut io.Writer) *cobra.Command {
	var formatStr string
	resolveFormat := func() output.Format {
		f, err := output.ParseFormat(formatStr)
		if err != nil {
			fmt.Fprintln(errOut, err)
			os.Exit(2)
		}
		return f
	}

	root := &cobra.Command{
		Use:          "everestctl",
		Short:        "OpenEverest command-line interface",
		SilenceUsage: true,
		Long: `everestctl is the command-line interface for OpenEverest.

This POC demonstrates the proposed command surface for the
"Transform everestctl into a Powerful Database Management CLI"
project (CNCF OpenEverest, 2026 Term 2). It uses an in-memory
backend so every subcommand is runnable without a Kubernetes
cluster.`,
		Version: Version,
	}
	root.SetOut(out)
	root.SetErr(errOut)
	root.PersistentFlags().StringVarP(&formatStr, "output", "o", "table", "output format: table|json|yaml")

	dbDeps := db.Deps{Backend: b, Out: out, Err: errOut, Format: resolveFormat}
	clDeps := cluster.Deps{Backend: b, Out: out, Err: errOut, Format: resolveFormat}
	plDeps := plugin.Deps{Backend: b, Out: out, Err: errOut, Format: resolveFormat}

	root.AddCommand(
		db.NewCommand(dbDeps),
		cluster.NewCommand(clDeps),
		plugin.NewCommand(plDeps),
		completion.NewCommand(out),
	)
	return root
}
