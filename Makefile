PLATFORMS := linux-amd64 linux-arm64 windows-amd64

.PHONY: build clean release test version $(PLATFORMS)

TARGET := $(notdir $(shell go list -m 2>/dev/null))
ifeq ($(TARGET),)
	TARGET := $(notdir $(CURDIR))
endif

export CGO_ENABLED=0

ARTIFACTS := $(foreach p,$(PLATFORMS),\
	$(TARGET)-$(p)$(if $(filter windows%,$(p)),.exe))

SEMVER := github.com/br-lemes/semver@latest

build: test
	@go build -ldflags "-s -w"

clean:
	$(RM) $(TARGET) $(ARTIFACTS)

$(PLATFORMS): test
	@$(eval GOOS := $(word 1,$(subst -, ,$@)))
	@$(eval GOARCH := $(word 2,$(subst -, ,$@)))
	@GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "-s -w" \
		-o $(TARGET)-$@$(if $(filter windows,$(GOOS)),.exe)

release: version $(PLATFORMS)
	@go run $(SEMVER) release $(ARTIFACTS)

test:
	@go test ./...

version: test
	@go run $(SEMVER)
