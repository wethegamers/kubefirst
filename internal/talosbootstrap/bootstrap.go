/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package talosbootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/konstructio/kubefirst/internal/talosbootstrap/civo"
	"github.com/konstructio/kubefirst/internal/talosbootstrap/git"
	"github.com/konstructio/kubefirst/internal/talosbootstrap/state"
	"github.com/konstructio/kubefirst/internal/talosbootstrap/talos"
	"github.com/konstructio/kubefirst/internal/talosbootstrap/utils"
	"github.com/rs/zerolog/log"
)

// Bootstrapper handles the Talos cluster bootstrap process
type Bootstrapper struct {
	config       *Config
	options      Options
	stateDir     string
	civoClient   *civo.Client
	gitClient    *git.Client
	talosClient  *talos.Client
	stateManager *state.Manager
}

// Config represents the Talos bootstrap configuration
type Config struct {
	// Cloud Provider Configuration
	CloudProvider string
	CivoAPIKey    string
	ClusterRegion string

	// DNS Provider Configuration
	DNSProvider        string
	DomainName         string
	Subdomain          string
	CloudflareAPIToken string
	CloudflareZoneID   string

	// Git Provider Configuration
	GitProvider string
	GitUser     string
	GitPAT      string
	GitRepo     string

	// Cluster Configuration
	ClusterName              string
	AlertsEmail              string
	TalosVersion             string
	KubernetesVersion        string
	ControlplaneNodeCount    int
	ControlplaneNodeSize     string
	WorkerNodeCount          int
	WorkerNodeSize           string

	// Advanced Configuration
	EnableCNI             bool
	EnableCSI             bool
	EnableIngress         bool
	EnableArgoCD          bool
	EnableVault           bool
	EnableTelemetry       bool
	EnableBackups         bool
	EnableMonitoring      bool
	EnableCertManager     bool
	EnableExternalDNS     bool
	EnableExternalSecrets bool
	EnableKubefirst       bool
	EnableMetaphor        bool
}

// Options for the bootstrap process
type Options struct {
	DryRun  bool
	Verbose bool
}

// NewBootstrapper creates a new Talos bootstrapper
func NewBootstrapper(config *Config, options Options) *Bootstrapper {
	// Set up state directory
	homeDir, _ := os.UserHomeDir()
	stateDir := filepath.Join(homeDir, ".k1", config.ClusterName)

	return &Bootstrapper{
		config:   config,
		options:  options,
		stateDir: stateDir,
	}
}

