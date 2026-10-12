// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cmd

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/azure/azure-dev/cli/azd/extensions/azure.springboot/internal/springboot"
	"github.com/azure/azure-dev/cli/azd/pkg/project"
	"github.com/otiai10/copy"
	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	promptErr := errors.New("prompt failed")
	for _, tt := range []struct {
		name     string
		accepted bool
		err      error
	}{
		{name: "accept", accepted: true},
		{name: "decline"},
		{name: "prompt error", err: promptErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			analysis := &springboot.Analysis{
				Name: "sora",
				Services: []springboot.Project{
					{
						Name: "sora", Port: 8080, External: true,
						ExternalOpenAI: &springboot.ExternalOpenAI{APIVersion: true, APIVersionDefault: "preview"},
					},
				},
			}
			var output bytes.Buffer
			called := false
			err := initialize(t.Context(), root, &output, analysis, func() (bool, error) {
				called = true
				require.Contains(t, output.String(), "Files to generate:")
				require.NoFileExists(t, filepath.Join(root, "azure.yaml"))
				return tt.accepted, tt.err
			})
			require.True(t, called)
			require.Contains(t, output.String(), "Connect to existing OpenAI")
			require.NotContains(t, output.String(), "Also supply AZURE_OPENAI_API_VERSION")
			require.NotContains(t, output.String(), "Provision:")
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else if !tt.accepted {
				require.ErrorContains(t, err, "generation cancelled")
			} else {
				require.NoError(t, err)
				config, err := project.Load(t.Context(), filepath.Join(root, "azure.yaml"))
				require.NoError(t, err)
				require.Nil(t, config.RequiredVersions)
				require.FileExists(t, filepath.Join(root, "infra", "main.bicep"))
				require.FileExists(t, filepath.Join(root, "infra", "sora.bicep"))
				require.FileExists(t, filepath.Join(root, "Dockerfile"))
				dockerignore, err := os.ReadFile(filepath.Join(root, ".dockerignore"))
				require.NoError(t, err)
				require.Contains(t, string(dockerignore), ".env\n.env.*\n")
				require.Contains(t, output.String(), "then run azd up")
			}
			if !tt.accepted {
				entries, err := os.ReadDir(root)
				require.NoError(t, err)
				require.Empty(t, entries)
			}
		})
	}
}

func TestInitializePreservesExistingFiles(t *testing.T) {
	for _, path := range []string{"azure.yaml", ".dockerignore", "Dockerfile", filepath.Join("infra", "main.bicep")} {
		t.Run(path, func(t *testing.T) {
			root := t.TempDir()
			target := filepath.Join(root, path)
			require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
			require.NoError(t, os.WriteFile(target, []byte("user-owned"), 0o600))
			analysis := &springboot.Analysis{
				Name: "app", Services: []springboot.Project{{Name: "app", Port: 8080, External: true}},
			}
			var output bytes.Buffer
			err := initialize(t.Context(), root, &output, analysis, func() (bool, error) { return true, nil })
			require.ErrorContains(t, err, "refusing to overwrite existing")
			data, err := os.ReadFile(target)
			require.NoError(t, err)
			require.Equal(t, "user-owned", string(data))
			if path != "azure.yaml" {
				require.NoFileExists(t, filepath.Join(root, "azure.yaml"))
			}
			require.NotContains(t, output.String(), "Generated azd project files.")
		})
	}
}

func TestSpringInitValidation(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{name: "confirmation required", args: []string{"init", "--no-prompt"}, want: "--yes"},
		{name: "environment ignored", args: []string{"init", "--environment", "dev"}, want: "--environment does not apply"},
		{name: "unsupported output", args: []string{"init", "--output", "json"}, want: "json"},
		{name: "positional argument", args: []string{"init", "unexpected"}, want: "unknown command"},
		{name: "listener removed", args: []string{"listen"}, want: "unknown command"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			cmd := NewRootCommand()
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs(tt.args)
			err := cmd.ExecuteContext(t.Context())
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestSpringInitExistingProject(t *testing.T) {
	t.Chdir(t.TempDir())
	require.NoError(t, os.WriteFile("azure.yaml", []byte("user-owned"), 0o600))
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"init", "--yes", "--no-prompt"})
	err := cmd.ExecuteContext(t.Context())
	require.ErrorContains(t, err, "refusing to overwrite existing")
	data, err := os.ReadFile("azure.yaml")
	require.NoError(t, err)
	require.Equal(t, "user-owned", string(data))
}

func TestSpringInitExistingCases(t *testing.T) {
	fixtures, err := filepath.Abs(filepath.Join("..", "..", "testdata"))
	require.NoError(t, err)
	// The shared Java detector caches the Maven wrapper across subtests.
	workspace := t.TempDir()
	for _, tt := range []struct {
		name     string
		services []string
		resource string
	}{
		{name: "spring-todo", services: []string{"spring-todo"}, resource: "postgres"},
		{name: "spring-ai-todo-agent", services: []string{"spring-ai-todo-agent"}, resource: "foundry"},
		{
			name:     "spring-orders-microservices",
			services: []string{"catalog-service", "orders-service"}, resource: "postgres",
		},
		{name: "spring-external-openai", services: []string{"spring-external-openai"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := filepath.Join(workspace, tt.name)
			require.NoError(t, copy.Copy(filepath.Join(fixtures, tt.name), root, copy.Options{
				Skip: func(info fs.FileInfo, _, _ string) (bool, error) {
					return info.IsDir() && (info.Name() == "target" || info.Name() == ".azure"), nil
				},
			}))
			originals := map[string][]byte{}
			require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
				if err != nil || entry.IsDir() {
					return err
				}
				data, err := os.ReadFile(path)
				originals[path] = data
				return err
			}))
			t.Chdir(root)
			var output bytes.Buffer
			cmd := NewRootCommand()
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs([]string{"init", "--yes", "--no-prompt"})
			require.NoError(t, cmd.ExecuteContext(t.Context()))
			config, err := project.Load(t.Context(), filepath.Join(root, "azure.yaml"))
			require.NoError(t, err)
			require.Nil(t, config.RequiredVersions)
			require.Len(t, config.Services, len(tt.services))
			for _, service := range tt.services {
				require.Contains(t, config.Services, service)
				require.FileExists(t, filepath.Join(root, "infra", service+".bicep"))
			}
			require.FileExists(t, filepath.Join(root, "infra", "main.bicep"))
			for _, resource := range []string{"postgres", "foundry"} {
				path := filepath.Join(root, "infra", resource+".bicep")
				if resource == tt.resource {
					require.FileExists(t, path)
				} else {
					require.NoFileExists(t, path)
				}
			}
			require.NoFileExists(t, filepath.Join(root, "order-contracts", "Dockerfile"))
			require.Contains(t, output.String(), "Generated azd project files.")
			for path, original := range originals {
				current, err := os.ReadFile(path)
				require.NoError(t, err)
				require.Equal(t, original, current, "application file changed: %s", path)
			}
			repeated := NewRootCommand()
			repeated.SetOut(&output)
			repeated.SetErr(&output)
			repeated.SetArgs([]string{"init", "--yes", "--no-prompt"})
			require.ErrorContains(t, repeated.ExecuteContext(t.Context()), "refusing to overwrite existing")
		})
	}
}
