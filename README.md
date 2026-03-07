# k-cli

`k-cli` is a production-grade, kubectl-like CLI tool for managing Kubernetes Pods. Built with Go and [Cobra](https://github.com/spf13/cobra), it provides a clean interface for the most common Pod lifecycle operations.

---

## Features

| Command | Description |
|---------|-------------|
| `k-cli exec` | Open an interactive TTY shell in a running Pod |
| `k-cli create` | Create a new Pod from a container image |
| `k-cli delete` | Delete a Pod (with optional force) |
| `k-cli describe` | Show detailed Pod specification and status |
| `k-cli sync` | Sync local files/directories to a Pod path |

---

## Installation

### Using `go install`

```bash
go install github.com/IceRiverDev/simple-cli@latest
```

### Build from source (with version injection)

```bash
git clone https://github.com/IceRiverDev/simple-cli.git
cd simple-cli

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
# Create a pod
k-cli create my-pod --image nginx:latest --port 80

# Describe it
k-cli describe my-pod

# Exec into it
k-cli exec my-pod -c my-pod

# Sync a local config file
k-cli sync my-pod ./nginx.conf /etc/nginx/nginx.conf

# Delete it
k-cli delete my-pod
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

### `k-cli create <pod-name>` — Create a Pod

Creates a Kubernetes Pod from a container image.

```bash
k-cli create <pod-name> --image <image> [flags]

Flags:
      --image string         Container image (required)
      --port int             Port to expose
      --env stringArray      Environment variables, KEY=VALUE (repeatable)
      --labels stringArray   Pod labels, KEY=VALUE (repeatable)
  -n, --namespace string     Namespace (default "default")
```

**Examples:**

```bash
# Create a basic pod
k-cli create my-pod --image nginx:latest

# Create with port and env vars
k-cli create my-pod --image nginx:latest --port 80 --env ENV=production --env VERSION=1.0

# Create with labels
k-cli create my-pod --image nginx:latest --labels app=my-app --labels tier=frontend
```

---

### `k-cli delete <pod-name>` — Delete a Pod

Deletes a Pod. Supports immediate (forced) deletion.

```bash
k-cli delete <pod-name> [flags]

Flags:
      --force              Grace period = 0 (immediate)
  -n, --namespace string   Namespace (default "default")
```

**Examples:**

```bash
# Graceful delete
k-cli delete my-pod -n default

# Force delete
k-cli delete my-pod -n default --force
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

Copies a local file or directory into a Pod using the `exec + tar` streaming mechanism (same as `kubectl cp`).

```bash
k-cli sync <pod-name> <local-path> <remote-path> [flags]

Flags:
  -c, --container string      Container name (defaults to first container)
      --delete                Remove remote files not present locally
      --exclude stringArray   Exclude pattern(s) (repeatable)
  -n, --namespace string      Namespace (default "default")
```

**Examples:**

```bash
# Sync a directory
k-cli sync my-pod ./src /app/src -n default -c main

# Sync a single file
k-cli sync my-pod ./config.yaml /app/config.yaml

# Sync with deletion of stale remote files
k-cli sync my-pod ./dist /app/dist --delete --exclude .git --exclude node_modules
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
simple-cli/
├── main.go
├── cmd/
│   ├── root.go       # Root command and global flags
│   ├── exec.go       # k-cli exec
│   ├── create.go     # k-cli create
│   ├── delete.go     # k-cli delete
│   ├── describe.go   # k-cli describe
│   └── sync.go       # k-cli sync
├── internal/
│   └── k8s/
│       └── client.go # Kubernetes client wrapper
├── go.mod
└── README.md
```
