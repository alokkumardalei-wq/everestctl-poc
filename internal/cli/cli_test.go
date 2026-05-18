package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/openeverest/everestctl-poc/internal/backend"
)

// run executes the root command with args and returns stdout/stderr.
func run(t *testing.T, b backend.Backend, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	root := NewRoot(b, &out, &errOut)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return out.String(), errOut.String(), err
}

func TestDBList_Table(t *testing.T) {
	t.Parallel()
	b := backend.NewMemoryBackend()
	out, _, err := run(t, b, "db", "list")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, want := range []string{"NAME", "orders-pg", "sessions-mongo"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestDBList_JSON(t *testing.T) {
	t.Parallel()
	b := backend.NewMemoryBackend()
	out, _, err := run(t, b, "db", "list", "-o", "json")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var dbs []backend.Database
	if err := json.Unmarshal([]byte(out), &dbs); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, out)
	}
	if len(dbs) < 2 {
		t.Fatalf("want >=2 dbs, got %d", len(dbs))
	}
}

func TestDBCreate_RequiredEngine(t *testing.T) {
	t.Parallel()
	b := backend.NewMemoryBackend()
	_, _, err := run(t, b, "db", "create", "x")
	if err == nil {
		t.Fatalf("expected error when --engine is missing")
	}
}

func TestDBCreateGetDelete_RoundTrip(t *testing.T) {
	t.Parallel()
	b := backend.NewMemoryBackend()

	if _, _, err := run(t, b, "db", "create", "billing-pg",
		"--engine", "postgresql", "--version", "16.2", "--replicas", "2"); err != nil {
		t.Fatalf("create: %v", err)
	}
	out, _, err := run(t, b, "db", "get", "billing-pg", "-o", "json")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out, "billing-pg") {
		t.Fatalf("get output missing name:\n%s", out)
	}
	if _, _, err := run(t, b, "db", "delete", "billing-pg"); err == nil {
		t.Fatalf("delete should require --yes")
	}
	if _, _, err := run(t, b, "db", "delete", "billing-pg", "--yes"); err != nil {
		t.Fatalf("delete --yes: %v", err)
	}
	if _, _, err := run(t, b, "db", "get", "billing-pg"); err == nil {
		t.Fatalf("get after delete should fail")
	}
}

func TestClusterRegisterAndList(t *testing.T) {
	t.Parallel()
	b := backend.NewMemoryBackend()
	if _, _, err := run(t, b, "cluster", "register", "edge",
		"--endpoint", "https://edge.example.com"); err != nil {
		t.Fatalf("register: %v", err)
	}
	out, _, err := run(t, b, "cluster", "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "edge") {
		t.Fatalf("list missing registered cluster:\n%s", out)
	}
}

func TestPluginInstallConfigure(t *testing.T) {
	t.Parallel()
	b := backend.NewMemoryBackend()
	if _, _, err := run(t, b, "plugin", "install", "pmm"); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, _, err := run(t, b, "plugin", "configure", "pmm",
		"--set", "endpoint=pmm.local", "--set", "tls=true"); err != nil {
		t.Fatalf("configure: %v", err)
	}
	out, _, err := run(t, b, "plugin", "list", "-o", "yaml")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "pmm.local") {
		t.Fatalf("config not surfaced in yaml output:\n%s", out)
	}
}

func TestPluginList_Table(t *testing.T) {
	t.Parallel()
	b := backend.NewMemoryBackend()
	out, _, err := run(t, b, "plugin", "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, want := range []string{"NAME", "VERSION", "INSTALLED", "backup-s3"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
}

func TestClusterStatus_NotFound(t *testing.T) {
	t.Parallel()
	b := backend.NewMemoryBackend()
	_, _, err := run(t, b, "cluster", "status", "ghost")
	if err == nil {
		t.Fatalf("expected error for unknown cluster")
	}
}

func TestDBLogs_OneShot(t *testing.T) {
	t.Parallel()
	b := backend.NewMemoryBackend()
	out, _, err := run(t, b, "db", "logs", "orders-pg")
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if !strings.Contains(out, "INFO") {
		t.Fatalf("expected log lines, got:\n%s", out)
	}
}

func TestDBGet_ShellCompletion(t *testing.T) {
	t.Parallel()
	b := backend.NewMemoryBackend()
	// Cobra exposes hidden __complete command; verify completions surface seed dbs.
	out, _, err := run(t, b, "__complete", "db", "get", "")
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if !strings.Contains(out, "orders-pg") {
		t.Fatalf("completion missing seeded db:\n%s", out)
	}
}

func TestSupportedEngines_FlagCompletion(t *testing.T) {
	t.Parallel()
	b := backend.NewMemoryBackend()
	out, _, err := run(t, b, "__complete", "db", "create", "x", "--engine", "")
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if !strings.Contains(out, "postgresql") {
		t.Fatalf("engine completion missing postgresql:\n%s", out)
	}
}

func TestCompletion_Bash(t *testing.T) {
	t.Parallel()
	b := backend.NewMemoryBackend()
	out, _, err := run(t, b, "completion", "bash")
	if err != nil {
		t.Fatalf("completion: %v", err)
	}
	if !strings.Contains(out, "everestctl") {
		t.Fatalf("bash completion looks empty:\n%s", out)
	}
}
