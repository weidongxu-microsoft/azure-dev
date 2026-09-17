// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package springboot

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
	//go:embed templates/postgres.bicep
	postgresBicep string
)

// Generate materializes the deterministic azd configuration for a detected
// Spring Boot application. Existing azd files are never overwritten.
func Generate(root string, analysis *Analysis) error {
	files, err := GenerateFiles(analysis)
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
func GenerateFiles(analysis *Analysis) ([]GeneratedFile, error) {
	if analysis == nil {
		return nil, errors.New("analysis is required")
	}
	if len(analysis.Services) != 1 {
		return nil, fmt.Errorf("exactly one analyzed service is required, got %d", len(analysis.Services))
	}
	if err := validateAnalysis(analysis); err != nil {
		return nil, err
	}

	project := &analysis.Services[0]
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
	resourceModules, resourceOutputs, serviceParams, secrets, environment, err := generateResourceBindings(
		analysis,
		project,
	)
	if err != nil {
		return nil, err
	}

	mainReplacer := strings.NewReplacer(
		"{{SERVICE_NAME}}", project.Name,
		"{{SERVICE_ENV_NAME}}", strings.ToUpper(strings.ReplaceAll(project.Name, "-", "_")),
		"{{RESOURCE_MODULES}}", resourceModules,
		"{{RESOURCE_OUTPUTS}}", resourceOutputs,
	)
	serviceReplacer := strings.NewReplacer(
		"{{SERVICE_NAME}}", project.Name,
		"{{SERVICE_ENV_NAME}}", strings.ToUpper(strings.ReplaceAll(project.Name, "-", "_")),
		"{{TARGET_PORT}}", fmt.Sprintf("%d", project.Port),
		"{{RESOURCE_PARAM_DECLARATIONS}}", serviceParams,
		"{{SECRETS}}", secrets,
		"{{ENVIRONMENT_VARIABLES}}", environment,
	)
	serviceParameters, err := generateServiceParameters(analysis, project)
	if err != nil {
		return nil, err
	}

	files := []GeneratedFile{
		{Path: "azure.yaml", Content: []byte(azureYAML(analysis, project))},
		{Path: "Dockerfile", Content: []byte(replacer.Replace(dockerfileTemplate))},
		{Path: filepath.Join("infra", "main.bicep"), Content: []byte(mainReplacer.Replace(mainBicepTemplate))},
		{Path: filepath.Join("infra", "main.parameters.json"), Content: []byte(mainParameters)},
		{Path: filepath.Join("infra", "resources.bicep"), Content: []byte(resourcesBicep)},
		{
			Path:    filepath.Join("infra", project.Name+".bicep"),
			Content: []byte(serviceReplacer.Replace(serviceBicepTemplate)),
		},
		{
			Path:    filepath.Join("infra", project.Name+".parameters.json"),
			Content: serviceParameters,
		},
	}
	if hasResource(analysis, ResourceTypePostgreSQL) {
		files = append(files, GeneratedFile{
			Path:    filepath.Join("infra", "postgres.bicep"),
			Content: []byte(postgresBicep),
		})
	}
	return files, nil
}

func azureYAML(analysis *Analysis, project *Project) string {
	var uses string
	var resources string
	for _, resource := range analysis.Resources {
		if !slices.Contains(resource.Consumers, project.Name) {
			continue
		}
		uses += fmt.Sprintf("      - %s\n", resource.Name)
		resources += fmt.Sprintf("  %s:\n    type: %s\n", resource.Name, azureResourceType(resource.Type))
	}
	if uses != "" {
		uses = "    uses:\n" + uses
	}
	if resources != "" {
		resources = "resources:\n" + resources
	}

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
%s    docker:
      remoteBuild: true
%s`, project.Name, project.Name, project.Name, uses, resources)
}

func generateResourceBindings(
	analysis *Analysis,
	project *Project,
) (string, string, string, string, string, error) {
	var modules strings.Builder
	var outputs strings.Builder
	var params strings.Builder
	var secrets strings.Builder
	var environment strings.Builder
	serviceEnvName := strings.ToUpper(strings.ReplaceAll(project.Name, "-", "_"))

	for _, resource := range analysis.Resources {
		if !slices.Contains(resource.Consumers, project.Name) {
			continue
		}

		switch resource.Type {
		case ResourceTypePostgreSQL:
			modules.WriteString(`
module postgres 'postgres.bicep' = {
  name: 'postgres'
  scope: resourceGroup
  params: {
    environmentName: environmentName
    location: location
    identityPrincipalId: resources.outputs.managedIdentityPrincipalId
  }
}
`)
			fmt.Fprintf(&outputs, `
output SERVICE_%s_POSTGRES_JDBC_URL string = postgres.outputs.jdbcUrl
output SERVICE_%s_POSTGRES_USERNAME string = postgres.outputs.administratorLogin
output SERVICE_%s_POSTGRES_CREDENTIAL_REFERENCE string = postgres.outputs.credentialReference
`, serviceEnvName, serviceEnvName, serviceEnvName)
			params.WriteString(`
param postgresJdbcUrl string
param postgresUsername string
param postgresPasswordSecretUri string
`)
			secrets.WriteString(`        {
          name: 'postgres-password'
          keyVaultUrl: postgresPasswordSecretUri
          identity: identityId
        }
`)
			environment.WriteString(`            {
              name: 'SPRING_DATASOURCE_URL'
              value: postgresJdbcUrl
            }
            {
              name: 'SPRING_DATASOURCE_USERNAME'
              value: postgresUsername
            }
            {
              name: 'SPRING_DATASOURCE_PASSWORD'
              secretRef: 'postgres-password'
            }
            {
              name: 'SPRING_SQL_INIT_MODE'
              value: 'always'
            }
`)
		default:
			return "", "", "", "", "", fmt.Errorf("unsupported resource type %q", resource.Type)
		}
	}

	return modules.String(), outputs.String(), params.String(), secrets.String(), environment.String(), nil
}

func generateServiceParameters(analysis *Analysis, project *Project) ([]byte, error) {
	serviceEnvName := strings.ToUpper(strings.ReplaceAll(project.Name, "-", "_"))
	parameters := map[string]any{
		"environmentName": map[string]string{"value": "${AZURE_ENV_NAME}"},
		"location":        map[string]string{"value": "${AZURE_LOCATION}"},
		"containerRegistryName": map[string]string{
			"value": "${AZURE_CONTAINER_REGISTRY_NAME}",
		},
		"containerAppsEnvironmentName": map[string]string{
			"value": "${AZURE_CONTAINER_ENVIRONMENT_NAME}",
		},
		"imageName": map[string]string{
			"value": fmt.Sprintf("${SERVICE_%s_IMAGE_NAME}", serviceEnvName),
		},
		"identityId": map[string]string{
			"value": fmt.Sprintf("${SERVICE_%s_IDENTITY_ID}", serviceEnvName),
		},
	}

	if serviceHasResource(analysis, project.Name, ResourceTypePostgreSQL) {
		parameters["postgresJdbcUrl"] = map[string]string{
			"value": fmt.Sprintf("${SERVICE_%s_POSTGRES_JDBC_URL}", serviceEnvName),
		}
		parameters["postgresUsername"] = map[string]string{
			"value": fmt.Sprintf("${SERVICE_%s_POSTGRES_USERNAME}", serviceEnvName),
		}
		parameters["postgresPasswordSecretUri"] = map[string]string{
			"value": fmt.Sprintf("${SERVICE_%s_POSTGRES_CREDENTIAL_REFERENCE}", serviceEnvName),
		}
	}

	document := map[string]any{
		"$schema":        "https://schema.management.azure.com/schemas/2019-04-01/deploymentParameters.json#",
		"contentVersion": "1.0.0.0",
		"parameters":     parameters,
	}
	content, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encoding service parameters: %w", err)
	}
	return append(content, '\n'), nil
}

func hasResource(analysis *Analysis, resourceType ResourceType) bool {
	return slices.ContainsFunc(analysis.Resources, func(resource ResourceRequirement) bool {
		return resource.Type == resourceType
	})
}

func serviceHasResource(analysis *Analysis, serviceName string, resourceType ResourceType) bool {
	return slices.ContainsFunc(analysis.Resources, func(resource ResourceRequirement) bool {
		return resource.Type == resourceType && slices.Contains(resource.Consumers, serviceName)
	})
}

func validateAnalysis(analysis *Analysis) error {
	services := make(map[string]struct{}, len(analysis.Services))
	for _, service := range analysis.Services {
		if service.Name == "" {
			return errors.New("analyzed service name is required")
		}
		if _, exists := services[service.Name]; exists {
			return fmt.Errorf("duplicate analyzed service %q", service.Name)
		}
		services[service.Name] = struct{}{}
	}

	resources := make(map[string]struct{}, len(analysis.Resources))
	for _, resource := range analysis.Resources {
		if resource.Name == "" {
			return errors.New("analyzed resource name is required")
		}
		if _, exists := resources[resource.Name]; exists {
			return fmt.Errorf("duplicate analyzed resource %q", resource.Name)
		}
		resources[resource.Name] = struct{}{}
		if len(resource.Evidence) == 0 {
			return fmt.Errorf("analyzed resource %q has no supporting evidence", resource.Name)
		}
		if len(resource.Consumers) == 0 {
			return fmt.Errorf("analyzed resource %q has no consuming service", resource.Name)
		}
		for _, consumer := range resource.Consumers {
			if _, exists := services[consumer]; !exists {
				return fmt.Errorf("analyzed resource %q references unknown service %q", resource.Name, consumer)
			}
		}
	}
	return nil
}

func azureResourceType(resourceType ResourceType) string {
	switch resourceType {
	case ResourceTypePostgreSQL:
		return "db.postgres"
	default:
		return string(resourceType)
	}
}
