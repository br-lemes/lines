.PHONY: build clean linux-amd64 linux-arm64 release test version windows

TARGET := $(notdir $(shell go list -m 2>/dev/null))
ifeq ($(TARGET),)
	TARGET := $(notdir $(CURDIR))
endif

export CGO_ENABLED=0

build: test
	@go build -ldflags "-s -w"

clean:
	$(RM) $(TARGET) $(TARGET)-linux-amd64 $(TARGET)-linux-arm64 $(TARGET).exe

linux-amd64: test
	@GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o $(TARGET)-linux-amd64

linux-arm64: test
	@GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o $(TARGET)-linux-arm64

release: version linux-amd64 linux-arm64 windows
	@go run ./tools/release/main.go

test:
	@go test ./...

version: test
	@go run ./tools/version/main.go

windows: test
	@GOOS=windows GOARCH=amd64 go build -ldflags "-s -w"
