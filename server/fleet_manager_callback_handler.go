package server

import (
	"net"
	"sync"

	"github.com/gofrs/uuid/v5"
	"github.com/u2u-labs/go-layerg-common/runtime"
)

type LocalFmCallbackHandler struct {
	callbackRegistry sync.Map
	idGenerator      uuid.Generator

	nodeHash [6]byte
}

func NewLocalFmCallbackHandler(config Config) runtime.FmCallbackHandler {
	hash := NodeToHash(config.GetName())
	callbackIdGen := uuid.NewGenWithHWAF(func() (net.HardwareAddr, error) {
		return hash[:], nil
	})

	return &LocalFmCallbackHandler{
		callbackRegistry: sync.Map{},
		idGenerator:      callbackIdGen,

		nodeHash: hash,
	}
}

func (fch *LocalFmCallbackHandler) GenerateCallbackId() string {
	return uuid.Must(fch.idGenerator.NewV1()).String()
}

func (fch *LocalFmCallbackHandler) SetCallback(callbackId string, fn runtime.FmCreateCallbackFn) {
	fch.callbackRegistry.Store(callbackId, fn)
}

func (fch *LocalFmCallbackHandler) InvokeCallback(callbackId string, status runtime.FmCreateStatus, instanceInfo *runtime.InstanceInfo, sessionInfo []*runtime.SessionInfo, metadata map[string]any, err error) {
	callback, ok := fch.callbackRegistry.LoadAndDelete(callbackId)
	if !ok || callback == nil {
		return
	}

	fn := callback.(runtime.FmCreateCallbackFn)
	fn(status, instanceInfo, sessionInfo, metadata, err)
}
