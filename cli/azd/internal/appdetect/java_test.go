// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package appdetect

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectDependenciesIgnoresNonRuntimeScopes(t *testing.T) {
	mavenProject := &mavenProject{
		Dependencies: []dependency{
			{GroupId: "org.postgresql", ArtifactId: "postgresql", Scope: "test"},
			{GroupId: "com.mysql", ArtifactId: "mysql-connector-j", Scope: "provided"},
			{
				GroupId:    "org.springframework.boot",
				ArtifactId: "spring-boot-starter-data-redis",
				Scope:      "runtime",
			},
		},
	}

	project, err := detectDependencies(mavenProject, &Project{})

	require.NoError(t, err)
	require.Equal(t, []DatabaseDep{DbRedis}, project.DatabaseDeps)
}
