/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"github.com/wethegamers/kubefirst-api/pkg/configs"
	"github.com/wethegamers/kubefirst/internal/step"
	"github.com/spf13/cobra"
)

func VersionCommand() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "print the version number for kubefirst-cli",
		Long:  `All software has versions. This is kubefirst's`,
		Run: func(cmd *cobra.Command, _ []string) {
			stepper := step.NewStepFactory(cmd.ErrOrStderr())
			versionMsg := "\n kubefirst-cli golang utility version: " + configs.K1Version

			stepper.InfoStepString(versionMsg)
		},
	}
	return versionCmd
}
