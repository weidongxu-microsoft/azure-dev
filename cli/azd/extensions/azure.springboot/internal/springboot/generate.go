// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package springboot

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	//go:embed templates/main.bicep.tmpl
	mainBicepTemplate string
	//go:embed templates/main.parameters.json
	mainParameters string
	//go:embed templates/Dockerfile
	dockerfileTemplate string
	//go:embed templates/resources.bicep
	resourcesBicep string
	//go:embed templates/service.bicep.tmpl
	serviceBicepTemplate string
	//go:embed templates/service.parameters.json.tmpl
	serviceParametersTemplate string
)

// Generate materializes the deterministic azd configuration for a detected
// Spring Boot application. Existing azd files are never overwritten.
func Generate(root string, project *Project) error {
	files, err := GenerateFiles(project)
	if err != nil {
		return err
	}

	for _, file := range files {
		path := filepath.Join(root, file.Path)
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("refusing to overwrite existing %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("checking %s: %w", path, err)
		}
	}

	for _, file := range files {
		path := filepath.Join(root, file.Path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("creating parent directory for %s: %w", path, err)
		}
		if err := os.WriteFile(path, file.Content, 0o600); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	return nil
}

// GenerateFiles returns the deterministic azd project files for a detected application.
func GenerateFiles(project *Project) ([]GeneratedFile, error) {
	if project == nil {
		return nil, errors.New("project is required")
	}
	if project.Name == "" {
		return nil, errors.New("project name is required")
	}

	if serviceName(project.Name) != project.Name {
		return nil, fmt.Errorf("project name %q is not a normalized service name", project.Name)
	}

	replacer := strings.NewReplacer(
		"{{SERVICE_NAME}}", project.Name,
		"{{SERVICE_ENV_NAME}}", strings.ToUpper(strings.ReplaceAll(project.Name, "-", "_")),
		"{{TARGET_PORT}}", fmt.Sprintf("%d", project.Port),
	)
	files := []GeneratedFile{
		{Path: "azure.yaml", Content: []byte(azureYAML(project))},
		{Path: "Dockerfile", Content: []byte(replacer.Replace(dockerfileTemplate))},
		{Path: filepath.Join("infra", "main.bicep"), Content: []byte(replacer.Replace(mainBicepTemplate))},
		{Path: filepath.Join("infra", "main.parameters.json"), Content: []byte(mainParameters)},
		{Path: filepath.Join("infra", "resources.bicep"), Content: []byte(resourcesBicep)},
		{
			Path:    filepath.Join("infra", project.Name+".bicep"),
			Content: []byte(replacer.Replace(serviceBicepTemplate)),
		},
		{
			Path:    filepath.Join("infra", project.Name+".parameters.json"),
			Content: []byte(replacer.Replace(serviceParametersTemplate)),
		},
	}
	return files, nil
}

func azureYAML(project *Project) string {
	return fmt.Sprintf(`# yaml-language-server: $schema=https://raw.githubusercontent.com/Azure/azure-dev/main/schemas/v1.0/azure.yaml.json
name: %s
requiredVersions:
  extensions:
    azure.springboot: ">=0.1.0"
services:
  %s:
    project: .
    host: containerapp
    language: java
    module: %s
    docker:
      remoteBuild: true
`, project.Name, project.Name, project.Name)
}
