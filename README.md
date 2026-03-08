# k-cli

`k-cli` is a production-grade, kubectl-like CLI tool for managing Kubernetes Pods. Built with Go and [Cobra](https://github.com/spf13/cobra), it provides a clean interface for the most common Pod lifecycle operations.

---

## Features

| Command | Description |
|---------|-------------|
| `k-cli exec` | Open an interactive TTY shell in a running Pod |
| `k-cli sync` | Sync local files/directories to a Pod (supports `--watch` hot-reload, recursive monitoring) |
| `k-cli pull` | Pull files/directories from a Pod to local |
| `k-cli diagnose` | Diagnose Pod health and give actionable suggestions |
| `k-cli secret` | View a Kubernetes Secret with automatically decoded base64 values |

> **Note:** For creating and deleting Pods, use the native `kubectl create` and `kubectl delete` commands.

---

## Installation

### Using `go install`

```bash
go install github.com/IceRiverDev/k-cli@latest
```

### Build from source (with version injection)

```bash
git clone https://github.com/IceRiverDev/k-cli.git
cd k-cli

VERSION=$(git describe --tags --always --dirty)
go build -ldflags "-X main.version=${VERSION}" -o k-cli .
```

Move the binary to somewhere on your `$PATH`:

```bash
mv k-cli /usr/local/bin/
k-cli --help
```

---

## Quick Start

```bash
# Exec into a pod
k-cli exec my-pod -c my-pod

# Sync a local config file
k-cli sync my-pod ./nginx.conf /etc/nginx/nginx.conf

# Pull a file from the pod
k-cli pull my-pod /app/config.yaml ./config.yaml

# Watch for local changes and auto-sync (recursive, debounced)
k-cli sync my-pod ./src /app/src --watch

# Diagnose pod health
k-cli diagnose my-pod

# View a secret with decoded values
k-cli secret my-secret -n default
```

---

## Commands

### `k-cli exec <pod-name>` — Enter a Pod Shell

Opens an interactive TTY session (`/bin/bash` with `/bin/sh` fallback).

```bash
k-cli exec <pod-name> [flags]

Flags:
  -c, --container string   Container name (defaults to first container)
  -n, --namespace string   Namespace (default "default")
```

**Examples:**

```bash
# Enter the default container
k-cli exec my-pod -n default

# Enter a specific container
k-cli exec my-pod -n default -c main
```

---

### `k-cli sync <pod-name> <local-path> <remote-path>` — Sync Files to Pod

Copies a local file or directory into a Pod using the `exec + tar` streaming mechanism (same as `kubectl cp`). Supports `--watch` for continuous hot-reload with recursive directory monitoring, debouncing, and compatibility with editor atomic-write patterns (vim, GoLand, etc.).

```bash
k-cli sync <pod-name> <local-path> <remote-path> [flags]

Flags:
  -c, --container string      Container name (defaults to first container)
      --delete                Remove remote files not present locally
      --exclude stringArray   Exclude pattern(s) (repeatable)
  -n, --namespace string      Namespace (default "default")
      --watch                 Watch local path for changes and auto-sync to pod
```

**Examples:**

```bash
# Sync a directory
k-cli sync my-pod ./src /app/src -n default -c main

# Sync a single file
k-cli sync my-pod ./config.yaml /app/config.yaml

# Sync with deletion of stale remote files
k-cli sync my-pod ./dist /app/dist --delete --exclude .git --exclude node_modules

# Watch for changes and auto-sync (hot-reload, recursive, editor-compatible)
k-cli sync my-pod ./src /app/src --watch
```

---

### `k-cli pull <pod-name> <remote-path> <local-path>` — Pull Files from Pod

Pulls a file or directory from inside a Kubernetes Pod to the local filesystem using the `exec + tar` streaming mechanism.

```bash
k-cli pull <pod-name> <remote-path> <local-path> [flags]

Flags:
  -c, --container string   Container name (defaults to first container)
  -n, --namespace string   Namespace (default "default")
```

**Examples:**

```bash
# Pull a directory from pod to local
k-cli pull my-pod /app/logs ./local-logs -n default

# Pull a single file
k-cli pull my-pod /app/config.yaml ./config.yaml -n default -c main
```

---

### `k-cli secret <secret-name>` — View Decoded Secret

Fetches a Kubernetes Secret and automatically decodes all base64-encoded values for easy inspection.

```bash
k-cli secret <secret-name> [flags]

Flags:
      --key string         Show only this specific key
  -n, --namespace string   Namespace (default "default")
      --show-encoded       Also show the original base64 encoded value
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

Flags:
  -n, --namespace string   Namespace (default "default")
```

**Checks performed:**
- Pod phase (Running / Pending / Failed / Unknown)
- Restart count and last termination reason
- OOMKilled detection with memory limit info
- Resource limits/requests configuration
- Container readiness status
- Recent warning events

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
| `--log` | Enable log output (disabled by default) |
| `-v, --verbose` | Enable verbose/debug logging (only effective when `--log` is also set) |

---

## Configuration

`k-cli` uses the standard Kubernetes kubeconfig discovery:

1. `--kubeconfig` flag
2. `$KUBECONFIG` environment variable
3. `~/.kube/config` (default)

```bash
# Use a specific kubeconfig
k-cli exec my-pod --kubeconfig /path/to/kubeconfig

# Use the KUBECONFIG environment variable
export KUBECONFIG=/path/to/kubeconfig
k-cli exec my-pod
```

---

## Project Structure

```
k-cli/
├── main.go
├── cmd/
│   ├── root.go       # Root command and global flags
│   ├── exec.go       # k-cli exec
│   ├── sync.go       # k-cli sync (--watch hot-reload, recursive)
│   ├── pull.go       # k-cli pull
│   ├── diagnose.go   # k-cli diagnose
│   └── secret.go     # k-cli secret
├── internal/
│   └── k8s/
│       └── client.go # Kubernetes client wrapper
├── go.mod
└── README.md
```

