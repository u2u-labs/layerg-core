package server

import (
	"context"

	"github.com/u2u-labs/go-layerg-common/api"
	"google.golang.org/grpc"
)

type AuthenticatorServiceClient interface {
	AuthenticateEmail(ctx context.Context, in *api.AuthenticateEmailRequest, opts ...grpc.CallOption) (*api.Session, error)
	AuthenticateGoogle(ctx context.Context, in *api.AuthenticateGoogleRequest, opts ...grpc.CallOption) (*api.Session, error)
	AuthenticateApple(ctx context.Context, in *api.AuthenticateAppleRequest, opts ...grpc.CallOption) (*api.Session, error)
	AuthenticateCustom(ctx context.Context, in *api.AuthenticateCustomRequest, opts ...grpc.CallOption) (*api.Session, error)
	AuthenticateDevice(ctx context.Context, in *api.AuthenticateDeviceRequest, opts ...grpc.CallOption) (*api.Session, error)
	AuthenticateFacebook(ctx context.Context, in *api.AuthenticateFacebookRequest, opts ...grpc.CallOption) (*api.Session, error)
	AuthenticateFacebookInstantGame(ctx context.Context, in *api.AuthenticateFacebookInstantGameRequest, opts ...grpc.CallOption) (*api.Session, error)
	AuthenticateGameCenter(ctx context.Context, in *api.AuthenticateGameCenterRequest, opts ...grpc.CallOption) (*api.Session, error)
	AuthenticateSteam(ctx context.Context, in *api.AuthenticateSteamRequest, opts ...grpc.CallOption) (*api.Session, error)
	AuthenticateEvm(ctx context.Context, in *api.AuthenticateEvmRequest, opts ...grpc.CallOption) (*api.Session, error)
	AuthenticateTelegram(ctx context.Context, in *api.AuthenticateTelegramRequest, opts ...grpc.CallOption) (*api.Session, error)
	SessionRefresh(ctx context.Context, in *api.SessionRefreshRequest, opts ...grpc.CallOption) (*api.Session, error)
}

type authenticatorServiceClient struct {
	cc grpc.ClientConnInterface
}

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
func (c *authenticatorServiceClient) AuthenticateApple(ctx context.Context, in *api.AuthenticateAppleRequest, opts ...grpc.CallOption) (*api.Session, error) {
	out := new(api.Session)
	err := c.cc.Invoke(ctx, "/layerg.api.LayerG/AuthenticateApple", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *authenticatorServiceClient) AuthenticateCustom(ctx context.Context, in *api.AuthenticateCustomRequest, opts ...grpc.CallOption) (*api.Session, error) {
	out := new(api.Session)
	err := c.cc.Invoke(ctx, "/layerg.api.LayerG/AuthenticateCustom", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *authenticatorServiceClient) AuthenticateDevice(ctx context.Context, in *api.AuthenticateDeviceRequest, opts ...grpc.CallOption) (*api.Session, error) {
	out := new(api.Session)
	err := c.cc.Invoke(ctx, "/layerg.api.LayerG/AuthenticateDevice", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *authenticatorServiceClient) AuthenticateFacebookInstantGame(ctx context.Context, in *api.AuthenticateFacebookInstantGameRequest, opts ...grpc.CallOption) (*api.Session, error) {
	out := new(api.Session)
	err := c.cc.Invoke(ctx, "/layerg.api.LayerG/AuthenticateFacebookInstantGame", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *authenticatorServiceClient) AuthenticateFacebook(ctx context.Context, in *api.AuthenticateFacebookRequest, opts ...grpc.CallOption) (*api.Session, error) {
	out := new(api.Session)
	err := c.cc.Invoke(ctx, "/layerg.api.LayerG/AuthenticateFacebookInstantGame", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *authenticatorServiceClient) AuthenticateGameCenter(ctx context.Context, in *api.AuthenticateGameCenterRequest, opts ...grpc.CallOption) (*api.Session, error) {
	out := new(api.Session)
	err := c.cc.Invoke(ctx, "/layerg.api.LayerG/AuthenticateGameCenter", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *authenticatorServiceClient) AuthenticateSteam(ctx context.Context, in *api.AuthenticateSteamRequest, opts ...grpc.CallOption) (*api.Session, error) {
	out := new(api.Session)
	err := c.cc.Invoke(ctx, "/layerg.api.LayerG/AuthenticateSteam", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *authenticatorServiceClient) AuthenticateEvm(ctx context.Context, in *api.AuthenticateEvmRequest, opts ...grpc.CallOption) (*api.Session, error) {
	out := new(api.Session)
	err := c.cc.Invoke(ctx, "/layerg.api.LayerG/AuthenticateEvm", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authenticatorServiceClient) AuthenticateTelegram(ctx context.Context, in *api.AuthenticateTelegramRequest, opts ...grpc.CallOption) (*api.Session, error) {
	out := new(api.Session)
	err := c.cc.Invoke(ctx, "/layerg.api.LayerG/AuthenticateTelegram", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *authenticatorServiceClient) SessionRefresh(ctx context.Context, in *api.SessionRefreshRequest, opts ...grpc.CallOption) (*api.Session, error) {
	out := new(api.Session)
	err := c.cc.Invoke(ctx, "/layerg.api.LayerG/SessionRefresh", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
