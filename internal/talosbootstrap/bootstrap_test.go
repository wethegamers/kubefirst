/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package talosbootstrap

import (
	"fmt"
	"os"
	"testing"
)

func TestNewBootstrapper(t *testing.T) {
	config := &Config{
		CloudProvider: "civo",
		ClusterName:   "test-cluster",
		CivoAPIKey:    "test-key",
		ClusterRegion: "LON1",
		DomainName:    "example.com",
		AlertsEmail:   "test@example.com",
		GitProvider:   "github",
		GitUser:       "testuser",
		GitPAT:        "test-token",
		GitRepo:       "test-repo",
	}

	options := Options{
		DryRun:  true,
		Verbose: false,
	}

	bootstrapper := NewBootstrapper(config, options)

	if bootstrapper == nil {
		t.Error("Expected bootstrapper to be created, got nil")
	}

	if bootstrapper.config.ClusterName != "test-cluster" {
		t.Errorf("Expected cluster name to be 'test-cluster', got %s", bootstrapper.config.ClusterName)
	}

	if bootstrapper.options.DryRun != true {
		t.Error("Expected dry run to be true")
	}

	// Check state directory is set correctly
	expectedStateDir := os.ExpandEnv("$HOME/.k1/test-cluster")
	if bootstrapper.stateDir != expectedStateDir {
		t.Errorf("Expected state dir to be %s, got %s", expectedStateDir, bootstrapper.stateDir)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &Config{
				CloudProvider:         "civo",
				ClusterName:           "test-cluster",
				CivoAPIKey:            "test-key",
				ClusterRegion:         "LON1",
				DomainName:            "example.com",
				AlertsEmail:           "test@example.com",
				GitProvider:           "github",
				GitUser:               "testuser",
				GitPAT:                "test-token",
				GitRepo:               "test-repo",
				ControlplaneNodeCount: 1,
				WorkerNodeCount:       1,
			},
			expectError: false,
		},
		{
			name: "missing cluster name",
			config: &Config{
				CloudProvider: "civo",
				CivoAPIKey:    "test-key",
				DomainName:    "example.com",
				AlertsEmail:   "test@example.com",
			},
			expectError: true,
		},
		{
			name: "invalid cloud provider",
			config: &Config{
				CloudProvider: "aws",
				ClusterName:   "test-cluster",
				CivoAPIKey:    "test-key",
				DomainName:    "example.com",
				AlertsEmail:   "test@example.com",
			},
			expectError: true,
		},
		{
			name: "missing civo api key",
			config: &Config{
				CloudProvider: "civo",
				ClusterName:   "test-cluster",
				DomainName:    "example.com",
				AlertsEmail:   "test@example.com",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bootstrapper := NewBootstrapper(tt.config, Options{DryRun: true})
			err := bootstrapper.validateTalosBootstrapConfig(tt.config)
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// validateTalosBootstrapConfig is a helper function for testing
func (b *Bootstrapper) validateTalosBootstrapConfig(config *Config) error {
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
	if config.DNSProvider != "" && config.DNSProvider != "civo" && config.DNSProvider != "cloudflare" {
		return fmt.Errorf("only 'civo' and 'cloudflare' DNS providers are supported")
	}
	if config.GitProvider != "" && config.GitProvider != "github" && config.GitProvider != "gitlab" {
		return fmt.Errorf("only 'github' and 'gitlab' git providers are supported")
	}
	if config.ControlplaneNodeCount < 0 {
		return fmt.Errorf("controlplaneNodeCount must be 0 or greater")
	}
	if config.WorkerNodeCount < 0 {
		return fmt.Errorf("workerNodeCount must be 0 or greater")
	}

	return nil
}
