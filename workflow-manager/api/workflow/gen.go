package workflow

//go:generate protoc -I . -I ../ -I ../../third_party/ --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --coredge-sdk_out . --coredge-sdk_opt paths=source_relative --grpc-gateway_out . --grpc-gateway_opt paths=source_relative --openapiv2_out ./swagger/OpenAPI --openapiv2_opt logtostderr=true --openapiv2_opt allow_merge=true base-image.proto module.proto template.proto workflow.proto
