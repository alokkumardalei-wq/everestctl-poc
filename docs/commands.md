# Command reference (POC)

Every command supports `-o table|json|yaml` (default `table`).

## `everestctl db`

| Command | Purpose |
| --- | --- |
| `db list [-n NAMESPACE]` | List databases, optionally filtered by namespace. |
| `db get NAME [-n NAMESPACE]` | Show a single database. |
| `db create NAME --engine ENG [--version V] [--replicas N] [--cluster C] [-n NAMESPACE]` | Provision a database. `--engine` is required: `postgresql\|mysql\|mongodb`. |
| `db delete NAME --yes [-n NAMESPACE]` | Delete a database. `--yes` is required (no interactive prompt in POC). |
| `db logs NAME [-f] [-n NAMESPACE]` | Tail synthetic logs from the database. |

## `everestctl cluster`

| Command | Purpose |
| --- | --- |
| `cluster list` | List registered Kubernetes clusters. |
| `cluster register NAME --endpoint URL [--context CTX] [--version V]` | Register a cluster. `--endpoint` required. |
| `cluster status NAME` | Show the latest status for a registered cluster. |

## `everestctl plugin`

| Command | Purpose |
| --- | --- |
| `plugin list` | List available and installed plugins. |
| `plugin install NAME [--version V]` | Install a plugin. |
| `plugin configure NAME --set k=v [--set k=v ...]` | Update plugin configuration. |

## `everestctl completion`

| Command | Purpose |
| --- | --- |
| `completion bash\|zsh\|fish\|powershell` | Emit a completion script for the chosen shell. |

## Global flags

| Flag | Description |
| --- | --- |
| `-o, --output` | `table` (default), `json`, or `yaml`. |
| `-h, --help` | Help for any command. |
| `-v, --version` | Print the CLI version. |
