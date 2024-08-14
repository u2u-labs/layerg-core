package console

//go:generate protoc -I. -I../vendor -I../vendor/github.com/u2u-labs/go-layerg-common -I../build/grpc-gateway-v2.3.0/third_party/googleapis -I../vendor/github.com/grpc-ecosystem/grpc-gateway/v2 --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative --grpc-gateway_opt=logtostderr=true --grpc-gateway_opt=generate_unbound_methods=true --openapiv2_out=. --openapiv2_opt=json_names_for_fields=false,logtostderr=true console.proto
//go:generate sh -c "(cd openapi-gen-angular && go run . -i '../console.swagger.json' -o '../ui/src/app/console.service.ts' -rm_prefix='console,layergconsole,layerg,Console_')"
