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
					Source: "maven-effective-pom",
					Value:  "PostgreSQL runtime dependency",
				},
			},
		},
	}, analysis.Resources)
}

func TestGenerateProject(t *testing.T) {
	root := t.TempDir()
	analysis := &Analysis{
		Services: []Project{
			{
				Name:              "spring-todo",
				ArtifactID:        "spring-todo",
				Version:           "0.0.1-SNAPSHOT",
				JavaVersion:       "21",
				Port:              8080,
				ActuatorAvailable: true,
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
		Services: []Project{
			{Name: "spring-todo", Port: 8080},
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

func TestGenerateRejectsResourceWithoutEvidence(t *testing.T) {
	analysis := &Analysis{
		Services: []Project{{Name: "spring-todo", Port: 8080}},
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
