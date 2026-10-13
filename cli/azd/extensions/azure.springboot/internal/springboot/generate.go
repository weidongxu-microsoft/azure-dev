// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package springboot

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/azure/azure-dev/cli/azd/pkg/infra/provisioning"
	"github.com/azure/azure-dev/cli/azd/pkg/project"
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
	//go:embed templates/foundry.bicep
	foundryBicep string
)

// Generate materializes the deterministic azd configuration for a detected
// Spring Boot application. Existing azd files are never overwritten.
func Generate(root string, analysis *Analysis) (resultErr error) {
	files, err := GenerateFiles(analysis)
	if err != nil {
		return err
	}

	directory, err := os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("opening project directory: %w", err)
	}
	defer directory.Close()

	for _, file := range files {
		path := filepath.Join(root, file.Path)
		if _, err := directory.Lstat(file.Path); err == nil {
			return fmt.Errorf("refusing to overwrite existing %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("checking %s: %w", path, err)
		}
	}

	var created []string
	defer func() {
		if resultErr != nil {
			for _, path := range created {
				if err := directory.Remove(path); err != nil {
					resultErr = errors.Join(resultErr, fmt.Errorf("removing generated %s: %w", path, err))
				}
			}
		}
	}()
	for _, file := range files {
		path := filepath.Join(root, file.Path)
		if err := directory.MkdirAll(filepath.Dir(file.Path), 0o755); err != nil {
			return fmt.Errorf("creating parent directory for %s: %w", path, err)
		}
		target, err := directory.OpenFile(file.Path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return fmt.Errorf("creating %s: %w", path, err)
		}
		created = append(created, file.Path)
		_, writeErr := target.Write(file.Content)
		if err := errors.Join(writeErr, target.Close()); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	if _, err := project.Load(context.Background(), filepath.Join(root, "azure.yaml")); err != nil {
		return fmt.Errorf("validating generated azure.yaml: %w", err)
	}
	return nil
}

// GenerateFiles returns the deterministic azd project files for a detected application.
func GenerateFiles(analysis *Analysis) ([]GeneratedFile, error) {
	if analysis == nil {
		return nil, errors.New("analysis is required")
	}
	if len(analysis.Services) == 0 {
		return nil, errors.New("at least one analyzed service is required")
	}
	if err := validateAnalysis(analysis); err != nil {
		return nil, err
	}

	if analysis.Name == "" {
		return nil, errors.New("analysis name is required")
	}
	if serviceName(analysis.Name) != analysis.Name {
		return nil, fmt.Errorf("analysis name %q is not a normalized project name", analysis.Name)
	}

	projectFile, err := azureYAML(context.Background(), analysis)
	if err != nil {
		return nil, fmt.Errorf("generating azure.yaml: %w", err)
	}
	resourceModules, resourceOutputs, err := generateResourceInfrastructure(analysis)
	if err != nil {
		return nil, err
	}
	mainReplacer := strings.NewReplacer(
		"{{RESOURCE_MODULES}}", resourceModules,
		"{{RESOURCE_OUTPUTS}}", resourceOutputs,
		"{{SERVICE_IDENTITY_OUTPUTS}}", generateIdentityOutputs(analysis),
	)

	files := []GeneratedFile{
		{Path: "azure.yaml", Content: projectFile},
		{Path: ".dockerignore", Content: []byte("**/target\n.azure\ninfra\n.git\n.env\n.env.*\n")},
		{Path: filepath.Join("infra", "main.bicep"), Content: []byte(mainReplacer.Replace(mainBicepTemplate))},
		{Path: filepath.Join("infra", "main.parameters.json"), Content: []byte(mainParameters)},
		{Path: filepath.Join("infra", "resources.bicep"), Content: []byte(resourcesBicep)},
	}
	for i := range analysis.Services {
		project := &analysis.Services[i]
		serviceParams, secrets, environment, err := generateServiceBindings(analysis, project)
		if err != nil {
			return nil, err
		}
		serviceReplacer := strings.NewReplacer(
			"{{SERVICE_NAME}}", project.Name,
			"{{SERVICE_ENV_NAME}}", serviceEnvironmentName(project.Name),
			"{{TARGET_PORT}}", fmt.Sprintf("%d", project.Port),
			"{{EXTERNAL_INGRESS}}", fmt.Sprintf("%t", project.External),
			"{{RESOURCE_PARAM_DECLARATIONS}}", serviceParams,
			"{{SECRETS}}", secrets,
			"{{ENVIRONMENT_VARIABLES}}", environment,
			"{{HEALTH_PROBES}}", healthProbes(project),
		)
		mavenModule := project.Path
		artifactDir := "/app"
		if mavenModule == "" {
			mavenModule = "."
		} else {
			artifactDir += "/" + filepath.ToSlash(mavenModule)
		}
		dockerfileReplacer := strings.NewReplacer(
			"{{MAVEN_MODULE}}", mavenModule,
			"{{ARTIFACT_DIR}}", artifactDir,
			"{{TARGET_PORT}}", fmt.Sprintf("%d", project.Port),
		)
		serviceParameters, err := generateServiceParameters(analysis, project)
		if err != nil {
			return nil, err
		}
		files = append(files,
			GeneratedFile{
				Path:    filepath.Join(filepath.FromSlash(project.Path), "Dockerfile"),
				Content: []byte(dockerfileReplacer.Replace(dockerfileTemplate)),
			},
			GeneratedFile{
				Path:    filepath.Join("infra", project.Name+".bicep"),
				Content: []byte(serviceReplacer.Replace(serviceBicepTemplate)),
			},
			GeneratedFile{
				Path:    filepath.Join("infra", project.Name+".parameters.json"),
				Content: serviceParameters,
			},
		)
	}
	if hasResource(analysis, ResourceTypePostgreSQL) {
		files = append(files, GeneratedFile{
			Path:    filepath.Join("infra", "postgres.bicep"),
			Content: []byte(postgresBicep),
		})
	}
	if hasResource(analysis, ResourceTypeFoundryModel) {
		files = append(files, GeneratedFile{
			Path:    filepath.Join("infra", "foundry.bicep"),
			Content: []byte(foundryBicep),
		})
	}
	return files, nil
}

func azureYAML(ctx context.Context, analysis *Analysis) (content []byte, resultErr error) {
	config := &project.ProjectConfig{
		Name:      analysis.Name,
		Infra:     provisioning.Options{Provider: provisioning.Bicep, Path: "infra"},
		Services:  make(map[string]*project.ServiceConfig, len(analysis.Services)),
		Resources: make(map[string]*project.ResourceConfig, len(analysis.Resources)),
	}
	for i := range analysis.Services {
		service := &analysis.Services[i]
		var uses []string
		for _, dependency := range service.Dependencies {
			uses = append(uses, dependency.Service)
		}
		for _, resource := range analysis.Resources {
			if slices.Contains(resource.Consumers, service.Name) {
				uses = append(uses, resource.Name)
			}
		}
		projectPath := service.Path
		dockerContext := ".."
		if projectPath == "" {
			projectPath = "."
			dockerContext = "."
		}
		config.Services[service.Name] = &project.ServiceConfig{
			RelativePath: projectPath,
			Host:         project.ContainerAppTarget,
			Language:     project.ServiceLanguageJava,
			Module:       service.Name,
			Uses:         uses,
			Docker: project.DockerProjectOptions{
				Path: "./Dockerfile", Context: dockerContext, RemoteBuild: true,
			},
		}
	}
	for _, resource := range analysis.Resources {
		config.Resources[resource.Name] = &project.ResourceConfig{
			Type: project.ResourceType(azureResourceType(resource.Type)),
		}
	}

	// Save owns YAML validation, quoting, path normalization, and schema annotation.
	// Stage outside the project so its overwrite behavior cannot bypass our exclusive writer.
	staging, err := os.MkdirTemp("", "azd-spring-project-")
	if err != nil {
		return nil, fmt.Errorf("creating project serialization directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(staging); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("removing project serialization directory: %w", err))
		}
	}()

	path := filepath.Join(staging, "azure.yaml")
	if err := project.Save(ctx, config, path); err != nil {
		return nil, fmt.Errorf("serializing project configuration: %w", err)
	}
	content, err = os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading serialized project configuration: %w", err)
	}
	return content, nil
}

