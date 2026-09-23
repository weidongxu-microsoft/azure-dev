// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package springboot

import (
	"os"
	"path/filepath"
	"testing"

	azdproject "github.com/azure/azure-dev/cli/azd/pkg/project"
	"github.com/stretchr/testify/require"
)

func TestDetectSpringTodo(t *testing.T) {
	project, err := Detect(fixturePath(t))

	require.NoError(t, err)
	require.Equal(t, &Project{
		Name:              "spring-todo",
		ArtifactID:        "spring-todo",
		Version:           "0.0.1-SNAPSHOT",
		JavaVersion:       "21",
		Port:              8080,
		ActuatorAvailable: true,
	}, project)
}

func TestDetectRejectsNonSpringProject(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "pom.xml"),
		[]byte("<project><artifactId>plain-java</artifactId><version>1.0.0</version></project>"),
		0o600,
	))

	_, err := Detect(root)

	require.ErrorIs(t, err, ErrNotSpringBoot)
}

func TestAnalyzeSpringTodo(t *testing.T) {
	analysis, err := Analyze(t.Context(), fixturePath(t))

	require.NoError(t, err)
	require.Len(t, analysis.Services, 1)
	require.Equal(t, "spring-todo", analysis.Services[0].Name)
	require.Equal(t, []ResourceRequirement{
		{
			Name:      "postgres",
			Type:      ResourceTypePostgreSQL,
			Consumers: []string{"spring-todo"},
			Evidence: []Evidence{
				{
					Type:   "dependency",
					Source: "pom.xml",
					Value:  "PostgreSQL runtime dependency in the effective Maven model",
				},
			},
		},
	}, analysis.Resources)
}

func TestAnalyzeSpringOrdersMicroservices(t *testing.T) {
	analysis, err := Analyze(t.Context(), microservicesFixturePath(t))

	require.NoError(t, err)
	require.Equal(t, "spring-orders-microservices", analysis.Name)
	require.Equal(t, []Project{
		{
			Name:              "catalog-service",
			ArtifactID:        "catalog-service",
			Version:           "0.0.1-SNAPSHOT",
			JavaVersion:       "21",
			Path:              "catalog-service",
			Port:              8081,
			ActuatorAvailable: true,
			External:          false,
		},
		{
			Name:              "orders-service",
			ArtifactID:        "orders-service",
			Version:           "0.0.1-SNAPSHOT",
			JavaVersion:       "21",
			Path:              "orders-service",
			Port:              8080,
			ActuatorAvailable: true,
			External:          true,
			Dependencies: []ServiceDependency{
				{
					Service:             "catalog-service",
					EnvironmentVariable: "CATALOG_SERVICE_URL",
					Evidence: []Evidence{
						{
							Type:   "configuration",
							Source: "orders-service/src/main/resources/application.properties",
							Value:  "catalog.service.url references CATALOG_SERVICE_URL",
						},
					},
				},
			},
		},
	}, analysis.Services)
	require.Equal(t, []ResourceRequirement{
		{
			Name:      "postgres",
			Type:      ResourceTypePostgreSQL,
			Consumers: []string{"orders-service"},
			Evidence: []Evidence{
				{
					Type:   "dependency",
					Source: "orders-service/pom.xml",
					Value:  "PostgreSQL runtime dependency in the effective Maven model",
				},
			},
		},
	}, analysis.Resources)
}

func TestAnalyzeSpringAiTodoAgent(t *testing.T) {
	analysis, err := Analyze(t.Context(), aiFixturePath(t))

	require.NoError(t, err)
	require.Len(t, analysis.Services, 1)
	require.Equal(t, "spring-ai-todo-agent", analysis.Services[0].Name)
	require.Equal(t, []ResourceRequirement{
		{
			Name:      "foundry-model",
			Type:      ResourceTypeFoundryModel,
			Consumers: []string{"spring-ai-todo-agent"},
			Evidence: []Evidence{
				{
					Type:   "dependency",
					Source: "pom.xml",
					Value:  "Spring AI OpenAI model starter with Azure Identity",
				},
				{
					Type:   "configuration",
					Source: "application.properties",
					Value:  "Microsoft Foundry Models inference environment contract",
				},
			},
		},
	}, analysis.Resources)
}

