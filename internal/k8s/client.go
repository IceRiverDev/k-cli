package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the Kubernetes clientset and REST config.
type Client struct {
	Clientset  *kubernetes.Clientset
	RestConfig *rest.Config
}

// NewClient creates a new Kubernetes client using the provided kubeconfig path.
// If kubeconfigPath is empty, it falls back to the KUBECONFIG env var or
// the default ~/.kube/config location.
func NewClient(kubeconfigPath string) (*Client, error) {
	if kubeconfigPath == "" {
		if env := os.Getenv("KUBECONFIG"); env != "" {
			kubeconfigPath = env
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("cannot determine home directory: %w", err)
			}
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig from %q: %w\nHint: ensure your kubeconfig file exists and is valid", kubeconfigPath, err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	return &Client{
		Clientset:  clientset,
		RestConfig: config,
	}, nil
}
