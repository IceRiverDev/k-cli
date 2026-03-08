package cmd

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	secretKey   string
	showEncoded bool
)

var secretCmd = &cobra.Command{
	Use:   "secret <secret-name>",
	Short: "View a Kubernetes Secret with decoded values",
	Long:  `Fetch a Kubernetes Secret and automatically decode all base64-encoded values for easy inspection.`,
	Example: `  # Show all keys in a secret (decoded)
  k-cli secret my-secret -n default

  # Show only a specific key
  k-cli secret my-secret --key DB_PASSWORD

  # Show both decoded and encoded values
  k-cli secret my-secret --show-encoded`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		secretName := args[0]
		secret, err := K8sClient.Clientset.CoreV1().Secrets(namespace).Get(
			cmd.Context(), secretName, metav1.GetOptions{},
		)
		if err != nil {
			return fmt.Errorf("failed to get secret %q in namespace %q: %w\nHint: check the secret name and namespace", secretName, namespace, err)
		}

		fmt.Printf("🔐 Secret: %s (namespace: %s, type: %s)\n", secret.Name, secret.Namespace, secret.Type)
		fmt.Println(strings.Repeat("─", 50))

		keys := make([]string, 0, len(secret.Data))
		for k := range secret.Data {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		if secretKey != "" {
			if _, ok := secret.Data[secretKey]; !ok {
				return fmt.Errorf("key %q not found in secret %q\nAvailable keys: %s", secretKey, secretName, strings.Join(keys, ", "))
			}
		}

		for _, k := range keys {
			if secretKey != "" && k != secretKey {
				continue
			}
			decoded := string(secret.Data[k])
			if showEncoded {
				encoded := base64.StdEncoding.EncodeToString(secret.Data[k])
				fmt.Printf("  %-30s %s\n              (base64) %s\n", k+":", decoded, encoded)
			} else {
				fmt.Printf("  %-30s %s\n", k+":", decoded)
			}
		}

		fmt.Println(strings.Repeat("─", 50))
		fmt.Printf("Total: %d key(s)\n", len(keys))
		return nil
	},
}

func init() {
	secretCmd.Flags().StringVar(&secretKey, "key", "", "show only this specific key")
	secretCmd.Flags().BoolVar(&showEncoded, "show-encoded", false, "also show the original base64 encoded value")
	rootCmd.AddCommand(secretCmd)
}
