package backend

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// MemoryBackend is an in-memory Backend implementation used by the POC.
// It is concurrency-safe so command tests can run in parallel.
type MemoryBackend struct {
	mu       sync.RWMutex
	dbs      map[string]Database // key = namespace/name
	clusters map[string]Cluster
	plugins  map[string]Plugin
	now      func() time.Time
}

// NewMemoryBackend returns a MemoryBackend seeded with a small,
// realistic set of sample resources so `list` commands have something
// to display out of the box.
func NewMemoryBackend() *MemoryBackend {
	m := &MemoryBackend{
		dbs:      map[string]Database{},
		clusters: map[string]Cluster{},
		plugins:  map[string]Plugin{},
		now:      time.Now,
	}
	m.seed()
	return m
}

func (m *MemoryBackend) seed() {
	t := m.now().Add(-24 * time.Hour)
	m.clusters["local"] = Cluster{
		Name: "local", Endpoint: "https://127.0.0.1:6443", Context: "kind-local",
		Version: "v1.30.0", Status: StatusReady, RegisteredAt: t,
	}
	m.dbs["default/orders-pg"] = Database{
		Name: "orders-pg", Namespace: "default", Engine: EnginePostgreSQL,
		Version: "16.2", Replicas: 3, Cluster: "local", Status: StatusReady, CreatedAt: t,
	}
	m.dbs["default/sessions-mongo"] = Database{
		Name: "sessions-mongo", Namespace: "default", Engine: EngineMongoDB,
		Version: "7.0", Replicas: 3, Cluster: "local", Status: StatusReady, CreatedAt: t,
	}
	m.plugins["backup-s3"] = Plugin{
		Name: "backup-s3", Version: "0.4.1",
		Description: "Scheduled backups to S3-compatible storage", Installed: true,
		Config: map[string]string{"bucket": "everest-backups", "region": "us-east-1"},
	}
	m.plugins["pmm"] = Plugin{
		Name: "pmm", Version: "2.41.0",
		Description: "Percona Monitoring and Management integration",
	}
	m.plugins["external-dns"] = Plugin{
		Name: "external-dns", Version: "0.14.0",
		Description: "Auto-publish database endpoints to DNS",
	}
}

func dbKey(ns, name string) string { return ns + "/" + name }

func (m *MemoryBackend) ListDatabases(_ context.Context, namespace string) ([]Database, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Database, 0, len(m.dbs))
	for _, d := range m.dbs {
		if namespace != "" && d.Namespace != namespace {
			continue
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func (m *MemoryBackend) GetDatabase(_ context.Context, namespace, name string) (*Database, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.dbs[dbKey(namespace, name)]
	if !ok {
		return nil, fmt.Errorf("%w: database %s/%s", ErrNotFound, namespace, name)
	}
	return &d, nil
}

func (m *MemoryBackend) CreateDatabase(_ context.Context, opts DBCreateOptions) (*Database, error) {
	if opts.Name == "" || opts.Namespace == "" {
		return nil, fmt.Errorf("%w: name and namespace are required", ErrInvalid)
	}
	if opts.Replicas <= 0 {
		opts.Replicas = 1
	}
	if opts.Cluster == "" {
		opts.Cluster = "local"
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.clusters[opts.Cluster]; !ok {
		return nil, fmt.Errorf("%w: target cluster %q is not registered", ErrInvalid, opts.Cluster)
	}
	key := dbKey(opts.Namespace, opts.Name)
	if _, ok := m.dbs[key]; ok {
		return nil, fmt.Errorf("%w: database %s/%s", ErrAlreadyExists, opts.Namespace, opts.Name)
	}
	d := Database{
		Name: opts.Name, Namespace: opts.Namespace, Engine: opts.Engine,
		Version: opts.Version, Replicas: opts.Replicas, Cluster: opts.Cluster,
		Status: StatusPending, CreatedAt: m.now(),
	}
	m.dbs[key] = d
	return &d, nil
}

func (m *MemoryBackend) DeleteDatabase(_ context.Context, namespace, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := dbKey(namespace, name)
	if _, ok := m.dbs[key]; !ok {
		return fmt.Errorf("%w: database %s/%s", ErrNotFound, namespace, name)
	}
	delete(m.dbs, key)
	return nil
}

// StreamLogs synthesizes a small log stream so the `logs` command has
// something to demo. In a real backend this would tail K8s pod logs.
func (m *MemoryBackend) StreamLogs(ctx context.Context, namespace, name string, follow bool, out chan<- LogLine) error {
	if _, err := m.GetDatabase(ctx, namespace, name); err != nil {
		return err
	}
	defer close(out)
	base := []LogLine{
		{Level: "INFO", Message: "starting database instance"},
		{Level: "INFO", Message: "primary elected"},
		{Level: "INFO", Message: "accepting connections on :5432"},
	}
	for _, l := range base {
		l.Timestamp = m.now()
		select {
		case <-ctx.Done():
			return nil
		case out <- l:
		}
	}
	if !follow {
		return nil
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case t := <-ticker.C:
			out <- LogLine{Timestamp: t, Level: "INFO", Message: "heartbeat ok"}
		}
	}
}

func (m *MemoryBackend) ListClusters(_ context.Context) ([]Cluster, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Cluster, 0, len(m.clusters))
	for _, c := range m.clusters {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *MemoryBackend) RegisterCluster(_ context.Context, c Cluster) (*Cluster, error) {
	if c.Name == "" || c.Endpoint == "" {
		return nil, fmt.Errorf("%w: name and endpoint are required", ErrInvalid)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.clusters[c.Name]; ok {
		return nil, fmt.Errorf("%w: cluster %q", ErrAlreadyExists, c.Name)
	}
	if c.Status == "" {
		c.Status = StatusReady
	}
	c.RegisteredAt = m.now()
	m.clusters[c.Name] = c
	return &c, nil
}

func (m *MemoryBackend) ClusterStatus(_ context.Context, name string) (*Cluster, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.clusters[name]
	if !ok {
		return nil, fmt.Errorf("%w: cluster %q", ErrNotFound, name)
	}
	return &c, nil
}

func (m *MemoryBackend) ListPlugins(_ context.Context) ([]Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *MemoryBackend) InstallPlugin(_ context.Context, name, version string) (*Plugin, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.plugins[name]
	if !ok {
		return nil, fmt.Errorf("%w: plugin %q", ErrNotFound, name)
	}
	if p.Installed {
		return nil, fmt.Errorf("%w: plugin %q already installed", ErrAlreadyExists, name)
	}
	if version != "" {
		p.Version = version
	}
	p.Installed = true
	m.plugins[name] = p
	return &p, nil
}

func (m *MemoryBackend) ConfigurePlugin(_ context.Context, name string, kv map[string]string) (*Plugin, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.plugins[name]
	if !ok {
		return nil, fmt.Errorf("%w: plugin %q", ErrNotFound, name)
	}
	if !p.Installed {
		return nil, fmt.Errorf("%w: plugin %q must be installed before configuring", ErrInvalid, name)
	}
	if p.Config == nil {
		p.Config = map[string]string{}
	}
	for k, v := range kv {
		p.Config[k] = v
	}
	m.plugins[name] = p
	return &p, nil
}
