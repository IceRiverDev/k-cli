package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose <pod-name>",
	Short: "Diagnose Pod health and give actionable suggestions",
	Long:  `Inspect a Kubernetes Pod's status, restart history, resource limits, container readiness, and recent events to produce a concise health report with suggestions.`,
	Example: `  # Diagnose a pod
  k-cli diagnose my-pod -n default

  # Diagnose in another namespace
  k-cli diagnose my-pod -n production`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		podName := args[0]

		Logger.Info("Diagnosing pod",
			zap.String("pod", podName),
			zap.String("namespace", namespace),
		)

		return diagnosePod(cmd.Context(), podName, namespace)
	},
}

func init() {
	rootCmd.AddCommand(diagnoseCmd)
}

func diagnosePod(ctx context.Context, podName, ns string) error {
	pod, err := K8sClient.Clientset.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod %q in namespace %q: %w\nHint: check the pod name and namespace", podName, ns, err)
	}

	// Fetch recent events (limit 10).
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s", pod.Name, ns)
	eventList, err := K8sClient.Clientset.CoreV1().Events(ns).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
		Limit:         10,
	})
	if err != nil {
		Logger.Warn("Could not fetch events", zap.Error(err))
	}

	var events []corev1.Event
	if eventList != nil {
		events = eventList.Items
	}

	printDiagnosis(pod, events)
	return nil
}

func printDiagnosis(pod *corev1.Pod, events []corev1.Event) {
	criticalCount := 0
	warningCount := 0

	fmt.Printf("🔍 Diagnosing pod: %s (namespace: %s)\n", pod.Name, pod.Namespace)
	fmt.Println(strings.Repeat("─", 45))

	// 1. Pod status.
	phase := string(pod.Status.Phase)
	switch pod.Status.Phase {
	case corev1.PodRunning:
		fmt.Printf("✅ Status:        %s\n", phase)
	case corev1.PodPending:
		fmt.Printf("⚠️  Status:        %s\n", phase)
		fmt.Println("   💡 Suggestion: check node resources or image pull status")
		warningCount++
	case corev1.PodFailed:
		fmt.Printf("❌ Status:        %s\n", phase)
		fmt.Println("   💡 Suggestion: inspect logs with kubectl logs " + pod.Name)
		criticalCount++
	default:
		fmt.Printf("⚠️  Status:        %s\n", phase)
		warningCount++
	}

	// 2. Restart count check.
	totalRestarts := int32(0)
	lastRestartReason := ""
	for _, cs := range pod.Status.ContainerStatuses {
		totalRestarts += cs.RestartCount
		if cs.LastTerminationState.Terminated != nil {
			lastRestartReason = cs.LastTerminationState.Terminated.Reason
		}
	}
	if totalRestarts > 5 {
		reasonStr := ""
		if lastRestartReason != "" {
			reasonStr = fmt.Sprintf(" (last reason: %s)", lastRestartReason)
		}
		fmt.Printf("❌ Restarts:      %d%s\n", totalRestarts, reasonStr)
		fmt.Println("   💡 Suggestion: check logs and consider resource limits or liveness probe tuning")
		criticalCount++
	} else if totalRestarts > 0 {
		reasonStr := ""
		if lastRestartReason != "" {
			reasonStr = fmt.Sprintf(" (last reason: %s)", lastRestartReason)
		}
		fmt.Printf("⚠️  Restarts:      %d%s\n", totalRestarts, reasonStr)
		warningCount++
	} else {
		fmt.Printf("✅ Restarts:      0\n")
	}

	// 3. OOMKilled detection.
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.LastTerminationState.Terminated != nil &&
			cs.LastTerminationState.Terminated.Reason == "OOMKilled" {
			memLimit := "-"
			suggestion := "set and increase memory limit"
			for _, c := range pod.Spec.Containers {
				if c.Name == cs.Name && c.Resources.Limits != nil {
					if q, ok := c.Resources.Limits[corev1.ResourceMemory]; ok {
						memLimit = q.String()
						// Suggest doubling the current limit.
						doubled := q.DeepCopy()
						doubled.Add(q)
						suggestion = fmt.Sprintf("increase memory limit to at least %s", doubled.String())
					}
				}
			}
			fmt.Printf("❌ Memory:        OOMKilled detected — current limit: %s\n", memLimit)
			fmt.Printf("   💡 Suggestion: %s\n", suggestion)
			criticalCount++
		}
	}

	// 4. Resource limits/requests check.
	for _, c := range pod.Spec.Containers {
		if c.Resources.Limits == nil {
			fmt.Printf("⚠️  Resources:     No limits set for container %q\n", c.Name)
			fmt.Println("   💡 Suggestion: set resources.limits.cpu and resources.limits.memory")
			warningCount++
		} else {
			if _, hasCPU := c.Resources.Limits[corev1.ResourceCPU]; !hasCPU {
				fmt.Printf("⚠️  Resources:     No CPU limit set for container %q\n", c.Name)
				fmt.Println("   💡 Suggestion: set resources.limits.cpu to avoid noisy neighbor")
				warningCount++
			}
		}
		if c.Resources.Requests == nil {
			fmt.Printf("⚠️  Resources:     No requests set for container %q\n", c.Name)
			fmt.Println("   💡 Suggestion: set resources.requests for proper scheduling")
			warningCount++
		}
	}

	// 5. Ready status check.
	totalContainers := len(pod.Status.ContainerStatuses)
	readyContainers := 0
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			readyContainers++
		}
	}
	if totalContainers > 0 {
		if readyContainers == totalContainers {
			fmt.Printf("✅ Containers:    %d/%d Ready\n", readyContainers, totalContainers)
		} else {
			fmt.Printf("❌ Containers:    %d/%d Ready\n", readyContainers, totalContainers)
			for _, cs := range pod.Status.ContainerStatuses {
				if !cs.Ready {
					reason := getContainerNotReadyReason(cs)
					fmt.Printf("   Container %q not ready: %s\n", cs.Name, reason)
				}
			}
			criticalCount++
		}
	}

	fmt.Println(strings.Repeat("─", 45))

	// 6. Recent events.
	if len(events) > 0 {
		displayCount := len(events)
		if displayCount > 5 {
			displayCount = 5
		}
		fmt.Printf("📋 Recent Events (last %d):\n", displayCount)
		for _, e := range events[:displayCount] {
			prefix := "   [Normal] "
			if e.Type == corev1.EventTypeWarning {
				prefix = "   [Warning]"
			}
			fmt.Printf("%s %s: %s\n", prefix, e.Reason, e.Message)
		}
		fmt.Println(strings.Repeat("─", 45))
	}

	fmt.Printf("🏁 Diagnosis complete: %d critical, %d warnings\n", criticalCount, warningCount)
}

func getContainerNotReadyReason(cs corev1.ContainerStatus) string {
	if cs.State.Waiting != nil {
		if cs.State.Waiting.Message != "" {
			return fmt.Sprintf("%s: %s", cs.State.Waiting.Reason, cs.State.Waiting.Message)
		}
		return cs.State.Waiting.Reason
	}
	if cs.State.Terminated != nil {
		return fmt.Sprintf("terminated with reason: %s", cs.State.Terminated.Reason)
	}
	return "unknown"
}
