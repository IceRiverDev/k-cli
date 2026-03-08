# k-cli

`k-cli` is a production-grade, kubectl-like CLI tool for managing Kubernetes Pods. Built with Go and [Cobra](https://github.com/spf13/cobra), it provides a clean interface for the most common Pod lifecycle operations.

---

## Features

| Command | Description |
|---------|-------------|
| `k-cli exec` | Open an interactive TTY shell in a running Pod |
| `k-cli describe` | Show detailed Pod specification and status |
| `k-cli sync` | Sync local files/directories to a Pod (supports `--watch` hot-reload) |
| `k-cli pull` | Pull files/directories from a Pod to local |
| `k-cli diagnose` | Diagnose Pod health and give actionable suggestions |

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
# Describe a pod
k-cli describe my-pod

# Exec into it
k-cli exec my-pod -c my-pod

# Sync a local config file
k-cli sync my-pod ./nginx.conf /etc/nginx/nginx.conf

# Pull a file from the pod
k-cli pull my-pod /app/config.yaml ./config.yaml

# Watch for local changes and auto-sync
k-cli sync my-pod ./src /app/src --watch

# Diagnose pod health
k-cli diagnose my-pod
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

### `k-cli describe <pod-name>` — Show Pod Details

Displays full Pod specification including containers, labels, annotations, and recent events.

```bash
k-cli describe <pod-name> [flags]

Flags:
  -o, --output string      Output format: yaml
  -n, --namespace string   Namespace (default "default")
```

**Examples:**

```bash
# Human-readable output
k-cli describe my-pod -n default

# Raw YAML output
k-cli describe my-pod -n default -o yaml
```

---

### `k-cli sync <pod-name> <local-path> <remote-path>` — Sync Files to Pod

Copies a local file or directory into a Pod using the `exec + tar` streaming mechanism (same as `kubectl cp`). Supports `--watch` for continuous hot-reload.

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

# Watch for changes and auto-sync (hot-reload)
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
k-cli describe my-pod --kubeconfig /path/to/kubeconfig

# Use the KUBECONFIG environment variable
export KUBECONFIG=/path/to/kubeconfig
k-cli describe my-pod
```

---

## Project Structure

```
k-cli/
├── main.go
├── cmd/
│   ├── root.go       # Root command and global flags
│   ├── exec.go       # k-cli exec
│   ├── describe.go   # k-cli describe
│   ├── sync.go       # k-cli sync (--watch hot-reload)
│   ├── pull.go       # k-cli pull
│   └── diagnose.go   # k-cli diagnose
├── internal/
│   └── k8s/
│       └── client.go # Kubernetes client wrapper
├── go.mod
└── README.md
```

