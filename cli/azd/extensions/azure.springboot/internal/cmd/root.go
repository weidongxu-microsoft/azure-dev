// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cmd

import (
	"github.com/azure/azure-dev/cli/azd/extensions/azure.springboot/internal/project"
	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/spf13/cobra"
)

// NewRootCommand creates the Spring Boot extension command.
func NewRootCommand() *cobra.Command {
	root, _ := azdext.NewExtensionRootCommand(azdext.ExtensionCommandOptions{
		Name:  "spring",
		Use:   "spring <command> [options]",
		Short: "Initializes Spring Boot applications for Azure Container Apps.",
	})
	root.AddCommand(azdext.NewListenCommand(func(host *azdext.ExtensionHost) {
		host.WithInitProvider("spring-boot", project.NewInitProvider)
	}))
	return root
}
