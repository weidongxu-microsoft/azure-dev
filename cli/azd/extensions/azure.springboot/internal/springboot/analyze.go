// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package springboot

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/azure/azure-dev/cli/azd/internal/appdetect"
)

// Analyze inspects the application and returns a deployment-neutral model of
// its services and inferred resource requirements.
func Analyze(ctx context.Context, root string) (*Analysis, error) {
	service, err := Detect(root)
	if err != nil {
		return nil, err
	}

	projects, err := appdetect.Detect(ctx, root, appdetect.WithJava())
	if err != nil {
		return nil, fmt.Errorf("analyzing Java build model: %w", err)
	}

	rootPath, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving application path: %w", err)
	}

	var javaProject *appdetect.Project
	for i := range projects {
		projectPath, err := filepath.Abs(projects[i].Path)
		if err != nil {
			return nil, fmt.Errorf("resolving detected project path: %w", err)
		}
		if filepath.Clean(projectPath) == filepath.Clean(rootPath) {
			javaProject = &projects[i]
			break
		}
	}
	if javaProject == nil {
		return nil, errorsForUnsupportedLayout(projects)
	}

	analysis := &Analysis{Services: []Project{*service}}
	if slices.Contains(javaProject.DatabaseDeps, appdetect.DbPostgres) {
		analysis.Resources = append(analysis.Resources, ResourceRequirement{
			Name:      "postgres",
			Type:      ResourceTypePostgreSQL,
			Consumers: []string{service.Name},
			Evidence: []Evidence{
				{
					Type:   "dependency",
					Source: "maven-effective-pom",
					Value:  "PostgreSQL runtime dependency",
				},
			},
		})
	}

	return analysis, nil
}

func errorsForUnsupportedLayout(projects []appdetect.Project) error {
	if len(projects) == 0 {
		return fmt.Errorf("Maven did not identify a deployable Java project at the repository root")
	}
	return fmt.Errorf(
		"the Spring Boot project is not a deployable root Maven module; detected %d nested Java modules",
		len(projects),
	)
}
