/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/konstructio/kubefirst/internal/talosbootstrap"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TalosBootstrapConfig represents the configuration for Talos bootstrap
type TalosBootstrapConfig struct {
	// Cloud Provider Configuration
	CloudProvider string `mapstructure:"cloudProvider"`
	CivoAPIKey    string `mapstructure:"civoAPIKey"`
	ClusterRegion string `mapstructure:"clusterRegion"`

	// DNS Provider Configuration
	DNSProvider        string `mapstructure:"dnsProvider"`
	DomainName         string `mapstructure:"domainName"`
	Subdomain          string `mapstructure:"subdomain"`
	CloudflareAPIToken string `mapstructure:"cloudflareAPIToken"`
	CloudflareZoneID   string `mapstructure:"cloudflareZoneId"`

	// Git Provider Configuration
	GitProvider string `mapstructure:"gitProvider"`
	GitUser     string `mapstructure:"gitUser"`
	GitPAT      string `mapstructure:"gitPat"`
	GitRepo     string `mapstructure:"gitRepo"`

	// Cluster Configuration
	ClusterName              string `mapstructure:"clusterName"`
	AlertsEmail              string `mapstructure:"alertsEmail"`
	TalosVersion             string `mapstructure:"talosVersion"`
	KubernetesVersion        string `mapstructure:"kubernetesVersion"`
	ControlplaneNodeCount    int    `mapstructure:"controlplaneNodeCount"`
	ControlplaneNodeSize     string `mapstructure:"controlplaneNodeSize"`
	WorkerNodeCount          int    `mapstructure:"workerNodeCount"`
	WorkerNodeSize           string `mapstructure:"workerNodeSize"`

	// Advanced Configuration
	EnableCNI             bool `mapstructure:"enableCNI"`
	EnableCSI             bool `mapstructure:"enableCSI"`
	EnableIngress         bool `mapstructure:"enableIngress"`
	EnableArgoCD          bool `mapstructure:"enableArgoCD"`
	EnableVault           bool `mapstructure:"enableVault"`
	EnableTelemetry       bool `mapstructure:"enableTelemetry"`
	EnableBackups         bool `mapstructure:"enableBackups"`
	EnableMonitoring      bool `mapstructure:"enableMonitoring"`
	EnableCertManager     bool `mapstructure:"enableCertManager"`
	EnableExternalDNS     bool `mapstructure:"enableExternalDNS"`
	EnableExternalSecrets bool `mapstructure:"enableExternalSecrets"`
	EnableKubefirst       bool `mapstructure:"enableKubefirst"`
	EnableMetaphor        bool `mapstructure:"enableMetaphor"`
}

// TalosBootstrap creates a new command for Talos bootstrap
func TalosBootstrap() *cobra.Command {
	talosBootstrapCmd := &cobra.Command{
		Use:   "talos-bootstrap",
		Short: "Bootstrap a kubefirst management cluster on Civo using Talos OS",
		Long: `Bootstrap a kubefirst management cluster on Civo using Talos OS.

This command will:
1. Validate the configuration
2. Provision Civo infrastructure (network, firewall, instances)
3. Bootstrap the Talos cluster
4. Set up GitOps repositories
5. Generate Terraform templates for subsequent deployment

Example:
  kubefirst civo talos-bootstrap --config-path ./config.yaml`,
		RunE: runTalosBootstrap,
	}

	// Add flags
	talosBootstrapCmd.Flags().String("config-path", "config.yaml", "Path to the configuration file")
	talosBootstrapCmd.Flags().Bool("dry-run", false, "Perform a dry run without making changes")
	talosBootstrapCmd.Flags().Bool("verbose", false, "Enable verbose logging")
	talosBootstrapCmd.MarkFlagRequired("config-path")

	return talosBootstrapCmd
}

