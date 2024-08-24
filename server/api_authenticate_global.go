package server

import (
	"context"

	"github.com/u2u-labs/go-layerg-common/api"
	"google.golang.org/grpc"
)

// AuthenticatorServiceClient is the client API for AuthenticatorService service.
type AuthenticatorServiceClient interface {
	AuthenticateEmail(ctx context.Context, in *api.AuthenticateEmailRequest, opts ...grpc.CallOption) (*api.Session, error)
	AuthenticateGoogle(ctx context.Context, in *api.AuthenticateGoogleRequest, opts ...grpc.CallOption) (*api.Session, error)
	// Other methods based on your .proto definition
}

type authenticatorServiceClient struct {
	cc grpc.ClientConnInterface
}

// NewAuthenticatorServiceClient creates a new client for the AuthenticatorService.
func NewAuthenticatorServiceClient(cc grpc.ClientConnInterface) AuthenticatorServiceClient {
	return &authenticatorServiceClient{cc}
}

func (c *authenticatorServiceClient) AuthenticateEmail(ctx context.Context, in *api.AuthenticateEmailRequest, opts ...grpc.CallOption) (*api.Session, error) {
	out := new(api.Session)
	err := c.cc.Invoke(ctx, "/layerg.api.LayerG/AuthenticateEmail", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authenticatorServiceClient) AuthenticateGoogle(ctx context.Context, in *api.AuthenticateGoogleRequest, opts ...grpc.CallOption) (*api.Session, error) {
	out := new(api.Session)
	err := c.cc.Invoke(ctx, "/layerg.api.LayerG/AuthenticateGoogle", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