func generateResourceInfrastructure(analysis *Analysis) (string, string, error) {
	var modules strings.Builder
	var outputs strings.Builder

	for _, resource := range analysis.Resources {
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
			for _, consumer := range resource.Consumers {
				serviceEnvName := serviceEnvironmentName(consumer)
				fmt.Fprintf(&outputs, `
output SERVICE_%s_POSTGRES_JDBC_URL string = postgres.outputs.jdbcUrl
output SERVICE_%s_POSTGRES_USERNAME string = postgres.outputs.administratorLogin
output SERVICE_%s_POSTGRES_CREDENTIAL_REFERENCE string = postgres.outputs.credentialReference
`, serviceEnvName, serviceEnvName, serviceEnvName)
			}
		case ResourceTypeFoundryModel:
			modules.WriteString(`
module foundry 'foundry.bicep' = {
  name: 'foundry'
  scope: resourceGroup
  params: {
    environmentName: environmentName
    location: location
    identityPrincipalId: resources.outputs.managedIdentityPrincipalId
  }
}
`)
			for _, consumer := range resource.Consumers {
				serviceEnvName := serviceEnvironmentName(consumer)
				fmt.Fprintf(&outputs, `
output SERVICE_%s_FOUNDRY_ENDPOINT string = foundry.outputs.endpoint
output SERVICE_%s_FOUNDRY_MODEL string = foundry.outputs.modelDeploymentName
`, serviceEnvName, serviceEnvName)
			}
		default:
			return "", "", fmt.Errorf("unsupported resource type %q", resource.Type)
		}
	}

	return modules.String(), outputs.String(), nil
}

