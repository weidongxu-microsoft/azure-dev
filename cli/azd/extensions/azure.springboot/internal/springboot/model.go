// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package springboot

// Project describes the Spring Boot application properties needed to
// materialize an azd project.
type Project struct {
	Name              string
	ArtifactID        string
	Version           string
	JavaVersion       string
	Port              int
	ActuatorAvailable bool
}

// GeneratedFile is a project-relative azd file produced from a detected application.
type GeneratedFile struct {
	Path    string
	Content []byte
}
