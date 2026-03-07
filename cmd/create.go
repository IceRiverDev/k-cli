package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	createImage  string
	createPort   int
	createEnvs   []string
	createLabels []string
)

var createCmd = &cobra.Command{
	Use:   "create <pod-name>",
	Short: "Create a Pod based on an image",
	Long:  `Create a new Kubernetes Pod from a specified container image with optional port, environment variables, and labels.`,
	Example: `  # Create a simple pod
  phoenix create my-pod --image nginx:latest -n default

  # Create with port and environment variables
  phoenix create my-pod --image nginx:latest --port 80 --env ENV=production --env VERSION=1.0

  # Create with labels
  phoenix create my-pod --image nginx:latest --labels app=my-app --labels tier=frontend`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		podName := args[0]

		if createImage == "" {
			return fmt.Errorf("--image is required\nHint: specify a container image, e.g. --image nginx:latest")
		}

		Logger.Info("Creating pod",
			zap.String("pod", podName),
			zap.String("namespace", namespace),
			zap.String("image", createImage),
		)

		pod, err := buildPod(podName, namespace, createImage, createPort, createEnvs, createLabels)
		if err != nil {
			return fmt.Errorf("failed to build pod spec: %w", err)
		}

		created, err := K8sClient.Clientset.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create pod %q in namespace %q: %w\nHint: check that the namespace exists and you have permission to create pods", podName, namespace, err)
		}

		fmt.Printf("Pod %q created successfully in namespace %q\n", created.Name, created.Namespace)
		fmt.Printf("Status: %s\n", created.Status.Phase)
		fmt.Printf("Image:  %s\n", createImage)
		if createPort > 0 {
			fmt.Printf("Port:   %d\n", createPort)
		}
		return nil
	},
}

func init() {
	createCmd.Flags().StringVar(&createImage, "image", "", "container image to use (required)")
	createCmd.Flags().IntVar(&createPort, "port", 0, "port to expose from the container (optional)")
	createCmd.Flags().StringArrayVar(&createEnvs, "env", nil, "environment variables in KEY=VALUE format (can be repeated)")
	createCmd.Flags().StringArrayVar(&createLabels, "labels", nil, "pod labels in KEY=VALUE format (can be repeated)")
	_ = createCmd.MarkFlagRequired("image")
	rootCmd.AddCommand(createCmd)
}

// buildPod constructs a Pod object from the given parameters.
func buildPod(podName, ns, image string, port int, envs, labels []string) (*corev1.Pod, error) {
	envVars, err := parseKeyValuePairs(envs)
	if err != nil {
		return nil, fmt.Errorf("invalid --env value: %w", err)
	}

	labelMap, err := parseKeyValuePairs(labels)
	if err != nil {
		return nil, fmt.Errorf("invalid --labels value: %w", err)
	}

	// Ensure the pod name label is always present for easier selection.
	if labelMap == nil {
		labelMap = map[string]string{}
	}
	labelMap["app"] = podName

	container := corev1.Container{
		Name:  podName,
		Image: image,
	}

	if port > 0 {
		container.Ports = []corev1.ContainerPort{
			{
				ContainerPort: int32(port),
				Protocol:      corev1.ProtocolTCP,
			},
		}
		container.ReadinessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(port),
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
		}
	}

	for k, v := range envVars {
		container.Env = append(container.Env, corev1.EnvVar{Name: k, Value: v})
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
			Labels:    labelMap,
		},
		Spec: corev1.PodSpec{
			Containers:    []corev1.Container{container},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	return pod, nil
}

// parseKeyValuePairs converts a slice of "KEY=VALUE" strings into a map.
func parseKeyValuePairs(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	result := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("%q is not in KEY=VALUE format", pair)
		}
		result[parts[0]] = parts[1]
	}
	return result, nil
}
