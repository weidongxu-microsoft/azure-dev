// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/azure/azure-dev/cli/azd/internal"
	"github.com/azure/azure-dev/cli/azd/internal/appdetect"
	"github.com/azure/azure-dev/cli/azd/pkg/environment/azdcontext"
	"github.com/azure/azure-dev/cli/azd/pkg/project"
	"github.com/stretchr/testify/require"
)

func TestInitializer_materializeInitProject(t *testing.T) {
	t.Parallel()

	validProject := []byte("name: test\n")
	tests := []struct {
		name        string
		files       []InitFile
		setup       func(t *testing.T, projectDir string)
		wantErr     string
		wantCreated []string
	}{
		{
			name: "success",
			files: []InitFile{
				{Path: "infra/main.bicep", Content: []byte("targetScope = 'subscription'\n")},
				{Path: "azure.yaml", Content: validProject},
			},
			wantCreated: []string{"azure.yaml", "infra/main.bicep"},
		},
		{
			name:    "unsafe parent path",
			files:   []InitFile{{Path: "../outside.txt"}, {Path: "azure.yaml", Content: validProject}},
			wantErr: "unsafe file path",
		},
		{
			name: "duplicate path",
			files: []InitFile{
				{Path: "infra/main.bicep"},
				{Path: filepath.Join("infra", ".", "main.bicep")},
				{Path: "azure.yaml", Content: validProject},
			},
			wantErr: "duplicate file path",
		},
		{
			name:    "missing project file",
			files:   []InitFile{{Path: "infra/main.bicep"}},
			wantErr: "did not return azure.yaml",
		},
		{
			name:  "existing file",
			files: []InitFile{{Path: "azure.yaml", Content: validProject}},
			setup: func(t *testing.T, projectDir string) {
				require.NoError(t, os.WriteFile(filepath.Join(projectDir, "azure.yaml"), []byte("existing"), 0600))
			},
			wantErr: "would overwrite existing file",
		},
		{
			name: "invalid project rolls back",
			files: []InitFile{
				{Path: "infra/main.bicep", Content: []byte("content")},
				{Path: "azure.yaml", Content: []byte("invalid: [")},
			},
			wantErr: "validating generated azure.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			projectDir := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, projectDir)
			}

			initializer := new(Initializer)
			err := initializer.materializeInitProject(
				t.Context(),
				azdcontext.NewAzdContextWithDirectory(projectDir),
				&InitProject{Provider: "test", Files: tt.files},
			)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				if tt.name == "invalid project rolls back" {
					require.NoFileExists(t, filepath.Join(projectDir, "azure.yaml"))
					require.NoFileExists(t, filepath.Join(projectDir, "infra", "main.bicep"))
				}
				return
			}

			require.NoError(t, err)
			for _, path := range tt.wantCreated {
				require.FileExists(t, filepath.Join(projectDir, path))
			}
		})
	}
}

