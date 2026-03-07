package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	k8sclient "github.com/IceRiverDev/simple-cli/internal/k8s"
)

var (
	kubeconfig string
	namespace  string
	verbose    bool

	// Logger is the global structured logger.
	Logger *zap.Logger

	// K8sClient is the shared Kubernetes client initialized in PersistentPreRunE.
	K8sClient *k8sclient.Client
)

// rootCmd is the base command for phoenix.
var rootCmd = &cobra.Command{
	Use:   "phoenix",
	Short: "A kubectl-like CLI tool for managing Kubernetes Pods",
	Long: `phoenix is a production-grade CLI tool built with Cobra that lets you
manage Kubernetes Pods — create, delete, describe, exec into, and sync files.`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		initLogger(verbose)

		client, err := k8sclient.NewClient(kubeconfig)
		if err != nil {
			return fmt.Errorf("could not connect to Kubernetes: %w\nHint: check --kubeconfig or the KUBECONFIG env var", err)
		}
		K8sClient = client

		if verbose {
			Logger.Info("Kubernetes client initialized",
				zap.String("kubeconfig", kubeconfig),
				zap.String("namespace", namespace),
			)
		}
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "path to the kubeconfig file (default: ~/.kube/config, or $KUBECONFIG)")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "default Kubernetes namespace")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")

	_ = viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig"))
	_ = viper.BindPFlag("namespace", rootCmd.PersistentFlags().Lookup("namespace"))
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables.
func initConfig() {
	viper.SetEnvPrefix("PHOENIX")
	viper.AutomaticEnv()

	if kc := viper.GetString("kubeconfig"); kc == "" {
		if env := os.Getenv("KUBECONFIG"); env != "" {
			viper.Set("kubeconfig", env)
		}
	}
}

// initLogger configures the global zap logger.
func initLogger(debug bool) {
	level := zapcore.InfoLevel
	if debug {
		level = zapcore.DebugLevel
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var err error
	Logger, err = cfg.Build()
	if err != nil {
		// Fall back to a no-op logger if build fails.
		Logger = zap.NewNop()
	}
}
