// Package db implements the `everestctl db` command tree.
package db

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openeverest/everestctl-poc/internal/backend"
	"github.com/openeverest/everestctl-poc/internal/cli/output"
)

// Deps groups what every db command needs. Passing it explicitly (instead
// of using package-level globals) keeps the commands trivially testable.
type Deps struct {
	Backend backend.Backend
	Out     io.Writer
	Err     io.Writer
	Format  func() output.Format // resolves the current --output flag
}

// NewCommand returns the `db` parent command and wires all subcommands.
func NewCommand(d Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Manage OpenEverest-managed databases",
		Long: `Manage databases (PostgreSQL, MySQL, MongoDB) provisioned through
OpenEverest. Lets you list, inspect, create, delete and tail logs without
leaving the terminal.`,
	}
	cmd.AddCommand(
		newListCmd(d),
		newGetCmd(d),
		newCreateCmd(d),
		newDeleteCmd(d),
		newLogsCmd(d),
	)
	return cmd
}

func newListCmd(d Deps) *cobra.Command {
	var namespace string
	c := &cobra.Command{
		Use:   "list",
		Short: "List databases",
		Example: `  everestctl db list
  everestctl db list -n production -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dbs, err := d.Backend.ListDatabases(cmd.Context(), namespace)
			if err != nil {
				return err
			}
			return output.Render(d.Out, d.Format(), databaseList(dbs))
		},
	}
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "filter by namespace")
	return c
}

func newGetCmd(d Deps) *cobra.Command {
	var namespace string
	c := &cobra.Command{
		Use:   "get NAME",
		Short: "Show details of a single database",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return completeDatabaseNames(cmd.Context(), d.Backend, namespace), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := d.Backend.GetDatabase(cmd.Context(), namespace, args[0])
			if err != nil {
				return err
			}
			return output.Render(d.Out, d.Format(), databaseList{*db})
		},
	}
	c.Flags().StringVarP(&namespace, "namespace", "n", "default", "namespace of the database")
	return c
}

func newCreateCmd(d Deps) *cobra.Command {
	var (
		namespace string
		engine    string
		version   string
		replicas  int
		cluster   string
	)
	c := &cobra.Command{
		Use:   "create NAME",
		Short: "Provision a new database",
		Example: `  everestctl db create orders-pg --engine postgresql --version 16.2 --replicas 3
  everestctl db create cache-mongo --engine mongodb -n staging`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, err := backend.ParseEngine(engine)
			if err != nil {
				return err
			}
			db, err := d.Backend.CreateDatabase(cmd.Context(), backend.DBCreateOptions{
				Name: args[0], Namespace: namespace, Engine: eng,
				Version: version, Replicas: replicas, Cluster: cluster,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(d.Out, "database %q created (status=%s)\n", db.Name, db.Status)
			return nil
		},
	}
	c.Flags().StringVarP(&namespace, "namespace", "n", "default", "target namespace")
	c.Flags().StringVar(&engine, "engine", "", "engine: postgresql|mysql|mongodb (required)")
	c.Flags().StringVar(&version, "version", "", "engine version")
	c.Flags().IntVar(&replicas, "replicas", 1, "replica count")
	c.Flags().StringVar(&cluster, "cluster", "", "target Kubernetes cluster (defaults to 'local')")
	_ = c.MarkFlagRequired("engine")
	_ = c.RegisterFlagCompletionFunc("engine", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		out := make([]string, 0, len(backend.SupportedEngines()))
		for _, e := range backend.SupportedEngines() {
			out = append(out, string(e))
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	})
	return c
}

func newDeleteCmd(d Deps) *cobra.Command {
	var (
		namespace string
		yes       bool
	)
	c := &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete a database",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return completeDatabaseNames(cmd.Context(), d.Backend, namespace), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Fprintf(d.Err, "refusing to delete %q without --yes\n", args[0])
				return fmt.Errorf("confirmation required")
			}
			if err := d.Backend.DeleteDatabase(cmd.Context(), namespace, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(d.Out, "database %q deleted\n", args[0])
			return nil
		},
	}
	c.Flags().StringVarP(&namespace, "namespace", "n", "default", "namespace of the database")
	c.Flags().BoolVar(&yes, "yes", false, "confirm deletion (required, no interactive prompt in POC)")
	return c
}

func newLogsCmd(d Deps) *cobra.Command {
	var (
		namespace string
		follow    bool
	)
	c := &cobra.Command{
		Use:   "logs NAME",
		Short: "Stream database logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ch := make(chan backend.LogLine, 16)
			errCh := make(chan error, 1)
			go func() { errCh <- d.Backend.StreamLogs(cmd.Context(), namespace, args[0], follow, ch) }()
			for line := range ch {
				fmt.Fprintf(d.Out, "%s  %-5s  %s\n",
					line.Timestamp.Format("15:04:05"), line.Level, line.Message)
			}
			return <-errCh
		},
	}
	c.Flags().StringVarP(&namespace, "namespace", "n", "default", "namespace of the database")
	c.Flags().BoolVarP(&follow, "follow", "f", false, "follow the log stream")
	return c
}

// databaseList is a slice-with-methods so we can render it as a table.
type databaseList []backend.Database

func (databaseList) TableHeader() []string {
	return []string{"NAMESPACE", "NAME", "ENGINE", "VERSION", "REPLICAS", "CLUSTER", "STATUS"}
}

func (l databaseList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, d := range l {
		rows = append(rows, []string{
			d.Namespace, d.Name, string(d.Engine), d.Version,
			fmt.Sprintf("%d", d.Replicas), d.Cluster, string(d.Status),
		})
	}
	return rows
}

// completeDatabaseNames powers shell completion for db names. Errors are
// swallowed because completion must never block the user's keystroke.
func completeDatabaseNames(ctx context.Context, b backend.Backend, ns string) []string {
	dbs, err := b.ListDatabases(ctx, ns)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(dbs))
	for _, d := range dbs {
		out = append(out, d.Name)
	}
	return out
}

// silences an unused-import warning if strings ever becomes unreferenced.
var _ = strings.TrimSpace
