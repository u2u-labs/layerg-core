package server

import (
	"context"

	"github.com/u2u-labs/layerg-core/console"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *ConsoleServer) GetRuntime(ctx context.Context, in *emptypb.Empty) (*console.RuntimeInfo, error) {
	toConsole := func(modules []*moduleInfo) []*console.RuntimeInfo_ModuleInfo {
		result := make([]*console.RuntimeInfo_ModuleInfo, 0, len(modules))
		for _, m := range modules {
			result = append(result, &console.RuntimeInfo_ModuleInfo{
				Path:    m.path,
				ModTime: &timestamppb.Timestamp{Seconds: m.modTime.UTC().Unix()},
			})
		}
		return result
	}

	return &console.RuntimeInfo{
		LuaRpcFunctions: s.runtimeInfo.LuaRpcFunctions,
		GoRpcFunctions:  s.runtimeInfo.GoRpcFunctions,
		JsRpcFunctions:  s.runtimeInfo.JavaScriptRpcFunctions,
		GoModules:       toConsole(s.runtimeInfo.GoModules),
		LuaModules:      toConsole(s.runtimeInfo.LuaModules),
		JsModules:       toConsole(s.runtimeInfo.JavaScriptModules),
	}, nil
}
