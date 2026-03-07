package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	deleteForce bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete <pod-name>",
	Short: "Delete a Pod",
	Long:  `Delete a Kubernetes Pod by name, with an option for a forced (immediate) deletion.`,
	Example: `  # Delete a pod gracefully
  phoenix delete my-pod -n default

  # Force-delete a pod immediately
  phoenix delete my-pod -n default --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		podName := args[0]

		Logger.Info("Deleting pod",
			zap.String("pod", podName),
			zap.String("namespace", namespace),
			zap.Bool("force", deleteForce),
		)

		opts := metav1.DeleteOptions{}
		if deleteForce {
			gracePeriod := int64(0)
			opts.GracePeriodSeconds = &gracePeriod
		}

		if err := K8sClient.Clientset.CoreV1().Pods(namespace).Delete(context.Background(), podName, opts); err != nil {
			return fmt.Errorf("failed to delete pod %q in namespace %q: %w\nHint: check that the pod exists and you have permission to delete it", podName, namespace, err)
		}

		if deleteForce {
			fmt.Printf("Pod %q in namespace %q force-deleted (grace period = 0)\n", podName, namespace)
		} else {
			fmt.Printf("Pod %q in namespace %q deleted successfully\n", podName, namespace)
		}
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVar(&deleteForce, "force", false, "force deletion with grace period of 0")
	rootCmd.AddCommand(deleteCmd)
}