func runTalosBootstrap(cmd *cobra.Command, args []string) error {
	// Get flags
	configPath, _ := cmd.Flags().GetString("config-path")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Load configuration
	config, err := loadTalosBootstrapConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := validateTalosBootstrapConfig(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Import and use the bootstrap logic from your kubefirst-talos project
	// For now, we'll integrate the core functionality here
	return executeTalosBootstrap(config, dryRun, verbose)
}

func loadTalosBootstrapConfig(path string) (*TalosBootstrapConfig, error) {
	// Clean and validate the path
	configPath := filepath.Clean(path)
	
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file does not exist: %s", configPath)
	}

	// Set up viper
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	
	// Set environment variable substitution
	viper.SetEnvPrefix("KUBEFIRST")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read configuration
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal to struct
	var config TalosBootstrapConfig
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func validateTalosBootstrapConfig(config *TalosBootstrapConfig) error {
	// Required fields validation
	if config.ClusterName == "" {
		return fmt.Errorf("clusterName is required")
	}
	if config.CloudProvider != "civo" {
		return fmt.Errorf("only 'civo' cloud provider is supported")
	}
	if config.CivoAPIKey == "" {
		return fmt.Errorf("civoAPIKey is required")
	}
	if config.DomainName == "" {
		return fmt.Errorf("domainName is required")
	}
	if config.AlertsEmail == "" {
		return fmt.Errorf("alertsEmail is required")
	}
	if config.DNSProvider != "civo" && config.DNSProvider != "cloudflare" {
		return fmt.Errorf("only 'civo' and 'cloudflare' DNS providers are supported")
	}
	if config.GitProvider != "github" && config.GitProvider != "gitlab" {
		return fmt.Errorf("only 'github' and 'gitlab' git providers are supported")
	}
	if config.ControlplaneNodeCount < 1 {
		return fmt.Errorf("controlplaneNodeCount must be at least 1")
	}
	if config.WorkerNodeCount < 0 {
		return fmt.Errorf("workerNodeCount must be 0 or greater")
	}

	// DNS provider specific validation
	if config.DNSProvider == "cloudflare" {
		if config.CloudflareAPIToken == "" {
			return fmt.Errorf("cloudflareAPIToken is required when using cloudflare DNS provider")
		}
	}

	// Git provider specific validation
	if config.GitProvider == "github" || config.GitProvider == "gitlab" {
		if config.GitPAT == "" {
			return fmt.Errorf("gitPat is required")
		}
		if config.GitUser == "" {
			return fmt.Errorf("gitUser is required")
		}
	}

	return nil
}

func executeTalosBootstrap(config *TalosBootstrapConfig, dryRun, verbose bool) error {
	if dryRun {
		fmt.Println("🧪 DRY RUN MODE - No resources will be created")
		fmt.Println("")
		fmt.Printf("Would bootstrap Talos cluster with configuration:\n")
		fmt.Printf("  Cluster Name: %s\n", config.ClusterName)
		fmt.Printf("  Region: %s\n", config.ClusterRegion)
		fmt.Printf("  Domain: %s\n", config.DomainName)
		fmt.Printf("  Control Plane Nodes: %d (%s)\n", config.ControlplaneNodeCount, config.ControlplaneNodeSize)
		fmt.Printf("  Worker Nodes: %d (%s)\n", config.WorkerNodeCount, config.WorkerNodeSize)
		fmt.Printf("  Talos Version: %s\n", config.TalosVersion)
		fmt.Printf("  Kubernetes Version: %s\n", config.KubernetesVersion)
		fmt.Printf("  DNS Provider: %s\n", config.DNSProvider)
		fmt.Printf("  Git Provider: %s\n", config.GitProvider)
		fmt.Println("")
		fmt.Println("✅ Dry run completed successfully!")
		return nil
	}

	// Convert config to internal format
	internalConfig := &talosbootstrap.Config{
		CloudProvider:            config.CloudProvider,
		CivoAPIKey:              config.CivoAPIKey,
		ClusterRegion:           config.ClusterRegion,
		DNSProvider:             config.DNSProvider,
		DomainName:              config.DomainName,
		Subdomain:               config.Subdomain,
		CloudflareAPIToken:      config.CloudflareAPIToken,
		CloudflareZoneID:        config.CloudflareZoneID,
		GitProvider:             config.GitProvider,
		GitUser:                 config.GitUser,
		GitPAT:                  config.GitPAT,
		GitRepo:                 config.GitRepo,
		ClusterName:             config.ClusterName,
		AlertsEmail:             config.AlertsEmail,
		TalosVersion:            config.TalosVersion,
		KubernetesVersion:       config.KubernetesVersion,
		ControlplaneNodeCount:   config.ControlplaneNodeCount,
		ControlplaneNodeSize:    config.ControlplaneNodeSize,
		WorkerNodeCount:         config.WorkerNodeCount,
		WorkerNodeSize:          config.WorkerNodeSize,
		EnableCNI:               config.EnableCNI,
		EnableCSI:               config.EnableCSI,
		EnableIngress:           config.EnableIngress,
		EnableArgoCD:            config.EnableArgoCD,
		EnableVault:             config.EnableVault,
		EnableTelemetry:         config.EnableTelemetry,
		EnableBackups:           config.EnableBackups,
		EnableMonitoring:        config.EnableMonitoring,
		EnableCertManager:       config.EnableCertManager,
		EnableExternalDNS:       config.EnableExternalDNS,
		EnableExternalSecrets:   config.EnableExternalSecrets,
		EnableKubefirst:         config.EnableKubefirst,
		EnableMetaphor:          config.EnableMetaphor,
	}

	// Create bootstrapper
	bootstrapper := talosbootstrap.NewBootstrapper(internalConfig, talosbootstrap.Options{
		DryRun:  dryRun,
		Verbose: verbose,
	})

	// Execute bootstrap
	return bootstrapper.Bootstrap()
}