func generateServiceBindings(
	analysis *Analysis,
	project *Project,
) (string, string, string, error) {
	var params strings.Builder
	var secrets strings.Builder
	var environment strings.Builder

	if serviceHasResource(analysis, project.Name, ResourceTypePostgreSQL) {
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
	}
	if serviceHasResource(analysis, project.Name, ResourceTypeFoundryModel) {
		params.WriteString(`
param foundryEndpoint string
param foundryModel string
`)
		environment.WriteString(`            {
              name: 'AZURE_INFERENCE_ENDPOINT'
              value: foundryEndpoint
            }
            {
              name: 'AZURE_INFERENCE_MODEL'
              value: foundryModel
            }
`)
	}
	if project.ExternalOpenAI != nil {
		params.WriteString(`
@minLength(1)
param azureOpenAiEndpoint string
@secure()
@minLength(1)
param azureOpenAiApiKey string
`)
		secrets.WriteString(`        {
          name: 'azure-openai-api-key'
          value: azureOpenAiApiKey
        }
`)
		environment.WriteString(`            {
              name: 'AZURE_OPENAI_ENDPOINT'
              value: azureOpenAiEndpoint
            }
            {
              name: 'AZURE_OPENAI_API_KEY'
              secretRef: 'azure-openai-api-key'
            }
`)
		if project.ExternalOpenAI.APIVersion {
			params.WriteString("@minLength(1)\nparam azureOpenAiApiVersion string\n")
			environment.WriteString(`            {
              name: 'AZURE_OPENAI_API_VERSION'
              value: azureOpenAiApiVersion
            }
`)
		}
	}
	for _, dependency := range project.Dependencies {
		paramName := serviceURLParameterName(dependency.Service)
		fmt.Fprintf(&params, "\nparam %s string\n", paramName)
		fmt.Fprintf(&environment, `            {
              name: '%s'
              value: %s
            }
`, dependency.EnvironmentVariable, paramName)
	}

	return params.String(), secrets.String(), environment.String(), nil
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
		"identityClientId": map[string]string{
			"value": fmt.Sprintf("${SERVICE_%s_IDENTITY_CLIENT_ID}", serviceEnvName),
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
	if serviceHasResource(analysis, project.Name, ResourceTypeFoundryModel) {
		parameters["foundryEndpoint"] = map[string]string{
			"value": fmt.Sprintf("${SERVICE_%s_FOUNDRY_ENDPOINT}", serviceEnvName),
		}
		parameters["foundryModel"] = map[string]string{
			"value": fmt.Sprintf("${SERVICE_%s_FOUNDRY_MODEL}", serviceEnvName),
		}
	}
	if project.ExternalOpenAI != nil {
		parameters["azureOpenAiEndpoint"] = map[string]string{"value": "${AZURE_OPENAI_ENDPOINT}"}
		parameters["azureOpenAiApiKey"] = map[string]string{"value": "${AZURE_OPENAI_API_KEY}"}
		if project.ExternalOpenAI.APIVersion {
			version := "${AZURE_OPENAI_API_VERSION}"
			if project.ExternalOpenAI.APIVersionDefault != "" {
				version = fmt.Sprintf("${AZURE_OPENAI_API_VERSION:-%s}", project.ExternalOpenAI.APIVersionDefault)
			}
			parameters["azureOpenAiApiVersion"] = map[string]string{"value": version}
		}
	}
	for _, dependency := range project.Dependencies {
		parameters[serviceURLParameterName(dependency.Service)] = map[string]string{
			"value": fmt.Sprintf("${SERVICE_%s_URL}", serviceEnvironmentName(dependency.Service)),
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

func generateIdentityOutputs(analysis *Analysis) string {
	var outputs strings.Builder
	for _, service := range analysis.Services {
		serviceEnvName := serviceEnvironmentName(service.Name)
		fmt.Fprintf(&outputs, "output SERVICE_%s_IDENTITY_ID string = resources.outputs.managedIdentityId\n",
			serviceEnvName)
		fmt.Fprintf(&outputs,
			"output SERVICE_%s_IDENTITY_CLIENT_ID string = resources.outputs.managedIdentityClientId\n",
			serviceEnvName)
	}
	return outputs.String()
}

func serviceEnvironmentName(service string) string {
	return strings.ToUpper(strings.ReplaceAll(service, "-", "_"))
}

func healthProbes(project *Project) string {
	if !project.ActuatorAvailable {
		return ""
	}
	return fmt.Sprintf(`          probes: [
            {
              type: 'Liveness'
              httpGet: {
                path: '/actuator/health/liveness'
                port: %d
                scheme: 'HTTP'
              }
            }
            {
              type: 'Readiness'
              httpGet: {
                path: '/actuator/health/readiness'
                port: %d
                scheme: 'HTTP'
              }
            }
          ]
`, project.Port, project.Port)
}

func serviceURLParameterName(service string) string {
	parts := strings.Split(service, "-")
	var name strings.Builder
	name.WriteString(parts[0])
	for _, part := range parts[1:] {
		if part == "" {
			continue
		}
		name.WriteString(strings.ToUpper(part[:1]))
		name.WriteString(part[1:])
	}
	name.WriteString("Url")
	return name.String()
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
		if serviceName(service.Name) != service.Name {
			return fmt.Errorf("service name %q is not normalized", service.Name)
		}
		if _, exists := services[service.Name]; exists {
			return fmt.Errorf("duplicate analyzed service %q", service.Name)
		}
		services[service.Name] = struct{}{}
	}
	for _, service := range analysis.Services {
		for _, dependency := range service.Dependencies {
			if _, exists := services[dependency.Service]; !exists {
				return fmt.Errorf("service %q references unknown service %q", service.Name, dependency.Service)
			}
			if dependency.Service == service.Name {
				return fmt.Errorf("service %q cannot depend on itself", service.Name)
			}
			if dependency.EnvironmentVariable == "" {
				return fmt.Errorf("service %q dependency %q has no environment variable", service.Name,
					dependency.Service)
			}
			if len(dependency.Evidence) == 0 {
				return fmt.Errorf("service %q dependency %q has no supporting evidence", service.Name,
					dependency.Service)
			}
		}
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
	case ResourceTypeFoundryModel:
		return "ai.openai.model"
	default:
		return string(resourceType)
	}
}
