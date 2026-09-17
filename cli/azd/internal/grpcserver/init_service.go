// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	v1beta "github.com/azure/azure-dev/cli/azd/pkg/azdext/contracts/v1beta"
	"github.com/azure/azure-dev/cli/azd/pkg/extensions"
	"github.com/azure/azure-dev/cli/azd/pkg/grpcbroker"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type initProviderRegistration struct {
	extension *extensions.Extension
	broker    *grpcbroker.MessageBroker[v1beta.InitMessage]
}

// InitService implements the preview init provider service.
type InitService struct {
	v1beta.UnimplementedInitServiceServer
	extensionManager *extensions.Manager
	providers        map[string]initProviderRegistration
	mu               sync.RWMutex
}

// NewInitService creates an init provider service.
func NewInitService(extensionManager *extensions.Manager) *InitService {
	return &InitService{
		extensionManager: extensionManager,
		providers:        make(map[string]initProviderRegistration),
	}
}

// Stream handles init provider registration and requests.
func (s *InitService) Stream(stream v1beta.InitService_StreamServer) error {
	ctx := stream.Context()
	claims, err := extensions.GetClaimsFromContext(ctx)
	if err != nil {
		return fmt.Errorf("getting extension claims: %w", err)
	}
	extension, err := s.extensionManager.GetInstalled(extensions.FilterOptions{Id: claims.Subject})
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "getting installed extension: %s", err)
	}
	if !extension.HasCapability(extensions.InitProviderCapability) {
		return status.Error(codes.PermissionDenied, "extension does not support init-provider capability")
	}

	broker := grpcbroker.NewMessageBroker(stream, new(azdext.InitEnvelope), extension.Id, log.Default())
	var registered []string
	var registeredMu sync.Mutex
	if err := broker.On(func(
		_ context.Context,
		request *v1beta.RegisterInitProviderRequest,
	) (*v1beta.InitMessage, error) {
		name := strings.TrimSpace(request.GetName())
		if name == "" {
			return nil, status.Error(codes.InvalidArgument, "init provider name cannot be empty")
		}

		s.mu.Lock()
		defer s.mu.Unlock()
		if _, exists := s.providers[name]; exists {
			return nil, status.Errorf(codes.AlreadyExists, "init provider %q is already registered", name)
		}
		s.providers[name] = initProviderRegistration{extension: extension, broker: broker}

		registeredMu.Lock()
		registered = append(registered, name)
		registeredMu.Unlock()
		return &v1beta.InitMessage{
			MessageType: &v1beta.InitMessage_RegisterInitProviderResponse{
				RegisterInitProviderResponse: &v1beta.RegisterInitProviderResponse{},
			},
		}, nil
	}); err != nil {
		return fmt.Errorf("registering init provider handler: %w", err)
	}

	if err := broker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("init provider broker: %w", err)
	}

	registeredMu.Lock()
	names := slices.Clone(registered)
	registeredMu.Unlock()
	s.mu.Lock()
	for _, name := range names {
		delete(s.providers, name)
	}
	s.mu.Unlock()
	return nil
}

// Detect asks every registered provider to inspect the project path.
func (s *InitService) Detect(ctx context.Context, projectPath string) (*v1beta.InitProject, string, error) {
	s.mu.RLock()
	providers := maps.Clone(s.providers)
	s.mu.RUnlock()

	var matched *v1beta.InitProject
	var matchedProvider string
	for _, name := range slices.Sorted(maps.Keys(providers)) {
		registration := providers[name]
		response, err := registration.broker.SendAndWait(ctx, &v1beta.InitMessage{
			RequestId: uuid.NewString(),
			MessageType: &v1beta.InitMessage_DetectRequest{
				DetectRequest: &v1beta.InitDetectRequest{
					ProjectPath:  projectPath,
					ProviderName: name,
				},
			},
		})
		if err != nil {
			return nil, "", fmt.Errorf("running init provider %q: %w", name, err)
		}
		detection := response.GetDetectResponse()
		if detection == nil || !detection.GetMatched() {
			continue
		}
		if matched != nil {
			return nil, "", fmt.Errorf(
				"multiple init providers matched the project: %q and %q",
				matchedProvider,
				name,
			)
		}
		if detection.GetProject() == nil {
			return nil, "", fmt.Errorf("init provider %q returned a match without a project", name)
		}
		matched = detection.GetProject()
		matchedProvider = name
	}

	return matched, matchedProvider, nil
}
