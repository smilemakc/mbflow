PROTO_DIR = ../../api/proto
PB_DIR = internal/pb

.PHONY: proto test lint

proto:
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(PB_DIR) --go_opt=paths=source_relative \
		--go_opt=Mservice_api.proto=github.com/smilemakc/mbflow/sdk/go/internal/pb \
		--go-grpc_out=$(PB_DIR) --go-grpc_opt=paths=source_relative \
		--go-grpc_opt=Mservice_api.proto=github.com/smilemakc/mbflow/sdk/go/internal/pb \
		$(PROTO_DIR)/service_api.proto

test:
	go test ./... -v -race

lint:
	golangci-lint run ./...