// Package cluster implements the `everestctl cluster` command tree.
package cluster

import (
	"fmt"
	"io"

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
		Use:   "cluster",
		Short: "Manage Kubernetes clusters registered with OpenEverest",
	}
	cmd.AddCommand(newListCmd(d), newRegisterCmd(d), newStatusCmd(d))
	return cmd
}

func newListCmd(d Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered clusters",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cs, err := d.Backend.ListClusters(cmd.Context())
			if err != nil {
				return err
			}
			return output.Render(d.Out, d.Format(), clusterList(cs))
		},
	}
}

func newRegisterCmd(d Deps) *cobra.Command {
	var (
		endpoint string
		context  string
		version  string
	)
	c := &cobra.Command{
		Use:   "register NAME",
		Short: "Register a new Kubernetes cluster with OpenEverest",
		Args:  cobra.ExactArgs(1),
		Example: `  everestctl cluster register prod --endpoint https://k8s.prod.example.com --context prod`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := d.Backend.RegisterCluster(cmd.Context(), backend.Cluster{
				Name: args[0], Endpoint: endpoint, Context: context, Version: version,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(d.Out, "cluster %q registered (status=%s)\n", c.Name, c.Status)
			return nil
		},
	}
	c.Flags().StringVar(&endpoint, "endpoint", "", "Kubernetes API endpoint (required)")
	c.Flags().StringVar(&context, "context", "", "kubeconfig context name")
	c.Flags().StringVar(&version, "version", "", "Kubernetes version")
	_ = c.MarkFlagRequired("endpoint")
	return c
}

func newStatusCmd(d Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "status NAME",
		Short: "Show health/status of a registered cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := d.Backend.ClusterStatus(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return output.Render(d.Out, d.Format(), clusterList{*c})
		},
	}
}

type clusterList []backend.Cluster

func (clusterList) TableHeader() []string {
	return []string{"NAME", "ENDPOINT", "CONTEXT", "VERSION", "STATUS"}
}

func (l clusterList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, c := range l {
		rows = append(rows, []string{c.Name, c.Endpoint, c.Context, c.Version, string(c.Status)})
	}
	return rows
}
