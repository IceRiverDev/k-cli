# phoenix

`phoenix` is a production-grade, kubectl-like CLI tool for managing Kubernetes Pods. Built with Go and [Cobra](https://github.com/spf13/cobra), it provides a clean interface for the most common Pod lifecycle operations.

---

## Features

| Command | Description |
|---------|-------------|
| `phoenix exec` | Open an interactive TTY shell in a running Pod |
| `phoenix create` | Create a new Pod from a container image |
| `phoenix delete` | Delete a Pod (with optional force) |
| `phoenix describe` | Show detailed Pod specification and status |
| `phoenix sync` | Sync local files/directories to a Pod path |

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
go build -ldflags "-X main.version=${VERSION}" -o phoenix .
```

Move the binary to somewhere on your `$PATH`:

```bash
mv phoenix /usr/local/bin/
phoenix --help
```

---

## Quick Start

```bash
# Create a pod
phoenix create my-pod --image nginx:latest --port 80

# Describe it
phoenix describe my-pod

# Exec into it
phoenix exec my-pod -c my-pod

# Sync a local config file
phoenix sync my-pod ./nginx.conf /etc/nginx/nginx.conf

# Delete it
phoenix delete my-pod
```

---

## Commands

### `phoenix exec <pod-name>` — Enter a Pod Shell

Opens an interactive TTY session (`/bin/bash` with `/bin/sh` fallback).

```bash
phoenix exec <pod-name> [flags]

Flags:
  -c, --container string   Container name (defaults to first container)
  -n, --namespace string   Namespace (default "default")
```

**Examples:**

```bash
# Enter the default container
phoenix exec my-pod -n default

# Enter a specific container
phoenix exec my-pod -n default -c main
```

---

### `phoenix create <pod-name>` — Create a Pod

Creates a Kubernetes Pod from a container image.

```bash
phoenix create <pod-name> --image <image> [flags]

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
phoenix create my-pod --image nginx:latest

# Create with port and env vars
phoenix create my-pod --image nginx:latest --port 80 --env ENV=production --env VERSION=1.0

# Create with labels
phoenix create my-pod --image nginx:latest --labels app=my-app --labels tier=frontend
```

---

### `phoenix delete <pod-name>` — Delete a Pod

Deletes a Pod. Supports immediate (forced) deletion.

```bash
phoenix delete <pod-name> [flags]

Flags:
      --force              Grace period = 0 (immediate)
  -n, --namespace string   Namespace (default "default")
```

**Examples:**

```bash
# Graceful delete
phoenix delete my-pod -n default

# Force delete
phoenix delete my-pod -n default --force
```

---

### `phoenix describe <pod-name>` — Show Pod Details

Displays full Pod specification including containers, labels, annotations, and recent events.

```bash
phoenix describe <pod-name> [flags]

Flags:
  -o, --output string      Output format: yaml
  -n, --namespace string   Namespace (default "default")
```

**Examples:**

```bash
# Human-readable output
phoenix describe my-pod -n default

# Raw YAML output
phoenix describe my-pod -n default -o yaml
```

---

### `phoenix sync <pod-name> <local-path> <remote-path>` — Sync Files to Pod

Copies a local file or directory into a Pod using the `exec + tar` streaming mechanism (same as `kubectl cp`).

```bash
phoenix sync <pod-name> <local-path> <remote-path> [flags]

Flags:
  -c, --container string      Container name (defaults to first container)
      --delete                Remove remote files not present locally
      --exclude stringArray   Exclude pattern(s) (repeatable)
  -n, --namespace string      Namespace (default "default")
```

**Examples:**

```bash
# Sync a directory
phoenix sync my-pod ./src /app/src -n default -c main

# Sync a single file
phoenix sync my-pod ./config.yaml /app/config.yaml

# Sync with deletion of stale remote files
phoenix sync my-pod ./dist /app/dist --delete --exclude .git --exclude node_modules
```

---

## Global Flags

These flags are available on every command:

| Flag | Description |
|------|-------------|
| `--kubeconfig string` | Path to kubeconfig file (default: `~/.kube/config` or `$KUBECONFIG`) |
| `-n, --namespace string` | Default namespace (default: `default`) |
| `-v, --verbose` | Enable verbose/debug logging |

---

## Configuration

`phoenix` uses the standard Kubernetes kubeconfig discovery:

1. `--kubeconfig` flag
2. `$KUBECONFIG` environment variable
3. `~/.kube/config` (default)

```bash
# Use a specific kubeconfig
phoenix describe my-pod --kubeconfig /path/to/kubeconfig

# Use the KUBECONFIG environment variable
export KUBECONFIG=/path/to/kubeconfig
phoenix describe my-pod
```

---

## Project Structure

```
simple-cli/
├── main.go
├── cmd/
│   ├── root.go       # Root command and global flags
│   ├── exec.go       # phoenix exec
│   ├── create.go     # phoenix create
│   ├── delete.go     # phoenix delete
│   ├── describe.go   # phoenix describe
│   └── sync.go       # phoenix sync
├── internal/
│   └── k8s/
│       └── client.go # Kubernetes client wrapper
├── go.mod
└── README.md
```
