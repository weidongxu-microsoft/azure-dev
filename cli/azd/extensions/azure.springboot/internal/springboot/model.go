// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package springboot

// Analysis describes the deployable services and resource requirements inferred
// from an application repository.
type Analysis struct {
	Services  []Project
	Resources []ResourceRequirement
}

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

// ResourceType identifies an Azure resource capability required by an analyzed service.
type ResourceType string

const (
	// ResourceTypePostgreSQL represents a PostgreSQL database requirement.
	ResourceTypePostgreSQL ResourceType = "postgresql"
)

// ResourceRequirement describes an inferred resource and the evidence that produced it.
type ResourceRequirement struct {
	Name      string
	Type      ResourceType
	Consumers []string
	Evidence  []Evidence
}

// Evidence records the source of an inferred application requirement.
type Evidence struct {
	Type   string
	Source string
	Value  string
}

// GeneratedFile is a project-relative azd file produced from a detected application.
type GeneratedFile struct {
	Path    string
	Content []byte
}
