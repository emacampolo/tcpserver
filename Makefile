.PHONY: all
all: tidy format lint test

.PHONY: tidy
tidy:
	@echo "=> Executing go mod tidy"
	@go mod tidy

.PHONY: format
format:
	@echo "=> Formatting code and organizing imports"
	@type "goimports" > /dev/null 2>&1 || go install golang.org/x/tools/cmd/goimports@latest
	@goimports -w ./

.PHONY: lint
lint:
	@echo "=> Executing staticcheck"
	@type "staticcheck" > /dev/null 2>&1 || go install honnef.co/go/tools/cmd/staticcheck@latest
	@staticcheck -checks=all,-ST1000 ./...
	@echo "=> Executing go vet"
	@go vet ./...

.PHONY: test
test:
	@echo "=> Running tests"
	@go test ./... -covermode=atomic -coverprofile=/tmp/coverage.out -coverpkg=./... -count=1 -race -shuffle=on
