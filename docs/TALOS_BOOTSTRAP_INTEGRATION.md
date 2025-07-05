# Kubefirst Talos Bootstrap Integration

This document describes the integration of the kubefirst-talos bootstrap functionality into the main kubefirst CLI as a new subcommand under the Civo provider.

## Overview

The `kubefirst civo talos-bootstrap` command provides a complete bootstrap solution for creating a kubefirst management cluster on Civo using Talos OS. This integration brings the functionality from the standalone kubefirst-talos project directly into the main kubefirst CLI.

## Usage

### Basic Command

```bash
kubefirst civo talos-bootstrap --config-path ./config.yaml
```

### Flags

- `--config-path`: Path to the configuration file (required)
- `--dry-run`: Perform a dry run without making changes
- `--verbose`: Enable verbose logging

### Configuration File

The command requires a YAML configuration file with the following structure:

```yaml
cloudProvider: civo
civoAPIKey: "your-civo-api-key"
clusterRegion: LON1

dnsProvider: civo  # or cloudflare
domainName: example.com
subdomain: k8s
cloudflareAPIToken: ""  # required if using cloudflare
cloudflareZoneId: ""    # required if using cloudflare

gitProvider: github  # or gitlab
gitUser: your-username
gitPAT: "your-personal-access-token"
gitRepo: your-repo

clusterName: my-cluster
alertsEmail: alerts@example.com
talosVersion: v1.7.0
kubernetesVersion: v1.30.0
controlplaneNodeCount: 3
controlplaneNodeSize: g4s.kube.medium
workerNodeCount: 3
workerNodeSize: g4s.kube.medium

# Optional feature flags
enableCNI: true
enableCSI: true
enableIngress: true
enableArgoCD: true
enableVault: true
enableTelemetry: false
enableBackups: true
enableMonitoring: true
enableCertManager: true
enableExternalDNS: true
enableExternalSecrets: true
enableKubefirst: true
enableMetaphor: true
```

## Bootstrap Process

The command performs the following steps:

1. **Initialize Clients**: Sets up Civo, Git, and Talos clients
2. **Validate Configuration**: Checks prerequisites and connectivity
3. **Provision Infrastructure**: Creates Civo network, firewall, and instances
4. **Bootstrap Talos Cluster**: Generates and applies Talos configurations
5. **Setup GitOps Repositories**: Creates and populates Git repositories
6. **Generate Terraform Templates**: Creates templates for subsequent deployment
7. **Final Validation**: Verifies cluster health and readiness

## Prerequisites

Before running the bootstrap command, ensure you have the following tools installed:

- `talosctl` - Talos CLI tool
- `kubectl` - Kubernetes CLI
- `terraform` - Infrastructure as Code tool
- `git` - Version control system
- `jq` - JSON processor
- `yq` - YAML processor

## Directory Structure

The integration follows this structure:

```
internal/talosbootstrap/
├── bootstrap.go      # Main bootstrap logic
├── civo/
│   └── client.go     # Civo API client
├── git/
│   └── client.go     # Git provider client
├── state/
│   └── manager.go    # State management
├── talos/
│   └── client.go     # Talos operations client
└── utils/
    └── utils.go      # Utility functions

cmd/civo/
├── command.go        # Modified to include talos-bootstrap
└── talos_bootstrap.go # New command implementation
```

## State Management

The bootstrap process saves state information in `~/.k1/<cluster-name>/`:

- Infrastructure state (network, firewall, instances)
- Talos configuration and kubeconfig
- Git repository information
- Generated Terraform templates

## Dry Run Mode

Use the `--dry-run` flag to validate configuration and preview changes without actually creating resources:

```bash
kubefirst civo talos-bootstrap --config-path ./config.yaml --dry-run
```

## Example Output

```
🧪 DRY RUN MODE - No resources will be created

Would bootstrap Talos cluster with configuration:
  Cluster Name: my-cluster
  Region: LON1
  Domain: example.com
  Control Plane Nodes: 3 (g4s.kube.medium)
  Worker Nodes: 3 (g4s.kube.medium)
  Talos Version: v1.7.0
  Kubernetes Version: v1.30.0
  DNS Provider: civo
  Git Provider: github

✅ Dry run completed successfully!
```

## Next Steps

After successful bootstrap:

1. Review the generated Terraform templates in `~/.k1/<cluster-name>/terraform/`
2. Run `terraform init` in the terraform directory
3. Run `terraform plan` to review changes
4. Run `terraform apply` to deploy applications

## Error Handling

The command provides detailed error messages for common issues:

- Missing required tools
- Invalid configuration
- API connectivity problems
- Resource creation failures

All operations are designed to be idempotent where possible.

## Integration Notes

This integration maintains compatibility with the existing kubefirst CLI patterns and follows the project's architectural conventions. The implementation can be extended to support additional cloud providers and configuration options as needed.
