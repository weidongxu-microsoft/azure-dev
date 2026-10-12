// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package springboot

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/azure/azure-dev/cli/azd/internal/appdetect"
	"github.com/azure/azure-dev/cli/azd/pkg/exec"
	"github.com/azure/azure-dev/cli/azd/pkg/tools/maven"
)

// Analyze inspects the application and returns a deployment-neutral model of
// its services and inferred resource requirements.
func Analyze(ctx context.Context, root string) (*Analysis, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving application path: %w", err)
	}
	rootPom, err := readPom(filepath.Join(root, "pom.xml"))
	if err != nil {
		return nil, err
	}

	projects, err := appdetect.Detect(ctx, root, appdetect.WithJava())
	if err != nil {
		return nil, fmt.Errorf("analyzing Java build model: %w", err)
	}

	analysis := &Analysis{Name: serviceName(rootPom.ArtifactID)}
	for i := range projects {
		service, err := Detect(projects[i].Path)
		if errors.Is(err, ErrNotSpringBoot) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("analyzing Maven module %s: %w", projects[i].Path, err)
		}

		relativePath, err := filepath.Rel(root, projects[i].Path)
		if err != nil {
			return nil, fmt.Errorf("resolving Maven module path: %w", err)
		}
		if relativePath == "." {
			relativePath = ""
		}
		service.Path = filepath.ToSlash(relativePath)
		service.JavaVersion = firstNonEmpty(service.JavaVersion, rootPom.Properties.JavaVersion)
		service.External = true
		analysis.Services = append(analysis.Services, *service)

		if slices.Contains(projects[i].DatabaseDeps, appdetect.DbPostgres) {
			runtimePostgres, err := detectRuntimePostgres(ctx, root, projects[i].Path)
			if err != nil {
				return nil, err
			}
			if runtimePostgres {
				analysis.Resources = append(analysis.Resources, ResourceRequirement{
					Name:      "postgres",
					Type:      ResourceTypePostgreSQL,
					Consumers: []string{service.Name},
					Evidence: []Evidence{
						{
							Type:   "dependency",
							Source: sourcePath(service.Path, "pom.xml"),
							Value:  "PostgreSQL runtime dependency in the effective Maven model",
						},
					},
				})
			}
		}

		foundryEvidence, err := detectFoundryModels(projects[i].Path)
		if err != nil {
			return nil, err
		}
		if len(foundryEvidence) > 0 {
			analysis.Resources = append(analysis.Resources, ResourceRequirement{
				Name:      "foundry-model",
				Type:      ResourceTypeFoundryModel,
				Consumers: []string{service.Name},
				Evidence:  foundryEvidence,
			})
		}
	}
	if len(analysis.Services) == 0 {
		return nil, ErrNotSpringBoot
	}
	if analysis.Name == "" {
		analysis.Name = analysis.Services[0].Name
	}

	slices.SortFunc(analysis.Services, func(a, b Project) int {
		return strings.Compare(a.Name, b.Name)
	})
	if err := detectServiceDependencies(root, analysis); err != nil {
		return nil, err
	}

	return analysis, nil
}

func detectRuntimePostgres(ctx context.Context, root, projectPath string) (bool, error) {
	cli := maven.NewCli(exec.NewCommandRunner(nil))
	cli.SetPath(projectPath, root)
	effectivePom, err := cli.EffectivePom(ctx, filepath.Join(projectPath, "pom.xml"))
	if err != nil {
		return false, fmt.Errorf("resolving PostgreSQL dependency scopes: %w", err)
	}
	var pom pomProject
	if err := xml.Unmarshal([]byte(effectivePom), &pom); err != nil {
		return false, fmt.Errorf("parsing effective Maven dependencies: %w", err)
	}
	return hasRuntimePostgres(pom.Dependencies), nil
}

