package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

var (
	execContainer string
)

var execCmd = &cobra.Command{
	Use:   "exec <pod-name>",
	Short: "Enter an interactive shell inside a Pod",
	Long:  `Open an interactive TTY shell session in a running Kubernetes Pod container.`,
	Example: `  # Enter the default container in a pod
  phoenix exec my-pod -n default

  # Enter a specific container
  phoenix exec my-pod -n default -c main`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		podName := args[0]

		Logger.Info("Exec into pod",
			zap.String("pod", podName),
			zap.String("namespace", namespace),
			zap.String("container", execContainer),
		)

		// Try /bin/bash first, fall back to /bin/sh.
		shells := []string{"/bin/bash", "/bin/sh"}
		var lastErr error
		for _, shell := range shells {
			if err := execInPod(cmd.Context(), podName, namespace, execContainer, shell); err != nil {
				lastErr = err
				if verbose {
					Logger.Warn("Shell not available, trying next",
						zap.String("shell", shell),
						zap.Error(err),
					)
				}
				continue
			}
			return nil
		}
		return fmt.Errorf("could not exec into pod %q: %w\nHint: ensure the pod is running and the container has a shell", podName, lastErr)
	},
}

func init() {
	execCmd.Flags().StringVarP(&execContainer, "container", "c", "", "container name (defaults to the first container)")
	rootCmd.AddCommand(execCmd)
}

// execInPod opens an interactive shell session in the given pod/container.
func execInPod(ctx context.Context, podName, ns, container, shell string) error {
	req := K8sClient.Clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(ns).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{shell},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(K8sClient.RestConfig, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to create SPDY executor: %w", err)
	}

	return executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    true,
	})
}
