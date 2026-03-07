package cmd

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

var (
	syncContainer string
	syncDelete    bool
	syncExcludes  []string
)

var syncCmd = &cobra.Command{
	Use:   "sync <pod-name> <local-path> <remote-path>",
	Short: "Sync a local file or directory to a Pod",
	Long: `Synchronize a local file or directory to a specified path inside a Kubernetes Pod
using kubectl cp-style tar streaming over the exec API.`,
	Example: `  # Sync a directory
  k-cli sync my-pod ./src /app/src -n default -c main

  # Sync a single file
  k-cli sync my-pod ./config.yaml /app/config.yaml -n default

  # Sync and delete remote files not present locally
  k-cli sync my-pod ./dist /app/dist --delete --exclude .git --exclude node_modules`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		podName := args[0]
		localPath := args[1]
		remotePath := args[2]

		Logger.Info("Syncing files",
			zap.String("pod", podName),
			zap.String("namespace", namespace),
			zap.String("container", syncContainer),
			zap.String("local", localPath),
			zap.String("remote", remotePath),
		)

		return syncToPod(cmd.Context(), podName, namespace, syncContainer, localPath, remotePath, syncDelete, syncExcludes)
	},
}

func init() {
	syncCmd.Flags().StringVarP(&syncContainer, "container", "c", "", "container name (defaults to the first container)")
	syncCmd.Flags().BoolVar(&syncDelete, "delete", false, "delete files in the remote directory that do not exist locally")
	syncCmd.Flags().StringArrayVar(&syncExcludes, "exclude", nil, "exclude files or directories matching this pattern (can be repeated)")
	rootCmd.AddCommand(syncCmd)
}

// syncToPod copies local path to the remote path in the pod via tar streaming.
func syncToPod(ctx context.Context, podName, ns, container, localPath, remotePath string, deleteRemote bool, excludes []string) error {
	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("local path %q not found: %w", localPath, err)
	}

	// Collect files to sync.
	type fileEntry struct {
		localFull  string // absolute local path
		archiveName string // relative path inside tar
	}

	var entries []fileEntry
	if info.IsDir() {
		err = filepath.Walk(localPath, func(path string, fi os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, _ := filepath.Rel(localPath, path)
			if shouldExclude(rel, excludes) {
				if fi.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !fi.IsDir() {
				entries = append(entries, fileEntry{
					localFull:  path,
					archiveName: rel,
				})
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk local directory %q: %w", localPath, err)
		}
	} else {
		entries = append(entries, fileEntry{
			localFull:  localPath,
			archiveName: filepath.Base(localPath),
		})
	}

	fmt.Printf("Syncing %d file(s) to %s:%s ...\n", len(entries), podName, remotePath)

	// Build tar archive in memory.
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		if err := addFileToTar(tw, e.localFull, e.archiveName); err != nil {
			return fmt.Errorf("failed to add %q to archive: %w", e.localFull, err)
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("failed to finalize tar archive: %w", err)
	}

	// Optionally delete remote directory first.
	if deleteRemote && info.IsDir() {
		cleanCmd := fmt.Sprintf("rm -rf %s && mkdir -p %s", remotePath, remotePath)
		if err := runRemoteCommand(ctx, podName, ns, container, []string{"sh", "-c", cleanCmd}); err != nil {
			return fmt.Errorf("failed to clean remote directory %q: %w", remotePath, err)
		}
	} else {
		// Ensure remote directory exists.
		mkdirCmd := fmt.Sprintf("mkdir -p %s", remotePath)
		if err := runRemoteCommand(ctx, podName, ns, container, []string{"sh", "-c", mkdirCmd}); err != nil {
			return fmt.Errorf("failed to create remote directory %q: %w", remotePath, err)
		}
	}

	// Stream tar to the pod using 'tar xf - -C <remotePath>'.
	if err := streamTarToPod(ctx, podName, ns, container, remotePath, &buf); err != nil {
		return fmt.Errorf("failed to stream files to pod: %w", err)
	}

	fmt.Printf("Sync complete: %d/%d files transferred\n", len(entries), len(entries))
	return nil
}

// addFileToTar adds a single file to the tar writer under the given archive name.
func addFileToTar(tw *tar.Writer, localPath, archiveName string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	hdr := &tar.Header{
		Name:    archiveName,
		Mode:    int64(fi.Mode()),
		Size:    fi.Size(),
		ModTime: fi.ModTime(),
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	_, err = io.Copy(tw, f)
	return err
}

// shouldExclude returns true if the relative path matches any of the exclusion patterns.
func shouldExclude(rel string, excludes []string) bool {
	for _, pattern := range excludes {
		base := filepath.Base(rel)
		if base == pattern || rel == pattern {
			return true
		}
		// Also check if any component of the path matches.
		for _, part := range strings.Split(rel, string(filepath.Separator)) {
			if part == pattern {
				return true
			}
		}
	}
	return false
}

// streamTarToPod pipes the tar archive into the pod via exec.
func streamTarToPod(ctx context.Context, podName, ns, container, remotePath string, tarData io.Reader) error {
	req := K8sClient.Clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(ns).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{"tar", "xf", "-", "-C", remotePath},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(K8sClient.RestConfig, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  tarData,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return fmt.Errorf("tar command failed: %s: %w", errMsg, err)
		}
		return err
	}
	return nil
}

// runRemoteCommand executes a shell command in the pod and returns any error.
func runRemoteCommand(ctx context.Context, podName, ns, container string, command []string) error {
	req := K8sClient.Clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(ns).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(K8sClient.RestConfig, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return fmt.Errorf("%s: %w", strings.TrimSpace(errMsg), err)
		}
		return err
	}
	return nil
}
