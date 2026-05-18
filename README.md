# everestctl POC

A proof-of-concept for the LFX mentorship project
**"CNCF - OpenEverest: Transform `everestctl` into a Powerful Database
Management CLI"** (2026 Term 2).

This POC implements the proposed command surface on top of a pluggable
backend interface. The shipped backend is in-memory, so every command is
runnable without a Kubernetes cluster — the real implementation later
swaps `internal/backend.MemoryBackend` for a Kubernetes / Everest API
client without touching the command layer.

## What the POC demonstrates

| Outcome from the project brief                         | Where it lives                         |
| ------------------------------------------------------ | -------------------------------------- |
| `db list / create / delete / get / logs`               | [`internal/cli/db`](internal/cli/db/db.go) |
| `cluster list / register / status`                     | [`internal/cli/cluster`](internal/cli/cluster/cluster.go) |
| `plugin list / install / configure`                    | [`internal/cli/plugin`](internal/cli/plugin/plugin.go) |
| Shell completion (bash / zsh / fish / powershell)      | [`internal/cli/completion`](internal/cli/completion/completion.go) |
| Multiple output formats (table / json / yaml)          | [`internal/cli/output`](internal/cli/output/output.go) |
| Pluggable backend seam (Kubernetes-ready)              | [`internal/backend`](internal/backend/backend.go) |
| Unit + integration tests (≥80% coverage)               | `*_test.go`, see below                 |

## Command tree

```mermaid
graph LR
    root([everestctl])
    root --> db[db]
    root --> cluster[cluster]
    root --> plugin[plugin]
    root --> completion[completion]

    db --> db_list[list]
    db --> db_get[get]
    db --> db_create[create]
    db --> db_delete[delete]
    db --> db_logs[logs]

    cluster --> cl_list[list]
    cluster --> cl_reg[register]
    cluster --> cl_status[status]

    plugin --> pl_list[list]
    plugin --> pl_install[install]
    plugin --> pl_config[configure]

    completion --> sh[bash / zsh / fish / powershell]

    classDef leaf fill:#eef,stroke:#557
    class db_list,db_get,db_create,db_delete,db_logs,cl_list,cl_reg,cl_status,pl_list,pl_install,pl_config,sh leaf
```

## Quickstart

```sh
go build -o bin/everestctl ./cmd/everestctl

./bin/everestctl db list
./bin/everestctl db list -o json
./bin/everestctl db create reports-mysql --engine mysql --version 8.0 --replicas 2
./bin/everestctl db get reports-mysql -o yaml
./bin/everestctl db logs orders-pg

./bin/everestctl cluster list
./bin/everestctl cluster register prod --endpoint https://k8s.prod.example.com --context prod

./bin/everestctl plugin list
./bin/everestctl plugin install pmm
./bin/everestctl plugin configure backup-s3 --set bucket=my-backups --set region=eu-west-1
```

Because the backend lives in memory, state does **not** persist across
process invocations of the binary — each run starts with the seeded
sample data. Inside a single process (e.g. a test, or a future REPL)
state behaves normally.

### Shell completion

```sh
# bash
source <(./bin/everestctl completion bash)

# zsh
./bin/everestctl completion zsh > "${fpath[1]}/_everestctl"

# fish
./bin/everestctl completion fish | source
```

Completion is wired for database names, engine choices, and shell
selection — see `examples/completion-demo.txt` for sample output.

## Architecture

```mermaid
flowchart TB
    user([user shell])

    subgraph entry["entrypoint"]
        main["cmd/everestctl/main.go<br/><i>signals, exit codes</i>"]
    end

    subgraph cli["internal/cli — cobra command tree"]
        root["root.go<br/><i>root cmd, -o flag, wiring</i>"]
        dbcmd["db/"]
        clcmd["cluster/"]
        plcmd["plugin/"]
        compcmd["completion/"]
        outpkg["output/<br/><i>table / json / yaml</i>"]
    end

    subgraph backend_layer["internal/backend — pluggable seam"]
        iface{{"Backend interface"}}
        mem["MemoryBackend<br/><i>shipped in POC</i>"]
        k8s["KubernetesBackend<br/><i>future</i>"]:::future
    end

    user --> main --> root
    root --> dbcmd & clcmd & plcmd & compcmd
    dbcmd & clcmd & plcmd --> outpkg
    dbcmd & clcmd & plcmd --> iface
    iface --> mem
    iface -.-> k8s

    classDef future stroke-dasharray: 5 5,fill:#fff5e6,stroke:#cc8a00
```

The dashed `KubernetesBackend` block is the *only* thing the real
implementation adds — nothing in the `cli` tree changes.

```mermaid
sequenceDiagram
    autonumber
    actor U as user
    participant M as main
    participant R as cobra root
    participant D as db/create cmd
    participant B as Backend (iface)
    participant I as MemoryBackend<br/>(future: K8s client)
    participant O as output renderer

    U->>M: everestctl db create orders-pg --engine postgresql -o json
    M->>R: ExecuteContext(ctx)
    R->>D: dispatch + parse flags
    D->>D: backend.ParseEngine("postgresql")
    D->>B: CreateDatabase(opts)
    B->>I: (impl call)
    I-->>B: Database{status: pending}
    B-->>D: Database
    D->>O: Render(stdout, json, db)
    O-->>U: JSON payload
```

Two design decisions worth flagging for the mentorship review:

1. **`backend.Backend` is the only seam the command layer depends on.**
   Adding a real Kubernetes-backed implementation is purely additive —
   the command tree, output formatting, and tests stay put.
2. **`Deps` structs instead of package-level globals.** Each command
   group receives an explicit `Deps{Backend, Out, Err, Format}` so tests
   can swap any of them. This is what makes the 88% coverage achievable
   without spinning up a real cluster.

## Tests & coverage

```sh
go test ./...
go test ./... -coverpkg=./internal/... -coverprofile=cover.out
go tool cover -func=cover.out | tail -1
# total:    (statements)    88.4%
```

Two layers of tests:

- `internal/backend/memory_test.go` — unit tests for the backend
  invariants (duplicate detection, namespace filtering, plugin
  lifecycle, log streaming, …).
- `internal/cli/cli_test.go` — integration tests that drive the root
  cobra command end-to-end (table / json / yaml output, required-flag
  enforcement, create→get→delete round trip, shell completion).

## Roadmap to the real implementation

| Phase | Work                                                                 |
| ----- | -------------------------------------------------------------------- |
| 1     | Replace `MemoryBackend` with a Kubernetes client using OpenEverest CRDs. |
| 2     | Add `--kubeconfig` / `--context` global flags, config file loader.   |
| 3     | Add `db backup` / `db restore` and async operation polling.          |
| 4     | Real plugin registry integration (OCI artifacts, Helm).              |
| 5     | Interactive prompts behind a `--interactive` flag for `delete`, `create`. |

## Reference

- Project brief: CNCF OpenEverest – Transform everestctl into a Powerful
  Database Management CLI (LFX 2026 Term 2).
- Issue tracker: https://github.com/openeverest/openeverest/issues/1818
