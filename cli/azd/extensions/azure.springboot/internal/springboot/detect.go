// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package springboot

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

const defaultPort = 8080

// ErrNotSpringBoot indicates that the inspected Maven project is not a Spring Boot application.
var ErrNotSpringBoot = errors.New("not a Spring Boot application")

type pomProject struct {
	Parent       pomParent       `xml:"parent"`
	ArtifactID   string          `xml:"artifactId"`
	Version      string          `xml:"version"`
	Properties   pomProperties   `xml:"properties"`
	Dependencies []pomDependency `xml:"dependencies>dependency"`
	Build        pomBuild        `xml:"build"`
}

type pomParent struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
}

type pomProperties struct {
	JavaVersion string `xml:"java.version"`
}

type pomDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
}

type pomBuild struct {
	Plugins []pomPlugin `xml:"plugins>plugin"`
}

type pomPlugin struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
}

// Detect inspects a single-module Maven project without executing build code.
func Detect(root string) (*Project, error) {
	pomPath := filepath.Join(root, "pom.xml")
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", pomPath, err)
	}

	var pom pomProject
	if err := xml.Unmarshal(data, &pom); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", pomPath, err)
	}

	if !isSpringBoot(pom) {
		return nil, ErrNotSpringBoot
	}
	if strings.TrimSpace(pom.ArtifactID) == "" {
		return nil, errors.New("pom.xml is missing artifactId")
	}
	if !hasSpringBootPlugin(pom) {
		return nil, errors.New("pom.xml is missing the spring-boot-maven-plugin")
	}
	if !hasDependency(pom, "org.springframework.boot", "spring-boot-starter-actuator") {
		return nil, errors.New("pom.xml is missing spring-boot-starter-actuator")
	}

	version := firstNonEmpty(pom.Version, pom.Parent.Version)
	if version == "" {
		return nil, errors.New("pom.xml is missing a project version")
	}
	name := serviceName(pom.ArtifactID)
	if name == "" {
		return nil, fmt.Errorf("artifactId %q cannot be converted to an Azure service name", pom.ArtifactID)
	}

	port, err := detectPort(filepath.Join(root, "src", "main", "resources", "application.properties"))
	if err != nil {
		return nil, err
	}

	return &Project{
		Name:              name,
		ArtifactID:        pom.ArtifactID,
		Version:           version,
		JavaVersion:       strings.TrimSpace(pom.Properties.JavaVersion),
		Port:              port,
		ActuatorAvailable: hasDependency(pom, "org.springframework.boot", "spring-boot-starter-actuator"),
	}, nil
}

func isSpringBoot(pom pomProject) bool {
	return (pom.Parent.GroupID == "org.springframework.boot" &&
		pom.Parent.ArtifactID == "spring-boot-starter-parent") ||
		hasDependencyGroup(pom, "org.springframework.boot")
}

func hasSpringBootPlugin(pom pomProject) bool {
	for _, plugin := range pom.Build.Plugins {
		if plugin.ArtifactID == "spring-boot-maven-plugin" &&
			(plugin.GroupID == "" || plugin.GroupID == "org.springframework.boot") {
			return true
		}
	}

	return false
}

func hasDependencyGroup(pom pomProject, groupID string) bool {
	for _, dependency := range pom.Dependencies {
		if dependency.GroupID == groupID {
			return true
		}
	}

	return false
}

func hasDependency(pom pomProject, groupID string, artifactID string) bool {
	for _, dependency := range pom.Dependencies {
		if dependency.GroupID == groupID && dependency.ArtifactID == artifactID {
			return true
		}
	}

	return false
}

func detectPort(path string) (int, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaultPort, nil
	}
	if err != nil {
		return 0, fmt.Errorf("reading %s: %w", path, err)
	}

	for line := range strings.Lines(string(data)) {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if !found || strings.TrimSpace(key) != "server.port" {
			continue
		}

		port, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || port < 1 || port > 65535 {
			return 0, fmt.Errorf("application.properties has invalid server.port %q", strings.TrimSpace(value))
		}

		return port, nil
	}

	return defaultPort, nil
}

func serviceName(artifactID string) string {
	var name strings.Builder
	separator := false
	for _, r := range strings.ToLower(strings.TrimSpace(artifactID)) {
		if r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			if separator && name.Len() > 0 {
				name.WriteByte('-')
			}
			name.WriteRune(r)
			separator = false
			continue
		}

		separator = true
	}

	return strings.Trim(name.String(), "-")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}

	return ""
}