func TestExternalOpenAIConfiguration(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "spring-external-openai")
	t.Run("properties", func(t *testing.T) {
		t.Setenv("AZURE_OPENAI_API_KEY", "synthetic-do-not-emit")
		project, err := Detect(root)
		require.NoError(t, err)
		require.False(t, project.ActuatorAvailable)
		require.Equal(t, 8090, project.Port)
		require.Equal(t, &ExternalOpenAI{APIVersion: true}, project.ExternalOpenAI)

		analysis, err := Analyze(t.Context(), root)
		require.NoError(t, err)
		require.Empty(t, analysis.Resources)
		require.Len(t, analysis.Services, 1)
		files, err := GenerateFiles(analysis)
		require.NoError(t, err)
		for _, file := range files {
			require.NotContains(t, string(file.Content), "synthetic-do-not-emit")
		}
		require.NotContains(t, generatedPaths(files), filepath.Join("infra", "foundry.bicep"))
		require.NotContains(t, generatedContent(files, "azure.yaml"), "ai.openai.model")
		require.NotContains(t, generatedContent(files, filepath.Join("infra", "main.bicep")), "module foundry")
		service := generatedContent(files, filepath.Join("infra", "spring-external-openai.bicep"))
		require.Contains(t, service, "@secure()\n@minLength(1)\nparam azureOpenAiApiKey string")
		require.Contains(t, service, "name: 'AZURE_OPENAI_API_KEY'\n              secretRef: 'azure-openai-api-key'")
		require.Contains(t, service, "value: azureOpenAiEndpoint")
		require.Contains(t, service, "value: azureOpenAiApiVersion")
		require.NotContains(t, service, "/actuator/health")
		parameters := generatedContent(files, filepath.Join("infra", "spring-external-openai.parameters.json"))
		require.Contains(t, parameters, `"azureOpenAiEndpoint"`)
		require.Contains(t, parameters, "${AZURE_OPENAI_ENDPOINT}")
		require.Contains(t, parameters, "${AZURE_OPENAI_API_KEY}")
		require.Contains(t, parameters, "${AZURE_OPENAI_API_VERSION}")
		require.NotContains(t, parameters, `"value": ""`)
	})
	t.Run("yaml only", func(t *testing.T) {
		yamlRoot := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(yamlRoot, "src", "main", "resources"), 0o755))
		pom, err := os.ReadFile(filepath.Join(root, "pom.xml"))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(yamlRoot, "pom.xml"), pom, 0o600))
		yml, err := os.ReadFile(filepath.Join(root, "src", "main", "resources", "application.yml"))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(
			filepath.Join(yamlRoot, "src", "main", "resources", "application.yml"), yml, 0o600))
		project, err := Detect(yamlRoot)
		require.NoError(t, err)
		require.Equal(t, &ExternalOpenAI{APIVersion: true}, project.ExternalOpenAI)
		require.Equal(t, 8080, project.Port)
	})
	t.Run("incomplete bindings fail", func(t *testing.T) {
		for _, properties := range []map[string]string{
			{"azure.openai.endpoint": "${AZURE_OPENAI_ENDPOINT}"},
			{"azure.openai.endpoint": "${AZURE_OPENAI_ENDPOINT}", "azure.openai.api-key": "${WRONG_KEY}"},
		} {
			_, err := detectExternalOpenAI(properties)
			require.ErrorContains(t, err, "AZURE_OPENAI_API_KEY")
		}
		_, err := detectExternalOpenAI(map[string]string{
			"azure.openai.endpoint":    "${AZURE_OPENAI_ENDPOINT}",
			"azure.openai.api-key":     "${AZURE_OPENAI_API_KEY}",
			"azure.openai.api-version": "${WRONG_VERSION}",
		})
		require.ErrorContains(t, err, "AZURE_OPENAI_API_VERSION")
	})
	t.Run("API version inherits Spring default unless overridden", func(t *testing.T) {
		properties := map[string]string{
			"azure.openai.endpoint":    "${AZURE_OPENAI_ENDPOINT}",
			"azure.openai.api-key":     "${AZURE_OPENAI_API_KEY}",
			"azure.openai.api-version": "${AZURE_OPENAI_API_VERSION:preview}",
		}
		openAI, err := detectExternalOpenAI(properties)
		require.NoError(t, err)
		require.Equal(t, &ExternalOpenAI{APIVersion: true, APIVersionDefault: "preview"}, openAI)
		files, err := GenerateFiles(&Analysis{
			Name: "example",
			Services: []Project{
				{Name: "example", Port: 8080, External: true, ExternalOpenAI: openAI},
			},
		})
		require.NoError(t, err)
		parameters := generatedContent(files, filepath.Join("infra", "example.parameters.json"))
		require.Contains(t, parameters, "${AZURE_OPENAI_API_VERSION:-preview}")
	})
	t.Run("missing inputs remain required", func(t *testing.T) {
		analysis := &Analysis{
			Name: "example",
			Services: []Project{
				{Name: "example", Port: 8080, External: true, ExternalOpenAI: &ExternalOpenAI{}},
			},
		}
		files, err := GenerateFiles(analysis)
		require.NoError(t, err)
		service := generatedContent(files, filepath.Join("infra", "example.bicep"))
		require.Contains(t, service, "@minLength(1)\nparam azureOpenAiEndpoint string")
		require.Contains(t, service, "@minLength(1)\nparam azureOpenAiApiKey string")
		require.NotContains(t, generatedContent(files, filepath.Join("infra", "example.parameters.json")),
			"AZURE_OPENAI_API_VERSION")
	})
}

