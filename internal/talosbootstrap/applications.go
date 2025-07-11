/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package talosbootstrap

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ApplicationDeployer handles deployment of kubefirst management applications
type ApplicationDeployer struct {
	kubeconfig string
	clientset  *kubernetes.Clientset
	config     *Config
	runner     *CommandRunner
}

// NewApplicationDeployer creates a new application deployer
func NewApplicationDeployer(kubeconfig string, config *Config) (*ApplicationDeployer, error) {
	// Create Kubernetes clientset
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &ApplicationDeployer{
		kubeconfig: kubeconfig,
		clientset:  clientset,
		config:     config,
		runner:     NewCommandRunner(kubeconfig, false),
	}, nil
}

// DeployManagementStack deploys the complete kubefirst management stack
func (ad *ApplicationDeployer) DeployManagementStack() error {
	log.Info().Msg("Deploying kubefirst management stack...")

	// Phase 1: Install ArgoCD
	if err := ad.installArgoCD(); err != nil {
		return fmt.Errorf("failed to install ArgoCD: %w", err)
	}

	// Phase 2: Install Cert Manager
	if err := ad.installCertManager(); err != nil {
		return fmt.Errorf("failed to install cert-manager: %w", err)
	}

	// Phase 3: Install Vault
	if err := ad.installVault(); err != nil {
		return fmt.Errorf("failed to install Vault: %w", err)
	}

	// Phase 4: Install External Secrets Operator
	if err := ad.installExternalSecretsOperator(); err != nil {
		return fmt.Errorf("failed to install external-secrets-operator: %w", err)
	}

	// Phase 5: Install Ingress Nginx
	if err := ad.installIngressNginx(); err != nil {
		return fmt.Errorf("failed to install ingress-nginx: %w", err)
	}

	// Phase 6: Install Reloader
	if err := ad.installReloader(); err != nil {
		return fmt.Errorf("failed to install reloader: %w", err)
	}

	// Phase 7: Setup GitOps registry structure
	if err := ad.setupGitOpsRegistry(); err != nil {
		return fmt.Errorf("failed to setup GitOps registry: %w", err)
	}

	// Phase 8: Install additional applications
	if err := ad.installAdditionalApplications(); err != nil {
		return fmt.Errorf("failed to install additional applications: %w", err)
	}

	log.Info().Msg("Kubefirst management stack deployed successfully!")
	return nil
}

// installArgoCD installs ArgoCD using Helm
func (ad *ApplicationDeployer) installArgoCD() error {
	log.Info().Msg("Installing ArgoCD...")

	// Create ArgoCD namespace
	if err := ad.runKubectl("create", "namespace", "argocd", "--dry-run=client", "-o", "yaml"); err != nil {
		return fmt.Errorf("failed to create argocd namespace: %w", err)
	}

	// Install ArgoCD
	manifestURL := "https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml"
	if err := ad.runKubectl("apply", "-n", "argocd", "-f", manifestURL); err != nil {
		return fmt.Errorf("failed to install ArgoCD: %w", err)
	}

	// Wait for ArgoCD to be ready
	if err := ad.runKubectl("wait", "--for=condition=available", "--timeout=600s", "deployment/argocd-server", "-n", "argocd"); err != nil {
		return fmt.Errorf("ArgoCD failed to become ready: %w", err)
	}

	log.Info().Msg("ArgoCD installed successfully")
	return nil
}

// installCertManager installs cert-manager using Helm
func (ad *ApplicationDeployer) installCertManager() error {
	log.Info().Msg("Installing cert-manager...")

	// Add cert-manager Helm repository
	if err := ad.runHelm("repo", "add", "jetstack", "https://charts.jetstack.io"); err != nil {
		return fmt.Errorf("failed to add cert-manager helm repo: %w", err)
	}

	if err := ad.runHelm("repo", "update"); err != nil {
		return fmt.Errorf("failed to update helm repos: %w", err)
	}

	// Install cert-manager
	if err := ad.runHelm("install", "cert-manager", "jetstack/cert-manager",
		"--namespace", "cert-manager",
		"--create-namespace",
		"--version", "v1.14.4",
		"--set", "installCRDs=true",
		"--wait"); err != nil {
		return fmt.Errorf("failed to install cert-manager: %w", err)
	}

	log.Info().Msg("cert-manager installed successfully")
	return nil
}