func hasRuntimePostgres(dependencies []pomDependency) bool {
	for _, dependency := range dependencies {
		if dependency.Scope != "" && dependency.Scope != "compile" && dependency.Scope != "runtime" {
			continue
		}
		if (dependency.GroupID == "org.postgresql" && dependency.ArtifactID == "postgresql") ||
			(dependency.GroupID == "com.azure.spring" &&
				dependency.ArtifactID == "spring-cloud-azure-starter-jdbc-postgresql") {
			return true
		}
	}
	return false
}

func detectServiceDependencies(root string, analysis *Analysis) error {
	internalServices := map[string]struct{}{}
	for i := range analysis.Services {
		service := &analysis.Services[i]
		configRoot := filepath.Join(root, filepath.FromSlash(service.Path))
		properties, err := readApplicationConfiguration(configRoot)
		if err != nil {
			return fmt.Errorf("reading service configuration for %s: %w", service.Name, err)
		}

		for _, target := range analysis.Services {
			if target.Name == service.Name {
				continue
			}
			variable := serviceURLVariable(target.Name)
			for property, value := range properties {
				if !referencesEnvironmentVariable(value, variable) {
					continue
				}
				service.Dependencies = append(service.Dependencies, ServiceDependency{
					Service:             target.Name,
					EnvironmentVariable: variable,
					Evidence: []Evidence{
						{
							Type:   "configuration",
							Source: sourcePath(service.Path, configurationSource(configRoot, property)),
							Value:  fmt.Sprintf("%s references %s", property, variable),
						},
					},
				})
				internalServices[target.Name] = struct{}{}
				break
			}
		}
		slices.SortFunc(service.Dependencies, func(a, b ServiceDependency) int {
			return strings.Compare(a.Service, b.Service)
		})
	}

	for i := range analysis.Services {
		_, internal := internalServices[analysis.Services[i].Name]
		analysis.Services[i].External = !internal
	}
	return nil
}

func configurationSource(root string, property string) string {
	for _, name := range []string{"application.properties", "application.yaml", "application.yml"} {
		path := filepath.Join(root, "src", "main", "resources", name)
		var properties map[string]string
		var err error
		if name == "application.properties" {
			properties, err = readProperties(path)
		} else {
			properties, err = readYAMLProperties(path)
		}
		if err == nil {
			if _, found := properties[property]; found {
				return "src/main/resources/" + name
			}
		}
	}
	return "src/main/resources/application.properties"
}

func serviceURLVariable(service string) string {
	prefix := strings.TrimSuffix(service, "-service")
	return strings.ToUpper(strings.ReplaceAll(prefix, "-", "_")) + "_SERVICE_URL"
}

func sourcePath(projectPath string, path string) string {
	if projectPath == "" {
		return path
	}
	return filepath.ToSlash(filepath.Join(projectPath, filepath.FromSlash(path)))
}

func detectFoundryModels(root string) ([]Evidence, error) {
	pomPath := filepath.Join(root, "pom.xml")
	pom, err := readPom(pomPath)
	if err != nil {
		return nil, err
	}
	if !hasDependency(pom, "org.springframework.ai", "spring-ai-starter-model-openai") {
		return nil, nil
	}
	if !hasDependency(pom, "com.azure", "azure-identity") {
		return nil, nil
	}

	propertiesPath := filepath.Join(root, "src", "main", "resources", "application.properties")
	properties, err := readProperties(propertiesPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading Foundry model configuration: %w", err)
	}

	required := map[string]string{
		"spring.ai.openai.base-url":           "AZURE_INFERENCE_ENDPOINT",
		"spring.ai.openai.chat.options.model": "AZURE_INFERENCE_MODEL",
	}
	for property, variable := range required {
		if !referencesEnvironmentVariable(properties[property], variable) {
			return nil, nil
		}
	}

	return []Evidence{
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
	}, nil
}

func readProperties(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	properties := map[string]string{}
	for line := range strings.Lines(string(data)) {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if !found {
			continue
		}
		properties[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return properties, nil
}

func referencesEnvironmentVariable(value string, variable string) bool {
	return strings.Contains(value, "${"+variable+"}") ||
		strings.Contains(value, "${"+variable+":")
}
