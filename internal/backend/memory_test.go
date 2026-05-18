package backend

import (
	"context"
	"errors"
	"testing"
	"time"
)

func newTestBackend(t *testing.T) *MemoryBackend {
	t.Helper()
	m := NewMemoryBackend()
	// Pin time so assertions are deterministic.
	m.now = func() time.Time { return time.Unix(1_700_000_000, 0).UTC() }
	return m
}

func TestListDatabases_Seeded(t *testing.T) {
	t.Parallel()
	m := newTestBackend(t)
	dbs, err := m.ListDatabases(context.Background(), "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(dbs) != 2 {
		t.Fatalf("want 2 seeded dbs, got %d", len(dbs))
	}
}

func TestListDatabases_NamespaceFilter(t *testing.T) {
	t.Parallel()
	m := newTestBackend(t)
	dbs, _ := m.ListDatabases(context.Background(), "no-such-ns")
	if len(dbs) != 0 {
		t.Fatalf("want 0 dbs in unknown ns, got %d", len(dbs))
	}
}

func TestCreateAndGetDatabase(t *testing.T) {
	t.Parallel()
	m := newTestBackend(t)
	ctx := context.Background()
	d, err := m.CreateDatabase(ctx, DBCreateOptions{
		Name: "x", Namespace: "default", Engine: EngineMySQL, Version: "8.0", Replicas: 1, Cluster: "local",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if d.Status != StatusPending {
		t.Fatalf("new db should be pending, got %s", d.Status)
	}
	got, err := m.GetDatabase(ctx, "default", "x")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Engine != EngineMySQL {
		t.Fatalf("engine mismatch: %s", got.Engine)
	}
}

func TestCreateDatabase_DuplicateRejected(t *testing.T) {
	t.Parallel()
	m := newTestBackend(t)
	ctx := context.Background()
	opts := DBCreateOptions{Name: "dup", Namespace: "default", Engine: EnginePostgreSQL, Cluster: "local"}
	if _, err := m.CreateDatabase(ctx, opts); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := m.CreateDatabase(ctx, opts)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("want ErrAlreadyExists, got %v", err)
	}
}

func TestCreateDatabase_UnknownCluster(t *testing.T) {
	t.Parallel()
	m := newTestBackend(t)
	_, err := m.CreateDatabase(context.Background(), DBCreateOptions{
		Name: "x", Namespace: "default", Engine: EnginePostgreSQL, Cluster: "ghost",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid for unknown cluster, got %v", err)
	}
}

func TestDeleteDatabase_NotFound(t *testing.T) {
	t.Parallel()
	m := newTestBackend(t)
	err := m.DeleteDatabase(context.Background(), "default", "nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestParseEngine(t *testing.T) {
	t.Parallel()
	cases := map[string]Engine{
		"postgresql": EnginePostgreSQL,
		"pg":         EnginePostgreSQL,
		"postgres":   EnginePostgreSQL,
		"mysql":      EngineMySQL,
		"mongodb":    EngineMongoDB,
		"mongo":      EngineMongoDB,
	}
	for in, want := range cases {
		got, err := ParseEngine(in)
		if err != nil || got != want {
			t.Errorf("ParseEngine(%q) = (%v, %v), want (%v, nil)", in, got, err, want)
		}
	}
	if _, err := ParseEngine("oracle"); !errors.Is(err, ErrInvalid) {
		t.Errorf("want ErrInvalid for unknown engine, got %v", err)
	}
}

func TestRegisterAndStatusCluster(t *testing.T) {
	t.Parallel()
	m := newTestBackend(t)
	ctx := context.Background()
	_, err := m.RegisterCluster(ctx, Cluster{Name: "prod", Endpoint: "https://k.example.com"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	c, err := m.ClusterStatus(ctx, "prod")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if c.Status != StatusReady {
		t.Fatalf("want Ready, got %s", c.Status)
	}
}

func TestPluginLifecycle(t *testing.T) {
	t.Parallel()
	m := newTestBackend(t)
	ctx := context.Background()
	if _, err := m.ConfigurePlugin(ctx, "pmm", map[string]string{"x": "y"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("configure-before-install must fail, got %v", err)
	}
	if _, err := m.InstallPlugin(ctx, "pmm", "2.42.0"); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := m.InstallPlugin(ctx, "pmm", ""); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("double install must fail, got %v", err)
	}
	p, err := m.ConfigurePlugin(ctx, "pmm", map[string]string{"endpoint": "pmm.local"})
	if err != nil {
		t.Fatalf("configure: %v", err)
	}
	if p.Config["endpoint"] != "pmm.local" {
		t.Fatalf("config not persisted: %#v", p.Config)
	}
}

func TestStreamLogs_OneShot(t *testing.T) {
	t.Parallel()
	m := newTestBackend(t)
	ch := make(chan LogLine, 8)
	err := m.StreamLogs(context.Background(), "default", "orders-pg", false, ch)
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	got := 0
	for range ch {
		got++
	}
	if got == 0 {
		t.Fatalf("expected at least one log line")
	}
}
