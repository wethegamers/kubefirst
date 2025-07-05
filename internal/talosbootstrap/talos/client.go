/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package talos

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/civo/civogo"
	"github.com/rs/zerolog/log"
)

// Client handles Talos operations
type Client struct {
	version string
}

// ConfigOptions represents options for generating Talos configuration
type ConfigOptions struct {
	ClusterName       string
	ClusterEndpoint   string
	KubernetesVersion string
	TalosVersion      string
}

// NewClient creates a new Talos client
func NewClient(version string) *Client {
	return &Client{
		version: version,
	}
}

// GenerateConfig generates Talos cluster configuration
func (c *Client) GenerateConfig(opts ConfigOptions) (map[string]interface{}, error) {
	log.Info().Msg("Generating Talos configuration...")

	// Create temporary directory for config generation
	tempDir, err := os.MkdirTemp("", "talos-config-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate configuration using talosctl
	cmd := exec.Command("talosctl", "gen", "config", opts.ClusterName, 
		fmt.Sprintf("https://%s:6443", opts.ClusterEndpoint),
		"--output-dir", tempDir,
		"--kubernetes-version", opts.KubernetesVersion,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Talos config: %w\nOutput: %s", err, string(output))
	}

	// Read the generated configuration files
	config := make(map[string]interface{})
	
	// Read controlplane.yaml
	controlplaneConfig, err := os.ReadFile(filepath.Join(tempDir, "controlplane.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read controlplane config: %w", err)
	}
	config["controlplane"] = string(controlplaneConfig)

	// Read worker.yaml
	workerConfig, err := os.ReadFile(filepath.Join(tempDir, "worker.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read worker config: %w", err)
	}
	config["worker"] = string(workerConfig)

	// Read talosconfig
	talosConfig, err := os.ReadFile(filepath.Join(tempDir, "talosconfig"))
	if err != nil {
		return nil, fmt.Errorf("failed to read talos config: %w", err)
	}
	config["talosconfig"] = string(talosConfig)

	log.Info().Msg("Talos configuration generated successfully")
	return config, nil
}

// ApplyMachineConfigurations applies machine configurations to instances
func (c *Client) ApplyMachineConfigurations(config map[string]interface{}, instances []*civogo.Instance) error {
	log.Info().Msg("Applying machine configurations...")

	// Create temporary directory for configurations
	tempDir, err := os.MkdirTemp("", "talos-apply-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write configurations to temporary files
	controlplanePath := filepath.Join(tempDir, "controlplane.yaml")
	workerPath := filepath.Join(tempDir, "worker.yaml")
	talosconfigPath := filepath.Join(tempDir, "talosconfig")

	if err := os.WriteFile(controlplanePath, []byte(config["controlplane"].(string)), 0644); err != nil {
		return fmt.Errorf("failed to write controlplane config: %w", err)
	}

	if err := os.WriteFile(workerPath, []byte(config["worker"].(string)), 0644); err != nil {
		return fmt.Errorf("failed to write worker config: %w", err)
	}

	if err := os.WriteFile(talosconfigPath, []byte(config["talosconfig"].(string)), 0644); err != nil {
		return fmt.Errorf("failed to write talos config: %w", err)
	}

	// Apply configurations to instances
	for _, instance := range instances {
		isControlPlane := false
		for _, tag := range instance.Tags {
			if tag == "control-plane" {
				isControlPlane = true
				break
			}
		}

		configPath := workerPath
		if isControlPlane {
			configPath = controlplanePath
		}

		log.Info().Msgf("Applying configuration to instance %s (%s)", instance.Hostname, instance.PublicIP)

		cmd := exec.Command("talosctl", "apply-config", 
			"--insecure",
			"--nodes", instance.PublicIP,
			"--file", configPath,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to apply config to instance %s: %w\nOutput: %s", instance.Hostname, err, string(output))
		}

		log.Info().Msgf("Configuration applied to instance %s", instance.Hostname)
	}

	log.Info().Msg("Machine configurations applied successfully")
	return nil
}

// Bootstrap bootstraps the Talos cluster
func (c *Client) Bootstrap(config map[string]interface{}, controlPlaneIP string) error {
	log.Info().Msgf("Bootstrapping Talos cluster on %s", controlPlaneIP)

	// Create temporary directory for talosconfig
	tempDir, err := os.MkdirTemp("", "talos-bootstrap-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write talosconfig
	talosconfigPath := filepath.Join(tempDir, "talosconfig")
	if err := os.WriteFile(talosconfigPath, []byte(config["talosconfig"].(string)), 0644); err != nil {
		return fmt.Errorf("failed to write talos config: %w", err)
	}

	// Bootstrap the cluster
	cmd := exec.Command("talosctl", "bootstrap", 
		"--talosconfig", talosconfigPath,
		"--nodes", controlPlaneIP,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to bootstrap cluster: %w\nOutput: %s", err, string(output))
	}

	log.Info().Msg("Talos cluster bootstrapped successfully")
	return nil
}

// WaitForClusterReady waits for the cluster to be ready
func (c *Client) WaitForClusterReady(config map[string]interface{}, controlPlaneIP string, timeout time.Duration) error {
	log.Info().Msgf("Waiting for cluster to be ready (timeout: %v)", timeout)

	// Create temporary directory for talosconfig
	tempDir, err := os.MkdirTemp("", "talos-wait-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write talosconfig
	talosconfigPath := filepath.Join(tempDir, "talosconfig")
	if err := os.WriteFile(talosconfigPath, []byte(config["talosconfig"].(string)), 0644); err != nil {
		return fmt.Errorf("failed to write talos config: %w", err)
	}

	start := time.Now()
	for {
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for cluster to be ready")
		}

		// Check cluster health
		cmd := exec.Command("talosctl", "health", 
			"--talosconfig", talosconfigPath,
			"--nodes", controlPlaneIP,
		)

		output, err := cmd.CombinedOutput()
		if err == nil && strings.Contains(string(output), "OK") {
			log.Info().Msg("Cluster is ready")
			return nil
		}

		log.Debug().Msgf("Cluster not ready yet, retrying... (error: %v)", err)
		time.Sleep(30 * time.Second)
	}
}

// GenerateKubeconfig generates a kubeconfig for the cluster
func (c *Client) GenerateKubeconfig(config map[string]interface{}, controlPlaneIP string) (string, error) {
	log.Info().Msg("Generating kubeconfig...")

	// Create temporary directory for talosconfig
	tempDir, err := os.MkdirTemp("", "talos-kubeconfig-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write talosconfig
	talosconfigPath := filepath.Join(tempDir, "talosconfig")
	if err := os.WriteFile(talosconfigPath, []byte(config["talosconfig"].(string)), 0644); err != nil {
		return "", fmt.Errorf("failed to write talos config: %w", err)
	}

	// Generate kubeconfig
	kubeconfigPath := filepath.Join(tempDir, "kubeconfig")
	cmd := exec.Command("talosctl", "kubeconfig", 
		"--talosconfig", talosconfigPath,
		"--nodes", controlPlaneIP,
		kubeconfigPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to generate kubeconfig: %w\nOutput: %s", err, string(output))
	}

	// Read the generated kubeconfig
	kubeconfig, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	log.Info().Msg("Kubeconfig generated successfully")
	return string(kubeconfig), nil
}

// ValidateClusterHealth validates the health of the cluster
func (c *Client) ValidateClusterHealth(config map[string]interface{}) error {
	log.Info().Msg("Validating cluster health...")

	// Create temporary directory for talosconfig
	tempDir, err := os.MkdirTemp("", "talos-health-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write talosconfig
	talosconfigPath := filepath.Join(tempDir, "talosconfig")
	if err := os.WriteFile(talosconfigPath, []byte(config["talosconfig"].(string)), 0644); err != nil {
		return fmt.Errorf("failed to write talos config: %w", err)
	}

	// Validate cluster health
	cmd := exec.Command("talosctl", "health", 
		"--talosconfig", talosconfigPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster health validation failed: %w\nOutput: %s", err, string(output))
	}

	log.Info().Msg("Cluster health validation passed")
	return nil
}