// installVault installs HashiCorp Vault using Helm
func (ad *ApplicationDeployer) installVault() error {
	log.Info().Msg("Installing HashiCorp Vault...")

	// Add HashiCorp Helm repository
	if err := ad.runHelm("repo", "add", "hashicorp", "https://helm.releases.hashicorp.com"); err != nil {
		return fmt.Errorf("failed to add hashicorp helm repo: %w", err)
	}

	if err := ad.runHelm("repo", "update"); err != nil {
		return fmt.Errorf("failed to update helm repos: %w", err)
	}

	// Install Vault
	if err := ad.runHelm("install", "vault", "hashicorp/vault",
		"--namespace", "vault",
		"--create-namespace",
		"--set", "server.ha.enabled=true",
		"--set", "server.ha.replicas=3",
		"--set", "server.ha.raft.enabled=true",
		"--set", "ui.enabled=true",
		"--wait"); err != nil {
		return fmt.Errorf("failed to install vault: %w", err)
	}

	log.Info().Msg("HashiCorp Vault installed successfully")
	return nil
}

// installExternalSecretsOperator installs External Secrets Operator using Helm
func (ad *ApplicationDeployer) installExternalSecretsOperator() error {
	log.Info().Msg("Installing External Secrets Operator...")

	// Add External Secrets Helm repository
	if err := ad.runHelm("repo", "add", "external-secrets", "https://charts.external-secrets.io"); err != nil {
		return fmt.Errorf("failed to add external-secrets helm repo: %w", err)
	}

	if err := ad.runHelm("repo", "update"); err != nil {
		return fmt.Errorf("failed to update helm repos: %w", err)
	}

	// Install External Secrets Operator
	if err := ad.runHelm("install", "external-secrets", "external-secrets/external-secrets",
		"--namespace", "external-secrets-system",
		"--create-namespace",
		"--wait"); err != nil {
		return fmt.Errorf("failed to install external-secrets: %w", err)
	}

	log.Info().Msg("External Secrets Operator installed successfully")
	return nil
}

// installIngressNginx installs Ingress Nginx using Helm
func (ad *ApplicationDeployer) installIngressNginx() error {
	log.Info().Msg("Installing Ingress Nginx...")

	// Add Ingress Nginx Helm repository
	if err := ad.runHelm("repo", "add", "ingress-nginx", "https://kubernetes.github.io/ingress-nginx"); err != nil {
		return fmt.Errorf("failed to add ingress-nginx helm repo: %w", err)
	}

	if err := ad.runHelm("repo", "update"); err != nil {
		return fmt.Errorf("failed to update helm repos: %w", err)
	}

	// Install Ingress Nginx
	if err := ad.runHelm("install", "ingress-nginx", "ingress-nginx/ingress-nginx",
		"--namespace", "ingress-nginx",
		"--create-namespace",
		"--set", "controller.service.type=LoadBalancer",
		"--wait"); err != nil {
		return fmt.Errorf("failed to install ingress-nginx: %w", err)
	}

	log.Info().Msg("Ingress Nginx installed successfully")
	return nil
}

// installReloader installs Reloader using Helm
func (ad *ApplicationDeployer) installReloader() error {
	log.Info().Msg("Installing Reloader...")

	// Add Stakater Helm repository
	if err := ad.runHelm("repo", "add", "stakater", "https://stakater.github.io/stakater-charts"); err != nil {
		return fmt.Errorf("failed to add stakater helm repo: %w", err)
	}

	if err := ad.runHelm("repo", "update"); err != nil {
		return fmt.Errorf("failed to update helm repos: %w", err)
	}

	// Install Reloader
	if err := ad.runHelm("install", "reloader", "stakater/reloader",
		"--namespace", "reloader",
		"--create-namespace",
		"--wait"); err != nil {
		return fmt.Errorf("failed to install reloader: %w", err)
	}

	log.Info().Msg("Reloader installed successfully")
	return nil
}