func TestGenerateProject(t *testing.T) {
	root := t.TempDir()
	analysis := &Analysis{
		Name: "spring-todo",
		Services: []Project{
			{
				Name:              "spring-todo",
				ArtifactID:        "spring-todo",
				Version:           "0.0.1-SNAPSHOT",
				JavaVersion:       "21",
				Port:              8080,
				ActuatorAvailable: true,
				External:          true,
			},
		},
		Resources: []ResourceRequirement{
			{
				Name:      "postgres",
				Type:      ResourceTypePostgreSQL,
				Consumers: []string{"spring-todo"},
				Evidence: []Evidence{
					{Type: "dependency", Source: "test", Value: "org.postgresql:postgresql"},
				},
			},
		},
	}

	require.NoError(t, Generate(root, analysis))

	azureYAML, err := os.ReadFile(filepath.Join(root, "azure.yaml"))
	require.NoError(t, err)
	require.Contains(t, string(azureYAML), "azure.springboot: \">=0.1.0\"")
	require.Contains(t, string(azureYAML), "language: java")
	require.Contains(t, string(azureYAML), "remoteBuild: true")
	require.Contains(t, string(azureYAML), "uses:\n      - postgres")
	require.Contains(t, string(azureYAML), "postgres:\n    type: db.postgres")
	_, err = azdproject.Load(t.Context(), filepath.Join(root, "azure.yaml"))
	require.NoError(t, err)

	dockerfile, err := os.ReadFile(filepath.Join(root, "Dockerfile"))
	require.NoError(t, err)
	require.Contains(t, string(dockerfile), "FROM maven:3.9.11-eclipse-temurin-21 AS build")
	require.Contains(t, string(dockerfile), "COPY --from=build /app/target/*.jar app.jar")
	require.Contains(t, string(dockerfile), "EXPOSE 8080")

	serviceBicep, err := os.ReadFile(filepath.Join(root, "infra", "spring-todo.bicep"))
	require.NoError(t, err)
	require.Contains(t, string(serviceBicep), "targetPort: 8080")
	require.Contains(t, string(serviceBicep), "path: '/actuator/health/readiness'")
	require.Contains(t, string(serviceBicep), "name: 'SPRING_DATASOURCE_URL'")
	require.Contains(t, string(serviceBicep), "keyVaultUrl: postgresPasswordSecretUri")

	serviceParameters, err := os.ReadFile(filepath.Join(root, "infra", "spring-todo.parameters.json"))
	require.NoError(t, err)
	require.Contains(t, string(serviceParameters), "${SERVICE_SPRING_TODO_IMAGE_NAME}")
	require.Contains(t, string(serviceParameters), "${SERVICE_SPRING_TODO_POSTGRES_JDBC_URL}")
	require.Contains(t, string(serviceParameters), "${SERVICE_SPRING_TODO_POSTGRES_CREDENTIAL_REFERENCE}")

	mainBicep, err := os.ReadFile(filepath.Join(root, "infra", "main.bicep"))
	require.NoError(t, err)
	require.Contains(t, string(mainBicep), "module postgres 'postgres.bicep'")
	require.Contains(t, string(mainBicep), "SERVICE_SPRING_TODO_POSTGRES_JDBC_URL")

	postgres, err := os.ReadFile(filepath.Join(root, "infra", "postgres.bicep"))
	require.NoError(t, err)
	require.Contains(t, string(postgres), "Microsoft.DBforPostgreSQL/flexibleServers@2024-08-01")
	require.Contains(t, string(postgres), "postgres-administrator-password")

	mainParameters, err := os.ReadFile(filepath.Join(root, "infra", "main.parameters.json"))
	require.NoError(t, err)
	require.Contains(t, string(mainParameters), "${AZURE_ENV_NAME}")
	require.Contains(t, string(mainParameters), "${AZURE_LOCATION}")

	err = Generate(root, analysis)
	require.ErrorContains(t, err, "refusing to overwrite existing")
}

