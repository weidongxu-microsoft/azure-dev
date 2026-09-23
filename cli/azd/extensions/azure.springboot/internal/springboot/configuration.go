// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package springboot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func readApplicationConfiguration(root string) (map[string]string, error) {
	resources := filepath.Join(root, "src", "main", "resources")
	properties := map[string]string{}
	for _, name := range []string{"application.yml", "application.yaml", "application.properties"} {
		path := filepath.Join(resources, name)
		var values map[string]string
		var err error
		if name == "application.properties" {
			values, err = readProperties(path)
		} else {
			values, err = readYAMLProperties(path)
		}
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		for key, value := range values {
			properties[key] = value
		}
	}
	return properties, nil
}

func readYAMLProperties(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	properties := map[string]string{}
	if len(document.Content) > 0 {
		flattenYAML(document.Content[0], "", properties)
	}
	return properties, nil
}

func flattenYAML(node *yaml.Node, prefix string, properties map[string]string) {
	if node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if key.Kind != yaml.ScalarNode {
			continue
		}
		name := key.Value
		if prefix != "" {
			name = prefix + "." + name
		}
		if value.Kind == yaml.MappingNode {
			flattenYAML(value, name, properties)
		} else if value.Kind == yaml.ScalarNode {
			properties[name] = strings.TrimSpace(value.Value)
		}
	}
}

func detectExternalOpenAI(properties map[string]string) (*ExternalOpenAI, error) {
	endpoint := properties["azure.openai.endpoint"]
	key := properties["azure.openai.api-key"]
	if !strings.Contains(endpoint, "${") && !strings.Contains(key, "${") {
		return nil, nil
	}
	if !referencesEnvironmentVariable(endpoint, "AZURE_OPENAI_ENDPOINT") ||
		!referencesEnvironmentVariable(key, "AZURE_OPENAI_API_KEY") {
		return nil, errors.New(
			"Azure OpenAI configuration requires azure.openai.endpoint and azure.openai.api-key " +
				"placeholders for AZURE_OPENAI_ENDPOINT and AZURE_OPENAI_API_KEY",
		)
	}
	version := properties["azure.openai.api-version"]
	if version != "" && strings.Contains(version, "${") &&
		!referencesEnvironmentVariable(version, "AZURE_OPENAI_API_VERSION") {
		return nil, errors.New("azure.openai.api-version must reference AZURE_OPENAI_API_VERSION")
	}
	openAI := &ExternalOpenAI{APIVersion: referencesEnvironmentVariable(version, "AZURE_OPENAI_API_VERSION")}
	if openAI.APIVersion && strings.HasPrefix(version, "${AZURE_OPENAI_API_VERSION:") && strings.HasSuffix(version, "}") {
		defaultValue := strings.TrimSuffix(strings.TrimPrefix(version, "${AZURE_OPENAI_API_VERSION:"), "}")
		if defaultValue != "" {
			for _, character := range defaultValue {
				if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-", character) {
					return nil, fmt.Errorf("unsupported Azure OpenAI API version default %q", defaultValue)
				}
			}
			openAI.APIVersionDefault = defaultValue
		}
	}
	return openAI, nil
}
