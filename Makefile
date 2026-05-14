docs:
	swag init --parseDependency -g ./cmd/api/main.go -o ./docs

lint:
	golangci-lint run

test-unit:
	go test ./internal/... -race -coverprofile=coverage.out -covermode=atomic -v

test-repository:
	go test ./internal/repository/... -race -coverprofile=coverage.out -covermode=atomic -v

test-service:
	go test ./internal/service/... -race -coverprofile=coverage.out -covermode=atomic -v

test-dto:
	go test ./internal/dto/... -race -coverprofile=coverage.out -covermode=atomic -v

test-integration:
	go test -tags integration ./internal/handler/... -race -coverprofile=coverage_integration.out -coverpkg=./internal/handler/... -covermode=atomic -v

mockgen-install:
	go install github.com/golang/mock/mockgen

mocks: repository-mocks

repository-mocks:
	mockgen -source=./internal/repository/notification.go -destination=./internal/repository/mocks/notification_mock.go -package=mocks
