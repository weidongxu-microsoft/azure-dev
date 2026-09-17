// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azdext

import (
	"context"

	v1beta "github.com/azure/azure-dev/cli/azd/pkg/azdext/contracts/v1beta"
	"github.com/azure/azure-dev/cli/azd/pkg/grpcbroker"
	"google.golang.org/protobuf/proto"
)

// InitEnvelope provides broker operations for init provider messages.
type InitEnvelope struct{}

var _ grpcbroker.MessageEnvelope[v1beta.InitMessage] = (*InitEnvelope)(nil)

func (e *InitEnvelope) GetRequestId(_ context.Context, message *v1beta.InitMessage) string {
	return message.GetRequestId()
}

func (e *InitEnvelope) SetRequestId(_ context.Context, message *v1beta.InitMessage, id string) {
	message.RequestId = id
}

func (e *InitEnvelope) GetError(message *v1beta.InitMessage) error {
	if message.GetError() == nil {
		return nil
	}
	stable := new(ExtensionError)
	data, err := proto.Marshal(message.GetError())
	if err != nil {
		return err
	}
	if err := proto.Unmarshal(data, stable); err != nil {
		return err
	}
	return UnwrapError(stable)
}

func (e *InitEnvelope) SetError(message *v1beta.InitMessage, source error) {
	stable := WrapError(source)
	data, err := proto.Marshal(stable)
	if err != nil {
		return
	}
	target := new(v1beta.ExtensionError)
	if err := proto.Unmarshal(data, target); err == nil {
		message.Error = target
	}
}

func (e *InitEnvelope) GetInnerMessage(message *v1beta.InitMessage) any {
	switch value := message.MessageType.(type) {
	case *v1beta.InitMessage_RegisterInitProviderRequest:
		return value.RegisterInitProviderRequest
	case *v1beta.InitMessage_RegisterInitProviderResponse:
		return value.RegisterInitProviderResponse
	case *v1beta.InitMessage_DetectRequest:
		return value.DetectRequest
	case *v1beta.InitMessage_DetectResponse:
		return value.DetectResponse
	default:
		return nil
	}
}

func (e *InitEnvelope) IsProgressMessage(*v1beta.InitMessage) bool {
	return false
}

func (e *InitEnvelope) GetProgressMessage(*v1beta.InitMessage) string {
	return ""
}

func (e *InitEnvelope) CreateProgressMessage(string, string) *v1beta.InitMessage {
	return nil
}