func TestGenerateProjectWithoutInferredResources(t *testing.T) {
	analysis := &Analysis{
		Name: "spring-todo",
		Services: []Project{
			{Name: "spring-todo", Port: 8080, External: true},
		},
	}

	files, err := GenerateFiles(analysis)

	require.NoError(t, err)
	require.NotContains(t, generatedPaths(files), filepath.Join("infra", "postgres.bicep"))
	require.NotContains(t, generatedContent(files, "azure.yaml"), "db.postgres")
	require.NotContains(t, generatedContent(files, filepath.Join("infra", "main.bicep")), "module postgres")
	require.NotContains(
		t,
		generatedContent(files, filepath.Join("infra", "spring-todo.bicep")),
		"SPRING_DATASOURCE_URL",
	)
}

func TestGenerateSpringOrdersMicroservices(t *testing.T) {
	analysis, err := Analyze(t.Context(), microservicesFixturePath(t))
	require.NoError(t, err)

	files, err := GenerateFiles(analysis)

	require.NoError(t, err)
	azureYAML := generatedContent(files, "azure.yaml")
	require.Contains(t, azureYAML, "name: spring-orders-microservices")
	require.Contains(t, azureYAML, "project: catalog-service")
	require.Contains(t, azureYAML, "project: orders-service")
	require.NotContains(t, azureYAML, "order-contracts:")
	require.Contains(t, azureYAML, "uses:\n      - catalog-service\n      - postgres")
	require.Contains(t, azureYAML, "context: ..")

	catalogDockerfile := generatedContent(files, filepath.Join("catalog-service", "Dockerfile"))
	require.Contains(t, catalogDockerfile, "-pl catalog-service -am package")
	require.Contains(t, catalogDockerfile, "/app/catalog-service/target/*.jar")
	ordersDockerfile := generatedContent(files, filepath.Join("orders-service", "Dockerfile"))
	require.Contains(t, ordersDockerfile, "-pl orders-service -am package")

	catalogBicep := generatedContent(files, filepath.Join("infra", "catalog-service.bicep"))
	require.Contains(t, catalogBicep, "external: false")
	require.Contains(t, catalogBicep, "targetPort: 8081")
	require.Contains(t, catalogBicep, "output SERVICE_CATALOG_SERVICE_URL")
	ordersBicep := generatedContent(files, filepath.Join("infra", "orders-service.bicep"))
	require.Contains(t, ordersBicep, "external: true")
	require.Contains(t, ordersBicep, "name: 'CATALOG_SERVICE_URL'")
	require.Contains(t, ordersBicep, "name: 'SPRING_DATASOURCE_URL'")

	ordersParameters := generatedContent(files, filepath.Join("infra", "orders-service.parameters.json"))
	require.Contains(t, ordersParameters, `"catalogServiceUrl"`)
	require.Contains(t, ordersParameters, "${SERVICE_CATALOG_SERVICE_URL}")

	mainBicep := generatedContent(files, filepath.Join("infra", "main.bicep"))
	require.Contains(t, mainBicep, "module postgres 'postgres.bicep'")
	require.Contains(t, mainBicep, "SERVICE_CATALOG_SERVICE_IDENTITY_ID")
	require.Contains(t, mainBicep, "SERVICE_ORDERS_SERVICE_IDENTITY_ID")
}