func TestInitializer_prjConfigFromDetect(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		detect       detectConfirm
		interactions []string
		want         project.ProjectConfig
	}{
		{
			name: "api",
			detect: detectConfirm{
				Services: []appdetect.Project{
					{
						Language: appdetect.DotNet,
						Path:     "dotnet",
					},
				},
			},
			interactions: []string{},
			want: project.ProjectConfig{
				Services: map[string]*project.ServiceConfig{
					"dotnet": {
						Language:     project.ServiceLanguageDotNet,
						RelativePath: "dotnet",
						Host:         project.ContainerAppTarget,
					},
				},
				Resources: map[string]*project.ResourceConfig{
					"dotnet": {
						Type: project.ResourceTypeHostContainerApp,
						Name: "dotnet",
						Props: project.ContainerAppProps{
							Port: 8080,
						},
					},
				},
			},
		},
		{
			name: "web",
			detect: detectConfirm{
				Services: []appdetect.Project{
					{
						Language: appdetect.JavaScript,
						Path:     "js",
						Dependencies: []appdetect.Dependency{
							appdetect.JsReact,
						},
					},
				},
			},
			interactions: []string{},
			want: project.ProjectConfig{
				Services: map[string]*project.ServiceConfig{
					"js": {
						Language:     project.ServiceLanguageJavaScript,
						Host:         project.ContainerAppTarget,
						RelativePath: "js",
						OutputPath:   "build",
					},
				},
				Resources: map[string]*project.ResourceConfig{
					"js": {
						Type: project.ResourceTypeHostContainerApp,
						Name: "js",
						Props: project.ContainerAppProps{
							Port: 80,
						},
					},
				},
			},
		},
		{
			name: "api with docker",
			detect: detectConfirm{
				Services: []appdetect.Project{
					{
						Language: appdetect.DotNet,
						Path:     "dotnet",
						Docker:   &appdetect.Docker{Path: "Dockerfile"},
					},
				},
			},
			interactions: []string{
				// prompt for port -- hit multiple validation cases
				"notAnInteger",
				"-2",
				"65536",
				"1234",
			},
			want: project.ProjectConfig{
				Services: map[string]*project.ServiceConfig{
					"dotnet": {
						Language:     project.ServiceLanguageDotNet,
						Host:         project.ContainerAppTarget,
						RelativePath: "dotnet",
						Docker: project.DockerProjectOptions{
							Path: "Dockerfile",
						},
					},
				},
				Resources: map[string]*project.ResourceConfig{
					"dotnet": {
						Type: project.ResourceTypeHostContainerApp,
						Name: "dotnet",
						Props: project.ContainerAppProps{
							Port: 1234,
						},
					},
				},
			},
		},
		{
			name: "api and web",
			detect: detectConfirm{
				Services: []appdetect.Project{
					{
						Language: appdetect.Python,
						Path:     "py",
					},
					{
						Language: appdetect.JavaScript,
						Path:     "js",
						Dependencies: []appdetect.Dependency{
							appdetect.JsReact,
						},
					},
				},
			},
			interactions: []string{},
			want: project.ProjectConfig{
				Services: map[string]*project.ServiceConfig{
					"py": {
						Language:     project.ServiceLanguagePython,
						Host:         project.ContainerAppTarget,
						RelativePath: "py",
					},
					"js": {
						Language:     project.ServiceLanguageJavaScript,
						Host:         project.ContainerAppTarget,
						RelativePath: "js",
						OutputPath:   "build",
					},
				},

				Resources: map[string]*project.ResourceConfig{
					"py": {
						Type: project.ResourceTypeHostContainerApp,
						Name: "py",
						Props: project.ContainerAppProps{
							Port: 80,
						},
					},
					"js": {
						Type: project.ResourceTypeHostContainerApp,
						Name: "js",
						Uses: []string{"py"},
						Props: project.ContainerAppProps{
							Port: 80,
						},
					},
				},
			},
		},
		{
			name: "full",
			detect: detectConfirm{
				Services: []appdetect.Project{
					{
						Language: appdetect.Python,
						Path:     "py",
						DatabaseDeps: []appdetect.DatabaseDep{
							appdetect.DbPostgres,
							appdetect.DbMongo,
							appdetect.DbRedis,
						},
					},
					{
						Language: appdetect.JavaScript,
						Path:     "js",
						Dependencies: []appdetect.Dependency{
							appdetect.JsReact,
						},
					},
				},
				Databases: map[appdetect.DatabaseDep]EntryKind{
					appdetect.DbPostgres: EntryKindDetected,
					appdetect.DbMongo:    EntryKindDetected,
					appdetect.DbRedis:    EntryKindDetected,
				},
			},
			interactions: []string{
				// prompt for db -- hit multiple validation cases
				"my app db",
				"N",
				"my$special$db",
				"N",
				"mongodb", // fill in db name
				// prompt for db -- hit multiple validation cases
				"my$special$db",
				"N",
				"postgres", // fill in db name
			},
			want: project.ProjectConfig{
				Services: map[string]*project.ServiceConfig{
					"py": {
						Language:     project.ServiceLanguagePython,
						Host:         project.ContainerAppTarget,
						RelativePath: "py",
					},
					"js": {
						Language:     project.ServiceLanguageJavaScript,
						Host:         project.ContainerAppTarget,
						RelativePath: "js",
						OutputPath:   "build",
					},
				},
				Resources: map[string]*project.ResourceConfig{
					"redis": {
						Type: project.ResourceTypeDbRedis,
						Name: "redis",
					},
					"mongodb": {
						Type: project.ResourceTypeDbMongo,
						Name: "mongodb",
					},
					"postgres": {
						Type: project.ResourceTypeDbPostgres,
						Name: "postgres",
					},
					"py": {
						Type: project.ResourceTypeHostContainerApp,
						Name: "py",
						Uses: []string{"postgres", "mongodb", "redis"},
						Props: project.ContainerAppProps{
							Port: 80,
						},
					},
					"js": {
						Type: project.ResourceTypeHostContainerApp,
						Name: "js",
						Uses: []string{"py"},
						Props: project.ContainerAppProps{
							Port: 80,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Initializer{
				console: newCapturedTestConsole(t, tt.interactions),
			}

			dir := t.TempDir()

			prjName := dir
			tt.want.Name = filepath.Base(prjName)
			tt.want.Metadata = &project.ProjectMetadata{
				Template: fmt.Sprintf("%s@%s", InitGenTemplateId, internal.VersionInfo().Version),
			}

			if tt.want.Resources == nil {
				tt.want.Resources = map[string]*project.ResourceConfig{}
			}

			for k, svc := range tt.want.Services {
				svc.Name = k
			}

			// Convert relative to absolute paths
			for idx, svc := range tt.detect.Services {
				tt.detect.Services[idx].Path = filepath.Join(dir, svc.Path)
				if tt.detect.Services[idx].Docker != nil {
					tt.detect.Services[idx].Docker.Path = filepath.Join(dir, svc.Path, svc.Docker.Path)
				}
			}

			spec, err := i.prjConfigFromDetect(
				t.Context(),
				dir,
				tt.detect)

			require.NoError(t, err)
			require.Equal(t, tt.want, spec)
		})
	}
}
