// Package plugin implements the `everestctl plugin` command tree.
package plugin

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openeverest/everestctl-poc/internal/backend"
	"github.com/openeverest/everestctl-poc/internal/cli/output"
)

type Deps struct {
	Backend backend.Backend
	Out     io.Writer
	Err     io.Writer
	Format  func() output.Format
}

func NewCommand(d Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage OpenEverest plugins",
	}
	cmd.AddCommand(newListCmd(d), newInstallCmd(d), newConfigureCmd(d))
	return cmd
}

func newListCmd(d Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available and installed plugins",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ps, err := d.Backend.ListPlugins(cmd.Context())
			if err != nil {
				return err
			}
			return output.Render(d.Out, d.Format(), pluginList(ps))
		},
	}
}

func newInstallCmd(d Deps) *cobra.Command {
	var version string
	c := &cobra.Command{
		Use:   "install NAME",
		Short: "Install a plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := d.Backend.InstallPlugin(cmd.Context(), args[0], version)
			if err != nil {
				return err
			}
			fmt.Fprintf(d.Out, "plugin %q installed (version=%s)\n", p.Name, p.Version)
			return nil
		},
	}
	c.Flags().StringVar(&version, "version", "", "specific version to install")
	return c
}

func newConfigureCmd(d Deps) *cobra.Command {
	var kvs []string
	c := &cobra.Command{
		Use:   "configure NAME --set key=value [--set key=value ...]",
		Short: "Update a plugin's configuration",
		Args:  cobra.ExactArgs(1),
		Example: `  everestctl plugin configure backup-s3 --set bucket=my-backups --set region=eu-west-1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			kv, err := parseKVs(kvs)
			if err != nil {
				return err
			}
			if len(kv) == 0 {
				return fmt.Errorf("at least one --set key=value is required")
			}
			p, err := d.Backend.ConfigurePlugin(cmd.Context(), args[0], kv)
			if err != nil {
				return err
			}
			fmt.Fprintf(d.Out, "plugin %q configured (%d keys set)\n", p.Name, len(kv))
			return nil
		},
	}
	c.Flags().StringArrayVar(&kvs, "set", nil, "configuration entry as key=value (repeatable)")
	return c
}

func parseKVs(in []string) (map[string]string, error) {
	out := map[string]string{}
	for _, s := range in {
		k, v, ok := strings.Cut(s, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid --set %q (want key=value)", s)
		}
		out[k] = v
	}
	return out, nil
}

type pluginList []backend.Plugin

func (pluginList) TableHeader() []string {
	return []string{"NAME", "VERSION", "INSTALLED", "DESCRIPTION"}
}

func (l pluginList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, p := range l {
		installed := "no"
		if p.Installed {
			installed = "yes"
		}
		rows = append(rows, []string{p.Name, p.Version, installed, p.Description})
	}
	return rows
}