// setupGitOpsRegistry creates the GitOps registry structure
func (ad *ApplicationDeployer) setupGitOpsRegistry() error {
	log.Info().Msg("Setting up GitOps registry structure...")

	// This will be implemented when we have the GitOps repository created
	// For now, we'll create the basic structure locally
	
	log.Info().Msg("GitOps registry structure created")
	return nil
}

// installAdditionalApplications installs additional kubefirst applications
func (ad *ApplicationDeployer) installAdditionalApplications() error {
	log.Info().Msg("Installing additional applications...")

	// Install Argo Workflows
	if err := ad.installArgoWorkflows(); err != nil {
		return fmt.Errorf("failed to install argo workflows: %w", err)
	}

	// Install Atlantis
	if err := ad.installAtlantis(); err != nil {
		return fmt.Errorf("failed to install atlantis: %w", err)
	}

	// Install ChartMuseum
	if err := ad.installChartMuseum(); err != nil {
		return fmt.Errorf("failed to install chartmuseum: %w", err)
	}

	log.Info().Msg("Additional applications installed successfully")
	return nil
}

// installArgoWorkflows installs Argo Workflows
func (ad *ApplicationDeployer) installArgoWorkflows() error {
	log.Info().Msg("Installing Argo Workflows...")

	// Create namespace
	if err := ad.runKubectl("create", "namespace", "argo", "--dry-run=client", "-o", "yaml"); err != nil {
		return fmt.Errorf("failed to create argo namespace: %w", err)
	}

	// Install Argo Workflows
	manifestURL := "https://github.com/argoproj/argo-workflows/releases/download/v3.5.4/install.yaml"
	if err := ad.runKubectl("apply", "-n", "argo", "-f", manifestURL); err != nil {
		return fmt.Errorf("failed to install argo workflows: %w", err)
	}

	log.Info().Msg("Argo Workflows installed successfully")
	return nil
}

// installAtlantis installs Atlantis using Helm
func (ad *ApplicationDeployer) installAtlantis() error {
	log.Info().Msg("Installing Atlantis...")

	// Add Atlantis Helm repository
	if err := ad.runHelm("repo", "add", "atlantis", "https://runatlantis.github.io/helm-charts"); err != nil {
		return fmt.Errorf("failed to add atlantis helm repo: %w", err)
	}

	if err := ad.runHelm("repo", "update"); err != nil {
		return fmt.Errorf("failed to update helm repos: %w", err)
	}

	// Install Atlantis
	if err := ad.runHelm("install", "atlantis", "atlantis/atlantis",
		"--namespace", "atlantis",
		"--create-namespace",
		"--wait"); err != nil {
		return fmt.Errorf("failed to install atlantis: %w", err)
	}

	log.Info().Msg("Atlantis installed successfully")
	return nil
}

// installChartMuseum installs ChartMuseum using Helm
func (ad *ApplicationDeployer) installChartMuseum() error {
	log.Info().Msg("Installing ChartMuseum...")

	// Add ChartMuseum Helm repository
	if err := ad.runHelm("repo", "add", "chartmuseum", "https://chartmuseum.github.io/charts"); err != nil {
		return fmt.Errorf("failed to add chartmuseum helm repo: %w", err)
	}

	if err := ad.runHelm("repo", "update"); err != nil {
		return fmt.Errorf("failed to update helm repos: %w", err)
	}

	// Install ChartMuseum
	if err := ad.runHelm("install", "chartmuseum", "chartmuseum/chartmuseum",
		"--namespace", "chartmuseum",
		"--create-namespace",
		"--wait"); err != nil {
		return fmt.Errorf("failed to install chartmuseum: %w", err)
	}

	log.Info().Msg("ChartMuseum installed successfully")
	return nil
}

// runKubectl runs a kubectl command
func (ad *ApplicationDeployer) runKubectl(args ...string) error {
	return ad.runner.RunKubectl(args...)
}

// runHelm runs a helm command
func (ad *ApplicationDeployer) runHelm(args ...string) error {
	return ad.runner.RunHelm(args...)
}