func TestGenerateProjectWithFoundryModel(t *testing.T) {
	analysis, err := Analyze(t.Context(), aiFixturePath(t))
	require.NoError(t, err)

	files, err := GenerateFiles(analysis)

	require.NoError(t, err)
	require.Contains(t, generatedPaths(files), filepath.Join("infra", "foundry.bicep"))
	require.Contains(t, generatedContent(files, "azure.yaml"), "type: ai.openai.model")
	require.Contains(
		t,
		generatedContent(files, filepath.Join("infra", "main.bicep")),
		"module foundry 'foundry.bicep'",
	)
	serviceBicep := generatedContent(files, filepath.Join("infra", "spring-ai-todo-agent.bicep"))
	require.Contains(t, serviceBicep, "name: 'AZURE_INFERENCE_ENDPOINT'")
	require.Contains(t, serviceBicep, "name: 'AZURE_INFERENCE_MODEL'")
	require.Contains(t, serviceBicep, "name: 'AZURE_CLIENT_ID'")
	require.NotContains(t, serviceBicep, "foundry-credential")
	foundry := generatedContent(files, filepath.Join("infra", "foundry.bicep"))
	require.Contains(t, foundry, "kind: 'AIServices'")
	require.Contains(t, foundry, "allowProjectManagement: true")
	require.Contains(t, foundry, "disableLocalAuth: true")
	require.Contains(t, foundry, "CognitiveServices/accounts@2025-06-01")
	require.Contains(t, foundry, "name: 'gpt-4.1-mini'")
}

func TestGenerateRejectsResourceWithoutEvidence(t *testing.T) {
	analysis := &Analysis{
		Name:     "spring-todo",
		Services: []Project{{Name: "spring-todo", Port: 8080, External: true}},
		Resources: []ResourceRequirement{
			{
				Name:      "postgres",
				Type:      ResourceTypePostgreSQL,
				Consumers: []string{"spring-todo"},
			},
		},
	}

	_, err := GenerateFiles(analysis)

	require.ErrorContains(t, err, `resource "postgres" has no supporting evidence`)
}

func generatedPaths(files []GeneratedFile) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return paths
}

func generatedContent(files []GeneratedFile, path string) string {
	for _, file := range files {
		if file.Path == path {
			return string(file.Content)
		}
	}
	return ""
}

func fixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", "spring-todo")
}

func aiFixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", "spring-ai-todo-agent")
}

func microservicesFixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", "spring-orders-microservices")
}
