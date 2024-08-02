package apigrpc

//go:generate protoc -I. -I../vendor -I../build/grpc-gateway-v2.3.0/third_party/googleapis -I../vendor/github.com/grpc-ecosystem/grpc-gateway/v2 --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative --grpc-gateway_opt=logtostderr=true --openapiv2_out=. --openapiv2_opt=logtostderr=true,allow_delete_body=true apigrpc.proto
