package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var (
	describeOutput string
)

var describeCmd = &cobra.Command{
	Use:   "describe <pod-name>",
	Short: "Show detailed information about a Pod",
	Long:  `Display detailed specification and status of a Kubernetes Pod, including containers, labels, annotations, and recent events.`,
	Example: `  # Describe a pod in human-readable format
  k-cli describe my-pod -n default

  # Output raw YAML
  k-cli describe my-pod -n default -o yaml`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		podName := args[0]

		Logger.Info("Describing pod",
			zap.String("pod", podName),
			zap.String("namespace", namespace),
		)

		pod, err := K8sClient.Clientset.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get pod %q in namespace %q: %w\nHint: check the pod name and namespace", podName, namespace, err)
		}

		if describeOutput == "yaml" {
			return printPodYAML(pod)
		}

		events, err := fetchPodEvents(namespace, pod)
		if err != nil && verbose {
			Logger.Warn("Could not fetch events", zap.Error(err))
		}

		printPodDetails(pod, events)
		return nil
	},
}

func init() {
	describeCmd.Flags().StringVarP(&describeOutput, "output", "o", "", "output format (yaml)")
	rootCmd.AddCommand(describeCmd)
}

// printPodYAML serializes the Pod object to YAML and prints it.
func printPodYAML(pod *corev1.Pod) error {
	data, err := yaml.Marshal(pod)
	if err != nil {
		return fmt.Errorf("failed to marshal pod to YAML: %w", err)
	}
	fmt.Print(string(data))
	return nil
}

// fetchPodEvents retrieves events associated with the given pod.
func fetchPodEvents(ns string, pod *corev1.Pod) ([]corev1.Event, error) {
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s,involvedObject.uid=%s",
		pod.Name, ns, string(pod.UID))

	eventList, err := K8sClient.Clientset.CoreV1().Events(ns).List(context.Background(), metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, err
	}
	return eventList.Items, nil
}

// printPodDetails prints a human-readable summary of a pod.
func printPodDetails(pod *corev1.Pod, events []corev1.Event) {
	fmt.Printf("Name:       %s\n", pod.Name)
	fmt.Printf("Namespace:  %s\n", pod.Namespace)
	fmt.Printf("Node:       %s\n", strOrDash(pod.Spec.NodeName))
	fmt.Printf("Status:     %s\n", string(pod.Status.Phase))
	fmt.Printf("IP:         %s\n", strOrDash(pod.Status.PodIP))
	fmt.Printf("Start Time: %s\n", timeOrDash(pod.Status.StartTime))

	// Labels
	fmt.Println("\nLabels:")
	if len(pod.Labels) == 0 {
		fmt.Println("  <none>")
	} else {
		for _, k := range sortedKeys(pod.Labels) {
			fmt.Printf("  %s=%s\n", k, pod.Labels[k])
		}
	}

	// Annotations
	fmt.Println("\nAnnotations:")
	if len(pod.Annotations) == 0 {
		fmt.Println("  <none>")
	} else {
		for _, k := range sortedKeys(pod.Annotations) {
			fmt.Printf("  %s=%s\n", k, pod.Annotations[k])
		}
	}

	// Containers
	fmt.Println("\nContainers:")
	for _, c := range pod.Spec.Containers {
		fmt.Printf("  %s:\n", c.Name)
		fmt.Printf("    Image: %s\n", c.Image)

		if len(c.Ports) > 0 {
			portStrs := make([]string, 0, len(c.Ports))
			for _, p := range c.Ports {
				portStrs = append(portStrs, fmt.Sprintf("%d/%s", p.ContainerPort, p.Protocol))
			}
			fmt.Printf("    Ports: %s\n", strings.Join(portStrs, ", "))
		}

		if c.Resources.Limits != nil || c.Resources.Requests != nil {
			fmt.Printf("    Resources:\n")
			if c.Resources.Limits != nil {
				fmt.Printf("      Limits: cpu=%s, memory=%s\n",
					resourceOrDash(c.Resources.Limits, corev1.ResourceCPU),
					resourceOrDash(c.Resources.Limits, corev1.ResourceMemory))
			}
			if c.Resources.Requests != nil {
				fmt.Printf("      Requests: cpu=%s, memory=%s\n",
					resourceOrDash(c.Resources.Requests, corev1.ResourceCPU),
					resourceOrDash(c.Resources.Requests, corev1.ResourceMemory))
			}
		}

		if len(c.Env) > 0 {
			fmt.Printf("    Environment:\n")
			for _, e := range c.Env {
				if e.ValueFrom != nil {
					fmt.Printf("      %s: <from ref>\n", e.Name)
				} else {
					fmt.Printf("      %s=%s\n", e.Name, e.Value)
				}
			}
		}
	}

	// Container statuses
	if len(pod.Status.ContainerStatuses) > 0 {
		fmt.Println("\nContainer Statuses:")
		for _, cs := range pod.Status.ContainerStatuses {
			ready := "false"
			if cs.Ready {
				ready = "true"
			}
			fmt.Printf("  %s: ready=%s, restarts=%d\n", cs.Name, ready, cs.RestartCount)
		}
	}

	// Events
	fmt.Println("\nEvents:")
	if len(events) == 0 {
		fmt.Println("  <none>")
	} else {
		fmt.Printf("  %-8s %-20s %-25s %s\n", "Type", "Reason", "Source", "Message")
		fmt.Printf("  %s\n", strings.Repeat("-", 80))
		for _, e := range events {
			fmt.Printf("  %-8s %-20s %-25s %s\n",
				e.Type,
				e.Reason,
				e.Source.Component,
				e.Message,
			)
		}
	}
}

// sortedKeys returns the keys of a map in sorted order.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// strOrDash returns the string or "-" if empty.
func strOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// timeOrDash returns a formatted time string or "-" if nil.
func timeOrDash(t *metav1.Time) string {
	if t == nil {
		return "-"
	}
	return t.String()
}

// resourceOrDash returns the string representation of a resource quantity or "-".
func resourceOrDash(rl corev1.ResourceList, name corev1.ResourceName) string {
	if q, ok := rl[name]; ok {
		return q.String()
	}
	return "-"
}
