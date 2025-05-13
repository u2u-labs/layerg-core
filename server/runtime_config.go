package server

import "github.com/u2u-labs/go-layerg-common/runtime"

type RuntimeConfigClone struct {
	Name          string
	ShutdownGrace int
	Logger        runtime.LoggerConfig
	Session       runtime.SessionConfig
	Socket        runtime.SocketConfig
	Runtime       runtime.RuntimeConfig
	Iap           runtime.IAPConfig
	// Social        runtime.SocialConfig
}

func (c *RuntimeConfigClone) GetName() string {
	return c.Name
}

func (c *RuntimeConfigClone) GetShutdownGraceSec() int {
	return c.ShutdownGrace
}

func (c *RuntimeConfigClone) GetLogger() runtime.LoggerConfig {
	return c.Logger
}

func (c *RuntimeConfigClone) GetSession() runtime.SessionConfig {
	return c.Session
}

func (c *RuntimeConfigClone) GetSocket() runtime.SocketConfig {
	return c.Socket
}

func (c *RuntimeConfigClone) GetRuntime() runtime.RuntimeConfig {
	return c.Runtime
}

func (c *RuntimeConfigClone) GetIAP() runtime.IAPConfig {
	return c.Iap
}

// func (c *RuntimeConfigClone) GetSocial() runtime.SocialConfig {
// 	return c.Social
// }