// Bootstrap executes the complete bootstrap process
func (b *Bootstrapper) Bootstrap() error {
	log.Info().Msg("Starting kubefirst Talos bootstrap process")

	// Phase 1: Initialize clients and state
	if err := b.initializeClients(); err != nil {
		return fmt.Errorf("failed to initialize clients: %w", err)
	}

	// Phase 2: Validate configuration
	if err := b.validateConfiguration(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Phase 3: Provision infrastructure
	if err := b.provisionInfrastructure(); err != nil {
		return fmt.Errorf("infrastructure provisioning failed: %w", err)
	}

	// Phase 4: Bootstrap Talos cluster
	if err := b.bootstrapTalosCluster(); err != nil {
		return fmt.Errorf("Talos cluster bootstrap failed: %w", err)
	}

	// Phase 5: Set up GitOps repositories
	if err := b.setupGitOpsRepositories(); err != nil {
		return fmt.Errorf("GitOps repository setup failed: %w", err)
	}

	// Phase 6: Generate Terraform templates
	if err := b.generateTerraformTemplates(); err != nil {
		return fmt.Errorf("Terraform template generation failed: %w", err)
	}

	// Phase 7: Final validation and cleanup
	if err := b.finalValidation(); err != nil {
		return fmt.Errorf("final validation failed: %w", err)
	}

	log.Info().Msg("Bootstrap process completed successfully!")
	b.displaySummary()

	return nil
}

// initializeClients initializes all required clients
func (b *Bootstrapper) initializeClients() error {
	log.Info().Msg("Initializing clients...")

	if b.options.DryRun {
		log.Info().Msg("DRY RUN: Would initialize clients")
		return nil
	}

	// Initialize state directory
	if err := os.MkdirAll(b.stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Initialize state manager
	b.stateManager = state.NewManager(b.stateDir)

	// Initialize Civo client
	var err error
	b.civoClient, err = civo.NewClient(b.config.CivoAPIKey, b.config.ClusterRegion)
	if err != nil {
		return fmt.Errorf("failed to initialize Civo client: %w", err)
	}

	// Initialize Git client
	b.gitClient, err = git.NewClient(b.config.GitProvider, b.config.GitPAT, b.config.GitUser, b.config.GitRepo)
	if err != nil {
		return fmt.Errorf("failed to initialize Git client: %w", err)
	}

	// Initialize Talos client
	b.talosClient = talos.NewClient(b.config.TalosVersion)

	return nil
}

// validateConfiguration validates the configuration and prerequisites
func (b *Bootstrapper) validateConfiguration() error {
	log.Info().Msg("Validating configuration...")

	if b.options.DryRun {
		log.Info().Msg("DRY RUN: Would validate configuration")
		return nil
	}

	// Validate prerequisites
	if err := b.validatePrerequisites(); err != nil {
		return fmt.Errorf("prerequisites validation failed: %w", err)
	}

	// Validate Civo connectivity
	if err := b.civoClient.ValidateConnection(); err != nil {
		return fmt.Errorf("Civo connection validation failed: %w", err)
	}

	// Validate Git connectivity
	if err := b.gitClient.ValidateConnection(); err != nil {
		return fmt.Errorf("Git connection validation failed: %w", err)
	}

	// Validate domain (placeholder for now)
	if err := b.validateDomain(); err != nil {
		return fmt.Errorf("domain validation failed: %w", err)
	}

	return nil
}

// provisionInfrastructure provisions the Civo infrastructure
func (b *Bootstrapper) provisionInfrastructure() error {
	log.Info().Msg("Provisioning Civo infrastructure...")

	if b.options.DryRun {
		log.Info().Msg("DRY RUN: Would provision Civo infrastructure")
		return nil
	}

	// Create network
	network, err := b.civoClient.CreateNetwork(b.config.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	// Create firewall
	firewall, err := b.civoClient.CreateFirewall(b.config.ClusterName, network.ID)
	if err != nil {
		return fmt.Errorf("failed to create firewall: %w", err)
	}

	// Create instances
	instances, err := b.civoClient.CreateInstances(civo.InstanceConfig{
		ClusterName:           b.config.ClusterName,
		NetworkID:             network.ID,
		FirewallID:            firewall.ID,
		ControlPlaneNodeCount: b.config.ControlplaneNodeCount,
		ControlPlaneNodeSize:  b.config.ControlplaneNodeSize,
		WorkerNodeCount:       b.config.WorkerNodeCount,
		WorkerNodeSize:        b.config.WorkerNodeSize,
	})
	if err != nil {
		return fmt.Errorf("failed to create instances: %w", err)
	}

	// Save infrastructure state
	infraState := &state.InfrastructureState{
		Network:   network,
		Firewall:  firewall,
		Instances: instances,
	}

	if err := b.stateManager.SaveInfrastructureState(infraState); err != nil {
		return fmt.Errorf("failed to save infrastructure state: %w", err)
	}

	log.Info().Msg("Civo infrastructure provisioned successfully")
	return nil
}

// bootstrapTalosCluster bootstraps the Talos cluster
func (b *Bootstrapper) bootstrapTalosCluster() error {
	log.Info().Msg("Bootstrapping Talos cluster...")

	if b.options.DryRun {
		log.Info().Msg("DRY RUN: Would bootstrap Talos cluster")
		return nil
	}

	// Load infrastructure state
	infraState, err := b.stateManager.LoadInfrastructureState()
	if err != nil {
		return fmt.Errorf("failed to load infrastructure state: %w", err)
	}

	// Generate Talos configuration
	talosConfig, err := b.talosClient.GenerateConfig(talos.ConfigOptions{
		ClusterName:       b.config.ClusterName,
		ClusterEndpoint:   infraState.GetControlPlaneEndpoint(),
		KubernetesVersion: b.config.KubernetesVersion,
		TalosVersion:      b.config.TalosVersion,
	})
	if err != nil {
		return fmt.Errorf("failed to generate Talos configuration: %w", err)
	}

	// Apply machine configurations
	if err := b.talosClient.ApplyMachineConfigurations(talosConfig, infraState.Instances); err != nil {
		return fmt.Errorf("failed to apply machine configurations: %w", err)
	}

	// Bootstrap the cluster
	if err := b.talosClient.Bootstrap(talosConfig, infraState.GetControlPlaneIPs()[0]); err != nil {
		return fmt.Errorf("failed to bootstrap cluster: %w", err)
	}

	// Wait for cluster to be ready
	if err := b.talosClient.WaitForClusterReady(talosConfig, infraState.GetControlPlaneIPs()[0], 10*time.Minute); err != nil {
		return fmt.Errorf("cluster failed to become ready: %w", err)
	}

	// Generate and save kubeconfig
	kubeconfig, err := b.talosClient.GenerateKubeconfig(talosConfig, infraState.GetControlPlaneIPs()[0])
	if err != nil {
		return fmt.Errorf("failed to generate kubeconfig: %w", err)
	}

	// Save Talos state
	talosState := &state.TalosState{
		Config:     talosConfig,
		Kubeconfig: kubeconfig,
	}

	if err := b.stateManager.SaveTalosState(talosState); err != nil {
		return fmt.Errorf("failed to save Talos state: %w", err)
	}

	log.Info().Msg("Talos cluster bootstrapped successfully")
	return nil
}

// setupGitOpsRepositories sets up GitOps repositories
func (b *Bootstrapper) setupGitOpsRepositories() error {
	log.Info().Msg("Setting up GitOps repositories...")

	if b.options.DryRun {
		log.Info().Msg("DRY RUN: Would set up GitOps repositories")
		return nil
	}

	// Create GitOps repository
	gitopsRepo, err := b.gitClient.CreateRepository("gitops", "GitOps repository for kubefirst management cluster")
	if err != nil {
		return fmt.Errorf("failed to create GitOps repository: %w", err)
	}

	// Create Metaphor repository
	metaphorRepo, err := b.gitClient.CreateRepository("metaphor", "Sample application for kubefirst")
	if err != nil {
		return fmt.Errorf("failed to create Metaphor repository: %w", err)
	}

	// Clone and populate repositories
	if err := b.gitClient.PopulateGitOpsRepository(gitopsRepo, b.config); err != nil {
		return fmt.Errorf("failed to populate GitOps repository: %w", err)
	}

	if err := b.gitClient.PopulateMetaphorRepository(metaphorRepo, b.config); err != nil {
		return fmt.Errorf("failed to populate Metaphor repository: %w", err)
	}

	// Save Git state
	gitState := &state.GitState{
		GitOpsRepo:   gitopsRepo,
		MetaphorRepo: metaphorRepo,
	}

	if err := b.stateManager.SaveGitState(gitState); err != nil {
		return fmt.Errorf("failed to save Git state: %w", err)
	}

	log.Info().Msg("GitOps repositories set up successfully")
	return nil
}

// generateTerraformTemplates generates Terraform templates
func (b *Bootstrapper) generateTerraformTemplates() error {
	log.Info().Msg("Generating Terraform templates...")

	if b.options.DryRun {
		log.Info().Msg("DRY RUN: Would generate Terraform templates")
		return nil
	}

	// Load all state
	infraState, err := b.stateManager.LoadInfrastructureState()
	if err != nil {
		return fmt.Errorf("failed to load infrastructure state: %w", err)
	}

	talosState, err := b.stateManager.LoadTalosState()
	if err != nil {
		return fmt.Errorf("failed to load Talos state: %w", err)
	}

	gitState, err := b.stateManager.LoadGitState()
	if err != nil {
		return fmt.Errorf("failed to load Git state: %w", err)
	}

	// Generate templates
	templates := b.generateTemplates(infraState, talosState, gitState)

	// Save templates
	terraformDir := filepath.Join(b.stateDir, "terraform")
	if err := os.MkdirAll(terraformDir, 0755); err != nil {
		return fmt.Errorf("failed to create terraform directory: %w", err)
	}

	for _, template := range templates {
		templatePath := filepath.Join(terraformDir, template.Path)
		if err := os.MkdirAll(filepath.Dir(templatePath), 0755); err != nil {
			return fmt.Errorf("failed to create template directory: %w", err)
		}

		if err := os.WriteFile(templatePath, []byte(template.Content), 0644); err != nil {
			return fmt.Errorf("failed to write template %s: %w", template.Name, err)
		}
	}

	log.Info().Msg("Terraform templates generated successfully")
	return nil
}

// finalValidation performs final validation
func (b *Bootstrapper) finalValidation() error {
	log.Info().Msg("Performing final validation...")

	if b.options.DryRun {
		log.Info().Msg("DRY RUN: Would perform final validation")
		return nil
	}

	// Validate cluster health
	talosState, err := b.stateManager.LoadTalosState()
	if err != nil {
		return fmt.Errorf("failed to load Talos state: %w", err)
	}

	if err := b.talosClient.ValidateClusterHealth(talosState.Config); err != nil {
		return fmt.Errorf("cluster health validation failed: %w", err)
	}

	log.Info().Msg("Final validation completed successfully")
	return nil
}

// displaySummary displays a summary of the bootstrap process
func (b *Bootstrapper) displaySummary() {
	log.Info().Msg("============================================")
	log.Info().Msg("🎉 Kubefirst Talos Bootstrap Complete!")
	log.Info().Msg("============================================")
	log.Info().Msgf("Cluster Name: %s", b.config.ClusterName)
	log.Info().Msgf("Region: %s", b.config.ClusterRegion)
	log.Info().Msgf("Domain: %s", b.config.DomainName)
	log.Info().Msgf("State Directory: %s", b.stateDir)
	log.Info().Msg("============================================")
	log.Info().Msg("Next Steps:")
	log.Info().Msg("1. Review the generated Terraform templates")
	log.Info().Msg("2. Run 'terraform init' in the terraform directory")
	log.Info().Msg("3. Run 'terraform plan' to review changes")
	log.Info().Msg("4. Run 'terraform apply' to deploy applications")
	log.Info().Msg("============================================")
}

// validatePrerequisites checks if all required tools are installed
func (b *Bootstrapper) validatePrerequisites() error {
	requiredTools := []string{
		"talosctl",
		"kubectl",
		"terraform",
		"git",
		"jq",
		"yq",
	}

	for _, tool := range requiredTools {
		if !utils.IsCommandAvailable(tool) {
			return fmt.Errorf("required tool not found: %s", tool)
		}
	}

	return nil
}

// validateDomain validates the domain configuration
func (b *Bootstrapper) validateDomain() error {
	// TODO: Implement domain validation logic
	// This could include DNS resolution tests, etc.
	return nil
}

// TerraformTemplate represents a Terraform template file
type TerraformTemplate struct {
	Name    string
	Path    string
	Content string
}

// generateTemplates generates all required Terraform templates
func (b *Bootstrapper) generateTemplates(infraState *state.InfrastructureState, talosState *state.TalosState, gitState *state.GitState) []TerraformTemplate {
	var templates []TerraformTemplate

	// Generate main.tf
	templates = append(templates, TerraformTemplate{
		Name:    "main.tf",
		Path:    "main.tf",
		Content: b.generateMainTF(infraState, talosState, gitState),
	})

	// Generate variables.tf
	templates = append(templates, TerraformTemplate{
		Name:    "variables.tf",
		Path:    "variables.tf",
		Content: b.generateVariablesTF(),
	})

	// Generate outputs.tf
	templates = append(templates, TerraformTemplate{
		Name:    "outputs.tf",
		Path:    "outputs.tf",
		Content: b.generateOutputsTF(infraState, talosState, gitState),
	})

	return templates
}

// generateMainTF generates the main Terraform configuration
func (b *Bootstrapper) generateMainTF(infraState *state.InfrastructureState, talosState *state.TalosState, gitState *state.GitState) string {
	return `# Kubefirst Talos Bootstrap - Main Configuration
# This file was generated by kubefirst-talos-bootstrap

terraform {
  required_providers {
    civo = {
      source  = "civo/civo"
      version = "~> 1.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.0"
    }
  }
}

provider "civo" {
  token  = var.civo_token
  region = var.civo_region
}

provider "kubernetes" {
  config_path = var.kubeconfig_path
}

provider "helm" {
  kubernetes {
    config_path = var.kubeconfig_path
  }
}

# Data sources for existing resources
data "civo_network" "cluster_network" {
  name = var.cluster_name
}

data "civo_firewall" "cluster_firewall" {
  name = var.cluster_name
}

# Example application deployment
resource "kubernetes_namespace" "applications" {
  metadata {
    name = "applications"
  }
}
`
}

// generateVariablesTF generates the variables.tf file
func (b *Bootstrapper) generateVariablesTF() string {
	return `# Kubefirst Talos Bootstrap - Variables
# This file was generated by kubefirst-talos-bootstrap

variable "civo_token" {
  description = "Civo API token"
  type        = string
  sensitive   = true
}

variable "civo_region" {
  description = "Civo region"
  type        = string
  default     = "LON1"
}

variable "cluster_name" {
  description = "Name of the cluster"
  type        = string
}

variable "domain_name" {
  description = "Domain name for the cluster"
  type        = string
}

variable "kubeconfig_path" {
  description = "Path to the kubeconfig file"
  type        = string
}

variable "git_provider" {
  description = "Git provider (github or gitlab)"
  type        = string
  default     = "github"
}

variable "git_user" {
  description = "Git username"
  type        = string
}

variable "git_token" {
  description = "Git personal access token"
  type        = string
  sensitive   = true
}
`
}

// generateOutputsTF generates the outputs.tf file
func (b *Bootstrapper) generateOutputsTF(infraState *state.InfrastructureState, talosState *state.TalosState, gitState *state.GitState) string {
	return `# Kubefirst Talos Bootstrap - Outputs
# This file was generated by kubefirst-talos-bootstrap

output "cluster_endpoint" {
  description = "Kubernetes cluster endpoint"
  value       = "https://` + infraState.GetControlPlaneEndpoint() + `:6443"
}

output "cluster_name" {
  description = "Name of the cluster"
  value       = var.cluster_name
}

output "network_id" {
  description = "Network ID"
  value       = data.civo_network.cluster_network.id
}

output "firewall_id" {
  description = "Firewall ID"
  value       = data.civo_firewall.cluster_firewall.id
}

output "gitops_repo_url" {
  description = "GitOps repository URL"
  value       = "` + gitState.GitOpsRepo["url"].(string) + `"
}

output "metaphor_repo_url" {
  description = "Metaphor repository URL"
  value       = "` + gitState.MetaphorRepo["url"].(string) + `"
}
`
}
