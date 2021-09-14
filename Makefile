# SyntropyAgent-GO build script

# Get git discribe. Github actions will pass this variable.
# If it is missing - then this is a local build and get it from git.
ifneq ($(AGENT_VERSION), ${EMPTY:Q})
FULL_VERSION := $(AGENT_VERSION)
else 
FULL_VERSION := $(shell git describe --tags --dirty --candidates=1)
endif
# Split git describe into version and subversion
# 1.0.4-14-g2414721 ==> version = 1.0.4, subversion = 14-g2414721
# NOTE: do not include `v` in versioning
VERSION = $(shell echo $(FULL_VERSION) | cut -d "-" -f1)
ifeq ($(FULL_VERSION), $(VERSION))
SUBVERSION := ""
else
SUBVERSION = $(shell echo $(FULL_VERSION) | cut -d "-" -f2-4)
endif

# Sanity fallback (should not happen in normal environment)
ifeq ($(VERSION), ${EMPTY:Q})
VERSION := "0.0.0"
endif


all: agent-go

agent-go:
	@echo Building $`sag$`  $(VERSION) - $(SUBVERSION)
	go build -o sag -ldflags \
		"-X github.com/SyntropyNet/syntropy-agent-go/internal/config.version=$(VERSION) \
		-X github.com/SyntropyNet/syntropy-agent-go/internal/config.subversion=$(SUBVERSION) -s -w" \
		./cmd/main.go

test:
	go test ./...

clean:
	go clean
	rm -f sag