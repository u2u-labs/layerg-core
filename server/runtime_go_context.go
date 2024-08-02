package server

import (
	"context"

	"github.com/u2u-labs/go-layerg-common/runtime"
)

// ignore warnings about strings being used as ctx keys
//
//nolint:staticcheck
func NewRuntimeGoContext(ctx context.Context, node, version string, env map[string]string, mode RuntimeExecutionMode, headers, queryParams map[string][]string, sessionExpiry int64, userID, username string, vars map[string]string, sessionID, clientIP, clientPort, lang string) context.Context {
	ctx = context.WithValue(ctx, runtime.RUNTIME_CTX_ENV, env)
	ctx = context.WithValue(ctx, runtime.RUNTIME_CTX_MODE, mode.String())
	ctx = context.WithValue(ctx, runtime.RUNTIME_CTX_NODE, node)
	ctx = context.WithValue(ctx, runtime.RUNTIME_CTX_VERSION, version)

	if headers != nil {
		ctx = context.WithValue(ctx, runtime.RUNTIME_CTX_HEADERS, headers)
	}
	if queryParams != nil {
		ctx = context.WithValue(ctx, runtime.RUNTIME_CTX_QUERY_PARAMS, queryParams)
	}

	if userID != "" {
		ctx = context.WithValue(ctx, runtime.RUNTIME_CTX_USER_ID, userID)
		ctx = context.WithValue(ctx, runtime.RUNTIME_CTX_USERNAME, username)
		if vars != nil {
			ctx = context.WithValue(ctx, runtime.RUNTIME_CTX_VARS, vars)
		}
		ctx = context.WithValue(ctx, runtime.RUNTIME_CTX_USER_SESSION_EXP, sessionExpiry)
		if sessionID != "" {
			ctx = context.WithValue(ctx, runtime.RUNTIME_CTX_SESSION_ID, sessionID)
			// Lang is never reported without session ID.
			ctx = context.WithValue(ctx, runtime.RUNTIME_CTX_LANG, lang)
		}
	}

	if clientIP != "" {
		ctx = context.WithValue(ctx, runtime.RUNTIME_CTX_CLIENT_IP, clientIP)
	}
	if clientPort != "" {
		ctx = context.WithValue(ctx, runtime.RUNTIME_CTX_CLIENT_PORT, clientPort)
	}

	return ctx
}
