/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package talosbootstrap

import (
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

// CommandRunner handles running system commands
type CommandRunner struct {
	kubeconfig string
	DryRun     bool
}

// NewCommandRunner creates a new command runner
func NewCommandRunner(kubeconfig string, dryRun bool) *CommandRunner {
	return &CommandRunner{
		kubeconfig: kubeconfig,
		DryRun:     dryRun,
	}
}

// RunKubectl runs a kubectl command
func (cr *CommandRunner) RunKubectl(args ...string) error {
	args = append([]string{"--kubeconfig", cr.kubeconfig}, args...)
	return cr.runCommand("kubectl", args...)
}

// RunHelm runs a helm command
func (cr *CommandRunner) RunHelm(args ...string) error {
	args = append([]string{"--kubeconfig", cr.kubeconfig}, args...)
	return cr.runCommand("helm", args...)
}

// runCommand runs a system command
func (cr *CommandRunner) runCommand(command string, args ...string) error {
	cmdStr := command + " " + strings.Join(args, " ")
	log.Debug().Msgf("Running command: %s", cmdStr)
	
	if cr.DryRun {
		log.Info().Msgf("DRY RUN: Would run: %s", cmdStr)
		return nil
	}

	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}
