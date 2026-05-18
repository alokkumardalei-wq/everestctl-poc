// Package backend defines the abstraction the CLI uses to talk to
// OpenEverest. The POC ships an in-memory implementation; a future
// version will swap this for a real Kubernetes / Everest API client
// without changing the command layer.
package backend

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrInvalid       = errors.New("invalid argument")
)

// DBCreateOptions are the inputs accepted by Backend.CreateDatabase.
type DBCreateOptions struct {
	Name      string
	Namespace string
	Engine    Engine
	Version   string
	Replicas  int
	Cluster   string
}

// Backend is the seam between command code and the underlying system.
// All command implementations depend only on this interface, which
// keeps the CLI testable and makes a future real backend a drop-in.
type Backend interface {
	// Databases
	ListDatabases(ctx context.Context, namespace string) ([]Database, error)
	GetDatabase(ctx context.Context, namespace, name string) (*Database, error)
	CreateDatabase(ctx context.Context, opts DBCreateOptions) (*Database, error)
	DeleteDatabase(ctx context.Context, namespace, name string) error
	StreamLogs(ctx context.Context, namespace, name string, follow bool, out chan<- LogLine) error

	// Clusters
	ListClusters(ctx context.Context) ([]Cluster, error)
	RegisterCluster(ctx context.Context, c Cluster) (*Cluster, error)
	ClusterStatus(ctx context.Context, name string) (*Cluster, error)

	// Plugins
	ListPlugins(ctx context.Context) ([]Plugin, error)
	InstallPlugin(ctx context.Context, name, version string) (*Plugin, error)
	ConfigurePlugin(ctx context.Context, name string, kv map[string]string) (*Plugin, error)
}

// ParseEngine maps a CLI string to a known Engine value.
func ParseEngine(s string) (Engine, error) {
	switch s {
	case string(EnginePostgreSQL), "pg", "postgres":
		return EnginePostgreSQL, nil
	case string(EngineMySQL):
		return EngineMySQL, nil
	case string(EngineMongoDB), "mongo":
		return EngineMongoDB, nil
	default:
		return "", fmt.Errorf("%w: unknown engine %q", ErrInvalid, s)
	}
}
