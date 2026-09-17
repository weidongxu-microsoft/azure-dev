// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package springboot

import (
	"os"
	"path/filepath"
	"testing"

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

func TestGenerateProject(t *testing.T) {
	root := t.TempDir()
	project := &Project{
		Name:              "spring-todo",
		ArtifactID:        "spring-todo",
		Version:           "0.0.1-SNAPSHOT",
		JavaVersion:       "21",
		Port:              8080,
		ActuatorAvailable: true,
	}

	require.NoError(t, Generate(root, project))

	azureYAML, err := os.ReadFile(filepath.Join(root, "azure.yaml"))
	require.NoError(t, err)
	require.Contains(t, string(azureYAML), "azure.springboot: \">=0.1.0\"")
	require.Contains(t, string(azureYAML), "language: java")
	require.Contains(t, string(azureYAML), "remoteBuild: true")

	dockerfile, err := os.ReadFile(filepath.Join(root, "Dockerfile"))
	require.NoError(t, err)
	require.Contains(t, string(dockerfile), "FROM maven:3.9.11-eclipse-temurin-21 AS build")
	require.Contains(t, string(dockerfile), "EXPOSE 8080")

	serviceBicep, err := os.ReadFile(filepath.Join(root, "infra", "spring-todo.bicep"))
	require.NoError(t, err)
	require.Contains(t, string(serviceBicep), "targetPort: 8080")
	require.Contains(t, string(serviceBicep), "path: '/actuator/health/readiness'")

	serviceParameters, err := os.ReadFile(filepath.Join(root, "infra", "spring-todo.parameters.json"))
	require.NoError(t, err)
	require.Contains(t, string(serviceParameters), "${SERVICE_SPRING_TODO_IMAGE_NAME}")

	mainParameters, err := os.ReadFile(filepath.Join(root, "infra", "main.parameters.json"))
	require.NoError(t, err)
	require.Contains(t, string(mainParameters), "${AZURE_ENV_NAME}")
	require.Contains(t, string(mainParameters), "${AZURE_LOCATION}")

	err = Generate(root, project)
	require.ErrorContains(t, err, "refusing to overwrite existing")
}

func fixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", "spring-todo")
}
