package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	k8sclient "github.com/IceRiverDev/k-cli/internal/k8s"
)

var (
	kubeconfig string
	namespace  string
	enableLog  bool

	// Logger is the global structured logger.
	Logger *zap.Logger

	// K8sClient is the shared Kubernetes client initialized in PersistentPreRunE.
	K8sClient *k8sclient.Client
)

// rootCmd is the base command for k-cli.
var rootCmd = &cobra.Command{
	Use:   "k-cli",
	Short: "A kubectl-like CLI tool for managing Kubernetes Pods",
	Long: `k-cli is a production-grade CLI tool built with Cobra that lets you
manage Kubernetes Pods — sync files, pull remote paths, diagnose health, and inspect secrets.

By default, log output is disabled (silent mode). Use --log to enable debug-level logging.`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		initLogger(enableLog)

		client, err := k8sclient.NewClient(kubeconfig)
		if err != nil {
			return fmt.Errorf("could not connect to Kubernetes: %w\nHint: check --kubeconfig or the KUBECONFIG env var", err)
		}
		K8sClient = client

		Logger.Info("Kubernetes client initialized",
			zap.String("kubeconfig", kubeconfig),
			zap.String("namespace", namespace),
		)
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
	rootCmd.PersistentFlags().BoolVar(&enableLog, "log", false, "enable debug-level log output")

	_ = viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig"))
	_ = viper.BindPFlag("namespace", rootCmd.PersistentFlags().Lookup("namespace"))

	// Disable Cobra's built-in completion command; replaced by cmd/completion.go.
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// initConfig reads in config file and ENV variables.
func initConfig() {
	viper.SetEnvPrefix("KCLI")
	viper.AutomaticEnv()

	if kc := viper.GetString("kubeconfig"); kc == "" {
		if env := os.Getenv("KUBECONFIG"); env != "" {
			viper.Set("kubeconfig", env)
		}
	}
}

// initLogger configures the global zap logger.
// When enableLog is true, debug-level structured logging is written to stderr.
func initLogger(enableLog bool) {
	if !enableLog {
		Logger = zap.NewNop()
		return
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var err error
	Logger, err = cfg.Build()
	if err != nil {
		Logger = zap.NewNop()
	}
}
