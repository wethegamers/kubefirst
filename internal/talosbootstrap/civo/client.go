/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the	result, err := c.client.CreateNetwork(config)

	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	// Get the created network by ID
	network, err := c.client.GetNetwork(result.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created network: %w", err)
	}

	log.Info().Msgf("Network %s created successfully", name)
	return network, nilr more details.
*/
package civo

import (
	"fmt"
	"time"

	"github.com/civo/civogo"
	"github.com/rs/zerolog/log"
)

// Client represents a Civo API client
type Client struct {
	client *civogo.Client
	region string
}

// InstanceConfig represents the configuration for creating instances
type InstanceConfig struct {
	ClusterName           string
	NetworkID             string
	FirewallID            string
	ControlPlaneNodeCount int
	ControlPlaneNodeSize  string
	WorkerNodeCount       int
	WorkerNodeSize        string
}

// NewClient creates a new Civo client
func NewClient(apiKey, region string) (*Client, error) {
	client, err := civogo.NewClient(apiKey, region)
	if err != nil {
		return nil, fmt.Errorf("failed to create Civo client: %w", err)
	}

	return &Client{
		client: client,
		region: region,
	}, nil
}

// ValidateConnection validates the connection to Civo API
func (c *Client) ValidateConnection() error {
	log.Debug().Msg("Validating Civo connection...")

	// Try to list networks as a connectivity test
	_, err := c.client.ListNetworks()
	if err != nil {
		return fmt.Errorf("failed to connect to Civo API: %w", err)
	}

	log.Debug().Msg("Civo connection validated successfully")
	return nil
}

// CreateNetwork creates a new network
func (c *Client) CreateNetwork(name string) (*civogo.Network, error) {
	log.Info().Msgf("Creating network: %s", name)

	// Check if network already exists
	networks, err := c.client.ListNetworks()
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	for _, network := range networks {
		if network.Name == name {
			log.Info().Msgf("Network %s already exists", name)
			return &network, nil
		}
	}

	// Create new network
	config := civogo.NetworkConfig{
		Label: name,
	}

	result, err := c.client.CreateNetwork(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	// Get the created network by ID
	network, err := c.client.GetNetwork(result.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created network: %w", err)
	}

	log.Info().Msgf("Network %s created successfully", name)
	return network, nil
}

// CreateFirewall creates a new firewall
func (c *Client) CreateFirewall(name, networkID string) (*civogo.Firewall, error) {
	log.Info().Msgf("Creating firewall: %s", name)

	// Check if firewall already exists
	firewalls, err := c.client.ListFirewalls()
	if err != nil {
		return nil, fmt.Errorf("failed to list firewalls: %w", err)
	}

	for _, firewall := range firewalls {
		if firewall.Name == name {
			log.Info().Msgf("Firewall %s already exists", name)
			return &firewall, nil
		}
	}

	// Create new firewall with Talos-specific rules
	config := &civogo.FirewallConfig{
		Name:      name,
		NetworkID: networkID,
		Rules: []civogo.FirewallRule{
			{
				Protocol:  "tcp",
				StartPort: "6443",
				EndPort:   "6443",
				Cidr:      []string{"0.0.0.0/0"},
				Direction: "ingress",
				Label:     "Kubernetes API",
			},
			{
				Protocol:  "tcp",
				StartPort: "50000",
				EndPort:   "50000",
				Cidr:      []string{"0.0.0.0/0"},
				Direction: "ingress",
				Label:     "Talos API",
			},
			{
				Protocol:  "tcp",
				StartPort: "50001",
				EndPort:   "50001",
				Cidr:      []string{"0.0.0.0/0"},
				Direction: "ingress",
				Label:     "Talos Trustd",
			},
		},
	}

	result, err := c.client.NewFirewall(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create firewall: %w", err)
	}

	// Convert result to firewall
	firewall := &civogo.Firewall{
		ID:   result.ID,
		Name: name,
	}

	log.Info().Msgf("Firewall %s created successfully", name)
	return firewall, nil
}

// CreateInstances creates the required instances for the cluster
func (c *Client) CreateInstances(config InstanceConfig) ([]*civogo.Instance, error) {
	log.Info().Msg("Creating cluster instances...")

	var instances []*civogo.Instance

	// Create control plane instances
	for i := 0; i < config.ControlPlaneNodeCount; i++ {
		instanceName := fmt.Sprintf("%s-control-plane-%d", config.ClusterName, i+1)
		instance, err := c.createInstance(instanceName, config.ControlPlaneNodeSize, config.NetworkID, config.FirewallID, true)
		if err != nil {
			return nil, fmt.Errorf("failed to create control plane instance %s: %w", instanceName, err)
		}
		instances = append(instances, instance)
	}

	// Create worker instances
	for i := 0; i < config.WorkerNodeCount; i++ {
		instanceName := fmt.Sprintf("%s-worker-%d", config.ClusterName, i+1)
		instance, err := c.createInstance(instanceName, config.WorkerNodeSize, config.NetworkID, config.FirewallID, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create worker instance %s: %w", instanceName, err)
		}
		instances = append(instances, instance)
	}

	// Wait for all instances to be ready
	if err := c.waitForInstancesReady(instances); err != nil {
		return nil, fmt.Errorf("instances failed to become ready: %w", err)
	}

	log.Info().Msgf("Created %d instances successfully", len(instances))
	return instances, nil
}

// createInstance creates a single instance
func (c *Client) createInstance(name, size, networkID, firewallID string, isControlPlane bool) (*civogo.Instance, error) {
	log.Info().Msgf("Creating instance: %s", name)

	// Check if instance already exists
	instances, err := c.client.ListInstances(1, 200)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	for _, instance := range instances.Items {
		if instance.Hostname == name {
			log.Info().Msgf("Instance %s already exists", name)
			return &instance, nil
		}
	}

	// Use Ubuntu template (we'll install Talos ourselves)
	templateID := "ubuntu-22.04-server"

	config := &civogo.InstanceConfig{
		Hostname:   name,
		Size:       size,
		TemplateID: templateID,
		NetworkID:  networkID,
		FirewallID: firewallID,
		Tags:       []string{"talos", "kubefirst"},
	}

	if isControlPlane {
		config.Tags = append(config.Tags, "control-plane")
	} else {
		config.Tags = append(config.Tags, "worker")
	}

	instance, err := c.client.CreateInstance(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	log.Info().Msgf("Instance %s created successfully", name)
	return instance, nil
}

// waitForInstancesReady waits for all instances to be ready
func (c *Client) waitForInstancesReady(instances []*civogo.Instance) error {
	log.Info().Msg("Waiting for instances to be ready...")

	timeout := 10 * time.Minute
	start := time.Now()

	for {
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for instances to be ready")
		}

		allReady := true
		for _, instance := range instances {
			current, err := c.client.GetInstance(instance.ID)
			if err != nil {
				return fmt.Errorf("failed to get instance status: %w", err)
			}

			if current.Status != "ACTIVE" {
				allReady = false
				break
			}
		}

		if allReady {
			log.Info().Msg("All instances are ready")
			return nil
		}

		log.Debug().Msg("Waiting for instances to be ready...")
		time.Sleep(30 * time.Second)
	}
}
