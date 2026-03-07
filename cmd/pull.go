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
	pullContainer string
)

var pullCmd = &cobra.Command{
	Use:   "pull <pod-name> <remote-path> <local-path>",
	Short: "Pull a file or directory from a Pod to local",
	Long: `Pull a file or directory from inside a Kubernetes Pod to the local filesystem
using kubectl cp-style tar streaming over the exec API.`,
	Example: `  # Pull a directory from pod to local
  k-cli pull my-pod /app/logs ./local-logs -n default

  # Pull a single file
  k-cli pull my-pod /app/config.yaml ./config.yaml -n default -c main`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		podName := args[0]
		remotePath := args[1]
		localPath := args[2]

		Logger.Info("Pulling files from pod",
			zap.String("pod", podName),
			zap.String("namespace", namespace),
			zap.String("container", pullContainer),
			zap.String("remote", remotePath),
			zap.String("local", localPath),
		)

		return pullFromPod(cmd.Context(), podName, namespace, pullContainer, remotePath, localPath)
	},
}

func init() {
	pullCmd.Flags().StringVarP(&pullContainer, "container", "c", "", "container name (defaults to the first container)")
	rootCmd.AddCommand(pullCmd)
}

// pullFromPod pulls a file or directory from a pod to the local filesystem via tar streaming.
func pullFromPod(ctx context.Context, podName, ns, container, remotePath, localPath string) error {
	fmt.Printf("Pulling %s:%s → %s ...\n", podName, remotePath, localPath)

	req := K8sClient.Clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(ns).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{"tar", "cf", "-", remotePath},
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(K8sClient.RestConfig, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
	})
	if err != nil {
		errMsg := stderrBuf.String()
		if errMsg != "" {
			return fmt.Errorf("tar command failed: %s: %w", errMsg, err)
		}
		return fmt.Errorf("failed to stream from pod: %w", err)
	}

	// Extract the tar archive to localPath.
	fileCount, err := extractTar(&stdoutBuf, localPath)
	if err != nil {
		return fmt.Errorf("failed to extract files: %w", err)
	}

	fmt.Printf("Pull complete: %d file(s) pulled to %s\n", fileCount, localPath)
	return nil
}

// extractTar reads a tar stream and extracts its contents into destDir.
func extractTar(r io.Reader, destDir string) (int, error) {
	tr := tar.NewReader(r)
	fileCount := 0

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fileCount, fmt.Errorf("error reading tar stream: %w", err)
		}

		// Strip the leading path component (the remote path prefix) so that
		// extracted paths are relative to destDir.
		name := filepath.Clean(hdr.Name)
		destPath := filepath.Join(destDir, name)

		// Guard against zip-slip path traversal.
		if !isSubPath(destDir, destPath) {
			return fileCount, fmt.Errorf("invalid path in archive: %q", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, os.FileMode(hdr.Mode)|0700); err != nil {
				return fileCount, fmt.Errorf("failed to create directory %q: %w", destPath, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(destPath), 0750); err != nil {
				return fileCount, fmt.Errorf("failed to create parent directory for %q: %w", destPath, err)
			}
			f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)|0600)
			if err != nil {
				return fileCount, fmt.Errorf("failed to create file %q: %w", destPath, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fileCount, fmt.Errorf("failed to write file %q: %w", destPath, err)
			}
			f.Close()
			fileCount++
			fmt.Printf("  pulled: %s\n", hdr.Name)
		}
	}

	return fileCount, nil
}

// isSubPath returns true if target is within (or equal to) base.
func isSubPath(base, target string) bool {
	base = filepath.Clean(base)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel == "." || !strings.HasPrefix(rel, "..")
}
