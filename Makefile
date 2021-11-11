# SyntropyAgent-GO build script

APPNAME:=syntropy_agent
# Get git discribe. Github actions will pass this variable.
# If it is missing - then this is a local build and get it from git.
# AGENT_VERSION is set by Docker build
FULL_VERSION := $(AGENT_VERSION)
ifeq ($(FULL_VERSION), ${EMPTY:Q})
FULL_VERSION := $(shell git describe --tags --dirty --candidates=1)
endif
# Split git describe into version and subversion
# 1.0.4-14-g2414721-dirty ==> version = 1.0.4, subversion = 14.g2414721.dirty
# NOTE: do not include `v` in versioning
VERSION = $(shell echo $(FULL_VERSION) | cut -d "-" -f1)
SUBVERSION = $(shell echo $(FULL_VERSION) | cut -d "-" -f2-6 | sed -r 's/-/+/' | sed -r 's/-/./g')
ifeq ($(FULL_VERSION), $(VERSION))
SUBVERSION:=
endif

# Sanity fallback (should not happen in normal environment)
ifeq ($(VERSION), ${EMPTY:Q})
VERSION:=0.0.0
endif


all: deps agent-go

deps:
	@echo Fetching dependencies:
	go get -d ./...

agent-go:
	@echo Building $(APPNAME)  $(VERSION) - $(SUBVERSION)
# build the agent
	go build -o $(APPNAME) -ldflags \
		"-X github.com/SyntropyNet/syntropy-agent-go/internal/config.version=$(VERSION) \
		-X github.com/SyntropyNet/syntropy-agent-go/internal/config.subversion=$(SUBVERSION) -s -w" \
		./cmd/main.go

test:
	go test ./...

clean:
	go clean
	rm -f $(APPNAME)