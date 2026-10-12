// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cmd

import (
	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/spf13/cobra"
)

// NewRootCommand creates the Spring Boot extension command.
func NewRootCommand() *cobra.Command {
	root, extCtx := azdext.NewExtensionRootCommand(azdext.ExtensionCommandOptions{
		Name:  "spring",
		Use:   "spring <command> [options]",
		Short: "Initializes Spring Boot applications for Azure Container Apps.",
	})
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.AddCommand(newInitCommand(extCtx))
	return root
}
