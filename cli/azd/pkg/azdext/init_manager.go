// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azdext

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	v1beta "github.com/azure/azure-dev/cli/azd/pkg/azdext/contracts/v1beta"
	"github.com/azure/azure-dev/cli/azd/pkg/grpcbroker"
	"github.com/google/uuid"
)

// InitProvider detects an application repository and returns the files needed
// to materialize an azd project.
type InitProvider interface {
	Detect(ctx context.Context, projectPath string) (*InitResult, error)
}

// InitProviderFactory creates an init provider.
type InitProviderFactory func() InitProvider

// InitResult is the result of running an init provider.
type InitResult struct {
	Matched     bool
	Name        string
	Description string
	Files       []InitFile
}

// InitFile is a project-relative file produced by an init provider.
type InitFile struct {
	Path    string
	Content []byte
}

// InitManager manages init providers registered by an extension.
type InitManager struct {
	extensionID string
	client      *AzdClient
	broker      *grpcbroker.MessageBroker[v1beta.InitMessage]
	logger      *log.Logger
	factories   map[string]InitProviderFactory
	mu          sync.RWMutex
}

// NewInitManager creates an init provider manager.
func NewInitManager(extensionID string, client *AzdClient, logger *log.Logger) *InitManager {
	return &InitManager{
		extensionID: extensionID,
		client:      client,
		logger:      logger,
		factories:   make(map[string]InitProviderFactory),
	}
}

// Close closes the init provider stream.
func (m *InitManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.broker != nil {
		m.broker.Close()
		m.broker = nil
	}
	clear(m.factories)
	return nil
}

func (m *InitManager) ensureStream(ctx context.Context) error {
	m.mu.RLock()
	if m.broker != nil {
		m.mu.RUnlock()
		return nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.broker != nil {
		return nil
	}

	stream, err := m.client.Init().Stream(ctx)
	if err != nil {
		return fmt.Errorf("creating init provider stream: %w", err)
	}
	m.broker = grpcbroker.NewMessageBroker(stream, new(InitEnvelope), m.extensionID, m.logger)
	if err := m.broker.On(m.onDetect); err != nil {
		m.broker.Close()
		m.broker = nil
		return fmt.Errorf("registering init provider handler: %w", err)
	}
	return nil
}

// Register registers an init provider with azd.
func (m *InitManager) Register(ctx context.Context, name string, factory InitProviderFactory) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("init provider name cannot be empty")
	}
	if factory == nil {
		return fmt.Errorf("init provider factory for %q cannot be nil", name)
	}
	if err := m.ensureStream(ctx); err != nil {
		return err
	}

	m.mu.Lock()
	if _, exists := m.factories[name]; exists {
		m.mu.Unlock()
		return fmt.Errorf("init provider %q already registered", name)
	}
	m.factories[name] = factory
	m.mu.Unlock()

	request := &v1beta.InitMessage{
		RequestId: uuid.NewString(),
		MessageType: &v1beta.InitMessage_RegisterInitProviderRequest{
			RegisterInitProviderRequest: &v1beta.RegisterInitProviderRequest{Name: name},
		},
	}
	response, err := m.broker.SendAndWait(ctx, request)
	if err != nil {
		m.mu.Lock()
		delete(m.factories, name)
		m.mu.Unlock()
		return fmt.Errorf("registering init provider %q: %w", name, err)
	}
	if response.GetRegisterInitProviderResponse() == nil {
		return fmt.Errorf("registering init provider %q: unexpected response %T", name, response.GetMessageType())
	}
	return nil
}

// Receive runs the init provider message dispatcher.
func (m *InitManager) Receive(ctx context.Context) error {
	if err := m.ensureStream(ctx); err != nil {
		return err
	}
	return m.broker.Run(ctx)
}

// Ready waits until the init provider message dispatcher is ready.
func (m *InitManager) Ready(ctx context.Context) error {
	if err := m.ensureStream(ctx); err != nil {
		return err
	}
	return m.broker.Ready(ctx)
}

func (m *InitManager) onDetect(ctx context.Context, request *v1beta.InitDetectRequest) (*v1beta.InitMessage, error) {
	m.mu.RLock()
	factory := m.factories[request.GetProviderName()]
	m.mu.RUnlock()
	if factory == nil {
		return nil, fmt.Errorf("init provider %q is not registered", request.GetProviderName())
	}

	result, err := factory().Detect(ctx, request.GetProjectPath())
	if err != nil {
		return nil, err
	}
	response := &v1beta.InitDetectResponse{}
	if result != nil {
		response.Matched = result.Matched
		if result.Matched {
			response.Project = &v1beta.InitProject{
				Name:        result.Name,
				Description: result.Description,
				Files:       make([]*v1beta.InitFile, 0, len(result.Files)),
			}
			for _, file := range result.Files {
				response.Project.Files = append(response.Project.Files, &v1beta.InitFile{
					Path:    file.Path,
					Content: file.Content,
				})
			}
		}
	}

	return &v1beta.InitMessage{
		MessageType: &v1beta.InitMessage_DetectResponse{DetectResponse: response},
	}, nil
}
