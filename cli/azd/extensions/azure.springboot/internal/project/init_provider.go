// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/azure/azure-dev/cli/azd/extensions/azure.springboot/internal/springboot"
	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
)

// InitProvider detects and materializes supported Spring Boot applications.
type InitProvider struct{}

// NewInitProvider creates the Spring Boot init provider.
func NewInitProvider() azdext.InitProvider {
	return new(InitProvider)
}

// Detect inspects a Maven project without executing application code.
func (p *InitProvider) Detect(_ context.Context, projectPath string) (*azdext.InitResult, error) {
	if _, err := os.Stat(filepath.Join(projectPath, "pom.xml")); errors.Is(err, os.ErrNotExist) {
		return &azdext.InitResult{}, nil
	} else if err != nil {
		return nil, err
	}

	detected, err := springboot.Detect(projectPath)
	if errors.Is(err, springboot.ErrNotSpringBoot) {
		return &azdext.InitResult{}, nil
	}
	if err != nil {
		return nil, err
	}

	generated, err := springboot.GenerateFiles(detected)
	if err != nil {
		return nil, err
	}
	files := make([]azdext.InitFile, 0, len(generated))
	for _, file := range generated {
		files = append(files, azdext.InitFile{Path: file.Path, Content: file.Content})
	}
	return &azdext.InitResult{
		Matched:     true,
		Name:        detected.Name,
		Description: "Spring Boot application",
		Files:       files,
	}, nil
}
