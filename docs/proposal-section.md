# POC: `everestctl` as a Database-Management CLI

> Content drafted for the LFX 2026 Term 2 mentorship proposal.
> Copy-paste the prose into your Google Doc; the screenshots
> referenced below live in `docs/screenshots/` of this repo —
> drag them into the doc manually at the marked positions.

---

## Proof of concept

Before applying, I built a runnable proof of concept that implements
the full command surface described in the issue brief on top of a
pluggable backend interface. The POC uses an in-memory backend, so
every subcommand can be exercised without a real Kubernetes cluster
— making it easy for mentors to evaluate the proposed UX in seconds.

**Repository:** https://github.com/alokkumardalei-wq/everestctl-poc

The README of that repository contains a ~75-second demo GIF, Mermaid
diagrams of the command tree and the architecture, a sequence diagram
of the `db create` flow, and clear build/test instructions.

### What the POC implements

The POC delivers each of the outcomes listed in the project brief:

- **Database management.** `everestctl db list / get / create / delete / logs`
  for PostgreSQL, MySQL, and MongoDB, with engine validation and
  namespace filtering.
- **Cluster management.** `everestctl cluster list / register / status`
  for managing the Kubernetes clusters that OpenEverest targets.
- **Plugin integration.** `everestctl plugin list / install / configure`
  with `--set key=value` for plugin configuration.
- **Shell completion.** Working scripts for bash, zsh, fish, and
  powershell, including dynamic completion for database names and
  engine values.
- **Multiple output formats.** Every command supports
  `-o table | json | yaml`, with table as the human-friendly default
  and `json` / `yaml` for scripting and piping.
- **Tests and coverage.** Unit tests for the backend and integration
  tests that drive the root cobra command end-to-end. Measured
  coverage across the new packages is **88.4 %**, exceeding the
  brief's ≥80 % target.

### Architecture

The POC is structured around a single narrow seam: a `Backend`
interface that every command depends on. The shipped implementation
is in-memory; the real implementation will be a thin Kubernetes /
OpenEverest CRD client that drops into the same seam without
touching the command layer. Two specific design choices make this
testable and easy to extend:

1. **One interface, two implementations.** Command code only ever
   sees `backend.Backend`. Adding the real Kubernetes-backed
   implementation is purely additive — the command tree, output
   formatting and tests stay put.
2. **Explicit dependency injection.** Each command group accepts a
   small `Deps{Backend, Out, Err, Format}` struct instead of using
   package-level globals. This is what enables 88 %+ test coverage
   without spinning up envtest or a kind cluster.

The repository README contains a full Mermaid architecture diagram
showing how `cmd → cli → Backend interface → KubernetesBackend`
fit together, plus a sequence diagram for the `db create` flow.

### Demo walkthrough

The screenshots below are stills extracted from the demo recording
(`demo.gif` in the repository root). The full ~75-second animated
walkthrough is embedded at the top of the repository README.

**[Insert `docs/screenshots/01-title.png`]**
*The terminal session opens by stating what is being demonstrated.*

**[Insert `docs/screenshots/02-tests-coverage.png`]**
*The session begins with `go build` and `go test ./... -coverpkg=./internal/...`,
which reports `total: (statements) 88.4 %`. Below the coverage line
is the `everestctl --help` output showing the top-level command tree.*

**[Insert `docs/screenshots/04-db-list-table.png`]**
*`db list` rendered as a table by default, and the same data rendered
as JSON for scripting via `-o json`.*

**[Insert `docs/screenshots/07-db-logs.png`]**
*The complete database lifecycle on one screen: `db get -o yaml`
(detailed inspection), `db create billing-pg --engine postgresql --version 16.2 --replicas 2`
(provisioning), `db delete sessions-mongo --yes` (deletion with
confirmation), and `db logs orders-pg | head -3` (log tailing).*

**[Insert `docs/screenshots/08-cluster.png`]**
*Cluster management: `cluster list` showing the registered local
cluster, followed by `cluster register prod --endpoint https://k8s.prod.example.com --context prod`
which adds a new cluster.*

**[Insert `docs/screenshots/09-plugin.png`]**
*Plugin management: `plugin list` shows available and installed plugins,
`plugin install pmm` installs one, and `plugin configure backup-s3 --set bucket=my-backups --set region=eu-west-1`
updates configuration with multiple `--set` flags.*

**[Insert `docs/screenshots/10-completion.png`]**
*Shell-completion script generation: `everestctl completion bash`
emits a complete bash completion script that users source from their
shell init file. Equivalent commands work for `zsh`, `fish`, and
`powershell`.*

### Quality signals

- **Build:** `go build ./...` succeeds with no warnings.
- **Tests:** `go test ./...` passes; integration tests cover the
  `db create → get → delete` round-trip end-to-end through the cobra
  command tree.
- **Coverage:** 88.4 % across `internal/...` (measured with
  `go test ./... -coverpkg=./internal/... -coverprofile=cover.out`,
  reported by `go tool cover -func`).
- **Reproducibility:** the demo recording can be regenerated via
  `scripts/record-demo.sh` plus `asciinema rec` and `agg`; both the
  cast file (`demo.cast`) and the rendered GIF (`demo.gif`) are
  committed to the repository.

### Mentorship contact and visibility

A draft pull request has been opened on `openeverest/openeverest`
([proposal/everestctl-db-management-cli branch](https://github.com/alokkumardalei-wq/openeverest/tree/proposal/everestctl-db-management-cli))
to make the POC visible to maintainers and to solicit direction
on the shape of the real implementation PRs. The PR contains only
a short pointer document, not the POC code itself — the POC remains
in its own repository as a runnable design artifact.

---

## How I will use the POC during the mentorship

The POC is a *design artifact*, not the final implementation. During
the mentorship I will:

1. Use the POC's `Backend` interface as the template for the real
   `Client` abstraction in `pkg/cli/`, adapting to whatever
   conventions exist in the OpenEverest codebase.
2. Port the proposed command surface into `commands/db/`,
   `commands/cluster/`, `commands/plugin/`, one slice per PR, each
   small enough to review independently.
3. Replace the in-memory backend with a Kubernetes client built on
   the existing OpenEverest CRDs.
4. Carry over the output, completion and test scaffolding from the
   POC so the ≥80 % coverage target is met from the first
   implementation PR.

---

**Tip for pasting into Google Docs:**

1. Open the doc, position the cursor where you want the section.
2. Copy this entire file's contents (Cmd+A / Cmd+C in your editor).
3. Paste into Google Docs (Cmd+V). Markdown headings (`##`),
   bold (`**`), italics (`*`), bullets (`-`) and inline code (`` ` ``)
   typically translate into formatted Google-Docs equivalents
   automatically; if not, use **Edit → Paste without formatting**
   and apply heading styles manually.
4. For each `[Insert docs/screenshots/NN-name.png]` marker, replace
   it with **Insert → Image → Upload from computer**, picking the
   matching PNG from `everestctl-poc/docs/screenshots/`.
5. Resize images to ~80 % page width for a clean look.
