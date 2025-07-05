/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/civo/civogo"
)

// Manager handles state persistence
type Manager struct {
	stateDir string
}

// InfrastructureState represents the infrastructure state
type InfrastructureState struct {
	Network   *civogo.Network    `json:"network"`
	Firewall  *civogo.Firewall   `json:"firewall"`
	Instances []*civogo.Instance `json:"instances"`
}

// TalosState represents the Talos cluster state
type TalosState struct {
	Config     map[string]interface{} `json:"config"`
	Kubeconfig string                 `json:"kubeconfig"`
}

// GitState represents the Git repositories state
type GitState struct {
	GitOpsRepo   map[string]interface{} `json:"gitops_repo"`
	MetaphorRepo map[string]interface{} `json:"metaphor_repo"`
}

// NewManager creates a new state manager
func NewManager(stateDir string) *Manager {
	return &Manager{
		stateDir: stateDir,
	}
}

// SaveInfrastructureState saves infrastructure state to disk
func (m *Manager) SaveInfrastructureState(state *InfrastructureState) error {
	return m.saveState("infrastructure.json", state)
}

// LoadInfrastructureState loads infrastructure state from disk
func (m *Manager) LoadInfrastructureState() (*InfrastructureState, error) {
	var state InfrastructureState
	err := m.loadState("infrastructure.json", &state)
	return &state, err
}

// SaveTalosState saves Talos state to disk
func (m *Manager) SaveTalosState(state *TalosState) error {
	return m.saveState("talos.json", state)
}

// LoadTalosState loads Talos state from disk
func (m *Manager) LoadTalosState() (*TalosState, error) {
	var state TalosState
	err := m.loadState("talos.json", &state)
	return &state, err
}

// SaveGitState saves Git state to disk
func (m *Manager) SaveGitState(state *GitState) error {
	return m.saveState("git.json", state)
}

// LoadGitState loads Git state from disk
func (m *Manager) LoadGitState() (*GitState, error) {
	var state GitState
	err := m.loadState("git.json", &state)
	return &state, err
}

// saveState saves state to a JSON file
func (m *Manager) saveState(filename string, state interface{}) error {
	filePath := filepath.Join(m.stateDir, filename)
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// loadState loads state from a JSON file
func (m *Manager) loadState(filename string, state interface{}) error {
	filePath := filepath.Join(m.stateDir, filename)
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return nil
}

// GetControlPlaneEndpoint returns the control plane endpoint
func (s *InfrastructureState) GetControlPlaneEndpoint() string {
	if len(s.Instances) == 0 {
		return ""
	}
	
	// Find the first control plane instance
	for _, instance := range s.Instances {
		for _, tag := range instance.Tags {
			if tag == "control-plane" {
				return instance.PublicIP
			}
		}
	}
	
	return ""
}

// GetControlPlaneIPs returns all control plane IPs
func (s *InfrastructureState) GetControlPlaneIPs() []string {
	var ips []string
	
	for _, instance := range s.Instances {
		for _, tag := range instance.Tags {
			if tag == "control-plane" {
				ips = append(ips, instance.PublicIP)
				break
			}
		}
	}
	
	return ips
}
