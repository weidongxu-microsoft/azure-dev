// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/azure/azure-dev/cli/azd/extensions/azure.springboot/internal/springboot"
	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/azure/azure-dev/cli/azd/pkg/input"
	"github.com/azure/azure-dev/cli/azd/pkg/project"
	"github.com/spf13/cobra"
)

func newInitCommand(extCtx *azdext.ExtensionContext) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate azure.yaml, Bicep, and Dockerfiles for an existing Maven Spring Boot app.",
		Long: "Inspect a trusted Maven Spring Boot project, review the deployment plan, and generate local azd files.\n" +
			"Analysis can execute Maven build extensions. No Azure resources are created until you run azd up.\n" +
			"Existing files are never overwritten. Use --yes to accept generation without prompting.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			previousLogOutput := log.Writer()
			log.SetOutput(io.Discard)
			if extCtx.Debug {
				log.SetOutput(cmd.ErrOrStderr())
			}
			defer log.SetOutput(previousLogOutput)
			if cmd.Flags().Changed("environment") {
				return errors.New("--environment does not apply to spring init; select it when running azd up")
			}
			if extCtx.NoPrompt && !yes {
				return fmt.Errorf("spring init requires --yes with --no-prompt: %w", &input.PromptRequiredError{
					PromptMessage: "Accept the Spring deployment plan with --yes, or run without --no-prompt to review it",
				})
			}
			root, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting project directory: %w", err)
			}
			path := filepath.Join(root, "azure.yaml")
			if _, err := os.Lstat(path); err == nil {
				return fmt.Errorf("refusing to overwrite existing %s", path)
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("checking %s: %w", path, err)
			}

			analysis, err := springboot.Analyze(cmd.Context(), root)
			if err != nil {
				return fmt.Errorf("analyzing Spring Boot project: %w", err)
			}
			confirm := func() (bool, error) {
				if yes {
					return true, nil
				}
				client, err := azdext.NewAzdClient()
				if err != nil {
					return false, fmt.Errorf("creating azd client: %w", err)
				}
				defer client.Close()
				response, err := client.Prompt().Confirm(cmd.Context(), &azdext.ConfirmRequest{
					Options: &azdext.ConfirmOptions{
						Message:      "Generate these azd project files?",
						DefaultValue: new(false),
					},
				})
				if err != nil {
					return false, fmt.Errorf("confirming generation: %w", err)
				}
				return response.GetValue(), nil
			}
			return initialize(cmd.Context(), root, cmd.OutOrStdout(), analysis, confirm)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Accept the generated deployment plan without prompting")
	return azdext.RegisterFlagOptions(cmd, azdext.FlagOptions{
		Name: "output", AllowedValues: []string{"default"},
	})
}

func initialize(
	ctx context.Context,
	root string,
	writer io.Writer,
	analysis *springboot.Analysis,
	confirm func() (bool, error),
) error {
	files, err := springboot.GenerateFiles(analysis)
	if err != nil {
		return fmt.Errorf("generating deployment plan: %w", err)
	}
	for _, file := range files {
		if file.Path == "azure.yaml" {
			if _, err := project.Parse(ctx, string(file.Content)); err != nil {
				return fmt.Errorf("validating generated azure.yaml: %w", err)
			}
		}
	}
	fmt.Fprintln(writer, "Deployment plan: Azure Container Apps, Container Registry, Log Analytics, and managed identity.")
	for _, service := range analysis.Services {
		fmt.Fprintf(writer, "  Service: %s (port %d)\n", service.Name, service.Port)
		if service.ExternalOpenAI != nil {
			fmt.Fprintln(writer, "    Connect to existing OpenAI: supply AZURE_OPENAI_ENDPOINT and AZURE_OPENAI_API_KEY.")
			if service.ExternalOpenAI.APIVersion && service.ExternalOpenAI.APIVersionDefault == "" {
				fmt.Fprintln(writer, "    Also supply AZURE_OPENAI_API_VERSION.")
			}
		}
	}
	for _, resource := range analysis.Resources {
		fmt.Fprintf(writer, "  Provision: %s (%s)\n", resource.Name, resource.Type)
	}
	fmt.Fprintln(writer, "Files to generate:")
	for _, file := range files {
		fmt.Fprintf(writer, "  %s\n", file.Path)
	}
	fmt.Fprintln(writer, "This command only writes local files; it does not provision or deploy.")
	accepted, err := confirm()
	if err != nil {
		return err
	}
	if !accepted {
		return errors.New("generation cancelled; no project files were written")
	}
	if err := springboot.Generate(root, analysis); err != nil {
		return fmt.Errorf("writing azd project files: %w", err)
	}
	fmt.Fprintln(writer, "Generated azd project files. Configure required inputs, then run azd up.")
	return nil
}
