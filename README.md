# k-cli

`k-cli` is a production-grade, kubectl-like CLI tool for managing Kubernetes Pods which kubectl can not do. Built with Go and [Cobra](https://github.com/spf13/cobra), it provides a clean interface for specific Pod operations.

---

## Features

| Command | Description |
|---------|-------------|
| `k-cli sync` | Sync local files/directories to a Pod (`--watch` enables recursive auto-sync with rsync-like stale-file cleanup) |
| `k-cli pull` | Pull files/directories from a Pod to local via tar streaming |
| `k-cli diagnose` | Diagnose Pod health and give actionable suggestions |
| `k-cli secret` | View a Kubernetes Secret with automatically decoded values |
| `k-cli completion` | Generate shell completion scripts (bash / zsh / fish / powershell) |

> **Note:** For creating and deleting Pods, use the native `kubectl create` and `kubectl delete` commands.

---

## Installation

### Using `go install`

```bash
go install github.com/IceRiverDev/k-cli@latest
```

### Build from source

```bash
git clone https://github.com/IceRiverDev/k-cli.git
cd k-cli
go build -o k-cli .
```

Move the binary to somewhere on your `$PATH`:

```bash
mv k-cli /usr/local/bin/
k-cli --help
```

### Shell Completion

After installing the binary, set up tab-completion for your shell:

**Bash**
```bash
# Load for the current session
source <(k-cli completion bash)

# Persist (Linux)
k-cli completion bash > /etc/bash_completion.d/k-cli

# Persist (macOS with Homebrew bash-completion@2)
k-cli completion bash > $(brew --prefix)/etc/bash_completion.d/k-cli
```

**Zsh**
```zsh
# Load for the current session
source <(k-cli completion zsh)

# Persist (add to ~/.zshrc)
echo 'source <(k-cli completion zsh)' >> ~/.zshrc

# With oh-my-zsh
k-cli completion zsh > ~/.oh-my-zsh/completions/_k-cli
```

**Fish**
```fish
k-cli completion fish > ~/.config/fish/completions/k-cli.fish
```

**PowerShell**
```powershell
# Load for the current session
k-cli completion powershell | Out-String | Invoke-Expression

# Persist (add to your PowerShell profile)
k-cli completion powershell >> $PROFILE
```

---

## Quick Start

```bash
# Sync a local directory to a remote directory
k-cli sync my-pod ./src /app/src

# Sync a single file into a remote directory
k-cli sync my-pod ./nginx.conf /etc/nginx

# Pull a remote path into a local directory (keeps relative structure)
k-cli pull my-pod /app/config.yaml ./downloads

# Watch for local changes and auto-sync (recursive + debounced)
k-cli sync my-pod ./src /app/src --watch

# Diagnose pod health
k-cli diagnose my-pod

# View a secret with decoded values
k-cli secret my-secret -n default
```

---

## Commands

### `k-cli sync <pod-name> <local-path> <remote-path>` — Sync Files to Pod

Copies a local file or directory into a Pod using tar streaming over `exec` (similar to `kubectl cp`).

- One-shot mode syncs content to `remote-path`.
- `--delete` is applied in one-shot mode for directory syncs.
- `--watch` enables recursive file watching with debouncing; each trigger runs a full rsync-like sync (including stale remote file cleanup).

```bash
k-cli sync <pod-name> <local-path> <remote-path> [flags]

Flags:
  -c, --container string      Container name (defaults to first container)
      --delete                Remove remote files not present locally (one-shot mode)
      --exclude stringArray   Exclude by exact name/path component (repeatable; not glob syntax)
      --watch                 Watch local path for changes and auto-sync to pod

Global Flags:
  -n, --namespace string      Namespace (default "default")
```

**Examples:**

```bash
# Sync a directory
k-cli sync my-pod ./src /app/src -n default -c main

# Sync a single file into a remote directory
k-cli sync my-pod ./config.yaml /app/config -n default

# One-shot sync with stale-file cleanup
k-cli sync my-pod ./dist /app/dist --delete --exclude .git --exclude node_modules

# Watch for changes and auto-sync (also cleans stale remote files)
k-cli sync my-pod ./src /app/src --watch
```

---

### `k-cli pull <pod-name> <remote-path> <local-path>` — Pull Files from Pod

Pulls a file or directory from inside a Kubernetes Pod to the local filesystem using tar streaming over `exec`.

> `local-path` is treated as a destination directory. Pulled entries keep their tar names under that directory.

```bash
k-cli pull <pod-name> <remote-path> <local-path> [flags]

Flags:
  -c, --container string   Container name (defaults to first container)

Global Flags:
  -n, --namespace string   Namespace (default "default")
```

**Examples:**

```bash
# Pull a directory from pod to local
k-cli pull my-pod /app/logs ./local-logs -n default

# Pull a single file into a local directory
k-cli pull my-pod /app/config.yaml ./downloads -n default -c main
```

---

### `k-cli secret <secret-name>` — View Decoded Secret

Fetches a Kubernetes Secret and automatically decodes all base64-encoded values for easy inspection.

```bash
k-cli secret <secret-name> [flags]

Flags:
      --key string      Show only this specific key
      --show-encoded    Also show the original base64 encoded value

Global Flags:
  -n, --namespace string   Namespace (default "default")
```

**Examples:**

```bash
# Show all keys in a secret (decoded)
k-cli secret my-secret -n default

# Show only a specific key
k-cli secret my-secret --key DB_PASSWORD

# Show both decoded and encoded values
k-cli secret my-secret --show-encoded
```

---

### `k-cli diagnose <pod-name>` — Diagnose Pod Health

Inspects a Pod's status, restart history, resource limits, container readiness, and recent events to produce a health report with actionable suggestions.

```bash
k-cli diagnose <pod-name> [flags]

Global Flags:
  -n, --namespace string   Namespace (default "default")
```

**Checks performed:**
- Pod phase (Running / Pending / Failed / Unknown)
- Restart count and last termination reason
- OOMKilled detection with memory limit info
- Resource limits/requests configuration
- Container readiness status
- Recent events (up to 5 shown)

**Examples:**

```bash
# Diagnose a pod
k-cli diagnose my-pod -n default

# Diagnose in another namespace
k-cli diagnose my-pod -n production
```

**Sample output:**
```
🔍 Diagnosing pod: my-pod (namespace: default)
─────────────────────────────────────────────
✅ Status:        Running
⚠️  Restarts:     5 (last reason: OOMKilled)
❌ Memory:        OOMKilled detected — current limit: 128Mi
   💡 Suggestion: increase memory limit to at least 256Mi
⚠️  Resources:    No CPU limit set for container "my-pod"
   💡 Suggestion: set resources.limits.cpu to avoid noisy neighbor
✅ Containers:    1/1 Ready
─────────────────────────────────────────────
📋 Recent Events (last 5):
   [Warning] OOMKilling: Memory limit reached
   [Normal]  Pulled: Successfully pulled image
─────────────────────────────────────────────
🏁 Diagnosis complete: 1 critical, 2 warnings
```

---

## Global Flags

These flags are available on every command:

| Flag | Description |
|------|-------------|
| `--kubeconfig string` | Path to kubeconfig file (default: `~/.kube/config` or `$KUBECONFIG`) |
| `-n, --namespace string` | Default namespace (default: `default`) |
| `--log` | Enable debug-level log output (disabled by default) |

---

## Configuration

`k-cli` discovers kubeconfig in this order:

1. `--kubeconfig` flag
2. `$KUBECONFIG` environment variable
3. `~/.kube/config` (default)

```bash
# Use a specific kubeconfig
k-cli sync my-pod ./src /app/src --kubeconfig /path/to/kubeconfig

# Use the KUBECONFIG environment variable
export KUBECONFIG=/path/to/kubeconfig
k-cli sync my-pod ./src /app/src
```

---

## Project Structure

```
k-cli/
├── main.go
├── cmd/
│   ├── root.go        # Root command and global flags
│   ├── completion.go  # k-cli completion
│   ├── sync.go        # k-cli sync (--watch recursive + debounced rsync)
│   ├── pull.go        # k-cli pull
│   ├── diagnose.go    # k-cli diagnose
│   └── secret.go      # k-cli secret
├── internal/
│   └── k8s/
│       └── client.go  # Kubernetes client wrapper
├── go.mod
└── README.md
```
